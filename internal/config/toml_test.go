package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTOMLReadsTheDocumentedFile(t *testing.T) {
	got, err := ParseTOML([]byte(`
# a comment
[model]
name      = "MiniMax-M3"   # trailing comment
transport = ""

[sandbox]
mode          = "read-only"
allow_network = false

[limits]
max_iterations = 40
`), "config.toml")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"model.name":            "MiniMax-M3",
		"model.transport":       "",
		"sandbox.mode":          "read-only",
		"sandbox.allow_network": "false",
		"limits.max_iterations": "40",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: got %q want %q", k, got[k], v)
		}
	}
	if len(got) != len(want) {
		t.Errorf("got %d keys, want %d: %v", len(got), len(want), got)
	}
}

// A typo silently ignored is the most frustrating configuration bug there is:
// everything reports success and nothing changed.
func TestUnknownKeyIsAnErrorAndNamesTheAlternatives(t *testing.T) {
	_, err := ParseTOML([]byte("[model]\nnaem = \"x\"\n"), "config.toml")
	if err == nil {
		t.Fatal("an unknown key must be rejected")
	}
	if !strings.Contains(err.Error(), "model.naem") {
		t.Errorf("the error must name the offending key: %v", err)
	}
	if !strings.Contains(err.Error(), "model.name") {
		t.Errorf("the error must list the known keys so the typo is fixable: %v", err)
	}
	if !strings.Contains(err.Error(), "config.toml:2") {
		t.Errorf("the error must name the line: %v", err)
	}
}

// Config files get versioned and synced, which is what makes this the most
// common credential leak there is.
func TestCredentialShapedKeysAreRefusedInAnySection(t *testing.T) {
	for _, body := range []string{
		"[model]\napi_key = \"sk-live\"\n",
		"[model]\napiKey = \"sk-live\"\n",
		"[whatever]\ntoken = \"abc\"\n",
		"[auth]\nPASSWORD = \"hunter2\"\n",
		"[x]\nclient_secret = \"s\"\n",
		"[x]\ncredential = \"s\"\n",
	} {
		err := func() error {
			_, err := ParseTOML([]byte(body), "config.toml")
			return err
		}()
		if err == nil {
			t.Fatalf("a credential must be refused: %q", body)
		}
		// Refusing without saying where it should go leaves the user stuck.
		if !strings.Contains(err.Error(), "DCODE_API_KEY") {
			t.Errorf("the error must say where credentials come from: %v", err)
		}
	}
}

// The unknown-section case is exactly where a secret would otherwise slip
// through, so the credential check has to run before the schema check.
func TestCredentialCheckBeatsTheUnknownKeyCheck(t *testing.T) {
	_, err := ParseTOML([]byte("[nowhere]\napi_key = \"x\"\n"), "c.toml")
	if err == nil || !strings.Contains(err.Error(), "credential") {
		t.Fatalf("want a credential error, got %v", err)
	}
}

func TestParseTOMLRejectsMalformedInput(t *testing.T) {
	for name, body := range map[string]string{
		"unterminated section": "[model\n",
		"empty section":        "[]\n",
		"no equals":            "[model]\nname\n",
		"empty key":            "[model]\n = \"x\"\n",
		"key outside section":  "name = \"x\"\n",
		"unterminated string":  "[model]\nname = \"x\n",
		"array":                "[limits]\nparallel = [1, 2]\n",
		"inline table":         "[limits]\nparallel = {a = 1}\n",
		"missing value":        "[model]\nname =\n",
		"bare word":            "[model]\nname = yes\n",
		"array of tables":      "[[model]]\n",
	} {
		if _, err := ParseTOML([]byte(body), "c.toml"); err == nil {
			t.Errorf("%s: must be rejected", name)
		}
	}
}

// A `#` inside a quoted value is data, not a comment.
func TestCommentStrippingHonoursQuotes(t *testing.T) {
	got, err := ParseTOML([]byte(`[model]
base_url = "https://h/#anchor"
`), "c.toml")
	if err != nil {
		t.Fatal(err)
	}
	if got["model.base_url"] != "https://h/#anchor" {
		t.Errorf("got %q", got["model.base_url"])
	}
}

// One key, one variable, in both directions. Without the bijection there is no
// single origin to report for a key, and `--config` becomes a guess.
func TestKeyToEnvMappingIsBijective(t *testing.T) {
	if len(EnvToKey) != len(KnownKeys) {
		t.Fatalf("two keys share an environment variable: %d keys, %d variables",
			len(KnownKeys), len(EnvToKey))
	}
	for key, env := range KnownKeys {
		if !strings.HasPrefix(env, "DCODE_") {
			t.Errorf("%s maps to %q, which is not a dcode variable", key, env)
		}
		if EnvToKey[env] != key {
			t.Errorf("%s does not round-trip: %s -> %s", key, env, EnvToKey[env])
		}
		if !strings.Contains(key, ".") {
			t.Errorf("%q is not a section.key", key)
		}
	}
}

func TestLoadFileTreatsAMissingFileAsAbsent(t *testing.T) {
	_, ok, err := LoadFile(filepath.Join(t.TempDir(), "nope.toml"), SourceUser)
	if err != nil {
		t.Fatalf("a missing config file is not an error: %v", err)
	}
	if ok {
		t.Error("a missing file must not produce a layer")
	}
}

func TestLoadFileSurfacesAParseError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("[model]\nbogus = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadFile(path, SourceUser); err == nil {
		t.Error("a malformed config file must fail loudly")
	}
}

func TestFileLayersLoadsUserThenProject(t *testing.T) {
	home := t.TempDir()
	ws := t.TempDir()

	write(t, filepath.Join(home, ConfigFileName), "[model]\nname = \"user-model\"\n")
	write(t, filepath.Join(ws, ".dcode", ConfigFileName), "[model]\nname = \"project-model\"\n")

	layers, err := FileLayers(Roots{Config: home}, ws, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(layers) != 2 {
		t.Fatalf("want both layers, got %d", len(layers))
	}
	if layers[0].Source != SourceUser || layers[1].Source != SourceProject {
		t.Errorf("got %v then %v", layers[0].Source, layers[1].Source)
	}

	// The repository is the more specific context, so it wins.
	r := Resolve(append([]Layer{{
		Source: SourceDefault, Origin: "built-in",
		Values: map[string]string{"model.name": "default-model"},
	}}, layers...))
	if got := r.String("model.name", ""); got != "project-model" {
		t.Errorf("project config must win over user config, got %q", got)
	}
	v, _ := r.Get("model.name")
	if v.Origin == "" {
		t.Error("every value needs an origin, or `--config` cannot answer where it came from")
	}
}

func TestFileLayersReportsABrokenProjectFile(t *testing.T) {
	ws := t.TempDir()
	write(t, filepath.Join(ws, ".dcode", ConfigFileName), "[model]\nunknown = 1\n")
	if _, err := FileLayers(Roots{Config: t.TempDir()}, ws, ""); err == nil {
		t.Error("a broken project config must fail")
	}
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
