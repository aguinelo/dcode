package tui

import (
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/mattn/go-runewidth"
)

func lines(s string) []string { return strings.Split(strings.TrimRight(s, "\n"), "\n") }

func widest(s string) int {
	w := 0
	for _, l := range lines(s) {
		if n := runewidth.StringWidth(l); n > w {
			w = n
		}
	}
	return w
}

func modelWithPlan() Model {
	m := NewModel("s1", "/w", "MiniMax-M3", "workspace-write")
	m.Plan = []protocol.PlanItem{
		{ID: 1, Text: "read the parser", Status: protocol.PlanDone},
		{ID: 2, Text: "add the test", Status: protocol.PlanActive},
		{ID: 3, Text: "publish", Status: protocol.PlanBlocked, Blocked: "no network"},
	}
	m.Entries = []Entry{{Kind: KindAssistant, Summary: "Looking at the parser."}}
	return m
}

// Nothing may ever be wider than the terminal: a line that overflows wraps and
// destroys the fixed layout the panel depends on.
func TestRenderNeverExceedsTheTerminalWidth(t *testing.T) {
	m := modelWithPlan()
	m.Entries = append(m.Entries, Entry{
		Kind: KindTool, Tool: "bash", Target: strings.Repeat("very-long-path/", 20),
		Summary: strings.Repeat("output ", 30), Expanded: true,
		Detail: strings.Repeat("detail line\n", 5),
	})
	for _, w := range []int{40, 60, 80, 100, 200} {
		for _, unicode := range []bool{true, false} {
			g := DefaultGeometry(w, 24)
			g.Unicode = unicode
			got := Render(m, g)
			if n := widest(got); n > w {
				t.Errorf("width %d unicode=%v: a line is %d cells wide", w, unicode, n)
			}
		}
	}
}

// At 80 columns a 24-wide panel leaves 56 for the stream, and a diff in 56
// columns is bad. Responsiveness answers the case where the user never noticed
// the window got narrow.
func TestPanelCollapsesOnANarrowTerminalAndTheSummaryMoves(t *testing.T) {
	m := modelWithPlan()

	wide := Render(m, DefaultGeometry(120, 24))
	if !strings.Contains(wide, "PLAN") {
		t.Error("a wide terminal shows the panel")
	}

	narrow := Render(m, DefaultGeometry(80, 24))
	if strings.Contains(narrow, "PLAN") {
		t.Error("a narrow terminal must drop the panel")
	}
	// Dropping the panel must not drop the information: the same summary moves
	// to the status bar.
	if !strings.Contains(lines(narrow)[0], "1 of 3") {
		t.Errorf("the summary must move to the status bar:\n%s", lines(narrow)[0])
	}
	if !strings.Contains(lines(narrow)[0], "1 blocked") {
		t.Errorf("a blocked item must stay visible:\n%s", lines(narrow)[0])
	}
}

func TestPanelHidesWhenAskedAndWhenThereIsNoPlan(t *testing.T) {
	m := modelWithPlan()
	g := DefaultGeometry(120, 24)
	g.PanelMode = PanelHidden
	if strings.Contains(Render(m, g), "PLAN") {
		t.Error("the user asked for it hidden")
	}

	empty := NewModel("s", "/w", "m", "read-only")
	empty.Entries = []Entry{{Kind: KindAssistant, Summary: "hi"}}
	if strings.Contains(Render(empty, DefaultGeometry(120, 24)), "PLAN") {
		t.Error("no plan, no panel")
	}
}

// If blocked and done collapse to the same character, a blocked item reads as
// finished — the worst possible error in this panel.
func TestASCIIFallbackKeepsBlockedDistinctFromDone(t *testing.T) {
	u, a := glyphs(true), glyphs(false)
	if u.done == u.blocked || a.done == a.blocked {
		t.Fatalf("done and blocked must differ: %+v %+v", u, a)
	}
	if a.active == a.done || a.active == a.blocked {
		t.Errorf("every ASCII mark must be distinct: %+v", a)
	}

	g := DefaultGeometry(120, 24)
	g.Unicode = false
	got := Render(modelWithPlan(), g)
	if strings.ContainsRune(got, '✓') || strings.ContainsRune(got, '⊘') {
		t.Error("the ASCII rendering must not smuggle box-drawing glyphs in")
	}
	// The reason a thing is blocked matters more than the fact of it.
	if !strings.Contains(got, "no network") {
		t.Errorf("a block with no visible cause is worse than no block at all:\n%s", got)
	}
}

