package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// A session the loop moved into that cannot be looked up is said, and the
// loop is still followed to its end: a note from the follower must not end
// the following.
func TestALoopSessionThatCannotBeReadIsSaidAndTheLoopFollowed(t *testing.T) {
	p, tr := loopProgram(t,
		protocol.SpecFolder{Path: "specs/a", Criteria: 1, Unmet: 1, Pending: true},
		protocol.SpecFolder{Path: "specs/b", Criteria: 1, Unmet: 1, Pending: true},
	)
	tr.getFails = 1
	typeLoop(t, p, "implemente tudo")
	said := false
	followLoop(t, p, func() bool {
		said = said || sawEntry(p, "could not open the loop's session")
		return said && sawEntry(p, "the loop is finished")
	})
}

// A message queued on a loop's session is not sent there while the loop
// lives: the daemon is about to leave that folder, and a turn started now
// would be a second writer in the tree with a reply nobody sees.
func TestAQueuedMessageIsNotSentIntoALoopsSession(t *testing.T) {
	p, tr := loopProgram(t, protocol.SpecFolder{Path: "specs/a", Criteria: 1, Unmet: 1, Pending: true})
	tr.holdTurns = make(chan struct{})
	typeLoop(t, p, "implemente tudo")
	followLoop(t, p, func() bool { return sawEntry(p, "specs/a") && p.opts.SessionID != "s1" })

	p.model, _ = p.model.Enqueue("e o footer?", 3)
	tr.events <- ev(t, 99, protocol.EventTurnCompleted, protocol.TurnCompleted{TurnID: "t"})
	followLoop(t, p, func() bool { return applied(p, 99) })
	settle(t, p)
	for _, s := range tr.submits() {
		if strings.Contains(s, "e o footer?") {
			t.Fatalf("the queued message went into the loop's session: %q", tr.submits())
		}
	}
	close(tr.holdTurns)
	followLoop(t, p, nil)
}

// The queue goes with the person when the loop moves on, and says so; it is
// sent when the loop ends.
func TestTheQueueGoesWithTheLoopAndIsSaid(t *testing.T) {
	p, tr := loopProgram(t,
		protocol.SpecFolder{Path: "specs/a", Criteria: 1, Unmet: 1, Pending: true},
		protocol.SpecFolder{Path: "specs/b", Criteria: 1, Unmet: 1, Pending: true},
	)
	tr.holdTurns = make(chan struct{})
	typeLoop(t, p, "implemente tudo")
	followLoop(t, p, func() bool { return sawEntry(p, "specs/a") && p.opts.SessionID != "s1" })
	p.model, _ = p.model.Enqueue("e o footer?", 3)

	close(tr.holdTurns)
	followLoop(t, p, func() bool { return sawEntry(p, "1 queued message", "specs/b") })
	if len(p.model.Queue) != 1 {
		t.Errorf("the queue was dropped when the loop moved on: %v", p.model.Queue)
	}
	followLoop(t, p, func() bool {
		for _, s := range tr.submits() {
			if strings.Contains(s, "e o footer?") {
				return true
			}
		}
		return false
	})
}

// The proposal note names the command that goes on with the loop: for a goal
// that is the sentence, not the folder.
func TestTheProposalNoteNamesTheCommandThatGoesOn(t *testing.T) {
	p, tr := loopProgram(t,
		protocol.SpecFolder{Path: "specs/b", Criteria: 0, Pending: true},
	)
	tr.committed = protocol.CommitDoneResponse{Path: "specs/b/done.toml", Criteria: 1, Summary: "written"}
	typeLoop(t, p, "implemente o que falta")
	followLoop(t, p, func() bool { return sawEntry(p, "specs/b/done.toml") })
	if !sawEntry(p, "specs/b/done.toml", "/loop implemente o que falta") {
		t.Errorf("the note does not name the command that continues: %q", summaries(p))
	}
}

// Switching to another session on purpose stops following the loop: its
// next session does not pull the person back.
func TestSwitchingAwayStopsFollowingTheLoop(t *testing.T) {
	p, tr := loopProgram(t,
		protocol.SpecFolder{Path: "specs/a", Criteria: 1, Unmet: 1, Pending: true},
		protocol.SpecFolder{Path: "specs/b", Criteria: 1, Unmet: 1, Pending: true},
	)
	tr.holdTurns = make(chan struct{})
	typeLoop(t, p, "implemente tudo")
	followLoop(t, p, func() bool { return sawEntry(p, "specs/a") && p.opts.SessionID != "s1" })

	p.Update(switchedMsg{session: protocol.Session{ID: "elsewhere", Workspace: "/w"}})
	close(tr.holdTurns)
	followLoop(t, p, func() bool { return tr.lastLoop(t).State == protocol.LoopFinished })
	settle(t, p)
	if p.opts.SessionID != "elsewhere" {
		t.Errorf("the loop pulled the person back to %s", p.opts.SessionID)
	}
}

// A follow that was replaced lets its waiting reader go, and a stream that
// fails ends the follow, said.
func TestAReplacedOrFailedFollowLeavesNothingWaiting(t *testing.T) {
	p, tr := loopProgram(t, protocol.SpecFolder{Path: "specs/a", Criteria: 1, Unmet: 1, Pending: true})
	tr.holdTurns = make(chan struct{})
	typeLoop(t, p, "implemente tudo")
	followLoop(t, p, func() bool { return p.loop != nil })

	wait := p.waitForLoop()
	done := make(chan struct{})
	go func() { wait(); close(done) }()
	p.stopFollowingLoop()
	select {
	case <-done:
	case <-time.After(loopWait):
		t.Fatalf("a replaced follow left its reader waiting %s", loopWait)
	}

	typeLoop(t, p, "implemente tudo")
	followLoop(t, p, func() bool { return p.loop != nil })
	tr.failLoopStream(t, p, errors.New("the daemon went away"))
	followLoop(t, p, func() bool { return sawEntry(p, "the daemon went away") })
	followLoop(t, p, func() bool { return p.loop == nil })
	close(tr.holdTurns)
}

// A stop that was not this client interrupting says so plainly, without
// claiming a turn was interrupted.
func TestAStopFromElsewhereIsSaidPlainly(t *testing.T) {
	p, tr := loopProgram(t, protocol.SpecFolder{Path: "specs/a", Criteria: 1, Unmet: 1, Pending: true})
	tr.holdTurns = make(chan struct{})
	typeLoop(t, p, "implemente tudo")
	followLoop(t, p, func() bool { return sawEntry(p, "specs/a") && p.opts.SessionID != "s1" })
	run := tr.lastLoop(t)
	if _, err := tr.StopLoop(p.ctx, run.ID, protocol.StopNow); err != nil {
		t.Fatal(err)
	}
	followLoop(t, p, func() bool { return sawEntry(p, "stopped") })
	if sawEntry(p, "interrupted") {
		t.Errorf("a stop from elsewhere claimed this client's turn was interrupted: %q", summaries(p))
	}
}
