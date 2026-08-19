package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/policy"
)

// Backend names appear in configuration and in error messages, so they are part
// of the user-facing contract rather than an internal label.
func TestBackendNames(t *testing.T) {
	for _, tc := range []struct {
		s    Sandbox
		want string
	}{
		{&seatbelt{}, BackendSeatbelt},
		{&bubblewrap{}, BackendBubblewrap},
		{noneSandbox{}, BackendNone},
	} {
		if got := tc.s.Name(); got != tc.want {
			t.Errorf("got %q want %q", got, tc.want)
		}
	}
}

// Both backends must build a command without executing it, on any platform.
// Construction is where the boundary is decided, so it is worth asserting even
// where the mechanism itself cannot run.
func TestWrapBuildsACommandOnAnyPlatform(t *testing.T) {
	ws := t.TempDir()
	ctx := context.Background()

	sb := &seatbelt{bin: "sandbox-exec"}
	cmd, err := sb.Wrap(ctx, ws, "echo hi", policy.ModeWorkspaceWrite)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(cmd.Args, " "), "-p") {
		t.Errorf("the profile must be passed to the binary: %v", cmd.Args)
	}
	if cmd.Dir != ws {
		t.Errorf("the command must run in the workspace, got %q", cmd.Dir)
	}

	bw := &bubblewrap{bin: "bwrap"}
	cmd, err = bw.Wrap(ctx, ws, "echo hi", policy.ModeWorkspaceWrite)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(cmd.Args, " ")
	if !strings.Contains(joined, "--unshare-net") {
		t.Errorf("network should be removed by default: %v", cmd.Args)
	}
	if !strings.Contains(joined, "echo hi") {
		t.Errorf("the command must survive wrapping: %v", cmd.Args)
	}
}

func TestWrapRejectsABadModeBeforeExecuting(t *testing.T) {
	ctx := context.Background()
	if _, err := (&seatbelt{bin: "x"}).Wrap(ctx, "/w", "echo", policy.SandboxMode("bad")); err == nil {
		t.Error("seatbelt must reject an unknown mode")
	}
	if _, err := (&bubblewrap{bin: "x"}).Wrap(ctx, "/w", "echo", policy.SandboxMode("bad")); err == nil {
		t.Error("bubblewrap must reject an unknown mode")
	}
}

// full-access is the one mode that runs unconfined, and only when asked for
// explicitly.
func TestNoneSandboxRunsUnconfined(t *testing.T) {
	s := noneSandbox{}
	if err := s.Available(); err != nil {
		t.Fatalf("the no-op backend is always available: %v", err)
	}
	ws := t.TempDir()
	out, code, err := Runner{Sandbox: s, Mode: policy.ModeFullAccess}.
		Run(context.Background(), ws, "echo unconfined")
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v out=%q", code, err, out)
	}
	if !strings.Contains(out, "unconfined") {
		t.Errorf("got %q", out)
	}
}

func TestFullAccessCanWriteOutsideTheWorkspace(t *testing.T) {
	// The mode exists precisely to lift the boundary, so this is the assertion
	// that the mode is not silently doing something else.
	root := t.TempDir()
	ws := filepath.Join(root, "ws")
	if err := os.MkdirAll(ws, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "written.txt")

	r := Runner{Sandbox: noneSandbox{}, Mode: policy.ModeFullAccess}
	if _, code, err := r.Run(context.Background(), ws, "echo x > "+shellQuote(outside)); err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Errorf("full-access should have written outside: %v", err)
	}
}

func TestRunSurfacesAWrapFailure(t *testing.T) {
	// A mode the backend cannot express must stop before anything executes.
	_, code, err := Runner{Sandbox: &seatbelt{bin: "x"}, Mode: policy.SandboxMode("bad")}.
		Run(context.Background(), t.TempDir(), "echo hi")
	if err == nil {
		t.Fatal("a wrap failure must surface")
	}
	if code != -1 {
		t.Errorf("a command that never ran has no exit code, got %d", code)
	}
}

func TestRunSurfacesAMissingBinary(t *testing.T) {
	_, code, err := Runner{
		Sandbox: &bubblewrap{bin: "definitely-not-a-real-binary-dcode"},
		Mode:    policy.ModeWorkspaceWrite,
	}.Run(context.Background(), t.TempDir(), "echo hi")
	if err == nil {
		t.Fatal("a missing binary must surface as an error, not as a silent success")
	}
	if code != -1 {
		t.Errorf("got exit %d", code)
	}
}

func TestBubblewrapAvailableReportsAMissingBinary(t *testing.T) {
	err := (&bubblewrap{bin: "definitely-not-a-real-binary-dcode"}).Available()
	if err == nil {
		t.Fatal("a missing binary must be reported")
	}
	if !strings.Contains(err.Error(), "Install") {
		t.Errorf("the message must say how to fix it: %v", err)
	}
}

func TestSeatbeltAvailableReportsAMissingBinary(t *testing.T) {
	err := (&seatbelt{bin: "definitely-not-a-real-binary-dcode"}).Available()
	if err == nil {
		t.Fatal("a missing binary must be reported")
	}
}

func TestCanonicalFallsBackToTheInput(t *testing.T) {
	// A workspace that does not exist yet still has to produce a usable
	// profile; failing here would block session creation on a directory the
	// user is about to create.
	p := filepath.Join(t.TempDir(), "not-created-yet")
	if got := canonical(p); got != p {
		t.Errorf("got %q want %q", got, p)
	}
}

func TestFullAccessProfileGrantsEverything(t *testing.T) {
	p, err := (&seatbelt{bin: "x"}).profile("/w", policy.ModeFullAccess, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(p, "(allow file-write*)") || !strings.Contains(p, "(allow network") {
		t.Errorf("full-access must grant both:\n%s", p)
	}

	args, err := (&bubblewrap{bin: "x"}).args("/w", policy.ModeFullAccess, nil)
	if err != nil {
		t.Fatal(err)
	}
	if contains(args, "--unshare-net") {
		t.Errorf("full-access must keep the network: %v", args)
	}
}
