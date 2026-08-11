package main

import (
	"errors"
	"strings"
	"testing"

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
