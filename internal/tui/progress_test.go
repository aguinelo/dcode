package tui

import (
	"encoding/json"
	"fmt"
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
	if out := RenderStatusBar(m, g); strings.Contains(out, "round") {
		t.Errorf("a round count was drawn with nothing reported:\n%s", out)
	}
}

// The ceiling reaches the bar once it is close, and not before.
//
// It used to be a section of the plan panel, which only opened when something
// else was already there — so most turns, which have no plan, had nowhere for
// it to appear. The panel is gone with the plan; the bar is where it lives now,
// on the same terms.
func TestTheCeilingReachesTheBarOnceItIsClose(t *testing.T) {
	m := NewModel("s", "/w", "m", "workspace-write", En)
	if m.turnSectionWorthDrawing() {
		t.Error("the ceiling was drawn with nothing in it")
	}
	// This used to open it, and that was the defect: on a real session the
	// panel spent thirty-three columns saying `iteração 0/2000`. Zero of two
	// thousand warns of nothing.
	m = m.Apply(progressEvent(1, protocol.Progress{
		TurnID: "t1", Kind: protocol.ProgressRounds, Done: 1, Total: 100}))
	if m.turnSectionWorthDrawing() {
		t.Error("one round of a hundred reached the bar")
	}
	m = m.Apply(progressEvent(2, protocol.Progress{
		TurnID: "t1", Kind: protocol.ProgressRounds, Done: 50, Total: 100}))
	if !m.turnSectionWorthDrawing() {
		t.Error("half the ceiling did not reach the bar")
	}
}

// And a limit being felt right now opens it whatever the round count says:
// every slot in flight is not a ceiling approaching, it is one reached.
func TestEverySlotInFlightReachesTheBar(t *testing.T) {
	m := NewModel("s", "/w", "m", "workspace-write", En)
	m.InFlight, m.MaxInFlight = 3, 4
	if m.turnSectionWorthDrawing() {
		t.Error("three of four slots reached the bar")
	}
	m.InFlight = 4
	if !m.turnSectionWorthDrawing() {
		t.Error("every slot in flight did not reach the bar")
	}
}

