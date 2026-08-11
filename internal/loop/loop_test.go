package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
	"github.com/aguinelo/dcode/internal/tools"
)

// ---------- doubles ----------

// scriptedProvider replays a fixed sequence of turns, so a whole loop can be
// exercised without a model or a network.
type scriptedProvider struct {
	turns  [][]provider.StreamEvent
	call   int
	limits provider.Limits
	mu     sync.Mutex
	seen   []provider.Request
}

func (s *scriptedProvider) Family() provider.Family       { return nil }
func (s *scriptedProvider) Transport() provider.Transport { return nil }
func (s *scriptedProvider) Window(string) (int, error)    { return 1_000_000, nil }
func (s *scriptedProvider) Limits() provider.Limits       { return s.limits }

func (s *scriptedProvider) Stream(ctx context.Context, req provider.Request) (<-chan provider.StreamEvent, error) {
	s.mu.Lock()
	idx := s.call
	s.call++
	s.seen = append(s.seen, req)
	s.mu.Unlock()

	ch := make(chan provider.StreamEvent, 8)
	go func() {
		defer close(ch)
		var evs []provider.StreamEvent
		if idx < len(s.turns) {
			evs = s.turns[idx]
		} else {
			// Running past the script means the loop asked for more turns than
			// expected; answer with a clean stop so a bug shows up as a wrong
			// assertion rather than a hang.
			evs = []provider.StreamEvent{{Type: provider.EventDone}}
		}
		for _, ev := range evs {
			select {
			case ch <- ev:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}

func text(s string) provider.StreamEvent {
	return provider.StreamEvent{Type: provider.EventTextDelta, Text: s}
}

func call(id, name, input string) provider.StreamEvent {
	return provider.StreamEvent{Type: provider.EventToolCall, ToolCall: &ce.ToolCall{
		ID: id, Name: name, Input: json.RawMessage(input),
	}}
}

func done() provider.StreamEvent { return provider.StreamEvent{Type: provider.EventDone} }

// spent ends a turn declaring what it cost, which is how the accounting tests
// can measure a debit instead of performing it themselves.
func spent(in, out int) provider.StreamEvent {
	return provider.StreamEvent{
		Type:  provider.EventDone,
		Usage: &provider.Usage{InputTokens: in, OutputTokens: out},
	}
}

// recorder captures emitted events.
type recorder struct {
	mu     sync.Mutex
	events []protocol.EventType
	last   map[protocol.EventType]any
}

func newRecorder() *recorder {
	return &recorder{last: map[protocol.EventType]any{}}
}

func (r *recorder) Emit(t protocol.EventType, payload any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, t)
	r.last[t] = payload
}

func (r *recorder) count(t protocol.EventType) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, e := range r.events {
		if e == t {
			n++
		}
	}
	return n
}

type fixedApprover struct {
	decision protocol.ApprovalDecision
	asked    int
	mu       sync.Mutex
}

func (f *fixedApprover) Approve(context.Context, protocol.ApprovalRequest) (protocol.ApprovalDecision, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.asked++
	return f.decision, nil
}

// slowTool records ordering so append-order can be checked against completion
// order.
type slowTool struct {
	name  string
	delay time.Duration
	path  string
	write bool
}

func (s slowTool) Name() string            { return s.name }
func (s slowTool) Description() string     { return "test tool" }
func (s slowTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }

func (s slowTool) Declare(json.RawMessage) (policy.Request, error) {
	return policy.Request{Tool: s.name, Paths: []policy.Access{{Path: s.path, Write: s.write}}}, nil
}

func (s slowTool) Execute(ctx context.Context, _ json.RawMessage, _ *tools.State) (tools.Result, error) {
	select {
	case <-time.After(s.delay):
	case <-ctx.Done():
	}
	return tools.Result{Output: "from " + s.name}, nil
}

func newEngine(t *testing.T, p provider.Provider, reg *tools.Registry, mut ...func(*Config)) (*Engine, *recorder) {
	t.Helper()
	ws := t.TempDir()
	res, err := policy.NewResolver(ws)
	if err != nil {
		t.Fatal(err)
	}
	rec := newRecorder()
	cfg := Config{
		Provider:  p,
		Tools:     reg,
		State:     tools.NewState(res, tools.DefaultLimits()),
		Emitter:   rec,
		Limits:    DefaultLimits(),
		Mode:      policy.ModeWorkspaceWrite,
		Policy:    policy.PolicyOnRequest,
		Model:     "test-model",
		Parallel:  4,
		Reminders: true,
		// The wait is real in production and instant here. A test that asserts
		// retry behaviour should not spend fifteen seconds proving it, and a
		// suite that slow is a suite people stop running.
		Sleep: func(context.Context, time.Duration) bool { return true },
	}
	for _, m := range mut {
		m(&cfg)
	}
	return New(cfg, ce.Session{Instructions: "You are dcode."}), rec
}

// ---------- the cycle ----------

func TestTurnEndsWhenTheModelStopsCallingTools(t *testing.T) {
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{text("all done"), done()},
	}}
	e, rec := newEngine(t, p, tools.NewRegistry())

	out, err := e.Run(context.Background(), "do nothing")
	if err != nil {
		t.Fatal(err)
	}
	if out.Reason != protocol.StopDone {
		t.Errorf("got %q want done", out.Reason)
	}
	if rec.count(protocol.EventTurnCompleted) != 1 {
		t.Errorf("exactly one turn.completed expected, got %d", rec.count(protocol.EventTurnCompleted))
	}
}

