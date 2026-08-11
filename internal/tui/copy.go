package tui

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// Copy mode is the debt RN-1 took on knowingly.
//
// The alternate screen buys a stable layout and costs the terminal's own
// scrollback and mouse selection. The spec calls that "custo aceito e
// explícito" and says both have to be reimplemented — scrolling was, and this
// was not. It is the kind of absence a user meets daily and reports as "I
// cannot copy the error", which reads as a bug rather than as a decision.

// CopyState is the selection while copy mode is open.
//
// Two line indices into the rendered stream rather than a start and a length,
// because the selection is dragged in both directions and an anchor that moves
// is how a selection ends up off by one at one end.
type CopyState struct {
	Active bool
	Anchor int
	Head   int
}

// Range returns the selected lines, low first.
func (c CopyState) Range() (int, int) {
	if c.Anchor <= c.Head {
		return c.Anchor, c.Head
	}
	return c.Head, c.Anchor
}

// Contains reports whether a rendered line is inside the selection.
func (c CopyState) Contains(i int) bool {
	if !c.Active {
		return false
	}
	lo, hi := c.Range()
	return i >= lo && i <= hi
}

// CopyText joins the selected lines.
//
// Undecorated: what goes to the clipboard is what the person meant to copy, not
// the cursor marks and box drawing around it. Pasting a diff with a gutter of
// ANSI escapes into an issue is the failure this avoids.
func CopyText(lines []string, c CopyState) string {
	if !c.Active || len(lines) == 0 {
		return ""
	}
	lo, hi := c.Range()
	if lo < 0 {
		lo = 0
	}
	if hi >= len(lines) {
		hi = len(lines) - 1
	}
	if lo > hi {
		return ""
	}
	out := make([]string, 0, hi-lo+1)
	for _, l := range lines[lo : hi+1] {
		out = append(out, strings.TrimRight(stripANSI(l), " "))
	}
	return strings.Join(out, "\n")
}

// OSC52 is the escape sequence that puts text on the system clipboard.
//
// It works over ssh and inside tmux, which is the whole reason to use it rather
// than shelling out to pbcopy or xclip: the terminal the person is looking at
// is not always on the machine dcode is running on, and a clipboard that only
// works locally is one that fails exactly when it is most wanted.
//
// A terminal that does not support it ignores the sequence. That is a silent
// failure, so the client says what it did rather than assuming it worked.
func OSC52(s string) string {
	return "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(s)) + "\x07"
}

// CopyHint is the line shown while copy mode is open.
//
// It names every key, because a mode with no visible way out is a mode people
// force-quit the program to escape.
func CopyHint(c CopyState, lang Lang) string {
	lo, hi := c.Range()
	n := hi - lo + 1
	t := Text(lang)
	return fmt.Sprintf("%s — %s", fmt.Sprintf(t.CopySelected, n), t.CopyKeys)
}

// EnterCopy opens copy mode anchored on the cursor, or on the last line.
func (m Model) EnterCopy(lastLine int) Model {
	at := m.Cursor
	if at < 0 {
		at = lastLine
	}
	if at < 0 {
		at = 0
	}
	m.Copy = CopyState{Active: true, Anchor: at, Head: at}
	m.Flash = ""
	return m
}

// LeaveCopy closes it.
func (m Model) LeaveCopy() Model {
	m.Copy = CopyState{}
	return m
}

// ExtendCopy moves the head of the selection, keeping the anchor.
//
// The anchor stays put so dragging back past the start shrinks the selection
// rather than inverting it — an anchor that moves is how a selection ends up
// off by one at one end.
func (m Model) ExtendCopy(delta, lastLine int) Model {
	if !m.Copy.Active {
		return m
	}
	h := m.Copy.Head + delta
	if h < 0 {
		h = 0
	}
	if lastLine >= 0 && h > lastLine {
		h = lastLine
	}
	m.Copy.Head = h
	return m
}
