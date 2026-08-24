package tui

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

func railModel(entries ...Entry) Model {
	return Model{Lang: En, Cursor: -1, Entries: entries}
}

func railGeometry(w int) Geometry {
	g := DefaultGeometry(w, 20)
	g.Palette = Palette{}
	return g
}

// A turn that touched nothing gets no column. A column of nothing costs the
// stream twenty characters and tells the reader something is missing.
func TestATurnThatTouchedNothingGetsNoSidebar(t *testing.T) {
	g := railGeometry(140)
	if g.ShowRail(false) {
		t.Error("an empty tree still opened the sidebar")
	}
	if g.StreamWidth(false) != 140 {
		t.Errorf("the stream lost width to a sidebar that is not drawn: %d", g.StreamWidth(false))
	}
}

// The side column comes back with a width rule, and this test has now been
// written three ways in one day. That is worth stating rather than quietly
// editing again: the rule went away because the column it governed held a
// repetition of the stream, and it comes back because the column that replaced
// it holds two panes of things that are nowhere else.
//
// What did not change is the measurement that started it. Two thresholds at the
// same hundred cost the conversation 46 columns across one column of terminal
// width; there is one threshold now, and it is the design's own.
func TestTheSideColumnAppearsOnATerminalWideEnoughForIt(t *testing.T) {
	for _, c := range []struct {
		w    int
		want bool
	}{{80, false}, {119, false}, {120, true}, {150, true}} {
		g := railGeometry(c.w)
		if got := g.ShowRail(true); got != c.want {
			t.Errorf("at %d columns the column is %v, want %v", c.w, got, c.want)
		}
	}
}

// And it is two fifths of the terminal, between its floor and its ceiling —
// the design's 57/43 split, clamped.
func TestTheSideColumnIsTwoFifths(t *testing.T) {
	for _, c := range []struct{ w, want int }{
		{120, 48}, {150, 60}, {400, 60}, {60, 28},
	} {
		if got := railGeometry(c.w).railWidth(); got != c.want {
			t.Errorf("at %d columns the column is %d wide, want %d", c.w, got, c.want)
		}
	}
}

// And an explicit choice wins at any width, in BOTH directions. A keypress is
// the user noticing; responsiveness answers the case where they did not.
func TestAnExplicitChoiceWinsAtAnyWidthBothWays(t *testing.T) {
	forced := railGeometry(80)
	forced.RailMode = RailShown
	if !forced.ShowRail(true) {
		t.Error("an explicitly shown sidebar was hidden by the width rule")
	}

	hidden := railGeometry(200)
	hidden.RailMode = RailHidden
	if hidden.ShowRail(true) {
		t.Error("an explicitly hidden sidebar came back on a wide terminal")
	}
}

// The stream pays for the column exactly once, in the function the layout and
// the renderer both read. Two places computing it is the defect; where it shows
// up is only the symptom.
func TestTheStreamPaysForEveryColumnExactlyOnce(t *testing.T) {
	g := railGeometry(140)
	if got, want := g.StreamWidth(true), 140-g.railWidth()-1; got != want {
		t.Errorf("with the column: got %d, want %d", got, want)
	}
	if got := g.StreamWidth(false); got != 140 {
		t.Errorf("without it the stream is not the whole terminal: %d", got)
	}
}

// Movement is never the only clue. In ASCII a running row, a finished one and a
// failed one still differ by character — a pulse cannot be the difference on a
// terminal where nothing pulses.
func TestTheStatesStayApartWithoutUnicode(t *testing.T) {
	gl := railGlyphs(false)
	seen := map[string]string{}
	for name, mark := range map[string]string{
		"folder": gl.folder, "running": gl.running, "done": gl.done, "failed": gl.failed,
	} {
		if other, clash := seen[mark]; clash {
			t.Errorf("%s and %s are both %q; blocked would read as finished", name, other, mark)
		}
		seen[mark] = name
	}
}

// The sidebar emits no escape without colour, like everything else on screen.
func TestTheSidebarEmitsNoEscapeWithoutColour(t *testing.T) {
	m := railModel(
		Entry{Kind: KindTool, Tool: "write", Target: "a/b.go", Added: 12},
		Entry{Kind: KindTool, Tool: "read", Target: "a/c.go", Running: true},
		Entry{Kind: KindTool, Tool: "edit", Target: "d.go", IsError: true},
	)
	for _, line := range renderRail(m, railGeometry(120), 8) {
		if strings.ContainsRune(line, 0x1b) {
			t.Errorf("an escape survived a monochrome palette: %q", line)
		}
	}
}

