package orchestrate

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// ResolveProject is the project a folder belongs to: the root of the tree's
// main worktree, links resolved, and Project.ID, 24 hex of its sha256.
//
// Every way into the same tree — the root, a subfolder, a link to it, a linked
// worktree — is the same project, because the root is read from git's common
// directory rather than from the folder that was named. A folder in no git
// tree is an error: a project is a tree.
//
// POST /v1/projects and `dcode decisions` resolve through this, so the log a
// reader opens is the one the orchestrator writes.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.config.spec.md (Project.ID).
func ResolveProject(ctx context.Context, dir string) (root, id string, err error) {
	if _, err := exec.LookPath("git"); err != nil {
		return "", "", errors.New("orchestrate: git is not on the PATH, and a project is resolved by it")
	}
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--git-common-dir")
	cmd.Dir = dir
	cmd.Env = gitEnv(cmd.Environ())
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", "", fmt.Errorf("orchestrate: %s is not a project: git does not find a tree there (%s)", dir, msg)
	}
	common := strings.TrimSpace(string(out))
	if !filepath.IsAbs(common) {
		common = filepath.Join(dir, common)
	}
	common = filepath.Clean(common)
	root = common // a bare repository has no working tree of its own
	if filepath.Base(common) == ".git" {
		root = filepath.Dir(common)
	}
	if root, err = filepath.EvalSymlinks(root); err != nil {
		return "", "", fmt.Errorf("orchestrate: resolve the root of %s: %w", dir, err)
	}
	sum := sha256.Sum256([]byte(root))
	return root, hex.EncodeToString(sum[:])[:24], nil
}

// gitEnv drops what would make git answer for another tree than the folder's.
func gitEnv(env []string) []string {
	drop := map[string]bool{"GIT_DIR": true, "GIT_WORK_TREE": true, "GIT_COMMON_DIR": true, "GIT_INDEX_FILE": true}
	out := env[:0:0]
	for _, kv := range env {
		if k, _, _ := strings.Cut(kv, "="); !drop[k] {
			out = append(out, kv)
		}
	}
	return out
}
