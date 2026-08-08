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
// from a partial log. protectedFrom must fall back to 0 rather than index -1.
func TestProtectedFromWithNoUserMessages(t *testing.T) {
	h := []Message{msg(RoleAssistant, "a"), msg(RoleAssistant, "b")}
	if got := protectedFrom(h, 0); got != 0 {
		t.Errorf("no user message must protect nothing, got %d", got)
	}
	if got := protectedFrom(nil, 4); got != 0 {
		t.Errorf("empty history, got %d", got)
	}
}

func TestProtectedFromClampsNegativeKeepTurns(t *testing.T) {
	h := []Message{msg(RoleUser, "a"), msg(RoleAssistant, "b"), msg(RoleUser, "c")}
	if got := protectedFrom(h, -3); got != protectedFrom(h, 0) {
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
