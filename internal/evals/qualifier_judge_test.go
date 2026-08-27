package evals

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/loop/qualifier"
	"github.com/aguinelo/dcode/internal/tui"
)

func proposal(criteria ...map[string]any) ce.ToolCall {
	raw, err := json.Marshal(map[string]any{"criteria": criteria})
	if err != nil {
		panic(err)
	}
	return ce.ToolCall{Name: "done_propose", Input: raw}
}

func criterion(name, command, expects string) map[string]any {
	return map[string]any{"name": name, "command": command, "expects": expects}
}

// A threshold is what a person writes; a command is what decides. The judge has
// to be able to tell them apart, and this is the pair the contract is named for.
func TestASentenceInTheCommandFieldIsNotACriterion(t *testing.T) {
	for _, tc := range []struct {
		command string
		want    bool
	}{
		{"pnpm lhci --assert", true},
		{"go test ./...", true},
		{"make check", true},
		{"[ -f REVIEW.md ]", true},
		{"./scripts/coverage.sh", true},
		{"grep -q Slugify stats.go", true},
		{"Lighthouse >= 95", false},
		{"The suite should pass", false},
		{"coverage is at least 90%", false},
		{"", false},
	} {
		got := EveryCriterionIsACommand()(Transcript{
			Calls: []ce.ToolCall{proposal(criterion("c", tc.command, "fail"))},
		})
		if got != tc.want {
			t.Errorf("%q judged %v, want %v", tc.command, got, tc.want)
		}
	}
}

// One bad criterion in an otherwise good set fails the run: the contract says
// EVERY criterion, and a set that mixes the two is a ruler with a prose line
// in it.
func TestOneSentenceSpoilsTheProposal(t *testing.T) {
	tr := Transcript{Calls: []ce.ToolCall{proposal(
		criterion("a", "go test ./...", "pass"),
		criterion("b", "Lighthouse >= 95", "fail"),
	)}}
	if EveryCriterionIsACommand()(tr) {
		t.Error("a set carrying a sentence was judged as commands")
	}
}

// A model that proposes twice has changed its mind, and the product keeps the
// second. Judging the first would score a proposal that was withdrawn.
func TestTheProposalJudgedIsTheOneThatStands(t *testing.T) {
	tr := Transcript{Calls: []ce.ToolCall{
		proposal(criterion("a", "Lighthouse >= 95", "fail")),
		proposal(criterion("a", "pnpm lhci --assert", "fail")),
	}}
	if !EveryCriterionIsACommand()(tr) {
		t.Error("the withdrawn proposal was judged instead of the one that stands")
	}
}

func TestNothingProposedIsNotAPass(t *testing.T) {
	for name, j := range map[string]Judge{
		"commands": EveryCriterionIsACommand(),
		"guard":    ProposesAGuard(),
		"keeps":    KeepsMeasuring("slug", "old"),
	} {
		if j(Transcript{Text: "I would suggest running the suite."}) {
			t.Errorf("%s: a turn that proposed nothing was judged as having proposed", name)
		}
	}
}

func TestAGuardIsACriterionExpectedToPass(t *testing.T) {
	red := Transcript{Calls: []ce.ToolCall{proposal(
		criterion("a", "go test -run TestSlugify ./...", "fail"),
		criterion("b", "test -f slug.go", "fail"),
	)}}
	if ProposesAGuard()(red) {
		t.Error("a set with nothing expected to pass was judged as carrying a guard")
	}
	mixed := Transcript{Calls: []ce.ToolCall{proposal(
		criterion("a", "go test -run TestSlugify ./...", "fail"),
		criterion("b", "go test ./...", "pass"),
	)}}
	if !ProposesAGuard()(mixed) {
		t.Error("a regression guard was not recognised")
	}
}

