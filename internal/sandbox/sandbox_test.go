package sandbox

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/policy"
)

// The rule the whole package rests on: no mechanism, no execution. Degrading to
// an unconfined run is worse than having no sandbox at all, because it promises
// a boundary it does not deliver.
func TestNoneBackendIsRejectedUnlessFullAccess(t *testing.T) {
	for _, mode := range []policy.SandboxMode{policy.ModeReadOnly, policy.ModeWorkspaceWrite} {
		_, err := New(Config{Backend: BackendNone}, mode)
		if err == nil {
			t.Fatalf("mode %s with no sandbox must be refused", mode)
		}
		if !errors.Is(err, ErrUnavailable) {
			t.Errorf("mode %s: want ErrUnavailable, got %v", mode, err)
		}
	}

	// Only an explicit full-access request may run unconfined.
	s, err := New(Config{Backend: BackendNone}, policy.ModeFullAccess)
	if err != nil {
		t.Fatalf("full-access with backend none should be allowed: %v", err)
	}
	if s.Name() != BackendNone {
		t.Errorf("got %q", s.Name())
	}
}

func TestUnknownBackendIsRejected(t *testing.T) {
	if _, err := New(Config{Backend: "docker"}, policy.ModeWorkspaceWrite); err == nil {
		t.Fatal("an unknown backend must be rejected")
	}
}

func TestMissingBinaryNamesItAndHowToGetIt(t *testing.T) {
	_, err := New(Config{
		Backend: BackendBubblewrap,
		Binary:  "definitely-not-installed-dcode-test",
	}, policy.ModeWorkspaceWrite)
	if err == nil {
		t.Fatal("a missing binary must fail")
	}
	if !errors.Is(err, ErrUnavailable) {
		t.Errorf("want ErrUnavailable, got %v", err)
	}
	// "sandbox unavailable" alone sends the user reading source code.
	if !strings.Contains(err.Error(), "definitely-not-installed") {
		t.Errorf("the error must name the binary: %v", err)
	}
	if !strings.Contains(err.Error(), "Install") {
		t.Errorf("the error must say how to fix it: %v", err)
	}
}

func TestRunnerWithoutASandboxRefuses(t *testing.T) {
	_, _, err := Runner{}.Run(context.Background(), t.TempDir(), "echo hi")
	if !errors.Is(err, ErrUnavailable) {
		t.Errorf("want ErrUnavailable, got %v", err)
	}
}

// ---------- profile and argument construction ----------

func TestSeatbeltProfileDeniesByDefault(t *testing.T) {
	s := &seatbelt{bin: "sandbox-exec"}
	p, err := s.profile("/w", policy.ModeWorkspaceWrite)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(p, "(deny default)") {
		t.Errorf("the profile must deny by default: %s", p)
	}
}

func TestSeatbeltReadOnlyGrantsNoWrite(t *testing.T) {
	s := &seatbelt{bin: "sandbox-exec"}
	p, err := s.profile("/w", policy.ModeReadOnly)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(p, "file-write") {
		t.Errorf("read-only must grant no write rule at all:\n%s", p)
	}
}

func TestSeatbeltWorkspaceWriteIsScopedToTheWorkspace(t *testing.T) {
	s := &seatbelt{bin: "sandbox-exec"}
	p, err := s.profile("/Users/x/proj", policy.ModeWorkspaceWrite)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(p, `(subpath "/Users/x/proj")`) {
		t.Errorf("the workspace must be writable:\n%s", p)
	}
	if strings.Contains(p, "(allow file-write*)\n(allow network") {
		t.Errorf("workspace-write must not grant blanket write:\n%s", p)
	}
}

func TestSeatbeltNetworkIsDeniedUnlessGranted(t *testing.T) {
	denied := &seatbelt{bin: "sandbox-exec"}
	p, _ := denied.profile("/w", policy.ModeWorkspaceWrite)
	if strings.Contains(p, "(allow network") {
		t.Errorf("network must be denied by default:\n%s", p)
	}

	granted := &seatbelt{bin: "sandbox-exec", allowNetwork: func() bool { return true }}
	p, _ = granted.profile("/w", policy.ModeWorkspaceWrite)
	if !strings.Contains(p, "(allow network") {
		t.Errorf("network should be granted when configured:\n%s", p)
	}
}

func TestSeatbeltRejectsAnUnknownMode(t *testing.T) {
	s := &seatbelt{bin: "sandbox-exec"}
	if _, err := s.profile("/w", policy.SandboxMode("yolo")); err == nil {
		t.Error("an unknown mode must be rejected, never treated as permissive")
	}
	if _, err := s.profile("", policy.ModeWorkspaceWrite); err == nil {
		t.Error("an empty workdir must be rejected")
	}
}

