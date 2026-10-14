package tui

import (
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// A goal over a backlog qualifies every pending folder that declares nothing,
// in one pass, and stops with the files written — one sitting to review them
// all — before any work runs (desktop/docs/proposals/loop.md §3.3).
//
// It worked them instead: the queue held every pending folder and opened a
// work session on one with no criteria, which reports done at the end of its
// first turn against a ruler nobody wrote.
func TestAGoalQualifiesEveryFolderThatDeclaresNothingBeforeAnyWork(t *testing.T) {
	p, tr := loopProgram(t,
		protocol.SpecFolder{Path: "specs/a", Criteria: 2, Unmet: 1, Pending: true},
		protocol.SpecFolder{Path: "specs/b", Criteria: 0, Pending: true},
		protocol.SpecFolder{Path: "specs/c", Criteria: 0, Pending: true},
	)
	tr.committed = protocol.CommitDoneResponse{Path: "done.toml", Criteria: 1, Summary: "written"}
	typeLoop(t, p, "implemente o que falta")
	followLoop(t, p, func() bool { return sawEntry(p, "/loop", "2") && tr.lastLoop(t).State == protocol.LoopAwaitingReview })

	if len(tr.created) != 2 {
		t.Fatalf("created %+v, want the two qualifying sessions only", tr.created)
	}
	for i, want := range []string{"specs/b", "specs/c"} {
		if got := tr.created[i]; got.LoopSpec != want || !got.Qualify {
			t.Fatalf("session %d is %+v, want a qualifying one on %s", i, got, want)
		}
	}
	last := p.model.Entries[len(p.model.Entries)-1].Summary
	if !strings.Contains(last, "2") || !strings.Contains(last, "/loop") {
		t.Errorf("the pass ended without saying what to review and how to go on: %q", last)
	}
}

// Typing the goal again after the review goes on with the same loop: every
// pending folder is worked, the reviewed ones included, and nothing is
// qualified twice.
func TestTypingTheLoopAgainAfterTheReviewGoesOn(t *testing.T) {
	p, tr := loopProgram(t,
		protocol.SpecFolder{Path: "specs/a", Criteria: 2, Unmet: 1, Pending: true},
		protocol.SpecFolder{Path: "specs/b", Criteria: 0, Pending: true},
	)
	tr.committed = protocol.CommitDoneResponse{Path: "done.toml", Criteria: 1, Summary: "written"}
	typeLoop(t, p, "implemente o que falta")
	followLoop(t, p, func() bool { return tr.lastLoop(t).State == protocol.LoopAwaitingReview })

	typeLoop(t, p, "implemente   o que falta")
	followLoop(t, p, nil)
	if run := tr.lastLoop(t); run.State != protocol.LoopFinished || len(tr.created) != 3 {
		t.Fatalf("run %s with sessions %+v; want the same loop worked to the end", run.State, tr.created)
	}
	for _, c := range tr.created[1:] {
		if c.Qualify {
			t.Errorf("a folder was qualified again after the review: %+v", c)
		}
	}
}

// Typing a different /loop while one waits for its review is refused, saying
// which one waits, rather than continuing a run the person did not mean.
func TestADifferentLoopWhileOneWaitsIsRefused(t *testing.T) {
	p, tr := loopProgram(t, protocol.SpecFolder{Path: "specs/x"})
	tr.committed = protocol.CommitDoneResponse{Path: "specs/x/done.toml", Criteria: 1, Summary: "written"}
	typeLoop(t, p, "specs/x")
	followLoop(t, p, func() bool { return tr.lastLoop(t).State == protocol.LoopAwaitingReview })

	typeLoop(t, p, "specs/y")
	followLoop(t, p, func() bool { return sawEntry(p, "/loop specs/x", "waiting") })
	if !sawEntry(p, "/loop specs/x", "waiting") {
		t.Errorf("a different loop was not refused naming the one waiting: %q", summaries(p))
	}
	if run := tr.lastLoop(t); run.State != protocol.LoopAwaitingReview || len(tr.created) != 1 {
		t.Errorf("the waiting loop moved: %s, %+v", run.State, tr.created)
	}
}

// Interrupting the turn of the person's own loop stops the loop, and says so:
// it used to skip to the next folder, and the daemon would otherwise fail the
// run on a turn that ended interrupted.
func TestInterruptingALoopsTurnStopsTheLoop(t *testing.T) {
	p, tr := loopProgram(t,
		protocol.SpecFolder{Path: "specs/a", Criteria: 1, Unmet: 1, Pending: true},
		protocol.SpecFolder{Path: "specs/b", Criteria: 1, Unmet: 1, Pending: true},
	)
	tr.holdTurns = make(chan struct{})
	typeLoop(t, p, "implemente tudo")
	followLoop(t, p, func() bool { return p.loop != nil && p.loop.session != "" })

	cmd := p.interrupt()
	if cmd == nil {
		t.Fatal("interrupting gave nothing to do")
	}
	cmd()
	followLoop(t, p, nil)
	if run := tr.lastLoop(t); run.State != protocol.LoopStopped {
		t.Errorf("the loop is %s after its turn was interrupted, want stopped", run.State)
	}
	if !sawEntry(p, "stopped") {
		t.Errorf("the stop was not said: %q", summaries(p))
	}
	for _, c := range tr.created {
		if c.LoopSpec == "specs/b" {
			t.Error("the next folder opened after the interrupt")
		}
	}
}
