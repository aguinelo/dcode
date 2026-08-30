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

// MaxCriterionOutput is how much of one failing criterion reaches the model.
//
// The same number as qualifier.MaxOutput, and the same by decision rather than
// by coincidence: it is the same information, from the same runner, read by
// different readers. Two ceilings for one concept would be two behaviours.
//
// Per criterion, never per report. A set with four red criteria delivers four
// blocks, because cutting the fourth on account of the first three would hide
// whatever the map's iteration order happened to hide — and a map's order is
// not a product decision.
const MaxCriterionOutput = 2000

// Output is what one criterion printed, bounded.
type Output struct {
	Text string
	// Truncated marks output that did not fit. Nothing in this codebase cuts
	// output without saying so.
	Truncated bool
}

// Report is what a turn ended knowing.
type Report struct {
	States map[string]CriterionState
	// TouchedProtected are protected paths written during the turn. Surfaced,
	// never counted as progress in silence.
	TouchedProtected []string
	// Outputs is what each criterion that did not pass printed, by name.
	//
	// A separate map rather than a field on the state: CriterionState is an
	// enum compared between cycles and printed to a person, and hanging text
	// off it would change what that comparison means. Progressed still reads
	// names and nothing else.
	//
	// Only the ones that did not pass. A green criterion's output is noise
	// paid for on every round, and what it had to say was said by its exit
	// code. This is what the loop used to throw away on the line that ran the
	// command — the model was told a criterion had failed and never what
	// broke, while the qualifier, reading the same runner, kept it.
	Outputs map[string]Output
}

// OutputTexts is what the unmet criteria printed, ready for the prefix.
//
// The truncation marker is added HERE rather than stored, because it is a
// sentence for a reader and Output.Truncated is a fact about the bytes. Storing
// the sentence would put prose in a struct the loop compares, and would have to
// be written in the product's voice by whoever ran the command.
func (r Report) OutputTexts() map[string]string {
	if len(r.Outputs) == 0 {
		return nil
	}
	out := make(map[string]string, len(r.Outputs))
	for name, o := range r.Outputs {
		text := o.Text
		if o.Truncated {
			text = "(only the last " + itoa(MaxCriterionOutput) + " bytes)\n" + text
		}
		out[name] = text
	}
	return out
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

// Movement is what one cycle did to the unmet set.
//
// Three answers where there used to be two. Progressed returned a boolean, so
// drawing and regressing collapsed into "not progress" and the loop counted
// both as a stall — it knew a cycle had made things worse and the fact died on
// that line, because there was nothing to do with it.
type Movement int

const (
	// MovedForward: the set shrank and everything left was already in it.
	MovedForward Movement = iota
	// MovedNowhere: nothing got better and nothing got worse.
	MovedNowhere
	// MovedBackward: something that was met is not met any more.
	MovedBackward
)

func (m Movement) String() string {
	switch m {
	case MovedForward:
		return "forward"
	case MovedBackward:
		return "backward"
	}
	return "nowhere"
}

// Moved classifies one cycle against the last.
//
// This is the exit condition, and it is deliberately not "everything is met".
// A loop that cannot exit until everything passes has four failure modes, and
// the fourth inverts the result: if the way out is green tests and the agent
// cannot get them, the shortest path out becomes weakening the test. The loop
// would exist to prevent a false report and would produce a false test.
//
// So the loop runs on movement — deterministic, cheap, and immune to effort
// that produces nothing.
//
// Regression is a name in `after` that was not in `before`: the criterion
// passed, and now it does not. Deliberately narrow — it is the only reading
// that justifies throwing away a cycle's work, and a wider one would undo
// cycles that merely failed to finish.
//
// Swapping one failure for another is REGRESSION, not a draw. {a,b} → {a,c}
// means c passed and stopped passing, and the fact that b was fixed in the same
// cycle does not put c back. Progressed already refused to call this progress;
// what changes is that it now has a consequence.
func Moved(before, after []string) Movement {
	had := make(map[string]struct{}, len(before))
	for _, n := range before {
		had[n] = struct{}{}
	}
	for _, n := range after {
		if _, ok := had[n]; !ok {
			return MovedBackward
		}
	}
	// Everything still unmet was already unmet. Fewer of them is work done;
	// the same number is a cycle that changed nothing about the ruler.
	if len(after) < len(before) {
		return MovedForward
	}
	return MovedNowhere
}

// VerificationOf collapses a report of one criterion into the client's seal.
//
// stale says a file was written AFTER the check ran. It is the boolean the seal
// was missing: without it "passed" means "the check exited zero at some point
// this turn", which is true of a session that ran the suite and then edited
// everything it touched. That is the precise false confidence this whole
// mechanism exists to prevent, wearing the badge that says it was prevented.
func VerificationOf(r Report, changed, stale bool) Verification {
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
	// Everything the check looked at was met — but only the check that ran
	// before the last edit. Staleness is applied HERE and nowhere earlier so it
	// can only ever remove reassurance: a failure followed by an edit stays a
	// failure, because replacing a red seal with a neutral one is the wrong
	// direction to be wrong in.
	if stale {
		return VerificationStale
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
		code, out, err := run(cctx, c.Command)
		switch {
		case err != nil && !isExitError(err):
			// The command could not be run at all — not found, or the timeout
			// fired. That is not the criterion failing, it is the criterion
			// being uncheckable, and the two must not read the same.
			//
			// No output is kept either: there was nothing to print, and
			// keeping whatever the failed launch wrote would put the harness's
			// own words where a criterion's belong.
			rep.States[c.Name] = CriterionUnavailable
		case code == c.ExitCode:
			rep.States[c.Name] = CriterionMet
		default:
			rep.States[c.Name] = CriterionUnmet
			if text, cut := tail(out, MaxCriterionOutput); text != "" {
				if rep.Outputs == nil {
					rep.Outputs = map[string]Output{}
				}
				rep.Outputs[c.Name] = Output{Text: text, Truncated: cut}
			}
		}
	}
	return rep
}

// tail keeps the last max bytes and reports whether it cut.
//
// The END, never the beginning. A test runner's summary, its failure count and
// its last assertion are at the bottom; the header is what can be lost. Cutting
// the tail to preserve the banner would preserve exactly the part that decides
// nothing.
//
// On a line boundary when there is one inside the window, and on the byte when
// there is not: an 8000-character line is machine output, and half of it beats
// nothing at all.
func tail(s string, max int) (string, bool) {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return "", false
	}
	if len(s) <= max {
		return s, false
	}
	cut := s[len(s)-max:]
	if i := strings.IndexByte(cut, '\n'); i >= 0 && i+1 < len(cut) {
		cut = cut[i+1:]
	}
	return cut, true
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

// itoa keeps this file free of a fmt import for one number.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
