package app

import (
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/credential"
)

// The family rather than the model: `MiniMax-M3` and a later `MiniMax-M4` reach
// the same account at the same provider, and asking someone to store the same
// key twice is asking them to forget one of them.
func TestCredentialNameIsTheFamily(t *testing.T) {
	for name, tc := range map[string]struct {
		opts Options
		want string
	}{
		"by model":                   {Options{Model: "MiniMax-M3"}, "minimax-m3"},
		"another model, same family": {Options{Model: "minimax-m3-preview"}, "minimax-m3"},
		"other family":               {Options{Model: "claude-sonnet-4"}, "claude"},
		"explicit override":          {Options{Model: "MiniMax-M3", Family: "claude"}, "claude"},
		"nothing claims it":          {Options{Model: "llama-9"}, ""},
	} {
		if got := CredentialName(tc.opts); got != tc.want {
			t.Errorf("%s: got %q want %q", name, got, tc.want)
		}
	}
}

// The environment is explicit and scoped to one invocation, so it wins; the
// store is what makes the ordinary case not require it.
func TestTheEnvironmentBeatsTheStore(t *testing.T) {
	dir := t.TempDir()
	store, err := credential.Open(credential.Options{Backend: credential.BackendFile, StateDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Set("minimax-m3", "sk-stored"); err != nil {
		t.Fatal(err)
	}

	opts := Options{Model: "MiniMax-M3", CredentialBackend: credential.BackendFile}
	secret, from := LookupCredential(config.Roots{State: dir}, opts)
	if secret != "sk-stored" {
		t.Fatalf("the store must answer when the environment is silent: %q", secret)
	}
	if from == "" {
		t.Error("where it came from is what makes `config` answerable")
	}

	// And with the environment set, FromEnv never consults the store at all.
	got, _, err := FromEnv(envFrom(map[string]string{
		"DCODE_API_KEY": "sk-from-env",
		"DCODE_HOME":    dir,
	}), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got.APIKey != "sk-from-env" {
		t.Errorf("got %q", got.APIKey)
	}
	if got.CredentialFrom != "DCODE_API_KEY" {
		t.Errorf("the origin must name the environment, got %q", got.CredentialFrom)
	}
}

// A store that cannot be reached is not a reason to refuse to start: the turn
// that needs the key reports a clear auth error instead.
func TestAnUnreachableStoreIsSilent(t *testing.T) {
	opts := Options{Model: "MiniMax-M3", CredentialBackend: credential.BackendFile}
	secret, from := LookupCredential(config.Roots{}, opts)
	if secret != "" || from != "" {
		t.Errorf("got %q from %q", secret, from)
	}
	// And a model no family claims has nowhere to look.
	if s, _ := LookupCredential(config.Roots{State: t.TempDir()}, Options{Model: "llama-9"}); s != "" {
		t.Errorf("got %q", s)
	}
}

// The one invariant that matters most: whatever else changes, the key must not
// reach the model, the prompt, or anything a client renders.
func TestTheKeyNeverReachesThePrompt(t *testing.T) {
	home := t.TempDir()
	env := envFrom(map[string]string{
		"DCODE_HOME":               home,
		"DCODE_CREDENTIAL_BACKEND": credential.BackendFile,
	})
	// Through the same root discovery the app uses, so the test cannot pass by
	// writing somewhere the product would never look.
	roots, err := config.DiscoverRoots(env)
	if err != nil {
		t.Fatal(err)
	}
	store, err := credential.Open(credential.Options{
		Backend: credential.BackendFile, StateDir: roots.State,
	})
	if err != nil {
		t.Fatal(err)
	}
	const secret = "sk-test-NUNCA-DEVE-APARECER-0123456789"
	if err := store.Set("minimax-m3", secret); err != nil {
		t.Fatal(err)
	}

	ws := t.TempDir()
	opts, _, err := FromEnv(env, ws)
	if err != nil {
		t.Fatal(err)
	}
	if opts.APIKey != secret {
		t.Fatalf("setup: the key did not load, got %q", opts.APIKey)
	}

	sess, err := New(opts, nil, nil)
	if err != nil {
		t.Skipf("no sandbox available: %v", err)
	}
	if strings.Contains(sess.Prompt, secret) {
		t.Error("the key reached the system prompt")
	}
	for _, m := range sess.Engine.Session().History {
		if strings.Contains(m.Text, secret) {
			t.Error("the key reached the history")
		}
	}
	// And the masked form is safe to show anywhere the value is not.
	if strings.Contains(credential.Mask(secret), "NUNCA-DEVE") {
		t.Error("the mask leaked the middle")
	}
}