// No row is wider than the column, whatever the path.
func TestNoSidebarRowOverflowsTheColumn(t *testing.T) {
	g := railGeometry(120)
	m := railModel(Entry{
		Kind: KindTool, Tool: "write", Added: 123456,
		Target: "internal/very/deeply/nested/directory/with/a/long/name/file.go",
	})
	for _, line := range renderRail(m, g, 6) {
		if w := visibleWidth(line); w > g.railWidth() {
			t.Errorf("a row is %d wide in a %d column: %q", w, g.railWidth(), line)
		}
	}
}

// The header says how much was touched even when the column is too narrow for
// the list, which is what makes a collapsed column still worth a glance.
func TestTheSidebarHeaderCountsWhatWasTouched(t *testing.T) {
	m := railModel(
		Entry{Kind: KindTool, Tool: "read", Target: "a/b.go"},
		Entry{Kind: KindTool, Tool: "read", Target: "a/c.go"},
	)
	head := renderRail(m, railGeometry(160), 5)[0]
	if !strings.Contains(head, "2 touched") {
		t.Errorf("the header does not count the files: %q", head)
	}
	if strings.Contains(head, "2 2") {
		t.Errorf("the count is printed twice: %q", head)
	}
}

// -- the conversations of this workspace -------------------------------------

func sessionsModel(open string, titles ...string) Model {
	m := railModel(Entry{Kind: KindTool, Tool: "read", Target: "a.go"})
	m.SessionID = open
	for i, ti := range titles {
		m.Sessions = append(m.Sessions, SessionChoice{ID: string(rune('a' + i)), Title: ti})
	}
	return m
}

// The open conversation is marked by a character, not only by colour. A
// selection shown in colour alone is no selection on a terminal without any —
// the rule copy mode already carries.
func TestTheOpenConversationIsMarkedByACharacter(t *testing.T) {
	m := sessionsModel("b", "first", "second")
	g := railGeometry(140)
	g.Palette = Palette{}
	lines := strings.Join(renderSessionList(m, g), "\n")

	for _, want := range []string{"● second", "  first"} {
		if !strings.Contains(lines, want) {
			t.Errorf("expected %q in:\n%s", want, lines)
		}
	}
}

// Conversations no longer open the column, because they are not in it. This
// test asserted the opposite and is kept, inverted, so the reason the column
// used to open for them leaves a record: it held a second section listing them,
// and that section is now an overlay.
func TestConversationsDoNotOpenTheFileColumn(t *testing.T) {
	m := Model{Lang: En, Cursor: -1, Sessions: []SessionChoice{{ID: "a", Title: "x"}}}
	if m.railHasContent() {
		t.Error("a recorded conversation still opens the file column")
	}
	touched := Model{Lang: En, Cursor: -1,
		Entries: []Entry{{Kind: KindTool, Tool: "write", Target: "a.go"}}}
	if !touched.railHasContent() {
		t.Error("a turn that wrote a file got no column")
	}
}

// A title that had to be cut says so. One that merely stops leaves the reader
// unable to tell a short conversation from a truncated one.
func TestATruncatedTitleSaysItWasTruncated(t *testing.T) {
	m := sessionsModel("z", strings.Repeat("catalogar os contratos ", 4))
	g := railGeometry(120)
	g.Palette = Palette{}
	var row string
	for _, l := range renderSessionList(m, g) {
		if strings.Contains(l, "catalogar") {
			row = l
			break
		}
	}
	if row == "" {
		t.Fatal("the conversation is not listed")
	}
	if !strings.Contains(row, "…") {
		t.Errorf("a cut title does not say it was cut: %q", row)
	}
	// Every row of the overlay is the same width, so a box that a long title
	// pushed out of shape shows up here rather than on somebody's screen.
	box := renderSessionList(m, g)
	for i, l := range box {
		if visibleWidth(l) != visibleWidth(box[0]) {
			t.Errorf("row %d is %d wide, the box is %d: %q",
				i, visibleWidth(l), visibleWidth(box[0]), l)
		}
	}
}

