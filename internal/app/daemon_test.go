package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/sandbox"
	"github.com/aguinelo/dcode/pkg/client"
)

// A Unix socket path is capped near 104 bytes on macOS, and the XDG state
// directory alone can exhaust that.
func TestDefaultSocketPathStaysShortAndSeparatesUsers(t *testing.T) {
	explicit := DefaultSocketPath(envFrom(map[string]string{"DCODE_SOCKET": "/tmp/x.sock"}))
	if explicit != "/tmp/x.sock" {
		t.Errorf("an explicit address wins, got %q", explicit)
	}

	runtime := DefaultSocketPath(envFrom(map[string]string{"XDG_RUNTIME_DIR": "/run/user/1000"}))
	if runtime != "/run/user/1000/dcode.sock" {
		t.Errorf("got %q", runtime)
	}

	fallback := DefaultSocketPath(envFrom(map[string]string{"TMPDIR": "/var/tmp"}))
	if !strings.HasPrefix(fallback, "/var/tmp/dcode-") {
		t.Errorf("got %q", fallback)
	}
	// Two users on one machine must not land on the same socket.
	if !strings.Contains(fallback, "-") || strings.HasSuffix(fallback, "dcode-.sock") {
		t.Errorf("the uid must be part of the path: %q", fallback)
	}

	bare := DefaultSocketPath(envFrom(map[string]string{}))
	if !strings.HasPrefix(bare, "/tmp/dcode-") {
		t.Errorf("got %q", bare)
	}
	for _, p := range []string{explicit, runtime, fallback, bare} {
		if len(p) > 100 {
			t.Errorf("%q is %d bytes, too long for a unix socket", p, len(p))
		}
	}
}

func TestNewDaemonFillsInDefaults(t *testing.T) {
	d := NewDaemon(DaemonOptions{SocketPath: "/tmp/x.sock"})
	if d.opts.MaxSessions <= 0 || d.opts.ApprovalTimeout <= 0 {
		t.Errorf("got %+v", d.opts)
	}
	if d.Manager() == nil {
		t.Error("a daemon without a manager cannot hold a session")
	}
	if d.Addr() != "/tmp/x.sock" {
		t.Errorf("got %q", d.Addr())
	}
}

