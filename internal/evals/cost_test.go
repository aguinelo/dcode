package evals

import (
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/provider"
)

// An exchange that produced nothing a judge can read is not a verdict.
//
// The first measurement against a new provider read 0%, with the evidence
// `1 round(s): no tool calls` — which is exactly what a model refusing would
// produce, and refusing was what that contract measured. It looked like a
// finding. Sending the scenario's own request body to the provider by hand
// returned a tool call twice and nothing three times out of five.
//
// So the rule, and its two edges, which are the whole of it.
func TestAnExchangeWithNothingReadableIsNotAVerdict(t *testing.T) {
	for _, tc := range []struct {
		name  string
		calls int
		text  string
		want  bool
	}{
		{"nothing at all", 0, "", true},
		{"only whitespace", 0, "  \n\t ", true},
		// The edge that matters most. A model answering in prose without
		// calling is a real failure, and several contracts exist to catch it.
		// Counting this as unmeasured would hide the thing they are for.
		{"prose without a call", 0, "I will not run that.", false},
		{"a call", 1, "", false},
		{"a call and prose", 1, "here goes", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := unreadable(tc.calls, tc.text); got != tc.want {
				t.Errorf("unreadable(%d, %q) = %v, want %v", tc.calls, tc.text, got, tc.want)
			}
		})
	}
}

// A measurement reports what it consumed, and retries are in it.
//
// Retried runs were paid for twice. A cost that omitted the second payment
// would forecast a suite cheaper than it is, which is the direction that gets
// a budget approved and then overrun.
func TestCostAccumulatesEveryExchangeIncludingRepeats(t *testing.T) {
	var c Cost
	c.Add(&provider.Usage{InputTokens: 100, OutputTokens: 10, CacheReadTokens: 80})
	c.Add(&provider.Usage{InputTokens: 200, OutputTokens: 20, CacheReadTokens: 150})
	// A call that produced no usage still cost a call. It was made.
	c.Add(nil)

	if c.Exchanges != 3 {
		t.Errorf("exchanges = %d, want 3: a call with no usage still happened", c.Exchanges)
	}
	if c.InputTokens != 300 || c.OutputTokens != 30 {
		t.Errorf("tokens are in %d out %d, want in 300 out 30", c.InputTokens, c.OutputTokens)
	}
	// Cache reads are kept apart because they are the direct measure that the
	// append-only prefix is working. Folded into input, that would vanish.
	if c.CacheReadTokens != 230 {
		t.Errorf("cache reads = %d, want 230", c.CacheReadTokens)
	}
}

// Per-run is what a forecast is built from, and zero runs has no per-run.
//
// Dividing by the runs that happened rather than the runs planned: the question
// a forecast answers is "what did one of these actually cost", and a
// measurement cut short did not run the rest.
func TestPerRunDividesByWhatRanAndRefusesToInventANumber(t *testing.T) {
	c := Cost{
		Elapsed:      20 * time.Second,
		Exchanges:    10,
		InputTokens:  1000,
		OutputTokens: 100,
	}
	per := c.PerRun(5)
	if per.Elapsed != 4*time.Second || per.Exchanges != 2 || per.InputTokens != 200 {
		t.Errorf("per run = %+v, want 4s, 2 exchanges, 200 input", per)
	}

	// A measurement that never ran has no per-run cost, and reporting one
	// would be inventing exactly the number this type exists to stop people
	// inventing.
	if got := c.PerRun(0); got != (Cost{}) {
		t.Errorf("PerRun(0) = %+v, want the zero cost", got)
	}
}
