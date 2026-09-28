package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
	"github.com/aguinelo/dcode/internal/tools"
)

// An interrupt could still let a model's tool calls run, and an interrupt that
// landed as a final answer ended was read as work nobody could verify.
//
// A stream can end `done` at the very instant the person presses stop. Every
// cancellation has a last instant it can be seen, and for the provider's pump
// it is the read from the transport: a terminal frame taken before the cancel
// is still decoded, and the stream ends done. #392 made that window as small
// as the pump can; it cannot close it.
//
// With calls in the answer, the loop took `done` at its word — it appended the
// answer and went straight to execute, and looked at the context again only at
// the top of the next iteration, after the calls had run. `write` and `edit`
// ignore their context, because a half-applied edit is worse than a slow one, so
// an auto-approved write landed on disk after the turn was over.
//
// With no calls in the answer, the loop went straight to the definition of done
// under the cancelled context. No criterion can start there — `exec.Cmd.Start`
// answers a cancelled context with its error before any process exists — so
// each one read as unavailable. The model was told its work could not be
// verified, a round was spent telling it, and the seal in `turn.completed`
// reported the criteria as unavailable. None of it was true: the person had
// stopped the turn.
//
// The rules these tests hold the loop to: a call that has not started when the
// interrupt becomes visible does not start; and a stop visible when the answer
// ends is the end of the turn — no criterion starts, and nothing about a check
// that never ran reaches the model or the seal.

// ---------- doubles ----------

// stopBeforeTheLastFrame streams the model's answer and ends it done, with the
// turn's context cancelled just before the terminal frame is handed over. The
// frame after the cancel is sent without looking at the context, as the pump
// decodes a frame it had already taken.
type stopBeforeTheLastFrame struct {
	frames []provider.StreamEvent
	cancel context.CancelFunc
}

func (s *stopBeforeTheLastFrame) Family() provider.Family       { return nil }
func (s *stopBeforeTheLastFrame) Transport() provider.Transport { return nil }
func (s *stopBeforeTheLastFrame) Window(string) (int, error)    { return 1_000_000, nil }
func (s *stopBeforeTheLastFrame) Limits() provider.Limits       { return provider.Limits{} }

func (s *stopBeforeTheLastFrame) Stream(context.Context, provider.Request) (<-chan provider.StreamEvent, error) {
	ch := make(chan provider.StreamEvent, len(s.frames)+1)
	go func() {
		defer close(ch)
		for _, ev := range s.frames {
			ch <- ev
		}
		s.cancel()
		ch <- done()
	}()
	return ch, nil
}

// recordingTool counts how often Execute is reached. It declares a write and
// ignores its context, as `write` and `edit` do: once a call like this starts,
// nothing inside it stops, so the loop is the only place an interrupt can.
type recordingTool struct {
	name, path string
	// during runs inside Execute, for a stop that lands while the call runs.
	during func()

	mu  sync.Mutex
	ran int
}

func (r *recordingTool) Name() string            { return r.name }
func (r *recordingTool) Description() string     { return "test tool" }
func (r *recordingTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }

func (r *recordingTool) Declare(json.RawMessage) (policy.Request, error) {
	return policy.Request{Tool: r.name, Paths: []policy.Access{{Path: r.path, Write: true}}}, nil
}

func (r *recordingTool) Execute(context.Context, json.RawMessage, *tools.State) (tools.Result, error) {
	r.mu.Lock()
	r.ran++
	r.mu.Unlock()
	if r.during != nil {
		r.during()
	}
	return tools.Result{Output: "wrote " + r.path}, nil
}

func (r *recordingTool) runs() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.ran
}

// grantAsTheStopLands grants, and the person presses stop while it does.
type grantAsTheStopLands struct{ cancel context.CancelFunc }

func (g grantAsTheStopLands) Approve(context.Context, protocol.ApprovalRequest) (protocol.ApprovalDecision, error) {
	g.cancel()
	return protocol.ApprovalAllow, nil
}

// ---------- the rule: a call that has not started does not start ----------

