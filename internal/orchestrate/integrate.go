package orchestrate

// The integration place: a detached worktree of the daemon's, one per project,
// where a child's branch is brought up to date and a local squash is made.
// Never the person's tree (RN-6): integrating there once reloaded the app the
// person was using.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §7.4, §7.6, §8.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aguinelo/dcode/internal/protocol"
)

const integrationDir = "integration"

// Integration is one project's integration place. One git at a time runs in
// it: the place has one HEAD.
type Integration struct {
	repo, place string
	mu          sync.Mutex
}

// OpenIntegration finds or makes <projectDir>/integration: a worktree of the
// repository at root, detached at its HEAD. A projectDir inside the person's
// tree is refused — the place is the daemon's, never theirs.
func OpenIntegration(ctx context.Context, root, projectDir string) (*Integration, error) {
	place := filepath.Join(projectDir, integrationDir)
	if rel, err := filepath.Rel(root, place); err != nil || rel == "." || !strings.HasPrefix(rel, "..") {
		return nil, fmt.Errorf("orchestrate: the integration place %s is inside the project's tree %s; it is the daemon's, never the person's", place, root)
	}
	in := &Integration{repo: root, place: place}
	if _, err := os.Stat(place); err == nil {
		top, err := forgeGit(ctx, place, "rev-parse", "--show-toplevel")
		// git names the top with links resolved; place may hold one (/var on
		// macOS is /private/var), and is the same place for it.
		resolved, rerr := filepath.EvalSymlinks(place)
		if err != nil || rerr != nil || filepath.Clean(top) != filepath.Clean(resolved) {
			return nil, fmt.Errorf("orchestrate: %s exists and is not the integration worktree (%v)", place, err)
		}
		return in, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("orchestrate: read %s: %w", place, err)
	}
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		return nil, fmt.Errorf("orchestrate: make %s: %w", projectDir, err)
	}
	if _, err := forgeGit(ctx, root, "worktree", "add", "-q", "--detach", place, "HEAD"); err != nil {
		return nil, fmt.Errorf("orchestrate: make the integration place: %w", err)
	}
	return in, nil
}

// Conflict is main and a child's branch that do not merge. It goes back to
// the child that owns the branch, with the paths; the orchestrator resolves
// nothing (Q-H).
type Conflict struct{ Paths []string }

func (c *Conflict) Error() string {
	return fmt.Sprintf("merging main into the branch conflicts in %s; it goes back to the child that owns it",
		strings.Join(c.Paths, ", "))
}

// Return is the child's return for the conflict.
func (c *Conflict) Return() protocol.Return {
	return protocol.Return{Why: "conflict", Paths: c.Paths}
}

// Update brings the child's branch up to date by merging main into it —
// `git merge --no-ff main` over head, in the integration place, never
// rewriting a commit that exists. The branch, checked out in the child's
// worktree, moves forward to the result H, which Update returns: H is what
// the gate measures. A head that has main already is H itself.
//
// A conflict is aborted in the integration place, which ends clean, and made
// again in the child's worktree with the markers left for the child; it comes
// back as *Conflict, which the pipeline hands back to the child — and writes
// child.returned, in the one shape every return has.
func (i *Integration) Update(ctx context.Context, l *DecisionLog, child, worktree, head, main string) (string, error) {
	if l == nil {
		return "", errors.New("orchestrate: no decision log to record the update in")
	}
	if !exactCommit.MatchString(head) || !exactCommit.MatchString(main) {
		return "", fmt.Errorf("orchestrate: an update merges exact commits; got %q over %q", head, main)
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	if err := i.reset(ctx, head); err != nil {
		return "", err
	}
	if _, err := forgeGit(ctx, i.place, "merge", "-q", "--no-ff", "--no-edit", "-m", "Merge main into the branch", main); err != nil {
		return "", i.conflict(ctx, worktree, main, err)
	}
	h, err := forgeGit(ctx, i.place, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	if h != head {
		if _, err := forgeGit(ctx, worktree, "merge", "-q", "--ff-only", h); err != nil {
			return "", fmt.Errorf("orchestrate: move the child's branch forward to %s: %w", short(h), err)
		}
	}
	data, _ := json.Marshal(map[string]string{"head": head, "main": main, "commit": h})
	why := fmt.Sprintf("the harness merged main at %s into the branch at %s: %s is what the gate measures", short(main), short(head), short(h))
	if h == head {
		why = fmt.Sprintf("the branch at %s has main at %s already: it is what the gate measures", short(head), short(main))
	}
	if _, err := l.Record(Decision{Kind: "branch.updated", Child: child, Why: why, Data: data}); err != nil {
		return "", err
	}
	return h, nil
}

// conflict aborts the merge in the place and makes it again in the child's
// worktree with the markers.
func (i *Integration) conflict(ctx context.Context, worktree, main string, mergeErr error) error {
	out, err := forgeGit(ctx, i.place, "diff", "--name-only", "--diff-filter=U")
	if err != nil || strings.TrimSpace(out) == "" {
		_ = i.reset(ctx, "HEAD")
		return fmt.Errorf("orchestrate: merging main into the branch failed, and not on a conflict: %w", mergeErr)
	}
	c := &Conflict{Paths: strings.Split(out, "\n")}
	if _, err := forgeGit(ctx, i.place, "merge", "--abort"); err != nil {
		return errors.Join(c, fmt.Errorf("orchestrate: the integration place could not abort the merge: %w", err))
	}
	// The same merge in the child's worktree, failing on the same paths, is
	// what leaves the markers for the child; its error is the conflict.
	_, _ = forgeGit(ctx, worktree, "merge", "-q", "--no-ff", "--no-edit", main)
	return c
}

// squash makes, in the place, one commit over old with measured's tree, and
// returns it. The place is left at that commit; nothing else moves.
func (i *Integration) squash(ctx context.Context, old, measured, message string) (string, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if err := i.reset(ctx, old); err != nil {
		return "", err
	}
	if _, err := forgeGit(ctx, i.place, "merge", "-q", "--squash", measured); err != nil {
		_ = i.reset(ctx, old)
		return "", fmt.Errorf("orchestrate: squash %s over %s: %w", short(measured), short(old), err)
	}
	if _, err := forgeGit(ctx, i.place, "commit", "-q", "--no-verify", "-m", message); err != nil {
		_ = i.reset(ctx, old)
		return "", fmt.Errorf("orchestrate: commit the squash of %s: %w", short(measured), err)
	}
	return forgeGit(ctx, i.place, "rev-parse", "HEAD")
}

// reset puts the place, detached, at commit, with nothing left over from a
// run before. It is the daemon's own tree, and no one else's work is in it.
func (i *Integration) reset(ctx context.Context, commit string) error {
	if _, err := forgeGit(ctx, i.place, "reset", "-q", "--hard"); err != nil {
		return fmt.Errorf("orchestrate: clean the integration place: %w", err)
	}
	if _, err := forgeGit(ctx, i.place, "clean", "-q", "-fd"); err != nil {
		return fmt.Errorf("orchestrate: clean the integration place: %w", err)
	}
	if _, err := forgeGit(ctx, i.place, "switch", "-q", "--detach", commit); err != nil {
		return fmt.Errorf("orchestrate: put the integration place at %s: %w", short(commit), err)
	}
	return nil
}
