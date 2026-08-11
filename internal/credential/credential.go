// Package credential stores model credentials outside the files a user syncs.
//
// The configuration file refuses anything shaped like a secret, which is right:
// it is meant to be versioned and shared. But refusing the wrong place without
// offering the right one does not protect anyone — it moves the secret to a
// shell profile or a pasted terminal line, which is somewhere nobody controls
// and nobody audits.
//
// Spec: docs/specs/architecture/configuration/202608081203-*.
package credential

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Store keeps one secret per name.
//
// The name is the model family, so switching family with `/model` reaches a
// different credential rather than reusing one that cannot work.
type Store interface {
	// Where describes the storage to a person, because "it is configured" is
	// not an answer anyone can act on.
	Where() string
	Get(name string) (string, error)
	Set(name, secret string) error
	Delete(name string) error
	List() ([]string, error)
}

// ErrNotFound is returned when nothing is stored under a name.
var ErrNotFound = fmt.Errorf("credential: not found")

// service is the keychain entry these live under.
const service = "dcode"

// nameRE bounds what a credential may be called.
//
// The name reaches a command line, and a name that could carry a flag or a
// shell metacharacter would be an injection point in the one package that
// handles secrets.
var nameRE = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$`)

func validName(name string) error {
	if !nameRE.MatchString(name) {
		return fmt.Errorf(
			"credential: %q is not a usable name; use letters, digits, dot, dash or underscore", name)
	}
	return nil
}

// ---------- presentation ----------

// Mask renders a secret so it can be recognised without being disclosed.
//
// Enough of both ends to tell two keys apart and to catch a paste from the
// wrong account, and never enough to use. A short secret is hidden entirely:
// showing six of twelve characters is not a mask.
func Mask(secret string) string {
	const head, tail = 6, 5
	r := []rune(secret)
	if len(r) == 0 {
		return ""
	}
	if len(r) < (head+tail)*2 {
		return strings.Repeat("•", 8)
	}
	return string(r[:head]) + "…" + string(r[len(r)-tail:])
}

// Fingerprint identifies a secret without revealing any of it.
//
// It is what lets someone compare against what the provider's own console
// shows, which is the question a masked value cannot always answer.
func Fingerprint(secret string) string {
	if secret == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])[:8]
}

// ---------- backend selection ----------

// Options configure which backend is chosen.
type Options struct {
	// StateDir is where the file backend writes when no keychain exists.
	StateDir string
	// Backend forces one: "keychain", "file", or "" to choose.
	Backend string
	// look overrides binary lookup, for tests.
	look func(string) (string, error)
	// run overrides command execution, for tests.
	run func(name string, args ...string) ([]byte, error)
}

// Backend names.
const (
	BackendKeychain = "keychain"
	BackendFile     = "file"
)

// Open selects a store.
//
// The keychain where one exists, a 0600 file where none does. The fallback is
// not a nicety: a headless server has no secret service, and refusing to store
// anything there would push the secret straight back into the environment this
// package exists to get it out of.
func Open(opts Options) (Store, error) {
	look, run := opts.look, opts.run
	if look == nil {
		look = exec.LookPath
	}
	if run == nil {
		run = func(name string, args ...string) ([]byte, error) {
			return exec.Command(name, args...).CombinedOutput()
		}
	}

	switch opts.Backend {
	case BackendFile:
		return newFileStore(opts.StateDir)
	case BackendKeychain:
		k := detectKeychain(look, run)
		if k == nil {
			return nil, fmt.Errorf(
				"credential: no keychain on this system; use the file backend or the environment")
		}
		return k, nil
	case "":
	default:
		return nil, fmt.Errorf("credential: unknown backend %q; valid: %s, %s",
			opts.Backend, BackendKeychain, BackendFile)
	}

	if k := detectKeychain(look, run); k != nil {
		return k, nil
	}
	return newFileStore(opts.StateDir)
}

func detectKeychain(look func(string) (string, error), run func(string, ...string) ([]byte, error)) Store {
	if bin, err := look("security"); err == nil {
		return &macKeychain{bin: bin, run: run}
	}
	if bin, err := look("secret-tool"); err == nil {
		return &secretTool{bin: bin, run: run}
	}
	return nil
}

// ---------- macOS keychain ----------

type macKeychain struct {
	bin string
	run func(string, ...string) ([]byte, error)
}

func (m *macKeychain) Where() string { return "macOS keychain" }

func (m *macKeychain) Get(name string) (string, error) {
	if err := validName(name); err != nil {
		return "", err
	}
	out, err := m.run(m.bin, "find-generic-password", "-s", service, "-a", name, "-w")
	if err != nil {
		return "", ErrNotFound
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// Set writes the secret.
//
// The value goes through an argument here, which is the one place this package
// cannot avoid it: `security` offers no stdin form for writing. It is a real
// exposure — the value is visible in `ps` for the life of the call — and it is
// accepted only because the alternative is not storing anything at all. The
// file backend has no such hole.
func (m *macKeychain) Set(name, secret string) error {
	if err := validName(name); err != nil {
		return err
	}
	out, err := m.run(m.bin, "add-generic-password",
		"-s", service, "-a", name, "-w", secret, "-U", "-T", "")
	if err != nil {
		return fmt.Errorf("credential: keychain refused the write: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func (m *macKeychain) Delete(name string) error {
	if err := validName(name); err != nil {
		return err
	}
	if _, err := m.run(m.bin, "delete-generic-password", "-s", service, "-a", name); err != nil {
		return ErrNotFound
	}
	return nil
}

// List reads the account names the keychain holds for this service.
func (m *macKeychain) List() ([]string, error) {
	out, err := m.run(m.bin, "dump-keychain")
	if err != nil {
		// Dumping asks for authorisation and is refused more often than not.
		// An empty list is the honest answer: this backend cannot enumerate,
		// and guessing would report credentials that may not exist.
		return nil, nil
	}
	return accountsFor(string(out), service), nil
}

// accountsFor scrapes account names out of a keychain dump. Pure, so the
// scraping is testable without a keychain.
func accountsFor(dump, svc string) []string {
	var (
		names   []string
		acct    string
		inEntry bool
	)
	for _, line := range strings.Split(dump, "\n") {
		switch {
		case strings.HasPrefix(line, "keychain:"):
			acct, inEntry = "", false
		case strings.Contains(line, `"acct"<blob>="`):
			acct = between(line, `"acct"<blob>="`, `"`)
		case strings.Contains(line, `"svce"<blob>="`+svc+`"`):
			inEntry = true
		}
		if inEntry && acct != "" {
			names = append(names, acct)
			acct, inEntry = "", false
		}
	}
	sort.Strings(names)
	return names
}

