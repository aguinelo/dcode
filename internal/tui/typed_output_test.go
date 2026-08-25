package tui

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// bashRun feeds the two events one command produces, and returns the model.
func bashRun(t *testing.T, m Model, id, command, output string, typed, ok bool) Model {
	t.Helper()
	req, err := json.Marshal(protocol.ToolRequested{
		TurnID: "t1", ToolCallID: id, Name: "bash",
		Input: json.RawMessage(`{"command":` + quote(command) + `}`),
		Typed: typed,
	})
	if err != nil {
		t.Fatal(err)
	}
	m = m.Apply(protocol.Event{Type: protocol.EventToolRequested, Seq: 1, Payload: req})

	done, err := json.Marshal(protocol.ToolCompleted{
		ToolCallID: id, OK: ok, Output: output, ExitCode: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	return m.Apply(protocol.Event{Type: protocol.EventToolCompleted, Seq: 2, Payload: done})
}

func quote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// TestATypedCommandShowsWhatItPrinted is the defect, asked of the screen.
//
// `!ls -la` put `exit 0` on the row and nothing else. The output was never
// lost — it reached the client, it sat in the entry, and three keystrokes
// (esc, up, tab) revealed it. That is worse than losing it: the screen
// answered a request to SEE something with a status code, and looked correct
// while doing it.
//
// So this asks the rendered frame, not the model field. A test that asserted
// Expanded would have passed on a build where the renderer ignored it.
func TestATypedCommandShowsWhatItPrinted(t *testing.T) {
	m := Model{Workspace: "/w", Window: 1000}
	m = bashRun(t, m, "x1-1", "ls -la", "exit 0\ntotal 16\nalpha.txt\nbeta.txt\n", true, true)

	screen := Render(m, DefaultGeometry(100, 30))
	if !strings.Contains(screen, "alpha.txt") {
		t.Errorf("a typed command hid its own output:\n%s", screen)
	}
}

// TestAModelsCommandStaysCollapsed keeps the fix narrow.
//
// The collapse rule is right for the model: it runs `ls` to orient itself and
// then says what mattered, so opening every call would bury the prose that
// carries the point. Only the person's own command overrides it.
func TestAModelsCommandStaysCollapsed(t *testing.T) {
	m := Model{Workspace: "/w", Window: 1000}
	m = bashRun(t, m, "c1", "ls -la", "exit 0\ntotal 16\nalpha.txt\nbeta.txt\n", false, true)

	screen := Render(m, DefaultGeometry(100, 30))
	if strings.Contains(screen, "alpha.txt") {
		t.Errorf("a model's successful call opened by itself:\n%s", screen)
	}
}

// TestAFailedCommandStillOpens: the older rule survives untouched.
func TestAFailedCommandStillOpens(t *testing.T) {
	m := Model{Workspace: "/w", Window: 1000}
	m = bashRun(t, m, "c2", "ls /nope", "exit 1\nls: /nope: No such file\n", false, false)

	screen := Render(m, DefaultGeometry(100, 30))
	if !strings.Contains(screen, "No such file") {
		t.Errorf("a failure stayed collapsed:\n%s", screen)
	}
}

// TestTheExitCodeIsNotPrintedTwice covers the duplication the fix exposed.
//
// bash prefixes its output with `exit 0` because the model reads the output as
// text. The row renders that code in its own column, so an open entry said it
// twice. Invisible while the output stayed collapsed.
func TestTheExitCodeIsNotPrintedTwice(t *testing.T) {
	m := Model{Workspace: "/w", Window: 1000}
	m = bashRun(t, m, "x1-1", "ls -la", "exit 0\ntotal 16\nalpha.txt\n", true, true)

	if got := strings.Count(Render(m, DefaultGeometry(100, 30)), "exit 0"); got != 1 {
		t.Errorf(`"exit 0" appears %d times, want 1`, got)
	}
	if !strings.Contains(m.Entries[0].Detail, "total 16") {
		t.Errorf("trimming the echo took the output with it: %q", m.Entries[0].Detail)
	}
}

// TestALineThatOnlyStartsWithTheSummaryIsKept keeps the trim exact.
func TestALineThatOnlyStartsWithTheSummaryIsKept(t *testing.T) {
	if got := withoutEchoOf("exit 0 but weird\nrest\n", "exit 0"); !strings.HasPrefix(got, "exit 0 but weird") {
		t.Errorf("a line saying more than the summary was dropped: %q", got)
	}
	if got := withoutEchoOf("exit 0", "exit 0"); got != "exit 0" {
		t.Errorf("a single-line output with no newline was eaten: %q", got)
	}
}
