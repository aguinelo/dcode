package tui

import (
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

func submitted(tr *fakeTransport, text string) bool {
	for _, s := range tr.submits() {
		if strings.Contains(s, text) {
			return true
		}
	}
	return false
}

// After switching away, typing the same /loop again follows the run that is
// working — it does not try to start a second one and get refused.
func TestTypingTheLoopAgainFollowsAWorkingRun(t *testing.T) {
	p, tr := loopProgram(t, protocol.SpecFolder{Path: "specs/a", Criteria: 1, Unmet: 1, Pending: true})
	tr.holdTurns = make(chan struct{})
	typeLoop(t, p, "implemente tudo")
	followLoop(t, p, func() bool { return sawEntry(p, "specs/a") && p.opts.SessionID != "s1" })
	loopSession := p.opts.SessionID

	p.Update(switchedMsg{session: protocol.Session{ID: "elsewhere", Workspace: "/w"}})
	typeLoop(t, p, "implemente   tudo")
	followLoop(t, p, func() bool { return p.loop != nil || sawEntry(p, "could not start") })
	if sawEntry(p, "could not start") {
		t.Fatalf("typing the loop again was refused: %q", summaries(p))
	}
	close(tr.holdTurns)
	followLoop(t, p, nil)
	if run := tr.lastLoop(t); run.State != protocol.LoopFinished || len(tr.created) != 1 {
		t.Errorf("a second loop was started, or the first did not end: %s, %+v", run.State, tr.created)
	}
	_ = loopSession
}

// A loop stream that closes ends the follow, and says so.
func TestALoopStreamThatClosesEndsTheFollowSaid(t *testing.T) {
	p, tr := loopProgram(t, protocol.SpecFolder{Path: "specs/a", Criteria: 1, Unmet: 1, Pending: true})
	tr.holdTurns = make(chan struct{})
	tr.watchFrames = make(chan protocol.LoopChange)
	typeLoop(t, p, "implemente tudo")
	followLoop(t, p, func() bool { return p.loop != nil })
	close(tr.watchFrames)
	followLoop(t, p, func() bool { return sawEntry(p, "could not be followed") && p.loop == nil })
	close(tr.holdTurns)
}

// Text typed while the loop's session sits idle between folders is held, not
// sent: the daemon is about to leave that folder.
func TestTextTypedBetweenFoldersIsHeld(t *testing.T) {
	p, tr := loopProgram(t, protocol.SpecFolder{Path: "specs/a", Criteria: 1, Unmet: 1, Pending: true})
	tr.holdTurns = make(chan struct{})
	typeLoop(t, p, "implemente tudo")
	followLoop(t, p, func() bool { return sawEntry(p, "specs/a") && p.opts.SessionID != "s1" })

	p.model = p.model.SetInput("e o footer?")
	_, cmd := p.onEnter()
	deliver(p, cmd)
	runFor(t, p, 200*1e6)
	if submitted(tr, "e o footer?") {
		t.Fatalf("text typed between folders was sent into the folder being left: %q", tr.submits())
	}
	if len(p.model.Queue) != 1 || !sawEntry(p, "held") {
		t.Errorf("the text was not held, or not said: queue %v, %q", p.model.Queue, summaries(p))
	}
	close(tr.holdTurns)
	followLoop(t, p, nil)
}

// A loop that ends stopped keeps what it held, and says how to send it: after
// a stop, nothing fires on its own.
func TestAStoppedLoopKeepsTheQueueAndSaysSo(t *testing.T) {
	p, tr := loopProgram(t, protocol.SpecFolder{Path: "specs/a", Criteria: 1, Unmet: 1, Pending: true})
	tr.holdTurns = make(chan struct{})
	typeLoop(t, p, "implemente tudo")
	followLoop(t, p, func() bool { return sawEntry(p, "specs/a") && p.opts.SessionID != "s1" })
	p.model, _ = p.model.Enqueue("e o footer?", 3)

	if _, err := tr.StopLoop(p.ctx, tr.lastLoop(t).ID, protocol.StopNow); err != nil {
		t.Fatal(err)
	}
	followLoop(t, p, func() bool { return sawEntry(p, "left in the queue") })
	runFor(t, p, 200*1e6)
	if submitted(tr, "e o footer?") || len(p.model.Queue) != 1 {
		t.Fatalf("a stopped loop sent what it held: %q, queue %v", tr.submits(), p.model.Queue)
	}

	// ↵ on an empty line sends it.
	_, cmd := p.onEnter()
	deliver(p, cmd)
	followLoop(t, p, func() bool { return submitted(tr, "e o footer?") })
}