// A failing tool must not end the turn. Aborting on the first error is the
// classic harness design mistake: the model almost always recovers, and
// recovering is what an agent is for.
func TestToolErrorFeedsTheModelAndTheTurnContinues(t *testing.T) {
	reg := tools.NewRegistry(tools.Read{})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "read", `{"path":"nope.go"}`), done()},
		{text("I see, let me try something else"), done()},
	}}
	e, rec := newEngine(t, p, reg)

	out, err := e.Run(context.Background(), "read a missing file")
	if err != nil {
		t.Fatal(err)
	}
	if out.Reason != protocol.StopDone {
		t.Fatalf("a tool error must not end the turn, got %q", out.Reason)
	}
	if out.Iterations < 1 {
		t.Error("the loop should have iterated after the error")
	}

	// The error has to reach the model, or it cannot recover from it.
	var sawError bool
	for _, m := range e.Session().History {
		if m.ToolResult != nil && m.ToolResult.IsError {
			sawError = true
		}
	}
	if !sawError {
		t.Error("the failure must be appended as a tool result")
	}
	if rec.count(protocol.EventToolCompleted) == 0 {
		t.Error("the failed call should still be reported")
	}
}

// The same call three times is a loop, not persistence.
func TestRepeatDetectorStopsBeforeTheIterationCap(t *testing.T) {
	reg := tools.NewRegistry(tools.Read{})
	same := call("c", "read", `{"path":"a.go"}`)
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{same, done()}, {same, done()}, {same, done()}, {same, done()},
	}}
	e, _ := newEngine(t, p, reg, func(c *Config) {
		c.Limits = Limits{MaxIterations: 100, MaxIdenticalCalls: 3}
	})

	out, err := e.Run(context.Background(), "loop")
	if err != nil {
		t.Fatal(err)
	}
	if out.Reason != protocol.StopRepeatLoop {
		t.Fatalf("got %q want repeat_loop", out.Reason)
	}
	if out.Iterations > 5 {
		t.Errorf("the detector should trip early, took %d iterations", out.Iterations)
	}
}

