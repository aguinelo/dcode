package app

import (
	"fmt"
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
