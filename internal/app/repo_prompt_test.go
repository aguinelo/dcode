package app

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func writeIn(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("git %v: %v\n%s", args, err, out)
	}
}

// The wiring, end to end: a session opened in a repository carries the branch
// into the prompt the model actually reads.
//
// Asserted here rather than only in behavior, because the two halves were
// separately correct and unconnected — the renderer knew how to say it and
// nothing was reading it. That is the shape this codebase keeps finding.
func TestASessionInARepositoryKnowsWhereItIs(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git here")
	}
	ws := t.TempDir()
	gitIn(t, ws, "init", "-q", "-b", "main")
	gitIn(t, ws, "config", "user.email", "t@example.com")
	gitIn(t, ws, "config", "user.name", "T")
	gitIn(t, ws, "config", "commit.gpgsign", "false")

	writeIn(t, ws, "a.txt", "one\n")
	gitIn(t, ws, "add", "a.txt")
	gitIn(t, ws, "commit", "-qm", "feat: a memorable subject")

	// Dirty it, so the tree state is a fact and not a default.
	writeIn(t, ws, "b.txt", "two\n")

	opts := baseOpts(t)
	opts.Workspace = ws
	requireSandbox(t, opts)

	sess, err := New(opts, &ConsoleEmitter{W: io.Discard}, DenyAll{})
	if err != nil {
		t.Fatalf("wiring a session failed: %v", err)
	}
	defer sess.Engine.Close()

	for _, want := range []string{
		"main",                // the branch, and the branch work is cut from
		"b.txt",               // what is uncommitted
		"a memorable subject", // what was done recently
		"snapshot",            // and that all of it is one
	} {
		if !strings.Contains(sess.Prompt, want) {
			t.Errorf("the prompt does not carry %q", want)
		}
	}
}

// A workspace that is not a repository opens exactly as it always did, with no
// section about a repository that is not there.
func TestASessionOutsideARepositoryIsUnchanged(t *testing.T) {
	opts := baseOpts(t)
	requireSandbox(t, opts)

	sess, err := New(opts, &ConsoleEmitter{W: io.Discard}, DenyAll{})
	if err != nil {
		t.Fatalf("wiring a session failed: %v", err)
	}
	defer sess.Engine.Close()

	if strings.Contains(sess.Prompt, "Current branch") {
		t.Error("a directory that is not a repository got a branch in its prompt")
	}
}