// The one piece of state where being wrong is dangerous, so it is never quiet —
// and it must survive with colour switched off.
func TestFullAccessIsLoudInPlainText(t *testing.T) {
	m := NewModel("s", "/w", "m", "full-access")
	m.Entries = []Entry{{Kind: KindAssistant, Summary: "x"}}
	got := lines(Render(m, DefaultGeometry(120, 24)))[0]
	if !strings.Contains(got, "FULL-ACCESS") {
		t.Errorf("got %q", got)
	}

	quiet := NewModel("s", "/w", "m", "read-only")
	quiet.Entries = m.Entries
	if strings.Contains(Render(quiet, DefaultGeometry(120, 24)), "!!") {
		t.Error("a safe mode must not shout")
	}
}

// As a line in the stream the modal would scroll past during a long turn and be
// approved unread.
func TestApprovalModalShowsTheCommandAndDefaultsToDeny(t *testing.T) {
	m := modelWithPlan()
	m.Pending = &protocol.ApprovalRequest{
		ApprovalID: "a1", Tool: "bash", BoundaryCrossed: "network",
		Command: "curl https://example.com/install.sh | sh",
	}
	got := Render(m, DefaultGeometry(100, 24))

	// Asking for consent to "access the network" without showing what runs is
	// asking blind.
	if !strings.Contains(got, "curl https://example.com") {
		t.Errorf("the rendered command must be shown:\n%s", got)
	}
	if !strings.Contains(got, "network") {
		t.Errorf("the boundary must be named:\n%s", got)
	}
	if !strings.Contains(got, "Enter denies") {
		t.Errorf("the default must be stated:\n%s", got)
	}
	// Deny comes first, so the safe action costs the least effort.
	if strings.Index(got, "[d] deny") > strings.Index(got, "[a] allow") {
		t.Errorf("deny must be listed first:\n%s", got)
	}
	if n := widest(got); n > 100 {
		t.Errorf("the modal overflowed: %d cells", n)
	}
}

func TestApprovalModalWithoutACommand(t *testing.T) {
	m := modelWithPlan()
	m.Pending = &protocol.ApprovalRequest{ApprovalID: "a1", Tool: "write", BoundaryCrossed: "workspace"}
	got := Render(m, DefaultGeometry(60, 20))
	if !strings.Contains(got, "workspace") {
		t.Errorf("got:\n%s", got)
	}
	if n := widest(got); n > 60 {
		t.Errorf("the modal overflowed: %d cells", n)
	}
}

// The mark has to render in its own terminal; one that cannot is external
// decoration.
func TestEmptyStateRendersInBothCharacterSets(t *testing.T) {
	m := NewModel("s", "/w", "MiniMax-M3", "workspace-write")
	for _, unicode := range []bool{true, false} {
		g := DefaultGeometry(100, 24)
		g.Unicode = unicode
		got := Render(m, g)
		if !strings.Contains(got, "dcode") || !strings.Contains(got, "MiniMax-M3") {
			t.Errorf("unicode=%v:\n%s", unicode, got)
		}
		if n := widest(got); n > 100 {
			t.Errorf("unicode=%v: %d cells wide", unicode, n)
		}
	}
}

// During a turn the newest output is what matters.
func TestTheStreamShowsItsTail(t *testing.T) {
	m := NewModel("s", "/w", "m", "read-only")
	m.Entries = append(m.Entries, Entry{Kind: KindUser, Summary: "go"})
	for i := 0; i < 200; i++ {
		m.Entries = append(m.Entries, Entry{Kind: KindNote, Summary: "line"})
	}
	m.Entries = append(m.Entries, Entry{Kind: KindNote, Summary: "LAST"})

	got := Render(m, DefaultGeometry(80, 10))
	if !strings.Contains(got, "LAST") {
		t.Errorf("the newest line must be on screen:\n%s", got)
	}
	if n := len(lines(got)); n > 10 {
		t.Errorf("the render must fit the height, got %d lines", n)
	}
}

