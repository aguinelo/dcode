package loop

import (
	"context"
	"os"
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/protocol"
)

// RN-9 of agent-loop, which was typed, documented and unreachable: the value
// existed in the protocol and no code path ever returned it.
//
// Files changed and nothing could check them. Ending as StopDone is exactly the
// false claim the rule exists to prevent — the turn delivered work nobody
// confirmed, and said it was done.
func TestChangedWithNothingAbleToCheckEndsUnverified(t *testing.T) {
	e := New(Config{
		DoneEnabled:  true,
		Done:         DoneSet{Criteria: []Criterion{{Name: "verify", Command: ""}}},
		WrittenPaths: func() []string { return []string{"a.go"} },
		RunCriterion: nil, // nothing can run
	}, ce.Session{Instructions: "x"})

	stall, unmet := 0, []string(nil)
	reason, more := e.checkDone(context.Background(), &stall, &unmet)
	if more != nil {
		t.Fatal("an unavailable criterion forced another iteration; there is nothing to run and insisting only produces a second guess")
	}
	if reason != protocol.StopUnverified {
		t.Fatalf("reason = %q, want %q", reason, protocol.StopUnverified)
	}
	if got := e.Verification(); got != VerificationUnavailable {
		t.Errorf("seal = %v, want unavailable", got)
	}
}

func TestAPassingCheckStillEndsDone(t *testing.T) {
	e := New(Config{
		DoneEnabled:  true,
		Done:         DoneSet{Criteria: []Criterion{{Name: "verify", Command: "make check"}}},
		WrittenPaths: func() []string { return []string{"a.go"} },
		RunCriterion: func(context.Context, string) (int, string, error) { return 0, "", nil },
	}, ce.Session{Instructions: "x"})

	stall, unmet := 0, []string(nil)
	reason, _ := e.checkDone(context.Background(), &stall, &unmet)
	if reason != protocol.StopDone {
		t.Fatalf("reason = %q, want done — a check that ran and passed is the one case that may claim success", reason)
	}
}

// A read-only turn is not unverified. There was nothing to check, which is a
// different fact from having failed to check.
func TestAReadOnlyTurnIsNotUnverified(t *testing.T) {
	e := New(Config{
		DoneEnabled:  true,
		Done:         DoneSet{Criteria: []Criterion{{Name: "verify", Command: ""}}},
		WrittenPaths: func() []string { return nil },
	}, ce.Session{Instructions: "x"})

	stall, unmet := 0, []string(nil)
	if reason, _ := e.checkDone(context.Background(), &stall, &unmet); reason != protocol.StopDone {
		t.Fatalf("reason = %q, want done", reason)
	}
}

// RN-5: a history that lies about what happened on disk is worse than an
// incomplete one.
//
// The tools that write ignore the context on purpose, because a half-applied
// edit is worse than a slow one. `bash` is the case that does not, and a
// cancelled command can leave the disk changed while its result says it failed.
func TestAnInterruptedTurnRecordsWhatWasAlreadyWritten(t *testing.T) {
	e := New(Config{
		Reminders:    true,
		WrittenPaths: func() []string { return []string{"internal/a.go", "internal/b.go"} },
	}, ce.Session{Instructions: "x"})

	out := e.finishInterrupted(Outcome{TurnID: "t1"})
	if out.Reason != protocol.StopInterrupted {
		t.Fatalf("reason = %q, want interrupted", out.Reason)
	}

	var joined string
	for _, m := range e.session.History {
		if m.Reminder {
			joined += m.Text
		}
	}
	for _, want := range []string{"internal/a.go", "internal/b.go", "interrupted", "half-done"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the history does not carry %q, so it says nothing about the disk:\n%s", want, joined)
		}
	}
}

func TestAnInterruptedTurnThatWroteNothingSaysNothing(t *testing.T) {
	e := New(Config{
		Reminders:    true,
		WrittenPaths: func() []string { return nil },
	}, ce.Session{Instructions: "x"})

	e.finishInterrupted(Outcome{TurnID: "t1"})
	for _, m := range e.session.History {
		if m.Reminder {
			t.Fatalf("a turn that changed nothing left a note about the disk: %q", m.Text)
		}
	}
}

