package orchestrate

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"
)

// A child's worktree and branch, made and taken away by the harness, and the
// commit it makes at the end of each of the child's turns. The child never
// runs git that writes (.p §10): this file is where that git lives.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §7.1, §8, §9.1.

const (
	childrenDir = "children"
	// slugMax keeps a branch name a person can read in a list.
	slugMax = 40
	// branchID is how much of the child's id ends its branch: the last six,
	// the id's random part, since the first six are its clock and repeat
	// across a batch.
	branchID = 6
)

// wtGit runs git in dir, its environment cleared of what would point it at
// another tree, and gives back stdout trimmed, or an error with stderr.
func wtGit(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = gitEnv(cmd.Environ())
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}

// slug is title as a branch segment: lower case letters and digits, dashes
// between, at most slugMax. A title with none of either is "child".
func slug(title string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(title) {
		if r < 0x80 && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			dash = false
			continue
		}
		dash = true
	}
	s := b.String()
	if len(s) > slugMax {
		s = strings.TrimRight(s[:slugMax], "-")
	}
	if s == "" {
		return "child"
	}
	return s
}

// branchFor is <prefix><slug of title>-<last six of the id> (.config §2).
func branchFor(prefix, title, id string) string {
	tail := id
	if len(tail) > branchID {
		tail = tail[len(tail)-branchID:]
	}
	return prefix + slug(title) + "-" + tail
}

// worktreeOf is where child id's worktree lives: under orchestrator.dir, never
// in the person's tree.
func worktreeOf(dir, projectID, childID string) string {
	return filepath.Join(dir, projectID, childrenDir, childID)
}

// mainTip is the commit main points at now.
func mainTip(ctx context.Context, root, main string) (string, error) {
	return wtGit(ctx, root, "rev-parse", "--verify", "refs/heads/"+main+"^{commit}")
}

// hooksPath is core.hooksPath as the root reads it; unset is empty.
func hooksPath(ctx context.Context, root string) string {
	v, err := wtGit(ctx, root, "config", "--get", "core.hooksPath")
	if err != nil {
		return ""
	}
	return v
}

// addWorktree makes branch at base and checks it out at path.
func addWorktree(ctx context.Context, root, path, branch, base string) error {
	_, err := wtGit(ctx, root, "worktree", "add", "-b", branch, path, base)
	return err
}

// removeWorktree takes the worktree away, what was not committed with it. The
// branch stays: it holds the harness's commits, and the log names it.
func removeWorktree(ctx context.Context, root, path string) error {
	_, err := wtGit(ctx, root, "worktree", "remove", "--force", path)
	return err
}

// commitTurn commits what the child's turn changed in its worktree, adding the
// paths by name — never -A — with message as the message. Nothing changed is
// no commit and no error; head is then empty.
func commitTurn(ctx context.Context, worktree, message string) (head string, paths []string, err error) {
	paths, err = Uncommitted(ctx, worktree)
	if err != nil {
		return "", nil, err
	}
	if len(paths) == 0 {
		return "", nil, nil
	}
	if _, err := wtGit(ctx, worktree, append([]string{"add", "--"}, paths...)...); err != nil {
		return "", paths, err
	}
	if _, err := wtGit(ctx, worktree, "commit", "-q", "-m", message); err != nil {
		return "", paths, err
	}
	head, err = wtGit(ctx, worktree, "rev-parse", "HEAD")
	return head, paths, err
}
