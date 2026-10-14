package loop

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
	"github.com/aguinelo/dcode/internal/tools"
)

// checks returns every done.checked the recorder saw, in emission order.
func checks(r *recorder) []protocol.DoneChecked {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []protocol.DoneChecked
	for i, t := range r.events {
		if t == protocol.EventDoneChecked {
			out = append(out, r.all[i].(protocol.DoneChecked))
		}
	}
	return out
}

// Every check of the criteria inside a turn is an event, numbered within the
// turn and carrying the stall counter against its ceiling.
//
// The re-entry used to happen in silence: a client learned where the criteria
// stood only at turn.completed, so "cycle 3, two of five" was not on the wire.
func TestEachCheckInsideATurnIsAnEvent(t *testing.T) {
	reg := tools.NewRegistry(slowTool{name: "one", delay: 0, path: "a"})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "one", `{}`), done()},
		{text("all set"), done()},
		{text("still all set"), done()},
		{text("really"), done()},
	}}
	e, rec := newEngine(t, p, reg, func(c *Config) {
		c.DoneEnabled = true
		c.MaxStallCycles = 2
		c.Done = DoneSet{Criteria: []Criterion{{Name: "verify", Command: "make check"}}}
		c.WrittenPaths = func() []string { return []string{"a.go"} }
		c.RunCriterion = func(context.Context, string) (int, string, error) { return 1, "", nil }
	})

	out, err := e.Run(context.Background(), "go")
	if err != nil {
		t.Fatal(err)
	}
	if out.Reason != protocol.StopIncomplete {
		t.Fatalf("reason = %q, want incomplete", out.Reason)
	}

	got := checks(rec)
	want := []protocol.DoneChecked{
		{TurnID: "t1", Cycle: 1, Unmet: []string{"verify"}, Moved: protocol.DoneMovedFirst, Stall: 0, StallLimit: 2},
		{TurnID: "t1", Cycle: 2, Unmet: []string{"verify"}, Moved: protocol.DoneMovedStalled, Stall: 1, StallLimit: 2},
		{TurnID: "t1", Cycle: 3, Unmet: []string{"verify"}, Moved: protocol.DoneMovedStalled, Stall: 2, StallLimit: 2},
	}
	if len(got) != len(want) {
		t.Fatalf("%d done.checked events, want one per check (%d): %+v", len(got), len(want), got)
	}
	for i := range want {
		if !sameCheck(got[i], want[i]) {
			t.Errorf("check %d = %+v, want %+v", i+1, got[i], want[i])
		}
	}

	// Every check happens before the turn ends, never after it.
	last := slices.Index(rec.events, protocol.EventTurnCompleted)
	for i, ev := range rec.events {
		if ev == protocol.EventDoneChecked && i > last {
			t.Error("a done.checked was emitted after turn.completed")
		}
	}
}

// A check that closes a criterion moves forward and resets the stall count,
// and the next turn counts its checks from one again.
func TestACheckThatClosesACriterionMovesForward(t *testing.T) {
	reg := tools.NewRegistry(slowTool{name: "one", delay: 0, path: "a"})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "one", `{}`), done()},
		{text("first try"), done()},
		{text("fixed"), done()},
		{call("c2", "one", `{}`), done()},
		{text("again"), done()},
	}}
	runs := 0
	e, rec := newEngine(t, p, reg, func(c *Config) {
		c.DoneEnabled = true
		c.MaxStallCycles = 3
		c.Done = DoneSet{Criteria: []Criterion{{Name: "verify", Command: "make check"}}}
		c.WrittenPaths = func() []string { return []string{"a.go"} }
		c.RunCriterion = func(context.Context, string) (int, string, error) {
			runs++
			if runs == 1 {
				return 1, "", nil
			}
			return 0, "", nil
		}
	})

	for _, input := range []string{"go", "and again"} {
		if _, err := e.Run(context.Background(), input); err != nil {
			t.Fatal(err)
		}
	}

	got := checks(rec)
	want := []protocol.DoneChecked{
		{TurnID: "t1", Cycle: 1, Unmet: []string{"verify"}, Moved: protocol.DoneMovedFirst, StallLimit: 3},
		{TurnID: "t1", Cycle: 2, Met: []string{"verify"}, Moved: protocol.DoneMovedForward, StallLimit: 3},
		{TurnID: "t2", Cycle: 1, Met: []string{"verify"}, Moved: protocol.DoneMovedFirst, StallLimit: 3},
	}
	if len(got) != len(want) {
		t.Fatalf("%d done.checked events, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if !sameCheck(got[i], want[i]) {
			t.Errorf("check %d = %+v, want %+v", i+1, got[i], want[i])
		}
	}
}

