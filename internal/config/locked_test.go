package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeLocked(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// RN-7 of sandbox-policy and RN-9 of configuration, which were unreachable:
// SourceLocked was defined, ranked highest in `authority`, and handled by
// Resolve — and no production code path ever built such a layer.
//
// The tests that existed only exercised hand-built layers, so nothing failed.
func TestTheLockedLayerBeatsEveryOtherSource(t *testing.T) {
	home, ws := t.TempDir(), t.TempDir()
	writeLocked(t, home, ConfigFileName, "[sandbox]\nmode = \"full-access\"\n")
	writeLocked(t, ws+"/.dcode", ConfigFileName, "[sandbox]\nmode = \"full-access\"\n")
	req := writeLocked(t, t.TempDir(), RequirementsFileName, "[sandbox]\nmode = \"read-only\"\n")

	layers, err := FileLayers(Roots{Config: home}, ws, req)
	if err != nil {
		t.Fatal(err)
	}
	// An environment layer too, since RN-7 says a variable does not override it
	// either.
	layers = append(layers, Layer{
		Source: SourceEnv, Origin: "environment",
		Values: map[string]string{"sandbox.mode": "full-access"},
	})

	got := Resolve(layers)
	v, ok := got.Get("sandbox.mode")
	if !ok {
		t.Fatal("sandbox.mode did not resolve at all")
	}
	if v.Value != "read-only" {
		t.Fatalf("sandbox.mode = %q, want the locked value: an administrator's policy that a user can override is not a policy", v.Value)
	}
	if !v.Locked {
		t.Error("the value is not marked locked, so `dcode config` cannot say why the user's setting lost")
	}
	if v.Source != SourceLocked {
		t.Errorf("source = %q, want locked", v.Source)
	}
}

// Losing silently is the failure that sends someone to spend an afternoon on
// it. RN-9 requires the attempt to be reported.
func TestAnOverriddenAttemptIsReported(t *testing.T) {
	home := t.TempDir()
	writeLocked(t, home, ConfigFileName, "[sandbox]\nallow_network = \"true\"\n")
	req := writeLocked(t, t.TempDir(), RequirementsFileName, "[sandbox]\nallow_network = \"false\"\n")

	layers, err := FileLayers(Roots{Config: home}, t.TempDir(), req)
	if err != nil {
		t.Fatal(err)
	}
	got := Resolve(layers)
	if len(got.Warnings) == 0 {
		t.Fatal("a setting was overridden by policy and nothing said so")
	}
	joined := strings.Join(got.Warnings, "\n")
	if !strings.Contains(joined, "allow_network") {
		t.Errorf("the warning does not name the key that lost:\n%s", joined)
	}
}

// The normal case: one person on their own machine, no administrator anywhere.
func TestNoRequirementsFileMeansNoLockedLayer(t *testing.T) {
	home := t.TempDir()
	writeLocked(t, home, ConfigFileName, "[sandbox]\nmode = \"full-access\"\n")

	layers, err := FileLayers(Roots{Config: home}, t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range layers {
		if l.Source == SourceLocked {
			t.Fatal("a locked layer appeared with no requirements file")
		}
	}
	if v, _ := Resolve(layers).Get("sandbox.mode"); v.Value != "full-access" {
		t.Errorf("the user's own setting did not win with nothing locking it: %q", v.Value)
	}
}

// A requirements file the deployment points at and that is not there is a
// deployment error, not a silent absence of policy. Failing closed here is the
// difference between "no policy" and "policy that did not load".
func TestAMissingRequirementsFileThatWasNamedIsAnError(t *testing.T) {
	_, err := FileLayers(Roots{Config: t.TempDir()}, t.TempDir(),
		filepath.Join(t.TempDir(), "absent.toml"))
	if err == nil {
		t.Fatal("a named requirements file that does not exist loaded as though there were no policy at all")
	}
}

// The omission that is the point.
//
// Every key in KnownKeys can be set from config.toml, which is a file the user
// owns. Letting that file say where the locked policy lives would let the user
// redirect the policy that binds them — the boundary would be a convention
// inside the thing it is meant to bind.
func TestTheRequirementsPathIsNotConfigurableFromConfigToml(t *testing.T) {
	for key, env := range KnownKeys {
		if env == "DCODE_REQUIREMENTS_FILE" {
			t.Fatalf("%q maps to DCODE_REQUIREMENTS_FILE: a user's own config.toml can now move the policy that binds them", key)
		}
	}
}
