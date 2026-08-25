package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/sandbox"
	"github.com/aguinelo/dcode/internal/session"
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
	// The discovery itself, not a one-line wrapper over it: the wrapper added
	// a name to the package surface and nothing else.
	roots, err := config.DiscoverRoots(envFrom(map[string]string{"DCODE_HOME": home}))
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
		AllowNetwork: func() bool { return opts.AllowNetwork },
	}, opts.SandboxMode); err != nil {
		t.Skipf("no sandbox available: %v", err)
	}
}

// A session opened through the daemon leaves a file somebody can read.
//
// This is the end the whole change exists for. The pieces were all present —
// an append-only log, a spill file, a config key, a state directory — and the
// two commands people actually run, `dcode` and `dcode tui`, wired none of it.
// So there was no way to audit what dcode did, no way to reconstruct a session
// afterwards, and no evidence to reason from when its behaviour needed work.
func TestASessionThroughTheDaemonLeavesAReadableRecord(t *testing.T) {
	dir := t.TempDir()
	d := NewDaemon(DaemonOptions{
		SocketPath:     filepath.Join(t.TempDir(), "d.sock"),
		EventRetention: 10000,
		RecordDir:      dir,
		Base:           baseOpts(t),
	})

	sess, err := d.build(protocol.CreateSessionRequest{Workspace: t.TempDir(), Model: "MiniMax-M3"})
	if err != nil {
		t.Skipf("a session cannot be built here: %v", err)
	}
	sess.Emit(protocol.EventToolRequested, map[string]string{"tool": "read"})
	sess.Emit(protocol.EventToolCompleted, map[string]string{"ok": "yes"})
	sess.Close()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("the session left %d files, want its record", len(entries))
	}
	body, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "tool.requested") ||
		!strings.Contains(string(body), "tool.completed") {
		t.Errorf("the record does not hold what the session did:\n%s", body)
	}
	// The file is the person's transcript. Anyone else on the machine reading
	// it is reading what they typed and what the agent saw.
	info, err := entries[0].Info()
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("record mode is %v, want 0600", perm)
	}
}

// Opening a session is when history is tidied, and the session being opened is
// never what gets tidied away.
//
// On a timer would mean something deleting a person's history while the program
// is not running. A readdir on the way into a session is cheap enough that
// nobody notices, and it happens exactly when the directory is about to grow.
func TestOpeningASessionPrunesOldRecordsAndKeepsItsOwn(t *testing.T) {
	dir := t.TempDir()
	stale := filepath.Join(dir, "s-from-last-year.jsonl")
	if err := os.WriteFile(stale, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-400 * 24 * time.Hour)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}

	d := NewDaemon(DaemonOptions{
		SocketPath:     filepath.Join(t.TempDir(), "d.sock"),
		EventRetention: 10000,
		RecordDir:      dir,
		RecordBudget:   session.PruneBudget{MaxAge: 24 * time.Hour},
		Base:           baseOpts(t),
		Log:            func(string) {},
	})

	sess, err := d.build(protocol.CreateSessionRequest{Workspace: t.TempDir(), Model: "MiniMax-M3"})
	if err != nil {
		t.Skipf("a session cannot be built here: %v", err)
	}
	sess.Emit(protocol.EventToolRequested, map[string]string{"tool": "read"})
	sess.Close()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("directory holds %d records, want only the new session's", len(entries))
	}
	if entries[0].Name() == "s-from-last-year.jsonl" {
		t.Error("the stale record survived and the new one did not")
	}
}

// A record that cannot be opened is said out loud and does not stop the
// session. An audit trail must not hold the product hostage, and a silent
// failure to record is the one outcome worse than not recording at all.
func TestARecordThatCannotBeOpenedIsReportedAndNotFatal(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocked, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	var said []string
	d := NewDaemon(DaemonOptions{
		SocketPath:     filepath.Join(t.TempDir(), "d.sock"),
		EventRetention: 10000,
		RecordDir:      blocked,
		RecordBudget:   session.PruneBudget{MaxAge: 24 * time.Hour},
		Base:           baseOpts(t),
		Log:            func(m string) { said = append(said, m) },
	})

	// Skip, not fail, when the machine cannot confine a command at all. A CI
	// runner without a usable namespace answers "bwrap: setting up uid map:
	// Permission denied", and that is the environment talking, not the
	// behaviour under test — which is why every sibling test here skips on it.
	//
	// Failing instead is what turned this green on macOS and red on Linux.
	sess, err := d.build(protocol.CreateSessionRequest{Workspace: t.TempDir(), Model: "MiniMax-M3"})
	if err != nil {
		t.Skipf("a session cannot be built here: %v", err)
	}
	sess.Emit(protocol.EventToolRequested, map[string]string{"tool": "read"})
	sess.Close()

	var told bool
	for _, m := range said {
		if strings.Contains(m, "not being recorded") || strings.Contains(m, "could not be pruned") {
			told = true
		}
	}
	if !told {
		t.Errorf("nothing was said about a session that is not being recorded: %v", said)
	}
}