// Retrying with a different approach is not an identical call, so recovery must
// survive the detector. If this broke, the detector would punish exactly the
// behaviour the loop wants.
func TestRetryingDifferentlyDoesNotTripTheDetector(t *testing.T) {
	reg := tools.NewRegistry(tools.Read{})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "read", `{"path":"a.go"}`), done()},
		{call("c2", "read", `{"path":"b.go"}`), done()},
		{call("c3", "read", `{"path":"c.go"}`), done()},
		{text("giving up, none of them exist"), done()},
	}}
	e, _ := newEngine(t, p, reg, func(c *Config) {
		c.Limits = Limits{MaxIterations: 100, MaxIdenticalCalls: 3}
	})

	out, err := e.Run(context.Background(), "find it")
	if err != nil {
		t.Fatal(err)
	}
	if out.Reason != protocol.StopDone {
		t.Errorf("different inputs are not a repeat, got %q", out.Reason)
	}
}

func TestIterationCapIsTheBackstop(t *testing.T) {
	reg := tools.NewRegistry(tools.Read{})
	var turns [][]provider.StreamEvent
	for i := 0; i < 20; i++ {
		turns = append(turns, []provider.StreamEvent{
			call(fmt.Sprintf("c%d", i), "read", fmt.Sprintf(`{"path":"f%d.go"}`, i)), done(),
		})
	}
	p := &scriptedProvider{turns: turns}
	e, _ := newEngine(t, p, reg, func(c *Config) {
		c.Limits = Limits{MaxIterations: 3, MaxIdenticalCalls: 0}
	})

	out, err := e.Run(context.Background(), "keep going")
	if err != nil {
		t.Fatal(err)
	}
	if out.Reason != protocol.StopMaxIterations {
		t.Fatalf("got %q want max_iterations", out.Reason)
	}
	if out.Iterations != 3 {
		t.Errorf("got %d iterations want 3", out.Iterations)
	}
}

// The cap comes from the family, because the work horizon is a property of the
// model rather than of the harness.
func TestIterationCapDefaultsToTheFamily(t *testing.T) {
	p := &scriptedProvider{limits: provider.Limits{MaxIterations: 200}}
	e, _ := newEngine(t, p, tools.NewRegistry(), func(c *Config) {
		c.Limits = Limits{MaxIdenticalCalls: 3} // MaxIterations left unset
	})
	if got := e.cfg.Limits.MaxIterations; got != 200 {
		t.Errorf("got %d want the family's 200", got)
	}
}

func TestExplicitLimitOverridesTheFamily(t *testing.T) {
	p := &scriptedProvider{limits: provider.Limits{MaxIterations: 200}}
	e, _ := newEngine(t, p, tools.NewRegistry(), func(c *Config) {
		c.Limits = Limits{MaxIterations: 7, MaxIdenticalCalls: 3}
	})
	if got := e.cfg.Limits.MaxIterations; got != 7 {
		t.Errorf("got %d want 7", got)
	}
}

// ---------- ordering ----------

// Results are appended at the emission index, never at completion order. The
// naive channel implementation appends on completion, which passes every small
// test and produces irreproducible history under load.
func TestResultsAppendInEmissionOrderNotCompletionOrder(t *testing.T) {
	// The first call is deliberately the slowest.
	reg := tools.NewRegistry(
		slowTool{name: "slow", delay: 80 * time.Millisecond, path: "a"},
		slowTool{name: "fast", delay: time.Millisecond, path: "b"},
	)
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "slow", `{}`), call("c2", "fast", `{}`), done()},
		{text("ok"), done()},
	}}
	e, _ := newEngine(t, p, reg)

	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}

	var results []string
	for _, m := range e.Session().History {
		if m.ToolResult != nil {
			results = append(results, m.ToolResult.Output)
		}
	}
	if len(results) < 2 {
		t.Fatalf("want 2 results, got %v", results)
	}
	if !strings.Contains(results[0], "slow") || !strings.Contains(results[1], "fast") {
		t.Errorf("results must follow emission order, got %v", results)
	}
}