// The defect as it was found: the answer arrives whole, with its calls, and the
// stop lands before the frame that ends it is handed over.
//
// None of the calls runs, and the history says so. The answer stays — the
// person watched it arrive, and the model has to know what it asked for — and
// every call in it gets a reply saying it did not run. A call with no reply is
// a conversation the provider rejects on the next turn; erasing the call
// leaves the model's own words promising work that never happened. Nothing
// after the replies may claim that a batch ran.
func TestAnInterruptAsTheStreamEndsRunsNoTool(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Two writes to different paths: one group, run together, and allowed in
	// workspace-write without asking anybody.
	a := &recordingTool{name: "put_a", path: "a.go"}
	b := &recordingTool{name: "put_b", path: "b.go"}
	p := &stopBeforeTheLastFrame{cancel: cancel, frames: []provider.StreamEvent{
		text("Writing both files."),
		call("c1", "put_a", `{}`),
		call("c2", "put_b", `{}`),
	}}
	e, rec := newEngine(t, p, tools.NewRegistry(a, b))

	out, err := e.Run(ctx, "write a.go and b.go")
	if err != nil {
		t.Fatalf("an interrupt is not an error: %v", err)
	}
	if out.Reason != protocol.StopInterrupted {
		t.Errorf("reason = %q, want %q", out.Reason, protocol.StopInterrupted)
	}
	if n := rec.count(protocol.EventTurnCompleted); n != 1 {
		t.Errorf("turn.completed emitted %d times, want exactly one", n)
	}
	for _, tool := range []*recordingTool{a, b} {
		if n := tool.runs(); n != 0 {
			t.Errorf("%s ran %d time(s) after the turn was interrupted", tool.name, n)
		}
	}

	h := e.Session().History
	if len(h) != 4 {
		t.Fatalf("history has %d messages, want the question, the answer and one reply per call:\n%s",
			len(h), transcript(h))
	}
	if m := h[1]; m.Role != ce.RoleAssistant || m.Text != "Writing both files." || len(m.ToolCalls) != 2 {
		t.Errorf("the model's answer did not survive whole:\n%s", transcript(h))
	}
	answeredNotRun(t, h, "c1", "c2")
}

// The stop can land while one group of a batch runs. That call finishes — it is
// under way, and RN-5 keeps its result — and it is the last one that starts:
// the later groups do not, and the call among them that crosses a boundary is
// not put to the person.
//
// Asking would be worse than noise. The session's approver answers a cancelled
// context with deny, and a denial reaches the model as the person refusing, with
// "do not retry it". The person refused nothing; they stopped the turn.
func TestAnInterruptDuringABatchStartsNoLaterCall(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// The person presses stop while this one runs.
	first := &recordingTool{name: "first", path: "a.go", during: cancel}
	// The same path, so it runs in the group after.
	second := &recordingTool{name: "second", path: "a.go"}
	// Outside the workspace: it needs the person, and runs alone, last.
	reach := &recordingTool{name: "reach", path: "/etc/hosts"}
	ap := &fixedApprover{decision: protocol.ApprovalAllow}

	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "first", `{}`), call("c2", "second", `{}`), call("c3", "reach", `{}`), done()},
	}}
	e, _ := newEngine(t, p, tools.NewRegistry(first, second, reach), func(c *Config) {
		c.Approver = ap
	})

	out, err := e.Run(ctx, "go")
	if err != nil {
		t.Fatalf("an interrupt is not an error: %v", err)
	}
	if out.Reason != protocol.StopInterrupted {
		t.Errorf("reason = %q, want %q", out.Reason, protocol.StopInterrupted)
	}
	if n := first.runs(); n != 1 {
		t.Fatalf("the first call ran %d time(s); the stop was meant to land inside it", n)
	}
	for _, tool := range []*recordingTool{second, reach} {
		if n := tool.runs(); n != 0 {
			t.Errorf("%s started %d time(s) after the turn was interrupted", tool.name, n)
		}
	}
	if ap.asked != 0 {
		t.Errorf("the person was asked %d time(s) about a call from a turn they had stopped", ap.asked)
	}

	h := e.Session().History
	var ran *ce.ToolResult
	for _, m := range h {
		if m.ToolResult != nil && m.ToolResult.ToolCallID == "c1" {
			ran = m.ToolResult
		}
	}
	if ran == nil || ran.IsError || ran.Output != "wrote a.go" {
		t.Errorf("the call that ran lost its result; the disk changed and the history must say so:\n%s",
			transcript(h))
	}
	answeredNotRun(t, h, "c2", "c3")
}

