package orchestrate

import (
	"errors"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// Spec: orchestration .p §10 — steer_child is a new turn for an idle or
// failed child, and steering of the turn a running one is in; a closed one
// takes no more work.
func TestSteeringIsANewTurnWhenIdleAndSteeringWhenRunning(t *testing.T) {
	p := openWith(t, nil)
	c := p.one(t, item("index", "internal/index"))
	if err := p.o.Steer(c.ID, "also the tests", "the tests were missing"); err != nil {
		t.Fatal(err)
	}
	if len(p.sp.steered) != 1 {
		t.Errorf("a running child was not steered: %v", p.sp.steered)
	}
	p.reg.Observe(c.ID, event(t, protocol.EventTurnCompleted, protocol.TurnCompleted{TurnID: "t1", Reason: protocol.StopError}))
	if err := p.o.Steer(c.ID, "try again", "it failed on a flake"); err != nil {
		t.Fatal(err)
	}
	if got := p.o.child(c.ID); got.State != stateWorking || got.Reason != "" || len(p.sp.submitted) != 2 {
		t.Errorf("a failed child steered is %s/%s, submitted %v", got.State, got.Reason, p.sp.submitted)
	}
	for name, err := range map[string]error{
		"unknown":  p.o.Steer("nobody", "x", "y"),
		"no why":   p.o.Steer(c.ID, "x", " "),
		"no child": p.o.Close("nobody", "y"),
		"no why 2": p.o.Close(c.ID, ""),
	} {
		if err == nil {
			t.Errorf("%s was taken", name)
		}
	}
	if err := p.o.Close(c.ID, "done with it"); err != nil {
		t.Fatal(err)
	}
	wantCode(t, p.o.Steer(c.ID, "x", "y"), protocol.CodeInvalidInput)
	wantCode(t, p.o.Close(c.ID, "again"), protocol.CodeInvalidInput)
}

// Spec: orchestration .p §6 — what fails after a batch is admitted fails that
// child alone: dispatched to failed, reason error, said.
func TestAChildThatCannotBeBuiltFailsAndSaysWhy(t *testing.T) {
	p := openWith(t, nil)
	p.sp.failChild = errors.New("no sandbox")
	c := p.one(t, item("index", "internal/index"))
	if c.State != stateFailed || c.Reason != protocol.StopError {
		t.Errorf("a child whose session was not built is %s/%s", c.State, c.Reason)
	}
	p.sp.failChild, p.sp.failSubmit = nil, errors.New("busy")
	c = p.one(t, item("docs", "docs"))
	if c.State != stateFailed {
		t.Errorf("a child whose first turn did not start is %s", c.State)
	}
	if p.w.count(protocol.EventSessionError) < 2 {
		t.Errorf("the failures were not said to the person: %v", p.w.events)
	}
}

func TestABatchIsRefusedForEveryShapeOfBadInput(t *testing.T) {
	p := openWith(t, nil)
	noWhy := item("a", "a")
	noWhy.Why = ""
	for name, items := range map[string][]Item{
		"empty":      nil,
		"no why":     {noWhy},
		"no owns":    {item("a")},
		"absolute":   {item("a", "/etc")},
		"climbs out": {item("a", "../x")},
		"not clean":  {item("a", "a/../b")},
		"the tree":   {item("a", ".")},
		"backslash":  {item("a", `a\b`)},
		"blank":      {item("a", "")},
	} {
		if _, err := p.o.Delegate(items); err == nil {
			t.Errorf("%s: the batch was taken", name)
		}
	}
	p.noWorktree(t)
}

func TestAnAnswerMustBeOneOfTheOptions(t *testing.T) {
	p := openWith(t, map[string]string{"orchestrator.token_ceiling": "100000"})
	p.o.spend(&protocol.Usage{InputTokens: 100000})
	_, _ = p.o.Delegate([]Item{item("a", "a")})
	q := p.o.Questions()[0]
	wantCode(t, func() error { _, err := p.o.Answer("q99", "stop", ""); return err }(), protocol.CodeQuestionNotFound)
	wantCode(t, func() error { _, err := p.o.Answer(q.ID, "maybe", ""); return err }(), protocol.CodeInvalidInput)
	if got, err := p.o.Answer(q.ID, "stop", "enough"); err != nil || got.Note != "enough" || got.Answered == nil {
		t.Errorf("stop = %+v, %v", got, err)
	}
	wantCode(t, func() error { _, err := p.o.Answer(q.ID, "continue", ""); return err }(), protocol.CodeQuestionAlreadyAnswered)
	if p.o.snapshot().Spent == 0 {
		t.Error("stop opened a new window")
	}
}

// Spec: orchestration .p §8 — a child that cannot be reopened fails, said in
// daemon.restart; the closed session of a gone child is not an error to close.
func TestAChildThatCannotBeReopenedFailsAndIsSaid(t *testing.T) {
	p := openWith(t, nil)
	c := p.one(t, item("index", "internal/index"))
	p.reg.Close()
	p.sp.failReopen = map[string]error{c.ID: errors.New("record gone")}
	again := p.registry(t, nil)
	if errs := again.Reload(t.Context()); len(errs) != 0 {
		t.Fatal(errs)
	}
	kids, _ := again.ChildrenOf(c.ParentID)
	if kids[0].State != stateFailed || kids[0].Reason != protocol.StopError {
		t.Errorf("an unreopened child is %s/%s", kids[0].State, kids[0].Reason)
	}
	o, _ := again.Get(p.o.Project().ID)
	p.sp.closeErr = protocol.Errorf(protocol.CodeSessionNotFound, "no session")
	if err := o.Close(c.ID, "the person closed it"); err != nil {
		t.Errorf("closing a child whose session is gone = %v", err)
	}
	if _, held := again.ChildrenOf("nobody"); held {
		t.Error("a session that conducts nothing was held")
	}
	if held, _ := again.CloseChild("nobody"); held {
		t.Error("a session that is no child was closed")
	}
	again.Observe("nobody", protocol.Event{Type: protocol.EventTurnCompleted})
}

func TestASlugIsAReadableBranchSegment(t *testing.T) {
	for title, want := range map[string]string{
		"The Index!": "the-index", "!!!": "child", "índice 2": "ndice-2",
		strings.Repeat("abc ", 20): "abc-abc-abc-abc-abc-abc-abc-abc-abc-abc",
	} {
		if got := slug(title); got != want {
			t.Errorf("slug(%q) = %q, want %q", title, got, want)
		}
	}
}

func TestClosedRetentionIsAnHourUnlessDeclared(t *testing.T) {
	if d, err := ClosedRetention(resolved(nil)); err != nil || d.Hours() != 1 {
		t.Errorf("default = %v, %v", d, err)
	}
	if d, err := ClosedRetention(resolved(map[string]string{"orchestrator.closed_retention": "0s"})); err != nil || d != 0 {
		t.Errorf("0s = %v, %v", d, err)
	}
	l, err := ReadLimits(t.Context(), resolved(map[string]string{"orchestrator.branch_prefix": "bot/", "orchestrator.max_children": "2",
		"orchestrator.token_ceiling": "200000", "orchestrator.child_profiles": "a,b"}), 0)
	if err != nil || l.BranchPrefix != "bot/" || l.MaxChildren != 2 || l.TokenCeiling != 200000 || len(l.ChildProfiles) != 2 {
		t.Errorf("limits = %+v, %v", l, err)
	}
}
