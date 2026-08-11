package loop

import (
	"context"
	"errors"
	"os"
	osexec "os/exec"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

// ---------- Progressed ----------

// The exit condition, and the reason it is progress rather than perfection.
func TestProgressedRequiresTheUnmetSetToShrinkStrictly(t *testing.T) {
	cases := []struct {
		name          string
		before, after []string
		want          bool
	}{
		{"one fixed", []string{"tests", "lint"}, []string{"tests"}, true},
		{"all fixed", []string{"tests"}, nil, true},
		{"nothing moved", []string{"tests", "lint"}, []string{"tests", "lint"}, false},
		{"same size, different names", []string{"tests"}, []string{"lint"}, false},
		// A set that GROWS is a regression, not movement.
		{"grew", []string{"tests"}, []string{"tests", "lint"}, false},
		// Shrinking while introducing something new is not progress on what was
		// already broken: two failures became one different failure.
		{"shrank but swapped", []string{"tests", "lint", "vet"}, []string{"build"}, false},
		{"nothing to nothing", nil, nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Progressed(c.before, c.after); got != c.want {
				t.Errorf("Progressed(%v, %v) = %v, want %v", c.before, c.after, got, c.want)
			}
		})
	}
}

// ---------- Check ----------

func runner(codes map[string]int, errs map[string]error) CriterionRunner {
	return func(_ context.Context, cmd string) (int, string, error) {
		if err, ok := errs[cmd]; ok {
			return 0, "", err
		}
		return codes[cmd], "", nil
	}
}

func TestCheckClassifiesByExitCode(t *testing.T) {
	set := DoneSet{Criteria: []Criterion{
		{Name: "tests", Command: "make test"},
		{Name: "lint", Command: "make lint"},
	}}
	rep := Check(context.Background(), set,
		runner(map[string]int{"make test": 0, "make lint": 1}, nil), 0)

	if rep.States["tests"] != CriterionMet {
		t.Errorf("tests = %v, want met", rep.States["tests"])
	}
	if rep.States["lint"] != CriterionUnmet {
		t.Errorf("lint = %v, want unmet", rep.States["lint"])
	}
}

func TestACriterionMayDeclareANonZeroExitCode(t *testing.T) {
	set := DoneSet{Criteria: []Criterion{{Name: "absent", Command: "grep -q TODO .", ExitCode: 1}}}
	rep := Check(context.Background(), set, runner(map[string]int{"grep -q TODO .": 1}, nil), 0)
	if rep.States["absent"] != CriterionMet {
		t.Fatalf("a criterion met by exit 1 was read as unmet")
	}
}

// A command that could not run at all is not the criterion failing. Reading the
// two the same would turn a missing binary into a behavioural verdict.
func TestACommandThatCannotRunIsUnavailableNotUnmet(t *testing.T) {
	set := DoneSet{Criteria: []Criterion{{Name: "tests", Command: "make test"}}}
	rep := Check(context.Background(), set,
		runner(nil, map[string]error{"make test": errors.New("executable file not found")}), 0)

	if rep.States["tests"] != CriterionUnavailable {
		t.Fatalf("tests = %v, want unavailable", rep.States["tests"])
	}
}

func TestAnExitErrorIsStillAVerdict(t *testing.T) {
	// exec.ExitError means the command ran and failed, which IS the criterion
	// being unmet — unlike a command that never started.
	set := DoneSet{Criteria: []Criterion{{Name: "tests", Command: "make test"}}}
	rep := Check(context.Background(), set,
		func(_ context.Context, _ string) (int, string, error) {
			return 1, "", &osexec.ExitError{}
		}, 0)
	if rep.States["tests"] != CriterionUnmet {
		t.Fatalf("tests = %v, want unmet", rep.States["tests"])
	}
}

func TestAnEmptyCommandIsUnavailable(t *testing.T) {
	set := DoneSet{Criteria: []Criterion{{Name: "tests", Command: "  "}}}
	rep := Check(context.Background(), set, runner(nil, nil), 0)
	if rep.States["tests"] != CriterionUnavailable {
		t.Fatalf("an empty command reported %v, want unavailable", rep.States["tests"])
	}
}

func TestNoRunnerLeavesEverythingUnavailable(t *testing.T) {
	set := DoneSet{Criteria: []Criterion{{Name: "tests", Command: "make test"}}}
	rep := Check(context.Background(), set, nil, time.Second)
	if rep.States["tests"] != CriterionUnavailable {
		t.Fatal("with nothing able to run, the state must be unavailable rather than met")
	}
}

// ---------- Unmet ordering ----------

func TestUnmetIsSortedSoTheSetCanBeCompared(t *testing.T) {
	rep := Report{States: map[string]CriterionState{
		"vet": CriterionUnmet, "build": CriterionUnmet,
		"tests": CriterionMet, "lint": CriterionUnmet,
	}}
	want := []string{"build", "lint", "vet"}
	for i := 0; i < 5; i++ {
		if got := rep.Unmet(); !reflect.DeepEqual(got, want) {
			t.Fatalf("Unmet() = %v, want %v — map order leaked into a compared set", got, want)
		}
	}
}

// ---------- Verification, the single-criterion case ----------

