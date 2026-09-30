package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aguinelo/dcode/internal/sandbox"
)

// DefaultSocketPath resolves where the daemon listens and where a client looks
// for it: DCODE_SOCKET when set, and otherwise one path per user.
//
// The default reads nothing else from the environment. It used to come from
// XDG_RUNTIME_DIR or TMPDIR, and a terminal, an SSH session and an app opened
// from the Dock do not agree on those — so two clients of the same person could
// each find "the" daemon and find different ones.
//
// Under /tmp, as tmux does it: the same for every process of the user, and
// short enough for the ~104-byte limit on a Unix socket path on macOS. Its
// directory is the one every sandbox keeps out of reach (sandboxConfig). /tmp
// is shared by every user of the machine, which is what SecureSocketDir is for.
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

// EmbeddedSocketDir is a private directory for a daemon that lives inside one
// client, made in the per-user directory beside the default socket.
//
// There rather than in the temp directory: the per-user directory is what
// every sandbox keeps out of reach, so a session in one terminal cannot reach
// the daemon embedded in another.
func EmbeddedSocketDir() (string, error) {
	dir := filepath.Dir(perUserSocketPath())
	// MkdirAll first for the platforms where secureDir checks nothing and /tmp
	// may not exist; where it checks, it runs on what this made or found.
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("the daemon's socket directory %s could not be made: %w", dir, err)
	}
	if err := secureDir(dir); err != nil {
		return "", err
	}
	return os.MkdirTemp(dir, "tui-")
}

// sandboxConfig is the boundary every command of a session runs under, built
// in one place for the session and for the checks that measure it.
func sandboxConfig(opts Options, allowNetwork func() bool) sandbox.Config {
	return sandbox.Config{
		Backend:      opts.Backend,
		AllowNetwork: allowNetwork,
		// Without these a compiled language cannot build inside the sandbox,
		// so the agent can change files and never check them.
		Scratch:    sandbox.Scratch(opts.Env),
		Sockets:    sandbox.LocalSockets(opts.Env),
		Unreadable: opts.Unreadable,
		Granted:    opts.Granted,
		Writable:   opts.Writable,
		Daemons:    daemonPaths(opts),
	}
}

// daemonPaths are the places a dcode daemon of this user may be listening:
// the one this session answers to, the one DCODE_SOCKET names, and the
// per-user directory where the default and every embedded daemon listen.
//
// All of them, not only this session's own: a daemon is unconfined, and a
// command that reaches any of them can ask for a session in full access or
// answer an approval — its own, or another session's.
func daemonPaths(opts Options) []string {
	out := []string{filepath.Dir(perUserSocketPath())}
	add := func(p string) {
		for _, have := range out {
			if p == "" || within(p, have) {
				return
			}
		}
		out = append(out, p)
	}
	add(opts.DaemonSocket)
	if opts.Env != nil {
		add(opts.Env("DCODE_SOCKET"))
	}
	return out
}

// grantedDaemons are the entries of sandbox.sockets that name a place a daemon
// listens. The grant does not open them, and the daemon says so at boot.
func grantedDaemons(opts Options) []string {
	var out []string
	for _, g := range opts.Granted {
		for _, d := range daemonPaths(opts) {
			if within(resolved(g), resolved(d)) {
				out = append(out, g)
				break
			}
		}
	}
	return out
}

// within reports whether path is dir or sits inside it.
func within(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// resolved is a path with the links in its directory resolved, so /tmp and
// /private/tmp compare equal on macOS. A path that is not there compares as
// written.
func resolved(p string) string {
	if dir, err := filepath.EvalSymlinks(filepath.Dir(p)); err == nil {
		return filepath.Join(dir, filepath.Base(p))
	}
	return filepath.Clean(p)
}
