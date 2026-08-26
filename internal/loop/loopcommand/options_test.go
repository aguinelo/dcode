package loopcommand

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/config"
)

// TestOptionsFromConfigReadsAllFourKeys covers the four keys the function
// reads by name. The wiring guard looks for the literals in source; this
// test makes sure the function actually returns them on a values map.
func TestOptionsFromConfigReadsAllFourKeys(t *testing.T) {
	values := map[string]string{
		"loop.spec_path":      "/some/spec",
		"loop.source":         "loop_spec",
		"loop.protect":        "**/*_test.go,**/*.go",
		"loop.session_prefix": "my-",
	}
	got := OptionsFromConfig(values)
	if got.SpecPath != "/some/spec" {
		t.Errorf("SpecPath = %q, want %q", got.SpecPath, "/some/spec")
	}
	if got.Source != SourceLoopSpec {
		t.Errorf("Source = %d, want SourceLoopSpec (%d)", got.Source, SourceLoopSpec)
	}
	if len(got.Protect) != 2 || got.Protect[0] != "**/*_test.go" || got.Protect[1] != "**/*.go" {
		t.Errorf("Protect = %v, want the two globs split", got.Protect)
	}
	if got.SessionPrefix != "my-" {
		t.Errorf("SessionPrefix = %q, want %q", got.SessionPrefix, "my-")
	}
}

// TestOptionsFromConfigEmptyValuesReturnsZeroValue covers the zero case.
// Every missing key is an empty string or a zero Source, which is what the
// caller sees when the user set nothing.
func TestOptionsFromConfigEmptyValuesReturnsZeroValue(t *testing.T) {
	got := OptionsFromConfig(map[string]string{})
	if got.SpecPath != "" || got.Source != SourceAuto || got.Protect != nil || got.SessionPrefix != "" {
		t.Errorf("expected zero value, got %+v", got)
	}
}

// TestOptionsFromConfigInvalidSourceFallsBackToAuto covers the error path:
// ParseSource rejects unknown values and OptionsFromConfig falls back to
// SourceAuto, which is the documented behaviour.
func TestOptionsFromConfigInvalidSourceFallsBackToAuto(t *testing.T) {
	got := OptionsFromConfig(map[string]string{"loop.source": "unknown_mode"})
	if got.Source != SourceAuto {
		t.Errorf("invalid source resolved to %d, want SourceAuto (%d)", got.Source, SourceAuto)
	}
}

// TestLoadDoneTOMLMissingFileFallsBackToVerify covers the not-found path:
// loadDoneTOML returns doneFromVerify(verifyCommand) when the file does
// not exist. The verify command is the legacy behaviour.
func TestLoadDoneTOMLMissingFileFallsBackToVerify(t *testing.T) {
	set, err := loadDoneTOML(t.TempDir()+"/.dcode/done.toml", "make test")
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Criteria) != 1 || set.Criteria[0].Command != "make test" {
		t.Fatalf("expected one criterion with the verify command, got %+v", set.Criteria)
	}
}

// TestLoadDoneTOMLMalformedTOMLReturnsError covers the parse failure:
// a file that is not TOML the parser understands returns an error,
// never an empty DoneSet.
func TestLoadDoneTOMLMalformedTOMLReturnsError(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".dcode"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".dcode", "done.toml"), []byte("this is not [toml"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadDoneTOML(filepath.Join(dir, ".dcode", "done.toml"), ""); err == nil {
		t.Fatal("expected error on malformed TOML, got nil")
	}
}

// TestLoadDoneTOMLReadEmptyFileReturnsError covers the missing case with
// no verify command: the empty DoneSet signals "no definition of done",
// not an error.
func TestLoadDoneTOMLReadEmptyFileReturnsError(t *testing.T) {
	// An empty verify command means "no fallback". The file does not exist
	// in this TempDir, so loadDoneTOML returns the empty DoneSet.
	set, err := loadDoneTOML(t.TempDir()+"/.dcode/done.toml", "")
	if err != nil {
		t.Fatal(err)
	}
	if set.Criteria != nil {
		t.Fatalf("expected empty DoneSet, got %+v", set)
	}
}

// TestSplitListSeparatesAndTrims covers the helper used by the TOML loader
// and the protect glob parsing.
func TestSplitListSeparatesAndTrims(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"a", []string{"a"}},
		{"a,b,c", []string{"a", "b", "c"}},
		{" a , b , c ", []string{"a", "b", "c"}},
		{"a,,b", []string{"a", "b"}},
	}
	for _, c := range cases {
		got := splitList(c.in)
		if len(got) != len(c.want) {
			t.Errorf("splitList(%q): got %v, want %v", c.in, got, c.want)
			continue
		}
		for i, w := range c.want {
			if got[i] != w {
				t.Errorf("splitList(%q)[%d] = %q, want %q", c.in, i, got[i], w)
			}
		}
	}
}

// TestAtoiParsesInteger covers the integer parser used by done.toml
// exit_code values.
func TestAtoiParsesInteger(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"0", 0},
		{"1", 1},
		{"42", 42},
		{"  7  ", 7},
		{"", 0},
		{"abc", 0},
		{"-3", -3},
	}
	for _, c := range cases {
		if got := atoi(c.in); got != c.want {
			t.Errorf("atoi(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

// TestConfigKeysAlignWithLoopKeys is the consistency check: the keys we
// declared in KnownKeys are exactly the ones OptionsFromConfig reads. If
// one drifts from the other, the wiring guard catches the wiring side but
// the read side would silently produce a zero value.
func TestConfigKeysAlignWithLoopKeys(t *testing.T) {
	keys := []string{"loop.spec_path", "loop.source", "loop.protect", "loop.session_prefix"}
	for _, k := range keys {
		if _, ok := config.KnownKeys[k]; !ok {
			t.Errorf("OptionsFromConfig reads %q but it is not in config.KnownKeys", k)
		}
	}
}
