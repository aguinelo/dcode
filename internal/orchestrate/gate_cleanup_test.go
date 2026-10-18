package orchestrate

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// halfMade is what a git killed in the middle of `worktree add` leaves: the
// tree on the disk, registered, and locked with the reason "initializing".
func halfMade(t *testing.T, repo, place, commit string) {
	t.Helper()
	git(t, repo, "worktree", "add", "--quiet", "--detach", place, commit)
	git(t, repo, "worktree", "lock", "--reason", "initializing", place)
}

// Spec: orchestration .p §7.4 — the throwaway worktree is removed even when
// the git that made it was killed half-way and left it locked.
func TestALockedHalfMadeGateWorktreeIsRemoved(t *testing.T) {
	repo, _, second := gateRepo(t)
	scratch := t.TempDir()
	place := filepath.Join(scratch, "tree")
	halfMade(t, repo, place, second)

	if err := removeTree(repo, place, scratch); err != nil {
		t.Fatalf("removeTree: %v", err)
	}
	if _, err := os.Stat(place); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the locked tree %s is still on the disk (%v)", place, err)
	}
	if n := worktrees(t, repo); n != 1 {
		t.Errorf("%d worktrees after removing a locked one, want only the person's", n)
	}
}

// Spec: orchestration .p §10 — a child's worktree left locked by a git killed
// half-way is removed when the child is closed.
func TestALockedHalfMadeChildWorktreeIsRemoved(t *testing.T) {
	repo, _, second := gateRepo(t)
	place := filepath.Join(t.TempDir(), "child")
	halfMade(t, repo, place, second)

	if err := removeWorktree(context.Background(), repo, place); err != nil {
		t.Fatalf("removeWorktree: %v", err)
	}
	if _, err := os.Stat(place); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the locked tree %s is still on the disk (%v)", place, err)
	}
	if n := worktrees(t, repo); n != 1 {
		t.Errorf("%d worktrees after removing a locked one, want only the person's", n)
	}
}

// Spec: orchestration .p §10 — an add that fails leaves no worktree behind,
// and a path that was there before the add is never the add's to remove.
func TestAFailedAddLeavesNothingAndKeepsWhatWasThere(t *testing.T) {
	repo, _, second := gateRepo(t)
	place := filepath.Join(t.TempDir(), "child")
	if err := addWorktree(context.Background(), repo, place, "b1", "no-such-commit"); err == nil {
		t.Fatal("an add from a commit that does not exist succeeded")
	}
	if _, err := os.Stat(place); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a failed add left %s (%v)", place, err)
	}

	kept := filepath.Join(t.TempDir(), "kept")
	if err := os.MkdirAll(kept, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(kept, "mine.txt"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := addWorktree(context.Background(), repo, kept, "b2", second); err == nil {
		t.Fatal("an add over a directory that is not empty succeeded")
	}
	if _, err := os.Stat(filepath.Join(kept, "mine.txt")); err != nil {
		t.Errorf("a failed add removed what was there before it: %v", err)
	}
	if n := worktrees(t, repo); n != 1 {
		t.Errorf("%d worktrees after failed adds, want only the person's", n)
	}
}

// Spec: orchestration .p §10 — a half-made integration place goes with the
// add that failed, so the next open is not refused by it.
func TestAFailedIntegrationAddLeavesNoPlace(t *testing.T) {
	root := filepath.Join(t.TempDir(), "empty")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, root, "init", "-q") // no commit: HEAD names nothing to check out
	dir := filepath.Join(t.TempDir(), "project")
	if _, err := OpenIntegration(context.Background(), root, dir); err == nil {
		t.Fatal("an integration place was made from a repository with no commit")
	}
	if _, err := os.Stat(filepath.Join(dir, integrationDir)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a failed add left the integration place (%v)", err)
	}
}

// Spec: AGENTS.md — what stays after a removal is said: on the disk, or with
// git that cannot be asked.
func TestAWorktreeThatStaysIsSaid(t *testing.T) {
	repo, _, _ := gateRepo(t)
	parent := filepath.Join(t.TempDir(), "locked")
	stuck := filepath.Join(parent, "tree")
	if err := os.MkdirAll(stuck, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stuck, "f"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })
	if err := removeWorktree(context.Background(), repo, stuck); err == nil {
		t.Error("a directory that could not be removed was reported removed")
	} else if !strings.Contains(err.Error(), "still on the disk") {
		t.Errorf("err = %v, want it to say the tree is still on the disk", err)
	}

	if err := removeWorktree(context.Background(), t.TempDir(), filepath.Join(t.TempDir(), "gone")); err == nil {
		t.Error("a removal git could not confirm was reported done")
	}
}

// Spec: orchestration .p §8 — a gate stopped before it read the commit is the
// cancellation too.
func TestAGateStoppedBeforeItStartsIsTheCancellation(t *testing.T) {
	repo, _, second := gateRepo(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (Gate{Command: "true", Timeout: time.Minute}).Measure(ctx, repo, second); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want the cancellation", err)
	}
}

// Spec: orchestration .p §8 — a caller that stops the gate while its worktree
// is being made gets the cancellation back, not git's kill, and no worktree is
// left behind. The cancel lands inside the add, by a seam, never by a timer.
func TestAGateStoppedWhileItsWorktreeIsMadeIsTheCancellation(t *testing.T) {
	repo, _, second := gateRepo(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	saved := addGateTree
	addGateTree = func(_ context.Context, dir, place, commit string) error {
		halfMade(t, dir, place, commit)
		cancel()
		return errors.New("git worktree add: signal: killed")
	}
	t.Cleanup(func() { addGateTree = saved })

	r, err := Gate{Command: "true", Timeout: time.Minute}.Measure(ctx, repo, second)

	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want the cancellation", err)
	}
	if r.Result != "" {
		t.Errorf("result = %q for a gate that never ran", r.Result)
	}
	if n := worktrees(t, repo); n != 1 {
		t.Errorf("%d worktrees after a cancel during the add, want only the person's", n)
	}
}
