package memory

import (
	"fmt"
	"strings"
)

// DefaultMax is how many memories reach the prefix.
//
// A starting value with no observation behind it, and saying so is the point: a
// ceiling has to exist before there is use, or the first session that writes too
// much eats the window and nobody finds out why. What to do with it is measure
// and move it, with a changelog saying what was seen. Fixing a ceiling by
// reasoning and never revisiting it is the mistake EVAL_TIMEOUT made twice.
const DefaultMax = 40

// Render turns what was learned into the block the prefix carries, or "" when
// there is nothing to say.
//
// known is the set of commits the repository still has; a memory whose commit is
// not in it is MARKED, never dropped. The heuristic will be wrong sometimes, and
// a wrong heuristic that deletes is knowledge lost with nobody told. An empty
// set means nothing could be checked, and then nothing is marked — "we did not
// look" must not read as "we looked and it is gone".
func Render(f File, max int, known map[string]bool) string {
	if max <= 0 {
		max = DefaultMax
	}
	if len(f.Entries) == 0 && len(f.Malformed) == 0 {
		return ""
	}

	// The most recent first, and the cut falls on the oldest: a memory written
	// last week is likelier to still be true than one from a year ago.
	entries := f.Entries
	cut := 0
	if len(entries) > max {
		cut = len(entries) - max
		entries = entries[cut:]
	}

	var b strings.Builder
	b.WriteString("What earlier sessions in this repository learned. " +
		"You noted these yourself, so weigh them below anything written by a person, " +
		"and say so when you act on one.\n")

	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		fmt.Fprintf(&b, "\n- **%s** — %s", e.Kind, e.Subject)
		if e.Commit != "" && len(known) > 0 && !known[e.Commit] {
			// It was true at a commit this repository no longer has. The model
			// reads that and weighs it; nothing decides for it.
			b.WriteString(" _(from a commit no longer in this repository)_")
		}
		if e.Body != "" {
			b.WriteString("\n  " + strings.ReplaceAll(e.Body, "\n", "\n  "))
		}
	}

	if cut > 0 {
		fmt.Fprintf(&b, "\n\n%d older memor%s not shown.", cut, plural(cut))
	}
	if len(f.Malformed) > 0 {
		fmt.Fprintf(&b, "\n\n%d block%s in the memory file could not be read.",
			len(f.Malformed), plurals(len(f.Malformed)))
	}
	return b.String()
}

func plural(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

func plurals(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
