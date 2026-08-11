package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The spec chose github.com/boyter/gocodewalker for .gitignore matching and the
// code hand-rolled it, with DECISIONS.md giving the reason. The spec's worry
// was legitimate and specific — negation with !, **, precedence of nested files
// — and "implementar do zero é fonte de bug sutil" is true.
//
// The answer to that worry is this test rather than a dependency: the cases are
// checked against git itself, so the claim is not "I believe it is right" but
// "git and this agree". A hand-rolled matcher with no oracle would have been
// the spec's point proved.
func TestGitignoreMatchesGit(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("no git to compare against")
	}

	ws := t.TempDir()
	write := func(p, body string) {
		full := filepath.Join(ws, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Every construct the spec names as hard, plus a nested .gitignore, whose
	// precedence is the case most implementations get wrong.
	write(".gitignore", strings.Join([]string{
		"build/",
		"*.log",
		"!keep.log",
		"vendor/**",
		"docs/**/*.tmp",
		"/only-at-root.txt",
		"*.bak",
	}, "\n")+"\n")
	write("docs/.gitignore", "!important.tmp\n*.draft\n")

	files := []string{
		"src/main.go",
		"keep.log",
		"a.log",
		"build/out.bin",
		"vendor/pkg/a.go",
		"docs/deep/nested/x.tmp",
		"docs/deep/nested/x.md",
		"docs/important.tmp",
		"docs/notes.draft",
		"only-at-root.txt",
		"sub/only-at-root.txt",
		"sub/thing.bak",
	}
	for _, f := range files {
		write(f, "x")
	}

	if out, err := exec.Command(git, "-C", ws, "init", "-q").CombinedOutput(); err != nil {
		t.Skipf("git init failed: %v: %s", err, out)
	}

	got, err := walkFiles(ws, true, func(string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	included := map[string]bool{}
	for _, g := range got {
		included[filepath.ToSlash(g)] = true
	}

	for _, f := range files {
		// git check-ignore exits 0 when the path IS ignored.
		cmd := exec.Command(git, "-C", ws, "check-ignore", "-q", f)
		ignoredByGit := cmd.Run() == nil
		if included[f] == ignoredByGit {
			verb := "included"
			if !included[f] {
				verb = "excluded"
			}
			gitVerb := "ignores"
			if !ignoredByGit {
				gitVerb = "does not ignore"
			}
			t.Errorf("%s: dcode %s it and git %s it", f, verb, gitVerb)
		}
	}
}

// Where the two ever differ, the direction is chosen. DECISIONS.md records it:
// a search showing something extra is recoverable, one silently hiding a file
// is not — so an unparseable rule includes rather than excludes.
func TestAnUnreadableRuleIncludesRatherThanHides(t *testing.T) {
	ws := t.TempDir()
	if err := os.WriteFile(filepath.Join(ws, ".gitignore"), []byte("[unclosed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, "a.go"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := walkFiles(ws, true, func(string) bool { return true })
	if err != nil {
		t.Fatalf("a malformed rule broke the walk: %v", err)
	}
	found := false
	for _, g := range got {
		if g == "a.go" {
			found = true
		}
	}
	if !found {
		t.Fatal("a file was hidden by a rule that could not be read; showing something extra is recoverable, hiding is not")
	}
}
