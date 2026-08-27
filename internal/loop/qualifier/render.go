package qualifier

import (
	"fmt"
	"strconv"
	"strings"
)

// Render writes the file a person reviews.
//
// The measurement goes in as comments beside each criterion, because the file
// IS the review surface: it is diffable, it survives the session that produced
// it, and it is what the next run reads. A number that only ever appeared on a
// screen is a number nobody can go back to.
//
// Being what the next run reads is also why a broken criterion is written
// commented out rather than declared: see the note at the loop below.
func Render(measured []Measured, protected []string, cond Conditions) []byte {
	var b strings.Builder
	b.WriteString("# Proposed by dcode and measured before any work.\n")
	b.WriteString("# Review it: a criterion is only worth having if failing it would stop you.\n")
	if cond.NoAcceptance {
		b.WriteString("#\n# NOTHING HERE IS RED. Every criterion already passes, so this spec\n")
		b.WriteString("# reports done without anything having to change. That is right for a\n")
		b.WriteString("# refactor and wrong for everything else.\n")
	}
	if len(protected) > 0 {
		fmt.Fprintf(&b, "\nprotected = %q\n", strings.Join(protected, ", "))
	}
	for _, m := range measured {
		b.WriteString("\n")
		if m.Why != "" {
			fmt.Fprintf(&b, "# %s\n", m.Why)
		}
		fmt.Fprintf(&b, "# now: %s (exit %d)", m.Class, m.Exit)
		if m.Mismatch {
			fmt.Fprintf(&b, " — proposed as %q and it did the opposite", m.Expects)
		}
		if m.Class == ClassBroken {
			b.WriteString(" — nothing ran; this measures the absence of a tool, not of work.\n" +
				"# It is commented out: as written it can never go green, and a loop " +
				"measured against it\n# would run forever. Fix the command and it counts again.")
		}
		// A broken criterion is written down and NOT declared. The file is
		// what the next run loads, so leaving it live handed the work session
		// a command that does not exist — red forever, with the loop unable to
		// finish and unable to qualify again, because the folder now declared
		// a criterion. Commented, it stays visible to the person, stays out of
		// the DoneSet, and leaves the folder pending with nothing declared,
		// which is what sends it back through qualification.
		prefix := ""
		if m.Class == ClassBroken {
			prefix = "# "
		}
		fmt.Fprintf(&b, "\n%s[%s]\n%scommand = %q\n", prefix, m.Name, prefix, m.Command)
		if m.ExitCode != 0 {
			fmt.Fprintf(&b, "%sexit_code = %q\n", prefix, strconv.Itoa(m.ExitCode))
		}
	}
	return []byte(b.String())
}

// Summary is what the loop prints once the proposal has been measured.
//
// It goes to the person, not to the model: the measurement happens after the
// qualifying turn has ended, so there is nobody left in the turn to correct
// anything. What it is for is the review — the same numbers as the file, on
// screen, so the person knows what they are about to read.
func Summary(measured []Measured, cond Conditions, path string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "wrote %s and ran every criterion against the repository as it stands:\n", path)
	for _, m := range measured {
		fmt.Fprintf(&b, "\n  %-24s %s (exit %d)", m.Name, m.Class, m.Exit)
		if m.Mismatch {
			b.WriteString(" ← proposed as " + string(m.Expects))
		}
		if m.Class == ClassBroken {
			b.WriteString(" — commented out, nothing will be measured against it")
		}
	}
	if cond.NoAcceptance {
		b.WriteString("\n\nNothing here is red. This set reports done with no work done, " +
			"which is right for a refactor and wrong for everything else.")
	}
	b.WriteString("\n\nNothing runs against this until it has been read. " +
		"A broken or contradicted criterion is where to look first.")
	return b.String()
}
