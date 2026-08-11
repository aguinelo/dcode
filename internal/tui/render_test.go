package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/mattn/go-runewidth"
)

func lines(s string) []string { return strings.Split(strings.TrimRight(s, "\n"), "\n") }

// widest measures display cells, which is what "fits the terminal" means.
// Counting the bytes of an escape sequence would report a coloured line as
// wider than the monochrome one that renders identically.
func widest(s string) int {
	w := 0
	for _, l := range lines(s) {
		if n := visibleWidth(l); n > w {
			w = n
		}
	}
	return w
}

func modelWithPlan() Model {
	m := NewModel("s1", "/w", "MiniMax-M3", "workspace-write", En)
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

	empty := NewModel("s", "/w", "m", "read-only", En)
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
	m := NewModel("s", "/w", "m", "full-access", En)
	m.Entries = []Entry{{Kind: KindAssistant, Summary: "x"}}
	got := lines(Render(m, DefaultGeometry(120, 24)))[0]
	if !strings.Contains(got, "FULL-ACCESS") {
		t.Errorf("got %q", got)
	}

	quiet := NewModel("s", "/w", "m", "read-only", En)
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
	m := NewModel("s", "/w", "MiniMax-M3", "workspace-write", En)
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
	m := NewModel("s", "/w", "m", "read-only", En)
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
	m := NewModel("s", "/w", "m", "read-only", En)
	m.Entries = []Entry{{
		Kind: KindTool, Tool: "bash", Summary: "ran",
		Detail: strings.Repeat("out\n", 100), Expanded: true,
	}}
	g := DefaultGeometry(80, 60)
	g.DiffMaxLines = 5
	got := Render(m, g)
	// How much is hidden and how to see it: "truncated" alone leaves the reader
	// unable to judge whether it matters.
	if !strings.Contains(got, "95 lines") {
		t.Errorf("the hidden count must be stated:\n%s", got)
	}
	if !strings.Contains(got, "Tab") {
		t.Errorf("and the way to see them:\n%s", got)
	}
}

func TestQueuedInputIsVisible(t *testing.T) {
	m := NewModel("s", "/w", "m", "read-only", En)
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
	m := NewModel("s", "/w", "m", "read-only", En)
	m.Entries = []Entry{{Kind: KindAssistant, Summary: "x"}}
	// The meter is derived from the tokens and the window, not from a
	// percentage set by hand: two fields that can disagree eventually do.
	m.InputTokens, m.Window, m.ContextPct = 62000, 100000, 62
	if got := lines(Render(m, DefaultGeometry(120, 10)))[0]; !strings.Contains(got, "ctx 62%") {
		t.Errorf("got %q", got)
	}
}

func TestStatusReflectsTheSessionState(t *testing.T) {
	for state, want := range map[protocol.SessionState]string{
		// Running animates: a static glyph cannot tell a long turn from a hung
		// one, which is the whole question the user is asking the screen.
		protocol.SessionStateRunning: Spinner(0, true),
		protocol.SessionStateBlocked: "!",
		protocol.SessionStateIdle:    "✓",
	} {
		m := NewModel("s", "/w", "m", "read-only", En)
		m.State = state
		m.Entries = []Entry{{Kind: KindAssistant, Summary: "x"}}
		if got := lines(Render(m, DefaultGeometry(80, 10)))[0]; !strings.HasPrefix(got, want) {
			t.Errorf("%s: got %q want a leading %q", state, got, want)
		}
	}
}