// The approval is a wait, and the stop can land inside it. The session's
// approver answers a cancelled context with deny, but a standing grant — or an
// allow-for-this-session given earlier — answers without looking at the
// context at all, and a person's "allow" can race their own stop. Whatever the
// answer, a call granted after the interrupt does not run.
func TestAGrantThatArrivesWithTheInterruptRunsNothing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reach := &recordingTool{name: "reach", path: "/etc/hosts"}
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "reach", `{}`), done()},
	}}
	e, _ := newEngine(t, p, tools.NewRegistry(reach), func(c *Config) {
		c.Approver = grantAsTheStopLands{cancel: cancel}
	})

	out, err := e.Run(ctx, "go")
	if err != nil {
		t.Fatalf("an interrupt is not an error: %v", err)
	}
	if out.Reason != protocol.StopInterrupted {
		t.Errorf("reason = %q, want %q", out.Reason, protocol.StopInterrupted)
	}
	if n := reach.runs(); n != 0 {
		t.Errorf("a call granted as the turn was interrupted ran %d time(s)", n)
	}
	answeredNotRun(t, e.Session().History, "c1")
}

// ---------- the rule: an answer with no call runs no criterion ----------

// The defect as it was found: files changed earlier in the turn, the model
// answers with text alone, and the stop lands before the frame that ends the
// answer is handed over.
//
// The criterion does not run. The spy answers as the real runner would under a
// cancelled context, so a run that reaches it shows the symptom a person saw:
// the reminder that nothing could confirm the work, a round spent on it, and a
// seal naming the criterion unavailable.
func TestAnInterruptAsTheAnswerEndsRunsNoCriterion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := &stopBeforeTheLastFrame{cancel: cancel, frames: []provider.StreamEvent{
		text("The parser is fixed."),
	}}
	e, rec := newEngine(t, p, tools.NewRegistry(), func(c *Config) {
		c.DoneEnabled = true
		c.Done = DoneSet{Criteria: []Criterion{{Name: "tests", Command: "make test"}}}
		c.WrittenPaths = func() []string { return []string{"parser.go"} }
		c.RunCriterion = func(ctx context.Context, command string) (int, string, error) {
			t.Errorf("criterion %q ran after the person stopped the turn", command)
			return -1, "", ctx.Err()
		}
	})

	out, err := e.Run(ctx, "fix the parser")
	if err != nil {
		t.Fatalf("an interrupt is not an error: %v", err)
	}
	if out.Reason != protocol.StopInterrupted {
		t.Errorf("reason = %q, want %q", out.Reason, protocol.StopInterrupted)
	}
	if n := rec.count(protocol.EventTurnCompleted); n != 1 {
		t.Errorf("turn.completed emitted %d times, want exactly one", n)
	}
	if out.Iterations != 0 {
		t.Errorf("the turn went round %d more time(s), for a check that never ran", out.Iterations)
	}

	h := e.Session().History
	for _, m := range h {
		if m.Reminder && strings.Contains(m.Text, "could not verify") {
			t.Errorf("the model was told its work could not be verified; the person stopped the turn:\n%s",
				transcript(h))
		}
	}
	answerKeptWhole(t, h, "The parser is fixed.")
	toldItWasInterrupted(t, h, "parser.go")

	rec.mu.Lock()
	tc, _ := rec.last[protocol.EventTurnCompleted].(protocol.TurnCompleted)
	rec.mu.Unlock()
	if c := tc.Completion; c != nil {
		t.Errorf("turn.completed carries a seal (%s; unavailable: %v) for a check that never ran",
			c.Verification, c.Unavailable)
	}
	if got := e.Report().States; len(got) != 0 {
		t.Errorf("the report holds %v from a check that never ran", got)
	}
}

