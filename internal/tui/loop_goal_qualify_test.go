package tui

import (
	"context"
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
	tr := newFakeTransport()
	p := &program{
		ctx:   context.Background(),
		model: Model{Lang: En, State: protocol.SessionStateRunning},
		opts:  Options{Transport: tr, Lang: En},
	}
	_, cmd := p.Update(specsFoundMsg{goal: "implemente o que falta", specs: []protocol.SpecFolder{
		{Path: "specs/a", Criteria: 2, Unmet: 1, Pending: true},
		{Path: "specs/b", Criteria: 0, Pending: true},
		{Path: "specs/c", Criteria: 0, Pending: true},
	}})
	opened := func(want string, i int) {
		t.Helper()
		if cmd == nil {
			t.Fatalf("nothing was opened; want a qualifying session on %s", want)
		}
		if _, ok := cmd().(loopOpenedMsg); !ok {
			t.Fatalf("the command did not open a session")
		}
		tr.mu.Lock()
		defer tr.mu.Unlock()
		if len(tr.created) != i+1 {
			t.Fatalf("created %+v, want %d sessions", tr.created, i+1)
		}
		got := tr.created[i]
		if got.LoopSpec != want || !got.Qualify {
			t.Fatalf("session %d is %+v, want a qualifying one on %s", i, got, want)
		}
	}
	opened("specs/b", 0)

	_, cmd = p.Update(proposalWrittenMsg{spec: "specs/b", note: "written b"})
	opened("specs/c", 1)

	_, cmd = p.Update(proposalWrittenMsg{spec: "specs/c", note: "written c"})
	if cmd != nil {
		if _, ok := cmd().(loopOpenedMsg); ok {
			t.Fatal("a session opened after the last proposal: the work ran before anyone reviewed")
		}
	}
	tr.mu.Lock()
	n := len(tr.created)
	tr.mu.Unlock()
	if n != 2 {
		t.Errorf("%d sessions were opened, want the two qualifying ones only", n)
	}
	last := p.model.Entries[len(p.model.Entries)-1].Summary
	if !strings.Contains(last, "2") || !strings.Contains(last, "/loop") {
		t.Errorf("the pass ended without saying what to review and how to go on: %q", last)
	}
	if p.loopWorked != 0 || len(p.loopQueue) != 0 {
		t.Errorf("the run kept going: worked=%d queue=%v", p.loopWorked, p.loopQueue)
	}
}
