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

// A unix socket is reachable exactly where writing already is.
//
// Both halves are asked of the kernel rather than of the profile text, and both
// are needed: the first version of this rule denied every unix socket and every
// listening port, which shut the daemon out and took the test suite with it.
//
// The sockets live under the home directory because that is somewhere
// workspace-write does not grant — the same standing as /var/run, where a
// container runtime listens, and unlike /tmp, which the mode does grant.
func TestAUnixSocketIsReachableWhereWritingIs(t *testing.T) {
	if _, err := exec.LookPath("nc"); err != nil {
		t.Skipf("nc is needed to attempt the connection: %v", err)
	}
	s, err := New(Config{AllowNetwork: func() bool { return true }}, policy.ModeWorkspaceWrite)
	if err != nil {
		t.Skipf("no sandbox available: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("a place outside the writable set is needed: %v", err)
	}
	// Short directories on purpose: a unix socket path is capped near 104
	// bytes, and t.TempDir() alone already spends most of that on the test name.
	outsideDir, err := os.MkdirTemp(home, "dco")
	if err != nil {
		t.Skipf("a directory outside the writable set could not be created: %v", err)
	}
	defer os.RemoveAll(outsideDir)
	ws, err := os.MkdirTemp(home, "dcw")
	if err != nil {
		t.Skipf("a workspace could not be created: %v", err)
	}
	defer os.RemoveAll(ws)

	outside := listen(t, filepath.Join(outsideDir, "p.sock"))
	inside := listen(t, filepath.Join(ws, "p.sock"))

	r := Runner{Sandbox: s, Mode: Fixed(policy.ModeWorkspaceWrite)}

	out, code, err := r.Run(context.Background(), ws, "nc -U "+shellQuote(outside)+" < /dev/null")
	if err != nil {
		t.Fatalf("running under the sandbox failed outright: %v", err)
	}
	if code == 0 {
		t.Errorf("a socket outside every writable place was reachable (output %q)", out)
	}

	out, code, err = r.Run(context.Background(), ws, "nc -U "+shellQuote(inside)+" < /dev/null")
	if err != nil {
		t.Fatalf("running under the sandbox failed outright: %v", err)
	}
	if code != 0 {
		t.Errorf("a socket inside the workspace must be reachable, got exit %d (output %q)", code, out)
	}
}

// A suite that cannot listen cannot run.
//
// Granting only outbound traffic left httptest.NewServer — and every test that
// stands up a server — failing at `bind: operation not permitted`, so the rule
// written to keep a daemon out shut the suite out with it. Found by an
// unattended session that could not get `make check` to pass and said so
// instead of reporting success.
func TestAPortCanBeBoundInsideTheSandbox(t *testing.T) {
	s, err := New(Config{AllowNetwork: func() bool { return true }}, policy.ModeWorkspaceWrite)
	if err != nil {
		t.Skipf("no sandbox available: %v", err)
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skipf("python3 is needed to bind a port: %v", err)
	}

	r := Runner{Sandbox: s, Mode: Fixed(policy.ModeWorkspaceWrite)}
	out, code, err := r.Run(context.Background(), t.TempDir(),
		`python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); s.listen(1)'`)
	if err != nil {
		t.Fatalf("running under the sandbox failed outright: %v", err)
	}
	if code != 0 {
		t.Errorf("binding a local port must work inside the sandbox, got exit %d: %s", code, out)
	}
}

func listen(t *testing.T, path string) string {
	t.Helper()
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatalf("the probe socket could not be created: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()
	// Proof it answers at all, so a refusal inside is the sandbox and not a
	// socket nobody was listening on.
	c, err := net.Dial("unix", path)
	if err != nil {
		t.Fatalf("the probe socket does not answer outside the sandbox: %v", err)
	}
	c.Close()
	return path
}
