package loopcommand

import (
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The grammar of `/loop` and the turns it submits. These moved here from
// internal/tui with the grammar itself, unchanged: the daemon parses what a
// client sent, and the terminal keeps no copy of its own.

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
