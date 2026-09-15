package config

import (
	"path/filepath"
	"testing"
)

// A profile is a name a person chose bundling everything /model needs to
// switch at once — not just the model string, which is all the ordinary path
// carries.

func TestAnAbsentModelsFileIsEmptyNotAnError(t *testing.T) {
	got, err := LoadModels(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("got %+v, want empty", got)
	}
}

func TestAProfileCarriesEverythingASwitchNeeds(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, ModelsFile), `
[profile.qwen-local]
model          = "qwen3.5-9b"
family         = "generic"
transport      = "openai"
base_url       = "http://192.168.0.149:1234/v1"
window         = 32000
max_iterations = 5000
`)
	got, err := LoadModels(root)
	if err != nil {
		t.Fatal(err)
	}
	p, ok := got["qwen-local"]
	if !ok {
		t.Fatalf("got %+v, want a profile named qwen-local", got)
	}
	want := Profile{
		Name: "qwen-local", Model: "qwen3.5-9b", Family: "generic",
		Transport: "openai", BaseURL: "http://192.168.0.149:1234/v1",
		Window: 32000, MaxIterations: 5000,
	}
	if p != want {
		t.Errorf("got %+v, want %+v", p, want)
	}
}

// The model itself is the one thing a profile cannot be worth having without.
func TestAProfileWithNoModelIsAnError(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, ModelsFile), "[profile.broken]\nfamily = \"generic\"\n")
	if _, err := LoadModels(root); err == nil {
		t.Fatal("a profile naming no model was accepted")
	}
}

// The name is a person's choice, not a bijective key: a section that does not
// look like a profile is the mistake worth naming, not a schema violation.
func TestASectionNotNamedProfileIsAnError(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, ModelsFile), "[qwen-local]\nmodel = \"qwen3.5-9b\"\n")
	if _, err := LoadModels(root); err == nil {
		t.Fatal("a bare section name was accepted; it must be [profile.<name>]")
	}
}

func TestAWindowThatIsNotANumberIsAnError(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, ModelsFile), "[profile.p]\nmodel = \"m\"\nwindow = \"soon\"\n")
	if _, err := LoadModels(root); err == nil {
		t.Fatal("a non-numeric window was accepted")
	}
}

// Credentials refused here for the same reason they are refused in
// config.toml and grants.toml: this file is meant to be versioned and synced.
func TestModelsFileRefusesAShapedSecret(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, ModelsFile), "[profile.p]\nmodel = \"m\"\napi_key = \"sk-test\"\n")
	if _, err := LoadModels(root); err == nil {
		t.Fatal("a key parked in models.toml was accepted")
	}
}

// A project profile under the same name as a user one is the project's own
// answer, not a collision — the same direction every other layer resolves in.
func TestAProjectProfileOverridesAUserOneByName(t *testing.T) {
	user := map[string]Profile{
		"cloud": {Name: "cloud", Model: "MiniMax-M3"},
		"qwen":  {Name: "qwen", Model: "qwen3.5-9b", Family: "generic"},
	}
	project := map[string]Profile{
		"cloud": {Name: "cloud", Model: "claude-sonnet"},
	}
	merged := MergeModels(user, project)
	if merged["cloud"].Model != "claude-sonnet" {
		t.Errorf("project did not win: got %+v", merged["cloud"])
	}
	if _, ok := merged["qwen"]; !ok {
		t.Error("a user profile the project did not name should still be reachable")
	}
}