func between(s, open, close string) string {
	i := strings.Index(s, open)
	if i < 0 {
		return ""
	}
	rest := s[i+len(open):]
	j := strings.Index(rest, close)
	if j < 0 {
		return ""
	}
	return rest[:j]
}

// ---------- libsecret ----------

type secretTool struct {
	bin string
	run func(string, ...string) ([]byte, error)
	// runIn is the same for a command that reads from stdin. Separate because
	// the ONE method that bypassed the injection point was Set — the one that
	// handles the secret, and therefore the one most worth being able to test.
	runIn func(stdin, name string, args ...string) ([]byte, error)
}

func (s *secretTool) Where() string { return "libsecret keyring" }

func (s *secretTool) Get(name string) (string, error) {
	if err := validName(name); err != nil {
		return "", err
	}
	out, err := s.run(s.bin, "lookup", "service", service, "account", name)
	if err != nil || len(out) == 0 {
		return "", ErrNotFound
	}
	return strings.TrimRight(string(out), "\n"), nil
}

func (s *secretTool) Set(name, secret string) error {
	if err := validName(name); err != nil {
		return err
	}
	// secret-tool reads the value from stdin, so it never reaches a command
	// line — unlike the macOS backend, which has no such form.
	run := s.runIn
	if run == nil {
		run = runWithStdin
	}
	if out, err := run(secret, s.bin, "store", "--label", service+": "+name,
		"service", service, "account", name); err != nil {
		return fmt.Errorf("credential: keyring refused the write: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func (s *secretTool) Delete(name string) error {
	if err := validName(name); err != nil {
		return err
	}
	if _, err := s.run(s.bin, "clear", "service", service, "account", name); err != nil {
		return ErrNotFound
	}
	return nil
}

// List is not offered by secret-tool without a search that returns the secrets
// themselves, which is a worse trade than not listing.
func (s *secretTool) List() ([]string, error) { return nil, nil }

// ---------- file ----------

// fileStore keeps secrets in one 0600 file under the state directory.
//
// It protects against the common leak — a secret committed or synced with the
// configuration — and not against another process reading the file as this
// user. Where it is the only option, saying so is part of the contract.
type fileStore struct{ path string }

// FileName is the file the fallback backend writes.
const FileName = "credentials"

func newFileStore(stateDir string) (Store, error) {
	if stateDir == "" {
		return nil, fmt.Errorf("credential: a state directory is required for the file backend")
	}
	return &fileStore{path: filepath.Join(stateDir, FileName)}, nil
}

func (f *fileStore) Where() string { return f.path + " (0600)" }

func (f *fileStore) Get(name string) (string, error) {
	if err := validName(name); err != nil {
		return "", err
	}
	all, err := f.read()
	if err != nil {
		return "", err
	}
	if v, ok := all[name]; ok {
		return v, nil
	}
	return "", ErrNotFound
}

func (f *fileStore) Set(name, secret string) error {
	if err := validName(name); err != nil {
		return err
	}
	if strings.ContainsAny(secret, "\n\r") {
		return fmt.Errorf("credential: a secret cannot contain a line break")
	}
	all, err := f.read()
	if err != nil {
		return err
	}
	all[name] = secret
	return f.write(all)
}

func (f *fileStore) Delete(name string) error {
	all, err := f.read()
	if err != nil {
		return err
	}
	if _, ok := all[name]; !ok {
		return ErrNotFound
	}
	delete(all, name)
	return f.write(all)
}

func (f *fileStore) List() ([]string, error) {
	all, err := f.read()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(all))
	for n := range all {
		names = append(names, n)
	}
	sort.Strings(names)
	return names, nil
}

func (f *fileStore) read() (map[string]string, error) {
	out := map[string]string{}
	data, err := os.ReadFile(f.path)
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	// Refuse to read a file anyone else can. A secret readable by the group is
	// not a stored secret, and continuing would report it as safely kept.
	if fi, err := os.Stat(f.path); err == nil && fi.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf(
			"credential: %s is readable by others (%v); fix it with `chmod 600 %s`",
			f.path, fi.Mode().Perm(), f.path)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, secret, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		out[name] = secret
	}
	return out, nil
}

func (f *fileStore) write(all map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(f.path), 0o700); err != nil {
		return err
	}
	names := make([]string, 0, len(all))
	for n := range all {
		names = append(names, n)
	}
	sort.Strings(names)

	var b strings.Builder
	b.WriteString("# dcode credentials. Never commit this file.\n")
	for _, n := range names {
		fmt.Fprintf(&b, "%s\t%s\n", n, all[n])
	}

	// Written to a temporary file in the same directory and renamed, so an
	// interrupted write cannot leave a truncated file where a secret was. The
	// mode is set before the content lands.
	tmp, err := os.CreateTemp(filepath.Dir(f.path), ".credentials-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.WriteString(b.String()); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), f.path)
}

// runWithStdin executes a command that reads its input from stdin.
//
// The secret goes down the pipe and never onto a command line, where it would
// be visible to anything that can list processes.
func runWithStdin(stdin, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(stdin)
	return cmd.CombinedOutput()
}
