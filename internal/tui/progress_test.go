package tui

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

func progressEvent(seq uint64, p protocol.Progress) protocol.Event {
	raw, _ := json.Marshal(p)
	return protocol.Event{Seq: seq, Type: protocol.EventProgress, Payload: raw}
}

func turnStarted(seq uint64, id string) protocol.Event {
	raw, _ := json.Marshal(protocol.TurnStarted{TurnID: id, Text: "go"})
	return protocol.Event{Seq: seq, Type: protocol.EventTurnStarted, Payload: raw}
}

// The count and its ceiling arrive together, and the client keeps the pair. A
// share cannot answer the question a ceiling raises, which is how many are
// left.
func TestTheTurnsCountersComeFromTheDaemon(t *testing.T) {
	m := NewModel("s", "/w", "m", "workspace-write", En)
	m = m.Apply(progressEvent(1, protocol.Progress{
		TurnID: "t1", Kind: protocol.ProgressRounds, Done: 7, Total: 100}))
	m = m.Apply(progressEvent(2, protocol.Progress{
		TurnID: "t1", Kind: protocol.ProgressInFlight, Done: 3, Total: 4}))

	if m.Rounds != 7 || m.MaxRounds != 100 {
		t.Errorf("rounds are %d/%d", m.Rounds, m.MaxRounds)
	}
	if m.InFlight != 3 || m.MaxInFlight != 4 {
		t.Errorf("in flight is %d/%d", m.InFlight, m.MaxInFlight)
	}
}

// A tool's own progress belongs to that call. Nothing draws it yet, so it is
// ignored rather than half-applied: half of a number is worse than none of it,
// because it looks like the whole one.
func TestProgressForAToolDoesNotMoveTheTurnsCounters(t *testing.T) {
	m := NewModel("s", "/w", "m", "workspace-write", En)
	m = m.Apply(progressEvent(1, protocol.Progress{
		TurnID: "t1", Kind: protocol.ProgressRounds, Done: 4, Total: 100}))
	m = m.Apply(progressEvent(2, protocol.Progress{
		TurnID: "t1", ToolCallID: "c1", Kind: protocol.ProgressRounds, Done: 99, Total: 100}))

	if m.Rounds != 4 {
		t.Errorf("a tool's progress moved the turn's counter to %d", m.Rounds)
	}
}

// The counters belong to the turn that is starting. Carrying the last one's
// forward would show a round number for work that has not begun, and the first
// thing anybody would do is trust it.
func TestANewTurnStartsItsCountersAtZero(t *testing.T) {
	m := NewModel("s", "/w", "m", "workspace-write", En)
	m = m.Apply(progressEvent(1, protocol.Progress{
		TurnID: "t1", Kind: protocol.ProgressRounds, Done: 40, Total: 100}))
	m = m.Apply(turnStarted(2, "t2"))

	if m.Rounds != 0 {
		t.Errorf("the new turn started at round %d", m.Rounds)
	}
	// The ceiling survives, because it is configuration rather than progress
	// and forgetting it would close the panel between turns.
	if m.MaxRounds != 100 {
		t.Errorf("the ceiling was forgotten: %d", m.MaxRounds)
	}
}

// Nothing is drawn before the daemon has said anything. Zero of a hundred is a
// number, and a number nobody sent is a number that will be believed.
func TestTheTurnSectionSaysNothingBeforeTheDaemonDoes(t *testing.T) {
	m := NewModel("s", "/w", "m", "workspace-write", En)
	g := DefaultGeometry(140, 24)
	g.Palette = Palette{}
	if out := strings.Join(renderPanel(m, g), "\n"); strings.Contains(out, "round") {
		t.Errorf("a round count was drawn with nothing reported:\n%s", out)
	}
}

// Most turns have no plan, and the ceiling was hiding in a panel that only
// opened when something else was already there.
func TestTheTurnsNumbersAloneOpenThePanel(t *testing.T) {
	m := NewModel("s", "/w", "m", "workspace-write", En)
	if m.panelHasContent() {
		t.Error("the panel opened with nothing in it")
	}
	m = m.Apply(progressEvent(1, protocol.Progress{
		TurnID: "t1", Kind: protocol.ProgressRounds, Done: 1, Total: 100}))
	if !m.panelHasContent() {
		t.Error("the turn's numbers did not open the panel")
	}
}

// Dim until it is close, and then not. The round ceiling is the roadmap's one
// item with measured evidence of harm, and what it lacked was anything saying
// it was coming.
func TestTheRoundCountWarnsAsItNearsTheCeiling(t *testing.T) {
	g := DefaultGeometry(140, 24)
	g.Palette = Palette{Enabled: true}

	early := NewModel("s", "/w", "m", "workspace-write", En)
	early.Rounds, early.MaxRounds = 10, 100
	late := early
	late.Rounds = 80

	a := strings.Join(turnSection(early, g, 30), "")
	b := strings.Join(turnSection(late, g, 30), "")
	if a == b {
		t.Error("the count looks the same at 10 of 100 as at 80 of 100")
	}
}

// And it says both numbers in the interface language.
func TestTheTurnSectionSpeaksTheInterfaceLanguage(t *testing.T) {
	for _, c := range []struct {
		lang Lang
		want string
	}{
		{PtBR, "iteração"}, {En, "round"},
	} {
		m := NewModel("s", "/w", "m", "workspace-write", c.lang)
		m.Rounds, m.MaxRounds = 3, 100
		g := DefaultGeometry(140, 24)
		g.Palette = Palette{}
		if out := strings.Join(turnSection(m, g, 30), "\n"); !strings.Contains(out, c.want) {
			t.Errorf("%s: %q missing from:\n%s", c.lang, c.want, out)
		}
	}
}