// The guard that would have caught this class in the first place.
//
// StopUnverified was declared in the protocol, documented in two specs, carried
// a comment explaining why it is a result rather than an error — and no code
// path returned it. Nothing failed, because a declared constant compiles
// whether or not anything reaches it.
//
// A stop reason nobody returns is a state the product claims to have and does
// not, which is the same defect class as a config key nobody reads.
func TestEveryStopReasonIsReachableFromTheLoop(t *testing.T) {
	src := loopSources(t)

	// StopError is returned from several places; the rest each have one site.
	for _, reason := range []string{
		"StopDone", "StopInterrupted", "StopMaxIterations", "StopRepeatLoop",
		"StopMaxTokens", "StopError", "StopUnverified", "StopIncomplete",
	} {
		if !strings.Contains(src, "protocol."+reason) {
			t.Errorf("protocol.%s is declared and no code in internal/loop returns it: "+
				"a stop reason nobody reaches is a state the product claims to have and does not", reason)
		}
	}
}

func loopSources(t *testing.T) string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		data, err := os.ReadFile(n)
		if err != nil {
			t.Fatal(err)
		}
		b.Write(data)
	}
	if b.Len() == 0 {
		t.Fatal("no loop sources read; the guard would pass vacuously")
	}
	return b.String()
}

// The seal must not say "passed" for a check that ran before the last edit.
//
// This is the shape the mechanism exists to catch, and the one it could not
// see: run the suite, watch it go green, then keep editing. Every input to the
// old seal still said met-and-changed, so it printed "passed" over code no
// command had ever looked at.
//
// Counted, not timed. A clock in the seal would put a varying value in
// something a person compares between turns; a generation number answers the
// only question being asked — did anything change after the check started.
func TestASealGoesStaleWhenTheCodeChangesAfterTheCheck(t *testing.T) {
	seq := uint64(1)
	e := New(Config{
		DoneEnabled:  true,
		Done:         DoneSet{Criteria: []Criterion{{Name: "verify", Command: "make check"}}},
		WrittenPaths: func() []string { return []string{"a.go"} },
		WriteSeq:     func() uint64 { return seq },
		RunCriterion: func(context.Context, string) (int, string, error) { return 0, "", nil },
	}, ce.Session{Instructions: "x"})

	stall, unmet := 0, []string(nil)
	if reason, _ := e.checkDone(context.Background(), &stall, &unmet); reason != protocol.StopDone {
		t.Fatalf("a passing check should end the turn, got %q", reason)
	}
	if got := e.Verification(); got != VerificationPassed {
		t.Fatalf("seal = %v, want passed — the check ran after the only edit", got)
	}

	// The model keeps working after the green run.
	seq++

	if got := e.Verification(); got != VerificationStale {
		t.Errorf("seal = %v, want stale — the check describes code that no longer exists", got)
	}
}

// Without a counter there is nothing to compare, and the honest answer is the
// one that claims least. Reporting "passed" because staleness is unknowable is
// how an absent signal turns into a positive one.
func TestWithoutAWriteCounterTheSealNeverClaimsPassed(t *testing.T) {
	e := New(Config{
		DoneEnabled:  true,
		Done:         DoneSet{Criteria: []Criterion{{Name: "verify", Command: "make check"}}},
		WrittenPaths: func() []string { return []string{"a.go"} },
		RunCriterion: func(context.Context, string) (int, string, error) { return 0, "", nil },
	}, ce.Session{Instructions: "x"})

	stall, unmet := 0, []string(nil)
	e.checkDone(context.Background(), &stall, &unmet)
	if got := e.Verification(); got != VerificationPassed {
		t.Errorf("seal = %v; with no counter wired the seal keeps its previous meaning", got)
	}
}
