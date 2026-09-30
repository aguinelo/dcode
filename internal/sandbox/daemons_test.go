package sandbox

import (
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/policy"
)

// A dcode daemon is unconfined, and a command that can talk to one can ask it
// for a session in full access or answer its own approval. So where a daemon
// listens is denied last, after every allow — Seatbelt takes the last matching
// rule — and a grant does not bring it back.
func TestADaemonIsDeniedAfterEveryAllowInTheProfile(t *testing.T) {
	sock := "/w/d.sock"
	s := &seatbelt{
		bin:          "x",
		allowNetwork: func() bool { return true },
		granted:      []string{sock},
		daemons:      []string{sock, "/var/dcode-501"},
	}
	p, err := s.profile("/w", policy.ModeWorkspaceWrite, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{sock, "/var/dcode-501"} {
		deny := `(deny network-outbound (subpath "` + d + `"))`
		at := strings.LastIndex(p, deny)
		if at < 0 {
			t.Fatalf("the profile does not deny the daemon at %s:\n%s", d, p)
		}
		if allow := strings.LastIndex(p, "(allow network-outbound"); allow > at {
			t.Errorf("an allow comes after the deny for %s, and Seatbelt takes the last match:\n%s", d, p)
		}
	}

	full, err := s.profile("/w", policy.ModeFullAccess, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(full, "(deny network-outbound (subpath") {
		t.Errorf("full access promises no boundary and must not keep one:\n%s", full)
	}
}

// On Linux the daemon's socket is covered like a container runtime's, and a
// grant does not uncover it. One under /tmp is left alone: the command's /tmp
// is the sandbox's own tmpfs, so the host's socket is not in there to cover.
func TestADaemonIsCoveredInTheArgumentsEvenWhenGranted(t *testing.T) {
	defer func(orig func(string) bool) { exists = orig }(exists)
	exists = func(string) bool { return true }
	defer func(orig func(string) bool) { isDir = orig }(isDir)
	isDir = func(p string) bool { return p == "/srv/daemons" }

	b := &bubblewrap{
		bin:     "x",
		granted: []string{"/home/u/d.sock"},
		daemons: []string{"/home/u/d.sock", "/srv/daemons", "/tmp/dcode-1000"},
	}
	args, err := b.args("/w", policy.ModeWorkspaceWrite, nil)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--ro-bind /dev/null /home/u/d.sock") {
		t.Errorf("a granted daemon socket is left reachable: %s", joined)
	}
	if !strings.Contains(joined, "--tmpfs /srv/daemons") {
		t.Errorf("a directory of daemons outside /tmp is left reachable: %s", joined)
	}
	if strings.Contains(joined, "/tmp/dcode-1000") {
		t.Errorf("a daemon under /tmp is already hidden by the tmpfs, and was mounted over: %s", joined)
	}

	full, err := b.args("/w", policy.ModeFullAccess, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(full, " "), "/home/u/d.sock") {
		t.Errorf("full access promises no boundary and must not keep one: %v", full)
	}
}
