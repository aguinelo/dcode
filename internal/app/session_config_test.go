package app

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/pkg/client"
)

// One daemon serves every project — the desktop starts it outside any of
// them, and a TUI attaches to whichever is running — so a session has to read
// the configuration of its own workspace, the project's file included, and
// not the one the daemon happened to start in.
func TestEachSessionReadsTheConfigurationOfItsOwnWorkspace(t *testing.T) {
	env := envFrom(map[string]string{"DCODE_HOME": t.TempDir()})
	booted := t.TempDir()
	other := t.TempDir()
	projectConfig(t, other, "[sandbox]\nmode = \"read-only\"\n\n[done]\ntimeout = \"3m\"\n")

	d, c := daemonBootedIn(t, env, booted)
	ctx := context.Background()

	sess, err := c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: other})
	if err != nil {
		t.Fatal(err)
	}
	if sess.SandboxMode != "read-only" {
		t.Errorf("the session in %s runs %q, from the workspace the daemon started in, not the read-only its own project asks for", other, sess.SandboxMode)
	}
	home, err := c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: booted})
	if err != nil {
		t.Fatal(err)
	}
	if home.SandboxMode != "workspace-write" {
		t.Errorf("the session in the daemon's own workspace runs %q, want the default workspace-write", home.SandboxMode)
	}

	// The measurement of a definition of done reads the same chain.
	q, err := d.qualifyOptions(other, protocol.CreateSessionRequest{Workspace: other, LoopSpec: "specs/x"})
	if err != nil {
		t.Fatal(err)
	}
	if q.DoneTimeout != 3*time.Minute {
		t.Errorf("criteria in %s are timed at %v, not the 3m its project sets", other, q.DoneTimeout)
	}
}

// A project configuration that cannot be read refuses the session and names
// what is wrong, rather than opening it with the daemon's configuration — a
// session that silently ignored its project's file would run under rules
// nobody in that project chose.
func TestAProjectConfigurationThatCannotBeReadRefusesTheSession(t *testing.T) {
	env := envFrom(map[string]string{"DCODE_HOME": t.TempDir()})
	broken := t.TempDir()
	projectConfig(t, broken, "[sandbox]\nmoed = \"read-only\"\n")

	_, c := daemonBootedIn(t, env, t.TempDir())
	_, err := c.CreateSession(context.Background(), protocol.CreateSessionRequest{Workspace: broken})
	if err == nil {
		t.Fatal("a session opened in a workspace whose configuration cannot be read")
	}
	if !strings.Contains(err.Error(), "moed") {
		t.Errorf("refused without saying what is wrong: %v", err)
	}
}

func projectConfig(t *testing.T, ws, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(ws, ".dcode"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, ".dcode", "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// daemonBootedIn starts a daemon whose configuration was resolved in ws, as
// `dcode serve` resolves it in the directory it starts in.
func daemonBootedIn(t *testing.T, env func(string) string, ws string) (*Daemon, *client.Client) {
	t.Helper()
	base, _, err := FromEnv(env, ws)
	if err != nil {
		t.Fatal(err)
	}
	requireSandbox(t, base)
	dir, err := os.MkdirTemp("", "dc")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	d := NewDaemon(DaemonOptions{SocketPath: filepath.Join(dir, "d.sock"), Base: base})
	if err := d.Listen(); err != nil {
		t.Skipf("cannot bind a unix socket here: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go d.Serve(ctx)
	c := client.New(d.Addr())
	for deadline := time.Now().Add(3 * time.Second); c.Health(ctx) != nil; {
		if time.Now().After(deadline) {
			t.Fatal("the daemon never became healthy")
		}
		time.Sleep(5 * time.Millisecond)
	}
	return d, c
}

// memory.enabled and memory.max_entries were read by fromResolved and absent
// from KnownKeys, so neither could be set at all: a config.toml naming them was
// refused as an unknown key, and the environment layer only looks up the
// variables KnownKeys maps, so DCODE_MEMORY_ENABLED was never consulted either.
// The learned-memory spec declared both, with their variables, and the only
// value a session could ever see was the default.
func TestTheMemoryKeysReachASession(t *testing.T) {
	ws := t.TempDir()
	projectConfig(t, ws, "[memory]\nenabled = false\nmax_entries = 5\n")
	opts, _, err := FromEnv(envFrom(map[string]string{"DCODE_HOME": t.TempDir()}), ws)
	if err != nil {
		t.Fatalf("a config.toml setting the memory keys is refused: %v", err)
	}
	if opts.Memory || opts.MemoryMax != 5 {
		t.Errorf("from config.toml the session has memory=%v max=%d, want false and 5", opts.Memory, opts.MemoryMax)
	}

	opts, _, err = FromEnv(envFrom(map[string]string{
		"DCODE_HOME":               t.TempDir(),
		"DCODE_MEMORY_ENABLED":     "false",
		"DCODE_MEMORY_MAX_ENTRIES": "7",
	}), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if opts.Memory || opts.MemoryMax != 7 {
		t.Errorf("from the environment the session has memory=%v max=%d, want false and 7", opts.Memory, opts.MemoryMax)
	}
}

// The wiring guard starts from KnownKeys and asks whether each key is read. It
// never asked the reverse, and the memory keys sat in that gap: read by name in
// fromResolved, unknown to the schema, so no layer but the default could ever
// reach them. A key FromEnv reads that KnownKeys does not hold is a setting
// that looks configurable in the code and is not.
func TestFromEnvReadsOnlyKnownKeys(t *testing.T) {
	read := regexp.MustCompile(`r\.(?:Bool|String|Int)\("([^"]+)"`)
	matches := read.FindAllStringSubmatch(readFromEnvBody(t), -1)
	if len(matches) == 0 {
		t.Fatal("no accessor calls found in FromEnv; the guard would pass vacuously")
	}
	for _, m := range matches {
		if _, ok := config.KnownKeys[m[1]]; !ok {
			t.Errorf("FromEnv reads %q, which KnownKeys does not hold: no file and no variable can set it", m[1])
		}
	}
}
