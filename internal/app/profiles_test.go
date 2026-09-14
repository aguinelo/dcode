package app

import (
	"os"
	"path/filepath"
	"testing"
)

// A profile read from models.toml reaches Options the same way every other
// piece of configuration does: through FromEnv, not a second entry point.
func TestFromEnvReadsModelsToml(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "models.toml"), []byte(`
[profile.qwen-local]
model     = "qwen3.5-9b"
family    = "generic"
transport = "openai"
base_url  = "http://192.168.0.149:1234/v1"
window    = 32000
`), 0o600); err != nil {
		t.Fatal(err)
	}

	opts, _, err := FromEnv(envFrom(map[string]string{"DCODE_HOME": home}), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p, ok := opts.Profiles["qwen-local"]
	if !ok {
		t.Fatalf("got %+v, want a qwen-local profile", opts.Profiles)
	}
	if p.Model != "qwen3.5-9b" || p.Family != "generic" || p.BaseURL != "http://192.168.0.149:1234/v1" || p.Window != 32000 {
		t.Errorf("got %+v", p)
	}
}

// A project's models.toml layers over the user's by name, the same direction
// as every other layer this product resolves.
func TestFromEnvLayersProjectProfilesOverUserOnes(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "models.toml"), []byte(
		"[profile.cloud]\nmodel = \"MiniMax-M3\"\n\n[profile.qwen]\nmodel = \"qwen3.5-9b\"\nfamily = \"generic\"\n",
	), 0o600); err != nil {
		t.Fatal(err)
	}

	ws := t.TempDir()
	if err := os.MkdirAll(filepath.Join(ws, ".dcode"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, ".dcode", "models.toml"),
		[]byte("[profile.cloud]\nmodel = \"claude-sonnet\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	opts, _, err := FromEnv(envFrom(map[string]string{"DCODE_HOME": home}), ws)
	if err != nil {
		t.Fatal(err)
	}
	if got := opts.Profiles["cloud"].Model; got != "claude-sonnet" {
		t.Errorf("the project profile did not win: got %q", got)
	}
	if _, ok := opts.Profiles["qwen"]; !ok {
		t.Error("a user profile the project did not name should still be reachable")
	}
}

// A models.toml that cannot be parsed fails session creation, the same as any
// other configuration file this product refuses to read silently.
func TestFromEnvSurfacesABrokenModelsToml(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "models.toml"), []byte("not = valid = toml\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := FromEnv(envFrom(map[string]string{"DCODE_HOME": home}), t.TempDir()); err == nil {
		t.Fatal("a broken models.toml was silently accepted")
	}
}

// No file at all is the ordinary case: an empty map, not an error, and
// nothing about the rest of Options changes.
func TestFromEnvWithNoModelsToml(t *testing.T) {
	opts, _, err := FromEnv(envFrom(map[string]string{}), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(opts.Profiles) != 0 {
		t.Errorf("got %+v, want no profiles", opts.Profiles)
	}
}
