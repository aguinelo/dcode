package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/protocol"
)

// `/loop` is recognised before the line becomes turn input.
//
// RN-3 of the loop-command family: the command text must never reach the
// history. Syntax in the prefix invalidates the cached prefix on every turn
// and spends tokens saying nothing to the model.
func TestLoopIsACommandAndNotTurnInput(t *testing.T) {
	got := ResolveInput("/loop specs/2026-08-25-home-page", config.CommandSet{})
	if got.Kind != CmdBuiltin {
		t.Fatalf("/loop resolved as kind %v, want a built-in", got.Kind)
	}
	if got.Name != "loop" {
		t.Fatalf("resolved name %q", got.Name)
	}
	if got.Text != "" {
		t.Errorf("the command text leaked into the turn: %q", got.Text)
	}
}

// A user command cannot shadow it, for the same reason none of the others can
// be shadowed: advice about dcode would stop being true of any installation.
func TestLoopCannotBeShadowed(t *testing.T) {
	user := config.CommandSet{Commands: map[string]config.Command{
		"loop": {Path: "somewhere.md", Body: "something else entirely"},
	}}
	if got := ResolveInput("/loop x", user); got.Kind != CmdBuiltin {
		t.Errorf("a user command shadowed /loop: %+v", got)
	}
}

// It appears in /help and in the completion menu, or nobody discovers it.
func TestLoopIsDiscoverable(t *testing.T) {
	help := HelpText(config.CommandSet{}, En)
	if !strings.Contains(help, "/loop") {
		t.Errorf("/help does not list /loop:\n%s", help)
	}
	found := false
	for _, c := range Complete("/lo", config.CommandSet{}, En) {
		if c.Name == "loop" {
			found = true
		}
	}
	if !found {
		t.Error("/loop does not complete from /lo")
	}
}

// The plan lists every folder, not only the ones with work left.
//
// Showing just the pending ones would leave someone unable to tell "this spec
// is finished" from "dcode did not see this spec", and those need different
// reactions.
func TestThePlanShowsEverySpecAndWhereItStands(t *testing.T) {
	got := LoopPlan([]protocol.SpecFolder{
		{Path: "specs/a", Criteria: 3, Unmet: 2, Pending: true},
		{Path: "specs/b", Criteria: 2},
		{Path: "specs/c", Criteria: 0, Pending: true},
		{Path: "specs/d", Error: "no tasks.md"},
	}, Text(En))

	for _, want := range []string{
		"specs/a", "specs/b", "specs/c", "specs/d",
		"2 of 3", "all 2 criteria", "no definition of done", "no tasks.md",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the plan does not carry %q:\n%s", want, got)
		}
	}
	// Two of the four are pending, and the head says so before any of it runs.
	if !strings.Contains(got, "2 of 4") {
		t.Errorf("the plan does not say how much is left:\n%s", got)
	}
}

// No specs at all is an answer, not an empty screen.
func TestAnEmptyPlanSaysSo(t *testing.T) {
	if got := LoopPlan(nil, Text(En)); !strings.Contains(got, "/loop <path>") {
		t.Errorf("an empty plan does not say what to do instead: %q", got)
	}
}

// The loop asks what a folder declares BEFORE opening anything, and opens a
// qualifying session when it declares nothing.
//
// Asking first is what keeps a discarded session and its record off the disk
// for every spec that needs qualifying.
func TestTheLoopAsksBeforeItOpens(t *testing.T) {
	specs := []protocol.SpecFolder{
		{Path: "specs/empty", Criteria: 0},
		{Path: "specs/ready", Criteria: 3},
	}

	// A folder with nothing declared: the session opened is a qualifying one.
	p, tr := loopProgram(t, specs...)
	typeLoop(t, p, "specs/empty")
	followLoop(t, p, func() bool { return tr.lastLoop(t).State == protocol.LoopAwaitingReview })
	if len(tr.created) != 1 || !tr.created[0].Qualify {
		t.Errorf("a folder declaring nothing was opened as work rather than qualified: %+v", tr.created)
	}

	// A folder that declares criteria is worked, not qualified.
	p, tr = loopProgram(t, specs...)
	typeLoop(t, p, "specs/ready")
	followLoop(t, p, nil)
	if len(tr.created) != 1 || tr.created[0].Qualify {
		t.Errorf("a folder with criteria was qualified anyway: %+v", tr.created)
	}
}