func TestVerificationSeal(t *testing.T) {
	cases := []struct {
		name    string
		states  map[string]CriterionState
		changed bool
		stale   bool
		want    Verification
	}{
		{"nothing changed", map[string]CriterionState{"v": CriterionMet}, false, false, VerificationClean},
		{"changed and passed", map[string]CriterionState{"v": CriterionMet}, true, false, VerificationPassed},
		{"changed and failed", map[string]CriterionState{"v": CriterionUnmet}, true, false, VerificationFailed},
		{"changed, nothing to run", map[string]CriterionState{"v": CriterionUnavailable}, true, false, VerificationUnavailable},
		{"changed, no criteria at all", nil, true, false, VerificationUnavailable},
		{"read-only session", nil, false, false, VerificationClean},

		// The state that had no producer. "passed" is documented as "ran after
		// the last edit"; edit after the check and the check describes code that
		// no longer exists.
		{"passed, then edited again", map[string]CriterionState{"v": CriterionMet}, true, true, VerificationStale},

		// Staleness only ever takes away reassurance. A failure that was
		// followed by an edit is still a failure worth showing: overriding it
		// with "stale" would replace a red seal with a neutral one, which is the
		// wrong direction to be wrong in.
		{"failed, then edited again", map[string]CriterionState{"v": CriterionUnmet}, true, true, VerificationFailed},
		{"nothing to run, then edited", map[string]CriterionState{"v": CriterionUnavailable}, true, true, VerificationUnavailable},
		{"stale is meaningless with no change", map[string]CriterionState{"v": CriterionMet}, false, true, VerificationClean},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := VerificationOf(Report{States: c.states}, c.changed, c.stale); got != c.want {
				t.Errorf("seal = %v, want %v", got, c.want)
			}
		})
	}
}

// ---------- Protected paths ----------

// Without this the rest is theatre. The agent must not be able to quietly move
// the thing that measures it.
func TestProtectedPathsAreSurfaced(t *testing.T) {
	set := DoneSet{Protected: []string{"**/*_test.go", "testdata/**"}}
	written := []string{
		"internal/loop/turn.go",
		"internal/loop/turn_test.go",
		"testdata/golden/a.txt",
		"README.md",
	}
	got := set.TouchedProtected(written)
	want := []string{"internal/loop/turn_test.go", "testdata/golden/a.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("touched protected = %v, want %v", got, want)
	}
}

func TestNoProtectedPatternsMeansNothingIsFlagged(t *testing.T) {
	if got := (DoneSet{}).TouchedProtected([]string{"a_test.go"}); len(got) != 0 {
		t.Fatalf("flagged %v with no protected patterns declared", got)
	}
}

func TestTheReportNamesTouchedProtectedPaths(t *testing.T) {
	rep := Report{
		States:           map[string]CriterionState{"tests": CriterionMet},
		TouchedProtected: []string{"internal/loop/turn_test.go"},
	}
	out := rep.String()
	if !strings.Contains(out, "turn_test.go") || !strings.Contains(out, "protected") {
		t.Fatalf("the report hides a change to what measures it:\n%s", out)
	}
}

// A declared state nothing can produce is the defect this guard exists for, and
// it is the one that was here: VerificationStale was declared, rendered by two
// branches of the client, and returned by no input at all.
//
// Stronger than a source sweep for the same reason a test beats a comment: it
// exercises the full input space and asks which values come OUT. A constant
// mentioned in a comment satisfies a substring search; nothing satisfies this
// but a reachable state.
func TestEveryVerificationStateIsReachable(t *testing.T) {
	declared := map[Verification]bool{}
	src, err := os.ReadFile("done.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range regexp.MustCompile(`Verification\s*=\s*"([a-z]+)"`).FindAllStringSubmatch(string(src), -1) {
		declared[Verification(m[1])] = false
	}
	if len(declared) < 2 {
		t.Fatalf("parsed %d states from done.go; the guard would pass vacuously", len(declared))
	}

	everyState := []map[string]CriterionState{
		nil,
		{"v": CriterionMet},
		{"v": CriterionUnmet},
		{"v": CriterionUnavailable},
	}
	for _, states := range everyState {
		for _, changed := range []bool{true, false} {
			for _, stale := range []bool{true, false} {
				declared[VerificationOf(Report{States: states}, changed, stale)] = true
			}
		}
	}

	for state, reached := range declared {
		if !reached {
			t.Errorf("Verification %q is declared and no combination of report, "+
				"change and staleness produces it — a seal the client renders and "+
				"the loop can never set", state)
		}
	}
}

// The seal is a function of two records, and the client shows it as a fact
// about the turn. If it could vary between two evaluations of the same records,
// it would be a fact about when it was asked instead — and a person comparing
// what they saw with what the transcript says would find them disagreeing with
// no way to tell which was wrong.
func TestTheSealIsAFunctionOfTheRecordsAndNothingElse(t *testing.T) {
	rep := Report{States: map[string]CriterionState{
		"tests": CriterionMet, "lint": CriterionMet, "vet": CriterionUnavailable,
	}}
	first := VerificationOf(rep, true, false)
	for i := 0; i < 50; i++ {
		if got := VerificationOf(rep, true, false); got != first {
			t.Fatalf("run %d gave %v, first gave %v — the seal reads something outside its arguments", i, got, first)
		}
	}
	// Map iteration order is the realistic way a seal derived from a report
	// would start varying: several states, one of them decisive.
	if first != VerificationUnavailable {
		t.Fatalf("seal = %v; with an unavailable criterion present the answer is fixed regardless of iteration order", first)
	}
}
