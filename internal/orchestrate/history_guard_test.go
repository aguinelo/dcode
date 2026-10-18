package orchestrate

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Spec: orchestration .p §7.6, §14 → §13; .config §4 — no path of this
// slice's code rebases, force-pushes or amends: a commit that exists is never
// rewritten. A source guard over every call in internal/orchestrate and
// internal/vcs that passes git its words as literals, which is how both
// packages call git. `worktree remove --force` throws a scratch tree away and
// rewrites nothing; a push with --force, -f, --force-with-lease or a +refspec
// does, and so do rebase and --amend anywhere.
func TestNoCodeOfThisSliceRebasesForcePushesOrAmends(t *testing.T) {
	var found []string
	for _, dir := range []string{".", filepath.Join("..", "vcs")} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			name := e.Name()
			if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			found = append(found, rewrites(t, filepath.Join(dir, name))...)
		}
	}
	for _, f := range found {
		t.Errorf("history rewritten: %s", f)
	}
}

// rewrites is every call in file whose literal arguments rewrite history.
func rewrites(t *testing.T, file string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		var words []string
		for _, a := range call.Args {
			if lit, ok := a.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				if s, err := strconv.Unquote(lit.Value); err == nil {
					words = append(words, s)
				}
			}
		}
		if why := rewriting(words); why != "" {
			out = append(out, fset.Position(call.Pos()).String()+": "+why)
		}
		return true
	})
	return out
}

func rewriting(words []string) string {
	push := false
	for _, w := range words {
		if w == "push" {
			push = true
		}
	}
	for _, w := range words {
		switch {
		case w == "rebase" || w == "--amend" || strings.HasPrefix(w, "--force-with-lease"):
			return w
		case push && (w == "--force" || w == "-f" || strings.HasPrefix(w, "+")):
			return "push " + w
		}
	}
	return ""
}

// The guard sees what it is there to see.
func TestTheHistoryGuardCatchesARewrite(t *testing.T) {
	for _, words := range [][]string{{"rebase", "main"}, {"commit", "--amend"}, {"push", "--force"},
		{"push", "-q", "origin", "+refs/heads/x:refs/heads/x"}, {"push", "--force-with-lease"}} {
		if rewriting(words) == "" {
			t.Errorf("%q was not caught", words)
		}
	}
	for _, words := range [][]string{{"worktree", "remove", "--force", "x"}, {"push", "-q", "-u", "origin", "refs/heads/x:refs/heads/x"}} {
		if why := rewriting(words); why != "" {
			t.Errorf("%q was caught as %s", words, why)
		}
	}
}
