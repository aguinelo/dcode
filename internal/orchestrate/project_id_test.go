package orchestrate

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(gitEnv(os.Environ()), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// Spec: orchestration .config — Project.ID is 24 hex of the sha256 of the
// resolved root, the same through every path to the same tree.
func TestAWorktreeALinkAndASubfolderResolveToTheSameProject(t *testing.T) {
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	if err := os.MkdirAll(filepath.Join(repo, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "init", "-q")
	git(t, repo, "commit", "-q", "--allow-empty", "-m", "first")
	git(t, repo, "worktree", "add", "-q", filepath.Join(base, "wt"))
	if err := os.Symlink(repo, filepath.Join(base, "link")); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	root, id, err := ResolveProject(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.EvalSymlinks(repo)
	if root != want {
		t.Errorf("root = %q, want %q", root, want)
	}
	if !projectID.MatchString(id) {
		t.Errorf("id %q is not 24 hex", id)
	}
	for _, other := range []string{filepath.Join(base, "wt"), filepath.Join(base, "link"), filepath.Join(repo, "sub")} {
		r, i, err := ResolveProject(ctx, other)
		if err != nil {
			t.Fatalf("%s: %v", other, err)
		}
		if r != root || i != id {
			t.Errorf("%s = (%s, %s), want (%s, %s)", other, r, i, root, id)
		}
	}
}

func TestAFolderOutsideGitIsNotAProject(t *testing.T) {
	_, _, err := ResolveProject(context.Background(), t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "git") {
		t.Fatalf("err = %v, want one saying the folder is in no git tree", err)
	}
}
