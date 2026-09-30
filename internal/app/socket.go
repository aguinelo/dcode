package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// DefaultSocketPath resolves where the daemon listens and where a client looks
// for it: DCODE_SOCKET when set, and otherwise one path per user.
//
// The default reads nothing else from the environment. It used to come from
// XDG_RUNTIME_DIR or TMPDIR, and a terminal, an SSH session and an app opened
// from the Dock do not agree on those — so two clients of the same person could
// each find "the" daemon and find different ones.
//
// Under /tmp, as tmux does it: the same for every process of the user, short
// enough for the ~104-byte limit on a Unix socket path on macOS, and out of
// reach of a confined command — every sandbox short of full access gives the
// command a /tmp of its own on Linux, and denies unix sockets outright on
// macOS. /tmp is shared by every user of the machine, which is what
// SecureSocketDir is for.
func DefaultSocketPath(env func(string) string) string {
	if v := env("DCODE_SOCKET"); v != "" {
		return v
	}
	return perUserSocketPath()
}

func perUserSocketPath() string {
	return filepath.Join("/tmp", fmt.Sprintf("dcode-%d", osUID()), "dcode.sock")
}

// SecureSocketDir makes the directory of the per-user default socket, or
// confirms that the one already there is this user's alone, before anything
// listens or connects in it.
//
// A path somebody chose, through DCODE_SOCKET or --socket, is left as chosen:
// its directory is theirs to answer for, and it is usually a shared one.
func SecureSocketDir(path string) error {
	if path != perUserSocketPath() {
		return nil
	}
	return secureDir(filepath.Dir(path))
}

// secureDir refuses a directory another user could have put a socket in.
//
// Refused rather than repaired: a directory that was open to others may already
// hold somebody else's socket, where every client of this user looks, and
// tightening the mode now would not take it back out.
func secureDir(dir string) error {
	if err := os.Mkdir(dir, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("the daemon's socket directory %s could not be made: %w", dir, err)
	}
	fi, err := os.Lstat(dir)
	if err != nil {
		return fmt.Errorf("the daemon's socket directory %s could not be read: %w", dir, err)
	}
	if fi.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("the daemon's socket directory %s is a symlink, not a directory of its own; remove it, or set DCODE_SOCKET", dir)
	}
	if !fi.IsDir() {
		return fmt.Errorf("the daemon's socket directory %s is not a directory; remove it, or set DCODE_SOCKET", dir)
	}
	if uid, known := ownerOf(fi); known && uid != osUID() {
		return fmt.Errorf("the daemon's socket directory %s belongs to uid %d, not to you (uid %d); remove it, or set DCODE_SOCKET", dir, uid, osUID())
	}
	if perm := fi.Mode().Perm(); perm&0o077 != 0 {
		return fmt.Errorf("the daemon's socket directory %s is open to other users (%v); run chmod 700 %s, or remove it", dir, perm, dir)
	}
	return nil
}
