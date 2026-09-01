package evals

import (
	"strings"
	"testing"
)

// A measurement belongs to a model AND to a prompt, and only the first half was
// ever written down.
//
// On 1 September the skills block began rendering in every session, including
// those with no skill installed. Every prompt in this suite changed, and all
// nineteen measurements silently became descriptions of a product that no
// longer existed. Nothing could tell — the same defect as a count copied from a
// truth that moved, one level up: not a stale number ABOUT the measurements,
// but stale measurements.
func TestEveryMeasurementSaysWhichPromptItSawOrAdmitsItCannot(t *testing.T) {
	if len(Measured) == 0 {
		t.Fatal("no measurements; this guard would pass reading nothing")
	}
	for _, m := range Measured {
		if m.Prompt == "" {
			continue // not recorded, which Unverifiable counts and reports
		}
		if len(m.Prompt) != 12 {
			t.Errorf("%s: fingerprint %q is not the twelve characters PromptFingerprint produces",
				m.ID, m.Prompt)
		}
		if strings.ToLower(m.Prompt) != m.Prompt {
			t.Errorf("%s: fingerprint %q is not lower case, so it will never match", m.ID, m.Prompt)
		}
	}
}

// The fingerprint changes when the prompt does, and not otherwise. A hash that
// moved on every run would mark everything stale and be switched off within a
// day; one that never moved would mark nothing and be worth nothing.
func TestTheFingerprintMovesWithThePromptAndNotOtherwise(t *testing.T) {
	a := PromptFingerprint("You are dcode.\n\n## Skills\n\nNone are installed.")
	again := PromptFingerprint("You are dcode.\n\n## Skills\n\nNone are installed.")
	b := PromptFingerprint("You are dcode.")

	if a != again {
		t.Error("the same prompt fingerprinted twice gave two answers")
	}
	if a == b {
		t.Error("adding a whole section to the prompt did not move the fingerprint")
	}
	if len(a) != 12 {
		t.Errorf("fingerprint is %d characters", len(a))
	}
}

// Empty means NOT RECORDED, and is counted as such rather than read as current.
// A measurement that cannot say what it saw is not a measurement that saw this.
func TestAnUnrecordedFingerprintIsNotReadAsCurrent(t *testing.T) {
	ms := []Measurement{
		{ID: "a", Prompt: ""},
		{ID: "b", Prompt: "abc123abc123"},
		{ID: "c", Prompt: "def456def456"},
	}
	if got := unverifiable(ms); got != 1 {
		t.Errorf("unverifiable = %d, want the one that recorded nothing", got)
	}
}