// Parallel execution must be announced through the reminder channel, and the
// text must be constant: any interpolated value would put volatile data in the
// history and cost the cache.
func TestConcurrentExecutionIsAnnouncedWithAConstantNote(t *testing.T) {
	reg := tools.NewRegistry(
		slowTool{name: "one", delay: time.Millisecond, path: "a"},
		slowTool{name: "two", delay: time.Millisecond, path: "b"},
	)
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "one", `{}`), call("c2", "two", `{}`), done()},
		{text("ok"), done()},
	}}

	var notes []string
	for run := 0; run < 3; run++ {
		e, _ := newEngine(t, p, reg)
		p.call = 0
		if _, err := e.Run(context.Background(), "go"); err != nil {
			t.Fatal(err)
		}
		for _, m := range e.Session().History {
			if m.Reminder && strings.Contains(m.Text, "at the same time") {
				notes = append(notes, m.Text)
			}
		}
	}
	if len(notes) < 2 {
		t.Fatalf("the note should be emitted on each concurrent step, got %d", len(notes))
	}
	for _, n := range notes[1:] {
		if n != notes[0] {
			t.Errorf("the note must be byte-identical between runs:\n%q\n%q", notes[0], n)
		}
	}
	// The invariant is that no timestamp, count or ID reaches the model. The
	// byte-identity above already forbids a value that VARIES; this forbids a
	// constant that merely looks like one, which identity cannot see.
	if m := regexp.MustCompile(`\d`).FindString(notes[0]); m != "" {
		t.Errorf("the note carries the digit %q; counts and clocks both start as digits:\n%q", m, notes[0])
	}
}

func TestSequentialExecutionGetsNoNote(t *testing.T) {
	reg := tools.NewRegistry(slowTool{name: "one", delay: time.Millisecond, path: "a"})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "one", `{}`), done()},
		{text("ok"), done()},
	}}
	e, _ := newEngine(t, p, reg)
	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}
	for _, m := range e.Session().History {
		if m.Reminder && strings.Contains(m.Text, "at the same time") {
			t.Error("a single tool call is not concurrent and needs no note")
		}
	}
}

// ---------- policy ----------

// No tool executes without a verdict. A spy in place of the real evaluator
// would be weaker: this asserts the observable consequence.
func TestBoundaryCrossingIsDeniedWithoutAnApprover(t *testing.T) {
	reg := tools.NewRegistry(slowTool{name: "reach", delay: 0, path: "/etc/passwd", write: true})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "reach", `{}`), done()},
		{text("understood"), done()},
	}}
	e, rec := newEngine(t, p, reg) // no Approver

	if _, err := e.Run(context.Background(), "write outside"); err != nil {
		t.Fatal(err)
	}
	if rec.count(protocol.EventApprovalRequired) == 0 {
		t.Error("the crossing should have been announced")
	}

	var denied bool
	for _, m := range e.Session().History {
		if m.ToolResult != nil && m.ToolResult.IsError &&
			strings.Contains(m.ToolResult.Output, "denied") {
			denied = true
		}
	}
	if !denied {
		t.Error("with nobody to ask, the only safe answer is deny")
	}
}

func TestApprovalGrantedLetsTheCallThrough(t *testing.T) {
	reg := tools.NewRegistry(slowTool{name: "reach", delay: 0, path: "/etc/hosts", write: true})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "reach", `{}`), done()},
		{text("ok"), done()},
	}}
	ap := &fixedApprover{decision: protocol.ApprovalAllow}
	e, rec := newEngine(t, p, reg, func(c *Config) { c.Approver = ap })

	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}
	if ap.asked != 1 {
		t.Errorf("the user should have been asked once, got %d", ap.asked)
	}
	if rec.count(protocol.EventApprovalResolved) != 1 {
		t.Error("the decision must be recorded")
	}
}

