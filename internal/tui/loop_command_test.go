package tui

import (
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
