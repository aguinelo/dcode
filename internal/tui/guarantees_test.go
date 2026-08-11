package tui

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// Rendering is a function of the model and the geometry, and nothing else.
//
// The reducer already has a purity test; this is the other half. A renderer
// that reads a clock, a terminal size it was not given, or a package-level
// counter produces a screen that differs between two draws of the same state —
// and the redraw path repaints constantly, so the symptom is a flicker nobody
// can reproduce rather than an error anyone can chase.
func TestRenderIsPureOverTheModelAndTheGeometry(t *testing.T) {
	m := modelWithPlan()
	m.Entries = append(m.Entries, Entry{
		Kind: KindTool, Tool: "edit", Target: "stats.go",
		Summary: "edited stats.go", Detail: "--- a\n+++ b\n@@\n-x\n+y\n", Expanded: true,
	})
	g := DefaultGeometry(100, 30)

	first := Render(m, g)
	for i := 0; i < 50; i++ {
		if got := Render(m, g); got != first {
			t.Fatalf("draw %d differs from the first; the renderer reads something "+
				"outside its arguments", i)
		}
	}
}

// RN-11: the session lives on the server, and closing the client loses nothing.
//
// Asserted the way a user would find out: attach, watch a session happen,
// close, reattach from seq 1 and replay the same journal. The two screens must
// be identical. Anything the client kept that the events do not carry shows up
// here as a difference — which is the only symptom the user would ever get, and
// they would read it as the session having been lost.
func TestReattachingAndReplayingReproducesTheSameScreen(t *testing.T) {
	journal := []protocol.Event{
		ev(t, 1, protocol.EventSessionCreated, protocol.Session{
			ID: "s1", Workspace: "/w", Model: "MiniMax-M3", SandboxMode: "workspace-write",
			State: protocol.SessionStateIdle,
		}),
		ev(t, 2, protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1"}),
		ev(t, 3, protocol.EventMessageDelta, protocol.MessageDelta{TurnID: "t1", Text: "Looking at "}),
		ev(t, 4, protocol.EventMessageDelta, protocol.MessageDelta{TurnID: "t1", Text: "the parser."}),
		ev(t, 5, protocol.EventToolRequested, protocol.ToolRequested{
			Name: "read", Input: json.RawMessage(`{"path":"parser.go"}`),
		}),
		ev(t, 6, protocol.EventToolCompleted, protocol.ToolCompleted{OK: true, Output: "120 lines"}),
		ev(t, 7, protocol.EventPlanUpdated, protocol.PlanUpdated{Items: []protocol.PlanItem{
			{ID: 1, Text: "read the parser", Status: protocol.PlanDone},
			{ID: 2, Text: "add the test", Status: protocol.PlanActive},
		}}),
		ev(t, 8, protocol.EventTurnCompleted, protocol.TurnCompleted{TurnID: "t1"}),
	}
	g := DefaultGeometry(100, 30)

	live := NewModel("s1", "/w", "MiniMax-M3", "workspace-write", En)
	for _, e := range journal {
		live = live.Apply(e)
	}
	watched := Render(live, g)

	// The client is closed and a fresh one replays the same journal from 1.
	resumed := NewModel("s1", "/w", "MiniMax-M3", "workspace-write", En)
	for _, e := range journal {
		resumed = resumed.Apply(e)
	}
	if got := Render(resumed, g); got != watched {
		t.Errorf("reattaching produced a different screen; the client is holding state "+
			"the events do not carry:\n--- watched\n%s\n--- resumed\n%s", watched, got)
	}
	if resumed.LastSeq != live.LastSeq {
		t.Errorf("sequence diverged: %d vs %d", resumed.LastSeq, live.LastSeq)
	}
}

// A diff shows what changed. Rendering the whole file would bury the
// conversation it belongs to, and the unchanged lines are the ones the reader
// already knows.
func TestADiffNeverRendersTheWholeFile(t *testing.T) {
	var body strings.Builder
	for i := 0; i < 300; i++ {
		body.WriteString("UNCHANGED-LINE\n")
	}
	m := modelWithPlan()
	m.Entries = []Entry{{
		Kind: KindTool, Tool: "edit", Target: "big.go", Summary: "edited big.go",
		Expanded: true,
		Detail: "--- a/big.go\n+++ b/big.go\n@@ -1,3 +1,3 @@\n" +
			"-old\n+new\n" + body.String(),
	}}

	out := Render(m, DefaultGeometry(100, 40))
	if n := strings.Count(out, "UNCHANGED-LINE"); n > 40 {
		t.Errorf("%d unchanged lines reached the screen; the diff is rendering the file", n)
	}
	if !strings.Contains(out, "new") {
		t.Errorf("the change itself is missing:\n%s", out)
	}
}

// The panel counter and the status counter are the same sentence. Two
// formulations of one fact is how a user comes to believe the panel and the
// status bar are describing different things — and then asks which is right.
func TestThePanelAndTheStatusCountThePlanTheSameWay(t *testing.T) {
	m := modelWithPlan()
	summary := m.PlanSummary()
	if summary == "" {
		t.Fatal("the plan summary is empty; there is nothing to compare")
	}

	wide := DefaultGeometry(140, 30)
	wide.PanelMode = PanelShown
	if out := Render(m, wide); !strings.Contains(out, summary) {
		t.Errorf("the panel does not carry %q:\n%s", summary, out)
	}
	narrow := DefaultGeometry(80, 30)
	if out := Render(m, narrow); !strings.Contains(out, summary) {
		t.Errorf("the status bar does not carry %q:\n%s", summary, out)
	}
}

// RN-7: the modal asks one question. The plan beside it is the work the answer
// is about, and showing it inside the box turns a decision into a reading
// exercise — at the one moment the product has the user's full attention and
// needs it on a single yes or no.
func TestTheApprovalModalDoesNotShowThePlan(t *testing.T) {
	m := modelWithPlan()
	m.Pending = &protocol.ApprovalRequest{
		ApprovalID: "a1", Tool: "bash", Command: "rm -rf build",
		BoundaryCrossed: "filesystem-write", Reason: "writing outside the workspace",
	}

	out := Render(m, DefaultGeometry(100, 30))
	box := modalBox(out)
	if box == "" {
		t.Fatalf("no modal was drawn:\n%s", out)
	}
	if !strings.Contains(box, "rm -rf build") {
		t.Fatalf("the modal does not show the command it is asking about:\n%s", box)
	}
	for _, item := range []string{"read the parser", "add the test", "publish"} {
		if strings.Contains(box, item) {
			t.Errorf("the plan item %q is inside the modal; the question is one "+
				"decision, not a reading exercise:\n%s", item, box)
		}
	}
}

// modalBox returns the lines between the modal's borders, so the assertion is
// about what the question shows rather than about what happens to be behind it.
func modalBox(screen string) string {
	var inside []string
	open := false
	for _, line := range strings.Split(screen, "\n") {
		switch {
		case strings.Contains(line, "┌") && strings.Contains(line, "┐"):
			open = true
		case strings.Contains(line, "└") && strings.Contains(line, "┘"):
			open = false
		case open:
			inside = append(inside, line)
		}
	}
	return strings.Join(inside, "\n")
}

// Reattaching to a session that already has history must not show the splash.
// The empty state says "nothing has happened yet"; showing it over a
// conversation says the opposite of what is true, and the obvious reading is
// that the session was lost.
func TestAResumedSessionNeverShowsTheEmptyState(t *testing.T) {
	m := NewModel("s1", "/w", "MiniMax-M3", "workspace-write", En)
	// Replayed history: entries present, no turn started in THIS attachment.
	m.Entries = []Entry{
		{Kind: KindUser, Summary: "fix the parser"},
		{Kind: KindAssistant, Summary: "Done."},
	}
	if m.ShowEmptyState() {
		t.Error("a session with history showed the empty state; the obvious reading " +
			"is that the session was lost")
	}
	if out := Render(m, DefaultGeometry(100, 30)); strings.Contains(out, "? help") {
		t.Errorf("the splash rendered over a replayed conversation:\n%s", out)
	}
}

// The mascot is the identity, and a mark that cannot render in its own terminal
// is external decoration. Both fallbacks keep the three boxes and the eye — a
// shape that survives losing Unicode and losing colour is a shape, not an
// effect.
func TestTheMascotKeepsItsShapeWithoutUnicodeAndWithoutColour(t *testing.T) {
	m := NewModel("s1", "/w", "MiniMax-M3", "workspace-write", En)

	for _, c := range []struct {
		name    string
		unicode bool
		colour  bool
	}{
		{"full", true, true},
		{"ascii", false, true},
		{"monochrome", true, false},
		{"ascii and monochrome", false, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			g := DefaultGeometry(100, 30)
			g.Unicode = c.unicode
			if !c.colour {
				g.Palette = Palette{}
			}
			out := Render(m, g)

			block := "█"
			if !c.unicode {
				block = "#"
			}
			// Three stacked boxes: the narrow head, the middle, and the base.
			// Counted by the rows that are solid body, which is what makes the
			// silhouette read as a stack rather than a blob.
			rows := 0
			for _, line := range strings.Split(out, "\n") {
				if strings.Count(line, block) >= 4 {
					rows++
				}
			}
			if rows < 3 {
				t.Errorf("only %d body rows survived; the three boxes are the mascot:\n%s", rows, out)
			}
			eye := "▀▀"
			if !c.unicode {
				eye = "oo"
			}
			if !strings.Contains(out, eye) {
				t.Errorf("the eye is missing:\n%s", out)
			}
		})
	}
}
