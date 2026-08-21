package loop

import (
	"context"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
	"github.com/aguinelo/dcode/internal/tools"
)

func progressOf(rec *recorder, kind string) []protocol.Progress {
	var out []protocol.Progress
	rec.mu.Lock()
	defer rec.mu.Unlock()
	for _, p := range rec.all {
		if pr, ok := p.(protocol.Progress); ok && pr.Kind == kind {
			out = append(out, pr)
		}
	}
	return out
}

// The turn says where it is against its ceiling, and the ceiling rides with the
// count. A count without its limit answers "how many" when the question is
// "how close", and a client carrying the limit separately would be carrying a
// copy of configuration it cannot see change.
func TestATurnReportsItsRoundAgainstItsCeiling(t *testing.T) {
	reg := tools.NewRegistry(tools.Read{})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "read", `{"path":"x"}`), done()},
		{text("done"), done()},
	}}
	e, rec := newEngine(t, p, reg)

	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}

	got := progressOf(rec, protocol.ProgressRounds)
	if len(got) == 0 {
		t.Fatal("the turn never said which round it was on")
	}
	for i, pr := range got {
		if pr.Total != e.cfg.Limits.MaxIterations {
			t.Errorf("round %d carries ceiling %d, want %d", i, pr.Total, e.cfg.Limits.MaxIterations)
		}
		if pr.Done != i+1 {
			t.Errorf("round %d reported Done=%d", i+1, pr.Done)
		}
		if pr.ToolCallID != "" {
			t.Errorf("the turn's own progress carries a tool call id: %q", pr.ToolCallID)
		}
	}
}

// How many are about to run together, against what this session allows. It is
// emitted where the number exists — inside a group every call would report the
// same figure, and after it the number is already history.
func TestABatchReportsHowManyRunTogether(t *testing.T) {
	reg := tools.NewRegistry(tools.Read{})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "read", `{"path":"a"}`), call("c2", "read", `{"path":"b"}`), done()},
		{text("done"), done()},
	}}
	e, rec := newEngine(t, p, reg)

	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}

	got := progressOf(rec, protocol.ProgressInFlight)
	if len(got) == 0 {
		t.Fatal("a batch never said how many ran together")
	}
	for _, pr := range got {
		if pr.Total != e.cfg.Parallel {
			t.Errorf("in flight carries ceiling %d, want %d", pr.Total, e.cfg.Parallel)
		}
		if pr.Done < 1 || pr.Done > pr.Total {
			t.Errorf("in flight reported %d of %d", pr.Done, pr.Total)
		}
	}
}

// A turn that answered in one pass reports no round, and that is the contract
// rather than a gap.
//
// Iterations counts cycles the loop went round AGAIN — a turn that answered and
// passed the done-check returns without incrementing. There is no ceiling
// approaching, so there is no number to show, and emitting 0 of 100 would put a
// figure on screen that means nothing is happening.
func TestATurnThatAnsweredInOnePassReportsNoRound(t *testing.T) {
	reg := tools.NewRegistry()
	p := &scriptedProvider{turns: [][]provider.StreamEvent{{text("just an answer"), done()}}}
	e, rec := newEngine(t, p, reg)

	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}
	if got := progressOf(rec, protocol.ProgressRounds); len(got) != 0 {
		t.Errorf("a turn that never looped reported %d rounds: %+v", len(got), got)
	}
}

// Every kind emitted is one the protocol declares. A kind invented at the point
// of emission is a word on a versioned surface that nothing documents, and the
// client would render it as an unknown counter rather than as nothing.
func TestEveryKindEmittedIsOneTheProtocolDeclares(t *testing.T) {
	known := map[string]bool{
		protocol.ProgressRounds:   true,
		protocol.ProgressInFlight: true,
	}
	reg := tools.NewRegistry(tools.Read{})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "read", `{"path":"a"}`), done()},
		{text("done"), done()},
	}}
	e, rec := newEngine(t, p, reg)
	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	for _, pl := range rec.all {
		pr, ok := pl.(protocol.Progress)
		if !ok {
			continue
		}
		if !known[pr.Kind] {
			t.Errorf("emitted an undeclared kind %q", pr.Kind)
		}
	}
}

// Progress never reaches the model. It is a person's window — the same rule
// StartedAt already carries — and a count that differs between two runs of one
// session is exactly what ADR-03 forbids in a prefix.
func TestProgressNeverEntersTheContextSentToTheModel(t *testing.T) {
	reg := tools.NewRegistry(tools.Read{})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "read", `{"path":"a"}`), done()},
		{text("done"), done()},
	}}
	e, _ := newEngine(t, p, reg)
	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}

	for _, m := range e.Session().History {
		for _, word := range []string{protocol.ProgressRounds, protocol.ProgressInFlight} {
			if strings.Contains(m.Text, word) {
				t.Errorf("progress reached the model's history: %q", m.Text)
			}
		}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, req := range p.seen {
		for _, m := range req.Messages {
			if strings.Contains(m.Text, protocol.ProgressRounds) {
				t.Errorf("progress was sent to the provider: %q", m.Text)
			}
		}
	}
}
