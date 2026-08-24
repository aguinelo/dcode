package tui

import (
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/mattn/go-runewidth"
)

// The stream is taller than the screen almost immediately, so what the user
// sees is a window onto it. Everything here is pure over the model and the
// geometry: the same state always produces the same window, which is what lets
// scrolling be tested without a terminal.

// BodyHeight is how many rows the stream gets: everything but the status bar,
// the input line, and the working line when a turn is running.
func BodyHeight(m Model, g Geometry) int {
	// Status line, the input box, and the bottom bar. The bar is not optional:
	// it is the region that is true regardless of what the stream shows.
	//
	// The box's height comes from InputHeight rather than being assumed to be
	// one, and InputHeight counts its two rules. It used to be folded into this
	// literal, so a box that grew painted over the stream.
	h := g.Height - 2 - InputHeight(m, g)
	if m.workingVisible() {
		h--
	}
	// The queue and the completion menu take their rows from the stream, or the
	// last lines of output are drawn underneath the input.
	h -= len(renderQueue(m, g))
	h -= len(renderCompletions(m, g))
	if h < 1 {
		h = 1
	}
	return h
}

// workingVisible reports whether the working line is drawn. It costs a row, so
// the stream has to know about it.
func (m Model) workingVisible() bool {
	return m.State == protocol.SessionStateRunning && !m.ShowEmptyState()
}

// MaxScroll is the furthest down the window can go.
func MaxScroll(total, height int) int {
	if total <= height {
		return 0
	}
	return total - height
}

// Window returns the visible slice of the stream and where it sits.
//
// The scroll position is clamped here rather than trusted, because the content
// grows underneath it: a position that was valid one event ago can be past the
// end now, and a client that trusted it would render blank.
func Window(m Model, g Geometry, body []string) (visible []string, top, total, height int) {
	height = BodyHeight(m, g)
	total = len(body)
	max := MaxScroll(total, height)

	top = m.ScrollTop
	if m.Follow {
		// Following means the newest line is the last line, always.
		top = max
	}
	if top > max {
		top = max
	}
	if top < 0 {
		top = 0
	}

	end := top + height
	if end > total {
		end = total
	}
	if top > total {
		top = total
	}
	return body[top:end], top, total, height
}

// AtBottom reports whether the window is showing the end of the stream.
func AtBottom(top, total, height int) bool { return top >= MaxScroll(total, height) }

// ScrollBy moves the window and decides whether to keep following.
//
// Scrolling up stops the follow: reading something while the stream pushes it
// off the screen is the single most irritating thing a live log can do.
// Arriving back at the bottom resumes it, because that is what the user just
// asked for by going there.
func (m Model) ScrollBy(lines int, g Geometry) Model {
	body := StreamLines(m, g)
	height := BodyHeight(m, g)
	max := MaxScroll(len(body), height)

	top := m.ScrollTop
	if m.Follow {
		top = max
	}
	top += lines

	if top < 0 {
		top = 0
	}
	if top > max {
		top = max
	}
	m.ScrollTop = top
	m.Follow = top >= max
	return m
}

// ScrollToTop jumps to the beginning of the session.
func (m Model) ScrollToTop() Model {
	m.ScrollTop = 0
	m.Follow = false
	return m
}

// ScrollToBottom jumps to the newest output and resumes following.
func (m Model) ScrollToBottom(g Geometry) Model {
	m.ScrollTop = MaxScroll(len(StreamLines(m, g)), BodyHeight(m, g))
	m.Follow = true
	return m
}

// PageSize is how far a page key moves: one screen less two rows of overlap,
// so the line you were reading is still there to anchor you.
func PageSize(m Model, g Geometry) int {
	h := BodyHeight(m, g)
	if h <= 3 {
		return 1
	}
	return h - 2
}

// EnsureCursorVisible scrolls the window so the selected entry is on screen.
//
// Moving a cursor you cannot see is how a keypress feels broken: something
// happened, and nothing on screen changed.
func (m Model) EnsureCursorVisible(g Geometry) Model {
	if m.Cursor < 0 || m.Cursor >= len(m.Entries) {
		return m
	}
	body := StreamLines(m, g)
	height := BodyHeight(m, g)
	start, end := entryRows(m, g, m.Cursor)
	if start < 0 {
		return m
	}

	top := m.ScrollTop
	if m.Follow {
		top = MaxScroll(len(body), height)
	}
	switch {
	case start < top:
		top = start
	case end >= top+height:
		top = end - height + 1
	default:
		return m
	}
	if top < 0 {
		top = 0
	}
	m.ScrollTop = top
	m.Follow = top >= MaxScroll(len(body), height)
	return m
}

// entryRows reports the first and last row an entry occupies in the rendered
// stream. It renders the prefix rather than guessing a height, because an
// expanded entry's height depends on wrapping and on the diff cap.
func entryRows(m Model, g Geometry, index int) (start, end int) {
	if index < 0 || index >= len(m.Entries) {
		return -1, -1
	}
	before := m
	before.Entries = m.Entries[:index]
	start = len(StreamLines(before, g))

	through := m
	through.Entries = m.Entries[:index+1]
	end = len(StreamLines(through, g)) - 1
	if end < start {
		end = start
	}
	return start, end
}

// ScrollHint is the one-line note that the newest output is off screen.
//
// Without it, a paused view is indistinguishable from a stalled agent: the
// screen stops changing either way.
func ScrollHint(m Model, g Geometry, top, total, height int) string {
	if AtBottom(top, total, height) {
		return ""
	}
	behind := MaxScroll(total, height) - top
	mark := "↓"
	if !g.Unicode {
		mark = "v"
	}
	return mark + " " + plural(behind, "line below", "lines below") + " · End to follow"
}

// clipWidth is a small helper shared by the renderers.
func clipWidth(s string) int { return runewidth.StringWidth(s) }
