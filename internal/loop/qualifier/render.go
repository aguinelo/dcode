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
			b.WriteString(" — nothing ran; this measures the absence of a tool, not of work")
		}
		fmt.Fprintf(&b, "\n[%s]\ncommand = %q\n", m.Name, m.Command)
		if m.ExitCode != 0 {
			fmt.Fprintf(&b, "exit_code = %q\n", strconv.Itoa(m.ExitCode))
		}
	}
	return []byte(b.String())
}

// Summary is what the model reads back after proposing.
//
// It carries the measurement so a broken or contradicted criterion is
// corrected by whoever wrote it, rather than by the person reviewing.
func Summary(measured []Measured, cond Conditions, path string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "wrote %s and ran every criterion against the repository as it stands:\n", path)
	for _, m := range measured {
		fmt.Fprintf(&b, "\n  %-24s %s (exit %d)", m.Name, m.Class, m.Exit)
		if m.Mismatch {
			b.WriteString(" ← you said it would " + string(m.Expects))
		}
	}
	if cond.NoAcceptance {
		b.WriteString("\n\nNothing here is red. This set reports done with no work done, " +
			"which is right for a refactor and wrong for everything else.")
	}
	b.WriteString("\n\nThe person reviews this file before anything runs against it. " +
		"Fix a broken or contradicted criterion now by proposing again; do not start the work.")
	return b.String()
}
