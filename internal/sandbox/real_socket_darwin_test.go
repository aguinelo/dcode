package sandbox

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/policy"
)

// The boundary itself, asked of the kernel rather than of the profile text.
//
// A unix socket is how an unconfined privileged process is reached from inside
// the sandbox, so the assertion is that no unix socket can be connected to at
// all — not that one particular daemon is absent. The socket here is inside the
// workspace and answers outside it, so a refusal inside is the sandbox and not
// a socket nobody was listening on.
func TestARealUnixSocketCannotBeReachedFromInside(t *testing.T) {
	if _, err := exec.LookPath("nc"); err != nil {
		t.Skipf("nc is needed to attempt the connection: %v", err)
	}
	s, err := New(Config{AllowNetwork: func() bool { return true }}, policy.ModeWorkspaceWrite)
	if err != nil {
		t.Skipf("no sandbox available: %v", err)
	}

	ws := t.TempDir()
	// A short directory on purpose: a unix socket path is capped near 104
	// bytes and t.TempDir() alone already spends most of that on the test name.
	dir, err := os.MkdirTemp("/tmp", "dcs")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	sock := filepath.Join(dir, "probe.sock")
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

	// Proof the socket answers at all, so a refusal inside means the sandbox
	// and not a socket nobody was listening on.
	c, err := net.Dial("unix", sock)
	if err != nil {
		t.Fatalf("the probe socket does not answer outside the sandbox: %v", err)
	}
	c.Close()

	r := Runner{Sandbox: s, Mode: policy.ModeWorkspaceWrite}
	out, code, err := r.Run(context.Background(), ws, "nc -U "+shellQuote(sock)+" < /dev/null")
	if err != nil {
		t.Fatalf("running under the sandbox failed outright: %v", err)
	}
	if code == 0 {
		t.Errorf("a unix socket was reachable from inside the sandbox (output %q)", out)
	}
}
