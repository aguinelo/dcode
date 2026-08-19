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
// Unset means the default set below, not "nothing". A session that never asked
// the question should still not be able to read a cloud credential.
//
// Setting it REPLACES the default, so a session that needs one of them back can
// say so; the literal "none" hides nothing at all, which is the only way to say
// that without a magic empty string.
func Unreadable(spec string, env func(string) string, granted []string) []string {
	if strings.TrimSpace(spec) == "" {
		return DefaultUnreadable(env, granted)
	}
	if strings.TrimSpace(spec) == "none" {
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

// DefaultUnreadable are the credential stores hidden when nobody said otherwise.
//
// The rule for being on this list is narrow: it holds a secret, and no ordinary
// tool needs to read it as a subprocess of an ordinary command. That is what
// makes hiding it free. `aws` and `kubectl` are the exceptions people will meet
// first, and they are the reason the setting REPLACES this list rather than
// adding to it.
//
// `~/.ssh` joins the list only when the agent's socket has been granted, and
// that condition is the whole design. A private key is the canonical secret,
// but hiding it while ssh must read it stops `git push` and every connection.
// With SSH_AUTH_SOCK reachable, ssh asks the agent to sign and never opens the
// key — so hiding it costs nothing, and the default takes it. Apart, each half
// is a bad trade; together there is none.
func DefaultUnreadable(env func(string) string, granted []string) []string {
	if env == nil {
		return nil
	}
	home := env("HOME")
	if home == "" {
		return nil
	}

	out := []string{
		filepath.Join(home, ".aws"),
		filepath.Join(home, ".gnupg"),
		filepath.Join(home, ".kube"),
		filepath.Join(home, ".config", "gcloud"),
		filepath.Join(home, ".azure"),
		filepath.Join(home, ".netrc"),
		filepath.Join(home, ".git-credentials"),
		filepath.Join(home, ".npmrc"),
		filepath.Join(home, ".pypirc"),
		filepath.Join(home, ".docker", "config.json"),
	}
	// The private key, but only once the agent can sign in its place.
	//
	// The two halves are worthless apart: hiding the key while ssh must read
	// it stops every connection, and granting the agent while the key stays
	// readable protects nothing. Together there is no trade, and this line is
	// where they meet.
	if agent := env("SSH_AUTH_SOCK"); agent != "" && contains(granted, agent) {
		out = append(out, filepath.Join(home, ".ssh"))
	}
	// And dcode's own key. A session that can read the credential it runs on
	// can hand it to anything it can write to, and the redaction that keeps it
	// out of transcripts does nothing about a file read.
	for _, root := range []string{
		filepath.Join(home, "Library", "Application Support", "dcode"),
		filepath.Join(home, ".config", "dcode"),
		filepath.Join(home, ".dcode"),
	} {
		out = append(out, filepath.Join(root, credentialFileName))
	}
	return out
}

// credentialFileName mirrors credential.FileName. Named here rather than
// imported because this package builds profiles and must not depend on the
// store it is hiding; the guard below is the test that keeps the two in step.
const credentialFileName = "credentials"

// SSHAgentToken is what a user writes instead of a path they cannot know.
//
// The agent's socket is per-boot and per-login — /var/run/com.apple.launchd.*
// on macOS, $XDG_RUNTIME_DIR elsewhere — so no configuration file can name it
// and no default can guess it. The token stands for whatever SSH_AUTH_SOCK
// holds at the time.
const SSHAgentToken = "ssh-agent"

// Paths parses one of the configured path lists, expanding `~` and the
// ssh-agent token, in the same shape Unreadable already reads.
//
// Shared by the granted-socket list and the writable list because they are the
// same kind of value said about different things, and two parsers would drift
// into disagreeing about what a tilde means.
func Paths(spec string, env func(string) string) []string {
	if strings.TrimSpace(spec) == "" {
		return nil
	}
	var home, agent string
	if env != nil {
		home = env("HOME")
		agent = env("SSH_AUTH_SOCK")
	}

	var out []string
	for _, p := range strings.Split(spec, string(os.PathListSeparator)) {
		p = strings.TrimSpace(p)
		switch {
		case p == "":
			continue
		case p == SSHAgentToken:
			// Nothing to grant when no agent is running, and granting the
			// empty string would name the whole filesystem in some backends.
			if agent == "" {
				continue
			}
			p = agent
		case home != "" && p == "~":
			p = home
		case home != "" && strings.HasPrefix(p, "~/"):
			p = filepath.Join(home, p[2:])
		}
		out = append(out, p)
	}
	return out
}

// contains reports membership, for the one condition above.
func contains(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}
