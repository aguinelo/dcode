package sandbox

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/policy"
)

// The boundary itself, asked of the kernel rather than of the argument list.
//
// On Linux the whole filesystem is bound read-only by design and a unix socket
// is not in any network namespace, so neither --ro-bind nor --unshare-net keeps
// a daemon's socket out of reach: connecting to one is not a write. Each named
// socket is covered with /dev/null instead, and this asks whether the cover
// holds.
//
// The socket is created inside the workspace, which is the most permissive
// place it could be — bound writable, and covered after that bind. If the cover
// survives there it survives under /var/run.
//
// The workspace is deliberately NOT t.TempDir(): on Linux that lives under
// /tmp, which the sandbox replaces with a fresh tmpfs, and the socket would
// then be missing rather than covered — a test passing for the wrong reason.
func TestARealRuntimeSocketIsCoveredInside(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("a workspace outside /tmp is needed and the home directory is unknown: %v", err)
	}
	ws, err := os.MkdirTemp(home, "dcs")
	if err != nil {
		t.Skipf("a workspace outside /tmp could not be created: %v", err)
	}
	defer os.RemoveAll(ws)

	sock := filepath.Join(ws, "docker.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("the probe socket could not be created: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()

	// Proof it answers at all, so a refusal inside is the cover and not an
	// address nobody was listening on.
	c, err := net.Dial("unix", sock)
	if err != nil {
		t.Fatalf("the probe socket does not answer outside the sandbox: %v", err)
	}
	c.Close()

	s, err := New(Config{
		AllowNetwork: func() bool { return true },
		Sockets:      []string{sock},
	}, policy.ModeWorkspaceWrite)
	if err != nil {
		t.Skipf("no sandbox available: %v", err)
	}

	r := Runner{Sandbox: s, Mode: policy.ModeWorkspaceWrite}
	out, code, err := r.Run(context.Background(), ws, "test -S "+shellQuote(sock))
	if err != nil {
		t.Fatalf("running under the sandbox failed outright: %v", err)
	}
	if code == 0 {
		t.Errorf("the socket is still a socket inside the sandbox (output %q)", out)
	}
}
