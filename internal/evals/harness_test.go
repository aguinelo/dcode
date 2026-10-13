package evals

import (
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/app"
	"github.com/aguinelo/dcode/internal/provider"
)

// The eval measures the model it names, with that model's credential.
//
// setup took the product's options as FromEnv resolved them, for model.name —
// MiniMax-M3 unless told otherwise — and only swapped the model at the call.
// The stored credential was therefore looked up for model.name's family: an
// eval of gemini-2.5-flash on a machine logged in to both read the MiniMax key
// and sent it to Google.
func TestTheEvalLooksUpTheCredentialOfTheModelItMeasures(t *testing.T) {
	var asked string
	lookup := func(o app.Options) (string, string) {
		asked = app.CredentialName(o)
		return "key-for-" + asked, "store"
	}
	opts := app.Options{Model: "MiniMax-M3"}

	got := measuredOptions(opts, "gemini-2.5-flash", lookup)

	if asked != provider.GeminiName {
		t.Errorf("the credential was looked up for %q, want the measured model's family %q", asked, provider.GeminiName)
	}
	if got.Model != "gemini-2.5-flash" || got.APIKey != "key-for-gemini" {
		t.Errorf("options = %q / %q, want the measured model and its key", got.Model, got.APIKey)
	}
}

// DCODE_API_KEY is explicit and wins, as it does in the product.
func TestTheEnvironmentKeyStillWins(t *testing.T) {
	opts := app.Options{Model: "MiniMax-M3", APIKey: "from-env", CredentialFrom: "DCODE_API_KEY"}
	got := measuredOptions(opts, "gemini-2.5-flash", func(app.Options) (string, string) {
		t.Error("the store was asked although DCODE_API_KEY was set")
		return "", ""
	})
	if got.APIKey != "from-env" {
		t.Errorf("key = %q, want the one from the environment", got.APIKey)
	}
}

// DCODE_FAMILY names the family, as it does in the product.
//
// setup resolved by prefix only, so a model no family claims — anything run
// through generic, which is every local model — could not be measured at all,
// and the family the product warns about most was the one no eval could reach.
func TestTheEvalHonoursANamedFamily(t *testing.T) {
	reg := provider.NewRegistry()
	reg.RegisterTransport(app.NewHTTPTransport(provider.TransportOpenAI, "http://localhost:1", "k"))
	reg.RegisterTransport(app.NewHTTPTransport(provider.TransportAnthropic, "http://localhost:1", "k"))
	for _, f := range app.Families() {
		if err := reg.RegisterFamily(f); err != nil {
			t.Fatal(err)
		}
	}

	p, err := measuredProvider(reg, app.Options{Model: "qwen3-coder", Family: provider.GenericName})
	if err != nil {
		t.Fatalf("a model run through the generic family did not resolve: %v", err)
	}
	if p.Family().Name() != provider.GenericName {
		t.Errorf("resolved to %s, want %s", p.Family().Name(), provider.GenericName)
	}

	// And without one, prefix resolution, which refuses a model nobody claims
	// rather than guessing.
	if _, err := measuredProvider(reg, app.Options{Model: "qwen3-coder"}); err == nil {
		t.Error("an unclaimed model resolved with no family named")
	}
}

// One contract, several families, each counted once under its own.
func TestOneContractIsMeasuredOncePerFamily(t *testing.T) {
	ms := []Measurement{
		{ID: "a", Model: "MiniMax-M3", Sound: true},
		{ID: "a", Model: "gemini-2.5-flash", Sound: true},
		{ID: "b", Model: "MiniMax-M3", Sound: true},
		{ID: "c", Model: "qwen3-coder", Family: provider.GenericName, Sound: true},
		{ID: "d", Model: "MiniMax-M3", Sound: false, Note: "lost a run"},
	}
	got, err := measuredByFamily(ms, app.Families())
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{"minimax-m3": 2, provider.GeminiName: 1, provider.GenericName: 1, provider.ClaudeName: 0}
	for name, n := range want {
		if got[name] != n {
			t.Errorf("%s: %d contracts measured, want %d", name, got[name], n)
		}
	}
	if n := contractsMeasured(ms); n != 3 {
		t.Errorf("%d contracts measured, want 3: one measured twice is still one contract", n)
	}
}

// A measurement against a model no family claims has to say which family it
// ran under, or it cannot be counted, compared or rebuilt.
func TestAnUnclaimedModelMustNameItsFamily(t *testing.T) {
	_, err := measuredByFamily([]Measurement{{ID: "a", Model: "qwen3-coder", Sound: true}}, app.Families())
	if err == nil {
		t.Error("a measurement no family claims was counted under none")
	}
}

// measuredOptions points the product's options at the model being measured.
//
// FromEnv resolves options for model.name, and the eval measures eval.model.
// Everything that depends on the model — its family, and so the credential and
// the formulation — has to be decided for the second, or the eval sends one
// model's key to another model's provider. The environment key is explicit and
// scoped to one invocation, so it wins here as it does in the product.
//
// lookup is app.LookupCredential with the roots bound, passed in so that what
// is asked of the store can be seen without a store.
func measuredOptions(opts app.Options, model string, lookup func(app.Options) (string, string)) app.Options {
	opts.Model = model
	if opts.CredentialFrom != "DCODE_API_KEY" {
		opts.APIKey, opts.CredentialFrom = lookup(opts)
	}
	return opts
}

// measuredProvider resolves the provider the way the product does: the family
// model.family names when it names one, and the model's prefix otherwise.
//
// The product's order, written once more because app keeps its builder
// private. The two branches are each one call into the same registry, and a
// test holds this one to both.
func measuredProvider(reg *provider.Registry, opts app.Options) (provider.Provider, error) {
	if opts.Family != "" {
		return reg.ResolveFamily(opts.Family, opts.Transport)
	}
	return reg.Resolve(opts.Model, opts.Transport)
}

// The line a measurement is recorded from names the family, because with a
// named family the model's name no longer says which one the run went through.
func TestTheResultLineNamesTheFamily(t *testing.T) {
	line := Result{ID: "a", Runs: 1, Passed: 1, Model: "qwen3-coder", Family: provider.GenericName}.String()
	if !strings.Contains(line, "family "+provider.GenericName) {
		t.Errorf("the result line does not name the family: %s", line)
	}
}
