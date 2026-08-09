package credential

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------- presentation ----------

// Enough of both ends to tell two keys apart and to catch a paste from the
// wrong account, and never enough to use.
func TestMaskShowsEnoughToRecogniseAndNotEnoughToUse(t *testing.T) {
	secret := "sk-cp-gCriuWYkSvSTBesa2c9mEbXyM3y-RQR7uBZxoo4nKjwg"
	got := Mask(secret)

	if !strings.HasPrefix(got, "sk-cp-") {
		t.Errorf("the start must be recognisable: %q", got)
	}
	if !strings.HasSuffix(got, "Kjwg") {
		t.Errorf("the end catches a paste from the wrong account: %q", got)
	}
	if strings.Contains(got, "iuWYkSvSTBesa") {
		t.Errorf("the middle must never appear: %q", got)
	}
	if len(got) >= len(secret) {
		t.Errorf("a mask that is not shorter is not a mask: %q", got)
	}
}

// Showing six characters of twelve is not a mask.
func TestMaskHidesAShortSecretEntirely(t *testing.T) {
	for _, s := range []string{"short", "0123456789", "0123456789ab"} {
		got := Mask(s)
		for _, r := range s {
			if strings.ContainsRune(got, r) {
				t.Errorf("%q leaked %q through the mask %q", s, string(r), got)
			}
		}
	}
	if got := Mask(""); got != "" {
		t.Errorf("nothing stored, nothing to show: %q", got)
	}
}

// The fingerprint answers the question a mask cannot: is this the same key the
// provider's console is showing.
func TestFingerprintIdentifiesWithoutRevealing(t *testing.T) {
	a := Fingerprint("sk-one")
	b := Fingerprint("sk-two")
	if a == b {
		t.Error("different secrets must fingerprint differently")
	}
	if a != Fingerprint("sk-one") {
		t.Error("the same secret must fingerprint the same")
	}
	if len(a) != 8 {
		t.Errorf("got %q", a)
	}
	if strings.Contains(a, "sk-") {
		t.Errorf("the fingerprint must not carry the secret: %q", a)
	}
	if got := Fingerprint(""); got != "" {
		t.Errorf("got %q", got)
	}
}

// ---------- the file backend ----------

