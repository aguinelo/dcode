package orchestrate

import (
	"fmt"
	"path/filepath"

	"github.com/aguinelo/dcode/internal/config"
)

// Dir is orchestrator.dir: where every project's state, decision log and
// worktrees live, <state>/orchestrate unless declared.
//
// A relative value is an error naming the key and the value, never resolved
// against whatever directory the daemon happened to start in. Without a state
// root there is no default to fall to, and that is an error too.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.config.spec.md §1.
func Dir(r config.Resolved, env func(string) string) (string, error) {
	if v := r.String("orchestrator.dir", ""); v != "" {
		if !filepath.IsAbs(v) {
			return "", fmt.Errorf("orchestrator.dir must be an absolute path, got %q", v)
		}
		return filepath.Clean(v), nil
	}
	roots, err := config.DiscoverRoots(env)
	if err != nil {
		return "", fmt.Errorf("orchestrator.dir is not set and its default needs the state root: %w", err)
	}
	return filepath.Join(roots.State, "orchestrate"), nil
}
