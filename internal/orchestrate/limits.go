package orchestrate

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/policy"
)

// The [orchestrator] keys a project reads when it opens: how many children, on
// which profiles, on which branches, how long a closed one stays listed, the
// token ceiling, how many returns before a child fails, and how long reports wait.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.config.spec.md §1, §2.

const (
	defaultMaxChildren     = 4
	defaultBranchPrefix    = "dcode/"
	defaultTokenCeiling    = 20_000_000
	defaultClosedRetention = time.Hour
	defaultMaxReturns      = 3
	defaultReportDebounce  = 2 * time.Second
	// sessionsBesideChildren are the orchestrator and one reviewer, which a
	// daemon's session ceiling has to leave room for (.config §2).
	sessionsBesideChildren = 2
)

// Limits are what a project's dispatch is held to, read from the
// configuration chain at its root when it opens.
type Limits struct {
	MaxChildren int
	// ChildProfiles is the allow-list of the child role; empty is only the
	// orchestrator's own model.
	ChildProfiles []string
	BranchPrefix  string
	TokenCeiling  int64
	// MaxReturns is orchestrator.max_returns: how many times the pipeline
	// hands a child its work back before the child fails (.p §6).
	MaxReturns int
	// ReportDebounce is orchestrator.report_debounce: how long the inbox
	// waits for more reports before it starts the orchestrator's turn.
	ReportDebounce time.Duration
}

// ReadLimits reads the step-6 keys of r. maxSessions is the daemon's session
// ceiling, zero when unknown. A value that does not read is an error naming
// the key and the value, never the default in its place.
func ReadLimits(ctx context.Context, r config.Resolved, maxSessions int) (Limits, error) {
	l := Limits{MaxChildren: defaultMaxChildren, BranchPrefix: defaultBranchPrefix, TokenCeiling: defaultTokenCeiling,
		MaxReturns: defaultMaxReturns, ReportDebounce: defaultReportDebounce}
	if v := strings.TrimSpace(r.String("orchestrator.max_children", "")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 16 {
			return Limits{}, fmt.Errorf("orchestrator.max_children must be an integer from 1 to 16, got %q", v)
		}
		l.MaxChildren = n
	}
	if maxSessions > 0 && l.MaxChildren >= maxSessions-sessionsBesideChildren {
		return Limits{}, fmt.Errorf("orchestrator.max_children is %d, and must be less than the daemon's %d sessions minus %d (the orchestrator and a reviewer)",
			l.MaxChildren, maxSessions, sessionsBesideChildren)
	}
	l.ChildProfiles = policy.SplitList(r.String("orchestrator.child_profiles", ""))
	if v := r.String("orchestrator.branch_prefix", ""); v != "" {
		cmd := exec.CommandContext(ctx, "git", "check-ref-format", "--branch", v+"x")
		cmd.Env = gitEnv(cmd.Environ())
		if err := cmd.Run(); err != nil {
			return Limits{}, fmt.Errorf("orchestrator.branch_prefix is %q, which git does not take as the start of a branch name", v)
		}
		l.BranchPrefix = v
	}
	if v := strings.TrimSpace(r.String("orchestrator.token_ceiling", "")); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 100_000 || n > 1_000_000_000 {
			return Limits{}, fmt.Errorf("orchestrator.token_ceiling must be an integer from 100000 to 1000000000, got %q", v)
		}
		l.TokenCeiling = n
	}
	if v := strings.TrimSpace(r.String("orchestrator.max_returns", "")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 10 {
			return Limits{}, fmt.Errorf("orchestrator.max_returns must be an integer from 1 to 10, got %q", v)
		}
		l.MaxReturns = n
	}
	if v := strings.TrimSpace(r.String("orchestrator.report_debounce", "")); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d < 0 || d > time.Minute {
			return Limits{}, fmt.Errorf("orchestrator.report_debounce must be a duration from 0s to 1m, got %q", v)
		}
		l.ReportDebounce = d
	}
	return l, nil
}

// ClosedRetention is orchestrator.closed_retention: how long a closed child
// stays in the answer of /children, from 0s to 24h, an hour unless declared.
// Read where the daemon starts, beside orchestrator.dir.
func ClosedRetention(r config.Resolved) (time.Duration, error) {
	v := strings.TrimSpace(r.String("orchestrator.closed_retention", ""))
	if v == "" {
		return defaultClosedRetention, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil || d < 0 || d > 24*time.Hour {
		return 0, fmt.Errorf("orchestrator.closed_retention must be a duration from 0s to 24h, got %q", v)
	}
	return d, nil
}
