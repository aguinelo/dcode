package tools

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Gitignore matching.
//
// The commonly cited dedicated packages are abandoned — sabhiram last published
// in 2021, denormal in 2018 — and go-git would pull twenty-odd transitive
// dependencies to match file patterns. This is a focused implementation of the
// subset that matters for a workspace walk: negation, directory-only rules,
// anchoring and `**`.
//
// It is deliberately not a complete gitignore engine. Where it differs from git
// it errs toward *including* a file, because a search that shows something
// extra is recoverable and one that silently hides a file the user expected is
// not.

type ignoreRule struct {
	re      *regexp.Regexp
	negate  bool
	dirOnly bool
}

type ignoreSet struct {
	rules []ignoreRule
}

// loadIgnores reads .gitignore from root. Nested ignore files are not walked:
// the cost is one more read per directory on every search, and the gain over a
// root-level file is small for a workspace.
func loadIgnores(root string, enabled bool) *ignoreSet {
	set := &ignoreSet{}
	if !enabled {
		return set
	}
	raw, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		return set
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if r, ok := parseIgnoreLine(line); ok {
			set.rules = append(set.rules, r)
		}
	}
	return set
}

func parseIgnoreLine(line string) (ignoreRule, bool) {
	line = strings.TrimRight(line, " \r")
	if line == "" || strings.HasPrefix(line, "#") {
		return ignoreRule{}, false
	}

	var r ignoreRule
	if strings.HasPrefix(line, "!") {
		r.negate = true
		line = line[1:]
	}
	if strings.HasSuffix(line, "/") {
		r.dirOnly = true
		line = strings.TrimSuffix(line, "/")
	}
	if line == "" {
		return ignoreRule{}, false
	}

	anchored := strings.HasPrefix(line, "/") || strings.Contains(strings.TrimSuffix(line, "/"), "/")
	line = strings.TrimPrefix(line, "/")

	re, err := compileGlob(line, anchored)
	if err != nil {
		return ignoreRule{}, false
	}
	r.re = re
	return r, true
}

// match reports whether rel is ignored. Later rules win, which is how git
// resolves a negation after a broad exclusion.
func (s *ignoreSet) match(rel string, isDir bool) bool {
	ignored := false
	for _, r := range s.rules {
		m := r.re.FindStringSubmatch(rel)
		if m == nil {
			continue
		}
		if r.dirOnly && !isDir {
			// A directory rule also covers everything beneath it. The walk
			// happens to skip the whole subtree anyway, but the rule has to
			// stand on its own: relying on the traversal would make the
			// meaning of `vendor/` depend on how it was reached.
			if len(m) < 2 || len(rel) <= len(m[1]) {
				continue // the path is the directory itself, and it is a file
			}
		}
		ignored = !r.negate
	}
	return ignored
}

// compileGlob turns a glob into a regexp.
//
//	**  crosses directory separators
//	*   matches within one segment
//	?   matches one character
//
// An unanchored pattern matches at any depth, which is what a bare name like
// `node_modules` means in a gitignore.
func compileGlob(pattern string, anchored bool) (*regexp.Regexp, error) {
	var b strings.Builder
	// Group 1 captures the pattern body without the "and everything under it"
	// tail, so match can tell a directory itself from its contents.
	b.WriteString("^(")
	if !anchored {
		b.WriteString("(?:.*/)?")
	}

	for i := 0; i < len(pattern); i++ {
		switch c := pattern[i]; c {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				i++
				// `**/` collapses to "any number of directories", including
				// none, so `**/*.go` also matches a file at the root.
				if i+1 < len(pattern) && pattern[i+1] == '/' {
					i++
					b.WriteString("(?:.*/)?")
				} else {
					b.WriteString(".*")
				}
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		case '.', '+', '(', ')', '|', '^', '$', '{', '}', '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		case '[':
			// Character classes pass through; a malformed one surfaces as a
			// compile error the caller turns into invalid_pattern.
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	// A directory pattern also covers everything beneath it.
	b.WriteString(")(?:/.*)?$")
	return regexp.Compile(b.String())
}

func globToRegexp(tool, pattern string) (*regexp.Regexp, *ToolError) {
	if strings.TrimSpace(pattern) == "" {
		return nil, errf(tool, CodeInvalidPattern, "", "pattern is required")
	}
	anchored := strings.Contains(pattern, "/")
	re, err := compileGlob(strings.TrimPrefix(pattern, "./"), anchored)
	if err != nil {
		return nil, errf(tool, CodeInvalidPattern,
			"Check the brackets and escapes in the pattern.",
			"invalid glob %q: %v", pattern, err)
	}
	return re, nil
}

func globToRegexpOrNil(pattern string) (*regexp.Regexp, error) {
	anchored := strings.Contains(pattern, "/")
	return compileGlob(strings.TrimPrefix(pattern, "./"), anchored)
}
