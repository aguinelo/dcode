package tui

import (
	"context"
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

func TestParseLoopArgs(t *testing.T) {
	for _, c := range []struct {
		name    string
		in      string
		spec    string
		protect []string
		task    string
		wantErr string // the message, or "" for no error
	}{
		{name: "just a path", in: "specs/home-page", spec: "specs/home-page"},
		{name: "protect", in: "specs/x --protect **/*_test.go",
			spec: "specs/x", protect: []string{"**/*_test.go"}},
		{name: "protect with equals", in: "specs/x --protect=**/*_test.go",
			spec: "specs/x", protect: []string{"**/*_test.go"}},
		{name: "protect twice", in: "specs/x --protect a --protect b",
			spec: "specs/x", protect: []string{"a", "b"}},
		{name: "flag before path", in: "--protect a specs/x",
			spec: "specs/x", protect: []string{"a"}},
		{name: "nothing", in: "", wantErr: ""},
		{name: "unknown flag", in: "specs/x --max-iterations 3", wantErr: "--max-iterations"},
		{name: "protect with nothing after it", in: "specs/x --protect", wantErr: "--protect needs a glob"},
		{name: "a task after the path", in: "specs/x refaça só o header",
			spec: "specs/x", task: "refaça só o header"},
		{name: "task and protect", in: "specs/x --protect a implemente tudo",
			spec: "specs/x", protect: []string{"a"}, task: "implemente tudo"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseLoopArgs(c.in)
			if c.name == "nothing" || c.wantErr != "" {
				if err == nil {
					t.Fatalf("expected an error, got %+v", got)
				}
				if c.wantErr != "" && !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("error %q does not name %q", err, c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Spec != c.spec {
				t.Errorf("spec = %q, want %q", got.Spec, c.spec)
			}
			if got.Task != c.task {
				t.Errorf("task = %q, want %q", got.Task, c.task)
			}
			if len(got.Protect) != len(c.protect) {
				t.Fatalf("protect = %v, want %v", got.Protect, c.protect)
			}
			for i := range c.protect {
				if got.Protect[i] != c.protect[i] {
					t.Errorf("protect = %v, want %v", got.Protect, c.protect)
				}
			}
		})
	}
}

// A mistyped flag stops the command rather than being ignored.
//
// Ignoring it opens a session measured against a definition of done the person
// did not ask for, and they find out at the end of the turn.
func TestAMistypedFlagStopsTheCommand(t *testing.T) {
	if _, err := ParseLoopArgs("specs/x --protct a"); err == nil {
		t.Fatal("a mistyped flag was accepted")
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

// A word after the path is what to do, not a mistyped flag.
//
// Someone typed `/loop <path> implementar ...` and was told "implementar is
// not a flag here". It never was one: only a thing that looks like a flag can
// be a mistyped flag, and the command was refusing a sentence.
func TestAWordAfterThePathIsTheTaskAndNotAFlag(t *testing.T) {
	got, err := ParseLoopArgs("specs/home-page implementar absolutamente tudo")
	if err != nil {
		t.Fatalf("a sentence after the path was refused: %v", err)
	}
	if got.Spec != "specs/home-page" {
		t.Errorf("spec = %q", got.Spec)
	}
	if got.Task != "implementar absolutamente tudo" {
		t.Errorf("task = %q", got.Task)
	}
}

// /loop submits a turn. Loading a definition of done and then waiting is the
// command doing half its job.
func TestLoopSubmitsSomethingToDo(t *testing.T) {
	got := LoopTask(LoopArgs{Spec: "specs/x"})
	if got == "" {
		t.Fatal("/loop with no task submits nothing at all")
	}
	if !strings.Contains(got, "specs/x") {
		t.Errorf("the default task does not name the spec: %q", got)
	}
	// It must not restate the criteria: they are the session's, the loop
	// checks them, and a copy in the first message is a second statement of
	// something that can move.
	if strings.Contains(strings.ToLower(got), "done.toml") {
		t.Errorf("the default task restates where the criteria live: %q", got)
	}
	// What the person said wins over the default, whole.
	if got := LoopTask(LoopArgs{Spec: "specs/x", Task: "só o header"}); got != "só o header" {
		t.Errorf("the person's own words were changed: %q", got)
	}
}

// A sentence is a goal, and a path is a path.
//
// `/loop implemente todas as specs pendentes` went looking for
// `implemente/tasks.md`: the first word became a folder name. Prose became a
// path, which is the same defect as prose becoming a criterion, pointing the
// other way.
func TestASentenceIsAGoalAndAPathIsAPath(t *testing.T) {
	for _, c := range []struct {
		in   string
		goal bool
		spec string
		task string
	}{
		{in: "specs/home-page", spec: "specs/home-page"},
		{in: "specs/home-page refaça o header", spec: "specs/home-page", task: "refaça o header"},
		{in: "home-page", spec: "home-page"},
		{in: "implemente todas as specs pendentes", goal: true, task: "implemente todas as specs pendentes"},
		{in: "termine o que falta", goal: true, task: "termine o que falta"},
		{in: "implemente todas --protect tests/**", goal: true, task: "implemente todas"},
	} {
		t.Run(c.in, func(t *testing.T) {
			got, err := ParseLoopArgs(c.in)
			if err != nil {
				t.Fatal(err)
			}
			if got.Goal != c.goal {
				t.Fatalf("goal = %v, want %v (%+v)", got.Goal, c.goal, got)
			}
			if got.Spec != c.spec {
				t.Errorf("spec = %q, want %q", got.Spec, c.spec)
			}
			if got.Task != c.task {
				t.Errorf("task = %q, want %q", got.Task, c.task)
			}
		})
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

// The loop qualifies a folder that declares nothing, before working it.
//
// Reading, projecting and qualifying come before executing, and the LOOP owns
// that order: a model that chose when to qualify would be choosing when to be
// measured.
func TestTheLoopQualifiesAFolderThatDeclaresNothing(t *testing.T) {
	// The default task for a qualifying run is a different job, and says so.
	got := LoopTask(LoopArgs{Spec: "specs/x", Qualify: true})
	for _, want := range []string{"specs/x", "done_propose", "plan mode", "Propose, and stop"} {
		if !strings.Contains(got, want) {
			t.Errorf("the qualifying task does not carry %q:\n%s", want, got)
		}
	}
	// And it does not tell the model to do the work.
	if strings.Contains(strings.ToLower(got), "implement the specification") {
		t.Errorf("the qualifying task asks for the work:\n%s", got)
	}
}

// What the person typed is not carried into a qualifying turn.
//
// `/loop specs/x refaça o header` says what the WORK is. Handing it to the
// turn that decides how the work will be measured would let the instruction
// shape the ruler.
func TestAQualifyingTurnDoesNotCarryTheTask(t *testing.T) {
	got := LoopTask(LoopArgs{Spec: "specs/x", Qualify: true, Task: "refaça só o header"})
	if strings.Contains(got, "refaça só o header") {
		t.Errorf("the person's instruction reached the qualifying turn:\n%s", got)
	}
}

// The loop asks what a folder declares BEFORE opening anything, and opens a
// qualifying session when it declares nothing.
//
// Asking first is what keeps a discarded session and its record off the disk
// for every spec that needs qualifying.
func TestTheLoopAsksBeforeItOpens(t *testing.T) {
	p, tr := newProgram(t)
	tr.specs = []protocol.SpecFolder{
		{Path: "specs/empty", Criteria: 0},
		{Path: "specs/ready", Criteria: 3},
	}

	// A folder with nothing declared: the session opened is a qualifying one.
	msg := p.loopOne(LoopArgs{Spec: "specs/empty"})()
	opened, ok := msg.(loopOpenedMsg)
	if !ok {
		t.Fatalf("got %T, want a session", msg)
	}
	if !opened.qualify {
		t.Error("a folder declaring nothing was opened as work rather than qualified")
	}

	// A folder that declares criteria is worked, not qualified.
	msg = p.loopOne(LoopArgs{Spec: "specs/ready"})()
	if opened, ok := msg.(loopOpenedMsg); !ok || opened.qualify {
		t.Errorf("a folder with criteria was qualified anyway: %+v", msg)
	}
}

// If the daemon cannot say what is there, the command still runs.
//
// Refusing the work because the survey failed would be the survey holding the
// work hostage.
func TestASurveyThatFailsDoesNotStopTheWork(t *testing.T) {
	p, tr := newProgram(t)
	tr.specsErr = errors.New("no")
	if _, ok := p.loopOne(LoopArgs{Spec: "specs/x"})().(loopOpenedMsg); !ok {
		t.Error("a failed survey stopped the command")
	}
}

// The proposal is committed by the LOOP, after the turn, and what comes back
// is a note plus the measurement — not the start of the work.
func TestCommittingAProposalReportsAndStops(t *testing.T) {
	p, tr := newProgram(t)
	tr.committed = protocol.CommitDoneResponse{
		Path: "specs/x/done.toml", Criteria: 2, Summary: "two of them, one red",
	}
	msg := p.commitProposal("s1", "specs/x")()
	written, ok := msg.(proposalWrittenMsg)
	if !ok {
		t.Fatalf("got %T", msg)
	}
	for _, want := range []string{"specs/x/done.toml", "two of them, one red"} {
		if !strings.Contains(written.note, want) {
			t.Errorf("the note does not carry %q: %s", want, written.note)
		}
	}
}

// A commit that fails says so rather than leaving the loop believing a file
// was written.
func TestAFailedCommitSaysSo(t *testing.T) {
	p, tr := newProgram(t)
	tr.commitErr = errors.New("disk is full")
	msg := p.commitProposal("s1", "specs/x")()
	note, ok := msg.(noteMsg)
	if !ok || !strings.Contains(string(note), "disk is full") {
		t.Errorf("got %T %v", msg, msg)
	}
}

// A goal turns into the specs it is about, and the plan is what the daemon
// said rather than what the client guessed.
func TestAGoalBecomesTheSpecsItIsAbout(t *testing.T) {
	p, tr := newProgram(t)
	tr.specs = []protocol.SpecFolder{{Path: "specs/a", Criteria: 1, Unmet: 1, Pending: true, Measured: true}}
	msg := p.loopEverySpec(LoopArgs{Goal: true, Task: "termine tudo"})()
	found, ok := msg.(specsFoundMsg)
	if !ok {
		t.Fatalf("got %T", msg)
	}
	if found.goal != "termine tudo" || len(found.specs) != 1 {
		t.Errorf("got %+v", found)
	}
}

// An empty queue ends the run rather than looping on nothing.
func TestAnEmptyQueueEndsTheRun(t *testing.T) {
	p, _ := newProgram(t)
	p.loopGoal = "termine tudo"
	if cmd := p.nextSpec(); cmd != nil {
		t.Error("an empty queue produced more work")
	}
	if p.loopGoal != "" {
		t.Error("the run did not end")
	}
}

// A goal with no spec folder anywhere is qualified, not refused.
//
// `/loop revise o projeto até entender` answered "no specs/ folder here, or
// nothing in it. /loop <path> works on one folder." — the command telling
// someone their request was the wrong shape, in a product whose done-qualifier
// family exists for exactly this case. Its own .r names the prose request as
// what motivated it: "Faça um cadastro de clientes" carries no tasks.md, and
// the constructive answer is to raise the criteria, measure them, and ask for a
// signature.
//
// `/loop <path>` already does that when the folder declares nothing. Only the
// sentence route dead-ended.
func TestAGoalWithNoSpecFolderIsQualified(t *testing.T) {
	got, ok := GoalToQualify(LoopArgs{Goal: true, Task: "revise o projeto até entender"}, nil)
	if !ok {
		t.Fatal("a goal with no spec folders must be qualified rather than refused")
	}
	if !got.Qualify {
		t.Error("the session has to be the qualifying one")
	}
	if !got.Goal {
		t.Error("the qualifying session lost the fact that its subject is a sentence")
	}
	if got.Task != "revise o projeto até entender" {
		t.Errorf("the sentence is the whole brief and it did not survive: %q", got.Task)
	}
	// Anchored where a workspace's definition of done already lives, and not in
	// a folder invented from the sentence. The loop-command .r records the
	// opposite defect — prose becoming a path — and inventing
	// `revise-o-projeto/` would be that defect again.
	if got.Spec != ".dcode" {
		t.Errorf("the proposal is anchored at %q, want .dcode", got.Spec)
	}
}

// With folders present, nothing changes: the goal still selects among them.
func TestAGoalWithSpecFoldersStillWorksThroughThem(t *testing.T) {
	found := []protocol.SpecFolder{{Path: "specs/home", Pending: true, Criteria: 3}}
	if _, ok := GoalToQualify(LoopArgs{Goal: true, Task: "implemente o que falta"}, found); ok {
		t.Error("a goal with folders to work must not be diverted into qualifying")
	}
}

// An empty sentence is not a brief. It cannot happen through ParseLoopArgs,
// which refuses an empty argument, and a qualifying turn whose subject is ""
// would ask the model to work out how nothing is finished.
func TestAnEmptyGoalIsNotQualified(t *testing.T) {
	if _, ok := GoalToQualify(LoopArgs{Goal: true, Task: "   "}, nil); ok {
		t.Error("an empty sentence was accepted as a brief")
	}
}

// The qualifying turn names the sentence, and does not send the model looking
// for a specification that does not exist.
func TestTheQualifyingTurnForAGoalNamesTheSentence(t *testing.T) {
	task := LoopTask(LoopArgs{Qualify: true, Goal: true, Task: "revise o projeto", Spec: ".dcode"})
	if !strings.Contains(task, "revise o projeto") {
		t.Errorf("the turn does not name what it is about:\n%s", task)
	}
	if strings.Contains(task, ".dcode") {
		t.Errorf("the turn names the anchor instead of the request:\n%s", task)
	}
	if strings.Contains(task, "Read the specification") {
		t.Errorf("it sends the model to read a specification that does not exist:\n%s", task)
	}
	if !strings.Contains(task, "done_propose") {
		t.Errorf("a qualifying turn that never mentions the tool proposes nothing:\n%s", task)
	}
}

// The diversion has to be wired, not merely available.
//
// GoalToQualify passing on its own says nothing about whether the handler calls
// it: a unit test of a helper the caller ignores is the shape that passes while
// the product still refuses. This one goes through Update, which is where the
// refusal was rendered.
func TestTheHandlerDivertsAGoalWithNoFoldersInsteadOfRefusing(t *testing.T) {
	p := &program{model: Model{Lang: En, State: protocol.SessionStateIdle}}
	_, cmd := p.Update(specsFoundMsg{goal: "revise o projeto até entender"})

	for _, e := range p.model.Entries {
		if strings.Contains(e.Summary, "no specs/") {
			t.Fatalf("the goal was refused instead of qualified: %q", e.Summary)
		}
	}
	if cmd == nil {
		t.Fatal("nothing was started; the goal went nowhere at all")
	}
}

// And a goal that DID find folders still draws the plan. The diversion must not
// swallow the case it was never about.
func TestTheHandlerStillDrawsThePlanWhenFoldersExist(t *testing.T) {
	p := &program{model: Model{Lang: En, State: protocol.SessionStateIdle}}
	p.Update(specsFoundMsg{goal: "implemente o que falta", specs: []protocol.SpecFolder{
		{Path: "specs/home", Pending: true, Criteria: 3},
	}})
	found := false
	for _, e := range p.model.Entries {
		if strings.Contains(e.Summary, "specs/home") {
			found = true
		}
	}
	if !found {
		t.Errorf("the plan was not drawn: %+v", p.model.Entries)
	}
}

// `/loop oi` is a goal, not a folder called oi.
//
// specArgument says a single word is a path, because "one word is what a folder
// name looks like and the error names it when it is not there". The error it
// names is this:
//
//	could not open a session: invalid_input: loopcommand: read
//	/Users/…/dcode_test/oi/tasks.md: open /Users/…/oi/tasks.md: no such file
//
// A raw daemon error, the absolute path twice, for someone who typed a word.
// The rule that a goal with no folder gets qualified never applied, because a
// bare word never became a goal in the first place.
func TestABareWordThatNamesNoFolderBecomesAGoal(t *testing.T) {
	spec, err := ParseLoopArgs("oi")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := ArgumentToQualify(spec, nil)
	if !ok {
		t.Fatal("a bare word naming no folder must be qualified, not read as a path")
	}
	if got.Task != "oi" {
		t.Errorf("the word is the brief and it did not survive: %q", got.Task)
	}
	if !got.Qualify || !got.Goal {
		t.Errorf("it did not become a qualifying goal: %+v", got)
	}
}

// A word that names a folder the survey found is still a path. The diversion is
// for arguments that name nothing, not for every single word.
func TestABareWordThatNamesAFolderStaysAPath(t *testing.T) {
	spec, _ := ParseLoopArgs("home")
	found := []protocol.SpecFolder{{Path: "home", Criteria: 3, Pending: true}}
	if _, ok := ArgumentToQualify(spec, found); ok {
		t.Error("an argument naming a real spec folder was diverted into qualifying")
	}
}

// Something written as a path stays an error when it is not there.
//
// `specs/hoem` is a typo, and a typo answered by qualifying it as a goal is a
// typo hidden. The separator is what separates the two cases: a person who
// wrote one meant a path.
func TestAMistypedPathIsNotQualifiedAway(t *testing.T) {
	spec, _ := ParseLoopArgs("specs/hoem")
	if _, ok := ArgumentToQualify(spec, nil); ok {
		t.Error("a mistyped path was turned into a goal, which hides the typo")
	}
}

// Through loopOne, which is where the survey already happens and where the
// fall-through opened a session against a path that was never there.
func TestLoopOneDivertsABareWordThatNamesNothing(t *testing.T) {
	p := &program{model: Model{Lang: En, State: protocol.SessionStateIdle},
		opts: Options{Transport: newFakeTransport()}}
	spec, _ := ParseLoopArgs("oi")

	msg := p.loopOne(spec)()
	note, refused := msg.(noteMsg)
	if refused {
		t.Fatalf("the word was refused instead of qualified: %v", string(note))
	}
	opened, ok := msg.(loopOpenedMsg)
	if !ok {
		t.Fatalf("got %T, want a session opened for the qualifying turn", msg)
	}
	if !opened.qualify {
		t.Error("a session was opened, and it is not the qualifying one")
	}
	if !strings.Contains(opened.task, "oi") {
		t.Errorf("the turn does not name what it is about:\n%s", opened.task)
	}
}

// The run ending says so, and says where done stands.
//
// It used to be silence: nextSpec returned nil when the queue emptied. From
// outside, a loop that worked four specs and stopped is indistinguishable from
// one that stalled on the fourth — and "it is over" reads exactly like "it is
// thinking".
func TestTheLoopSaysItFinishedAndWhereDoneStands(t *testing.T) {
	p := &program{ctx: context.Background(),
		model: Model{Lang: En, State: protocol.SessionStateIdle},
		opts:  Options{Transport: newFakeTransport()}}

	p.Update(loopFinishedMsg{worked: 4, results: []loopResult{{
		spec: "docs/specs/architecture/one",
		standing: &protocol.Completion{
			Met:         []string{"tests", "vet"},
			Unmet:       []string{"coverage"},
			Unavailable: []string{"integration"},
		},
	}}})

	if len(p.model.Entries) != 1 {
		t.Fatalf("the run ended and drew %d entries", len(p.model.Entries))
	}
	got := p.model.Entries[0].Summary
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
	p := &program{ctx: context.Background(), model: Model{Lang: En},
		opts: Options{Transport: newFakeTransport()}}
	if cmd := p.nextSpec(); cmd != nil {
		t.Error("an empty run announced itself")
	}
}

// And the standing report survives a completion with nothing left: "4 of 4" is
// the answer somebody wants at the end, not an omission.
func TestAFinishedRunWithEverythingMetStillReportsIt(t *testing.T) {
	p := &program{ctx: context.Background(), model: Model{Lang: En},
		opts: Options{Transport: newFakeTransport()}}
	p.Update(loopFinishedMsg{worked: 1, results: []loopResult{{
		spec:     "docs/specs/architecture/one",
		standing: &protocol.Completion{Met: []string{"a", "b", "c", "d"}},
	}}})
	got := p.model.Entries[0].Summary
	if !strings.Contains(got, "4 of 4") {
		t.Errorf("a fully met run does not say so:\n%s", got)
	}
}
