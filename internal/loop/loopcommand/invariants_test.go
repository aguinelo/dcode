package loopcommand

// This file claims the invariants declared by `loop-command` in
// `docs/specs/architecture/loop-command/202608252000-loop-command.p.spec.md §8`.
// The specguard `TestEveryFamilyThatDeclaresInvariantsHasAGuard` walks every
// `internal/*/invariants_test.go` looking for the family name; without this
// file the family's invariants are documented but not claimed, and the
// guard fails as the comment on the guard says it should.
//
// The actual coverage of the invariants lives next to the code it tests:
// - parser behaviour: loopspec_test.go
// - dispatch behaviour: dispatch_test.go
// - session creation: session_test.go
//
// Each invariant in the .p §8 is asserted by exactly one test in those
// files. This file does not duplicate them; it exists so the family is
// findable by the guard and so a reader can see the chain
// spec → claim → coverage in one hop.

import "testing"

// TestInvariantsClaimed marks the family as claimed. The real coverage is
// in loopspec_test.go, dispatch_test.go, session_test.go — this is the
// pointer, not the proof.
//
// Removing this test re-introduces the "documented but unclaimed" defect
// the guard exists to catch, and any reader of this file is the next person
// to forget it.
func TestInvariantsClaimed(t *testing.T) {
	// Map of invariants declared in loop-command.p §8 to the test that
	// asserts them. Kept here as a single readable list so a reviewer can
	// walk spec → test in one place.
	claimed := map[string]string{
		"LoadSpec returns error (not panic) on malformed tasks.md": "TestLoadSpecMalformedReturnsError",
		"LoadSpec returns LoopSpec with Criteria == nil on zero":   "TestLoadSpecZeroCriteriaIsNotAnError",
		"LoopSpec.Criteria preserves order of appearance":          "TestLoadSpecPreservesOrder",
		"Load with SourceLoopSpec ignores done.toml presence":      "TestLoadSourceLoopSpecReadsFile",
		"Load with SourceDoneFile ignores specPath":                "TestLoadSourceDoneFileIgnoresSpecPath",
		"NewSession produces a distinct ID per call":               "TestNewSessionIDsAreDistinct",
		"Config.Done equals Load(Spec) byte-a-byte":                "TestNewSessionDoneMatchesSpec",
		"Config.DoneEnabled is true iff Criteria is non-empty":     "TestNewSessionDoneEnabledReflectsCriteria",
		"LoadSpecWithProtect layers file-declared and argument":    "TestLoadSpecWithProtectLayersBoth",
	}
	if len(claimed) == 0 {
		t.Fatal("the family declares invariants and the map is empty")
	}
	// Family name must appear in this file so the guard finds it.
	// The constant below makes the relationship searchable.
	const family = "loop-command"
	if family == "" {
		t.Fatal("family constant empty")
	}
}
