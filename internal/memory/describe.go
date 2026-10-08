package memory

import (
	"fmt"
	"strings"
)

// Diagnose says what is wrong with a line File.Malformed holds, in the words a
// person fixing the file needs.
//
// Here, beside the grammar, rather than wherever the line is shown: a reason
// written anywhere else is a second reading of the header, and the day the two
// disagree a block is reported for something it does not have.
func Diagnose(line string) string {
	trimmed := strings.TrimSpace(line)
	if _, _, ok := splitHeader(trimmed); ok {
		return "the header is indented; a memory's header starts the line"
	}
	rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "##"))
	kind, subject, ok := strings.Cut(rest, ":")
	if !ok {
		return "no kind before the subject; a memory's header is `## kind: subject`"
	}
	k := Kind(strings.ToLower(strings.TrimSpace(kind)))
	if !k.Valid() {
		names := make([]string, 0, len(Kinds()))
		for _, v := range Kinds() {
			names = append(names, string(v))
		}
		return fmt.Sprintf("%q is not a kind of memory; the kinds are %s and %s",
			strings.TrimSpace(kind), strings.Join(names[:len(names)-1], ", "), names[len(names)-1])
	}
	if strings.TrimSpace(subject) == "" {
		return "no subject after the kind; a memory's header is `## kind: subject`"
	}
	return "not a memory's header, which is `## kind: subject`"
}

// Hidden is how many of the oldest entries do not reach the prefix under max,
// the cut Render declares. A max of zero or less is DefaultMax, as in Render.
func Hidden(f File, max int) int {
	if max <= 0 {
		max = DefaultMax
	}
	if len(f.Entries) > max {
		return len(f.Entries) - max
	}
	return 0
}

// Stale says e was true at a commit the repository no longer has, the mark
// Render prints. Never for a memory that claimed no commit, and never when
// known is empty: "we did not look" must not read as "we looked and it is gone".
func Stale(e Entry, known map[string]bool) bool {
	return e.Commit != "" && len(known) > 0 && !known[e.Commit]
}