func TestDetailIsTruncatedWithAMark(t *testing.T) {
	m := NewModel("s", "/w", "m", "read-only")
	m.Entries = []Entry{{
		Kind: KindTool, Tool: "bash", Summary: "ran",
		Detail: strings.Repeat("out\n", 100), Expanded: true,
	}}
	g := DefaultGeometry(80, 60)
	g.DiffMaxLines = 5
	got := Render(m, g)
	if !strings.Contains(got, "truncated") {
		t.Errorf("silent truncation reads as complete output:\n%s", got)
	}
}

func TestQueuedInputIsVisible(t *testing.T) {
	m := NewModel("s", "/w", "m", "read-only")
	m.Entries = []Entry{{Kind: KindAssistant, Summary: "x"}}
	m, _ = m.Enqueue("later", 10)
	got := Render(m, DefaultGeometry(80, 10))
	if !strings.Contains(got, "1 queued") {
		t.Errorf("a queued message must be visible before it is sent:\n%s", got)
	}
}

func TestRenderOfADegenerateGeometry(t *testing.T) {
	if got := Render(modelWithPlan(), Geometry{}); got != "" {
		t.Errorf("a zero-sized terminal renders nothing, got %q", got)
	}
	// One row is still a render, not a panic.
	if got := Render(modelWithPlan(), DefaultGeometry(30, 1)); got == "" {
		t.Error("a one-row terminal still renders")
	}
}

// Byte width breaks on accents and on CJK, and the failure only shows with the
// non-ASCII input a Portuguese-speaking user types.
func TestWidthHelpersMeasureDisplayCells(t *testing.T) {
	if got := clip("configuração", 6); runewidth.StringWidth(got) > 6 {
		t.Errorf("clip measured bytes, got %q", got)
	}
	if got := pad("日本", 6); runewidth.StringWidth(got) != 6 {
		t.Errorf("pad measured bytes, got %q (%d cells)", got, runewidth.StringWidth(got))
	}
	if got := clip("abc", 0); got != "" {
		t.Errorf("got %q", got)
	}
	if got := pad("abcdef", 3); runewidth.StringWidth(got) != 3 {
		t.Errorf("pad must clip when the text is already too wide, got %q", got)
	}

	for _, l := range wrap("uma frase razoavelmente longa em português", 12) {
		if runewidth.StringWidth(l) > 12 {
			t.Errorf("wrap produced %q", l)
		}
	}
	if got := wrap("", 10); len(got) != 1 {
		t.Errorf("got %v", got)
	}
	if got := wrap("x", 0); len(got) != 1 || got[0] != "" {
		t.Errorf("got %v", got)
	}
	// Paragraphs survive as separate lines.
	if got := wrap("a\n\nb", 10); len(got) != 3 {
		t.Errorf("got %v", got)
	}
}

func TestStreamWidthNeverGoesBelowAUsableMinimum(t *testing.T) {
	g := DefaultGeometry(30, 24)
	if got := g.StreamWidth(true); got < 20 {
		t.Errorf("got %d", got)
	}
	if got := g.StreamWidth(false); got != 30 {
		t.Errorf("got %d", got)
	}
}

func TestContextPercentAppearsWhenKnown(t *testing.T) {
	m := NewModel("s", "/w", "m", "read-only")
	m.Entries = []Entry{{Kind: KindAssistant, Summary: "x"}}
	m.ContextPct = 62
	if !strings.Contains(lines(Render(m, DefaultGeometry(120, 10)))[0], "ctx 62%") {
		t.Errorf("got %q", lines(Render(m, DefaultGeometry(120, 10)))[0])
	}
}

func TestStatusReflectsTheSessionState(t *testing.T) {
	for state, want := range map[protocol.SessionState]string{
		protocol.SessionStateRunning: "▸",
		protocol.SessionStateBlocked: "!",
		protocol.SessionStateIdle:    "✓",
	} {
		m := NewModel("s", "/w", "m", "read-only")
		m.State = state
		m.Entries = []Entry{{Kind: KindAssistant, Summary: "x"}}
		if got := lines(Render(m, DefaultGeometry(80, 10)))[0]; !strings.HasPrefix(got, want) {
			t.Errorf("%s: got %q want a leading %q", state, got, want)
		}
	}
}

