package orchestrate

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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
