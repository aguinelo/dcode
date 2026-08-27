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