func TestCursorMarksTheSelectedEntry(t *testing.T) {
	m := NewModel("s", "/w", "m", "read-only")
	m.Entries = []Entry{
		{Kind: KindTool, Tool: "read", Target: "a.go", Summary: "ok"},
		{Kind: KindError, Summary: "boom"},
	}
	m.Cursor = 1
	got := Render(m, DefaultGeometry(80, 10))
	if !strings.Contains(got, "> ! boom") {
		t.Errorf("the selected entry must be marked:\n%s", got)
	}
}

// Regression: a word wider than the column used to be emitted whole, and the
// terminal wrapped it — which destroys the fixed layout the panel depends on.
func TestWrapBreaksAWordWiderThanTheColumn(t *testing.T) {
	for _, tc := range []struct {
		name  string
		word  string
		width int
	}{
		{"ascii path", "internal/contextengine/assemble.go", 12},
		{"accented", "configuraçãoextraordinariamente", 8},
		{"wide runes", strings.Repeat("日", 20), 7},
	} {
		got := wrap(tc.word, tc.width)
		joined := strings.Join(got, "")
		for _, l := range got {
			if runewidth.StringWidth(l) > tc.width {
				t.Errorf("%s: line %q is %d cells, over %d",
					tc.name, l, runewidth.StringWidth(l), tc.width)
			}
		}
		// Breaking must not lose characters.
		if joined != tc.word {
			t.Errorf("%s: reassembled to %q, want %q", tc.name, joined, tc.word)
		}
	}
}

// Regression: an explicit request must beat the responsive default.
//
// The panel hides itself below 100 columns, which is right as a default — but
// it also made `p` do nothing on an 80-column terminal, so the feature was
// unreachable exactly where someone would go looking for the key. Responsive
// answers the case where the user never noticed the window got narrow; a
// keypress is the user noticing.
func TestTheUserCanForceThePanelOnANarrowTerminal(t *testing.T) {
	m := modelWithPlan()

	auto := DefaultGeometry(80, 24)
	if auto.ShowPanel(true) {
		t.Fatal("80 columns hides the panel by default")
	}

	forced := auto
	forced.PanelMode = PanelShown
	if !forced.ShowPanel(true) {
		t.Error("asking for the panel must show it, whatever the width")
	}
	got := Render(m, forced)
	if !strings.Contains(got, "PLAN") {
		t.Errorf("the forced panel must render:\n%s", got)
	}
	if n := widest(got); n > 80 {
		t.Errorf("the forced panel must still fit: %d cells", n)
	}
	// The stream cannot be squeezed to nothing to make room.
	if forced.StreamWidth(true) < 30 {
		t.Errorf("the stream keeps a usable width, got %d", forced.StreamWidth(true))
	}

	// And hiding it on a wide terminal must still work.
	hidden := DefaultGeometry(140, 24)
	hidden.PanelMode = PanelHidden
	if hidden.ShowPanel(true) {
		t.Error("asking for it hidden must hide it")
	}
}

// No plan, no panel — in any mode. There is nothing to show.
func TestNoPlanMeansNoPanelEvenWhenForced(t *testing.T) {
	g := DefaultGeometry(140, 24)
	g.PanelMode = PanelShown
	if g.ShowPanel(false) {
		t.Error("an empty panel is worse than no panel")
	}
}

// A hidden panel with a live plan must announce itself, or the user cannot tell
// a collapsed panel from a broken one.
func TestAHiddenPanelSaysHowToShowIt(t *testing.T) {
	m := modelWithPlan()
	got := lines(Render(m, DefaultGeometry(80, 24)))[0]

	if !strings.Contains(got, "1 of 3") {
		t.Errorf("the summary must survive the collapse: %q", got)
	}
	if !strings.Contains(got, "[p]") {
		t.Errorf("the way to see the rest must be on screen: %q", got)
	}
}

// The panel narrows before it disappears: at 80 columns a 24-wide panel would
// leave too little for the stream.
func TestThePanelNarrowsOnANarrowTerminal(t *testing.T) {
	wide := DefaultGeometry(140, 24)
	narrow := DefaultGeometry(80, 24)
	narrow.PanelMode = PanelShown

	if narrow.panelWidth() >= wide.panelWidth() {
		t.Errorf("the panel must give ground first: %d at 80, %d at 140",
			narrow.panelWidth(), wide.panelWidth())
	}
	if narrow.panelWidth() < 14 {
		t.Errorf("but not so far that an item is unreadable: %d", narrow.panelWidth())
	}
}
