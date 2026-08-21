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

// -- a call's own progress ---------------------------------------------------

func toolRequested(seq uint64, id, name, input string) protocol.Event {
	raw, _ := json.Marshal(protocol.ToolRequested{
		TurnID: "t1", ToolCallID: id, Name: name, Input: json.RawMessage(input)})
	return protocol.Event{Seq: seq, Type: protocol.EventToolRequested, Payload: raw}
}

func toolCompleted(seq uint64, id, out string) protocol.Event {
	raw, _ := json.Marshal(protocol.ToolCompleted{ToolCallID: id, OK: true, Output: out, Files: 4})
	return protocol.Event{Seq: seq, Type: protocol.EventToolCompleted, Payload: raw}
}

// Progress lands on the call it names, and on no other.
func TestACallsProgressLandsOnThatCall(t *testing.T) {
	m := NewModel("s", "/w", "m", "workspace-write", En)
	m = m.Apply(toolRequested(1, "c1", "grep", `{"pattern":"x"}`))
	m = m.Apply(toolRequested(2, "c2", "glob", `{"pattern":"**/*.go"}`))
	m = m.Apply(progressEvent(3, protocol.Progress{
		TurnID: "t1", ToolCallID: "c1", Kind: protocol.ProgressFiles, Done: 25, Total: 184}))

	for _, e := range m.Entries {
		switch e.CallID {
		case "c1":
			if e.Done != 25 || e.Total != 184 {
				t.Errorf("c1 reports %d/%d", e.Done, e.Total)
			}
		case "c2":
			if e.Done != 0 {
				t.Errorf("c2 borrowed c1's count: %d", e.Done)
			}
		}
	}
}

// A result lands on the call it belongs to, which is a fix rather than a
// feature: the old reading took the LAST running entry, which is right exactly
// while one call runs at a time. With two in flight the first result landed on
// the second call's line — real numbers on the wrong row.
func TestAResultLandsOnItsOwnCallAndNotTheLastOneStarted(t *testing.T) {
	m := NewModel("s", "/w", "m", "workspace-write", En)
	m = m.Apply(toolRequested(1, "c1", "read", `{"path":"a.go"}`))
	m = m.Apply(toolRequested(2, "c2", "read", `{"path":"b.go"}`))
	m = m.Apply(toolCompleted(3, "c1", "the first one finished"))

	for _, e := range m.Entries {
		if e.CallID == "c1" && e.Running {
			t.Error("the call that finished is still running")
		}
		if e.CallID == "c2" && !e.Running {
			t.Error("a result landed on a call that had not reported back")
		}
	}
}

// The count on screen, and the ellipsis only when there is nothing to say.
func TestACallInFlightShowsWhatItHasGotThrough(t *testing.T) {
	gl := glyphs(true)
	for _, c := range []struct {
		done, total int
		want        string
	}{
		{25, 184, "25/184"},
		{25, 0, "25"},
		{0, 0, gl.ell},
	} {
		got := runningMeta(Entry{Done: c.done, Total: c.total, Running: true}, gl)
		if got != c.want {
			t.Errorf("%d of %d rendered %q, want %q", c.done, c.total, got, c.want)
		}
	}
}
