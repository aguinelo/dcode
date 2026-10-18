package orchestrate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/sandbox"
)

// boundaryHere is the sandbox the agent's commands would run in on this
// machine. A test of the gate's boundary that cannot get one fails and says
// why: a skipped boundary test reads as a passing one.
func boundaryHere(t *testing.T) {
	t.Helper()
	if _, err := sandbox.New(sandbox.Config{Backend: sandbox.BackendAuto}, policy.ModeWorkspaceWrite); err != nil {
		t.Fatalf("the gate runs inside the sandbox, and this machine has none: %v", err)
	}
}

// Spec: orchestration .p §7.4 — the gate runs inside the sandbox the agent's
// bash runs in, writable only in its throwaway directory: a gate that writes to
// the person's repository or to the home is red, and nothing lands there.
func TestTheGateCannotWriteOutsideItsWorktree(t *testing.T) {
	boundaryHere(t)
	repo, _, second := gateRepo(t)
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	for name, target := range map[string]string{
		"the repository": filepath.Join(repo, "escaped-by-gate"),
		"the home":       filepath.Join(home, fmt.Sprintf(".dcode-gate-escape-%d", os.Getpid())),
	} {
		t.Run(name, func(t *testing.T) {
			t.Cleanup(func() { _ = os.Remove(target) })
			g := Gate{Command: fmt.Sprintf("echo escaped > '%s'", target), Timeout: time.Minute}

			r := measure(t, g, repo, second)

			if r.Result != GateRed {
				t.Errorf("a gate that writes to %s is %s (exit %d), not red; output:\n%s",
					name, r.Result, r.ExitCode, r.OutputTail)
			}
			if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("the gate wrote %s outside its worktree (%v)", target, err)
			}
			if r.Sandbox == "" || r.SandboxMode != string(policy.ModeWorkspaceWrite) {
				t.Errorf("the measurement names sandbox %q in mode %q; it must name the boundary it ran in",
					r.Sandbox, r.SandboxMode)
			}
		})
	}
}

// Spec: orchestration .p §7.4 — inside the boundary an ordinary gate still
// works: it writes in its tree and in a temporary directory of its own.
func TestAGateWritesItsTreeAndItsOwnTemporaryDirectory(t *testing.T) {
	boundaryHere(t)
	repo, _, second := gateRepo(t)
	person := os.TempDir()
	g := Gate{Command: `echo built > out.txt && mkdir -p "$TMPDIR/stage" && echo staged > "$TMPDIR/stage/x" && ` +
		`cat out.txt "$TMPDIR/stage/x" && echo "tmp=$TMPDIR"`, Timeout: time.Minute}

	r := measure(t, g, repo, second)

	if r.Result != GateGreen {
		t.Fatalf("an ordinary gate is %s (exit %d) inside the sandbox; output:\n%s", r.Result, r.ExitCode, r.OutputTail)
	}
	if r.Sandbox == "" || r.SandboxMode != string(policy.ModeWorkspaceWrite) {
		t.Errorf("the measurement names sandbox %q in mode %q", r.Sandbox, r.SandboxMode)
	}
	i := strings.LastIndex(r.OutputTail, "tmp=")
	if i < 0 {
		t.Fatalf("no TMPDIR in the output:\n%s", r.OutputTail)
	}
	if tmp := strings.TrimSpace(r.OutputTail[i+len("tmp="):]); tmp == person || tmp == "" {
		t.Errorf("the gate's TMPDIR is %q, the daemon's own; a measurement has a temporary directory of its own", tmp)
	}
}

// unavailable is a sandbox this machine cannot establish.
type unavailable struct{ wrapped *bool }

func (unavailable) Name() string { return "stand-in" }
func (unavailable) Available() error {
	return fmt.Errorf("%w: the stand-in cannot confine anything", sandbox.ErrUnavailable)
}
func (u unavailable) Wrap(ctx context.Context, workdir, command string, _ policy.SandboxMode) (*exec.Cmd, error) {
	*u.wrapped = true
	return exec.CommandContext(ctx, "sh", "-c", command), nil
}

// Spec: orchestration .p §7.4 — without the sandbox the gate does not run
// unconfined: nothing is measured, and the error says the sandbox is why.
func TestWithoutTheSandboxTheGateDoesNotRunAndSaysWhy(t *testing.T) {
	repo, _, second := gateRepo(t)
	marker := filepath.Join(t.TempDir(), "ran")
	cmd := fmt.Sprintf("touch '%s'", marker)
	wrapped := false
	for name, s := range map[string]GateSandbox{
		"one that is not available": func() (sandbox.Sandbox, error) { return unavailable{&wrapped}, nil },
		"one that cannot be made": func() (sandbox.Sandbox, error) {
			return nil, fmt.Errorf("%w: no backend here", sandbox.ErrUnavailable)
		},
		"none at all": func() (sandbox.Sandbox, error) { return nil, nil },
	} {
		t.Run(name, func(t *testing.T) {
			r, err := Gate{Command: cmd, Timeout: time.Minute, Sandbox: s}.Measure(context.Background(), repo, second)
			if err == nil {
				t.Fatalf("a gate without a sandbox measured %+v", r)
			}
			if !errors.Is(err, sandbox.ErrUnavailable) || !strings.Contains(err.Error(), "sandbox") {
				t.Errorf("the error does not say the sandbox is why: %v", err)
			}
			if r.Result != "" {
				t.Errorf("a gate that did not run has result %q", r.Result)
			}
		})
	}
	if wrapped {
		t.Error("the command was wrapped by a sandbox that said it was unavailable")
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the gate ran without a sandbox (%v)", err)
	}
	if n := worktrees(t, repo); n != 1 {
		t.Errorf("%d worktrees after a gate that never ran; a throwaway tree was left", n)
	}
}
