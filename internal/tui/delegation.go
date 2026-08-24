package tui

import (
	"fmt"
	"strings"
)

// Delegation on screen: one header, and the children under it.
//
// A delegated turn is several `explore` calls made together, and drawn as
// separate lines they read as unrelated work that happened to be adjacent. The
// thing a person needs to see is that ONE decision produced them, that each was
// given a boundary, and — above all — which of them did not come back.
//
// `delegated-writing` promises that a child which never answered is named with
// its reason, never summarised in with the ones that answered. That promise is
// kept in the loop and was, until now, invisible: the client drew a failed
// explore exactly like any other failed call, in a row of siblings, where it
// read as one line of six rather than as the one line that matters.

// delegationRun is how many adjacent calls at i belong to one delegation, or 0.
//
// Adjacency is the signal because the events arrive in emission order and a
// delegated batch is emitted together. Two separate delegations in one turn
// would read as one, and that is the right way to be wrong: they were still one
// decision to divide the work, and the children still carry their own names.
func delegationRun(entries []Entry, i int) int {
	n := 0
	for ; i+n < len(entries); n++ {
		e := entries[i+n]
		if e.Kind != KindTool || !strings.EqualFold(e.Tool, "explore") {
			break
		}
	}
	// One child is a call, not a delegation. Drawing a card around it would put
	// a frame and a count around something the ordinary line already says.
	if n < 2 {
		return 0
	}
	return n
}

// renderDelegation draws the header and one sub-line per child.
func renderDelegation(children []Entry, gl marks, p Palette, w int, t Strings) []string {
	done, failed, running := 0, 0, 0
	for _, c := range children {
		switch {
		case c.Running:
			running++
		case c.IsError:
			failed++
		default:
			done++
		}
	}

	head := fmt.Sprintf("  %s %-*s %s",
		p.Apply(StyleAccent, gl.bullet), toolNameWidth, "explore",
		p.Apply(StyleMeta, plural(len(children), t.ChildOne, t.ChildMany)))
	if failed > 0 {
		// Counted in the header because it is the reason to look, and a person
		// scrolling past should not have to read every sub-line to learn that
		// one of them is missing.
		head += p.Apply(StyleError, fmt.Sprintf("  %d %s", failed, t.ChildNoAnswer))
	}
	out := []string{clipStyled(head, w)}

	for _, c := range children {
		out = append(out, clipStyled(delegationChild(c, gl, p, w, t), w))
	}
	return out
}

func delegationChild(e Entry, gl marks, p Palette, w int, t Strings) string {
	mark, style := gl.active, StyleAccent
	switch {
	case e.Running:
	case e.IsError:
		mark, style = gl.blocked, StyleError
	default:
		mark, style = gl.done, StyleOK
	}

	name := e.Target
	if name == "" {
		name = t.ChildUnnamed
	}

	// The boundary the child was given. Shown whether or not it answered: a
	// child that never came back still declared what it would write, and that
	// is half of why its refusal is legible.
	owns := ""
	if len(e.Owns) > 0 {
		owns = t.ChildOwns + " " + strings.Join(e.Owns, " ")
	}

	// The failure says what happened, in place, on the child's own line. Never
	// folded in with the ones that answered — that is the guarantee this whole
	// block exists to make visible.
	result := e.Summary
	if e.Running {
		result = runningMeta(e, gl)
	}
	if e.IsError && result == "" {
		result = t.ChildNoAnswer
	}

	line := fmt.Sprintf("    %s %s", p.Apply(style, mark), name)
	if owns != "" {
		line += "  " + p.Apply(StyleMeta, owns)
	}
	if result != "" {
		line += "  " + p.Apply(styleFor(e), result)
	}
	return line
}

func styleFor(e Entry) Style {
	if e.IsError {
		return StyleError
	}
	return StyleMeta
}