func TestReadOnlyModeDeniesWritesWithoutAsking(t *testing.T) {
	reg := tools.NewRegistry(slowTool{name: "w", delay: 0, path: "x.txt", write: true})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "w", `{}`), done()},
		{text("ok"), done()},
	}}
	ap := &fixedApprover{decision: protocol.ApprovalAllow}
	e, rec := newEngine(t, p, reg, func(c *Config) {
		c.Mode = policy.ModeReadOnly
		c.Approver = ap
	})

	if _, err := e.Run(context.Background(), "write"); err != nil {
		t.Fatal(err)
	}
	// The boundary is not negotiable: read-only denies rather than escalating,
	// so the user is never asked to approve something impossible.
	if ap.asked != 0 {
		t.Errorf("read-only must not ask, it must deny; asked %d times", ap.asked)
	}
	if rec.count(protocol.EventApprovalRequired) != 0 {
		t.Error("no approval should have been requested")
	}
}

func TestUnknownToolIsReportedToTheModel(t *testing.T) {
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "teleport", `{}`), done()},
		{text("ok, that does not exist"), done()},
	}}
	e, _ := newEngine(t, p, tools.NewRegistry(tools.Read{}))

	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}
	var msg string
	for _, m := range e.Session().History {
		if m.ToolResult != nil && m.ToolResult.IsError {
			msg = m.ToolResult.Output
		}
	}
	if !strings.Contains(msg, "teleport") || !strings.Contains(msg, "read") {
		t.Errorf("the error should name the missing tool and what is available: %q", msg)
	}
}

// ---------- interruption ----------

func TestInterruptEndsTheTurnCleanly(t *testing.T) {
	reg := tools.NewRegistry(slowTool{name: "slow", delay: 2 * time.Second, path: "a"})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "slow", `{}`), done()},
		{text("ok"), done()},
	}}
	e, rec := newEngine(t, p, reg)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	out, err := e.Run(ctx, "go")
	if err != nil {
		t.Fatalf("an interrupt is not an error: %v", err)
	}
	if out.Reason != protocol.StopInterrupted && out.Reason != protocol.StopDone {
		t.Errorf("got %q", out.Reason)
	}
	if rec.count(protocol.EventTurnCompleted) != 1 {
		t.Errorf("exactly one turn.completed, got %d", rec.count(protocol.EventTurnCompleted))
	}
}

func TestCancelledContextStopsBeforeCallingTheModel(t *testing.T) {
	p := &scriptedProvider{turns: [][]provider.StreamEvent{{text("should not run"), done()}}}
	e, _ := newEngine(t, p, tools.NewRegistry())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	out, err := e.Run(ctx, "go")
	if err != nil {
		t.Fatal(err)
	}
	if out.Reason != protocol.StopInterrupted {
		t.Errorf("got %q want interrupted", out.Reason)
	}
	if p.call != 0 {
		t.Error("the model should not have been called")
	}
}

// ---------- events ----------

// A session with no listener must behave identically to one with a listener.
// Anything else means the client is participating in the session rather than
// observing it.
func TestNoEmitterChangesNothing(t *testing.T) {
	reg := tools.NewRegistry(tools.Read{})
	script := [][]provider.StreamEvent{
		{call("c1", "read", `{"path":"x.go"}`), done()},
		{text("done"), done()},
	}

	withEmitter, _ := newEngine(t, &scriptedProvider{turns: script}, reg)
	if _, err := withEmitter.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}

	silent, _ := newEngine(t, &scriptedProvider{turns: script}, reg, func(c *Config) {
		c.Emitter = nil
	})
	if _, err := silent.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}

	a, _ := json.Marshal(withEmitter.Session().History)
	b, _ := json.Marshal(silent.Session().History)
	if string(a) != string(b) {
		t.Errorf("history differs with and without a listener:\n%s\n%s", a, b)
	}
}

