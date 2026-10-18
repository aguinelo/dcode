package orchestrate

// The gate's boundary. A gate command is configuration a person reviewed,
// which is why it may run at all, and "reviewed" is not "unconfined": it runs
// in the same sandbox the agent's bash runs in, writable only in the
// measurement's own throwaway directory.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §7.4.

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/sandbox"
)

// gateMode is the boundary every gate runs under, whatever mode the sessions
// of the project are in: workspace-write, with the throwaway directory as the
// workspace. A gate builds and tests, so it writes; it writes nowhere the
// measurement does not own, and the toolchain caches the policy already grants
// are the only exception. The network follows the sandbox's own decision,
// which is closed unless configuration opens it.
const gateMode = policy.ModeWorkspaceWrite

// GateSandbox makes the boundary one measurement runs in. It is asked once per
// measurement, so a configuration that changed is the one in force.
type GateSandbox func() (sandbox.Sandbox, error)

// DefaultGateSandbox is the boundary of a gate nobody wired one for: the
// platform's backend, the toolchain caches of this process's environment, and
// no network — nil is denied.
func DefaultGateSandbox() (sandbox.Sandbox, error) {
	return sandbox.New(sandbox.Config{Backend: sandbox.BackendAuto, Scratch: GateScratch(os.Getenv)}, gateMode)
}

// GateScratch is the toolchain caches a gate may write, read from env: those
// sandbox.Scratch names, less the daemon's own temporary directory. A
// measurement has one of its own, inside its throwaway directory, and the
// daemon's is where other things live.
func GateScratch(env func(string) string) []string {
	if env == nil {
		return nil
	}
	return sandbox.Scratch(func(k string) string {
		if k == "TMPDIR" {
			return ""
		}
		return env(k)
	})
}

// boundary is the sandbox this measurement runs in, established before
// anything is made. None is an error that says why, never a gate that runs
// unconfined.
func (g Gate) boundary() (sandbox.Sandbox, error) {
	build := g.Sandbox
	if build == nil {
		build = DefaultGateSandbox
	}
	sb, err := build()
	if err == nil && sb == nil {
		err = fmt.Errorf("%w: no sandbox was made", sandbox.ErrUnavailable)
	}
	if err == nil {
		err = sb.Available()
	}
	if err != nil {
		if !errors.Is(err, sandbox.ErrUnavailable) {
			err = fmt.Errorf("%w: %w", sandbox.ErrUnavailable, err)
		}
		return nil, fmt.Errorf("orchestrate: the gate runs only inside the sandbox the agent's commands run in, "+
			"and none could be established here, so nothing was measured and nothing merges: %w", err)
	}
	return sb, nil
}

// shQuote is s as one word of sh.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
