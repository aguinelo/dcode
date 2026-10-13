package evals

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/app"
)

// The fingerprint a measurement records is the prompt the run SENT.
//
// It used to hash Prompt(""): the markdown formulation, no repository block and
// no memory. The run sends PromptIn under the provider's family, which for
// claude is the tagged formulation and for a scenario with no repository carries
// the block saying so. Two of the four recorded fingerprints were of a prompt no
// model ever saw.
func TestTheRecordedFingerprintIsOfThePromptSent(t *testing.T) {
	f, err := LoadFixture(FixtureRoot, "floor-yields-to-project")
	if err != nil {
		t.Fatal(err)
	}
	if f.World.Repo != RepoAbsent {
		t.Fatal("the fixture no longer declares a missing repository; pick one that does")
	}
	dir := t.TempDir()
	if _, err := NewWorkspace(dir, f.Files, f.ToolNames()); err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, family := range []string{"claude", "minimax-m3"} {
		sent, err := f.PromptIn(context.Background(), family, dir)
		if err != nil {
			t.Fatal(err)
		}
		msgs, recorded, err := f.Messages(context.Background(), family, dir, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(msgs[0].Text, sent) {
			t.Fatalf("%s: the system message does not open with the prompt PromptIn builds", family)
		}
		if want := PromptFingerprint(sent); recorded != want {
			t.Errorf("%s: the run records %s and sends a prompt fingerprinted %s", family, recorded, want)
		}
		seen[family] = recorded
	}
	// The two formulations are different prompts, so they are different
	// fingerprints. One hash for both would be the family left out again.
	if seen["claude"] == seen["minimax-m3"] {
		t.Error("claude and minimax-m3 are sent different prompts and recorded the same fingerprint")
	}
}

// Stale is counted, and so is current, and neither swallows "not recorded".
//
// A measurement whose prompt has moved since describes a product that no longer
// exists. The count is what makes that visible without forcing every prompt
// change to pay for fifty re-measurements.
func TestAMeasurementWhosePromptMovedIsStale(t *testing.T) {
	ms := []Measurement{
		{ID: "unrecorded"},
		{ID: "same", Prompt: "aaaaaaaaaaaa"},
		{ID: "moved", Prompt: "bbbbbbbbbbbb"},
	}
	today := func(m Measurement) (string, error) {
		if m.ID == "unrecorded" {
			t.Errorf("asked for today's prompt of a measurement that recorded none; it would be invented")
		}
		return "aaaaaaaaaaaa", nil
	}
	current, stale, err := freshness(ms, today)
	if err != nil {
		t.Fatal(err)
	}
	if current != 1 || stale != 1 {
		t.Errorf("current=%d stale=%d, want 1 and 1", current, stale)
	}
}

// A prompt that cannot be rebuilt is a failure to say, never a quiet "current".
func TestAPromptThatCannotBeRebuiltIsAnError(t *testing.T) {
	_, _, err := freshness([]Measurement{{ID: "x", Prompt: "aaaaaaaaaaaa"}},
		func(Measurement) (string, error) { return "", errors.New("no such fixture") })
	if err == nil {
		t.Error("a measurement whose prompt could not be rebuilt was counted")
	}
}

// todayFingerprint is what the scenario a measurement ran would be sent now,
// under the family it ran through.
//
// Through Messages, the same call the eval makes, in a workspace built the way
// the eval builds one. A second construction of the prompt here would be the
// defect this file exists for.
func todayFingerprint(t *testing.T) func(Measurement) (string, error) {
	t.Helper()
	return func(m Measurement) (string, error) {
		c, ok := ContractByID(m.ID)
		if !ok {
			return "", fmt.Errorf("no contract %s", m.ID)
		}
		family, ok := m.family(app.Families())
		if !ok {
			return "", fmt.Errorf("no family claims model %q and the measurement names none, so no prompt can be rebuilt for it", m.Model)
		}
		f, err := LoadFixture(FixtureRoot, c.FixtureID())
		if err != nil {
			return "", err
		}
		dir := t.TempDir()
		if _, err := NewWorkspace(dir, f.Files, f.ToolNames()); err != nil {
			return "", err
		}
		_, fp, err := f.Messages(context.Background(), family, dir, nil)
		return fp, err
	}
}

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