// Met, unmet and unavailable are three answers, carried apart: a criterion
// that could not run taught nothing, and showing it as failed would be a lie.
func TestACheckNamesWhatPassedFailedAndCouldNotRun(t *testing.T) {
	rec := newRecorder()
	e := New(Config{
		DoneEnabled: true, Emitter: rec, MaxStallCycles: 3,
		Done: DoneSet{
			Criteria: []Criterion{
				{Name: "unit", Command: "unit"},
				{Name: "e2e", Command: "e2e"},
				{Name: "a11y", Command: ""},
			},
			Protected: []string{"**/*_test.go"},
		},
		WrittenPaths: func() []string { return []string{"a.go", "a_test.go"} },
		RunCriterion: func(_ context.Context, cmd string) (int, string, error) {
			if cmd == "e2e" {
				return 1, "", nil
			}
			return 0, "", nil
		},
	}, ce.Session{Instructions: "x"})

	stall, unmet := 0, []string(nil)
	e.checkDone(context.Background(), &stall, &unmet, new(bool))

	got := checks(rec)
	if len(got) != 1 {
		t.Fatalf("%d done.checked events, want 1", len(got))
	}
	want := protocol.DoneChecked{
		Cycle: 1, Met: []string{"unit"}, Unmet: []string{"e2e"}, Unavailable: []string{"a11y"},
		Moved: protocol.DoneMovedFirst, StallLimit: 3, TouchedProtected: []string{"a_test.go"},
	}
	if !sameCheck(got[0], want) {
		t.Errorf("check = %+v, want %+v", got[0], want)
	}
}

// A turn that wrote nothing runs no check, and so announces none.
func TestATurnThatChangedNothingAnnouncesNoCheck(t *testing.T) {
	rec := newRecorder()
	e := New(Config{
		DoneEnabled: true, Emitter: rec,
		Done:         DoneSet{Criteria: []Criterion{{Name: "tests", Command: "make test"}}},
		WrittenPaths: func() []string { return nil },
		RunCriterion: func(context.Context, string) (int, string, error) { return 0, "", nil },
	}, ce.Session{Instructions: "x"})

	stall, unmet := 0, []string(nil)
	e.checkDone(context.Background(), &stall, &unmet, new(bool))
	if n := rec.count(protocol.EventDoneChecked); n != 0 {
		t.Errorf("%d done.checked for a turn that changed nothing, want none", n)
	}
}

// A check that broke a criterion says so, and names what the loop put back.
func TestABackwardCheckNamesWhatWasUndone(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "work.go")
	if err := os.WriteFile(file, []byte("before\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	st := stateFor(t, dir)
	rec := newRecorder()
	e := New(Config{
		DoneEnabled: true, State: st, Emitter: rec, MaxStallCycles: 3,
		Done:         DoneSet{Criteria: []Criterion{{Name: "tests", Command: "tests"}, {Name: "lint", Command: "lint"}}},
		WrittenPaths: st.Written,
		RunCriterion: func(context.Context, string) (int, string, error) { return 1, "", nil },
	}, ce.Session{Instructions: "x"})

	st.BeginTurn()
	st.BeginCycle()
	st.Snapshot(file)
	if err := os.WriteFile(file, []byte("after\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	st.MarkRead(file, "after\n", 0)
	st.MarkWritten(file)

	stall, unmet := 0, []string{"tests"}
	e.checkDone(context.Background(), &stall, &unmet, new(bool))

	got := checks(rec)
	if len(got) != 1 {
		t.Fatalf("%d done.checked events, want 1", len(got))
	}
	if got[0].Moved != protocol.DoneMovedBackward || got[0].Stall != 1 {
		t.Errorf("moved %q with stall %d, want backward and counted as a stall", got[0].Moved, got[0].Stall)
	}
	if !slices.Contains(got[0].Undone, file) {
		t.Errorf("undone = %v, want it to name %s", got[0].Undone, file)
	}
}

func sameCheck(a, b protocol.DoneChecked) bool {
	return a.TurnID == b.TurnID && a.Cycle == b.Cycle &&
		slices.Equal(a.Met, b.Met) && slices.Equal(a.Unmet, b.Unmet) &&
		slices.Equal(a.Unavailable, b.Unavailable) && a.Moved == b.Moved &&
		a.Stall == b.Stall && a.StallLimit == b.StallLimit &&
		slices.Equal(a.Undone, b.Undone) &&
		slices.Equal(a.TouchedProtected, b.TouchedProtected)
}
