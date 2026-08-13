// Package evals measures the behavioural contracts declared in the `.p` specs.
//
// A behavioural contract is not verifiable by assertion. It is a rate over
// repeated runs against a real model, compared with a threshold — the regime
// every `.r.spec.md` in this project declares for itself. That is why this
// package sits outside the coverage denominator and why everything that
// reaches a model is behind the `eval` build tag: measuring costs money, and a
// suite that costs money is a suite nobody runs.
//
// What lives here without the tag is the arithmetic, and it is here rather
// than beside the tagged code on purpose. Deciding whether a measurement met
// its threshold is exactly the kind of thing that must not itself be measured.
package evals

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/aguinelo/dcode/internal/version"
)

// DefaultRuns is how many times a scenario runs when nothing says otherwise.
//
// Twenty is the floor the provider spec justifies: below it the confidence
// interval is wider than the distance between the thresholds being checked, so
// a smaller number does not measure faster, it measures nothing.
const DefaultRuns = 20

// Demanding is the threshold above which twenty runs stop being enough, and
// DemandingRuns is what such a contract measures instead.
//
// The arithmetic is the whole argument. Over twenty runs one failure moves the
// rate five points, so a measurement cannot distinguish 90% from 100% — and a
// contract declaring 95% is asking exactly that question. Fifty runs put one
// failure at two points, which is finer than the distance being judged.
//
// It is not free: nineteen of the thirty-five contracts sit at or above this
// line, so the suite roughly doubles in time and in spend. That is the price of
// a number that means what it says, and the alternative was lowering every
// strong claim the product makes — including the ones about not bypassing
// safety — to what a cheap measurement can support.
const (
	Demanding     = 0.95
	DemandingRuns = 50
)

// Reader is the slice of resolved configuration this package needs.
//
// An interface rather than *config.Resolved because the harness has no business
// knowing where a value came from, and because a fake is then three lines.
type Reader interface {
	Bool(key string, def bool) bool
	String(key, def string) string
	Int(key string, def int) int
}

// Config is what a measurement run needs to know about itself.
type Config struct {
	Enabled bool
	Model   string
	Runs    int
}

// FromReader pulls the three eval keys out of resolved configuration.
//
// A run count at or below zero falls back to the default rather than erroring:
// the value arrives from a file a human typed, and refusing to measure is a
// worse answer to `runs = 0` than measuring the standard number of times.
func FromReader(r Reader) Config {
	runs := r.Int("eval.runs", DefaultRuns)
	if runs <= 0 {
		runs = DefaultRuns
	}
	return Config{
		Enabled: r.Bool("eval.enabled", false),
		Model:   r.String("eval.model", ""),
		Runs:    runs,
	}
}

// Unavailable reports why a measurement cannot run, or "" if it can.
//
// A reason rather than a boolean because this is what a skipped test prints,
// and "eval skipped" tells nobody which of the two knobs was missing.
func (c Config) Unavailable() string {
	if !c.Enabled {
		return "eval is off: set eval.enabled (DCODE_EVAL_ENABLED=true). It runs a real model and costs money, which is why it is never on by default."
	}
	if c.Model == "" {
		return "no eval model: set eval.model (DCODE_EVAL_MODEL). A threshold belongs to a model, so a measurement that cannot name one measures nothing."
	}
	return ""
}

// Result is one scenario measured.
//
// Model and Build travel with the number because a threshold belongs to a
// model, not to a scenario: the same fixture against a different model is a
// different measurement, and a result quoted without them cannot be compared
// with anything.
type Result struct {
	ID        string
	Threshold float64
	// Planned is how many runs the contract asked for; Runs is how many
	// actually happened.
	//
	// They differ only when a deadline cut the measurement short, and keeping
	// both is what lets that be reported instead of guessed at. Collapsing
	// them either invents runs that never ran or loses the fact that the
	// number is short of what the threshold needs.
	Planned int
	Runs    int
	Passed  int
	Errors  int
	Model   string
	Build   string
	// FirstError is why the errored runs errored, or empty when none did.
	//
	// The count alone cannot be acted on. A measurement that reports "20
	// run(s) errored" and nothing else costs an afternoon and a provider bill
	// and leaves nobody able to say whether it was a rate limit, a revoked
	// key, or a bug in the harness — which is the same defect as a rate with
	// no transcript behind it, in the one place it is most expensive.
	FirstError string
}

