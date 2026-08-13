package evals

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// fakeReader stands in for resolved configuration. Three lines, which is the
// reason Reader is an interface.
type fakeReader map[string]string

func (f fakeReader) String(key, def string) string {
	if v, ok := f[key]; ok {
		return v
	}
	return def
}

func (f fakeReader) Bool(key string, def bool) bool {
	v, ok := f[key]
	if !ok {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func (f fakeReader) Int(key string, def int) int {
	v, ok := f[key]
	if !ok {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func TestEvalIsOffUnlessAskedFor(t *testing.T) {
	cfg := FromReader(fakeReader{})
	if cfg.Enabled {
		t.Fatal("eval enabled with nothing configured: measuring costs money, so the default must be off")
	}
	if cfg.Runs != DefaultRuns {
		t.Fatalf("runs = %d, want the default %d", cfg.Runs, DefaultRuns)
	}
}

func TestNonPositiveRunCountFallsBackToTheDefault(t *testing.T) {
	for _, v := range []string{"0", "-3"} {
		cfg := FromReader(fakeReader{"eval.runs": v})
		if cfg.Runs != DefaultRuns {
			t.Errorf("runs = %q gave %d, want the default %d", v, cfg.Runs, DefaultRuns)
		}
	}
}

func TestConfiguredValuesAreCarried(t *testing.T) {
	cfg := FromReader(fakeReader{
		"eval.enabled": "true",
		"eval.model":   "MiniMax-M3",
		"eval.runs":    "50",
	})
	if !cfg.Enabled || cfg.Model != "MiniMax-M3" || cfg.Runs != 50 {
		t.Fatalf("got %+v, want enabled MiniMax-M3 over 50 runs", cfg)
	}
}

func TestUnavailableNamesWhichKnobIsMissing(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		want string // substring the reason must carry
	}{
		{"off", Config{Enabled: false, Model: "MiniMax-M3", Runs: 20}, "eval.enabled"},
		{"no model", Config{Enabled: true, Model: "", Runs: 20}, "eval.model"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.cfg.Unavailable()
			if got == "" {
				t.Fatalf("no reason given for %+v; a skipped measurement must say which knob is missing", c.cfg)
			}
			if !strings.Contains(got, c.want) {
				t.Errorf("reason %q does not name %q", got, c.want)
			}
		})
	}
}

func TestUnavailableIsEmptyWhenItCanRun(t *testing.T) {
	cfg := Config{Enabled: true, Model: "MiniMax-M3", Runs: 20}
	if got := cfg.Unavailable(); got != "" {
		t.Fatalf("refused to run a fully configured measurement: %q", got)
	}
}

func TestRateCountsErroredRunsAgainstTheTotal(t *testing.T) {
	// Eighteen behaved, two could not be measured. The rate is over twenty,
	// not over eighteen: a run that failed to measure is not a run that passed.
	r := Result{Runs: 20, Passed: 18, Errors: 2}
	if got := r.Rate(); got != 0.9 {
		t.Fatalf("rate = %v, want 0.9", got)
	}
}

func TestAnErroredRunMakesTheMeasurementUnsound(t *testing.T) {
	// Every run that was measured behaved, and the rate clears the threshold.
	// It still must not read as met: the evidence has a hole in it.
	r := Result{ID: "toolcall-recover", Threshold: 0.90, Runs: 20, Passed: 19, Errors: 1}
	if r.Sound() {
		t.Error("Sound() true with an errored run")
	}
	if r.Met() {
		t.Error("Met() true with an errored run: a network blip must never read as a passing contract")
	}
}

func TestZeroRunsIsNotAPass(t *testing.T) {
	r := Result{ID: "no-phantom-tool", Threshold: 1.0}
	if r.Met() {
		t.Fatal("Met() true with no runs: a threshold nothing was measured against is not met")
	}
}

func TestThresholdIsMetAtTheBoundary(t *testing.T) {
	r := Result{Threshold: 0.95, Runs: 20, Passed: 19}
	if !r.Met() {
		t.Fatalf("19 of 20 is exactly 95%%, which meets a 95%% threshold; rate was %v", r.Rate())
	}
}

func TestAHundredPercentThresholdNeedsEveryRun(t *testing.T) {
	r := Result{ID: "no-phantom-tool", Threshold: 1.0, Runs: 20, Passed: 19}
	if r.Met() {
		t.Fatal("19 of 20 met a 100% threshold")
	}
}

func TestResultAlwaysCarriesTheRunCountAndTheModel(t *testing.T) {
	r := Result{
		ID: "toolcall-schema-valid", Threshold: 0.97,
		Runs: 20, Passed: 20, Model: "MiniMax-M3", Build: "0.0.0-dev+abc123",
	}
	s := r.String()
	for _, want := range []string{"toolcall-schema-valid", "MET", "20 runs", "MiniMax-M3", "0.0.0-dev+abc123"} {
		if !strings.Contains(s, want) {
			t.Errorf("result line is missing %q: a rate quoted without its run count and model cannot be compared\n%s", want, s)
		}
	}
}

func TestUnsoundResultSaysSoInItsLine(t *testing.T) {
	r := Result{ID: "toolcall-recover", Threshold: 0.9, Runs: 20, Passed: 18, Errors: 2}
	s := r.String()
	if !strings.Contains(s, "unsound") {
		t.Errorf("an unsound measurement must say so where it is read:\n%s", s)
	}
}

func TestMeasureRunsTheWholeCountEvenAfterAFailure(t *testing.T) {
	// A 90% contract is expected to fail one run in ten. Stopping at the first
	// failure would make every threshold below 100% unmeasurable.
	calls := 0
	r := Measure(context.Background(), Config{Runs: 20, Model: "m"}, "id", 0.9,
		func(context.Context) (bool, error) {
			calls++
			return calls != 3, nil // the third run misbehaves
		})
	if calls != 20 {
		t.Fatalf("attempt ran %d times, want 20: Measure stopped early", calls)
	}
	if r.Passed != 19 {
		t.Errorf("passed = %d, want 19", r.Passed)
	}
	if !r.Met() {
		t.Errorf("19 of 20 is 95%%, which meets a 90%% threshold; got %s", r)
	}
}

func TestMeasureSeparatesAVerdictFromAFailureToMeasure(t *testing.T) {
	// Every run errors. None of them is evidence that the model misbehaved,
	// so the result must be unsound rather than a rate of zero presented as
	// a contract regression.
	r := Measure(context.Background(), Config{Runs: 5, Model: "m"}, "id", 0.9,
		func(context.Context) (bool, error) {
			return false, errors.New("connection reset")
		})
	if r.Errors != 5 {
		t.Fatalf("errors = %d, want 5", r.Errors)
	}
	if r.Passed != 0 {
		t.Errorf("passed = %d, want 0", r.Passed)
	}
	if r.Sound() {
		t.Error("a run of nothing but transport failures reported itself as sound")
	}
}

func TestMeasureCarriesTheModelAndTheBuild(t *testing.T) {
	r := Measure(context.Background(), Config{Runs: 1, Model: "MiniMax-M3"}, "id", 1.0,
		func(context.Context) (bool, error) { return true, nil })
	if r.Model != "MiniMax-M3" {
		t.Errorf("model = %q, want MiniMax-M3", r.Model)
	}
	if r.Build == "" {
		t.Error("build is empty: a measurement that cannot say which binary produced it cannot be reproduced")
	}
}

func TestReportPutsTheWorstFirst(t *testing.T) {
	// `unsound` is the case that separates the two ordering rules. Its rate
	// clears its threshold by a wide margin, so ordering by margin alone would
	// bury it at the bottom — but it is not met, and a result that is not met
	// is the reason someone is reading this output at all.
	rs := []Result{
		{ID: "comfortable", Threshold: 0.80, Runs: 20, Passed: 20},        // met, +20 points
		{ID: "unsound", Threshold: 0.80, Runs: 20, Passed: 20, Errors: 1}, // NOT met, +20 points
		{ID: "failing", Threshold: 0.95, Runs: 20, Passed: 10},            // not met, -45 points
		{ID: "narrow", Threshold: 0.95, Runs: 20, Passed: 19},             // met, +0
	}
	lines := strings.Split(Report(rs), "\n")
	if len(lines) != 4 {
		t.Fatalf("got %d lines, want 4", len(lines))
	}
	order := []string{"failing", "unsound", "narrow", "comfortable"}
	for i, want := range order {
		if !strings.HasPrefix(strings.TrimSpace(lines[i]), want) {
			t.Errorf("line %d is %q, want it to start with %q", i, lines[i], want)
		}
	}
}

// The line that had to exist: `make eval` printed PASS having measured nothing,
// because a skipped test is a passing test. The exit status was never going to
// carry this, so the count does.
func TestSummarySaysPlainlyWhenNothingWasMeasured(t *testing.T) {
	got := Summary(35, 1, nil)
	if !strings.Contains(got, "0 of 35") {
		t.Errorf("the summary does not say how much of the set ran: %q", got)
	}
	if !strings.Contains(got, "nothing here is evidence") {
		t.Errorf("a run that measured nothing does not say so: %q", got)
	}
}

// A partial run is the dangerous shape: some numbers on screen make the whole
// list look covered. It has to name what never ran.
func TestSummaryCountsWhatNeverRan(t *testing.T) {
	rs := []Result{{ID: "a", Threshold: 0.9, Runs: 20, Passed: 20}}
	got := Summary(35, 0, rs)
	for _, want := range []string{"1 of 35", "1 met", "34 never ran"} {
		if !strings.Contains(got, want) {
			t.Errorf("the summary lost %q: %q", want, got)
		}
	}
	if strings.Contains(got, "asserted") {
		t.Errorf("nothing was asserted and the summary mentions it: %q", got)
	}
}

// An unsound result is neither met nor unmet: the measurement failed, not the
// model. Folding it into either count would report a transport blip as a
// behavioural verdict.
func TestSummaryKeepsUnsoundOutOfBothCounts(t *testing.T) {
	rs := []Result{
		{ID: "ok", Threshold: 0.9, Runs: 20, Passed: 20},
		{ID: "bad", Threshold: 0.9, Runs: 20, Passed: 4},
		{ID: "blip", Threshold: 0.9, Runs: 20, Passed: 20, Errors: 3},
	}
	got := Summary(3, 0, rs)
	for _, want := range []string{"3 of 3", "1 met", "1 not met", "1 unsound"} {
		if !strings.Contains(got, want) {
			t.Errorf("the summary lost %q: %q", want, got)
		}
	}
	if strings.Contains(got, "never ran") {
		t.Errorf("everything ran and the summary says otherwise: %q", got)
	}
}

// A contract settled by assertion is not a contract that was measured, and a
// complete run must not look one short because of it.
func TestSummarySeparatesWhatWasAssertedFromWhatWasMeasured(t *testing.T) {
	rs := []Result{{ID: "a", Threshold: 0.9, Runs: 20, Passed: 20}}
	got := Summary(1, 1, rs)
	if !strings.Contains(got, "1 of 1") {
		t.Errorf("a complete run does not read as complete: %q", got)
	}
	if !strings.Contains(got, "1 asserted deterministically") {
		t.Errorf("the asserted contract is invisible: %q", got)
	}
}

// A measurement that reports "20 run(s) errored" and nothing else costs an
// afternoon and a provider bill, and leaves nobody able to say whether it was
// a rate limit, a revoked key or a bug in the harness.
//
// One whole run came back 34 contracts unsound, and its output could not tell
// me which of those it had been.
func TestAnUnsoundMeasurementSaysWhatWentWrong(t *testing.T) {
	r := Measure(context.Background(), Config{Runs: 3, Model: "m"}, "id", 0.9,
		func(context.Context) (bool, error) {
			return false, errors.New("429 rate limited: too many requests for this key")
		})

	if r.Sound() {
		t.Fatal("a run of nothing but failures reported itself as sound")
	}
	line := r.String()
	if !strings.Contains(line, "rate limited") {
		t.Errorf("the summary does not say why the runs errored:\n%s", line)
	}
	if !strings.Contains(line, "3 run(s) errored") {
		t.Errorf("the summary lost the count:\n%s", line)
	}
}

// A sound measurement says nothing about errors, or every green line carries
// an empty field nobody reads.
func TestASoundMeasurementCarriesNoError(t *testing.T) {
	r := Measure(context.Background(), Config{Runs: 2, Model: "m"}, "id", 0.9,
		func(context.Context) (bool, error) { return true, nil })
	if r.FirstError != "" {
		t.Errorf("a clean measurement carries an error: %q", r.FirstError)
	}
	if strings.Contains(r.String(), "errored") {
		t.Errorf("a clean line mentions errors:\n%s", r.String())
	}
}

// The first failure is the one kept: later ones are usually the same cause
// repeating, and a line that grew with every retry would stop being one line.
func TestTheFirstFailureIsTheOneKept(t *testing.T) {
	n := 0
	r := Measure(context.Background(), Config{Runs: 3, Model: "m"}, "id", 0.9,
		func(context.Context) (bool, error) {
			n++
			return false, fmt.Errorf("failure number %d", n)
		})
	if !strings.Contains(r.FirstError, "number 1") {
		t.Errorf("kept %q, want the first failure", r.FirstError)
	}
}

// A provider error can be a page of JSON. The summary is one line.
func TestALongFailureIsTrimmed(t *testing.T) {
	r := Measure(context.Background(), Config{Runs: 1, Model: "m"}, "id", 0.9,
		func(context.Context) (bool, error) {
			return false, errors.New(strings.Repeat("x", errorText*4))
		})
	if len(r.FirstError) > errorText+8 {
		t.Errorf("the failure is %d chars and the summary is meant to be one line", len(r.FirstError))
	}
	if !strings.Contains(r.FirstError, "…") {
		t.Error("the failure was cut without saying so")
	}
}