// Dim until it is close, and then not. The round ceiling is the roadmap's one
// item with measured evidence of harm, and what it lacked was anything saying
// it was coming.
func TestTheRoundCountWarnsAsItNearsTheCeiling(t *testing.T) {
	g := DefaultGeometry(140, 24)
	g.Palette = Palette{Enabled: true}

	// Both halfway or past it, so both are drawn at all: what is compared is
	// how they are drawn, and a count that is not drawn compares equal for the
	// wrong reason.
	early := NewModel("s", "/w", "m", "workspace-write", En)
	early.Rounds, early.MaxRounds = 50, 100
	late := early
	late.Rounds = 80

	a, okA := ceilingSegment(early, g)
	b, okB := ceilingSegment(late, g)
	if !okA || !okB {
		t.Fatalf("the ceiling is not drawn at 50 (%v) or at 80 (%v)", okA, okB)
	}
	if a.text == b.text {
		t.Error("the count looks the same at 50 of 100 as at 80 of 100")
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
		m.Rounds, m.MaxRounds = 60, 100
		g := DefaultGeometry(140, 24)
		g.Palette = Palette{}
		seg, _ := ceilingSegment(m, g)
		if out := seg.text; !strings.Contains(out, c.want) {
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

// -- a call that is still arriving -------------------------------------------

// The line appears the moment the call opens, before a single argument has
// landed. Waiting for tool.requested meant nothing on screen through the part
// of the turn where the work is happening — for a large write, most of it.
func TestACallAppearsWhileItIsStillArriving(t *testing.T) {
	m := NewModel("s", "/w", "m", "workspace-write", En)
	m = m.Apply(progressEvent(1, protocol.Progress{
		TurnID: "t1", ToolCallID: "c1", Name: "write",
		Kind: protocol.ProgressArguments, Done: 0}))

	if len(m.Entries) != 1 {
		t.Fatalf("no line was drawn: %+v", m.Entries)
	}
	e := m.Entries[0]
	if e.Tool != "write" || !e.Running || !e.Arriving {
		t.Errorf("the line does not say what is happening: %+v", e)
	}
}

// It is counted in bytes of itself, and says bytes. A bare number beside a tool
// that has done nothing would read as work already done.
func TestAnArrivingCallIsCountedInBytesAndSaysSo(t *testing.T) {
	gl := glyphs(true)
	if got := runningMeta(Entry{Arriving: true, Done: 0}, gl); got != gl.ell {
		t.Errorf("nothing arrived yet and it said %q", got)
	}
	if got := runningMeta(Entry{Arriving: true, Done: 2048}, gl); got != "2.0k" {
		t.Errorf("got %q", got)
	}
	if got := runningMeta(Entry{Arriving: true, Done: 512}, gl); got != "512B" {
		t.Errorf("got %q", got)
	}
}

// The complete call fills the line that was already there. Drawing a second one
// would leave the same call on screen twice, once half-arrived and once real.
func TestTheCompleteCallFillsTheLineRatherThanAddingOne(t *testing.T) {
	m := NewModel("s", "/w", "m", "workspace-write", En)
	m = m.Apply(progressEvent(1, protocol.Progress{
		TurnID: "t1", ToolCallID: "c1", Name: "write",
		Kind: protocol.ProgressArguments, Done: 900}))
	m = m.Apply(toolRequested(2, "c1", "write", `{"path":"a.go"}`))

	if n := len(m.Entries); n != 1 {
		t.Fatalf("the call is on screen %d times: %+v", n, m.Entries)
	}
	e := m.Entries[0]
	if e.Arriving {
		t.Error("a call that has been requested is still marked as arriving")
	}
	if e.Target != "a.go" {
		t.Errorf("the target did not fill in: %q", e.Target)
	}
	if e.Done != 0 {
		t.Errorf("the byte count survived into execution: %d", e.Done)
	}
}

// A provider that says nothing while a call arrives still works: the line
// appears at tool.requested exactly as it always did.
func TestAProviderThatSaysNothingStillDrawsTheCall(t *testing.T) {
	m := NewModel("s", "/w", "m", "workspace-write", En)
	m = m.Apply(toolRequested(1, "c1", "read", `{"path":"a.go"}`))
	if len(m.Entries) != 1 || m.Entries[0].Target != "a.go" {
		t.Errorf("the old path stopped working: %+v", m.Entries)
	}
}

// The plan is a block in the stream, at the place it first appeared, always
// showing the current one.
//
// It used to be a column of its own. Putting it where the model made it is
// worth more than a permanent column: a plan is read when it changes and
// ignored the rest of the time, and a resident panel spends width on the rest
// of the time.
//
// One block, updated in place. Appending would stack every revision of the same
// plan down the screen, and the plan would read as four plans rather than as
// one being worked through.
func TestThePlanIsABlockInTheStream(t *testing.T) {
	m := NewModel("s", "/w", "m", "workspace-write", En)
	m = m.Apply(ev(t, 1, protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1", Text: "do it"}))

	plan := func(active int) protocol.PlanUpdated {
		var items []protocol.PlanItem
		for i := 1; i <= 3; i++ {
			st := protocol.PlanPending
			if i < active {
				st = protocol.PlanDone
			} else if i == active {
				st = protocol.PlanActive
			}
			items = append(items, protocol.PlanItem{ID: i, Text: fmt.Sprintf("step %d", i), Status: st})
		}
		return protocol.PlanUpdated{Items: items}
	}

	m = m.Apply(ev(t, 2, protocol.EventPlanUpdated, plan(1)))
	at := -1
	for i, e := range m.Entries {
		if e.Kind == KindPlan {
			at = i
		}
	}
	if at < 0 {
		t.Fatal("the plan did not reach the stream")
	}

	// Three more revisions, and a tool call between them so the position is
	// not merely the last thing appended.
	m = m.Apply(ev(t, 3, protocol.EventToolRequested, protocol.ToolRequested{
		ToolCallID: "c1", Name: "read", Input: json.RawMessage(`{"path":"a.go"}`)}))
	for i, active := range []int{2, 3} {
		m = m.Apply(ev(t, uint64(4+i), protocol.EventPlanUpdated, plan(active)))
	}

	blocks := 0
	for i, e := range m.Entries {
		if e.Kind != KindPlan {
			continue
		}
		blocks++
		if i != at {
			t.Errorf("the plan block moved from %d to %d", at, i)
		}
		if len(e.Plan) != 3 || e.Plan[2].Status != protocol.PlanActive {
			t.Errorf("the block does not carry the current plan: %+v", e.Plan)
		}
	}
	if blocks != 1 {
		t.Errorf("%d plan blocks for one plan", blocks)
	}

	// And the same log replayed produces the same single block in the same
	// place, which is what keeps the reducer's guarantee true here too.
	g := DefaultGeometry(100, 30)
	g.Palette = Palette{}
	first := Render(m, g)
	replay := NewModel("s", "/w", "m", "workspace-write", En)
	for _, e := range []protocol.Event{
		ev(t, 1, protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1", Text: "do it"}),
		ev(t, 2, protocol.EventPlanUpdated, plan(1)),
		ev(t, 3, protocol.EventToolRequested, protocol.ToolRequested{
			ToolCallID: "c1", Name: "read", Input: json.RawMessage(`{"path":"a.go"}`)}),
		ev(t, 4, protocol.EventPlanUpdated, plan(2)),
		ev(t, 5, protocol.EventPlanUpdated, plan(3)),
	} {
		replay = replay.Apply(e)
	}
	if got := Render(replay, g); got != first {
		t.Errorf("replaying the same log drew a different screen")
	}
}

// The context meter measures the context, and a share of a window cannot
// exceed the window.
//
// It used to be `100 * InputTokens / Window`, and InputTokens is CUMULATIVE
// across a turn's rounds: every round re-sends the context, so a forty-round
// turn sums forty readings of it. The meter read `ctx 175%` — which is not a
// context that is 175% full, it is a turn that spent 1.75 windows of input.
//
// The fixture is that shape exactly: a turn whose cumulative input is nearly
// twice the window, while the context it actually holds is a quarter of it.
func TestTheContextMeterMeasuresTheContextAndNotTheTurnsCost(t *testing.T) {
	const window = 200_000
	m := NewModel("s", "/w", "m", "workspace-write", En)
	m = m.Apply(ev(t, 1, protocol.EventSessionCreated, protocol.Session{
		ID: "s", Workspace: "/w", State: protocol.SessionStateIdle, ContextWindow: window,
	}))
	m = m.Apply(ev(t, 2, protocol.EventTurnCompleted, protocol.TurnCompleted{
		TurnID: "t1", Reason: protocol.StopDone,
		Usage: &protocol.Usage{
			InputTokens:   350_000, // forty rounds of a 50k context, summed
			OutputTokens:  4_000,
			ContextTokens: 50_000, // what it actually holds
		},
	}))

	if m.ContextPct != 25 {
		t.Errorf("the meter says %d%%, want 25 — 50k of a 200k window", m.ContextPct)
	}
	if m.InputTokens != 350_000 {
		t.Errorf("the turn's cost was lost: %d", m.InputTokens)
	}

	// And it is capped, whatever arrives. A share of a window that exceeds the
	// window is a wrong number, and saying 100 is the smaller lie.
	m = m.Apply(ev(t, 3, protocol.EventTurnCompleted, protocol.TurnCompleted{
		TurnID: "t2", Reason: protocol.StopDone,
		Usage: &protocol.Usage{ContextTokens: window * 3},
	}))
	if m.ContextPct != 100 {
		t.Errorf("the meter says %d%%, want it capped at 100", m.ContextPct)
	}
}

// The person is warned that the context is filling BEFORE it is cut, and told
// how much went when it is.
//
// The model has been told this for a while — the bands exist and are announced
// to it. Nobody told the reader, so the summary arrived as one line saying it
// had happened: after the fact, with no warning it was coming and no chance to
// finish a thought first.
func TestTheContextSaysItIsFillingBeforeItIsCut(t *testing.T) {
	m := NewModel("s", "/w", "m", "workspace-write", En)
	m = m.Apply(ev(t, 1, protocol.EventContextBand, protocol.ContextBand{
		Band: 2, Fraction: 0.80,
	}))
	if len(m.Entries) != 1 {
		t.Fatalf("the crossing produced %d entries", len(m.Entries))
	}
	if got := m.Entries[0].Summary; !strings.Contains(got, "80%") ||
		!strings.Contains(got, "summary") {
		t.Errorf("the warning does not say how close it is: %q", got)
	}

	// And the cut says how much, not merely that it happened.
	m = m.Apply(ev(t, 2, protocol.EventSessionCompacted, protocol.SessionCompacted{
		FromSeq: 0, ToSeq: 40, Messages: 40, Kept: 12,
	}))
	got := m.Entries[len(m.Entries)-1].Summary
	if !strings.Contains(got, "40") || !strings.Contains(got, "12") {
		t.Errorf("the summary note says nothing about its size: %q", got)
	}

	// A crossing with nothing to report draws nothing: a note that says the
	// context is 0% of the way anywhere is a row spent on nothing.
	before := len(m.Entries)
	m = m.Apply(ev(t, 3, protocol.EventContextBand, protocol.ContextBand{}))
	if len(m.Entries) != before {
		t.Error("an empty crossing drew a note")
	}
}
