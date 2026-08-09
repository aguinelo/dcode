package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// longStream builds a model with more output than any screen holds.
func longStream(n int) Model {
	m := NewModel("s1", "/w", "m", "read-only")
	for i := 0; i < n; i++ {
		m.Entries = append(m.Entries, Entry{
			Kind: KindNote, Summary: fmt.Sprintf("linha %d", i),
		})
	}
	return m
}

// Following means the newest line is always the last line. It is the default
// because a live log that has to be chased is not a log anyone reads.
func TestTheStreamFollowsTheNewestOutputByDefault(t *testing.T) {
	g := DefaultGeometry(80, 12)
	m := longStream(100)

	got := Render(m, g)
	if !strings.Contains(got, "linha 99") {
		t.Errorf("the newest line must be on screen:\n%s", got)
	}
	if strings.Contains(got, "linha 0\n") {
		t.Errorf("the oldest line must have scrolled away:\n%s", got)
	}
}

// Reading something while the stream pushes it off the screen is the single
// most irritating thing a live log can do.
func TestScrollingUpStopsFollowingAndComingBackResumesIt(t *testing.T) {
	g := DefaultGeometry(80, 12)
	m := longStream(100)

	up := m.ScrollBy(-20, g)
	if up.Follow {
		t.Fatal("scrolling up must stop the follow")
	}

	// New output arrives while the user is reading: the lines on screen must
	// not move. The hint below them does change, and should — it is what says
	// more has arrived.
	body := func(m Model) []string {
		visible, _, _, _ := Window(m, g, StreamLines(m, g))
		return visible
	}
	before := body(up)
	grown := up
	grown.Entries = append(grown.Entries, Entry{Kind: KindNote, Summary: "recem chegada"})
	after := body(grown)

	if len(before) != len(after) {
		t.Fatalf("the view moved under the reader: %d rows then %d", len(before), len(after))
	}
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("row %d moved under the reader:\n%q\n%q", i, before[i], after[i])
		}
	}
	if !strings.Contains(Render(grown, g), "below") {
		t.Error("but the hint must say more arrived")
	}
	if strings.Contains(Render(grown, g), "recem chegada") {
		t.Error("new output must not yank the reader to the bottom")
	}

	// Going back to the bottom resumes it, because that is what going there
	// means.
	back := grown.ScrollToBottom(g)
	if !back.Follow {
		t.Fatal("returning to the bottom resumes the follow")
	}
	if !strings.Contains(Render(back, g), "recem chegada") {
		t.Error("and the newest line is on screen again")
	}
}

func TestScrollBoundsAreClamped(t *testing.T) {
	g := DefaultGeometry(80, 12)
	m := longStream(100)

	top := m.ScrollBy(-9999, g)
	if top.ScrollTop != 0 {
		t.Errorf("cannot scroll above the first line, got %d", top.ScrollTop)
	}
	if !strings.Contains(Render(top, g), "linha 0") {
		t.Error("the first line must be reachable")
	}

	bottom := top.ScrollBy(9999, g)
	if !bottom.Follow {
		t.Error("scrolling to the end resumes the follow")
	}

	// A stream shorter than the screen has nowhere to go.
	short := longStream(3)
	if got := short.ScrollBy(-5, g); got.ScrollTop != 0 {
		t.Errorf("got %d", got.ScrollTop)
	}
}

func TestScrollToTopAndBottom(t *testing.T) {
	g := DefaultGeometry(80, 12)
	m := longStream(100)

	top := m.ScrollToTop()
	if top.ScrollTop != 0 || top.Follow {
		t.Errorf("got %d, follow %v", top.ScrollTop, top.Follow)
	}
	if !strings.Contains(Render(top, g), "linha 0") {
		t.Error("the top must show the beginning")
	}
}

// A paused view is otherwise indistinguishable from a stalled agent: the screen
// stops changing either way.
func TestAScrolledViewSaysHowMuchIsBelow(t *testing.T) {
	g := DefaultGeometry(80, 12)
	m := longStream(100).ScrollBy(-30, DefaultGeometry(80, 12))

	got := Render(m, g)
	if !strings.Contains(got, "below") {
		t.Errorf("the hint must say there is more:\n%s", got)
	}
	if !strings.Contains(got, "End") {
		t.Errorf("and how to get back to it:\n%s", got)
	}

	// At the bottom there is nothing to say.
	if strings.Contains(Render(longStream(100), g), "below") {
		t.Error("following needs no hint")
	}
}

func TestScrollHintDegradesToASCII(t *testing.T) {
	g := DefaultGeometry(80, 12)
	g.Unicode = false
	m := longStream(100).ScrollBy(-30, g)
	got := Render(m, g)
	if strings.ContainsRune(got, '↓') {
		t.Errorf("the ASCII rendering must not smuggle an arrow in:\n%s", got)
	}
}

// Moving a cursor you cannot see is how a keypress feels broken: something
// happened, and nothing on screen changed.
func TestTheCursorPullsTheViewToIt(t *testing.T) {
	g := DefaultGeometry(80, 12)
	m := longStream(100)
	m.Cursor = 2
	m = m.EnsureCursorVisible(g)

	if !strings.Contains(Render(m, g), "linha 2") {
		t.Errorf("the selected entry must be on screen:\n%s", Render(m, g))
	}

	m.Cursor = 97
	m = m.EnsureCursorVisible(g)
	if !strings.Contains(Render(m, g), "linha 97") {
		t.Errorf("and still is after jumping to the end:\n%s", Render(m, g))
	}

	// No selection, nothing to do.
	none := longStream(100)
	none.Cursor = -1
	if got := none.EnsureCursorVisible(g); got.ScrollTop != none.ScrollTop {
		t.Error("an unselected stream must not be scrolled")
	}
}

func TestPageSizeOverlapsByTwoRows(t *testing.T) {
	// The line you were reading has to still be there to anchor you.
	if got := PageSize(longStream(10), DefaultGeometry(80, 22)); got != BodyHeight(longStream(10), DefaultGeometry(80, 22))-2 {
		t.Errorf("got %d", got)
	}
	// A screen too short to overlap moves one line at a time rather than zero.
	if got := PageSize(longStream(10), DefaultGeometry(80, 4)); got != 1 {
		t.Errorf("got %d", got)
	}
}

// The working line costs a row, so the stream has to know about it or the last
// line is drawn under the input.
func TestTheWorkingLineTakesARowFromTheStream(t *testing.T) {
	g := DefaultGeometry(80, 12)
	idle := longStream(100)
	running := longStream(100)
	running.State = protocol.SessionStateRunning

	if BodyHeight(running, g) != BodyHeight(idle, g)-1 {
		t.Errorf("got %d running, %d idle", BodyHeight(running, g), BodyHeight(idle, g))
	}
	if n := len(lines(Render(running, g))); n > 12 {
		t.Errorf("the render must still fit the height, got %d lines", n)
	}
}

func TestWindowOnADegenerateGeometry(t *testing.T) {
	m := longStream(5)
	// One row of body is still one row, never zero or negative.
	if got := BodyHeight(m, DefaultGeometry(80, 1)); got != 1 {
		t.Errorf("got %d", got)
	}
	if got := MaxScroll(3, 10); got != 0 {
		t.Errorf("got %d", got)
	}
	if !AtBottom(0, 3, 10) {
		t.Error("a stream shorter than the screen is always at the bottom")
	}
}
