package loopcommand

import (
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/loop"
)

// Two consecutive NewSession calls on the same spec path produce distinct
// IDs — the timestamp is the only thing separating them. The invariant
// holds even when the basenames are identical.
func TestNewSessionIDsAreDistinct(t *testing.T) {
	opts := SessionOptions{
		Spec:          LoopSpec{Path: "/specs/foo"},
		SessionPrefix: "loop-",
	}
	_, id1 := NewSession(opts)
	time.Sleep(1100 * time.Millisecond)
	_, id2 := NewSession(opts)
	if id1 == id2 {
		t.Fatalf("two NewSession calls produced the same ID %q", id1)
	}
}

// DoneEnabled is true iff there is at least one criterion. A spec with no
// criteria is still loadable — the engine just stops by old behaviour
// because there is nothing to re-enter on.
func TestNewSessionDoneEnabledReflectsCriteria(t *testing.T) {
	withCrit := LoopSpec{Criteria: []loop.Criterion{{Name: "1", Command: "true"}}}
	withNone := LoopSpec{}

	cfgOn, _ := NewSession(SessionOptions{Spec: withCrit, SessionPrefix: "loop-"})
	cfgOff, _ := NewSession(SessionOptions{Spec: withNone, SessionPrefix: "loop-"})

	if !cfgOn.DoneEnabled {
		t.Fatal("DoneEnabled should be true when Criteria is non-empty")
	}
	if cfgOff.DoneEnabled {
		t.Fatal("DoneEnabled should be false when Criteria is empty")
	}
}

// Config.Done must reflect the LoopSpec exactly — no transformation,
// no reordering, no filtering.
func TestNewSessionDoneMatchesSpec(t *testing.T) {
	spec := LoopSpec{
		Path: "/specs/foo",
		Criteria: []loop.Criterion{
			{Name: "1", Command: "make test"},
			{Name: "2", Command: "make lint"},
		},
		Protected: []string{"**/*_test.go"},
	}
	cfg, _ := NewSession(SessionOptions{Spec: spec, SessionPrefix: "loop-"})

	if len(cfg.Done.Criteria) != 2 {
		t.Fatalf("expected 2 criteria, got %d", len(cfg.Done.Criteria))
	}
	if cfg.Done.Criteria[0].Name != "1" || cfg.Done.Criteria[1].Name != "2" {
		t.Fatalf("order not preserved: %+v", cfg.Done.Criteria)
	}
	if cfg.Done.Protected[0] != "**/*_test.go" {
		t.Fatalf("protected not propagated: %+v", cfg.Done.Protected)
	}
}

// The ID format carries the prefix the operator configured — the default
// "loop-" is what the user sees in the rail, and a custom prefix must
// reach the wire.
func TestNewSessionPrefixIsHonored(t *testing.T) {
	_, id := NewSession(SessionOptions{
		Spec:          LoopSpec{Path: "/specs/foo"},
		SessionPrefix: "custom-",
	})
	if !strings.HasPrefix(id, "custom-") {
		t.Fatalf("ID %q does not carry the configured prefix", id)
	}
}
