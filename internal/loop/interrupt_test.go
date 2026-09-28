package loop

import (
	"context"
	"fmt"
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
	"github.com/aguinelo/dcode/internal/tools"
)

// An interrupt that landed as a final answer ended was read as work nobody
// could verify.
//
// A stream can end `done` at the very instant the person presses stop: the
// provider's pump still decodes a terminal frame it took before the cancel, a
// window it can narrow and not close. With no calls in the answer, the loop went
// straight to the definition of done under the cancelled context. No criterion
// can start there — `exec.Cmd.Start` answers a cancelled context with its error
// before any process exists — so each one read as unavailable. The model was
// told its work could not be verified, a round was spent telling it, and the
// seal in `turn.completed` reported the criteria as unavailable. None of it was
// true: the person had stopped the turn.
//
// The rule these tests hold the loop to: a stop visible when the answer ends is
// the end of the turn. No criterion starts, and nothing about a check that never
// ran reaches the model or the seal.

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

// ---------- the rule ----------

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
