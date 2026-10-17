package loopcommand

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/config"
)

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

// /loop has no configuration of its own. The four loop.* keys were declared,
// accepted and shown by `dcode config`, and read by a function nothing called;
// a key that is accepted and changes nothing promises a control that is not
// there. They stay out of the schema, so a config.toml naming one is refused
// rather than silently ignored.
func TestLoopHasNoConfigurationKeys(t *testing.T) {
	for _, k := range []string{"loop.spec_path", "loop.source", "loop.protect", "loop.session_prefix"} {
		if _, ok := config.KnownKeys[k]; ok {
			t.Errorf("%q is a configuration key again; /loop reads its folder and --protect, nothing else", k)
		}
	}
	for env := range config.EnvToKey {
		if strings.HasPrefix(env, "DCODE_LOOP_") {
			t.Errorf("%s is a configuration variable again; /loop has none", env)
		}
	}
}
