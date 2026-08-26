package loopcommand

import (
	"os"
	"path/filepath"
	"testing"
)

// SourceLoopSpec with an empty path is an error — silently falling back to
// the legacy verify command would hide the user's mistake.
func TestLoadSourceLoopSpecRequiresPath(t *testing.T) {
	if _, err := Load("/workspace", "", SourceLoopSpec, ""); err == nil {
		t.Fatal("expected error on empty spec path")
	}
}

// SourceLoopSpec reads the spec's tasks.md and returns its DoneSet.
func TestLoadSourceLoopSpecReadsFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(
		"- [ ] 1. `a.ts` — desc. verify: `true`\n",
	), 0o644); err != nil {
		t.Fatal(err)
	}
	set, err := Load("/anywhere", dir, SourceLoopSpec, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Criteria) != 1 || set.Criteria[0].Name != "1" {
		t.Fatalf("expected one criterion named 1, got %+v", set.Criteria)
	}
}

// SourceDoneFile reads done.toml even when a spec path is set — the explicit
// source wins over the alternative, by the dispatch contract.
func TestLoadSourceDoneFileIgnoresSpecPath(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".dcode"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".dcode", "done.toml"), []byte(
		"[tests]\ncommand = \"make test\"\n",
	), 0o644); err != nil {
		t.Fatal(err)
	}
	set, err := Load(root, "/some/spec", SourceDoneFile, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Criteria) != 1 || set.Criteria[0].Command != "make test" {
		t.Fatalf("expected criterion from done.toml, got %+v", set.Criteria)
	}
}

// SourceAuto prefers done.toml when present, even with a spec path.
func TestLoadSourceAutoPrefersDoneToml(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".dcode"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".dcode", "done.toml"), []byte(
		"[tests]\ncommand = \"make test\"\n",
	), 0o644); err != nil {
		t.Fatal(err)
	}
	set, err := Load(root, "/anywhere", SourceAuto, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Criteria) != 1 || set.Criteria[0].Command != "make test" {
		t.Fatalf("expected done.toml to win, got %+v", set.Criteria)
	}
}

// SourceAuto falls through to LoopSpec when done.toml is missing.
func TestLoadSourceAutoFallsThroughToLoopSpec(t *testing.T) {
	root := t.TempDir()
	spec := t.TempDir()
	if err := os.WriteFile(filepath.Join(spec, "tasks.md"), []byte(
		"- [ ] 1. `a.ts` — desc. verify: `make test`\n",
	), 0o644); err != nil {
		t.Fatal(err)
	}
	set, err := Load(root, spec, SourceAuto, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Criteria) != 1 || set.Criteria[0].Command != "make test" {
		t.Fatalf("expected LoopSpec fallback, got %+v", set.Criteria)
	}
}

// SourceAuto with neither file falls through to the legacy verifyCommand.
func TestLoadSourceAutoFallsThroughToVerify(t *testing.T) {
	set, err := Load(t.TempDir(), "", SourceAuto, "make check")
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Criteria) != 1 || set.Criteria[0].Command != "make check" {
		t.Fatalf("expected verify command, got %+v", set.Criteria)
	}
}

// SourceAuto with nothing at all yields an empty DoneSet — "no definition of
// done" is what the engine reports, not an error.
func TestLoadSourceAutoNothingYieldsEmpty(t *testing.T) {
	set, err := Load(t.TempDir(), "", SourceAuto, "")
	if err != nil {
		t.Fatal(err)
	}
	if set.Criteria != nil {
		t.Fatalf("expected nil Criteria, got %+v", set.Criteria)
	}
}

func TestParseSourceValid(t *testing.T) {
	cases := map[string]Source{
		"":          SourceAuto,
		"auto":      SourceAuto,
		"done_file": SourceDoneFile,
		"loop_spec": SourceLoopSpec,
	}
	for in, want := range cases {
		got, err := ParseSource(in)
		if err != nil {
			t.Fatalf("ParseSource(%q): %v", in, err)
		}
		if got != want {
			t.Fatalf("ParseSource(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestParseSourceInvalid(t *testing.T) {
	if _, err := ParseSource("loop_spec_typo"); err == nil {
		t.Fatal("expected error on unknown source")
	}
}
