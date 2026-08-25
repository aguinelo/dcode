//go:build unix

package sandbox

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/policy"
)

// Every other background test uses noneSandbox, which is the development
// backend: a plain shell with no confinement. So the whole feature was proven
// against the one path that does not wrap the command at all.
//
// This one asks the machine for its real backend — seatbelt on macOS,
// bubblewrap on Linux — and drives the case the feature exists for: start
// something that does not end, read what it printed, stop it, and confirm it is
// gone. It skips where no backend can be established, because a suite that
// fails on a machine without bubblewrap is a suite people switch off.
//
// It matters more than the count suggests. Setpgid and a group kill are exactly
// the kind of thing that works against `sh -c` and quietly does not against a
// command wrapped in sandbox-exec, and nothing else here would have noticed.
func TestTheRealBoundaryStartsAndStopsABackgroundProcess(t *testing.T) {
	sb, err := New(Config{Backend: BackendAuto}, policy.ModeWorkspaceWrite)
	if err != nil {
		t.Skipf("no real sandbox here: %v", err)
	}
	t.Logf("backend = %s", sb.Name())

	dir := t.TempDir()
	p, err := (Runner{Sandbox: sb, Mode: Fixed(policy.ModeWorkspaceWrite)}).
		Start(context.Background(), dir, "echo up; sleep 30")
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) && !strings.Contains(p.Output(), "up") {
		time.Sleep(50 * time.Millisecond)
	}
	if !strings.Contains(p.Output(), "up") {
		t.Fatalf("no output from a confined background process: %q", p.Output())
	}
	if _, done := p.Exited(); done {
		t.Fatal("it finished; Start waited for it")
	}
	pid := p.cmd.Process.Pid
	p.Stop()

	deadline = time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if err := alive(pid); err != nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Error("the confined process outlived Stop")
}

// alive reports whether the pid still exists. Signal 0 asks without delivering.
func alive(pid int) error {
	pr, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return pr.Signal(os.Signal(nil))
}