func TestPlanUpdatesAreEmitted(t *testing.T) {
	reg := tools.NewRegistry(tools.Plan{})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "plan", `{"items":[{"id":1,"text":"step","status":"active"}]}`), done()},
		{text("ok"), done()},
	}}
	e, rec := newEngine(t, p, reg)

	if _, err := e.Run(context.Background(), "plan it"); err != nil {
		t.Fatal(err)
	}
	if rec.count(protocol.EventPlanUpdated) != 1 {
		t.Fatalf("the client needs the plan as data, got %d events",
			rec.count(protocol.EventPlanUpdated))
	}
	got := rec.last[protocol.EventPlanUpdated].(protocol.PlanUpdated)
	if len(got.Items) != 1 || got.Items[0].Status != protocol.PlanActive {
		t.Errorf("got %+v", got)
	}
}

func TestARejectedPlanEmitsNoUpdate(t *testing.T) {
	reg := tools.NewRegistry(tools.Plan{})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "plan", `{"items":[{"id":1,"text":"a","status":"active"},{"id":2,"text":"b","status":"active"}]}`), done()},
		{text("ok"), done()},
	}}
	e, rec := newEngine(t, p, reg)

	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}
	// The panel must not be told about a plan that was refused.
	if rec.count(protocol.EventPlanUpdated) != 0 {
		t.Error("a rejected plan must not be announced")
	}
}

// ---------- provider errors ----------

func TestUnrecoverableProviderErrorEndsTheTurn(t *testing.T) {
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{{Type: provider.EventError, Err: &provider.ProviderError{
			Class: provider.ErrClassAuth, Message: "bad key",
		}}},
	}}
	e, rec := newEngine(t, p, tools.NewRegistry())

	out, err := e.Run(context.Background(), "go")
	if err == nil {
		t.Fatal("an auth failure must surface")
	}
	if out.Reason != protocol.StopError {
		t.Errorf("got %q", out.Reason)
	}
	if rec.count(protocol.EventSessionError) != 1 {
		t.Error("the failure should be announced to the client")
	}
}

func TestToolSchemaErrorIsFedBackNotFatal(t *testing.T) {
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{{Type: provider.EventError, Err: &provider.ProviderError{
			Class: provider.ErrClassToolSchema, Message: `tool "nope" was not declared`,
		}}},
		{text("understood, using a real tool"), done()},
	}}
	e, _ := newEngine(t, p, tools.NewRegistry(tools.Read{}))

	out, err := e.Run(context.Background(), "go")
	if err != nil {
		t.Fatalf("a schema error is the model's to fix, not fatal: %v", err)
	}
	if out.Reason != protocol.StopDone {
		t.Errorf("got %q", out.Reason)
	}
}

func TestCancelClassEndsQuietly(t *testing.T) {
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{{Type: provider.EventError, Err: &provider.ProviderError{
			Class: provider.ErrClassCanceled, Message: "canceled",
		}}},
	}}
	e, rec := newEngine(t, p, tools.NewRegistry())

	out, err := e.Run(context.Background(), "go")
	if err != nil {
		t.Fatalf("cancellation is not an error: %v", err)
	}
	if out.Reason != protocol.StopInterrupted {
		t.Errorf("got %q", out.Reason)
	}
	if rec.count(protocol.EventSessionError) != 0 {
		t.Error("a deliberate cancellation should not be reported as a failure")
	}
}