// If the daemon cannot say what is there, the command still runs.
//
// Refusing the work because the survey failed would be the survey holding the
// work hostage.
func TestASurveyThatFailsDoesNotStopTheWork(t *testing.T) {
	p, tr := loopProgram(t)
	tr.specsErr = errors.New("no")
	typeLoop(t, p, "specs/x")
	followLoop(t, p, nil)
	if len(tr.created) != 1 || tr.created[0].Qualify {
		t.Errorf("a failed survey stopped the command: %+v", tr.created)
	}
}

// The proposal is committed by the LOOP, after the turn, and what comes back
// is a note plus the measurement — not the start of the work.
func TestCommittingAProposalReportsAndStops(t *testing.T) {
	p, tr := loopProgram(t, protocol.SpecFolder{Path: "specs/x"})
	tr.committed = protocol.CommitDoneResponse{
		Path: "specs/x/done.toml", Criteria: 2, Summary: "two of them, one red",
	}
	typeLoop(t, p, "specs/x")
	followLoop(t, p, func() bool { return sawEntry(p, "specs/x/done.toml") })
	for _, want := range []string{"specs/x/done.toml", "two of them, one red"} {
		if !sawEntry(p, want) {
			t.Errorf("the note does not carry %q: %q", want, summaries(p))
		}
	}
}

// A commit that fails says so rather than leaving the loop believing a file
// was written.
func TestAFailedCommitSaysSo(t *testing.T) {
	p, tr := loopProgram(t, protocol.SpecFolder{Path: "specs/x"})
	tr.commitErr = errors.New("disk is full")
	typeLoop(t, p, "specs/x")
	followLoop(t, p, nil)
	if !sawEntry(p, "disk is full") {
		t.Errorf("a failed commit was not said: %q", summaries(p))
	}
}

// A goal turns into the specs it is about, and the plan is what the daemon
// said rather than what the client guessed.
func TestAGoalBecomesTheSpecsItIsAbout(t *testing.T) {
	p, tr := loopProgram(t, protocol.SpecFolder{Path: "specs/a", Criteria: 1, Unmet: 1, Pending: true, Measured: true})
	typeLoop(t, p, "termine tudo")
	followLoop(t, p, func() bool { return sawEntry(p, "specs/a") })
	if run := tr.lastLoop(t); run.Goal != "termine tudo" || len(run.Specs) != 1 {
		t.Errorf("got %+v", run)
	}
}

// An empty queue ends the run rather than looping on nothing.
func TestAnEmptyQueueEndsTheRun(t *testing.T) {
	p, tr := loopProgram(t, protocol.SpecFolder{Path: "specs/a", Criteria: 1})
	typeLoop(t, p, "termine tudo")
	followLoop(t, p, nil)
	if len(tr.created) != 0 {
		t.Error("an empty queue produced more work")
	}
	if run := tr.lastLoop(t); run.State != protocol.LoopFinished {
		t.Errorf("the run did not end: %s", run.State)
	}
}

// The diversion has to be wired, not merely available.
//
// GoalToQualify passing on its own says nothing about whether the handler calls
// it: a unit test of a helper the caller ignores is the shape that passes while
// the product still refuses. This one goes through Update, which is where the
// refusal was rendered.
func TestTheHandlerDivertsAGoalWithNoFoldersInsteadOfRefusing(t *testing.T) {
	p, tr := loopProgram(t)
	typeLoop(t, p, "revise o projeto até entender")
	followLoop(t, p, func() bool { return tr.lastLoop(t).State == protocol.LoopAwaitingReview })

	for _, e := range p.model.Entries {
		if strings.Contains(e.Summary, "no specs/") {
			t.Fatalf("the goal was refused instead of qualified: %q", e.Summary)
		}
	}
	if len(tr.created) == 0 {
		t.Fatal("nothing was started; the goal went nowhere at all")
	}
}

