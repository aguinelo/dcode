package orchestrate

import (
	"path"
	"sort"
	"strings"

	"github.com/aguinelo/dcode/internal/protocol"
)

// What a child owns: the paths it alone may write, disjoint from every other
// live child's.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §4, §10.

// globMeta are the characters policy.Glob reads as more than themselves.
const globMeta = "*?"

// ownsPrefix is the literal part of an owns entry: up to its first glob
// metacharacter, cut back to the last separator before it. "." and "" are the
// whole tree, the empty prefix.
func ownsPrefix(e string) string {
	e = strings.TrimPrefix(e, "./")
	if e == "." {
		return ""
	}
	if i := strings.IndexAny(e, globMeta); i >= 0 {
		e = e[:i]
		if j := strings.LastIndex(e, "/"); j >= 0 {
			return e[:j]
		}
		return ""
	}
	return strings.TrimSuffix(e, "/")
}

// OwnsOverlap reports whether two owns entries may claim one path. It is
// conservative: each entry is its literal prefix, and two cross when one
// prefix contains the other by path component. So `a/*.go` crosses `a/*.md` —
// the price of never letting a real crossing through, said in the refusal.
func OwnsOverlap(a, b string) bool {
	pa, pb := ownsPrefix(a), ownsPrefix(b)
	return underPrefix(pa, pb) || underPrefix(pb, pa)
}

// underPrefix reports whether p sits at or under dir, by component: "a/bc" is not
// under "a/b". The empty dir is the whole tree.
func underPrefix(dir, p string) bool {
	if dir == "" || dir == p {
		return true
	}
	return strings.HasPrefix(p, dir+"/")
}

// validOwns refuses an owns that is empty, or names the whole tree, or a path
// that is absolute, not clean, or climbs out of it.
func validOwns(title string, owns []string) error {
	if len(owns) == 0 {
		return protocol.Errorf(protocol.CodeInvalidInput,
			"%q owns nothing: a child writes only the paths it owns, so owns names at least one", title)
	}
	for _, e := range owns {
		switch {
		case e == "" || e == ".":
			return protocol.Errorf(protocol.CodeInvalidInput,
				"%q owns %q: owning the whole tree is owning nothing in particular, so name the paths", title, e)
		case strings.HasPrefix(e, "/") || strings.Contains(e, `\`):
			return protocol.Errorf(protocol.CodeInvalidInput,
				"%q owns %q: owns are slash paths relative to the project's root", title, e)
		case path.Clean(e) != e:
			return protocol.Errorf(protocol.CodeInvalidInput,
				"%q owns %q, which is not a clean path; write %q", title, e, path.Clean(e))
		case e == ".." || strings.HasPrefix(e, "../"):
			return protocol.Errorf(protocol.CodeInvalidInput,
				"%q owns %q, which is outside the project's tree", title, e)
		}
	}
	return nil
}

// Leases are the owns of the live children, by child id: what a new batch
// may not cross.
type Leases struct {
	byChild map[string][]string
}

// leasesOf are the owns every child that is not terminal still holds — a
// failed one keeps its worktree, and so its paths.
func leasesOf(children []protocol.Child) Leases {
	l := Leases{byChild: map[string][]string{}}
	for _, c := range children {
		if !terminal(c.State) {
			l.byChild[c.ID] = c.Owns
		}
	}
	return l
}

// Crossing names the first lease an entry crosses: the child and its path.
func (l Leases) Crossing(entry string) (child, held string, ok bool) {
	ids := make([]string, 0, len(l.byChild))
	for id := range l.byChild {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		for _, h := range l.byChild[id] {
			if OwnsOverlap(entry, h) {
				return id, h, true
			}
		}
	}
	return "", "", false
}

// overlapIn refuses a batch whose items cross each other or a live child,
// naming both paths. The whole batch, before any worktree.
func overlapIn(items []Item, leases Leases) error {
	for i, it := range items {
		for _, e := range it.Owns {
			if child, held, ok := leases.Crossing(e); ok {
				return protocol.Errorf(protocol.CodeOwnsOverlap,
					"%q owns %q, which crosses %q held by the live child %s; the batch is refused whole, and nothing was created. "+
						"Owns are compared by their literal prefix up to the first * or ?, so two globs in one folder cross",
					it.Title, e, held, child)
			}
			for _, other := range items[i+1:] {
				for _, f := range other.Owns {
					if OwnsOverlap(e, f) {
						return protocol.Errorf(protocol.CodeOwnsOverlap,
							"%q owns %q and %q owns %q, which cross; the batch is refused whole, and nothing was created. "+
								"Owns are compared by their literal prefix up to the first * or ?, so two globs in one folder cross",
							it.Title, e, other.Title, f)
					}
				}
			}
		}
	}
	return nil
}
