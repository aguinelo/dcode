package contextengine

import (
	"encoding/json"
	"strings"
	"testing"
)

// A zero Config must not divide by zero or compact on a nonsense threshold.
// Callers construct Config from user configuration, and a missing key there is
// routine.
func TestEstimateFillsMissingConfig(t *testing.T) {
	msgs := []Message{msg(RoleUser, strings.Repeat("x", 350))}
	got := Estimate(msgs, Config{})
	want := Estimate(msgs, DefaultConfig())
	if got != want {
		t.Errorf("a zero Config must fall back to defaults: got %d want %d", got, want)
	}
	if got <= 0 {
		t.Errorf("estimate should be positive, got %d", got)
	}
}

func TestWithDefaultsFillsEachFieldIndependently(t *testing.T) {
	d := DefaultConfig()
	for _, tc := range []struct {
		name string
		in   Config
		want Config
	}{
		{"all zero", Config{}, Config{CompactAt: d.CompactAt, CharsPerToken: d.CharsPerToken, Margin: d.Margin}},
		{"keeps set values", Config{CompactAt: 0.5, CharsPerToken: 4, Margin: 0.2},
			Config{CompactAt: 0.5, CharsPerToken: 4, Margin: 0.2}},
		{"negative margin resets", Config{CompactAt: 0.5, CharsPerToken: 4, Margin: -1},
			Config{CompactAt: 0.5, CharsPerToken: 4, Margin: d.Margin}},
		// Zero is unset, not a choice: a margin of zero removes the protection
		// the margin exists to provide.
		{"zero margin resets", Config{CompactAt: 0.5, CharsPerToken: 4, Margin: 0},
			Config{CompactAt: 0.5, CharsPerToken: 4, Margin: d.Margin}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := withDefaults(tc.in)
			if got.CompactAt != tc.want.CompactAt || got.CharsPerToken != tc.want.CharsPerToken ||
				got.Margin != tc.want.Margin {
				t.Errorf("got %+v want %+v", got, tc.want)
			}
		})
	}
}

// A session whose instructions are unusable cannot be measured, so it cannot be
// compacted either. Returning "no plan" is right: the failure surfaces on the
// next Assemble, where the caller can act on it.
func TestPlanWithUnassemblableSessionDoesNothing(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Window = 10
	s := longSession(30)
	s.Instructions = ""
	if _, ok := Plan(s, cfg); ok {
		t.Error("a session that cannot be assembled must not produce a plan")
	}
}

// When everything left is already protected there is nothing left to cut. The
// caller must get "no plan" rather than an empty span it would apply anyway,
// which would install a summary covering nothing and lose the earlier one.
func TestPlanReturnsNothingWhenEverythingIsProtected(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Window = 1
	cfg.KeepTurns = 100

	s := longSession(3)
	if plan, ok := Plan(s, cfg); ok {
		t.Errorf("nothing is compactable here, got plan %+v", plan)
	}
}

// Compacting twice must make progress. If the second Plan returned the same
// span, the loop would compact forever without ever freeing room.
func TestPlanAdvancesAfterApply(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Window = 200
	cfg.KeepTurns = 0

	s := longSession(40)
	first, ok := Plan(s, cfg)
	if !ok {
		t.Skip("no plan produced")
	}
	s = Apply(s, first, "summary one")

	if second, ok := Plan(s, cfg); ok {
		if second.FromIdx != first.ToIdx {
			t.Errorf("the second cut must start where the first ended: %d vs %d",
				second.FromIdx, first.ToIdx)
		}
		if second.ToIdx <= first.ToIdx {
			t.Errorf("the second cut must advance: %d <= %d", second.ToIdx, first.ToIdx)
		}
	}
}

// History with no user message at all: reachable when a session is restored
// from a partial log. The turn floor must fall back to 0 rather than index -1.
func TestProtectedFromWithNoUserMessages(t *testing.T) {
	h := []Message{msg(RoleAssistant, "a"), msg(RoleAssistant, "b")}
	if got := protectedByTurns(h, 0); got != 0 {
		t.Errorf("no user message must protect nothing, got %d", got)
	}
	if got := protectedByTurns(nil, 4); got != 0 {
		t.Errorf("empty history, got %d", got)
	}
}

