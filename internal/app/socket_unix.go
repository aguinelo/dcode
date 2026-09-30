//go:build unix

package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"
)

// secureDir refuses a directory another user could have put a socket in.
//
// Refused rather than repaired: a directory that was open to others may
// already hold somebody else's socket, where every client of this user looks,
// and tightening the mode now would not take it back out.
func secureDir(dir string) error {
	if err := os.Mkdir(dir, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("the daemon's socket directory %s could not be made: %w", dir, err)
	}
	fi, err := os.Lstat(dir)
	if err != nil {
		return fmt.Errorf("the daemon's socket directory %s could not be read: %w", dir, err)
	}
	if fi.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("the daemon's socket directory %s is a symlink, not a directory of its own; remove it, or set DCODE_SOCKET to a path of your own", dir)
	}
	if !fi.IsDir() {
		return fmt.Errorf("the daemon's socket directory %s is not a directory; remove it, or set DCODE_SOCKET to a path of your own", dir)
	}
	// /tmp keeps each user's files from the others: only their owner, or root,
	// can remove this one, so the way out is a path of one's own.
	if st, ok := fi.Sys().(*syscall.Stat_t); ok && int(st.Uid) != osUID() {
		return fmt.Errorf("the daemon's socket directory %s belongs to uid %d, not to you (uid %d); set DCODE_SOCKET to a path of your own", dir, st.Uid, osUID())
	}
	if perm := fi.Mode().Perm(); perm&0o077 != 0 {
		return fmt.Errorf("the daemon's socket directory %s is open to other users (%v), so something of theirs may already be in it; remove it (rm -r %s) and dcode makes it again", dir, perm, dir)
	}
	return nil
}
