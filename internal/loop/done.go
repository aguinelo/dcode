package loop

import (
	"context"
	"errors"
	"fmt"
	osexec "os/exec"
	"sort"
	"strings"
	"time"

	"github.com/aguinelo/dcode/internal/policy"
)

// Criterion is a condition of done that can be CHECKED.
//
// Prose does not qualify. "Tests pass" is a fact; "the code is clean" is not,
// and a criterion judged by the model hands the decision about done back to the
// model — now with twenty turns spent on the way there.
type Criterion struct {
	Name     string // how it appears in the report
	Command  string // what runs
	ExitCode int    // what counts as met; zero by default
}

// CriterionState is the outcome of checking one.
type CriterionState string

const (
	CriterionMet         CriterionState = "met"
	CriterionUnmet       CriterionState = "unmet"
	CriterionUnavailable CriterionState = "unavailable" // nothing to run
)

// DoneSet is the definition of done for a workspace.
//
// Protected are the paths that ARE the measurement — test files, typically.
// Without this the rest is theatre: an agent that cannot leave the loop finds
// that the shortest way out is to weaken the thing measuring it, and a false
// test is strictly worse than a false report. A false report is discovered by
// running something; a false test sits in the repository pretending to be
// coverage forever.
//
// It is not a prohibition. Sometimes fixing the test IS the work. It is
// visibility, and that is the whole difference between a loop that ensures
// quality and one that manufactures the appearance of it.
type DoneSet struct {
	Criteria  []Criterion
	Protected []string
}

// Verification is the single-criterion case, which is what the client shows.
//
// There are not two mechanisms. A verify command is a DoneSet of one, and this
// type is how that one is named on screen.
type Verification string

const (
	VerificationClean       Verification = "clean"       // nothing changed; nothing to check
	VerificationPassed      Verification = "passed"      // ran after the last edit, exited zero
	VerificationFailed      Verification = "failed"      // ran, exited non-zero
	VerificationStale       Verification = "stale"       // changed since the last check
	VerificationUnavailable Verification = "unavailable" // changed, and no command is known
)

// Report is what a turn ended knowing.
type Report struct {
	States map[string]CriterionState
	// TouchedProtected are protected paths written during the turn. Surfaced,
	// never counted as progress in silence.
	TouchedProtected []string
}

// Unmet returns the names of the criteria not met, sorted.
//
// Sorted because the set is compared between cycles and printed to a person,
// and a set that reshuffles is one nobody can diff.
func (r Report) Unmet() []string {
	var out []string
	for name, st := range r.States {
		if st == CriterionUnmet {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// Progressed reports whether the unmet set shrank strictly between cycles.
//
// This is the exit condition, and it is deliberately not "everything is met".
// A loop that cannot exit until everything passes has four failure modes, and
// the fourth inverts the result: if the way out is green tests and the agent
// cannot get them, the shortest path out becomes weakening the test. The loop
// would exist to prevent a false report and would produce a false test.
//
// So the loop runs on progress. Strict shrinkage of a set — deterministic,
// cheap, and immune to effort that produces nothing. A set that GROWS is a
// regression and counts as non-progress, not as movement.
func Progressed(before, after []string) bool {
	if len(after) >= len(before) {
		return false
	}
	prev := make(map[string]struct{}, len(before))
	for _, n := range before {
		prev[n] = struct{}{}
	}
	// Shrinking is necessary but not sufficient: swapping two failures for one
	// different failure is not progress on the ones that were already there.
	for _, n := range after {
		if _, ok := prev[n]; !ok {
			return false
		}
	}
	return true
}

// VerificationOf collapses a report of one criterion into the client's seal.
func VerificationOf(r Report, changed bool) Verification {
	if len(r.States) == 0 {
		if changed {
			return VerificationUnavailable
		}
		return VerificationClean
	}
	if !changed {
		return VerificationClean
	}
	for _, st := range r.States {
		switch st {
		case CriterionUnmet:
			return VerificationFailed
		case CriterionUnavailable:
			return VerificationUnavailable
		}
	}
	return VerificationPassed
}

// Runner runs one criterion. Injected so the loop stays testable without
// spawning processes, and so every command still passes through the sandbox.
type CriterionRunner func(ctx context.Context, command string) (exitCode int, output string, err error)

// Check runs every criterion and reports the states.
//
// An unavailable criterion never provokes re-entry: there is nothing to run,
// and insisting only produces another guess. It appears in the final report as
// what could not be checked.
func Check(ctx context.Context, set DoneSet, run CriterionRunner, timeout time.Duration) Report {
	rep := Report{States: map[string]CriterionState{}}
	for _, c := range set.Criteria {
		if strings.TrimSpace(c.Command) == "" || run == nil {
			rep.States[c.Name] = CriterionUnavailable
			continue
		}
		cctx := ctx
		if timeout > 0 {
			var cancel context.CancelFunc
			cctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}
		code, _, err := run(cctx, c.Command)
		switch {
		case err != nil && !isExitError(err):
			// The command could not be run at all — not found, or the timeout
			// fired. That is not the criterion failing, it is the criterion
			// being uncheckable, and the two must not read the same.
			rep.States[c.Name] = CriterionUnavailable
		case code == c.ExitCode:
			rep.States[c.Name] = CriterionMet
		default:
			rep.States[c.Name] = CriterionUnmet
		}
	}
	return rep
}

func isExitError(err error) bool {
	var ee *osexec.ExitError
	return errors.As(err, &ee)
}

// TouchedProtected returns the written paths that match a protected pattern.
//
// Matching reuses policy.Glob, which already exists and already governs the
// confirm-write rules — one glob dialect in the product, not two.
func (d DoneSet) TouchedProtected(written []string) []string {
	var out []string
	for _, w := range written {
		for _, pat := range d.Protected {
			if policy.Glob(pat, w) {
				out = append(out, w)
				break
			}
		}
	}
	sort.Strings(out)
	return out
}

// Names returns the criteria in a given state, sorted.
func (r Report) Names(want CriterionState) []string {
	var out []string
	for name, st := range r.States {
		if st == want {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// String renders the report for a person.
func (r Report) String() string {
	if len(r.States) == 0 {
		return "no definition of done"
	}
	names := make([]string, 0, len(r.States))
	for n := range r.States {
		names = append(names, n)
	}
	sort.Strings(names)

	var b strings.Builder
	for _, n := range names {
		fmt.Fprintf(&b, "  %-24s %s\n", n, r.States[n])
	}
	if len(r.TouchedProtected) > 0 {
		fmt.Fprintf(&b, "  protected paths changed during this turn: %s\n",
			strings.Join(r.TouchedProtected, ", "))
	}
	return b.String()
}
