package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
