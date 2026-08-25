package sandbox

import (
	"context"
	"os/exec"
	"sync"
	"testing"

	"github.com/aguinelo/dcode/internal/policy"
)

// recordingSandbox remembers the mode each Wrap was asked for.
type recordingSandbox struct {
	mu    sync.Mutex
	modes []policy.SandboxMode
}

func (r *recordingSandbox) Name() string     { return BackendNone }
func (r *recordingSandbox) Available() error { return nil }

func (r *recordingSandbox) Wrap(ctx context.Context, workdir, command string, mode policy.SandboxMode) (*exec.Cmd, error) {
	r.mu.Lock()
	r.modes = append(r.modes, mode)
	r.mu.Unlock()
	return noneSandbox{}.Wrap(ctx, workdir, command, mode)
}

func (r *recordingSandbox) seen() []policy.SandboxMode {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]policy.SandboxMode(nil), r.modes...)
}

// TestTheRunnerAsksTheModeOncePerCommand is the defect, at its root.
//
// The mode used to be a value copied when the runner was built, so `/mode auto`
// changed the policy's answer and nothing else: the verdict said allow, the
// badge said auto, and a write outside the workspace still came back EPERM
// because the OS was enforcing the boundary the session had STARTED with. A
// mode whose entire promise is "no boundary" left one standing.
func TestTheRunnerAsksTheModeOncePerCommand(t *testing.T) {
	current := policy.ModeWorkspaceWrite
	rec := &recordingSandbox{}
	r := Runner{Sandbox: rec, Mode: func() policy.SandboxMode { return current }}

	if _, _, err := r.Run(context.Background(), t.TempDir(), "true"); err != nil {
		t.Fatalf("first run: %v", err)
	}
	current = policy.ModeFullAccess
	if _, _, err := r.Run(context.Background(), t.TempDir(), "true"); err != nil {
		t.Fatalf("second run: %v", err)
	}

	got := rec.seen()
	if len(got) != 2 {
		t.Fatalf("wrapped %d commands, want 2", len(got))
	}
	if got[0] != policy.ModeWorkspaceWrite {
		t.Errorf("first command ran under %q, want workspace-write", got[0])
	}
	if got[1] != policy.ModeFullAccess {
		t.Errorf("second command ran under %q, want full-access — the switch never reached the sandbox", got[1])
	}
}

// TestARunnerWithNoModeFailsClosed: RN-3, at the one place that could quietly
// become permissive by omission.
//
// A runner nobody gave a mode to is a runner whose boundary nobody decided, and
// the reading of "nobody said" that this repository holds is never "allow".
func TestARunnerWithNoModeFailsClosed(t *testing.T) {
	rec := &recordingSandbox{}
	r := Runner{Sandbox: rec}

	if _, _, err := r.Run(context.Background(), t.TempDir(), "true"); err != nil {
		t.Fatalf("run: %v", err)
	}
	if got := rec.seen(); len(got) != 1 || got[0] != policy.ModeReadOnly {
		t.Errorf("a runner with no mode ran under %v, want [read-only]", got)
	}
}

// TestFixedNeverMoves covers the other half of the pair.
//
// A one-shot command run by the daemon itself — a done criterion — runs under
// the mode it was configured with, whatever the session has since switched to.
func TestFixedNeverMoves(t *testing.T) {
	src := Fixed(policy.ModeReadOnly)
	if got := src(); got != policy.ModeReadOnly {
		t.Errorf("Fixed returned %q", got)
	}
	if got := src(); got != policy.ModeReadOnly {
		t.Errorf("Fixed changed between calls: %q", got)
	}
}
