package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/pkg/client"
)

// The same person has to find the same daemon from a terminal, an SSH session
// and an app opened from the Dock, and those three do not agree on
// XDG_RUNTIME_DIR or TMPDIR. So the default reads neither.
func TestTheDefaultSocketDoesNotDependOnTheEnvironment(t *testing.T) {
	want := fmt.Sprintf("/tmp/dcode-%d/dcode.sock", os.Getuid())
	for name, env := range map[string]map[string]string{
		"a terminal with a runtime dir": {"XDG_RUNTIME_DIR": "/run/user/1000", "TMPDIR": "/var/folders/xy/T/"},
		"an app opened from the Dock":   {"TMPDIR": "/var/folders/xy/T/"},
		"an SSH session with neither":   {},
	} {
		if got := DefaultSocketPath(envFrom(env)); got != want {
			t.Errorf("%s: the default socket is %q, want %q", name, got, want)
		}
	}

	chosen := envFrom(map[string]string{"DCODE_SOCKET": "/srv/d.sock", "XDG_RUNTIME_DIR": "/run/user/1000"})
	if got := DefaultSocketPath(chosen); got != "/srv/d.sock" {
		t.Errorf("DCODE_SOCKET is the explicit choice and wins, got %q", got)
	}

	// A Unix socket path is capped near 104 bytes on macOS.
	if len(want) > 100 {
		t.Errorf("%q is %d bytes, too long for a unix socket", want, len(want))
	}
}

