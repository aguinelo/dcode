package orchestrate

import (
	"fmt"
	"strings"

	"github.com/aguinelo/dcode/internal/config"
)

// The [orchestrator] keys step 5 reads, besides orchestrator.dir (dir.go).
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.config.spec.md §1, §2, §3.

// Enabled is orchestrator.enabled, false unless declared: the orchestrator is
// opt-in in this slice. A value that is not a boolean is an error naming the
// key and the value, never read as the default.
func Enabled(r config.Resolved) (bool, error) {
	v := strings.TrimSpace(r.String("orchestrator.enabled", ""))
	switch strings.ToLower(v) {
	case "":
		return false, nil
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	}
	return false, fmt.Errorf("orchestrator.enabled must be a boolean, got %q", v)
}

// Profile is orchestrator.profile: the models.toml profile the orchestrator
// runs on. Empty is model.name; whoever builds the session checks that a named
// profile exists.
func Profile(r config.Resolved) string {
	return strings.TrimSpace(r.String("orchestrator.profile", ""))
}

// MainBranch is orchestrator.main_branch. Empty is the branch the repository
// says is its main; Open checks that a named one exists.
func MainBranch(r config.Resolved) string {
	return strings.TrimSpace(r.String("orchestrator.main_branch", ""))
}