// Continuing carries the conversation and nothing else.
//
// A new session either way: the old one ended with the client that ran it, and
// nothing survives what created it. What crosses over is what was said, which
// is the only part a person cannot reconstruct themselves.
func TestContinuingASessionCarriesItsConversation(t *testing.T) {
	dir := t.TempDir()
	body := `{"seq":1,"type":"session.created","at":"2026-08-14T12:00:00Z","payload":{"id":"old","workspace":"/w"}}
{"seq":2,"type":"turn.started","at":"2026-08-14T12:00:01Z","payload":{"turn_id":"t1","text":"what does Rows do?"}}
{"seq":3,"type":"message.delta","at":"2026-08-14T12:00:02Z","payload":{"turn_id":"t1","text":"count minus one"}}
`
	if err := os.WriteFile(filepath.Join(dir, "old.jsonl"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	d := NewDaemon(DaemonOptions{
		SocketPath:     filepath.Join(t.TempDir(), "d.sock"),
		EventRetention: 10000,
		RecordDir:      dir,
		Base:           baseOpts(t),
	})

	sess, err := d.build(protocol.CreateSessionRequest{
		Workspace: t.TempDir(), Model: "MiniMax-M3", Resume: "old",
	})
	if err != nil {
		t.Skipf("a session cannot be built here: %v", err)
	}
	defer sess.Close()

	if sess.ID == "old" {
		t.Error("continuing reopened the old session rather than starting one that carries it")
	}
}

// Asking to continue something that is not there fails loudly. Starting fresh
// instead would be discovered only once the model had forgotten everything.
func TestContinuingAMissingSessionIsAnError(t *testing.T) {
	d := NewDaemon(DaemonOptions{
		SocketPath:     filepath.Join(t.TempDir(), "d.sock"),
		EventRetention: 10000,
		RecordDir:      t.TempDir(),
		Base:           baseOpts(t),
	})

	if _, err := d.build(protocol.CreateSessionRequest{
		Workspace: t.TempDir(), Model: "MiniMax-M3", Resume: "never-existed",
	}); err == nil {
		t.Fatal("continuing a session that does not exist reported success")
	}
}

// Continuing shows the conversation it continues, and records it.
//
// The history reached the model and never the person: the screen is built from
// events, and nothing emitted one for what was carried. Someone who continued a
// session saw a blank window and the only available reading was that the work
// was gone.
//
// Recording it is the same fix for a second failure. A new session's record
// held only its own turns, so continuing something that was itself a
// continuation dropped everything before the last hop — that one losing the
// model too, silently.
func TestContinuingShowsAndRecordsWhatItCarries(t *testing.T) {
	dir := t.TempDir()
	body := `{"seq":1,"type":"session.created","at":"2026-08-14T12:00:00Z","payload":{"id":"old","workspace":"/w"}}
{"seq":2,"type":"turn.started","at":"2026-08-14T12:00:01Z","payload":{"turn_id":"t1","text":"what does Rows do?"}}
{"seq":3,"type":"message.delta","at":"2026-08-14T12:00:02Z","payload":{"turn_id":"t1","text":"count minus one"}}
{"seq":4,"type":"turn.completed","at":"2026-08-14T12:00:03Z","payload":{"turn_id":"t1"}}
`
	if err := os.WriteFile(filepath.Join(dir, "old.jsonl"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	d := NewDaemon(DaemonOptions{
		SocketPath:     filepath.Join(t.TempDir(), "d.sock"),
		EventRetention: 10000,
		RecordDir:      dir,
		Base:           baseOpts(t),
	})
	sess, err := d.build(protocol.CreateSessionRequest{
		Workspace: t.TempDir(), Model: "MiniMax-M3", Resume: "old",
	})
	if err != nil {
		t.Skipf("a session cannot be built here: %v", err)
	}
	defer sess.Close()

	sess.Emit(protocol.EventSessionCreated, sess.Describe())
	sess.EmitCarried()

	events, err := sess.Log.Replay(1)
	if err != nil {
		t.Fatal(err)
	}
	var saw []protocol.EventType
	for _, ev := range events {
		saw = append(saw, ev.Type)
	}
	want := []protocol.EventType{
		protocol.EventSessionCreated,
		protocol.EventSessionResumed,
		protocol.EventTurnStarted,
		protocol.EventMessageDelta,
		protocol.EventTurnCompleted,
	}
	if len(saw) != len(want) {
		t.Fatalf("the stream is %v, want %v", saw, want)
	}
	for i := range want {
		if saw[i] != want[i] {
			t.Fatalf("the stream is %v, want %v", saw, want)
		}
	}

	// The marker says where it came from. Without it the replayed conversation
	// reads as something that happened in this session, which it did not.
	var r protocol.SessionResumed
	if err := json.Unmarshal(events[1].Payload, &r); err != nil {
		t.Fatal(err)
	}
	if r.SourceID != "old" {
		t.Errorf("the marker names %q, want the session it continues", r.SourceID)
	}
	if r.Turns != 1 {
		t.Errorf("the marker carries %d turns, want 1", r.Turns)
	}

	// The carried events keep this session's identity and its sequence. Two
	// events with the same seq is a replay a client cannot tell apart.
	for i, ev := range events {
		if ev.SessionID != sess.ID {
			t.Errorf("event %d belongs to %q, not to this session", i, ev.SessionID)
		}
		if ev.Seq != uint64(i+1) {
			t.Errorf("event %d has seq %d, want %d", i, ev.Seq, i+1)
		}
	}
}

// A second hop carries everything, not just the last leg.
func TestContinuingAContinuationKeepsTheWholeConversation(t *testing.T) {
	dir := t.TempDir()
	first := `{"seq":1,"type":"session.created","at":"2026-08-14T12:00:00Z","payload":{"id":"a","workspace":"/w"}}
{"seq":2,"type":"turn.started","at":"2026-08-14T12:00:01Z","payload":{"turn_id":"t1","text":"the first question"}}
{"seq":3,"type":"message.delta","at":"2026-08-14T12:00:02Z","payload":{"turn_id":"t1","text":"the first answer"}}
{"seq":4,"type":"turn.completed","at":"2026-08-14T12:00:03Z","payload":{"turn_id":"t1"}}
`
	if err := os.WriteFile(filepath.Join(dir, "a.jsonl"), []byte(first), 0o600); err != nil {
		t.Fatal(err)
	}

	d := NewDaemon(DaemonOptions{
		SocketPath:     filepath.Join(t.TempDir(), "d.sock"),
		EventRetention: 10000,
		RecordDir:      dir,
		Base:           baseOpts(t),
	})
	b, err := d.build(protocol.CreateSessionRequest{
		Workspace: t.TempDir(), Model: "MiniMax-M3", Resume: "a",
	})
	if err != nil {
		t.Skipf("a session cannot be built here: %v", err)
	}
	b.Emit(protocol.EventSessionCreated, b.Describe())
	b.EmitCarried()
	b.Close()

	// B never asked anything of its own. Everything it can offer came from A,
	// which is exactly the case that used to come back empty.
	carried, turns, err := session.Carry(filepath.Join(dir, b.ID+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if turns != 1 {
		t.Fatalf("the second hop counts %d turns, want the one from the first", turns)
	}
	var asked string
	for _, ev := range carried {
		if ev.Type != protocol.EventTurnStarted {
			continue
		}
		var d protocol.TurnStarted
		if json.Unmarshal(ev.Payload, &d) == nil {
			asked = d.Text
		}
	}
	if asked != "the first question" {
		t.Errorf("the second hop carries %q, want the question from the first session", asked)
	}
}

// Daemon.Listen and Daemon.Serve bind a unix socket and accept on it; together
// they are the daemon's only public entry point after NewDaemon. The existing
// daemonFor helper binds inside itself and skips when it cannot, which leaves
// these two functions unwalked on machines with no sandbox.
//
// Bind a short path directly and confirm Addr reports it; Serve blocks until
// the context is cancelled, which is what the rest of the suite relies on.
func TestDaemonListensAndServesOnAShortPath(t *testing.T) {
	dir, err := os.MkdirTemp("", "dc")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	d := NewDaemon(DaemonOptions{SocketPath: filepath.Join(dir, "d.sock")})
	if err := d.Listen(); err != nil {
		t.Skipf("cannot bind a unix socket here: %v", err)
	}
	if d.Addr() == "" {
		t.Error("Addr must report the bound address after Listen")
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	// Serve blocks; cancelling the context makes it return.
	go func() { _ = d.Serve(ctx) }()
}