// Cut in cells, never in bytes: a rune is not a column.
func TestATitleIsCutInCellsAndNotInBytes(t *testing.T) {
	if got := trimTo("sessões", 4); visibleWidth(got) > 4 {
		t.Errorf("%q is %d columns, wanted at most 4", got, visibleWidth(got))
	}
	if trimTo("abc", 0) != "" {
		t.Error("a zero width produced text")
	}
}

// -- no box-drawing rune reaches a terminal that cannot draw one -------------

// Four separate literals in this package assumed Unicode: the column divider,
// the diff gutter, the running marker and the path ellipsis. Each was found by
// looking at an ASCII render rather than at the code, and each was found AFTER
// the previous one had been fixed.
//
// So the assertion is over the whole screen rather than over one of them: a
// fifth would otherwise wait for a fifth pair of eyes.
func TestNoBoxDrawingRuneSurvivesAsciiMode(t *testing.T) {
	m := railModel(
		Entry{Kind: KindTool, Tool: "read", Target: "docs/ROADMAP.md", Running: true},
		Entry{Kind: KindTool, Tool: "edit", Target: "internal/tui/rail.go", Added: 38,
			Diff: "--- a/x\n+++ b/x\n@@ -1,2 +1,3 @@\n+added\n"},
		Entry{Kind: KindTool, Tool: "read",
			Target: "internal/very/deeply/nested/path/that/must/be/shortened.go"},
	)
	m.Sessions = []SessionChoice{{ID: "s1", Title: "uma conversa"}}
	m.SessionID = "s1"

	g := DefaultGeometry(118, 18)
	g.Palette = Palette{}
	g.Unicode = false
	g.RailMode = RailShown

	// The forbidden set is DERIVED from the Unicode glyph sets rather than
	// listed here. Listing them is what let the filter caret slip in as the
	// fifth of these: a new glyph would have to be remembered in two places,
	// and the second place is a test nobody edits when adding a mark.
	m.Nav = RailNav{Active: true, Filter: "co"}
	out := Render(m, g)
	for _, forbidden := range unicodeGlyphs() {
		if forbidden == "" || forbidden == " " {
			continue
		}
		if strings.Contains(out, forbidden) {
			t.Errorf("%q reached a terminal that declared it cannot draw one:\n%s",
				forbidden, out)
		}
	}
	// The two the layout reaches for directly, outside either set.
	for _, forbidden := range []string{"⋯", "│"} {
		if strings.Contains(out, forbidden) {
			t.Errorf("%q reached a terminal that declared it cannot draw one:\n%s",
				forbidden, out)
		}
	}
}

// unicodeGlyphs is every mark either set draws when the terminal can.
func unicodeGlyphs() []string {
	var out []string
	for _, v := range []any{glyphs(true), railGlyphs(true)} {
		rv := reflect.ValueOf(v)
		for i := 0; i < rv.NumField(); i++ {
			out = append(out, rv.Field(i).String())
		}
	}
	return out
}

// -- a column that hides itself says so --------------------------------------

// The sidebar disappears below a hundred columns, which is most terminals, and
// it said nothing at all. A column that had been built read as a column that
// had not — and the key that brings it back was documented only inside the
// column that was not on screen.
//
// The plan panel already carried this debt and already paid it; the sidebar
// inherited the behaviour and not the hint.
func TestASidebarHiddenByWidthSaysSoAndNamesTheKey(t *testing.T) {
	m := railModel(Entry{Kind: KindTool, Tool: "read", Target: "a.go"})
	g := railGeometry(80)

	if g.ShowRail(m.railHasContent()) {
		t.Fatal("the fixture is not narrow enough to hide the column")
	}
	status := renderStatus(m, g, false)
	if !strings.Contains(status, "^b") {
		t.Errorf("a hidden column does not name the key that brings it back: %q", status)
	}
}

// And it stops saying so once it is on screen: a hint for something already
// visible is noise, and noise is what teaches people to stop reading the line.
func TestAVisibleSidebarSaysNothingAboutItself(t *testing.T) {
	m := railModel(Entry{Kind: KindTool, Tool: "read", Target: "a.go"})
	g := railGeometry(140)
	g.RailMode = RailShown

	if !g.ShowRail(m.railHasContent()) {
		t.Fatal("the fixture should show the column")
	}
	if status := renderStatus(m, g, false); strings.Contains(status, "^b") {
		t.Errorf("a visible column advertises itself: %q", status)
	}
}

