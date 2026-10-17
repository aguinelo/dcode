package orchestrate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// ResolveProject answers which project dir belongs to: the root of the main
// worktree, links resolved, and Project.ID, the first 24 hex of the sha256 of
// that root. Every worktree of one tree, a subdirectory of it, and a link to
// it are the same project, because they share one git common directory.
//
// The common directory's parent is the root; a bare repository has no working
// tree of its own, and its common directory is the root.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §5.2, §9.2.
func ResolveProject(ctx context.Context, dir string) (root, id string, err error) {
	if !filepath.IsAbs(dir) {
		return "", "", fmt.Errorf("orchestrate: %q is not an absolute path", dir)
	}
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--path-format=absolute", "--git-common-dir")
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(), "GIT_OPTIONAL_LOCKS=0", "LC_ALL=C")
	out, err := cmd.Output()
	if err != nil {
		said := err.Error()
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			said = strings.TrimSpace(string(ee.Stderr))
		}
		return "", "", fmt.Errorf("orchestrate: %s is not in a git repository: %s", dir, said)
	}
	common := filepath.Clean(strings.TrimSpace(string(out)))
	root = common
	if filepath.Base(common) == ".git" {
		root = filepath.Dir(common)
	}
	if root, err = filepath.EvalSymlinks(root); err != nil {
		return "", "", fmt.Errorf("orchestrate: resolve the root of %s: %w", dir, err)
	}
	sum := sha256.Sum256([]byte(root))
	return root, hex.EncodeToString(sum[:])[:24], nil
}
