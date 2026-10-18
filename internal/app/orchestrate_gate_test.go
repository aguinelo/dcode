package app

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/sandbox"
)

// Spec: orchestration .p §7.4 — the daemon gives a project's gate the sandbox
// a session there gets, and a configuration that would run it unconfined —
// backend none — is a gate that does not run, never one that runs without a
// boundary.
func TestTheGateGetsTheProjectsSandboxAndNeverNone(t *testing.T) {
	root := t.TempDir()
	daemon := func(backend string) *Daemon {
		d := NewDaemon(DaemonOptions{SocketPath: filepath.Join(t.TempDir(), "d.sock"), Log: func(string) {},
			Base: Options{Backend: backend}})
		t.Cleanup(d.manager.CloseAll)
		return d
	}

	if sb, err := daemon(sandbox.BackendNone).gateSandbox(root)(); err == nil || !errors.Is(err, sandbox.ErrUnavailable) {
		t.Errorf("backend none gave the gate %v (err %v); it must be unavailable", sb, err)
	}

	sb, err := daemon(sandbox.BackendAuto).gateSandbox(root)()
	if err != nil {
		t.Fatalf("the gate runs inside the sandbox, and this machine has none: %v", err)
	}
	if n := sb.Name(); n != sandbox.BackendSeatbelt && n != sandbox.BackendBubblewrap {
		t.Errorf("the gate's sandbox is %q, not the platform's backend", n)
	}
}
