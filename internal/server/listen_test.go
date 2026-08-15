package server

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/session"
)

func serverOn(path string) *Server {
	return New(Config{SocketPath: path, Manager: session.NewManager(4)})
}

// A daemon that is already listening must not be evicted. Removing the socket
// blindly would take a healthy daemon down and hand the terminal to whoever
// started second, which is the opposite of what a lock is for.
func TestASecondDaemonRefusesRatherThanEvictingTheFirst(t *testing.T) {
	path := filepath.Join(t.TempDir(), "d.sock")

	first := serverOn(path)
	if err := first.Listen(); err != nil {
		t.Skipf("cannot bind a unix socket here: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go first.Serve(ctx)

	// Give the accept loop a moment; the check dials, and a socket bound but
	// not yet accepting would read as stale.
	time.Sleep(50 * time.Millisecond)

	err := serverOn(path).Listen()
	if err == nil {
		t.Fatal("a second daemon bound over a live one")
	}
	if !strings.Contains(err.Error(), "already listening") {
		t.Errorf("err = %v, want it to name what is in the way", err)
	}
}

// A socket left behind by a crashed daemon is removed, but only after a
// connection attempt fails. Nothing answers it, so it is not a daemon — it is
// litter, and refusing to start over it would need a person to clean up after
// a crash they did not cause.
func TestAStaleSocketIsClearedRatherThanBlockingStartup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "d.sock")

	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Skipf("cannot bind a unix socket here: %v", err)
	}
	// Close the listener but leave the file: exactly what a crash leaves.
	ln.Close()
	if _, err := os.Stat(path); err != nil {
		t.Skipf("this platform removes the socket on close: %v", err)
	}

	s := serverOn(path)
	if err := s.Listen(); err != nil {
		t.Fatalf("a stale socket blocked startup: %v", err)
	}
	defer s.listener.Close()

	// 0700 is the whole access-control story for the socket, which is why it is
	// not optional and why it is asserted rather than assumed.
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o700 {
		t.Errorf("mode = %v, want 0700 — the socket is the access boundary", fi.Mode().Perm())
	}
}

// No path is an error the caller can act on, not a daemon listening somewhere
// nobody can find.
func TestADaemonWithNoAddressRefusesToStart(t *testing.T) {
	if err := serverOn("").Listen(); err == nil {
		t.Fatal("a daemon with no socket path started")
	}
}

// Serving without listening is a programming error and says so, rather than
// blocking forever on a listener that is not there.
func TestServingBeforeListeningIsRefused(t *testing.T) {
	if err := serverOn(filepath.Join(t.TempDir(), "d.sock")).Serve(context.Background()); err == nil {
		t.Fatal("serving without a listener reported success")
	}
}