func TestProtectedFromClampsNegativeKeepTurns(t *testing.T) {
	h := []Message{msg(RoleUser, "a"), msg(RoleAssistant, "b"), msg(RoleUser, "c")}
	if got := protectedByTurns(h, -3); got != protectedByTurns(h, 0) {
		t.Errorf("a negative KeepTurns must behave as zero, got %d", got)
	}
}

// Index past the end is reachable when KeepTurns exceeds the history length.
func TestTurnBoundaryClampsBeyondEnd(t *testing.T) {
	h := []Message{msg(RoleUser, "a"), msg(RoleAssistant, "b")}
	if got := turnBoundaryAtOrBefore(h, 99); got > len(h) {
		t.Errorf("boundary must clamp to len(history), got %d", got)
	}
}

// A turn whose tool results never arrive — the process died mid-execution.
// There is no clean boundary after the orphaned call, so the cut must fall
// before it rather than produce history the provider will reject.
func TestTurnBoundaryWithOrphanedToolCall(t *testing.T) {
	h := []Message{
		msg(RoleUser, "go"),
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "x", Name: "bash", Input: json.RawMessage(`{}`)}}},
		msg(RoleUser, "still there?"),
	}
	got := turnBoundaryAtOrBefore(h, len(h))
	if got > 1 {
		t.Errorf("cut must not fall after an unresolved tool call, got %d", got)
	}
	if !isCleanBoundary(h, got) {
		t.Errorf("boundary %d is not clean", got)
	}
}

// The tail survives in TOKENS, not only in turns.
//
// KeepTurns alone is a count, and a count is the wrong unit for this: turns vary
// by an order of magnitude — a one-line question and a forty-tool investigation
// are both one turn. Four short ones protect almost nothing.
//
// The fixture is that shape exactly: one enormous exchange followed by three
// tiny ones. By the count, the big one is compacted; by the fraction, it is what
// the tail is made of.
func TestTheTailSurvivesInTokensAndNotOnlyInTurns(t *testing.T) {
	big := strings.Repeat("x", 40_000)
	h := []Message{
		msg(RoleUser, "primeiro"), msg(RoleAssistant, "resposta antiga"),
		msg(RoleUser, "a investigação"), msg(RoleAssistant, big),
		msg(RoleUser, "e?"), msg(RoleAssistant, "sim"),
		msg(RoleUser, "certo"), msg(RoleAssistant, "ok"),
		msg(RoleUser, "agora"),
	}
	cfg := Config{Window: 40_000, KeepTurns: 2, KeepFraction: 0.30}

	byTurns := protectedByTurns(h, cfg.KeepTurns)
	both := protectedFrom(h, withDefaults(cfg), h)
	if both >= byTurns {
		t.Fatalf("the fraction protected nothing: turns %d, both %d", byTurns, both)
	}
	// And what it protected is the big exchange — the tail is made of it.
	if both > 3 {
		t.Errorf("the tail starts at %d; the big exchange is at 3 and must be in it", both)
	}

	// The other direction: many even turns, where the COUNT protects more.
	//
	// Sized so both floors are reachable — a window the history actually
	// fills. A window nobody could fill is not a case: Plan only asks this
	// question once the context is already at 80% of the window.
	even := make([]Message, 0, 40)
	for i := 0; i < 20; i++ {
		even = append(even, msg(RoleUser, strings.Repeat("q", 100)),
			msg(RoleAssistant, strings.Repeat("a", 100)))
	}
	c2 := withDefaults(Config{Window: Estimate(even, Config{}), KeepTurns: 10, KeepFraction: 0.30})
	byCount := protectedByTurns(even, 10)
	byFrac := protectedByTokens(even, c2, even)
	if byCount >= byFrac {
		t.Fatalf("the fixture does not exercise the count: count %d, fraction %d", byCount, byFrac)
	}
	if got := protectedFrom(even, c2, even); got != byCount {
		t.Errorf("whichever protects more must win: got %d, want the count at %d", got, byCount)
	}
}

// A window nobody reported leaves the fraction out of it rather than protecting
// everything: with no denominator there is no fraction, and a rule that
// protects the whole history would stop compaction happening at all.
func TestTheFractionNeedsAWindow(t *testing.T) {
	h := []Message{msg(RoleUser, "a"), msg(RoleAssistant, "b"), msg(RoleUser, "c")}
	cfg := withDefaults(Config{KeepTurns: 1})
	if got, want := protectedFrom(h, cfg, h), protectedByTurns(h, 1); got != want {
		t.Errorf("without a window the count must decide: got %d, want %d", got, want)
	}
}
