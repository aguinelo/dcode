package sandbox

import (
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/policy"
)

// The grant is for the network, not for this machine's own daemons.
//
// `(allow network*)` also covers unix sockets, and a unix socket is how an
// unconfined privileged process is reached: `docker run -v /:/host` writes
// anywhere, as root, from inside workspace-write. Found by the harness in
// itself — an unattended session ran `docker version`, `docker images` and
// `docker run --rm ubuntu:26.10 ...` from inside the sandbox, and all three
// succeeded.
func TestSeatbeltGrantsTheNetworkWithoutTheMachinesOwnSockets(t *testing.T) {
	s := &seatbelt{bin: "sandbox-exec", allowNetwork: func() bool { return true }}
	p, err := s.profile("/w", policy.ModeWorkspaceWrite, nil)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(p, "(allow network*)") {
		t.Errorf("a blanket network grant also grants every unix socket:\n%s", p)
	}
	if !strings.Contains(p, `(allow network-outbound (remote ip "*:*"))`) {
		t.Errorf("granting the network must still grant IP traffic:\n%s", p)
	}
}

// Denying every unix socket denies name resolution too: on macOS getaddrinfo
// talks to mDNSResponder over one. It is named because it resolves names — it
// does not act on the caller's behalf, which is the property that separates it
// from a container runtime's socket.
func TestSeatbeltStillResolvesNamesWhenTheNetworkIsGranted(t *testing.T) {
	s := &seatbelt{bin: "sandbox-exec", allowNetwork: func() bool { return true }}
	p, _ := s.profile("/w", policy.ModeWorkspaceWrite, nil)
	if !strings.Contains(p, "mDNSResponder") {
		t.Errorf("DNS needs the resolver socket named:\n%s", p)
	}
}

// Full access claims no boundary at all, so it keeps the blanket grant. Making
// it narrower would be pretending to confine a mode that says it does not.
func TestSeatbeltFullAccessKeepsTheBlanketGrant(t *testing.T) {
	s := &seatbelt{bin: "sandbox-exec"}
	p, _ := s.profile("/w", policy.ModeFullAccess, nil)
	if !strings.Contains(p, "(allow network*)") {
		t.Errorf("full access must not be narrowed:\n%s", p)
	}
}

// bubblewrap binds the whole filesystem read-only by design, and connecting to
// a unix socket is not a write — so a read-only bind hands the socket over. The
// sockets have to be named and covered one by one.
func TestBubblewrapCoversAContainerRuntimeSocket(t *testing.T) {
	present := "/var/run/docker.sock"
	restore := exists
	exists = func(p string) bool { return p == present }
	defer func() { exists = restore }()

	b := &bubblewrap{bin: "bwrap", sockets: []string{present, "/run/podman/podman.sock"}}
	args, err := b.args("/w", policy.ModeWorkspaceWrite, nil)
	if err != nil {
		t.Fatal(err)
	}

	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--ro-bind /dev/null "+present) {
		t.Errorf("the runtime socket is still reachable:\n%s", joined)
	}
	if strings.Contains(joined, "/run/podman/podman.sock") {
		t.Errorf("bubblewrap refuses a bind whose target is absent, so an absent socket must be skipped:\n%s", joined)
	}
}

// Full access claims no boundary, so it covers nothing.
func TestBubblewrapFullAccessCoversNothing(t *testing.T) {
	restore := exists
	exists = func(string) bool { return true }
	defer func() { exists = restore }()

	b := &bubblewrap{bin: "bwrap", sockets: []string{"/var/run/docker.sock"}}
	args, _ := b.args("/w", policy.ModeFullAccess, nil)
	if strings.Contains(strings.Join(args, " "), "/dev/null") {
		t.Error("full access must not be narrowed")
	}
}

func TestLocalSocketsNamesTheFixedPathsWithoutAnEnvironment(t *testing.T) {
	got := LocalSockets(nil)
	if !containsString(got, "/var/run/docker.sock") {
		t.Errorf("the default docker socket must be named: %v", got)
	}
}

// A rootless runtime puts its socket under the user's runtime directory, and a
// client can be pointed anywhere at all. Neither is on a fixed path, so neither
// can be guessed from a list of defaults.
func TestLocalSocketsFollowsTheEnvironmentItIsGiven(t *testing.T) {
	env := map[string]string{
		"XDG_RUNTIME_DIR": "/run/user/1000",
		"DOCKER_HOST":     "unix:///home/me/.docker/run/docker.sock",
	}
	got := LocalSockets(func(k string) string { return env[k] })

	for _, want := range []string{
		"/run/user/1000/docker.sock",
		"/run/user/1000/podman/podman.sock",
		"/home/me/.docker/run/docker.sock",
	} {
		if !containsString(got, want) {
			t.Errorf("%s was not named: %v", want, got)
		}
	}
}

// A TCP DOCKER_HOST is not a socket on this machine, and adding it as a path
// would ask bubblewrap to bind something that is not there.
func TestLocalSocketsIgnoresARemoteDockerHost(t *testing.T) {
	got := LocalSockets(func(k string) string {
		if k == "DOCKER_HOST" {
			return "tcp://10.0.0.1:2375"
		}
		return ""
	})
	for _, p := range got {
		if strings.Contains(p, "10.0.0.1") {
			t.Errorf("a tcp host is not a local socket: %v", got)
		}
	}
}

func containsString(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}
