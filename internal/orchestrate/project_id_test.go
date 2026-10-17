package orchestrate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// gitRepo makes a repository with one commit on main and returns its root,
// links resolved.
func gitRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	git(t, root, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(root, "README"), []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "README")
	git(t, root, "commit", "-q", "-m", "first")
	return root
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// Spec: orchestration .i step 5 — the project is the main worktree's root: a
// subdirectory, another worktree of the same tree and a link to it all resolve
// to the same root and the same id.
func TestEveryPathIntoOneTreeResolvesToOneProject(t *testing.T) {
	root := gitRepo(t)
	sub := filepath.Join(root, "pkg")
	if err := os.Mkdir(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(t.TempDir(), "wt")
	git(t, root, "worktree", "add", "-q", "-b", "side", other)
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}

	sum := sha256.Sum256([]byte(root))
	want := hex.EncodeToString(sum[:])[:24]
	for _, dir := range []string{root, sub, other, link} {
		gotRoot, id, err := ResolveProject(context.Background(), dir)
		if err != nil {
			t.Fatalf("ResolveProject(%s): %v", dir, err)
		}
		if gotRoot != root || id != want {
			t.Errorf("ResolveProject(%s) = %s, %s; want %s, %s", dir, gotRoot, id, root, want)
		}
	}
}

func TestADirectoryOutsideAnyRepositoryIsNoProject(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	if _, _, err := ResolveProject(context.Background(), t.TempDir()); err == nil {
		t.Error("a directory outside any repository resolved to a project")
	}
}