func TestBubblewrapUnsharesTheNetworkUnlessGranted(t *testing.T) {
	b := &bubblewrap{bin: "bwrap"}
	args, err := b.args("/w", policy.ModeWorkspaceWrite)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(args, "--unshare-net") {
		t.Errorf("network must be removed by default: %v", args)
	}

	b.allowNetwork = func() bool { return true }
	args, _ = b.args("/w", policy.ModeWorkspaceWrite)
	if contains(args, "--unshare-net") {
		t.Errorf("network should stay when granted: %v", args)
	}
}

func TestBubblewrapReadOnlyBindsNothingWritable(t *testing.T) {
	b := &bubblewrap{bin: "bwrap"}
	args, err := b.args("/w", policy.ModeReadOnly)
	if err != nil {
		t.Fatal(err)
	}
	// A --bind of the workspace would make it writable, which is exactly what
	// read-only must not do.
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--bind" {
			t.Errorf("read-only must not bind anything writable: %v", args)
		}
	}
	if !contains(args, "--ro-bind") {
		t.Errorf("the filesystem should be mounted read-only: %v", args)
	}
}

func TestBubblewrapWorkspaceWriteBindsOnlyTheWorkspace(t *testing.T) {
	b := &bubblewrap{bin: "bwrap"}
	args, err := b.args("/home/u/proj", policy.ModeWorkspaceWrite)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for i := 0; i < len(args)-2; i++ {
		if args[i] == "--bind" && args[i+1] == "/home/u/proj" {
			found = true
		}
		if args[i] == "--bind" && args[i+1] == "/" {
			t.Errorf("workspace-write must not bind the whole filesystem: %v", args)
		}
	}
	if !found {
		t.Errorf("the workspace must be writable: %v", args)
	}
}

func TestBubblewrapRejectsBadInput(t *testing.T) {
	b := &bubblewrap{bin: "bwrap"}
	if _, err := b.args("/w", policy.SandboxMode("nope")); err == nil {
		t.Error("an unknown mode must be rejected")
	}
	if _, err := b.args("", policy.ModeWorkspaceWrite); err == nil {
		t.Error("an empty workdir must be rejected")
	}
}

// ---------- the assertion that actually matters ----------

// The test that proves the boundary is real: the write must fail because the
// operating system refused it, not because a Go check returned early. A test
// that only inspects our own return value would pass with the sandbox switched
// off entirely, which is the classic useless sandbox test.
func TestReadOnlyBlocksWritesAtTheOSLevel(t *testing.T) {
	s := realSandbox(t)
	ws := t.TempDir()
	target := filepath.Join(ws, "should-not-appear.txt")

	r := Runner{Sandbox: s, Mode: policy.ModeReadOnly}
	out, code, err := r.Run(context.Background(), ws, "echo written > "+shellQuote(target))
	if err != nil {
		t.Fatalf("running under the sandbox failed outright: %v", err)
	}

	if _, statErr := os.Stat(target); statErr == nil {
		t.Fatalf("read-only mode allowed a write (exit %d, output %q)", code, out)
	}
	if code == 0 {
		t.Errorf("the write should have failed inside the sandbox, got exit 0: %q", out)
	}
}

