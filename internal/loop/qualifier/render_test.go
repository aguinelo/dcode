package qualifier

import (
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/config"
)

func sample() []Measured {
	return []Measured{
		{Proposed: Proposed{Name: "unit", Command: "pnpm test", Expects: ExpectFail, Why: "o trabalho tem que fazer isto passar"},
			Class: ClassAcceptance, Exit: 1},
		{Proposed: Proposed{Name: "lint", Command: "pnpm lint", Expects: ExpectPass},
			Class: ClassRegression, Exit: 0},
		{Proposed: Proposed{Name: "typo", Command: "pnmp typecheck", Expects: ExpectFail},
			Class: ClassBroken, Exit: 127},
		{Proposed: Proposed{Name: "weak", Command: "test -f x", Expects: ExpectFail, ExitCode: 2},
			Class: ClassRegression, Exit: 2, Mismatch: true},
	}
}

// The file is the review surface: it is diffable, it outlives the session, and
// it is what the next run reads. So the measurement goes IN it, beside each
// criterion — a number that only appeared on a screen is one nobody can go
// back to.
func TestTheProposalCarriesItsMeasurement(t *testing.T) {
	got := string(Render(sample(), []string{"tests/**"}, Conditions{}))

	for _, want := range []string{
		`[unit]`, `command = "pnpm test"`,
		"# now: acceptance (exit 1)",
		"# now: regression (exit 0)",
		"# now: broken (exit 127)",
		"o trabalho tem que fazer isto passar",
		`protected = "tests/**"`,
		`exit_code = "2"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the file does not carry %q:\n%s", want, got)
		}
	}
}

// A criterion that did the opposite of what was claimed says so, and a broken
// one says it measured the absence of a tool rather than of work. Those are
// the two lines a reviewer's eye should land on.
func TestTheProposalNamesWhatWentWrong(t *testing.T) {
	got := string(Render(sample(), nil, Conditions{}))
	if !strings.Contains(got, "did the opposite") {
		t.Errorf("a contradicted criterion is not flagged:\n%s", got)
	}
	if !strings.Contains(got, "absence of a tool") {
		t.Errorf("a broken criterion is not explained:\n%s", got)
	}
}

// A set with nothing red says so at the top of the file, because it will
// report done with nothing having changed.
func TestASetWithNothingRedWarnsInTheFile(t *testing.T) {
	got := string(Render(sample()[1:2], nil, Conditions{NoAcceptance: true}))
	if !strings.Contains(got, "NOTHING HERE IS RED") {
		t.Errorf("a set with nothing red does not say so:\n%s", got)
	}
	// And it does not refuse: a genuine refactor has nothing new to prove.
	if !strings.Contains(got, "[lint]") {
		t.Errorf("the criteria were dropped:\n%s", got)
	}
}

// The summary goes to the PERSON, and carries the measurement, because the
// turn that proposed has already ended by the time anything is measured.
func TestTheSummaryTellsThePersonWhatHappened(t *testing.T) {
	got := Summary(sample(), Conditions{NoAcceptance: true}, "specs/x/done.toml")
	for _, want := range []string{
		"specs/x/done.toml", "unit", "acceptance", "broken",
		"proposed as", "Nothing here is red", "commented out",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the summary does not carry %q:\n%s", want, got)
		}
	}
}

// The rendered file is what the loader reads back. Round-tripping it is the
// only thing that makes the proposal usable at all.
func TestTheProposalIsWhatTheLoaderReads(t *testing.T) {
	got := string(Render(sample(), []string{"tests/**"}, Conditions{}))
	if strings.Count(got, "[") < 4 {
		t.Fatalf("not every criterion made it into the file:\n%s", got)
	}
	set, err := load(t, got)
	if err != nil {
		t.Fatalf("the file the qualifier writes does not parse: %v\n%s", err, got)
	}
	if len(set) != 3 {
		t.Fatalf("the loader reads back %d criteria, want the 3 that are not broken:\n%s", len(set), got)
	}
}

// A broken criterion is written down and not declared.
//
// It used to be declared, and the file is what the next run loads: the work
// session was then measured against a command that does not exist — red
// forever, so the loop could never finish — and the folder now declared a
// criterion, so it could never be sent back through qualification either. Two
// dead ends from one line.
func TestABrokenCriterionIsWrittenDownAndNotDeclared(t *testing.T) {
	got := string(Render(sample(), nil, Conditions{}))

	set, err := load(t, got)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := set["typo"]; ok {
		t.Errorf("the broken criterion is declared, so the next run will be measured against it:\n%s", got)
	}
	// Written down, though: commenting it away entirely would hide the one
	// thing the person has to fix.
	for _, want := range []string{"# [typo]", "pnmp typecheck", "commented out"} {
		if !strings.Contains(got, want) {
			t.Errorf("the broken criterion left no trace of %q:\n%s", want, got)
		}
	}
}

// load reads the rendered file the way the product reads it back.
//
// Through config.ParseSections, which is what app.loadDoneSet uses. Anything
// else here would be a second parser agreeing with itself.
func load(t *testing.T, rendered string) (map[string]string, error) {
	t.Helper()
	sections, err := config.ParseSections(rendered, "done.toml")
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, name := range sections.Order {
		if name == "" {
			continue
		}
		out[name] = sections.Values[name]["command"]
	}
	return out, nil
}