func daemonFor(t *testing.T, ws string) (*Daemon, *client.Client, context.CancelFunc) {
	t.Helper()
	// Short path: macOS caps a socket at ~104 bytes and t.TempDir() carries the
	// test name, which is enough to blow it.
	dir, err := os.MkdirTemp("", "dc")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	base, _, err := FromEnv(envFrom(map[string]string{}), ws)
	if err != nil {
		t.Fatal(err)
	}
	base.SandboxMode = policy.ModeReadOnly
	requireSandbox(t, base)

	d := NewDaemon(DaemonOptions{
		SocketPath: filepath.Join(dir, "d.sock"),
		Base:       base,
	})
	if err := d.Listen(); err != nil {
		t.Skipf("cannot bind a unix socket here: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go d.Serve(ctx)
	t.Cleanup(cancel)

	c := client.New(d.Addr())
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if err := c.Health(ctx); err == nil {
			return d, c, cancel
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("the daemon never became healthy")
	return nil, nil, cancel
}

// The daemon has to hold a real session end to end, because that is the only
// thing that proves the client, the protocol and the engine agree.
func TestDaemonServesASessionOverASocket(t *testing.T) {
	ws := t.TempDir()
	d, c, cancel := daemonFor(t, ws)
	ctx := context.Background()

	sess, err := c.CreateSession(ctx, protocol.CreateSessionRequest{
		Workspace: ws, Model: "MiniMax-M3", SandboxMode: "read-only",
	})
	if err != nil {
		t.Fatal(err)
	}
	if sess.ID == "" || sess.Workspace != ws {
		t.Fatalf("got %+v", sess)
	}
	if sess.SandboxMode != "read-only" {
		t.Errorf("the requested mode must reach the session, got %q", sess.SandboxMode)
	}

	list, err := c.ListSessions(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("got %v, %v", list, err)
	}
	if d.Manager().Count() != 1 {
		t.Errorf("got %d", d.Manager().Count())
	}

	// The session is created before any turn runs, so its log already carries
	// the creation event a reattaching client would replay.
	events, _ := c.Subscribe(ctx, sess.ID, 1)
	select {
	case ev := <-events:
		if ev.Type != protocol.EventSessionCreated {
			t.Errorf("got %s", ev.Type)
		}
	case <-time.After(3 * time.Second):
		t.Error("the creation event never arrived")
	}

	if err := c.DeleteSession(ctx, sess.ID); err != nil {
		t.Fatal(err)
	}
	cancel()
}

// A session is the unit of confinement, so a request naming a mode that does
// not exist must be refused rather than quietly downgraded — or, far worse,
// upgraded.
func TestDaemonRefusesAnUnknownSandboxMode(t *testing.T) {
	ws := t.TempDir()
	_, c, _ := daemonFor(t, ws)

	_, err := c.CreateSession(context.Background(), protocol.CreateSessionRequest{
		Workspace: ws, SandboxMode: "yolo",
	})
	if err == nil {
		t.Fatal("an unknown mode must be refused")
	}
}

func TestDaemonEachSessionGetsItsOwnWorkspace(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	_, c, _ := daemonFor(t, first)
	ctx := context.Background()

	a, err := c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: first})
	if err != nil {
		t.Fatal(err)
	}
	b, err := c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: second})
	if err != nil {
		t.Fatal(err)
	}
	if a.Workspace == b.Workspace {
		t.Errorf("one workspace's boundary must not apply to another: %q", a.Workspace)
	}
	if a.ID == b.ID {
		t.Errorf("two sessions must not share an id: %q", a.ID)
	}
}

// The session is its own emitter and its own approver: that is what lets any
// attached client answer a crossing.
func TestSessionEmitterAndApproverAreBoundToTheSession(t *testing.T) {
	var got protocol.EventType
	emitterFunc(func(tp protocol.EventType, _ any) { got = tp }).
		Emit(protocol.EventTurnStarted, nil)
	if got != protocol.EventTurnStarted {
		t.Errorf("got %q", got)
	}

	d, err := approverFunc(func(context.Context, protocol.ApprovalRequest) (protocol.ApprovalDecision, error) {
		return protocol.ApprovalDeny, nil
	}).Approve(context.Background(), protocol.ApprovalRequest{})
	if err != nil || d != protocol.ApprovalDeny {
		t.Errorf("got %v, %v", d, err)
	}
}

func TestRandomUint32Varies(t *testing.T) {
	seen := map[uint32]struct{}{}
	for i := 0; i < 32; i++ {
		seen[randomUint32()] = struct{}{}
	}
	// Session ids carry this to keep two sessions created in the same
	// millisecond apart.
	if len(seen) < 16 {
		t.Errorf("only %d distinct values in 32 draws", len(seen))
	}
}

