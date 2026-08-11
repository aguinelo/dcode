package loop

import (
	"context"
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/protocol"
)

// doneEngine builds an engine with a definition of done and a scripted runner.
func doneEngine(t *testing.T, written []string, codes ...map[string]int) *Engine {
	t.Helper()
	call := 0
	e := New(Config{
		DoneEnabled:    true,
		MaxStallCycles: 2,
		Done: DoneSet{
			Criteria:  []Criterion{{Name: "tests", Command: "make test"}},
			Protected: []string{"**/*_test.go"},
		},
		WrittenPaths: func() []string { return written },
		RunCriterion: func(context.Context, string) (int, string, error) {
			c := codes[min(call, len(codes)-1)]
			call++
			return c["make test"], "", nil
		},
	}, ce.Session{Instructions: "x"})
	return e
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// A read-only turn runs no check at all. "Always run the tests" burns a turn
// answering "what does this function do", and two weeks of that is an
// uninstalled tool.
func TestATurnThatChangedNothingRunsNoCheck(t *testing.T) {
	ran := false
	e := New(Config{
		DoneEnabled:  true,
		Done:         DoneSet{Criteria: []Criterion{{Name: "tests", Command: "make test"}}},
		WrittenPaths: func() []string { return nil },
		RunCriterion: func(context.Context, string) (int, string, error) {
			ran = true
			return 0, "", nil
		},
	}, ce.Session{Instructions: "x"})

	stall, unmet := 0, []string(nil)
	reason, more := e.checkDone(context.Background(), &stall, &unmet)
	if ran {
		t.Error("a read-only turn ran the verification")
	}
	if more != nil || reason != protocol.StopDone {
		t.Errorf("read-only turn gave (%q, %v), want it to end cleanly", reason, more)
	}
}

func TestAMetCriterionEndsTheTurn(t *testing.T) {
	e := doneEngine(t, []string{"a.go"}, map[string]int{"make test": 0})
	stall, unmet := 0, []string(nil)
	reason, more := e.checkDone(context.Background(), &stall, &unmet)
	if more != nil {
		t.Fatal("the turn was continued although everything passed")
	}
	if reason != protocol.StopDone {
		t.Errorf("reason = %q, want done", reason)
	}
}

func TestAnUnmetCriterionContinuesTheTurnWithAReminder(t *testing.T) {
	e := doneEngine(t, []string{"a.go"}, map[string]int{"make test": 1})
	stall, unmet := 0, []string(nil)
	reason, more := e.checkDone(context.Background(), &stall, &unmet)
	if more == nil {
		t.Fatalf("the turn ended with an unmet criterion; reason %q", reason)
	}
	if !more[0].Reminder {
		t.Error("the continuation message is not marked as a reminder, so a client would render it as the user speaking")
	}
}

// Without a stall limit, a project whose check never passes spins to the
// iteration ceiling.
func TestNoProgressEndsTheTurnAsIncomplete(t *testing.T) {
	e := doneEngine(t, []string{"a.go"}, map[string]int{"make test": 1})
	stall, unmet := 0, []string(nil)

	// First check: nothing to compare against, so it continues.
	if _, more := e.checkDone(context.Background(), &stall, &unmet); more == nil {
		t.Fatal("the first unmet check ended the turn")
	}
	// Two more cycles with the same unmet set and no progress.
	e.checkDone(context.Background(), &stall, &unmet)
	reason, more := e.checkDone(context.Background(), &stall, &unmet)
	if more != nil {
		t.Fatalf("the turn is still going after %d stalled cycles", stall)
	}
	if reason != protocol.StopIncomplete {
		t.Errorf("reason = %q, want incomplete — a result, not an error", reason)
	}
}

// A protected path written during the turn is surfaced, never counted as
// progress in silence. Without this the whole loop is theatre.
func TestWritingATestFileIsSurfaced(t *testing.T) {
	e := doneEngine(t, []string{"internal/loop/turn_test.go"}, map[string]int{"make test": 1})
	stall, unmet := 0, []string(nil)
	_, more := e.checkDone(context.Background(), &stall, &unmet)
	if more == nil {
		t.Fatal("expected the turn to continue")
	}
	var joined string
	for _, m := range more {
		joined += m.Text
	}
	if !containsAll(joined, "turn_test.go", "checked") {
		t.Fatalf("changing what measures the work was not surfaced:\n%s", joined)
	}
	if got := e.Report().TouchedProtected; len(got) != 1 {
		t.Errorf("report carries %v, want the protected path", got)
	}
}

func TestDoneDisabledEndsTheTurnAsBefore(t *testing.T) {
	e := doneEngine(t, []string{"a.go"}, map[string]int{"make test": 1})
	e.cfg.DoneEnabled = false
	stall, unmet := 0, []string(nil)
	if reason, more := e.checkDone(context.Background(), &stall, &unmet); more != nil || reason != protocol.StopDone {
		t.Fatal("switching the definition of done off did not restore the old behaviour")
	}
}

func TestTheSealFollowsTheLastCheck(t *testing.T) {
	e := doneEngine(t, []string{"a.go"}, map[string]int{"make test": 0})
	stall, unmet := 0, []string(nil)
	e.checkDone(context.Background(), &stall, &unmet)
	if got := e.Verification(); got != VerificationPassed {
		t.Errorf("seal = %v, want passed", got)
	}

	e2 := doneEngine(t, []string{"a.go"}, map[string]int{"make test": 1})
	stall, unmet = 0, nil
	e2.checkDone(context.Background(), &stall, &unmet)
	if got := e2.Verification(); got != VerificationFailed {
		t.Errorf("seal = %v, want failed", got)
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