func fileStoreIn(t *testing.T) (Store, string) {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(Options{Backend: BackendFile, StateDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	return s, filepath.Join(dir, FileName)
}

func TestFileStoreRoundTrip(t *testing.T) {
	s, path := fileStoreIn(t)

	if _, err := s.Get("minimax-m3"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
	if err := s.Set("minimax-m3", "sk-one"); err != nil {
		t.Fatal(err)
	}
	if err := s.Set("claude", "sk-two"); err != nil {
		t.Fatal(err)
	}

	got, err := s.Get("minimax-m3")
	if err != nil || got != "sk-one" {
		t.Fatalf("got %q, %v", got, err)
	}
	// One credential per family is the whole point: switching family must reach
	// a different secret rather than reusing one that cannot work.
	if got, _ := s.Get("claude"); got != "sk-two" {
		t.Errorf("got %q", got)
	}

	names, err := s.List()
	if err != nil || len(names) != 2 || names[0] != "claude" {
		t.Fatalf("got %v, %v", names, err)
	}

	if err := s.Delete("claude"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get("claude"); !errors.Is(err, ErrNotFound) {
		t.Errorf("got %v", err)
	}
	if err := s.Delete("claude"); !errors.Is(err, ErrNotFound) {
		t.Errorf("deleting what is not there says so: %v", err)
	}

	// Replacing keeps one entry rather than appending a second.
	if err := s.Set("minimax-m3", "sk-three"); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Get("minimax-m3"); got != "sk-three" {
		t.Errorf("got %q", got)
	}
	if names, _ := s.List(); len(names) != 1 {
		t.Errorf("got %v", names)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

// A secret readable by the group is not a stored secret, and continuing would
// report it as safely kept.
func TestFileStoreRefusesAWorldReadableFile(t *testing.T) {
	s, path := fileStoreIn(t)
	if err := s.Set("a", "sk-one"); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := s.Get("a")
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("a loose file must be refused loudly, got %v", err)
	}
	// And the refusal has to say how to fix it, or it is only a complaint.
	if !strings.Contains(err.Error(), "chmod 600") {
		t.Errorf("got %v", err)
	}
}

func TestFileStoreWritesOwnerOnly(t *testing.T) {
	s, path := fileStoreIn(t)
	if err := s.Set("a", "sk-one"); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm&0o077 != 0 {
		t.Errorf("the file is reachable by others: %v", perm)
	}
}

// A newline would end the record and turn the rest of the secret into a second
// entry, so it is refused at the door.
func TestFileStoreRefusesALineBreakInASecret(t *testing.T) {
	s, _ := fileStoreIn(t)
	if err := s.Set("a", "sk-one\nb\tsk-two"); err == nil {
		t.Error("a line break must be refused")
	}
}

// The name reaches a command line in the keychain backends, so a name that
// could carry a flag is refused everywhere for one rule, not two.
func TestNamesAreBounded(t *testing.T) {
	s, _ := fileStoreIn(t)
	for _, bad := range []string{"", "-w", "a b", "a/b", "a;rm -rf /", "a$(x)", strings.Repeat("a", 65)} {
		if err := s.Set(bad, "x"); err == nil {
			t.Errorf("%q must be refused", bad)
		}
	}
	for _, good := range []string{"minimax-m3", "claude", "a.b_c-1"} {
		if err := s.Set(good, "x"); err != nil {
			t.Errorf("%q must be accepted: %v", good, err)
		}
	}
}

func TestFileStoreNeedsAStateDirectory(t *testing.T) {
	if _, err := Open(Options{Backend: BackendFile}); err == nil {
		t.Error("the file backend cannot guess where to write")
	}
}

// ---------- backend selection ----------

func lookOnly(available ...string) func(string) (string, error) {
	return func(name string) (string, error) {
		for _, a := range available {
			if a == name {
				return "/usr/bin/" + name, nil
			}
		}
		return "", fmt.Errorf("not found")
	}
}

// The keychain where one exists, a 0600 file where none does. The fallback is
// not a nicety: a headless server has no secret service, and refusing there
// would push the secret back into the environment this package exists to get
// it out of.
func TestOpenPrefersTheKeychainAndFallsBackToAFile(t *testing.T) {
	dir := t.TempDir()

	mac, err := Open(Options{StateDir: dir, look: lookOnly("security")})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mac.Where(), "keychain") {
		t.Errorf("got %q", mac.Where())
	}

	linux, err := Open(Options{StateDir: dir, look: lookOnly("secret-tool")})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(linux.Where(), "keyring") {
		t.Errorf("got %q", linux.Where())
	}

	headless, err := Open(Options{StateDir: dir, look: lookOnly()})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(headless.Where(), FileName) {
		t.Errorf("got %q", headless.Where())
	}
	// Where the file is the only option, saying so is part of the contract.
	if !strings.Contains(headless.Where(), "0600") {
		t.Errorf("the storage must describe itself: %q", headless.Where())
	}
}

func TestOpenHonoursAnExplicitBackend(t *testing.T) {
	dir := t.TempDir()

	// Asking for the keychain where none exists is an error, not a silent
	// downgrade to something weaker than what was asked for.
	if _, err := Open(Options{Backend: BackendKeychain, StateDir: dir, look: lookOnly()}); err == nil {
		t.Error("a keychain that does not exist must be reported")
	}
	// And asking for the file gets the file, even where a keychain exists.
	s, err := Open(Options{Backend: BackendFile, StateDir: dir, look: lookOnly("security")})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s.Where(), FileName) {
		t.Errorf("got %q", s.Where())
	}
	if _, err := Open(Options{Backend: "nonsense", StateDir: dir}); err == nil {
		t.Error("an unknown backend must be rejected")
	}
}

// ---------- the keychain backends, driven without a keychain ----------

type fakeRunner struct {
	out  map[string][]byte
	err  map[string]error
	seen [][]string
}

func (f *fakeRunner) run(name string, args ...string) ([]byte, error) {
	f.seen = append(f.seen, append([]string{name}, args...))
	key := strings.Join(args, " ")
	if e, ok := f.err[key]; ok {
		return nil, e
	}
	return f.out[key], nil
}

func TestMacKeychainSpeaksSecurity(t *testing.T) {
	r := &fakeRunner{
		out: map[string][]byte{
			"find-generic-password -s dcode -a minimax-m3 -w": []byte("sk-one\n"),
		},
		err: map[string]error{
			"find-generic-password -s dcode -a missing -w": fmt.Errorf("exit 44"),
		},
	}
	s, err := Open(Options{Backend: BackendKeychain, look: lookOnly("security"), run: r.run})
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.Get("minimax-m3")
	if err != nil || got != "sk-one" {
		t.Fatalf("got %q, %v", got, err)
	}
	// The trailing newline `security` prints is framing, not secret.
	if strings.HasSuffix(got, "\n") {
		t.Error("the newline must be stripped")
	}
	if _, err := s.Get("missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("got %v", err)
	}
	if err := s.Set("claude", "sk-two"); err != nil {
		t.Fatal(err)
	}
	// -U so a second login replaces rather than erroring on a duplicate.
	var sawUpdate bool
	for _, call := range r.seen {
		if call[1] == "add-generic-password" {
			for _, a := range call {
				if a == "-U" {
					sawUpdate = true
				}
			}
		}
	}
	if !sawUpdate {
		t.Error("writing must update an existing entry rather than fail on it")
	}
}

// The dump is scraped rather than trusted, and the scraping is pure so it can
// be tested without a keychain and without authorising a dump.
func TestAccountsForReadsTheDump(t *testing.T) {
	dump := `keychain: "/Users/x/Library/Keychains/login.keychain-db"
    "acct"<blob>="minimax-m3"
    "svce"<blob>="dcode"
keychain: "/Users/x/Library/Keychains/login.keychain-db"
    "acct"<blob>="someone-else"
    "svce"<blob>="other-app"
keychain: "/Users/x/Library/Keychains/login.keychain-db"
    "acct"<blob>="claude"
    "svce"<blob>="dcode"
`
	got := accountsFor(dump, "dcode")
	if len(got) != 2 || got[0] != "claude" || got[1] != "minimax-m3" {
		t.Fatalf("got %v", got)
	}
	// Another application's secrets are none of our business.
	for _, n := range got {
		if n == "someone-else" {
			t.Error("only this service's accounts may be listed")
		}
	}
}

// Dumping asks for authorisation and is refused more often than not. An empty
// list is the honest answer; guessing would report credentials that may not
// exist.
func TestMacKeychainListIsEmptyWhenTheDumpIsRefused(t *testing.T) {
	r := &fakeRunner{err: map[string]error{"dump-keychain": fmt.Errorf("denied")}}
	s, _ := Open(Options{Backend: BackendKeychain, look: lookOnly("security"), run: r.run})
	got, err := s.List()
	if err != nil {
		t.Fatalf("a refused dump is not an error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

func TestSecretToolSpeaksLibsecret(t *testing.T) {
	r := &fakeRunner{
		out: map[string][]byte{
			"lookup service dcode account minimax-m3": []byte("sk-one\n"),
		},
		err: map[string]error{
			"lookup service dcode account missing": fmt.Errorf("exit 1"),
			"clear service dcode account missing":  fmt.Errorf("exit 1"),
		},
	}
	s, err := Open(Options{Backend: BackendKeychain, look: lookOnly("secret-tool"), run: r.run})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := s.Get("minimax-m3"); err != nil || got != "sk-one" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := s.Get("missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("got %v", err)
	}
	if err := s.Delete("missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("got %v", err)
	}
	if _, err := s.List(); err != nil {
		t.Errorf("got %v", err)
	}
}

// A name that could carry a flag would be an injection point in the one package
// that handles secrets, so it is refused before it reaches a command line.
func TestKeychainBackendsRefuseAnUnusableName(t *testing.T) {
	r := &fakeRunner{}
	for _, look := range []func(string) (string, error){lookOnly("security"), lookOnly("secret-tool")} {
		s, err := Open(Options{Backend: BackendKeychain, look: look, run: r.run})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.Get("-w"); err == nil {
			t.Error("a flag-shaped name must be refused")
		}
		if err := s.Set("a b", "x"); err == nil {
			t.Error("a name with a space must be refused")
		}
		if err := s.Delete("a;b"); err == nil {
			t.Error("a name with a separator must be refused")
		}
	}
	for _, call := range r.seen {
		for _, a := range call {
			if a == "-w" || strings.Contains(a, ";") {
				t.Errorf("an unusable name reached the command line: %v", call)
			}
		}
	}
}