// Nothing to show is not something to advertise. Offering a key for an empty
// column sends somebody to look at nothing.
func TestAnEmptySidebarIsNotAdvertised(t *testing.T) {
	m := Model{Lang: En, Cursor: -1}
	if status := renderStatus(m, railGeometry(80), false); strings.Contains(status, "^b") {
		t.Errorf("an empty column was advertised: %q", status)
	}
}

// The box-drawing guard above asks whether a known set of glyphs got through.
// That set is derived from the two glyph tables, so it only ever covers what
// those tables know about — and the approval modal, which is drawn entirely
// from literals and never appeared in a model this test built, was outside it
// from the day it was written.
//
// This asks the decisive question instead: in ASCII mode, is every rune ASCII?
// The model is built entirely from ASCII so that any rune above 127 in the
// output came from the layout drawing it, and it is checked with the modal
// open, which is the one screen where being unreadable is most expensive.
func TestAsciiModeDrawsNothingButAscii(t *testing.T) {
	m := railModel(
		Entry{Kind: KindTool, Tool: "read", Target: "docs/ROADMAP.md", Running: true},
		Entry{Kind: KindTool, Tool: "edit", Target: "internal/tui/rail.go", Added: 38, Removed: 2,
			Diff: "--- a/x\n+++ b/x\n@@ -1,2 +1,3 @@\n+added\n", Expanded: true},
		Entry{Kind: KindReasoning, Summary: "a thought", Expanded: true},
		Entry{Kind: KindAssistant, Summary: "a paragraph with `code` and a - list item"},
		Entry{Kind: KindTool, Tool: "read",
			Target: "internal/very/deeply/nested/path/that/must/be/shortened.go"},
	)
	m.Sessions = []SessionChoice{{ID: "s1", Title: "a conversation with a title too long for any column"}}
	m.SessionID = "s1"
	m.DiffAdded, m.DiffRemoved, m.DiffFiles = 38, 2, 3
	m.Rounds, m.MaxRounds, m.InFlight, m.MaxInFlight = 16, 20, 1, 4
	m.Plan = []protocol.PlanItem{
		{ID: 1, Text: "a plan item longer than the panel can hold", Status: protocol.PlanBlocked,
			Blocked: "the reason it is blocked, at length"},
	}

	for _, name := range []string{"stream", "modal"} {
		m := m
		if name == "modal" {
			m.Pending = &protocol.ApprovalRequest{
				ApprovalID: "a1", Tool: "bash", Command: "rm -rf /tmp/x",
				BoundaryCrossed: "the workspace", Reason: "it writes outside", Rule: "deny:rm",
			}
		}
		g := DefaultGeometry(118, 24)
		g.Palette = Palette{}
		g.Unicode = false
		g.RailMode = RailShown

		for i, line := range lines(Render(m, g)) {
			for _, r := range line {
				if r > 127 {
					t.Errorf("%s: %q on line %d reached a terminal that declared ASCII:\n%s",
						name, r, i, line)
					break
				}
			}
		}
	}
}

// Four rows read "Write DCODE.md at the root of this workspace." because four
// conversations genuinely began with the same question. When the titles
// collide, WHEN is the only thing that tells them apart — so the meta takes its
// width first and the title gives way to it, which is the opposite of the rule
// the file rows follow, and deliberately so.
func TestConversationRowsSayWhenAndHowMuch(t *testing.T) {
	now := time.Date(2026, 8, 24, 15, 30, 0, 0, time.UTC)
	m := Model{Lang: En, Cursor: -1, Now: now, Nav: RailNav{Active: true}}
	const same = "Write DCODE.md at the root of this workspace."
	for i, ago := range []time.Duration{2 * time.Hour, 26 * time.Hour, 50 * time.Hour} {
		m.Sessions = append(m.Sessions, SessionChoice{
			ID: string(rune('a' + i)), Title: same, Turns: 3 + i, When: now.Add(-ago),
		})
	}
	g := DefaultGeometry(100, 30)
	g.Palette = Palette{}

	rows := renderSessionList(m, g)
	seen := map[string]bool{}
	for _, l := range rows {
		if !strings.Contains(l, "Write DCODE") {
			continue
		}
		if seen[l] {
			t.Errorf("two conversations draw the same row: %q", l)
		}
		seen[l] = true
	}
	if len(seen) != 3 {
		t.Fatalf("expected 3 distinct rows, got %d:\n%s", len(seen), strings.Join(rows, "\n"))
	}

	joined := strings.Join(rows, "\n")
	for _, want := range []string{"yesterday", "3 turns"} {
		if !strings.Contains(joined, want) {
			t.Errorf("%q is missing from:\n%s", want, joined)
		}
	}
	// The count is a real plural, not the "%d turn(s)" nobody ever replaces.
	if strings.Contains(joined, "(s)") {
		t.Errorf("the turn count still carries a parenthetical plural:\n%s", joined)
	}
}

