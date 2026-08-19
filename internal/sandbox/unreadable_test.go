package sandbox

import (
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/credential"
	"github.com/aguinelo/dcode/internal/policy"
)

// Reading is allowed everywhere, and that is deliberate — refusing it outright
// would stop the interpreter loading before the command ever runs. The cost is
// that the sandbox protects no secret at all: measured on this machine, a
// command under workspace-write read ~/.ssh/id_ed25519 without a murmur.
//
// For editing code that is a fair trade. For a session that reaches servers it
// is the most valuable thing on the machine left on the table, so a named set
// can be put out of reach.
func TestSeatbeltPutsANamedStoreOutOfReach(t *testing.T) {
	s := &seatbelt{bin: "sandbox-exec", unreadable: []string{"/Users/me/.aws"}}
	p, err := s.profile("/w", policy.ModeWorkspaceWrite, nil)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(p, `(deny file-read* (subpath "/Users/me/.aws"))`) {
		t.Errorf("the named store is still readable:\n%s", p)
	}
	// Order is the whole point: Seatbelt takes the last matching rule, so a
	// deny written before the blanket allow would be overruled by it.
	if strings.Index(p, "(deny file-read*") < strings.Index(p, "(allow file-read*)") {
		t.Error("the deny must come after the blanket allow, or it does nothing")
	}
}

// Full access claims no boundary, so it hides nothing. Pretending otherwise
// would be a mode that says it does not confine while confining.
func TestSeatbeltFullAccessHidesNothing(t *testing.T) {
	s := &seatbelt{bin: "sandbox-exec", unreadable: []string{"/Users/me/.aws"}}
	p, _ := s.profile("/w", policy.ModeFullAccess, nil)
	if strings.Contains(p, "deny file-read*") {
		t.Errorf("full access must not hide anything:\n%s", p)
	}
}

// Naming nothing hides nothing, and the profile is byte-identical to what it
// was before the capability existed.
func TestSeatbeltWithoutANamedStoreIsUnchanged(t *testing.T) {
	with := &seatbelt{bin: "sandbox-exec"}
	p, _ := with.profile("/w", policy.ModeWorkspaceWrite, nil)
	if strings.Contains(p, "deny file-read*") {
		t.Errorf("nothing was named, so nothing is denied:\n%s", p)
	}
}

// bubblewrap binds the whole filesystem read-only by design, so a named store
// is covered rather than left out — the same move the runtime sockets get, and
// for the same reason: the mount is what the kernel reads, not a rule of ours.
func TestBubblewrapCoversANamedStore(t *testing.T) {
	restore := exists
	exists = func(string) bool { return true }
	defer func() { exists = restore }()

	b := &bubblewrap{bin: "bwrap", unreadable: []string{"/home/me/.aws"}}
	args, err := b.args("/w", policy.ModeWorkspaceWrite, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(args, " "), "--tmpfs /home/me/.aws") {
		t.Errorf("the named store is still readable:\n%s", strings.Join(args, " "))
	}
}

func TestBubblewrapFullAccessCoversNoStore(t *testing.T) {
	restore := exists
	exists = func(string) bool { return true }
	defer func() { exists = restore }()

	b := &bubblewrap{bin: "bwrap", unreadable: []string{"/home/me/.aws"}}
	args, _ := b.args("/w", policy.ModeFullAccess, nil)
	if strings.Contains(strings.Join(args, " "), "/home/me/.aws") {
		t.Error("full access must not hide anything")
	}
}

func TestUnreadableExpandsHomeAndRefusesIt(t *testing.T) {
	env := func(k string) string {
		if k == "HOME" {
			return "/Users/me"
		}
		return ""
	}
	got := Unreadable("~/.aws:/etc/secrets: :~", env)

	want := []string{"/Users/me/.aws", "/etc/secrets"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v — the home itself is refused and blanks are dropped", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestUnreadableWithNothingConfiguredNamesNothing(t *testing.T) {
	if got := Unreadable("", nil); got != nil {
		t.Errorf("nothing configured must name nothing, got %v", got)
	}
	if got := Unreadable("   ", nil); got != nil {
		t.Errorf("blank configured must name nothing, got %v", got)
	}
}

// Nobody asking is not nobody caring. A session that never set the key should
// still not be able to read a cloud credential.
func TestUnreadableDefaultsToHidingCredentialStores(t *testing.T) {
	env := func(k string) string {
		if k == "HOME" {
			return "/Users/me"
		}
		return ""
	}
	got := strings.Join(Unreadable("", env), " ")

	for _, want := range []string{
		"/Users/me/.aws", "/Users/me/.gnupg", "/Users/me/.kube",
		"/Users/me/.config/gcloud", "/Users/me/.netrc",
		"/Users/me/.git-credentials", "/Users/me/.docker/config.json",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("%s is readable by default: %s", want, got)
		}
	}
}

// A session that can read the credential it runs on can hand it to anything it
// can write to, and redacting transcripts does nothing about a file read.
func TestTheDefaultHidesDcodesOwnCredential(t *testing.T) {
	env := func(k string) string {
		if k == "HOME" {
			return "/Users/me"
		}
		return ""
	}
	got := strings.Join(Unreadable("", env), " ")
	if !strings.Contains(got, credentialFileName) {
		t.Errorf("dcode's own key is readable by default: %s", got)
	}
}

// The file name is duplicated rather than imported, so that duplication has to
// be checked: a rename on one side that misses the other would leave the key
// readable and nothing would say so.
func TestTheHiddenCredentialNameMatchesTheStore(t *testing.T) {
	if credentialFileName != credential.FileName {
		t.Errorf("hiding %q while the store writes %q", credentialFileName, credential.FileName)
	}
}

// Setting the key replaces the default, and "none" hides nothing — the only way
// to say that without an empty string meaning two different things.
func TestUnreadableCanBeReplacedOrCleared(t *testing.T) {
	env := func(string) string { return "/Users/me" }

	got := Unreadable("/only/this", env)
	if len(got) != 1 || got[0] != "/only/this" {
		t.Errorf("setting the key must replace the default, got %v", got)
	}
	if got := Unreadable("none", env); got != nil {
		t.Errorf(`"none" must hide nothing, got %v`, got)
	}
}
