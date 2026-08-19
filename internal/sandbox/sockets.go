package sandbox

import (
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
