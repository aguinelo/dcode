package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/app"
	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/credential"
)

// fakeStore stands in for a keychain.
type fakeStore struct {
	values  map[string]string
	setErr  error
	listErr error
}

func newFakeStore() *fakeStore { return &fakeStore{values: map[string]string{}} }

func (f *fakeStore) Where() string { return "test store" }

func (f *fakeStore) Get(name string) (string, error) {
	v, ok := f.values[name]
	if !ok {
		return "", credential.ErrNotFound
	}
	return v, nil
}

func (f *fakeStore) Set(name, secret string) error {
	if f.setErr != nil {
		return f.setErr
	}
	f.values[name] = secret
	return nil
}

func (f *fakeStore) Delete(name string) error {
	if _, ok := f.values[name]; !ok {
		return credential.ErrNotFound
	}
	delete(f.values, name)
	return nil
}

func (f *fakeStore) List() ([]string, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]string, 0, len(f.values))
	for n := range f.values {
		out = append(out, n)
	}
	return out, nil
}

// Listing shows what is held and, for each, enough to tell two keys apart —
// never enough to use one.
func TestListingShowsFingerprintsAndNotSecrets(t *testing.T) {
	s := newFakeStore()
	const secret = "sk-live-0123456789abcdefghij"
	if err := s.Set("openai", secret); err != nil {
		t.Fatal(err)
	}

	out, _ := capture(t, func() {
		if err := listCredentials(s, "openai"); err != nil {
			t.Error(err)
		}
	})
	if strings.Contains(out, secret) {
		t.Fatalf("the credential was printed:\n%s", out)
	}
	if !strings.Contains(out, "openai") {
		t.Errorf("the stored name is missing:\n%s", out)
	}
}

func TestAnEmptyStoreSaysSo(t *testing.T) {
	out, _ := capture(t, func() {
		if err := listCredentials(newFakeStore(), ""); err != nil {
			t.Error(err)
		}
	})
	if strings.TrimSpace(out) == "" {
		t.Fatal("an empty store printed nothing, which reads as a broken command")
	}
}

// Revealing is deliberate and explicit, and it is the only path that prints a
// secret. Asking for one that is not there must not print an empty line as
// though it had succeeded.
func TestRevealingSomethingAbsentIsAnError(t *testing.T) {
	_, _ = capture(t, func() {
		if err := revealCredential(newFakeStore(), "openai"); err == nil {
			t.Error("revealing an absent credential reported success")
		}
	})
}

func TestRevealingPrintsExactlyTheSecret(t *testing.T) {
	s := newFakeStore()
	const secret = "sk-live-value"
	if err := s.Set("openai", secret); err != nil {
		t.Fatal(err)
	}
	out, _ := capture(t, func() {
		if err := revealCredential(s, "openai"); err != nil {
			t.Error(err)
		}
	})
	if strings.TrimSpace(out) != secret {
		t.Fatalf("reveal printed %q, want exactly the secret", strings.TrimSpace(out))
	}
}

// A store that refuses the write must say why. "It did not work" sends someone
// to read a keychain by hand.
func TestAStoreThatRefusesReportsWhy(t *testing.T) {
	s := newFakeStore()
	s.listErr = errors.New("the keyring daemon is not running")
	_, _ = capture(t, func() {
		if err := listCredentials(s, ""); err == nil {
			t.Error("a store that could not be read reported success")
		} else if !strings.Contains(err.Error(), "keyring daemon") {
			t.Errorf("the reason was lost: %v", err)
		}
	})
}

// --family wins over a profile named alongside it — the explicit override
// this flag already had, unchanged by profiles existing.
func TestCredentialNameFamilyWinsOverProfile(t *testing.T) {
	opts := app.Options{
		Model: "MiniMax-M3",
		Profiles: map[string]config.Profile{
			"qwen-local": {Name: "qwen-local", Model: "qwen3.5-9b", Family: "generic"},
		},
	}
	got, err := credentialName("claude", "qwen-local", opts)
	if err != nil {
		t.Fatal(err)
	}
	if got != "claude" {
		t.Errorf("got %q, want the explicit family", got)
	}
}

// A profile resolves through the same path any other model does: its own
// Family when it set one.
func TestCredentialNameResolvesAProfilesFamily(t *testing.T) {
	opts := app.Options{
		Model: "MiniMax-M3",
		Profiles: map[string]config.Profile{
			"qwen-local": {Name: "qwen-local", Model: "qwen3.5-9b", Family: "generic"},
		},
	}
	got, err := credentialName("", "qwen-local", opts)
	if err != nil {
		t.Fatal(err)
	}
	if got != "generic" {
		t.Errorf("got %q, want the profile's family", got)
	}
}

// A profile that only renames a known cloud model needs no family of its
// own — the model's prefix resolves it, same as the ordinary path.
func TestCredentialNameResolvesAProfileWithNoFamilyByModelPrefix(t *testing.T) {
	opts := app.Options{
		Model: "qwen3.5-9b",
		Profiles: map[string]config.Profile{
			"cloud": {Name: "cloud", Model: "MiniMax-M3"},
		},
	}
	got, err := credentialName("", "cloud", opts)
	if err != nil {
		t.Fatal(err)
	}
	if got != "minimax-m3" {
		t.Errorf("got %q, want the family MiniMax-M3's prefix resolves to", got)
	}
}

// An unknown profile name is an error naming what is actually configured,
// not a silent fall-through to the current model.
func TestCredentialNameUnknownProfileNamesWhatExists(t *testing.T) {
	opts := app.Options{
		Model: "MiniMax-M3",
		Profiles: map[string]config.Profile{
			"qwen-local": {Name: "qwen-local", Model: "qwen3.5-9b", Family: "generic"},
		},
	}
	_, err := credentialName("", "typo-d", opts)
	if err == nil {
		t.Fatal("an unknown profile name was accepted")
	}
	if !strings.Contains(err.Error(), "qwen-local") {
		t.Errorf("the error does not name what is configured: %v", err)
	}
}

// With neither flag, the current model decides — exactly the behaviour from
// before --profile existed.
func TestCredentialNameDefaultsToTheCurrentModel(t *testing.T) {
	opts := app.Options{Model: "claude-sonnet"}
	got, err := credentialName("", "", opts)
	if err != nil {
		t.Fatal(err)
	}
	if got != "claude" {
		t.Errorf("got %q", got)
	}
}
