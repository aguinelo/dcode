package sandbox

import (
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/policy"
)

// A socket reaches out of the sandbox by design, so granting one is naming the
// exception rather than widening the rule.
//
// The rule stands: a unix socket is reachable exactly where writing already is,
// which keeps a container runtime out. A session that coordinates machines needs
// one specific socket back — the ssh-agent's — and naming it is how that is said
// without saying "any socket".
func TestSeatbeltReachesAGrantedSocket(t *testing.T) {
	s := &seatbelt{
		bin:          "sandbox-exec",
		allowNetwork: func() bool { return true },
		granted:      []string{"/private/tmp/agent.sock"},
	}
	p, err := s.profile("/w", policy.ModeWorkspaceWrite, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(p, `(allow network-outbound (literal "/private/tmp/agent.sock"))`) {
		t.Errorf("the granted socket is still out of reach:\n%s", p)
	}
}

// A path granted for writing is added to the writable set, which means it is
// also reachable as a socket — the two answers come from one list, and that is
// what keeps them from disagreeing.
func TestSeatbeltWritesToAGrantedPath(t *testing.T) {
	s := &seatbelt{bin: "sandbox-exec", writable: []string{"/Users/me/.ssh/known_hosts"}}
	p, _ := s.profile("/w", policy.ModeWorkspaceWrite, nil)

	if !strings.Contains(p, `(allow file-write* (subpath "/Users/me/.ssh/known_hosts"))`) {
		t.Errorf("the granted path is not writable:\n%s", p)
	}
}

// Read-only grants no writable path. A mode that writes nowhere does not gain a
// place to write by someone naming one.
func TestSeatbeltReadOnlyIgnoresAGrantedPath(t *testing.T) {
	s := &seatbelt{bin: "sandbox-exec", writable: []string{"/Users/me/.ssh/known_hosts"}}
	p, _ := s.profile("/w", policy.ModeReadOnly, nil)

	if strings.Contains(p, "known_hosts") {
		t.Errorf("read-only must write nowhere:\n%s", p)
	}
}

// On Linux the covering is what keeps a runtime socket out, so a granted one is
// simply not covered.
func TestBubblewrapLeavesAGrantedSocketAlone(t *testing.T) {
	restore := exists
	exists = func(string) bool { return true }
	defer func() { exists = restore }()

	b := &bubblewrap{
		bin:     "bwrap",
		sockets: []string{"/var/run/docker.sock", "/run/user/1000/keyring/ssh"},
		granted: []string{"/run/user/1000/keyring/ssh"},
	}
	args, err := b.args("/w", policy.ModeWorkspaceWrite, nil)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "--ro-bind /dev/null /var/run/docker.sock") {
		t.Errorf("an ungranted socket must stay covered:\n%s", joined)
	}
	if strings.Contains(joined, "/dev/null /run/user/1000/keyring/ssh") {
		t.Errorf("a granted socket must not be covered:\n%s", joined)
	}
}

func TestBubblewrapBindsAGrantedPath(t *testing.T) {
	restore := exists
	exists = func(string) bool { return true }
	defer func() { exists = restore }()

	b := &bubblewrap{bin: "bwrap", writable: []string{"/home/me/.ssh/known_hosts"}}
	args, _ := b.args("/w", policy.ModeWorkspaceWrite, nil)

	if !strings.Contains(strings.Join(args, " "), "--bind /home/me/.ssh/known_hosts /home/me/.ssh/known_hosts") {
		t.Errorf("the granted path is not writable:\n%s", strings.Join(args, " "))
	}
}

// The two halves only work together, and this is where they meet.
//
// Hiding a private key while ssh has to read it breaks every connection;
// granting the agent while the key stays readable protects nothing. With the
// agent reachable, ssh asks it to sign and never opens the key — so hiding it
// costs nothing, and the default takes it.
func TestTheKeyIsHiddenOnceTheAgentIsReachable(t *testing.T) {
	env := func(k string) string {
		switch k {
		case "HOME":
			return "/Users/me"
		case "SSH_AUTH_SOCK":
			return "/private/tmp/agent.sock"
		}
		return ""
	}
	agent := Paths(SSHAgentToken, env)
	if len(agent) != 1 || agent[0] != "/private/tmp/agent.sock" {
		t.Fatalf("the token must stand for whatever SSH_AUTH_SOCK holds, got %v", agent)
	}

	withAgent := strings.Join(Unreadable("", env, agent), " ")
	if !strings.Contains(withAgent, "/Users/me/.ssh") {
		t.Errorf("with the agent reachable the key costs nothing to hide: %s", withAgent)
	}

	without := strings.Join(Unreadable("", env, nil), " ")
	if strings.Contains(without, "/Users/me/.ssh") {
		t.Errorf("without the agent, hiding the key would stop every ssh: %s", without)
	}
}

// No agent running is not an error and not a grant. Naming the token when
// nothing answers must not leave an empty path standing for everything.
func TestTheAgentTokenGrantsNothingWhenNoAgentIsRunning(t *testing.T) {
	env := func(string) string { return "" }
	if got := Paths(SSHAgentToken, env); got != nil {
		t.Errorf("no agent must grant nothing, got %v", got)
	}
}
