package sandbox

import (
	"os"
	"path/filepath"
	"strings"
)

// LocalSockets are the unix sockets on this machine that lead OUT of the
// sandbox: a container runtime listening for orders.
//
// Reaching one of them is not reaching the network. It is reaching a privileged
// process that is not confined, and asking it to act. `docker run -v /:/host`
// writes anywhere on the machine, as root, from inside a workspace-write
// sandbox — so the containment the mode promises stops existing the moment the
// socket is reachable.
//
// This list was written because the harness found the hole in itself. Asked to
// fix a guard, and reasoning about the Linux side of CI, a model ran
// `docker version`, `docker images` and then `docker run --rm ubuntu:26.10 ...`
// from inside workspace-write. All three succeeded.
//
// A list is a denylist, and a denylist is weaker than a boundary. On macOS the
// profile denies every unix socket instead, and this list is not consulted; on
// Linux the sandbox binds the whole filesystem read-only by design, so the
// sockets have to be named. Naming them is worse than denying them and better
// than handing them over.
func LocalSockets(env func(string) string) []string {
	fixed := []string{
		"/var/run/docker.sock",
		"/run/docker.sock",
		"/var/run/podman/podman.sock",
		"/run/podman/podman.sock",
		"/var/run/containerd/containerd.sock",
		"/run/containerd/containerd.sock",
		"/var/run/crio/crio.sock",
	}
	if env == nil {
		return fixed
	}

	out := fixed
	// Rootless runtimes put their socket under the user's runtime directory,
	// which is per-user and therefore not on any fixed path.
	if run := env("XDG_RUNTIME_DIR"); run != "" {
		out = append(out,
			filepath.Join(run, "docker.sock"),
			filepath.Join(run, "podman", "podman.sock"),
		)
	}
	// And whatever the user pointed the client at, which is the one path a list
	// of defaults can never guess.
	if host := env("DOCKER_HOST"); strings.HasPrefix(host, "unix://") {
		out = append(out, strings.TrimPrefix(host, "unix://"))
	}
	return out
}

// Unreadable parses the configured list of paths this session may not read.
//
// A list separated by the platform's path separator, like PATH itself, because
// that is the one convention every shell and every user already knows for "a
// list of paths in an environment variable".
//
// `~` is expanded here rather than left to the shell: the value arrives from a
// config file as often as from an export, and a tilde that works in one and not
// the other is a setting that looks broken.
//
// There is no default, and the absence is a decision. Every candidate is needed
// by some ordinary tool — hiding ~/.ssh breaks `git push`, hiding ~/.aws breaks
// the aws CLI — and a default that breaks the ordinary case is a default people
// switch off entirely, which protects nothing at all. Naming what to hide is
// the caller's call, because only the caller knows what this session is for.
func Unreadable(spec string, env func(string) string) []string {
	if strings.TrimSpace(spec) == "" {
		return nil
	}
	var home string
	if env != nil {
		home = env("HOME")
	}

	var out []string
	for _, p := range strings.Split(spec, string(os.PathListSeparator)) {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if home != "" {
			if p == "~" {
				p = home
			} else if strings.HasPrefix(p, "~/") {
				p = filepath.Join(home, p[2:])
			}
		}
		// The home directory itself is refused, for the same reason Scratch
		// refuses to grant it: a rule that covers everything under it is not a
		// named set, it is a different mode wearing one.
		if home != "" && p == home {
			continue
		}
		out = append(out, p)
	}
	return out
}