// Reasoning must never reach the history.
//
// A model that reads its own thinking back as something it said out loud
// starts defending it, and the text would be paid for on every subsequent turn
// of the session.
func TestReasoningNeverEntersTheHistory(t *testing.T) {
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{
			{Type: provider.EventReasoningDelta, Text: "Let me think about deleting everything."},
			{Type: provider.EventTextDelta, Text: "Pronto."},
			done(),
		},
	}}
	e, rec := newEngine(t, p, tools.NewRegistry(), func(c *Config) { c.ShowReasoning = true })
	if _, err := e.Run(context.Background(), "oi"); err != nil {
		t.Fatal(err)
	}

	for _, m := range e.Session().History {
		if strings.Contains(m.Text, "deleting everything") {
			t.Fatalf("reasoning reached the history: %q", m.Text)
		}
	}

	// Forwarded to the client, though: showing it is the whole point, and the
	// history is the one place it must not reach.
	if d, ok := rec.last[protocol.EventMessageReasoning].(protocol.MessageReasoning); !ok ||
		!strings.Contains(d.Text, "deleting everything") {
		t.Error("the thinking must reach the client")
	}
	// The answer itself survives.
	var found bool
	for _, m := range e.Session().History {
		if m.Role == ce.RoleAssistant && strings.Contains(m.Text, "Pronto") {
			found = true
		}
	}
	if !found {
		t.Error("the answer must still be recorded")
	}
	// And it never arrives as an answer: the two are different events, and a
	// client that confused them would show thinking as something the model
	// said out loud.
	if d, ok := rec.last[protocol.EventMessageDelta].(protocol.MessageDelta); ok {
		if strings.Contains(d.Text, "deleting everything") {
			t.Errorf("reasoning was streamed as an answer: %q", d.Text)
		}
	}
}

// Thinking runs five to ten times the size of the answer on a tool-calling
// turn, and it shares the session's event budget with everything worth
// replaying — so it has to be switchable.
func TestReasoningIsNotForwardedWhenSwitchedOff(t *testing.T) {
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{
			{Type: provider.EventReasoningDelta, Text: "pensando alto"},
			{Type: provider.EventTextDelta, Text: "pronto"},
			done(),
		},
	}}
	e, rec := newEngine(t, p, tools.NewRegistry(), func(c *Config) { c.ShowReasoning = false })
	if _, err := e.Run(context.Background(), "oi"); err != nil {
		t.Fatal(err)
	}
	if rec.count(protocol.EventMessageReasoning) != 0 {
		t.Error("nothing should have been forwarded")
	}
	// And the turn is otherwise unchanged.
	if d, ok := rec.last[protocol.EventMessageDelta].(protocol.MessageDelta); !ok || d.Text != "pronto" {
		t.Errorf("the answer must still arrive: %+v", rec.last[protocol.EventMessageDelta])
	}
}

// RN-3 makes emission order and execution order deliberately different: results
// are appended by emission index so history stays reproducible, while the calls
// themselves run together. The real order was computed inline, folded into a
// duration, and discarded — and a duration cannot answer which of two
// concurrent calls actually went first, which is the question a log is for.
func TestTheRealExecutionOrderSurvivesTheBatch(t *testing.T) {
	reg := tools.NewRegistry(tools.Read{}, tools.Glob{})
	e, _ := newEngine(t, &scriptedProvider{turns: [][]provider.StreamEvent{
		{
			call("c1", "glob", `{"pattern":"**/*.go"}`),
			call("c2", "glob", `{"pattern":"**/*.md"}`),
			done(),
		},
		{text("done"), done()},
	}}, reg)

	if _, err := e.Run(context.Background(), "look around"); err != nil {
		t.Fatal(err)
	}

	timing := e.Timing()
	if len(timing) != 2 {
		t.Fatalf("timing has %d entries, want one per call", len(timing))
	}
	for i, pair := range timing {
		if pair[0].IsZero() || pair[1].IsZero() {
			t.Errorf("call %d has no start or finish recorded", i)
		}
		if pair[1].Before(pair[0]) {
			t.Errorf("call %d finished before it started", i)
		}
	}
}

// ---------- what the protocol promises about event order ----------

