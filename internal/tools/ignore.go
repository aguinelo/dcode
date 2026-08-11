package tools

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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

// scopedRules are the rules from one .gitignore, and the directory they apply
// under.
//
// Scoped because a rule in docs/.gitignore is written relative to docs/ and
// means nothing outside it: `*.draft` there hides docs/notes.draft and must not
// touch src/notes.draft.
type scopedRules struct {
	// prefix is the directory, workspace-relative, with a trailing slash. Empty
	// for the root file.
	prefix string
	rules  []ignoreRule
}

type ignoreSet struct {
	scopes []scopedRules
}

// loadIgnores collects every .gitignore under root, ordered shallowest first.
//
// Nested files used to be skipped, on the reasoning that the gain over a
// root-level file was small. A parity test against git says otherwise, and the
// tool-suite spec named this exact case — "precedência de arquivos aninhados" —
// as one of the hard ones. Two real files in a normal repository were wrong:
// docs/.gitignore un-ignoring one path and hiding another, with dcode
// disagreeing with git in both directions.
//
// Disagreeing by hiding a file git shows is the failure DECISIONS.md rules out,
// because a search that silently omits something is not recoverable by the
// person reading it.
//
// The cost is one directory walk before the search, which is the same walk the
// search is about to do anyway.
func loadIgnores(root string, enabled bool) *ignoreSet {
	set := &ignoreSet{}
	if !enabled {
		return set
	}

	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != ".gitignore" {
			return nil
		}
		rel, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return nil
		}
		prefix := ""
		if rel != "." {
			prefix = filepath.ToSlash(rel) + "/"
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		sc := scopedRules{prefix: prefix}
		for _, line := range strings.Split(string(raw), "\n") {
			if r, ok := parseIgnoreLine(line); ok {
				sc.rules = append(sc.rules, r)
			}
		}
		if len(sc.rules) > 0 {
			set.scopes = append(set.scopes, sc)
		}
		return nil
	})

	// Shallowest first, so a deeper file has the last word — which is how git
	// resolves the two disagreeing.
	sort.SliceStable(set.scopes, func(i, j int) bool {
		return strings.Count(set.scopes[i].prefix, "/") < strings.Count(set.scopes[j].prefix, "/")
	})
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
	for _, sc := range s.scopes {
		// A scoped file only speaks about what is under it, and its rules are
		// written relative to it.
		if sc.prefix != "" && !strings.HasPrefix(rel, sc.prefix) {
			continue
		}
		local := strings.TrimPrefix(rel, sc.prefix)
		if v, ok := matchScope(sc.rules, local, isDir); ok {
			ignored = v
		}
	}
	return ignored
}

// matchScope applies one file's rules, reporting whether any of them decided.
//
// The second return matters: a scope that says nothing must leave a shallower
// decision standing, and a scope that says "not ignored" through a negation
// must overturn one. Collapsing the two into a bool loses the difference.
func matchScope(rules []ignoreRule, rel string, isDir bool) (ignored, decided bool) {
	for _, r := range rules {
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
		ignored, decided = !r.negate, true
	}
	return ignored, decided
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
