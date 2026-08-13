//go:build unix

package sandbox

import (
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/policy"
)

func backgroundRunner() Runner {
	return Runner{Sandbox: noneSandbox{}, Mode: policy.ModeFullAccess}
}

// waitFor polls until cond holds or the budget runs out. Real processes are
// asynchronous and a fixed sleep is either flaky or slow; this is neither.
func waitFor(t *testing.T, why string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", why)
}

// The point of the whole feature: the call returns and the command carries on.
func TestStartReturnsWhileTheCommandIsStillRunning(t *testing.T) {
	p, err := backgroundRunner().Start(context.Background(), t.TempDir(), "sleep 30")
	if err != nil {
		t.Fatal(err)
	}
	defer p.Stop()

	if _, done := p.Exited(); done {
		t.Fatal("Start waited for the command, which is the thing it must not do")
	}
}

// A server started in one turn is meant to be there in the next. Binding the
// command to the turn's context would kill it the moment the turn ended, which
// would leave the flag technically present and useless.
func TestACommandOutlivesTheTurnThatStartedIt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p, err := backgroundRunner().Start(ctx, t.TempDir(), "sleep 30")
	if err != nil {
		t.Fatal(err)
	}
	defer p.Stop()

	cancel()
	time.Sleep(100 * time.Millisecond)

	if _, done := p.Exited(); done {
		t.Error("the command died with the turn, so nothing could survive to be read")
	}
}

// A turn already abandoned gets nothing started on its behalf.
func TestStartRefusesACancelledTurn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := backgroundRunner().Start(ctx, t.TempDir(), "sleep 30"); err == nil {
		t.Error("a cancelled turn still started a process")
	}
}

func TestOutputAccumulatesWhileItRuns(t *testing.T) {
	p, err := backgroundRunner().Start(context.Background(), t.TempDir(),
		"echo first; sleep 30")
	if err != nil {
		t.Fatal(err)
	}
	defer p.Stop()

	waitFor(t, "the first line", func() bool {
		return strings.Contains(p.Output(), "first")
	})
}

func TestExitedReportsTheCodeOnceItFinishes(t *testing.T) {
	p, err := backgroundRunner().Start(context.Background(), t.TempDir(), "exit 3")
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the command to finish", func() bool {
		_, done := p.Exited()
		return done
	})
	if code, _ := p.Exited(); code != 3 {
		t.Errorf("exit code came back as %d, want 3", code)
	}
}

// The orphan test, and the reason Stop kills the group rather than the process.
//
// `sh -c "sleep 30"` is the shape of every real case: the process that was
// launched is not the process holding the resource. Killing only the one whose
// pid we know leaves the other alive, unnamed and unreachable — which is worse
// than a frozen session, because a frozen session is visible.
func TestStopKillsWhatTheCommandStartedToo(t *testing.T) {
	dir := t.TempDir()
	pidFile := dir + "/child.pid"

	p, err := backgroundRunner().Start(context.Background(), dir,
		"sh -c 'sleep 30 & echo $! > "+pidFile+"; wait'")
	if err != nil {
		t.Fatal(err)
	}

	var pid int
	waitFor(t, "the grandchild to report its pid", func() bool {
		b, err := os.ReadFile(pidFile)
		if err != nil {
			return false
		}
		_, err = fmt.Sscan(strings.TrimSpace(string(b)), &pid)
		return err == nil && pid > 0
	})

	p.Stop()

	waitFor(t, "the grandchild to die with the group", func() bool {
		// Signal 0 tests for existence without delivering anything.
		return syscall.Kill(pid, 0) != nil
	})
}

// Stopping something that already finished is a no-op, not a panic. A model
// that stops a crashed server is doing the sensible thing.
func TestStoppingAFinishedCommandDoesNothing(t *testing.T) {
	p, err := backgroundRunner().Start(context.Background(), t.TempDir(), "true")
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the command to finish", func() bool {
		_, done := p.Exited()
		return done
	})
	p.Stop()
	p.Stop()
}

func TestStartWithoutASandboxIsUnavailable(t *testing.T) {
	if _, err := (Runner{}).Start(context.Background(), t.TempDir(), "true"); err == nil {
		t.Error("a command started with no boundary to confine it")
	}
}