// With no definition of done there is no check to keep from starting, and the
// stop still ends the turn as a stop. The loop saw it before the turn was over,
// and what the turn would have done next is configuration, not what happened:
// the same instant reads the same way whatever is configured.
func TestAnInterruptAsTheAnswerEndsReadsInterruptedWithNothingToCheck(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := &stopBeforeTheLastFrame{cancel: cancel, frames: []provider.StreamEvent{
		text("The parser is fixed."),
	}}
	e, rec := newEngine(t, p, tools.NewRegistry(), func(c *Config) {
		c.WrittenPaths = func() []string { return []string{"parser.go"} }
	})

	out, err := e.Run(ctx, "fix the parser")
	if err != nil {
		t.Fatalf("an interrupt is not an error: %v", err)
	}
	if out.Reason != protocol.StopInterrupted {
		t.Errorf("reason = %q, want %q: the person stopped the turn, and the loop saw it before the turn ended",
			out.Reason, protocol.StopInterrupted)
	}
	if n := rec.count(protocol.EventTurnCompleted); n != 1 {
		t.Errorf("turn.completed emitted %d times, want exactly one", n)
	}

	h := e.Session().History
	answerKeptWhole(t, h, "The parser is fixed.")
	toldItWasInterrupted(t, h, "parser.go")
}

// ---------- helpers ----------

// answeredNotRun fails unless each named call has exactly one reply, and the
// reply is an error saying the call did not run.
func answeredNotRun(t *testing.T, h []ce.Message, ids ...string) {
	t.Helper()
	for _, id := range ids {
		var replies []*ce.ToolResult
		for _, m := range h {
			if m.ToolResult != nil && m.ToolResult.ToolCallID == id {
				replies = append(replies, m.ToolResult)
			}
		}
		if len(replies) != 1 {
			t.Errorf("call %s has %d replies, want exactly one:\n%s", id, len(replies), transcript(h))
			continue
		}
		if r := replies[0]; !r.IsError || !strings.Contains(r.Output, "not run") {
			t.Errorf("call %s is answered %q; it never ran, and the reply has to say so", id, r.Output)
		}
	}
}

// answerKeptWhole fails unless the model's answer is in the history exactly as
// it arrived. The person watched it arrive, and the stop came after it.
func answerKeptWhole(t *testing.T, h []ce.Message, want string) {
	t.Helper()
	for _, m := range h {
		if m.Role == ce.RoleAssistant && m.Text == want {
			return
		}
	}
	t.Errorf("the model's answer %q is not in the history whole:\n%s", want, transcript(h))
}

// toldItWasInterrupted fails unless the history says the turn was interrupted
// and names the files changed before it was: what the model is told instead of
// a verdict nobody reached.
func toldItWasInterrupted(t *testing.T, h []ce.Message, written ...string) {
	t.Helper()
	for _, m := range h {
		if !m.Reminder || !strings.Contains(m.Text, "The turn was interrupted") {
			continue
		}
		for _, w := range written {
			if !strings.Contains(m.Text, w) {
				t.Errorf("the note that the turn was interrupted does not name %s:\n%s", w, transcript(h))
			}
		}
		return
	}
	t.Errorf("nothing in the history says the turn was interrupted:\n%s", transcript(h))
}

// transcript writes a history one line per message, in the order the model
// reads it, so a failure shows what the loop did rather than a struct.
func transcript(h []ce.Message) string {
	var b strings.Builder
	for _, m := range h {
		switch {
		case m.ToolResult != nil:
			fmt.Fprintf(&b, "  tool %s: %s\n", m.ToolResult.ToolCallID, m.ToolResult.Output)
		case m.Reminder:
			s := strings.Join(strings.Fields(m.Text), " ")
			if len(s) > 100 {
				s = s[:100] + "…"
			}
			fmt.Fprintf(&b, "  reminder: %s\n", s)
		default:
			fmt.Fprintf(&b, "  %s: %s", m.Role, m.Text)
			for _, c := range m.ToolCalls {
				fmt.Fprintf(&b, " [%s %s]", c.ID, c.Name)
			}
			b.WriteByte('\n')
		}
	}
	return b.String()
}