// Rate is the share of runs that behaved as contracted.
//
// Errored runs are in the denominator on purpose: a run that could not be
// measured is not a run that behaved.
func (r Result) Rate() float64 {
	if r.Runs <= 0 {
		return 0
	}
	return float64(r.Passed) / float64(r.Runs)
}

// Sound reports whether the measurement is worth comparing at all.
//
// A run that errored failed to measure the model, which is a different thing
// from the model failing the contract — a network blip must never be readable
// as a behavioural regression. Zero runs is the same class of non-answer.
func (r Result) Sound() bool {
	// A measurement cut short is not a lenient measurement, it is not a
	// measurement. Eight runs of a fifty-run contract compared against 95%
	// produces a verdict with no evidence under it.
	if r.Planned > 0 && r.Runs < r.Planned {
		return false
	}
	return r.Runs > 0 && r.Errors == 0
}

// Met reports whether the contract held.
//
// An unsound measurement never counts as met. The alternative is a green
// result whose evidence is missing, which is the one outcome worse than red.
func (r Result) Met() bool {
	if !r.Sound() {
		return false
	}
	return r.Rate() >= r.Threshold
}

// String renders the result as one line, always carrying the run count.
//
// The count is not decoration. "97%" is not a finding; "97% over 20 runs of
// MiniMax-M3" is, and printing them apart is how the first gets quoted.
func (r Result) String() string {
	var b strings.Builder
	verdict := "MET"
	if !r.Met() {
		verdict = "NOT MET"
	}
	fmt.Fprintf(&b, "%-32s %s  %.1f%% of %d runs (threshold %.1f%%)",
		r.ID, verdict, r.Rate()*100, r.Runs, r.Threshold*100)
	if r.Planned > 0 && r.Runs < r.Planned {
		fmt.Fprintf(&b, " — stopped after %d of %d planned, measurement unsound", r.Runs, r.Planned)
	}
	if r.Errors > 0 {
		fmt.Fprintf(&b, " — %d run(s) errored, measurement unsound", r.Errors)
		if r.FirstError != "" {
			fmt.Fprintf(&b, ": %s", r.FirstError)
		}
	}
	if r.Model != "" {
		fmt.Fprintf(&b, " · model %s", r.Model)
	}
	if r.Build != "" {
		fmt.Fprintf(&b, " · build %s", r.Build)
	}
	return b.String()
}

// Attempt is one run of a scenario against a real model.
//
// The two return values are deliberately different questions. The bool is the
// verdict — did the model behave as the contract says. The error means the run
// could not be measured at all, which is a transport problem and not a verdict.
// Collapsing them would let a flaky network read as a behavioural regression,
// and that is the one misreading this whole package exists to prevent.
type Attempt func(ctx context.Context) (bool, error)

// Measure runs a scenario cfg.Runs times and reports the rate.
//
// It never stops early. A contract at 90% is *expected* to fail one run in ten,
// so exiting on the first failure would make every threshold below 100%
// unmeasurable — and the thresholds below 100% are the reason this exists.
//
// Measure takes an Attempt rather than reaching for a model itself, which is
// what keeps it outside the build tag and inside the test suite.
func Measure(ctx context.Context, cfg Config, id string, threshold float64, attempt Attempt) Result {
	r := Result{
		ID:        id,
		Threshold: threshold,
		Planned:   cfg.Runs,
		Model:     cfg.Model,
		Build:     version.Short(),
	}
	for i := 0; i < cfg.Runs; i++ {
		// A dead context does not produce twenty more failures, it produces
		// nothing. Calling attempt anyway is how v11 came back with 34 of 35
		// contracts unsound and no reason on any of them: every run after the
		// first hang errored instantly, and the harness reported its own
		// invented errors as if the model had produced them.
		if ctx.Err() != nil {
			break
		}
		r.Runs++
		ok, err := attempt(ctx)
		switch {
		case err != nil:
			r.Errors++
			if r.FirstError == "" {
				r.FirstError = trimError(err.Error())
			}
		case ok:
			r.Passed++
		}
	}
	return r
}