func TestResolveRootsAndHelpers(t *testing.T) {
	home := t.TempDir()
	roots, err := ResolveRoots(envFrom(map[string]string{"DCODE_HOME": home}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(roots.Config, home) {
		t.Errorf("DCODE_HOME must collapse the roots, got %+v", roots)
	}

	if _, err := parseMode("read-only"); err != nil {
		t.Fatal(err)
	}
	if _, err := parseMode("nonsense"); err == nil {
		t.Error("an unknown mode must be rejected")
	}
	if osUID() < 0 {
		t.Error("the uid must be usable in a path")
	}
}

func TestReadFileText(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.md")
	if err := os.WriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readFileText(path)
	if err != nil || got != "hello" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := readFileText(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Error("a missing file must be reported")
	}
}

// The user's own root is the only place an instruction from outside the
// workspace may enter, and it enters by an explicit path rather than by
// discovery.
func TestLoadInstructionsIncludesTheUserRootAndReturnsTheChain(t *testing.T) {
	home, ws := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "AGENTS.md"), []byte("USER-RULE"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, "DCODE.md"), []byte("PROJECT-RULE"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, chain, err := loadInstructions(configRoots(home), ws)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %+v", got)
	}
	// The user's base comes first and lowest; the project is more specific and
	// therefore later, which is the position of greater weight.
	if !strings.Contains(got[0].Text, "USER-RULE") || !strings.Contains(got[1].Text, "PROJECT-RULE") {
		t.Errorf("got %+v", got)
	}
	if len(chain) != 2 {
		t.Fatalf("the chain is what tells a loaded instruction from an unloaded one: %v", chain)
	}
	for _, p := range chain {
		if !filepath.IsAbs(p) {
			t.Errorf("a chain entry must be a path that can be compared: %q", p)
		}
	}
}

func TestLoadInstructionsWithNothingToLoad(t *testing.T) {
	got, chain, err := loadInstructions(configRoots(filepath.Join(t.TempDir(), "absent")), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 || len(chain) != 0 {
		t.Errorf("got %+v, %v", got, chain)
	}
}

func TestDaemonBuildHonoursTheRequestedModel(t *testing.T) {
	ws := t.TempDir()
	base, _, err := FromEnv(envFrom(map[string]string{}), ws)
	if err != nil {
		t.Fatal(err)
	}
	base.SandboxMode = policy.ModeReadOnly
	requireSandbox(t, base)

	d := NewDaemon(DaemonOptions{SocketPath: "/tmp/unused.sock", Base: base})
	sess, err := d.build(protocol.CreateSessionRequest{Workspace: ws, Model: "claude-sonnet"})
	if err != nil {
		t.Fatal(err)
	}
	if got := sess.Describe().Model; got != "claude-sonnet" {
		t.Errorf("got %q", got)
	}

}

// configRoots builds a Roots whose four directories collapse onto one, which is
// what DCODE_HOME does in production.
func configRoots(home string) config.Roots {
	return config.Roots{Config: home, Data: home, State: home, Cache: home}
}

// The workspace is the anchor of every boundary the session enforces, and it is
// the one field a client fully controls.
func TestDaemonRefusesAnUnusableWorkspace(t *testing.T) {
	ws := t.TempDir()
	file := filepath.Join(ws, "a-file")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	base, _, err := FromEnv(envFrom(map[string]string{}), ws)
	if err != nil {
		t.Fatal(err)
	}
	d := NewDaemon(DaemonOptions{SocketPath: "/tmp/unused.sock", Base: base})

	// No requireSandbox here on purpose: the workspace is rejected before a
	// sandbox is ever built, and that is exactly what this asserts.
	for name, path := range map[string]string{
		"empty":     "",
		"relative":  "relative/path",
		"missing":   filepath.Join(ws, "no-such-dir"),
		"is a file": file,
	} {
		_, err := d.build(protocol.CreateSessionRequest{Workspace: path})
		if err == nil {
			t.Errorf("%s: must be refused", name)
			continue
		}
		pe, ok := protocol.AsError(err)
		if !ok || pe.Code != protocol.CodeWorkspaceInvalid {
			t.Errorf("%s: the code is what a client branches on, got %v", name, err)
		}
	}
}

// requireSandbox skips when the machine has no confining mechanism.
//
// Without a real boundary a daemon test asserts nothing — a session that cannot
// confine its own commands refuses to start, by design. Skipping is the honest
// outcome, and it matches what the end-to-end wiring tests already do.
func requireSandbox(t *testing.T, opts Options) {
	t.Helper()
	if _, err := sandbox.New(sandbox.Config{
		Backend:      opts.Backend,
		AllowNetwork: opts.AllowNetwork,
	}, opts.SandboxMode); err != nil {
		t.Skipf("no sandbox available: %v", err)
	}
}
