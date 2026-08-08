package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Declare parses the same arguments Execute does, so it fails the same way on
// malformed input. It must never panic: the loop calls it before it has any
// error handling of its own for that call.
func TestDeclareRejectsMalformedArguments(t *testing.T) {
	bad := json.RawMessage(`{"path":`)
	for _, tool := range []Tool{Read{}, Write{}, Edit{}, Glob{}, Grep{}, Bash{}} {
		if _, err := tool.Declare(bad); err == nil {
			t.Errorf("%s: malformed arguments should be reported", tool.Name())
		}
	}
	// plan takes no path and no network, so it declares cleanly even here.
	if _, err := (Plan{}).Declare(bad); err != nil {
		t.Errorf("plan should declare without parsing: %v", err)
	}
}

func TestGlobAndGrepDefaultToTheWorkspaceRoot(t *testing.T) {
	req, err := (Glob{}).Declare(json.RawMessage(`{"pattern":"*.go"}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(req.Paths) != 1 || req.Paths[0].Path != "." {
		t.Errorf("an omitted root must mean the workspace, got %+v", req.Paths)
	}
	if req.Paths[0].Write {
		t.Error("searching is a read; declaring a write would ask for permission it does not need")
	}

	req, err = (Grep{}).Declare(json.RawMessage(`{"pattern":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(req.Paths) != 1 || req.Paths[0].Path != "." {
		t.Errorf("got %+v", req.Paths)
	}
}

func TestGlobHonoursAnExplicitRoot(t *testing.T) {
	req, err := (Glob{}).Declare(json.RawMessage(`{"pattern":"*.go","path":"internal"}`))
	if err != nil {
		t.Fatal(err)
	}
	if req.Paths[0].Path != "internal" {
		t.Errorf("got %q", req.Paths[0].Path)
	}
}

// An unreadable file inside the tree must not abort a whole search: one
// permission problem should cost one file, not the entire result.
func TestSearchSurvivesAnUnreadableEntry(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root; permissions do not apply")
	}
	s, ws := setup(t)
	writeFileT(t, ws, "readable.go", "package x\n")

	locked := filepath.Join(ws, "locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFileT(t, ws, "locked/hidden.go", "package y\n")
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Skipf("cannot restrict permissions here: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	res := run(t, Glob{}, s, GlobInput{Pattern: "**/*.go"})
	if res.IsError {
		t.Fatalf("one unreadable directory must not fail the search: %+v", res)
	}
	if !strings.Contains(res.Output, "readable.go") {
		t.Errorf("readable files should still be listed: %q", res.Output)
	}
}

// Writing into a path whose parent is a file cannot succeed; the failure has to
// reach the model as something it can act on rather than as a crash.
func TestWriteFailureIsReportedNotRaised(t *testing.T) {
	s, ws := setup(t)
	writeFileT(t, ws, "blocker", "i am a file\n")

	res := run(t, Write{}, s, WriteInput{Path: "blocker/child.go", Content: "x"})
	if !res.IsError {
		t.Fatalf("writing under a file must fail: %+v", res)
	}
}

func TestReadReportsAPermissionProblem(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root; permissions do not apply")
	}
	s, ws := setup(t)
	p := writeFileT(t, ws, "secret.txt", "shh\n")
	if err := os.Chmod(p, 0o000); err != nil {
		t.Skipf("cannot restrict permissions here: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(p, 0o644) })

	res := run(t, Read{}, s, ReadInput{Path: "secret.txt"})
	if !res.IsError {
		t.Fatalf("an unreadable file must be reported: %+v", res)
	}
	// It exists, so the message must not claim otherwise — that would send the
	// model looking for a path that is right there.
	if strings.Contains(res.Output, "does not exist") {
		t.Errorf("a permission problem must not be reported as absence: %q", res.Output)
	}
}

func TestAtomicWriteLeavesNoTemporaryBehind(t *testing.T) {
	s, ws := setup(t)
	if res := run(t, Write{}, s, WriteInput{Path: "a.go", Content: "x\n"}); res.IsError {
		t.Fatal(res)
	}
	entries, err := os.ReadDir(ws)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".dcode-") {
			t.Errorf("a temporary file survived: %s", e.Name())
		}
	}
}

func TestIgnorePatterns(t *testing.T) {
	for _, tc := range []struct {
		name    string
		rules   string
		path    string
		isDir   bool
		ignored bool
	}{
		{"bare name at any depth", "node_modules\n", "a/b/node_modules", true, true},
		{"extension anywhere", "*.log\n", "deep/x.log", false, true},
		{"anchored to root", "/build\n", "build", true, true},
		{"anchored does not match deeper", "/build\n", "sub/build", true, false},
		{"directory only", "dist/\n", "dist", true, true},
		{"directory rule skips files", "dist/\n", "dist", false, false},
		{"negation wins when later", "*.log\n!keep.log\n", "keep.log", false, false},
		{"contents of an ignored dir", "vendor/\n", "vendor/pkg/a.go", false, true},
		{"comment ignored", "# a comment\n", "a.go", false, false},
		{"blank line ignored", "\n\n", "a.go", false, false},
		{"double star", "**/testdata/**\n", "a/testdata/b.json", false, true},
		{"question mark", "?.go\n", "a.go", false, true},
		{"question mark does not cross", "?.go\n", "ab.go", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(tc.rules), 0o644); err != nil {
				t.Fatal(err)
			}
			set := loadIgnores(dir, true)
			if got := set.match(tc.path, tc.isDir); got != tc.ignored {
				t.Errorf("rules %q, path %q dir=%v: got ignored=%v want %v",
					tc.rules, tc.path, tc.isDir, got, tc.ignored)
			}
		})
	}
}

func TestIgnoreDisabledMatchesNothing(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if loadIgnores(dir, false).match("a.go", false) {
		t.Error("with gitignore off, nothing should be ignored")
	}
}

func TestIgnoreWithNoFileMatchesNothing(t *testing.T) {
	if loadIgnores(t.TempDir(), true).match("anything", false) {
		t.Error("no .gitignore means nothing is ignored")
	}
}

// A malformed rule must be skipped rather than break the whole file: one bad
// line should not disable every other rule the user wrote.
func TestMalformedIgnoreLineIsSkipped(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"),
		[]byte("[unclosed\n*.log\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	set := loadIgnores(dir, true)
	if !set.match("x.log", false) {
		t.Error("a valid rule after a malformed one must still apply")
	}
}

func TestGitDirectoryIsAlwaysSkipped(t *testing.T) {
	s, ws := setup(t)
	writeFileT(t, ws, ".git/config", "[core]\n")
	writeFileT(t, ws, "a.go", "package x\n")

	res := run(t, Glob{}, s, GlobInput{Pattern: "**/*"})
	if strings.Contains(res.Output, ".git/") {
		t.Errorf("the git directory must never be searched: %q", res.Output)
	}
}

func TestGrepTruncatesAtTheMatchLimit(t *testing.T) {
	s, ws := setup(t)
	var b strings.Builder
	for i := 0; i < 100; i++ {
		b.WriteString("needle\n")
	}
	writeFileT(t, ws, "many.txt", b.String())
	s.Limits.GrepMaxMatches = 5

	res := run(t, Grep{}, s, GrepInput{Pattern: "needle"})
	if !res.Truncated {
		t.Errorf("truncation must be declared: %+v", res)
	}
	if strings.Count(res.Output, "many.txt:") > 5 {
		t.Errorf("more matches than the limit: %q", res.Output)
	}
}

func TestGrepClampsAVeryLongLine(t *testing.T) {
	s, ws := setup(t)
	writeFileT(t, ws, "min.js", "needle"+strings.Repeat("x", 5000)+"\n")
	res := run(t, Grep{}, s, GrepInput{Pattern: "needle"})
	if len(res.Output) > 1000 {
		t.Errorf("a matching minified line must not flood the context: %d bytes", len(res.Output))
	}
}

func TestGrepSkipsBinaryFiles(t *testing.T) {
	s, ws := setup(t)
	if err := os.WriteFile(filepath.Join(ws, "bin"), []byte{0xff, 0xfe, 0x00}, 0o644); err != nil {
		t.Fatal(err)
	}
	writeFileT(t, ws, "a.txt", "needle\n")
	res := run(t, Grep{}, s, GrepInput{Pattern: "."})
	if strings.Contains(res.Output, "bin:") {
		t.Errorf("binary content must not be searched: %q", res.Output)
	}
}
