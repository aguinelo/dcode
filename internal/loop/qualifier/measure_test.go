package qualifier

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/loop"
)

// runner answers each command from a table, and records what it was asked.
func runner(t *testing.T, answers map[string]struct {
	exit   int
	output string
	err    error
}) (loop.CriterionRunner, *[]string) {
	t.Helper()
	var seen []string
	return func(_ context.Context, cmd string) (int, string, error) {
		seen = append(seen, cmd)
		a, ok := answers[cmd]
		if !ok {
			t.Fatalf("the runner was asked for %q, which the test did not set up", cmd)
		}
		return a.exit, a.output, a.err
	}, &seen
}

type answer = struct {
	exit   int
	output string
	err    error
}

// The rule the family is named for, in both directions.
//
// A criterion that FAILS before the work is acceptance: it can testify that
// the work happened. One that PASSES is a regression guard: its job is to stay
// green, and a green suite at this point is exactly what is wanted from it.
//
// Without the first half, a criterion that was already green would come out of
// the loop green and be read as evidence — when it would have been green with
// nothing done at all.
func TestFailingIsAcceptanceAndPassingIsRegression(t *testing.T) {
	run, _ := runner(t, map[string]answer{
		"pnpm test:new": {exit: 1, output: "no such test"},
		"pnpm test":     {exit: 0, output: "all good"},
	})
	got, cond, err := Measure(context.Background(), Proposal{Criteria: []Proposed{
		{Name: "1", Command: "pnpm test:new", Expects: ExpectFail},
		{Name: "2", Command: "pnpm test", Expects: ExpectPass},
	}}, run, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Class != ClassAcceptance {
		t.Errorf("a criterion that failed is %q, want acceptance", got[0].Class)
	}
	if got[1].Class != ClassRegression {
		t.Errorf("a criterion that passed is %q, want regression", got[1].Class)
	}
	if got[0].Mismatch || got[1].Mismatch {
		t.Errorf("agreement was reported as a mismatch: %+v", got)
	}
	if cond.NoAcceptance {
		t.Error("a set with an acceptance criterion was reported as having none")
	}
}

// "Passed" is compared against the declared ExitCode, never against zero.
//
// A criterion declared with exit 1 is met by exiting 1. Comparing to zero
// would classify an already-green criterion as acceptance, which is the exact
// error this package exists to avoid.
func TestPassingIsComparedAgainstTheDeclaredExitCode(t *testing.T) {
	run, _ := runner(t, map[string]answer{"grep -q TODO .": {exit: 1}})
	got, _, err := Measure(context.Background(), Proposal{Criteria: []Proposed{
		{Name: "1", Command: "grep -q TODO .", ExitCode: 1, Expects: ExpectPass},
	}}, run, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Class != ClassRegression {
		t.Errorf("exit 1 against a declared exit 1 is %q, want regression", got[0].Class)
	}
}

// A command that does not exist fails, and by failing it disguises itself as
// acceptance while measuring the absence of a tool rather than the absence of
// work. It would stay red forever.
func TestNothingToRunIsBrokenAndNotRed(t *testing.T) {
	run, _ := runner(t, map[string]answer{
		"pnmp test":   {exit: 127, output: "pnmp: command not found"},
		"./script.sh": {exit: 126, output: "permission denied"},
		"boom":        {exit: 0, err: errors.New("could not start")},
	})
	got, cond, err := Measure(context.Background(), Proposal{Criteria: []Proposed{
		{Name: "1", Command: "pnmp test", Expects: ExpectFail},
		{Name: "2", Command: "./script.sh", Expects: ExpectFail},
		{Name: "3", Command: "boom", Expects: ExpectFail},
	}}, run, 0)
	if err != nil {
		t.Fatal(err)
	}
	for i, m := range got {
		if m.Class != ClassBroken {
			t.Errorf("criterion %d is %q, want broken", i+1, m.Class)
		}
		if m.Mismatch {
			t.Errorf("criterion %d reported a mismatch; broken is its own alarm", i+1)
		}
	}
	if !cond.NoAcceptance {
		t.Error("a set of nothing but broken criteria was not reported as having no acceptance")
	}
}

// The disagreement is the most informative line the operator can be given.
//
// Without Expects, "criterion 2 passed" is a neutral fact. With it, it is a
// fact AGAINST what was declared — the exact signature of a criterion that
// does not measure what it should.
func TestTheDisagreementIsFlagged(t *testing.T) {
	run, _ := runner(t, map[string]answer{
		"test -f README.md": {exit: 0},
		"pnpm test":         {exit: 1},
	})
	got, _, err := Measure(context.Background(), Proposal{Criteria: []Proposed{
		{Name: "1", Command: "test -f README.md", Expects: ExpectFail},
		{Name: "2", Command: "pnpm test", Expects: ExpectPass},
	}}, run, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !got[0].Mismatch {
		t.Error("said it would fail and it passed; that is not flagged")
	}
	if !got[1].Mismatch {
		t.Error("said it would pass and it failed; that is not flagged")
	}
}

// Nothing red means the turn will report done without anything having to
// change. Named, never refused: a genuine refactor has nothing new to prove,
// and the harness cannot tell that from a proposal empty of content.
func TestASetWithNothingRedIsNamedAndNotRefused(t *testing.T) {
	run, _ := runner(t, map[string]answer{"pnpm test": {exit: 0}, "pnpm lint": {exit: 0}})
	got, cond, err := Measure(context.Background(), Proposal{Criteria: []Proposed{
		{Name: "1", Command: "pnpm test", Expects: ExpectPass},
		{Name: "2", Command: "pnpm lint", Expects: ExpectPass},
	}}, run, 0)
	if err != nil {
		t.Fatalf("a set with nothing red was refused: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d measured", len(got))
	}
	if !cond.NoAcceptance {
		t.Error("a set with nothing red was not named as such")
	}
}

// A proposal with no criteria is an error, never an empty DoneSet.
//
// An empty DoneSet means "nothing to verify", which the loop reports as done.
// The absence of a definition is not a permissive definition.
func TestAnEmptyProposalIsAnError(t *testing.T) {
	run, _ := runner(t, nil)
	_, _, err := Measure(context.Background(), Proposal{}, run, 0)
	if !errors.Is(err, ErrEmptyProposal) {
		t.Fatalf("an empty proposal gave %v, want ErrEmptyProposal", err)
	}
}

// A proposal nobody can run is a proposal nobody can classify.
func TestNoRunnerIsAnError(t *testing.T) {
	_, _, err := Measure(context.Background(), Proposal{Criteria: []Proposed{{Name: "1", Command: "x"}}}, nil, 0)
	if err == nil {
		t.Fatal("measuring with no runner came back with no error")
	}
}

// Every criterion runs exactly once, and in the order it was proposed. Twice
// would be a different measurement of the same thing, and out of order would
// make the report unreadable next to the proposal.
func TestEveryCriterionRunsOnceInOrder(t *testing.T) {
	run, seen := runner(t, map[string]answer{"a": {exit: 1}, "b": {exit: 0}, "c": {exit: 1}})
	if _, _, err := Measure(context.Background(), Proposal{Criteria: []Proposed{
		{Name: "1", Command: "a"}, {Name: "2", Command: "b"}, {Name: "3", Command: "c"},
	}}, run, 0); err != nil {
		t.Fatal(err)
	}
	if want := []string{"a", "b", "c"}; len(*seen) != 3 {
		t.Fatalf("ran %v, want %v exactly once each", *seen, want)
	}
	for i, want := range []string{"a", "b", "c"} {
		if (*seen)[i] != want {
			t.Fatalf("ran %v, out of proposal order", *seen)
		}
	}
}

// The output is what separates a criterion red because the work is missing
// from one red because the world is. It is capped, and it says when it was.
func TestOutputIsCappedAndSaysSo(t *testing.T) {
	run, _ := runner(t, map[string]answer{"x": {exit: 1, output: strings.Repeat("y", MaxOutput*2)}})
	got, _, err := Measure(context.Background(), Proposal{Criteria: []Proposed{{Name: "1", Command: "x"}}}, run, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got[0].Output) != MaxOutput || !got[0].Truncated {
		t.Errorf("output is %d bytes, truncated=%v", len(got[0].Output), got[0].Truncated)
	}
}

// A timeout bounds one criterion. A check that never finishes is not a check.
func TestATimeoutBoundsOneCriterion(t *testing.T) {
	run := loop.CriterionRunner(func(ctx context.Context, _ string) (int, string, error) {
		<-ctx.Done()
		return 0, "", ctx.Err()
	})
	start := time.Now()
	got, _, err := Measure(context.Background(), Proposal{Criteria: []Proposed{{Name: "1", Command: "sleep"}}}, run, 50*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("the timeout did not bound the run")
	}
	// A criterion that could not be run at all is broken, not red: it measures
	// nothing about the work.
	if got[0].Class != ClassBroken {
		t.Errorf("a timed-out criterion is %q, want broken", got[0].Class)
	}
}

// Measure never writes, and the only thing it may do is call the runner. The
// proposal it was handed comes back unchanged.
func TestMeasureLeavesTheProposalAlone(t *testing.T) {
	run, _ := runner(t, map[string]answer{"a": {exit: 1}})
	p := Proposal{Criteria: []Proposed{{Name: "1", Command: "a", Expects: ExpectFail, Why: "because"}}, Protected: []string{"x"}}
	if _, _, err := Measure(context.Background(), p, run, 0); err != nil {
		t.Fatal(err)
	}
	if p.Criteria[0].Command != "a" || p.Criteria[0].Why != "because" || len(p.Protected) != 1 {
		t.Errorf("the proposal was modified: %+v", p)
	}
}
