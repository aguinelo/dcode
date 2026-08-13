package tui

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/protocol"
)

func TestNothingIsShownWhenThereIsNothingToCheck(t *testing.T) {
	if _, ok := completionEntry(nil, En); ok {
		t.Error("a turn with no definition of done produced a line")
	}
	clean := &protocol.Completion{Verification: string(loop.VerificationClean)}
	if _, ok := completionEntry(clean, En); ok {
		t.Error("a turn that changed nothing produced a line; shown every turn it would stop being read")
	}
}

func TestAFailedCheckSaysSoInWordsNotOnlyInColour(t *testing.T) {
	e, ok := completionEntry(&protocol.Completion{
		Verification: string(loop.VerificationFailed),
		Met:          []string{"lint"},
		Unmet:        []string{"tests"},
	}, En)
	if !ok {
		t.Fatal("a failed check produced no line")
	}
	if !strings.Contains(e.Summary, "NOT verified") {
		t.Errorf("the summary does not read as failing without colour: %q", e.Summary)
	}
	if !strings.Contains(e.Summary, "tests") {
		t.Errorf("the summary does not name what failed: %q", e.Summary)
	}
	if !strings.Contains(e.Detail, "met") || !strings.Contains(e.Detail, "lint") {
		t.Errorf("the detail loses what did pass:\n%s", e.Detail)
	}
}

func TestUnavailableIsNotReportedAsFailure(t *testing.T) {
	e, _ := completionEntry(&protocol.Completion{
		Verification: string(loop.VerificationUnavailable),
		Unavailable:  []string{"tests"},
	}, En)
	if strings.Contains(e.Summary, "NOT verified") {
		t.Errorf("nothing to run was reported as a failing check: %q", e.Summary)
	}
	if !strings.Contains(e.Summary, "not verified") {
		t.Errorf("the summary does not say the work went unchecked: %q", e.Summary)
	}
}

// Changing what measures the work is sometimes right and always worth seeing.
func TestAChangeToTheMeasurementIsNeverFoldedAwaySilently(t *testing.T) {
	e, _ := completionEntry(&protocol.Completion{
		Verification:     string(loop.VerificationPassed),
		Met:              []string{"tests"},
		TouchedProtected: []string{"internal/loop/turn_test.go"},
	}, En)
	if !strings.Contains(e.Detail, "turn_test.go") {
		t.Fatalf("a change to the measurement is not on screen:\n%s", e.Detail)
	}
	if !strings.Contains(e.Detail, "measurement changed") {
		t.Errorf("the detail does not say what that line is:\n%s", e.Detail)
	}
}

// The seal is undroppable for the same reason the sandbox mode is: a guarantee
// that vanishes on a narrow terminal is not a guarantee.
func TestTheSealSurvivesANarrowTerminal(t *testing.T) {
	m := Model{
		Lang: En, Sandbox: "workspace-write", Model: "MiniMax-M3",
		Verification: string(loop.VerificationFailed),
		Window:       100000, InputTokens: 50000,
	}
	// Forty columns is the floor: below it the whole status line is clipped
	// rather than fielded, and no arrangement fits. Above it, the seal is never
	// the thing given up — the model name and the context meter are.
	for _, width := range []int{120, 80, 60, 40} {
		out := renderStatus(m, DefaultGeometry(width, 40), false, true)
		if !strings.Contains(out, "NOT VERIFIED") {
			t.Errorf("at width %d the seal was dropped:\n%s", width, out)
		}
	}
}

func TestNoSealWhenThereIsNothingToClaim(t *testing.T) {
	for _, v := range []string{"", string(loop.VerificationClean)} {
		if label, _ := VerificationLabel(v, En); label != "" {
			t.Errorf("verification %q produced the label %q; a permanent label stops being read", v, label)
		}
	}
}

func TestTheModelKeepsTheSealOfTheLastTurn(t *testing.T) {
	m := Model{Lang: En}
	m = m.Apply(completedEvent(t, &protocol.Completion{
		Verification: string(loop.VerificationFailed), Unmet: []string{"tests"},
	}))
	if m.Verification != string(loop.VerificationFailed) {
		t.Fatalf("Verification = %q after a failed turn", m.Verification)
	}
	var found bool
	for _, e := range m.Entries {
		if e.Kind == KindCompletion {
			found = true
		}
	}
	if !found {
		t.Error("no completion entry reached the stream")
	}
}

func completedEvent(t *testing.T, c *protocol.Completion) protocol.Event {
	t.Helper()
	payload, err := json.Marshal(protocol.TurnCompleted{TurnID: "t1", Completion: c})
	if err != nil {
		t.Fatal(err)
	}
	return protocol.Event{Type: protocol.EventTurnCompleted, Payload: payload}
}