func TestCursorMarksTheSelectedEntry(t *testing.T) {
	m := NewModel("s", "/w", "m", "read-only", En)
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
	if !strings.Contains(got, "^p") {
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

// ---------- the drop order in the status bar ----------

// What the status bar gives up when the terminal narrows is a safety decision,
// not a layout one. The sandbox mode is the only field where being wrong is
// dangerous, so it is never what goes.
func TestTheSandboxModeSurvivesEveryWidth(t *testing.T) {
	for _, mode := range []string{"read-only", "workspace-write", "full-access"} {
		m := NewModel("s", "/w", "MiniMax-M3", mode, En)
		m.Entries = []Entry{{Kind: KindAssistant, Summary: "x"}}
		m.Plan = modelWithPlan().Plan
		m.InputTokens, m.Window = 34000, 100000

		for _, w := range []int{40, 50, 60, 80, 100, 140} {
			got := lines(Render(m, DefaultGeometry(w, 12)))[0]
			want := mode
			if mode == "full-access" {
				want = "FULL-ACCESS"
			}
			if !strings.Contains(got, want) {
				t.Errorf("%s at %d columns: the mode must survive, got %q", mode, w, got)
			}
			if visibleWidth(got) > w {
				t.Errorf("%s at %d columns: the bar overflowed (%d cells)", mode, w, visibleWidth(got))
			}
		}
	}
}

// The model name is the first thing given up, and the plan summary the last —
// the counter is the field that tells the user work is still moving.
func TestTheStatusBarDropsInAStatedOrder(t *testing.T) {
	m := NewModel("s", "/w", "MiniMax-M3", "workspace-write", En)
	m.Entries = []Entry{{Kind: KindAssistant, Summary: "x"}}
	m.Plan = modelWithPlan().Plan
	m.InputTokens, m.Window = 34000, 100000

	// The panel is hidden, so the plan counter is in the bar — that is the only
	// case where all three optional fields compete for the same line.
	g := DefaultGeometry(140, 12)
	g.PanelMode = PanelHidden
	wide := lines(Render(m, g))[0]
	for _, want := range []string{"MiniMax-M3", "ctx 34%", "1 of 3"} {
		if !strings.Contains(wide, want) {
			t.Fatalf("a wide terminal shows everything, %q missing from %q", want, wide)
		}
	}

	// Narrow enough to lose the model name but keep the rest.
	narrow := DefaultGeometry(56, 12)
	narrow.PanelMode = PanelHidden
	mid := lines(Render(m, narrow))[0]
	if strings.Contains(mid, "MiniMax-M3") {
		t.Errorf("the model name goes first: %q", mid)
	}
	if !strings.Contains(mid, "1 of 3") {
		t.Errorf("the plan counter is the last to go: %q", mid)
	}
}

// ---------- the panel takes a little more room when there is room ----------

func TestThePanelGrowsOnAWideTerminalAndIsCapped(t *testing.T) {
	narrow := DefaultGeometry(80, 24)
	narrow.PanelMode = PanelShown
	wide := DefaultGeometry(200, 24)

	if narrow.panelWidth() >= DefaultGeometry(140, 24).panelWidth() {
		t.Error("the panel must give ground on a narrow terminal")
	}
	if got := wide.panelWidth(); got != wide.PanelMaxWidth {
		t.Errorf("a wide terminal should reach the ceiling, got %d", got)
	}
	// And never past it: past a point the panel is the interface and the stream
	// is the sidebar.
	if got := DefaultGeometry(400, 24).panelWidth(); got > 34 {
		t.Errorf("got %d", got)
	}
}

// ---------- the mascot wears the brand ----------

func TestTheMascotIsColouredWithTheBrandPalette(t *testing.T) {
	g := DefaultGeometry(100, 20)
	g.Palette = Palette{Enabled: true}
	got := Render(NewModel("s", "/w", "MiniMax-M3", "read-only", En), g)

	for name, code := range map[string]string{
		"highlight": ansi[StyleHighlight],
		"body":      ansi[StyleBody],
		"shadow":    ansi[StyleShadow],
		"eye":       ansi[StyleEye],
	} {
		if !strings.Contains(got, "\x1b["+code+"m") {
			t.Errorf("the %s tone is missing from the mark", name)
		}
	}
	// The eye is the one terracotta in the whole interface.
	if n := strings.Count(got, "\x1b["+ansi[StyleEye]+"m"); n != 1 {
		t.Errorf("the eye must appear exactly once, got %d", n)
	}
	if n := widest(got); n > 100 {
		t.Errorf("the mark overflowed: %d cells", n)
	}
}

func TestTheEmptyStateStillWorksWithoutColour(t *testing.T) {
	for _, unicode := range []bool{true, false} {
		g := DefaultGeometry(100, 20)
		g.Unicode = unicode
		got := Render(NewModel("s", "/w", "MiniMax-M3", "read-only", En), g)
		if strings.ContainsRune(got, 0x1b) {
			t.Errorf("unicode=%v: monochrome must emit no escapes", unicode)
		}
		if !strings.Contains(got, "dcode") || !strings.Contains(got, "MiniMax-M3") {
			t.Errorf("unicode=%v:\n%s", unicode, got)
		}
	}
}

// full-access has to be loud even on the splash, where a user is most likely to
// be starting something without having checked how they started it.
func TestTheEmptyStateShoutsFullAccess(t *testing.T) {
	m := NewModel("s", "/w", "MiniMax-M3", "full-access", En)
	if !strings.Contains(Render(m, DefaultGeometry(100, 20)), "FULL-ACCESS") {
		t.Error("the splash must carry the warning too")
	}
}

// ---------- tool lines stack into a column ----------

// Ragged summaries are read one at a time, which is exactly what a wall of tool
// calls must not be.
func TestToolSummariesAlignIntoAColumn(t *testing.T) {
	m := NewModel("s", "/w", "m", "read-only", En)
	m.Entries = []Entry{
		{Kind: KindTool, Tool: "read", Target: "a.go", Summary: "240 lines"},
		{Kind: KindTool, Tool: "grep", Target: "func validate", Summary: "18 matches in 4 files"},
		{Kind: KindTool, Tool: "edit", Target: "internal/domain/validate.go", Summary: "+24 −2"},
	}
	rows := lines(Render(m, DefaultGeometry(110, 12)))

	var cols []int
	for _, r := range rows {
		for _, s := range []string{"240 lines", "18 matches", "+24"} {
			if i := strings.Index(r, s); i >= 0 {
				cols = append(cols, visibleWidth(r[:i]))
			}
		}
	}
	if len(cols) != 3 {
		t.Fatalf("expected three tool lines, found %d", len(cols))
	}
	for _, c := range cols[1:] {
		if c != cols[0] {
			t.Errorf("summaries must start in one column, got %v", cols)
		}
	}
}

// The end of a path is what identifies a file; the directories leading to it are
// what everything in a repository has in common.
func TestALongTargetIsShortenedFromTheFront(t *testing.T) {
	got := ellipsis("internal/http/handler/very/deep/validate.go", 20)
	if visibleWidth(got) > 20 {
		t.Errorf("got %d cells: %q", visibleWidth(got), got)
	}
	if !strings.HasSuffix(got, "validate.go") {
		t.Errorf("the identifying end must survive: %q", got)
	}
	if !strings.HasPrefix(got, "…") {
		t.Errorf("the cut must be visible: %q", got)
	}
	// Short enough to fit is left alone.
	if got := ellipsis("a.go", 20); got != "a.go" {
		t.Errorf("got %q", got)
	}
}

// ---------- the diff ----------

func TestADiffShowsWithoutBeingAskedForAndExpandsOnTab(t *testing.T) {
	diff := "@@ -1,3 +1,3 @@ x.go\n"
	for i := 0; i < 30; i++ {
		diff += fmt.Sprintf("+linha %d\n", i)
	}
	m := NewModel("s", "/w", "m", "read-only", En)
	m.Entries = []Entry{{Kind: KindTool, Tool: "edit", Target: "x.go", Summary: "+30 −0", Diff: diff}}

	g := DefaultGeometry(100, 60)
	collapsed := Render(m, g)
	if !strings.Contains(collapsed, "linha 0") {
		t.Errorf("a diff is what gets reviewed, so some of it shows:\n%s", collapsed)
	}
	if strings.Contains(collapsed, "linha 25") {
		t.Errorf("but not all of it, collapsed:\n%s", collapsed)
	}
	if !strings.Contains(collapsed, "Tab") {
		t.Errorf("and the hint must say how to see the rest:\n%s", collapsed)
	}

	m.Entries[0].Expanded = true
	if !strings.Contains(Render(m, g), "linha 25") {
		t.Error("Tab must actually reveal more, or the hint is a lie")
	}
}

// The diff wins over the raw output: the tool's prose says nothing a reviewer
// needs.
func TestTheDiffReplacesTheRawOutput(t *testing.T) {
	m := NewModel("s", "/w", "m", "read-only", En)
	m.Entries = []Entry{{
		Kind: KindTool, Tool: "edit", Target: "x.go", Summary: "+1 −1",
		Detail: "edited x.go (1 replacement)",
		Diff:   "@@ -1 +1 @@ x.go\n-antes\n+depois",
	}}
	got := Render(m, DefaultGeometry(100, 20))
	if !strings.Contains(got, "-antes") || !strings.Contains(got, "+depois") {
		t.Errorf("got:\n%s", got)
	}
	if strings.Contains(got, "1 replacement") {
		t.Errorf("the prose must give way to the diff:\n%s", got)
	}
}

func TestDiffLinesAreColouredBySign(t *testing.T) {
	g := DefaultGeometry(100, 20)
	g.Palette = Palette{Enabled: true}
	m := NewModel("s", "/w", "m", "read-only", En)
	m.Entries = []Entry{{
		Kind: KindTool, Tool: "edit", Target: "x.go",
		Diff: "@@ -1 +1 @@ x.go\n-antes\n+depois",
	}}
	got := Render(m, g)
	if !strings.Contains(got, "\x1b["+ansi[StyleAdded]+"m") {
		t.Error("an added line must be green")
	}
	if !strings.Contains(got, "\x1b["+ansi[StyleRemoved]+"m") {
		t.Error("a removed line must be red")
	}
}
