package tui

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
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

	// Asserted over the STREAM and not over the rendered window.
	//
	// It used to render a 40-row terminal and look for the change in it, which
	// made a claim about the diff depend on how many rows the layout happened
	// to leave: the stream is anchored to its end, the change is near the start
	// of a forty-line block, and the fixture was one row from failing. Adding
	// two rules around the input area spent that row, and the test reported the
	// diff as broken when the diff was fine.
	out := strings.Join(StreamLines(m, DefaultGeometry(100, 40)), "\n")
	if n := strings.Count(out, "UNCHANGED-LINE"); n > 40 {
		t.Errorf("%d unchanged lines reached the screen; the diff is rendering the file", n)
	}
	if !strings.Contains(out, "new") {
		t.Errorf("the change itself is missing:\n%s", out)
	}
}

// The plan block and the status counter count the same plan the same way. Two
// formulations of one fact is how a user comes to believe two regions are
// describing different things — and then asks which is right.
//
// The panel this used to compare against is gone with the plan that justified
// it; the comparison survives it, between the block in the stream and the bar.
func TestThePlanBlockAndTheStatusCountThePlanTheSameWay(t *testing.T) {
	m := modelWithPlan()
	summary := m.PlanSummary()
	if summary == "" {
		t.Fatal("the plan summary is empty; there is nothing to compare")
	}

	done, total, _ := m.PlanCounts()
	for _, w := range []int{80, 140} {
		out := Render(m, DefaultGeometry(w, 30))
		if !strings.Contains(out, summary) {
			t.Errorf("at %d columns the status bar does not carry %q:\n%s", w, summary, out)
		}
		if want := fmt.Sprintf("%d/%d", done, total); !strings.Contains(out, want) {
			t.Errorf("at %d columns the plan block does not count %q:\n%s", w, want, out)
		}
	}
}

