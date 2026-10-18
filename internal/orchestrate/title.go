package orchestrate

import (
	"strings"
	"unicode"
)

// A child's title is a short topic title, such as "listas: Unicos": it names
// the child wherever the person reads it — the list of conversations, the
// agent list, the plan, the merge question's header — and its branch and its
// commits. The first line of a task is what a child was called before, and a
// row of the sidebar cannot hold it.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §10.

// TitleLimit is how many characters a child's title holds.
const TitleLimit = 40

// ShortTitle is the title a child carries: the item's own, trimmed, when it
// is one line that fits; else one derived from it — or from the task's first
// line when the item has none — cut at the last word that fits, with "…".
// derived says the child carries something other than what was asked, which
// the harness then says. The same input always gives the same title.
func ShortTitle(title, task string) (short string, derived bool) {
	asked := strings.TrimSpace(title)
	from := firstLine(asked)
	if from == "" {
		from = firstLine(task)
	}
	short = cutTitle(from, TitleLimit)
	return short, short != asked
}

// firstLine is the first non-blank line of s, trimmed and without control
// characters.
func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		clean := strings.TrimSpace(strings.Map(func(r rune) rune {
			if unicode.IsControl(r) {
				return ' '
			}
			return r
		}, line))
		if clean != "" {
			return clean
		}
	}
	return ""
}

// cutTitle keeps s when it fits in limit characters; else the longest run of its
// words that fits with "…", or its first limit-1 characters with "…" when its
// first word alone does not fit.
func cutTitle(s string, limit int) string {
	r := []rune(s)
	if len(r) <= limit {
		return s
	}
	head := r[:limit-1]
	if i := lastSpace(head); i > 0 {
		head = head[:i]
	}
	return strings.TrimRightFunc(string(head), func(c rune) bool {
		return unicode.IsSpace(c) || c == ',' || c == ';' || c == ':' || c == '-'
	}) + "…"
}

func lastSpace(r []rune) int {
	for i := len(r) - 1; i >= 0; i-- {
		if unicode.IsSpace(r[i]) {
			return i
		}
	}
	return -1
}