// /tmp is shared by every user of the machine, so the directory the default
// socket lives in is used only when it is this user's alone. A directory
// someone else made first — open to others, a symlink, or theirs — would let
// them put their own socket where every client of this user looks.
func TestASocketDirectoryNotOwnedAloneIsRefused(t *testing.T) {
	base := t.TempDir()

	fresh := filepath.Join(base, "fresh")
	if err := secureDir(fresh); err != nil {
		t.Fatalf("a directory that is not there yet is not made: %v", err)
	}
	fi, err := os.Lstat(fresh)
	if err != nil || !fi.IsDir() || fi.Mode().Perm() != 0o700 {
		t.Fatalf("made %v (%v), want a directory with 0700", fi.Mode(), err)
	}
	if err := secureDir(fresh); err != nil {
		t.Errorf("this user's own 0700 directory is refused: %v", err)
	}

	open := filepath.Join(base, "open")
	if err := os.Mkdir(open, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(open, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := secureDir(open); err == nil || !strings.Contains(err.Error(), open) {
		t.Errorf("a directory open to others was accepted, or refused without naming it: %v", err)
	}

	link := filepath.Join(base, "link")
	if err := os.Symlink(fresh, link); err != nil {
		t.Fatal(err)
	}
	if err := secureDir(link); err == nil {
		t.Error("a symlink was accepted as the socket's directory")
	}

	// Somebody else's: a system directory this user did not make. Root owns
	// everything and proves nothing here.
	if os.Getuid() != 0 {
		if err := secureDir("/usr"); err == nil || !strings.Contains(err.Error(), "uid") {
			t.Errorf("a directory owned by another user was accepted, or refused without saying whose: %v", err)
		}
	}
}

// A path somebody chose, through DCODE_SOCKET or --socket, is theirs to answer
// for, and its directory is usually a shared one — /tmp itself, often.
func TestAChosenSocketIsLeftAsChosen(t *testing.T) {
	if err := SecureSocketDir("/tmp/chosen.sock"); err != nil {
		t.Errorf("a chosen path was checked like the default: %v", err)
	}
}

// Every sandbox a daemon builds keeps its user's daemons out of reach: its own
// socket, the one DCODE_SOCKET names, and the per-user directory where the
// default and every embedded daemon listen — so a session in one terminal
// cannot reach the daemon behind another either.
func TestEverySandboxKeepsTheDaemonsOutOfReach(t *testing.T) {
	d := NewDaemon(DaemonOptions{SocketPath: "/srv/own.sock", Base: Options{
		Env: envFrom(map[string]string{"DCODE_SOCKET": "/srv/chosen.sock"}),
	}})
	if got := d.opts.Base.DaemonSocket; got != "/srv/own.sock" {
		t.Fatalf("the sessions of a daemon do not know its socket, got %q", got)
	}
	cfg := sandboxConfig(d.opts.Base, nil)
	for _, want := range []string{"/srv/own.sock", "/srv/chosen.sock", filepath.Dir(perUserSocketPath())} {
		if !slices.Contains(cfg.Daemons, want) {
			t.Errorf("the sandbox does not keep %s out of reach: %v", want, cfg.Daemons)
		}
	}
}

// Naming a daemon's socket in sandbox.sockets does not open it, and the daemon
// says so once, where it starts, rather than leaving a grant that silently does
// nothing.
func TestAGrantOfADaemonSocketIsSaidAtBoot(t *testing.T) {
	var said []string
	NewDaemon(DaemonOptions{
		SocketPath: "/srv/own.sock",
		Base:       Options{Granted: []string{"/srv/own.sock", "/run/agent.sock"}},
		Log:        func(m string) { said = append(said, m) },
	})
	if len(said) != 1 || !strings.Contains(said[0], "/srv/own.sock") || strings.Contains(said[0], "agent") {
		t.Errorf("the grant of the daemon's own socket was said as %q", said)
	}
}

// An embedded daemon listens in the per-user directory too, beside the default
// one, which is what puts it out of reach of every other session's sandbox.
func TestAnEmbeddedDaemonListensBesideTheOthers(t *testing.T) {
	dir, err := EmbeddedSocketDir()
	if err != nil {
		t.Fatalf("no place for an embedded daemon: %v", err)
	}
	defer os.RemoveAll(dir)
	if filepath.Dir(dir) != filepath.Dir(perUserSocketPath()) {
		t.Errorf("the embedded daemon listens in %s, outside %s", dir, filepath.Dir(perUserSocketPath()))
	}
	if fi, err := os.Stat(dir); err != nil || fi.Mode().Perm() != 0o700 {
		t.Errorf("the embedded daemon's directory is %v (%v), want 0700", fi.Mode(), err)
	}
}

// A confined command cannot reach the daemon that confines it, through the
// daemon's own wiring: a real daemon, a real session, a command run inside its
// sandbox. Placed where a socket was reachable before: under /tmp on macOS,
// which workspace-write makes writable, and outside /tmp on Linux, where only
// /tmp is replaced. A socket in the workspace, reachable by design, is the
// control that proves the command could connect at all.
func TestAConfinedCommandCannotReachItsDaemon(t *testing.T) {
	if _, err := exec.LookPath("nc"); err != nil {
		t.Skipf("nc is needed to attempt the connection: %v", err)
	}
	parent := "/tmp"
	if runtime.GOOS != "darwin" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skipf("a place outside /tmp is needed: %v", err)
		}
		parent = home
	}
	dir, err := os.MkdirTemp(parent, "dcd")
	if err != nil {
		t.Skipf("no place for the daemon: %v", err)
	}
	defer os.RemoveAll(dir)
	ws, err := os.MkdirTemp("", "dcw")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(ws)
	if ws, err = filepath.EvalSymlinks(ws); err != nil {
		t.Fatal(err)
	}

	base, _, err := FromEnv(envFrom(map[string]string{}), ws)
	if err != nil {
		t.Fatal(err)
	}
	base.SandboxMode = policy.ModeWorkspaceWrite
	base.AllowNetwork = true
	requireSandbox(t, base)

	sock := filepath.Join(dir, "d.sock")
	d := NewDaemon(DaemonOptions{SocketPath: sock, Base: base})
	if err := d.Listen(); err != nil {
		t.Skipf("cannot bind a unix socket here: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go d.Serve(ctx)
	c := client.New(sock)
	for deadline := time.Now().Add(3 * time.Second); c.Health(ctx) != nil; {
		if time.Now().After(deadline) {
			t.Fatal("the daemon never became healthy")
		}
		time.Sleep(5 * time.Millisecond)
	}

	sess, err := c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: ws, SandboxMode: "workspace-write"})
	if err != nil {
		t.Fatal(err)
	}
	events, _ := c.Subscribe(ctx, sess.ID, 1)
	control := filepath.Join(ws, "o.sock")
	probe := fmt.Sprintf("nc -U %s < /dev/null && echo CONTROL; nc -U %s < /dev/null && echo REACHED", control, sock)
	stop := serveProbe(t, control)
	defer stop()
	if err := c.Exec(ctx, sess.ID, probe); err != nil {
		t.Fatal(err)
	}

	timeout := time.After(20 * time.Second)
	for {
		select {
		case ev := <-events:
			switch ev.Type {
			case protocol.EventApprovalRequired:
				var req protocol.ApprovalRequest
				if err := json.Unmarshal(ev.Payload, &req); err != nil {
					t.Fatal(err)
				}
				if err := c.Resolve(ctx, sess.ID, req.ApprovalID, protocol.ApprovalAllow); err != nil {
					t.Fatal(err)
				}
			case protocol.EventToolCompleted:
				var done protocol.ToolCompleted
				if err := json.Unmarshal(ev.Payload, &done); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(done.Output, "CONTROL") {
					t.Fatalf("the command could not reach an ordinary socket either, so it proves nothing: %q", done.Output)
				}
				if strings.Contains(done.Output, "REACHED") {
					t.Fatalf("a command inside the session's sandbox reached the daemon at %s: %q", sock, done.Output)
				}
				return
			}
		case <-timeout:
			t.Fatal("the command never completed")
		}
	}
}

// serveProbe listens on path and closes whatever connects: enough for nc to
// report a connection, and nothing that could be mistaken for the daemon.
func serveProbe(t *testing.T, path string) (stop func()) {
	t.Helper()
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatalf("the control socket could not be created: %v", err)
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()
	return func() { ln.Close() }
}
