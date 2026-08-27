package qualifier

import (
	"strings"
	"testing"
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

// What comes back to the model carries the measurement, so a broken or
// contradicted criterion is corrected by whoever wrote it.
func TestTheSummaryTellsTheModelWhatHappened(t *testing.T) {
	got := Summary(sample(), Conditions{NoAcceptance: true}, "specs/x/done.toml")
	for _, want := range []string{
		"specs/x/done.toml", "unit", "acceptance", "broken",
		"you said it would", "Nothing here is red", "do not start the work",
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
	// Commented lines carry the measurement and must not be mistaken for
	// criteria: every section header starts a line.
	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(line, "#") && strings.Contains(line, "command =") {
			t.Errorf("a comment carries a command and could be read as one: %q", line)
		}
	}
}