// A blocked session is waiting on a person, and nothing is being generated
// while it waits. A delta arriving in that window would be text produced past
// an unanswered boundary question — the answer the user has not given yet,
// already acted upon.
//
// Asserted from INSIDE the approver, which is the only moment the session is
// genuinely blocked. Checking afterwards would ask a different question: by
// then the turn has resumed and any delta is legitimate.
func TestNothingIsStreamedWhileTheTurnIsBlocked(t *testing.T) {
	reg := tools.NewRegistry(slowTool{name: "reach", delay: 0, path: "/etc/hosts", write: true})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{text("about to reach out"), call("c1", "reach", `{}`), done()},
		{text("done"), done()},
	}}

	var rec *recorder
	var duringBlock int
	ap := &funcApprover{fn: func() protocol.ApprovalDecision {
		before := rec.count(protocol.EventMessageDelta)
		// Long enough that a streaming goroutine would have produced something
		// if one were running; the point is that none is.
		time.Sleep(50 * time.Millisecond)
		duringBlock = rec.count(protocol.EventMessageDelta) - before
		return protocol.ApprovalAllow
	}}
	e, r := newEngine(t, p, reg, func(c *Config) { c.Approver = ap })
	rec = r

	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}
	if duringBlock != 0 {
		t.Errorf("%d message.delta events were emitted while the turn was blocked", duringBlock)
	}
	// The guard is only meaningful if deltas happen at all in this turn.
	if rec.count(protocol.EventMessageDelta) == 0 {
		t.Fatal("no delta was emitted in the whole turn; the assertion above proved nothing")
	}
}

// An approval nobody answers must resolve exactly once, and to deny. Once,
// because a second resolution is a second decision recorded for one question;
// deny, because the alternative to denying is granting because everyone looked
// away.
func TestAnApprovalNobodyAnswersResolvesOnceAndDenies(t *testing.T) {
	reg := tools.NewRegistry(slowTool{name: "reach", delay: 0, path: "/etc/hosts", write: true})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "reach", `{}`), done()},
		{text("understood"), done()},
	}}
	// Stands in for the deadline passing: the session's own timeout answers
	// deny, and this is what the loop sees when it does.
	ap := &funcApprover{fn: func() protocol.ApprovalDecision { return protocol.ApprovalDeny }}
	e, rec := newEngine(t, p, reg, func(c *Config) { c.Approver = ap })

	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}
	if got := rec.count(protocol.EventApprovalResolved); got != 1 {
		t.Errorf("%d resolutions recorded for one question, want exactly 1", got)
	}
	res, _ := rec.last[protocol.EventApprovalResolved].(protocol.ApprovalResolved)
	if res.Decision != protocol.ApprovalDeny {
		t.Errorf("decision = %q, want deny — with nobody answering, granting is granting because everyone looked away", res.Decision)
	}
}

// turn.completed is the last word on a turn. An event carrying the same turn_id
// after it is a client rendering into a turn it has already closed, and the
// user sees output attached to work they were told had finished.
func TestNoEventCarriesACompletedTurnIDAfterItCompleted(t *testing.T) {
	reg := tools.NewRegistry(slowTool{name: "one", delay: 0, path: "a"})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "one", `{}`), done()},
		{text("finished"), done()},
	}}
	e, rec := newEngine(t, p, reg)

	out, err := e.Run(context.Background(), "go")
	if err != nil {
		t.Fatal(err)
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	completed := -1
	for i, t := range rec.events {
		if t == protocol.EventTurnCompleted {
			completed = i
		}
	}
	if completed < 0 {
		t.Fatal("the turn never completed; there is nothing to assert about what follows")
	}
	if completed != len(rec.events)-1 {
		t.Errorf("%d events follow turn.completed for %s: %v",
			len(rec.events)-completed-1, out.TurnID, rec.events[completed+1:])
	}
}

// funcApprover answers with whatever the test decides at the moment of asking,
// which is what lets an assertion run while the turn is genuinely blocked.
type funcApprover struct {
	fn    func() protocol.ApprovalDecision
	asked int
}

func (a *funcApprover) Approve(context.Context, protocol.ApprovalRequest) (protocol.ApprovalDecision, error) {
	a.asked++
	return a.fn(), nil
}