// And a goal that DID find folders still draws the plan. The diversion must not
// swallow the case it was never about.
func TestTheHandlerStillDrawsThePlanWhenFoldersExist(t *testing.T) {
	p, _ := loopProgram(t, protocol.SpecFolder{Path: "specs/home", Pending: true, Criteria: 3, Unmet: 3})
	typeLoop(t, p, "implemente o que falta")
	followLoop(t, p, func() bool { return sawEntry(p, "specs/home") })
	if !sawEntry(p, "specs/home") {
		t.Errorf("the plan was not drawn: %+v", p.model.Entries)
	}
}

// Through loopOne, which is where the survey already happens and where the
// fall-through opened a session against a path that was never there.
func TestLoopOneDivertsABareWordThatNamesNothing(t *testing.T) {
	p, tr := loopProgram(t)
	typeLoop(t, p, "oi")
	followLoop(t, p, func() bool { return tr.lastLoop(t).State != protocol.LoopSurveying && len(tr.submits()) > 0 })

	if !tr.lastLoop(t).Live() && tr.lastLoop(t).State == protocol.LoopFailed {
		t.Fatalf("the word was refused instead of qualified: %q", tr.lastLoop(t).Error)
	}
	if len(tr.created) == 0 {
		t.Fatal("no session was opened for the qualifying turn")
	}
	if !tr.created[0].Qualify {
		t.Error("a session was opened, and it is not the qualifying one")
	}
	if task := tr.submits()[0]; !strings.Contains(task, "oi") {
		t.Errorf("the turn does not name what it is about:\n%s", task)
	}
}

// The run ending says so, and says where done stands.
//
// It used to be silence: nextSpec returned nil when the queue emptied. From
// outside, a loop that worked four specs and stopped is indistinguishable from
// one that stalled on the fourth — and "it is over" reads exactly like "it is
// thinking".
func TestTheLoopSaysItFinishedAndWhereDoneStands(t *testing.T) {
	var specs []protocol.SpecFolder
	for i := 0; i < 4; i++ {
		specs = append(specs, protocol.SpecFolder{Path: specPath(i), Criteria: 4, Unmet: 1, Pending: true})
	}
	p, tr := loopProgram(t, specs...)
	tr.completions = map[string]*protocol.Completion{specPath(0): {
		Met:         []string{"tests", "vet"},
		Unmet:       []string{"coverage"},
		Unavailable: []string{"integration"},
	}}
	typeLoop(t, p, "finish the backlog")
	followLoop(t, p, nil)

	// The end draws one notice, after the line of the session it ended in.
	if n := len(p.model.Entries); n < 2 || !strings.Contains(p.model.Entries[n-2].Summary, specPath(3)) {
		t.Fatalf("the run ended and drew more than its notice: %q", summaries(p))
	}
	got := p.model.Entries[len(p.model.Entries)-1].Summary
	for _, want := range []string{"finished", "4", "2 of 4", "coverage", "integration"} {
		if !strings.Contains(got, want) {
			t.Errorf("the notice does not carry %q:\n%s", want, got)
		}
	}
}

// A run that worked nothing says nothing. The queue empties on every commit,
// including the ones where there was never a queue — and a line announcing the
// end of a run that never ran is a line about the feature, not the session.
func TestARunThatWorkedNothingSaysNothing(t *testing.T) {
	p, _ := loopProgram(t, protocol.SpecFolder{Path: "specs/a", Criteria: 2})
	typeLoop(t, p, "termine tudo")
	followLoop(t, p, nil)
	if sawEntry(p, "the loop is finished") {
		t.Errorf("an empty run announced itself: %q", summaries(p))
	}
}

// And the standing report survives a completion with nothing left: "4 of 4" is
// the answer somebody wants at the end, not an omission.
func TestAFinishedRunWithEverythingMetStillReportsIt(t *testing.T) {
	p, tr := loopProgram(t, protocol.SpecFolder{Path: "docs/specs/architecture/one", Criteria: 4, Unmet: 4, Pending: true})
	tr.completions = map[string]*protocol.Completion{
		"docs/specs/architecture/one": {Met: []string{"a", "b", "c", "d"}},
	}
	typeLoop(t, p, "finish the backlog")
	followLoop(t, p, nil)
	got := p.model.Entries[len(p.model.Entries)-1].Summary
	if !strings.Contains(got, "4 of 4") {
		t.Errorf("a fully met run does not say so:\n%s", got)
	}
}
