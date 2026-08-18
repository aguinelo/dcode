package loop

import (
	"context"
	"strings"
	"sync"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
	"github.com/aguinelo/dcode/internal/tools"
)

// waiting is a queue of corrections a test fills and the engine drains.
type waiting struct {
	mu   sync.Mutex
	msgs []string
}

func (w *waiting) add(s ...string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.msgs = append(w.msgs, s...)
}

func (w *waiting) take() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.msgs) == 0 {
		return ""
	}
	out := w.msgs[0]
	w.msgs = w.msgs[1:]
	return out
}

func userSays(e *Engine, fragment string) *ce.Message {
	for i := range e.Session().History {
		m := &e.Session().History[i]
		if m.Role == ce.RoleUser && strings.Contains(m.Text, fragment) {
			return m
		}
	}
	return nil
}

// Watching it go wrong at round three of twenty and having two options — let it
// finish wrong, or kill it and lose everything it learned — is the daily
// friction this removes.
//
// A correction arrives while the turn runs and lands at the next round, as the
// person speaking. Not as a reminder: a reminder is the product talking, and
// filing the user's correction as one misattributes the most important thing
// anybody says during a turn.
func TestACorrectionArrivingMidTurnLandsAtTheNextRound(t *testing.T) {
	w := &waiting{}
	w.add("stop, do the other thing instead")

	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "read", `{"path":"a.go"}`), done()},
		{text("understood"), done()},
	}}
	e, rec := newEngine(t, p, tools.NewRegistry(tools.Read{}),
		func(c *Config) { c.Steer = w.take })

	out, err := e.Run(context.Background(), "start")
	if err != nil {
		t.Fatal(err)
	}
	if out.Reason != protocol.StopDone {
		t.Fatalf("stopped as %q", out.Reason)
	}

	got := userSays(e, "the other thing")
	if got == nil {
		t.Fatal("the correction never reached the conversation")
	}
	if got.Reminder {
		t.Error("the person's correction was filed as a product reminder")
	}
	if rec.count(protocol.EventTurnSteered) != 1 {
		t.Errorf("emitted %d steer events; a correction nobody can see happened invisibly",
			rec.count(protocol.EventTurnSteered))
	}
}

// The model reads the correction on the very next request, not eventually. A
// correction the model has not seen yet is a correction that has not happened.
func TestTheModelSeesTheCorrectionOnTheNextRequest(t *testing.T) {
	w := &waiting{}
	w.add("use tabs, not spaces")

	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "read", `{"path":"a.go"}`), done()},
		{text("ok"), done()},
	}}
	e, _ := newEngine(t, p, tools.NewRegistry(tools.Read{}),
		func(c *Config) { c.Steer = w.take })

	if _, err := e.Run(context.Background(), "start"); err != nil {
		t.Fatal(err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.seen) < 2 {
		t.Fatalf("only %d requests were made", len(p.seen))
	}
	var carried bool
	for _, m := range p.seen[1].Messages {
		if strings.Contains(m.Text, "use tabs") {
			carried = true
		}
	}
	if !carried {
		t.Error("the second request did not carry the correction")
	}
}

// With nothing to say, nothing changes. The puller runs every round, so a turn
// nobody steers has to be exactly the turn it was before this existed.
func TestATurnNobodySteersIsUnchanged(t *testing.T) {
	p := &scriptedProvider{turns: [][]provider.StreamEvent{{text("all done"), done()}}}
	e, rec := newEngine(t, p, tools.NewRegistry(),
		func(c *Config) { c.Steer = func() string { return "" } })

	before := len(e.Session().History)
	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}
	// The question and the answer, and nothing else.
	if got := len(e.Session().History) - before; got != 2 {
		t.Errorf("history grew by %d, want 2", got)
	}
	if rec.count(protocol.EventTurnSteered) != 0 {
		t.Error("an empty correction was announced")
	}
}

// No puller configured is the old behaviour exactly, and must not be a nil
// dereference in the hottest loop in the product.
func TestNoSteererIsTheOldBehaviour(t *testing.T) {
	p := &scriptedProvider{turns: [][]provider.StreamEvent{{text("fine"), done()}}}
	e, _ := newEngine(t, p, tools.NewRegistry())
	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}
}

// Corrections arrive in the order they were said. Two reordered is one
// instruction answering the wrong question.
func TestCorrectionsKeepTheirOrder(t *testing.T) {
	w := &waiting{}
	w.add("first", "second")

	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "read", `{"path":"a.go"}`), done()},
		{call("c2", "read", `{"path":"a.go"}`), done()},
		{text("ok"), done()},
	}}
	e, _ := newEngine(t, p, tools.NewRegistry(tools.Read{}),
		func(c *Config) { c.Steer = w.take })

	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}

	var order []string
	for _, m := range e.Session().History {
		if m.Role == ce.RoleUser && (m.Text == "first" || m.Text == "second") {
			order = append(order, m.Text)
		}
	}
	if len(order) != 2 || order[0] != "first" || order[1] != "second" {
		t.Errorf("got %v, want first then second", order)
	}
}