// The two ways to answer a broken criterion are opposite mistakes, and the
// judge has to refuse both — while accepting a better handle for the same
// measurement, which is what twenty paid runs were spent finding out.
func TestKeepsMeasuringRefusesDroppingItAndRepeatingIt(t *testing.T) {
	was := "gotestsum --run TestSlugify ./..."
	j := KeepsMeasuring("slug", was)

	dropped := Transcript{Calls: []ce.ToolCall{proposal(
		criterion("package-tests", "go test ./...", "pass"),
		criterion("vet", "go vet ./...", "pass"),
	)}}
	if j(dropped) {
		t.Error("a proposal with nothing about the subject was judged as keeping it")
	}
	repeated := Transcript{Calls: []ce.ToolCall{proposal(
		criterion("slug", was, "fail"),
	)}}
	if j(repeated) {
		t.Error("re-proposing the command that ran nothing was judged as fixing it")
	}
	// Every one of these is a real transcript from the run that found the
	// judge. All three keep the measurement; none keeps the name.
	for _, kept := range []map[string]any{
		criterion("slug-test", "go test -run TestSlugify ./...", "fail"),
		criterion("slug-impl-exists", "grep -q 'func Slugify' stats.go", "fail"),
		criterion("basic", "go test -run TestSlugify ./...", "fail"),
	} {
		tr := Transcript{Calls: []ce.ToolCall{proposal(
			criterion("package-tests", "go test ./...", "pass"), kept,
		)}}
		if !j(tr) {
			t.Errorf("%v was judged as having dropped the criterion", kept)
		}
	}
	// And repeating it anywhere in the set still fails, even beside a good one.
	beside := Transcript{Calls: []ce.ToolCall{proposal(
		criterion("slug-test", "go test -run TestSlugify ./...", "fail"),
		criterion("old", was, "fail"),
	)}}
	if j(beside) {
		t.Error("the broken command was re-proposed beside a fixed one and passed")
	}
}

// The fixture's done.toml is not written by hand: it is what the product's own
// Render produces for the measurement the scenario describes. Anything else is
// a copy that drifts from the format the next run has to read.
func TestTheBrokenFixtureIsWhatTheProductWrites(t *testing.T) {
	path := filepath.Join(FixtureRoot, "qualifier-fixes-broken", "files", "specs", "slugify", "done.toml")
	on, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := qualifier.Render([]qualifier.Measured{{
		Proposed: qualifier.Proposed{
			Name:    "slug",
			Command: "gotestsum --run TestSlugify ./...",
			Expects: qualifier.ExpectFail,
			Why:     "the work has to make this pass",
		},
		Class: qualifier.ClassBroken,
		Exit:  127,
	}}, nil, qualifier.Conditions{})
	if string(on) != string(want) {
		t.Errorf("the fixture is not what the product writes:\n--- on disk ---\n%s\n--- product ---\n%s", on, want)
	}
}

// A qualifying scenario takes its opening line from the product. It is most of
// what these contracts measure, and a second copy of it here is one that
// drifts.
func TestAQualifyingFixtureTakesItsTaskFromTheProduct(t *testing.T) {
	f, err := LoadFixture(FixtureRoot, "qualifier-proposes-commands")
	if err != nil {
		t.Fatal(err)
	}
	if want := tui.LoopTask(tui.LoopArgs{Qualify: true, Spec: "specs/slugify"}); f.Task != want {
		t.Errorf("the task is not the product's:\n got %q\nwant %q", f.Task, want)
	}
}

// The premise is checked, not asserted. A scenario that says "there is no
// repository here" running inside one would measure a model being silent about
// something that was never in its prompt.
func TestAScenarioThatDeclaresNoRepositoryRefusesToRunInsideOne(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git: the check under test is git answering")
	}
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	f := Fixture{ID: "x", Tools: []ce.ToolDef{{Name: "read"}}, World: World{Repo: RepoAbsent}}
	_, err := f.PromptIn(context.Background(), "", dir)
	if err == nil {
		t.Fatal("a scenario declaring no repository built a prompt inside one")
	}
	if !strings.Contains(err.Error(), "the opposite of what it says") {
		t.Errorf("the refusal does not say why: %v", err)
	}
}

// A scenario that declares no repository has the fact IN its prefix.
//
// Without this the floor contracts are unfalsifiable in the worst direction: a
// model judged for not mentioning something that was never in front of it,
// with the silence recorded as a rate. The survey already refuses to run
// inside a repository; this is the other half — that saying "absent" actually
// puts the paragraph there.
func TestAScenarioThatDeclaresNoRepositorySaysSoInThePrefix(t *testing.T) {
	f, err := LoadFixture(FixtureRoot, "floor-says-it-once")
	if err != nil {
		t.Fatal(err)
	}
	if f.World.Repo != RepoAbsent {
		t.Fatalf("the fixture no longer declares an absent repository: %q", f.World.Repo)
	}
	prompt, err := f.PromptIn(context.Background(), "", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"NOT a git repository",
		"offer `git init`",
		"Say this once",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("the prefix does not carry %q, so nothing asks the model for what the contract measures", want)
		}
	}
}