// The clock is an argument, so two draws of the same model are the same screen.
// The picker could read one directly; the overlay is inside the main render,
// which is pure over the model and the geometry.
func TestTheConversationListReadsNoClock(t *testing.T) {
	m := Model{Lang: En, Cursor: -1, Now: time.Date(2026, 8, 24, 15, 30, 0, 0, time.UTC),
		Nav:      RailNav{Active: true},
		Sessions: []SessionChoice{{ID: "a", Title: "x", Turns: 2, When: time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC)}}}
	g := DefaultGeometry(100, 30)

	first := strings.Join(renderSessionList(m, g), "\n")
	for i := 0; i < 20; i++ {
		if got := strings.Join(renderSessionList(m, g), "\n"); got != first {
			t.Fatalf("draw %d differs; the list reads something outside its arguments", i)
		}
	}
}

// The side column holds two panes, and neither repeats the stream.
//
// That is the whole difference between it and the file list it replaces. The
// list said `+188`; this says what KIND of change it was, how much room is
// left, how much of what was asked the person allowed, and what the model has
// been doing by the clock — none of which is anywhere else on the screen.
func TestTheSideColumnSaysWhatIsNowhereElse(t *testing.T) {
	m := railModel(
		Entry{Kind: KindTool, Tool: "edit", Target: "internal/tui/render.go", Added: 42, Removed: 6},
		Entry{Kind: KindTool, Tool: "write", Target: "internal/tui/side.go", Added: 188},
	)
	m.DiffAdded, m.DiffRemoved = 230, 6
	m.InputTokens, m.Window, m.ContextPct = 18200, 200000, 9
	m.Asked, m.Allowed = 7, 6

	g := railGeometry(150)
	g.RailMode = RailShown
	out := strings.Join(renderSide(m, g, 24), "\n")

	for _, want := range []string{
		"DIFF", "render.go", "+42", "SESSION", "18.2k", "200.0k", "6 / 7", "RECENT",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("%q is missing from the column:\n%s", want, out)
		}
	}
	// Every row fits, and the column is exactly the height it was given.
	rows := renderSide(m, g, 24)
	if len(rows) != 24 {
		t.Errorf("the column drew %d rows for 24", len(rows))
	}
	for i, l := range rows {
		if w := visibleWidth(l); w > g.railWidth() {
			t.Errorf("row %d is %d cells in a %d column: %q", i, w, g.railWidth(), l)
		}
	}
}

// The bars are scaled to the turn's largest change, and the pane says so.
//
// The first version used each change as its own denominator, so every file with
// no removals drew a full-width bar: a row of identical bars, each saying "100%
// of what I did to this file, I did to this file". The design's denominator is
// the file's length, which a tool never reports.
func TestTheBarsAreScaledToTheLargestChangeAndSaySo(t *testing.T) {
	g := railGeometry(150)
	gl := glyphs(g.Unicode)
	p := Palette{}

	big := bar(200, 0, 200, gl, p, 20)
	small := bar(20, 0, 200, gl, p, 20)
	if strings.Count(big, gl.barFull) <= strings.Count(small, gl.barFull) {
		t.Errorf("a ten-times-larger change did not draw a longer bar: %q vs %q", big, small)
	}
	if !strings.Contains(small, gl.barTrack) {
		t.Error("a small change fills the whole bar; the scale is not being applied")
	}

	m := railModel(Entry{Kind: KindTool, Tool: "edit", Target: "a.go", Added: 200})
	if out := strings.Join(renderDiffPane(m, g, g.railWidth(), 10), "\n"); !strings.Contains(out, "200") {
		t.Errorf("the pane does not say what the bars are scaled to:\n%s", out)
	}
}