func TestWorkspaceWriteAllowsTheWorkspaceAndBlocksOutside(t *testing.T) {
	s := realSandbox(t)
	root := t.TempDir()
	ws := filepath.Join(root, "ws")
	if err := os.MkdirAll(ws, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside.txt")

	r := Runner{Sandbox: s, Mode: policy.ModeWorkspaceWrite}

	inside := filepath.Join(ws, "ok.txt")
	if _, code, err := r.Run(context.Background(), ws, "echo hi > "+shellQuote(inside)); err != nil || code != 0 {
		t.Fatalf("writing inside the workspace must work: code=%d err=%v", code, err)
	}
	if _, err := os.Stat(inside); err != nil {
		t.Fatalf("the file should exist: %v", err)
	}

	out, code, err := r.Run(context.Background(), ws, "echo leak > "+shellQuote(outside))
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if _, statErr := os.Stat(outside); statErr == nil {
		t.Fatalf("a write escaped the workspace (exit %d, output %q)", code, out)
	}
}

func TestNonZeroExitIsAnAnswerNotAFailure(t *testing.T) {
	s := realSandbox(t)
	r := Runner{Sandbox: s, Mode: policy.ModeWorkspaceWrite}
	out, code, err := r.Run(context.Background(), t.TempDir(), "exit 7")
	if err != nil {
		t.Fatalf("a failing command must not be a run error: %v (%q)", err, out)
	}
	if code != 7 {
		t.Errorf("exit code should reach the caller, got %d", code)
	}
}

// realSandbox returns the platform sandbox, skipping when the machine cannot
// provide one. Skipping is honest here: the assertion needs a real kernel
// boundary, and faking it would test nothing.
func realSandbox(t *testing.T) Sandbox {
	t.Helper()
	s, err := New(Config{}, policy.ModeWorkspaceWrite)
	if err != nil {
		t.Skipf("no sandbox available on %s: %v", runtime.GOOS, err)
	}
	return s
}

func contains(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}

func TestShellQuoteHandlesQuotes(t *testing.T) {
	if got := shellQuote("it's"); got != `'it'\''s'` {
		t.Errorf("got %s", got)
	}
}

func TestDefaultBackendPerPlatform(t *testing.T) {
	got := defaultBackend()
	switch runtime.GOOS {
	case "darwin":
		if got != BackendSeatbelt {
			t.Errorf("got %q", got)
		}
	case "linux":
		if got != BackendBubblewrap {
			t.Errorf("got %q", got)
		}
	default:
		// An unsupported platform must not silently pick something that
		// confines nothing.
		if got != "unsupported" {
			t.Errorf("got %q", got)
		}
	}
}

// The permission can arrive mid-session: the user is asked at the first
// crossing and answers "this project". A boundary decided once, at
// construction, would leave that answer with no effect until a restart — the
// user grants it, the command fails anyway, and the only reading available to
// them is that granting did not work.
func TestTheNetworkDecisionIsAskedPerCommandNotOncePerSession(t *testing.T) {
	granted := false
	s := &seatbelt{bin: "sandbox-exec", allowNetwork: func() bool { return granted }}

	before, err := s.profile("/w", policy.ModeWorkspaceWrite)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(before, "(allow network") {
		t.Fatal("the network was open before anyone granted it")
	}

	granted = true

	after, err := s.profile("/w", policy.ModeWorkspaceWrite)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(after, "(allow network") {
		t.Error("a grant made during the session had no effect on the next command")
	}
}

// A wiring mistake must not open a boundary, and must not crash a session
// either. Both are bad, in opposite directions: a silent yes grants what nobody
// was asked about, and a panic takes down the session over a nil field.
func TestAnAbsentNetworkDecisionIsRefusedRatherThanAssumed(t *testing.T) {
	s := &seatbelt{bin: "sandbox-exec"} // nothing wired
	profile, err := s.profile("/w", policy.ModeWorkspaceWrite)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(profile, "(allow network") {
		t.Error("an unwired decision was read as permission")
	}

	b := &bubblewrap{bin: "bwrap"}
	args, err := b.args("/w", policy.ModeWorkspaceWrite)
	if err != nil {
		t.Fatal(err)
	}
	var isolated bool
	for _, a := range args {
		if a == "--unshare-net" {
			isolated = true
		}
	}
	if !isolated {
		t.Error("an unwired decision left the network shared")
	}
}

// The ceiling has to be a ceiling.
//
// A turn was held for 3m38s against a 120s timeout, because the model worked
// around the missing background support the only way it could:
//
//	nohup node src/server.js > /tmp/log 2>&1 & disown; echo "PID=$!"
//
// CombinedOutput waits for the output pipe to reach EOF, not for the process to
// exit. Any grandchild that inherited the pipe holds it open, and killing the
// direct child when the context expires does not close it. So the one guarantee
// the timeout exists to make — the session comes back — was the guarantee it
// could not keep.
func TestARunReturnsAtItsCeilingEvenWithAChildHoldingTheOutput(t *testing.T) {
	r := Runner{Sandbox: noneSandbox{}, Mode: policy.ModeFullAccess}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	// `sleep` inherits stdout and holds the pipe long after the shell exits.
	out, _, _ := r.Run(ctx, t.TempDir(), "sleep 30 & echo started")
	elapsed := time.Since(start)

	if elapsed > 3*time.Second {
		t.Errorf("the run took %s against a 500ms ceiling; the session is held by a process nobody is waiting for", elapsed)
	}
	if !strings.Contains(out, "started") {
		t.Errorf("what the command produced before the cut was lost: %q", out)
	}
}

// And a command that finishes returns its output whole, or the fix has traded
// a hang for truncation.
func TestARunThatFinishesReturnsEverything(t *testing.T) {
	r := Runner{Sandbox: noneSandbox{}, Mode: policy.ModeFullAccess}
	out, code, err := r.Run(context.Background(), t.TempDir(), "echo one; echo two >&2; echo three")
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Errorf("exit %d", code)
	}
	for _, want := range []string{"one", "two", "three"} {
		if !strings.Contains(out, want) {
			t.Errorf("the output lost %q: %q", want, out)
		}
	}
}
