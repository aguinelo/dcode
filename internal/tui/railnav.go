package tui

import "strings"

// Navigating the session list, which is the design's second rail mode.
//
// While it is open the rail OWNS THE KEYBOARD, the way copy mode and the
// approval modal do. That is not a flourish: a list you move through with keys
// that also type into the input line is a list where every keystroke does two
// things, and the one time it does the wrong one it opens somebody else's
// afternoon.

// RailNav is where the cursor is in the session list, and what has been typed
// to narrow it.
type RailNav struct {
	// Active says the rail has the keyboard.
	Active bool
	// Cursor is an index into the FILTERED list, never into the whole one.
	// Keeping it against the filtered view is what stops a narrowing filter
	// from leaving the cursor pointing past the end of what is on screen.
	Cursor int
	Filter string
	// Naming says the row under the cursor is being given a name, and Draft is
	// what has been typed. A separate mode rather than a second meaning for
	// the filter: one takes you to a conversation and the other changes it, and
	// a key that sometimes does each is a key nobody trusts.
	Naming bool
	Draft  string
}

// Matches reports whether a conversation survives the filter.
//
// Case-insensitive on the title, because somebody typing a filter is
// remembering a phrase, not reproducing one.
func (n RailNav) Matches(c SessionChoice) bool {
	if n.Filter == "" {
		return true
	}
	return strings.Contains(strings.ToLower(c.Title), strings.ToLower(n.Filter))
}

// Visible is the list as the filter leaves it.
func (n RailNav) Visible(all []SessionChoice) []SessionChoice {
	if n.Filter == "" {
		return all
	}
	out := make([]SessionChoice, 0, len(all))
	for _, c := range all {
		if n.Matches(c) {
			out = append(out, c)
		}
	}
	return out
}

// Move walks the cursor and stops at both ends.
//
// It does not wrap, and the reason is the one the picker already writes down:
// wrapping turns "one too far" into "somewhere else entirely", and this is a
// list where landing somewhere else opens the wrong afternoon's work.
func (n RailNav) Move(d, n_ int) RailNav {
	if n_ == 0 {
		n.Cursor = 0
		return n
	}
	n.Cursor += d
	if n.Cursor < 0 {
		n.Cursor = 0
	}
	if n.Cursor > n_-1 {
		n.Cursor = n_ - 1
	}
	return n
}

// Chosen is the conversation under the cursor, empty when the filter left
// nothing. Empty has to be distinguishable from the first row, or a filter that
// matched nothing would open the newest conversation instead of doing nothing.
func (n RailNav) Chosen(all []SessionChoice) string {
	v := n.Visible(all)
	if len(v) == 0 || n.Cursor < 0 || n.Cursor >= len(v) {
		return ""
	}
	return v[n.Cursor].ID
}

// TypeName edits the draft.
func (n RailNav) TypeName(r string) RailNav {
	if len([]rune(n.Draft)) >= NameLimit {
		return n
	}
	n.Draft += r
	return n
}

// BackspaceName drops a rune from the draft.
func (n RailNav) BackspaceName() RailNav {
	if n.Draft == "" {
		return n
	}
	r := []rune(n.Draft)
	n.Draft = string(r[:len(r)-1])
	return n
}

// NameLimit is what the daemon accepts, repeated here so the keyboard stops at
// the same place the server would refuse. A client that lets somebody type past
// the limit and then reports a failure has wasted the typing.
const NameLimit = 120

// Type narrows the filter and pulls the cursor back into range.
//
// Back to the top rather than to the nearest surviving row: after typing, what
// the person is looking at is a different list, and holding a position in it
// would be holding a position in something they have not read.
func (n RailNav) Type(r string, all []SessionChoice) RailNav {
	n.Filter += r
	n.Cursor = 0
	return n
}

// Backspace widens it, and drops the last rune rather than the last byte.
func (n RailNav) Backspace() RailNav {
	if n.Filter == "" {
		return n
	}
	r := []rune(n.Filter)
	n.Filter = string(r[:len(r)-1])
	n.Cursor = 0
	return n
}

// StartNaming opens the name of the row under the cursor for editing, seeded
// with the name it already has.
//
// Seeded with the NAME and not the derived title: offering the title as a draft
// would turn "give this a name" into "confirm the one you were given", and the
// first Enter would quietly promote a derived title into a chosen one.
func (n RailNav) StartNaming(all []SessionChoice) RailNav {
	v := n.Visible(all)
	if len(v) == 0 || n.Cursor < 0 || n.Cursor >= len(v) {
		return n
	}
	n.Naming, n.Draft = true, v[n.Cursor].Name
	return n
}

// Escape backs out of one thing at a time: the filter first, then the mode.
//
// The same layering `esc` already has everywhere else here — close the
// expansion, then the selection, then the modal. Escape means "back out of what
// I opened", and the outermost thing opened is the last thing it abandons.
func (n RailNav) Escape() RailNav {
	if n.Naming {
		// The derived title is kept, which is the design's own rule: backing
		// out of naming leaves what was there rather than clearing it.
		n.Naming, n.Draft = false, ""
		return n
	}
	if n.Filter != "" {
		n.Filter, n.Cursor = "", 0
		return n
	}
	return RailNav{}
}