// RN-7: the modal asks one question. The plan beside it is the work the answer
// is about, and showing it inside the box turns a decision into a reading
// exercise — at the one moment the product has the user's full attention and
// needs it on a single yes or no.
func TestTheApprovalBlockAsksOneQuestion(t *testing.T) {
	m := modelWithPlan()
	m.Pending = &protocol.ApprovalRequest{
		ApprovalID: "a1", Tool: "bash", Command: "rm -rf build",
		BoundaryCrossed: "filesystem-write", Reason: "writing outside the workspace",
	}
	m.Entries = append(m.Entries, Entry{Kind: KindApproval, Approval: m.Pending})

	g := DefaultGeometry(100, 30)
	g.Palette = Palette{}
	block := renderApprovalBlock(m.Entries[len(m.Entries)-1], "  ", "  ",
		glyphs(g.Unicode), g, m.Lang, g.StreamWidth(false))
	got := strings.Join(block, "\n")

	if !strings.Contains(got, "rm -rf build") {
		t.Fatalf("the block does not show the command it is asking about:\n%s", got)
	}
	// The question is one decision, not a reading exercise. The plan is in the
	// stream above it now rather than in a box behind it, and it must not be
	// INSIDE the block — which is the same rule, applied to the shape the
	// question actually has.
	for _, item := range []string{"read the parser", "add the test", "publish"} {
		if strings.Contains(got, item) {
			t.Errorf("the plan item %q is inside the block:\n%s", item, got)
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

// A continued conversation is on the screen, and says it came from before.
//
// The history reached the model and never the person: the screen is built from
// events, and continuing emitted none. Someone who ran `-r` found a blank
// window, and the only available reading was that the work was gone.
func TestAContinuedConversationIsOnTheScreenAndSaysWhereItCameFrom(t *testing.T) {
	m := NewModel("new", "/w", "MiniMax-M3", "workspace-write", En)
	m = m.Apply(ev(t, 2, protocol.EventSessionResumed,
		protocol.SessionResumed{SourceID: "1a015fb", Turns: 2}))
	m = m.Apply(ev(t, 3, protocol.EventTurnStarted,
		protocol.TurnStarted{TurnID: "t1", Text: "tem acesso a rede?"}))

	out := Render(m, DefaultGeometry(100, 30))
	if !strings.Contains(out, "1a015fb") {
		t.Errorf("the screen does not say which session was continued:\n%s", out)
	}
	if !strings.Contains(out, "tem acesso a rede?") {
		t.Errorf("the carried conversation is not on the screen:\n%s", out)
	}
	if m.ShowEmptyState() {
		t.Error("a continued session showed the empty state; the obvious reading " +
			"is that the conversation was lost")
	}
}

// Colour may change how the screen looks and never what it says.
//
// The Palette's own comment states this as design — "every caller measures
// display cells before styling" — and nothing checked it. `turnSection` clipped
// a string it had already styled, and `clip` measures with runewidth, which
// counts the printable bytes of an escape sequence as six cells of text. So the
// panel cut six characters early on a colour terminal and kept them on a
// monochrome one, and `runewidth.Truncate` could cut inside the escape itself,
// leaving the terminal in that colour for the rest of the screen.
//
// Asserted over the whole frame rather than per call site. A rule with one
// counter-example has more, and per-site tests only ever find the site that was
// already suspected.
func TestColourNeverChangesWhatIsOnTheScreen(t *testing.T) {
	m := modelWithPlan()
	m.Rounds, m.MaxRounds = 16, 2000
	m.InFlight, m.MaxInFlight = 1, 4
	m.Plan = append(m.Plan, protocol.PlanItem{
		ID: 4, Text: "a plan item long enough that the panel has to cut it",
		Status: protocol.PlanActive,
	})
	m.Entries = append(m.Entries,
		Entry{Kind: KindTool, Tool: "write", Target: "internal/tui/render.go", Summary: "created, 21 lines", Added: 21},
		Entry{Kind: KindTool, Tool: "bash", Target: "go test ./...", Summary: "exit 1", IsError: true},
		Entry{Kind: KindAssistant, Summary: "A paragraph long enough to wrap across the stream more than once, so the wrapping is exercised too."},
	)
	m.Sessions = []SessionChoice{{ID: "a", Title: "a conversation whose title is far too long for the column"}}

	for _, w := range []int{80, 100, 132, 160} {
		mono := DefaultGeometry(w, 30)
		colour := DefaultGeometry(w, 30)
		colour.Palette = Palette{Enabled: true}

		got := stripANSI(Render(m, colour))
		want := Render(m, mono)
		if got == want {
			continue
		}
		gl, wl := lines(got), lines(want)
		for i := range wl {
			if i < len(gl) && gl[i] != wl[i] {
				t.Fatalf("at %d columns, line %d differs\n colour: %q\n   mono: %q", w, i, gl[i], wl[i])
			}
		}
		t.Fatalf("at %d columns the frames differ in length: %d vs %d", w, len(gl), len(wl))
	}
}

// A boundary decision is a record, not a dialogue. The modal deleted itself on
// the way out, taking with it both the question and the answer: half an hour
// later there was no way to see what had been allowed, or why it was asked.
// Matching by ApprovalID and not "the last one" is the same guarantee from the
// other side — with two questions in flight, the last one answers the wrong.
func TestTheAnsweredQuestionStaysInTheStream(t *testing.T) {
	m := NewModel("s1", "/w", "opus", "workspace-write", En)
	first := protocol.ApprovalRequest{
		ApprovalID: "a1", Tool: "bash", Command: "curl https://example.com",
		BoundaryCrossed: "network",
	}
	second := protocol.ApprovalRequest{
		ApprovalID: "a2", Tool: "write", Command: "write /etc/hosts",
		BoundaryCrossed: "filesystem-write",
	}
	m = apply(t, m,
		ev(t, 1, protocol.EventApprovalRequired, first),
		ev(t, 2, protocol.EventApprovalRequired, second),
		ev(t, 3, protocol.EventApprovalResolved, protocol.ApprovalResolved{
			ApprovalID: "a1", Decision: protocol.ApprovalAllowSession,
		}))

	var answered, open int
	for _, e := range m.Entries {
		if e.Kind != KindApproval {
			continue
		}
		if e.Decision == "" {
			open++
			if e.Approval.ApprovalID != "a2" {
				t.Errorf("the resolution landed on %s, not on the request it named",
					e.Approval.ApprovalID)
			}
			continue
		}
		answered++
		if e.Approval.ApprovalID != "a1" {
			t.Errorf("the answer landed on %s, not on a1", e.Approval.ApprovalID)
		}
	}
	if answered != 1 || open != 1 {
		t.Fatalf("want one answered question and one still open, got %d and %d",
			answered, open)
	}

	out := Render(m, DefaultGeometry(100, 30))
	if !strings.Contains(out, "curl https://example.com") {
		t.Errorf("the answered question left the stream:\n%s", out)
	}
	if !strings.Contains(out, "allowed for this session") {
		t.Errorf("the answer is not shown next to the question:\n%s", out)
	}
}

// A fatal must outlive the screen it was drawn on.
//
// The alternate screen takes the last frame with it when the program ends, and
// the last frame is where the fatal was written — so the one message the person
// needed was the one guaranteed to be wiped. `dcode -c` failing looked like
// `dcode -c` doing nothing.
func TestAFatalOutlivesTheAlternateScreen(t *testing.T) {
	p, _ := newProgram(t)
	p.fatal = "events before 1 are no longer held"

	// What Run returns, not a second copy of the rule.
	if err := outcome(nil, p.fatal); err == nil {
		t.Fatal("the run reported success while the client had a fatal on screen")
	} else if !strings.Contains(err.Error(), "no longer held") {
		t.Errorf("the fatal was replaced by something else: %v", err)
	}
	// A real error is not replaced by the screen's copy of it.
	real := errors.New("the daemon went away")
	if err := outcome(real, p.fatal); !errors.Is(err, real) {
		t.Errorf("the run's own error was overwritten: %v", err)
	}
}

// No screen shows what the turn cost as if it were the context.
//
// Three times now, in three places: the model computed it right, and the status
// bar, then the side column, went on drawing from the cumulative input count
// beside it. Each was found by a person looking at their own screen, and each
// was fixed with a test that asked about the one place it had just been found
// in — a guard that can only ever confirm the fix it was written for.
//
// This one asks the whole screen. InputTokens is what a turn COST across its
// rounds, and every round re-sends the context, so it is routinely several
// times the window; ContextTokens is what the context IS. Nothing anywhere may
// show the first against the second.
func TestNoScreenShowsTheTurnsCostAsTheContext(t *testing.T) {
	m := NewModel("s", "/w", "m", "workspace-write", En)
	// A real record from this machine: 5,917,178 cumulative against a
	// million-token window whose context measured 363,500.
	m.Window, m.ContextTokens, m.ContextPct = 1_000_000, 363_500, 36
	m.InputTokens = 5_917_178
	m.DiffAdded, m.DiffRemoved, m.DiffFiles = 4758, 246, 99
	m.Entries = append(m.Entries,
		Entry{Kind: KindTool, Tool: "edit", Target: "internal/tui/side.go", Added: 3},
	)

	for _, width := range []int{80, 100, 132, 200} {
		g := DefaultGeometry(width, 40)
		g.Palette = Palette{}
		g.RailMode = RailShown
		screen := Render(m, g)

		// The cumulative figure, however it is spelled.
		for _, forbidden := range []string{"5.9M", "5917178", "5,917,178"} {
			if strings.Contains(screen, forbidden) {
				t.Errorf("at %d columns the screen shows %q, which is the turn's "+
					"cost and not the context:\n%s", width, forbidden, screen)
			}
		}
		// And no share of a window may exceed the window, anywhere.
		for _, m := range percentPattern.FindAllStringSubmatch(screen, -1) {
			n, err := strconv.Atoi(m[1])
			if err == nil && n > 100 {
				t.Errorf("at %d columns the screen shows %s%%:\n%s", width, m[1], screen)
			}
		}
	}
}

// percentPattern finds every percentage drawn on a screen.
var percentPattern = regexp.MustCompile(`(\d+)%`)
