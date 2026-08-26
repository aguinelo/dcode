package loopcommand

import (
	"os"
	"path/filepath"
	"reflect"
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

// SourceAuto with a spec path that is not there falls through to the legacy
// verify command.
//
// This is the branch that never ran. LoadSpec wraps its error with %w and the
// dispatcher asked os.IsNotExist, which does not follow a wrapped chain — so
// every missing spec path came back as a hard error while a comment beside
// the check said it fell through. The one existing fall-through test passed a
// path of "", which never reaches this code at all.
func TestLoadSourceAutoMissingSpecFallsThroughToVerify(t *testing.T) {
	set, err := Load(t.TempDir(), filepath.Join(t.TempDir(), "not-there"), SourceAuto, "make check")
	if err != nil {
		t.Fatalf("a missing spec should fall through, not fail: %v", err)
	}
	if len(set.Criteria) != 1 || set.Criteria[0].Command != "make check" {
		t.Fatalf("expected the legacy verify command, got %+v", set.Criteria)
	}
}

// A spec that IS there and cannot be read is an error, not a fall-through.
// "Not there" and "unreadable" are different answers and the dispatcher has
// to keep them apart: falling through on an unreadable spec would run the
// legacy command under a spec the user thought was loaded.
func TestLoadSourceAutoMalformedSpecIsAnError(t *testing.T) {
	spec := t.TempDir()
	if err := os.WriteFile(filepath.Join(spec, "tasks.md"), []byte("nada aqui é tarefa\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(t.TempDir(), spec, SourceAuto, "make check"); err == nil {
		t.Fatal("an unreadable spec fell through to the legacy command")
	}
}

// A Source that is not one of the three is an error rather than a silent
// SourceAuto — the same reason ParseSource refuses an unknown string.
func TestLoadUnknownSourceIsAnError(t *testing.T) {
	if _, err := Load(t.TempDir(), "", Source(99), "make check"); err == nil {
		t.Fatal("an unknown source came back with no error")
	}
}

// Same inputs, same DoneSet. The dispatcher reads the filesystem and nothing
// else; a second call on an unchanged tree cannot answer differently.
func TestLoadIsDeterministic(t *testing.T) {
	spec := t.TempDir()
	if err := os.WriteFile(filepath.Join(spec, "tasks.md"), []byte(
		"- [ ] 1. desc. verify: `true`\n- [ ] 2. desc. verify: `false`, exit: 1\n",
	), 0o644); err != nil {
		t.Fatal(err)
	}
	first, err := Load(t.TempDir(), spec, SourceLoopSpec, "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Load(t.TempDir(), spec, SourceLoopSpec, "")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("two identical calls disagreed:\n %+v\n %+v", first, second)
	}
}

// SourceDoneFile with no done.toml falls back to the legacy verify command
// rather than failing — the file is the preferred source, not a requirement.
func TestLoadSourceDoneFileWithoutTheFileUsesVerify(t *testing.T) {
	set, err := Load(t.TempDir(), "", SourceDoneFile, "make check")
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Criteria) != 1 || set.Criteria[0].Command != "make check" {
		t.Fatalf("expected the legacy verify command, got %+v", set.Criteria)
	}
}

// done.toml carries protected in its root section and exit codes per
// criterion; both reach the DoneSet.
func TestLoadDoneTOMLCarriesProtectedAndExitCodes(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".dcode"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".dcode", "done.toml"), []byte(
		"protected = \"**/*_test.go, docs/**\"\n\n[tests]\ncommand = \"make test\"\n\n[no-todo]\ncommand = \"grep -q TODO .\"\nexit_code = \"1\"\n",
	), 0o644); err != nil {
		t.Fatal(err)
	}
	set, err := Load(root, "", SourceDoneFile, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Protected) != 2 || set.Protected[0] != "**/*_test.go" || set.Protected[1] != "docs/**" {
		t.Fatalf("protected not read: %+v", set.Protected)
	}
	if len(set.Criteria) != 2 || set.Criteria[1].ExitCode != 1 {
		t.Fatalf("exit code not read: %+v", set.Criteria)
	}
}

// An exit_code that does not read as a number stays at zero rather than
// failing the load. This mirrors internal/app/done.go, which is the parser
// this one duplicates; changing it here alone would give the same file two
// meanings depending on which door it came through.
func TestLoadDoneTOMLUnreadableExitCodeStaysZero(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".dcode"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".dcode", "done.toml"), []byte(
		"[tests]\ncommand = \"make test\"\nexit_code = \"não é número\"\n",
	), 0o644); err != nil {
		t.Fatal(err)
	}
	set, err := Load(root, "", SourceDoneFile, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Criteria) != 1 || set.Criteria[0].ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %+v", set.Criteria)
	}
}

// A done.toml that cannot be read at all — a directory where a file was
// promised — is an error, distinct from an absent one.
func TestLoadDoneTOMLUnreadableFileIsAnError(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".dcode", "done.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root, "", SourceDoneFile, "make check"); err == nil {
		t.Fatal("an unreadable done.toml came back with no error")
	}
}
