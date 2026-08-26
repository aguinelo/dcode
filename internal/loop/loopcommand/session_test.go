package loopcommand

import (
	"regexp"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/policy"
)

// The session name carries the prefix, the spec's basename and a UTC instant.
//
// It does NOT claim to be unique, and the spec no longer says it is: the
// timestamp has second granularity, so two /loop calls on the same spec
// within the same second produce the same name. The invariant that used to
// be written here — "distinct from any pre-existing session" — was false of
// this code, and the test that claimed it slept 1.1s to prove the clock
// ticks. What the name is for is letting a human tell two runs apart in the
// rail, and that is what this asserts.
func TestSessionConfigNameCarriesPrefixSpecAndInstant(t *testing.T) {
	_, name := SessionConfig(SessionOptions{
		Spec:          LoopSpec{Path: "/specs/2026-08-25-home-page"},
		SessionPrefix: "custom-",
	})

	want := regexp.MustCompile(`^custom-2026-08-25-home-page-\d{14}$`)
	if !want.MatchString(name) {
		t.Fatalf("session name %q does not read as <prefix><basename>-<YYYYMMDDHHMMSS>", name)
	}
}

// A path that has no basename to speak of still produces a readable name
// rather than a stray separator.
func TestSessionConfigNameSurvivesAPathWithoutABasename(t *testing.T) {
	for _, path := range []string{"", ".", "/"} {
		_, name := SessionConfig(SessionOptions{Spec: LoopSpec{Path: path}, SessionPrefix: "loop-"})
		if !regexp.MustCompile(`^loop-[^-]+-\d{14}$`).MatchString(name) {
			t.Fatalf("path %q produced %q", path, name)
		}
	}
}

// DoneEnabled is true iff there is at least one criterion. A spec with no
// criteria is still loadable — the engine just stops by old behaviour
// because there is nothing to re-enter on.
func TestSessionConfigDoneEnabledReflectsCriteria(t *testing.T) {
	withCrit := LoopSpec{Criteria: []loop.Criterion{{Name: "1", Command: "true"}}}
	withNone := LoopSpec{}

	cfgOn, _ := SessionConfig(SessionOptions{Spec: withCrit, SessionPrefix: "loop-"})
	cfgOff, _ := SessionConfig(SessionOptions{Spec: withNone, SessionPrefix: "loop-"})

	if !cfgOn.DoneEnabled {
		t.Fatal("DoneEnabled should be true when Criteria is non-empty")
	}
	if cfgOff.DoneEnabled {
		t.Fatal("DoneEnabled should be false when Criteria is empty")
	}
}

// Config.Done must reflect the LoopSpec exactly — no transformation,
// no reordering, no filtering.
func TestSessionConfigDoneMatchesSpec(t *testing.T) {
	spec := LoopSpec{
		Path: "/specs/foo",
		Criteria: []loop.Criterion{
			{Name: "1", Command: "make test"},
			{Name: "2", Command: "make lint"},
		},
		Protected: []string{"**/*_test.go"},
	}
	cfg, _ := SessionConfig(SessionOptions{Spec: spec, SessionPrefix: "loop-"})

	if len(cfg.Done.Criteria) != 2 {
		t.Fatalf("expected 2 criteria, got %d", len(cfg.Done.Criteria))
	}
	if cfg.Done.Criteria[0].Name != "1" || cfg.Done.Criteria[1].Name != "2" {
		t.Fatalf("order not preserved: %+v", cfg.Done.Criteria)
	}
	if len(cfg.Done.Protected) != 1 || cfg.Done.Protected[0] != "**/*_test.go" {
		t.Fatalf("protected not propagated: %+v", cfg.Done.Protected)
	}
}

// The limits reach the Config untouched. /loop is a façade and a façade has
// no budget of its own — a limit invented here would be a second ceiling for
// the same concept.
func TestSessionConfigCarriesTheLimitsUntouched(t *testing.T) {
	limits := loop.Limits{MaxIterations: 7, MaxIdenticalCalls: 2, MaxTurnTokens: 1234}
	cfg, _ := SessionConfig(SessionOptions{
		Spec:        LoopSpec{Criteria: []loop.Criterion{{Name: "1", Command: "true"}}},
		Limits:      limits,
		MaxStall:    5,
		DoneTimeout: 42 * time.Second,
		SandboxMode: policy.ModeWorkspaceWrite,
		Model:       "a-model",
	})

	if cfg.Limits != limits {
		t.Fatalf("limits changed on the way in: %+v", cfg.Limits)
	}
	if cfg.MaxStallCycles != 5 || cfg.DoneTimeout != 42*time.Second {
		t.Fatalf("stall or timeout changed: %d %v", cfg.MaxStallCycles, cfg.DoneTimeout)
	}
	if cfg.Mode != policy.ModeWorkspaceWrite || cfg.Model != "a-model" {
		t.Fatalf("mode or model changed: %v %q", cfg.Mode, cfg.Model)
	}
}