// Summary states how much of the declared set was actually measured, and it is
// the last line a measurement run prints.
//
// It exists because `go test` ends in PASS whether the suite measured every
// contract or none of them, and a skipped measurement under a PASS reads as a
// measurement that succeeded. That is the exact misreading this package was
// built to prevent, and for a while it lived in the package built to prevent
// it: `make eval` printed PASS having run nothing against a model.
//
// The count goes last rather than the exit status, because the exit status is
// the wrong sentence. Measuring nothing is not a failure — it is the default,
// on purpose, since a suite that costs money and fails by default is a suite
// somebody switches off. What must not happen is measuring nothing quietly.
func Summary(declared, asserted int, rs []Result) string {
	var b strings.Builder
	if len(rs) == 0 {
		fmt.Fprintf(&b, "evals: 0 of %d contracts measured — nothing here is evidence about behaviour", declared)
	} else {
		var met, unsound int
		for _, r := range rs {
			switch {
			case !r.Sound():
				unsound++
			case r.Met():
				met++
			}
		}
		fmt.Fprintf(&b, "evals: %d of %d contracts measured · %d met · %d not met",
			len(rs), declared, met, len(rs)-met-unsound)
		if unsound > 0 {
			fmt.Fprintf(&b, " · %d unsound", unsound)
		}
		if missing := declared - len(rs); missing > 0 {
			fmt.Fprintf(&b, " · %d never ran", missing)
		}
	}
	if asserted > 0 {
		fmt.Fprintf(&b, " · %d asserted deterministically, not measured", asserted)
	}
	return b.String()
}

// errorText is how much of a failure is worth carrying into the summary line.
const errorText = 160

// trimError keeps a failure readable on one line.
func trimError(msg string) string {
	msg = strings.Join(strings.Fields(msg), " ")
	if len(msg) > errorText {
		msg = msg[:errorText] + "…"
	}
	return msg
}

// Report renders a set of results, worst first.
//
// Worst first because the reason to read this output at all is to find what
// regressed, and a met contract at the top of a long list is noise.
func Report(rs []Result) string {
	sorted := make([]Result, len(rs))
	copy(sorted, rs)
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if a.Met() != b.Met() {
			return !a.Met()
		}
		return a.Rate()-a.Threshold < b.Rate()-b.Threshold
	})
	lines := make([]string, 0, len(sorted))
	for _, r := range sorted {
		lines = append(lines, r.String())
	}
	return strings.Join(lines, "\n")
}

// MaxEvidence is how many distinct failing transcripts a contract keeps.
//
// Three, and the number is a compromise between two ways of learning nothing.
// One digest — what this kept before — is a sample of one, and the one it is
// happens to be whichever failed first, which is not the same as
// representative. Twenty digests of four hundred characters is a wall nobody
// reads, and a wall nobody reads is where a second cause hides.
const MaxEvidence = 3

// Evidence collects what a contract's failing runs actually did.
//
// It exists because every diagnosis in this suite has been made by reading a
// digest, never by reading a rate. A rate says a contract is at 70%; only the
// transcript says whether the model refused sensibly, ran out of rounds, or was
// asked a question the fixture never gave it the tools to answer.
type Evidence struct {
	cap   int
	total int
	seen  map[string]struct{}
	kept  []string
}

// NewEvidence returns a collector holding at most cap distinct digests.
func NewEvidence(cap int) *Evidence {
	return &Evidence{cap: cap, seen: map[string]struct{}{}}
}

// Record notes one failing run.
//
// Repeats are counted and not kept. Twenty runs failing the same way is one
// finding, and spending the cap on copies of it is how the second cause stays
// invisible.
func (e *Evidence) Record(digest string) {
	e.total++
	if _, ok := e.seen[digest]; ok {
		return
	}
	e.seen[digest] = struct{}{}
	if len(e.kept) < e.cap {
		e.kept = append(e.kept, digest)
	}
}

// String renders the evidence, declaring what it left out.
//
// The count of failures and the count of transcripts are printed separately on
// purpose: they differ whenever the cap bit or a failure repeated, and a reader
// who cannot see the difference will read three transcripts as three failures.
// Same rule as every truncated tool output — the cut is declared (RN-5).
func (e *Evidence) String() string {
	if e.total == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d run(s) failed, %d distinct transcript(s):", e.total, len(e.kept))
	for _, d := range e.kept {
		fmt.Fprintf(&b, "\n  %s", d)
	}
	return b.String()
}
