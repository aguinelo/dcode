package config

import (
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

// The invariant that matters most here, and it is structural rather than
// reviewed: a command read from a stranger's repository is never executed to
// see whether it works. `npm install` fires a postinstall script, and the
// sandbox does not help — the command would run inside the workspace, which is
// exactly where the damage would be.
func TestTranslateNeverReachesForExecution(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "translate.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, imp := range f.Imports {
		switch imp.Path.Value {
		case `"os/exec"`, `"syscall"`:
			t.Errorf("translate.go imports %s: verifying by running is exactly what RN-6.1 forbids", imp.Path.Value)
		}
	}
}

func TestVerifyToolsAccusesWhatTheProductDoesNotHave(t *testing.T) {
	have := []string{"read", "write", "edit", "glob", "grep", "bash", "plan"}
	text := "Use the `Task` tool to spawn sub-agents. Prefer `read` over cat. " +
		"Call the memory_store tool for coordination."

	got := VerifyTools(text, have)
	var names []string
	for _, f := range got {
		names = append(names, f.Subject)
	}
	if !reflect.DeepEqual(names, []string{"Task", "memory_store"}) {
		t.Fatalf("findings = %v, want Task and memory_store", names)
	}
	for _, f := range got {
		if !strings.Contains(f.Reason, "read") {
			t.Errorf("the reason does not say what the real tools are: %q", f.Reason)
		}
	}
}

func TestVerifyToolsDoesNotAccuseAToolThatExists(t *testing.T) {
	have := []string{"read", "write", "symbol"}
	if got := VerifyTools("Use `read` and the `symbol` tool.", have); len(got) != 0 {
		t.Fatalf("accused tools that exist: %v", got)
	}
}

// Without the stop list every "the tool" becomes a finding, and a report full
// of noise is a report nobody reads.
func TestOrdinaryProseIsNotAFinding(t *testing.T) {
	have := []string{"read"}
	text := "Use the dedicated tool rather than a shell command. Any tool may fail."
	if got := VerifyTools(text, have); len(got) != 0 {
		t.Fatalf("plain prose produced findings: %v", got)
	}
}

func TestProbeCommandsAccusesWhatThisRepositoryCannotRun(t *testing.T) {
	fsys := fstest.MapFS{
		"go.mod":   &fstest.MapFile{Data: []byte("module x\n")},
		"Makefile": &fstest.MapFile{Data: []byte("check:\n")},
	}
	text := "Build with `npm run build`, then `go build ./...`, then `make check`, " +
		"then `cargo test`."

	got := ProbeCommands(text, fsys)
	var subjects []string
	for _, f := range got {
		subjects = append(subjects, f.Subject)
	}
	if !reflect.DeepEqual(subjects, []string{"cargo test", "npm run build"}) {
		t.Fatalf("findings = %v, want the npm and cargo commands only", subjects)
	}
	if !strings.Contains(got[1].Reason, "package.json") {
		t.Errorf("the reason does not name the file that is missing: %q", got[1].Reason)
	}
}

func TestProbeCommandsIsSilentWhenTheFileIsThere(t *testing.T) {
	fsys := fstest.MapFS{"package.json": &fstest.MapFile{Data: []byte("{}")}}
	if got := ProbeCommands("Run `npm run build`.", fsys); len(got) != 0 {
		t.Fatalf("accused a command this repository can run: %v", got)
	}
}

// ---------- digest and divergence ----------

func TestDigestChangesOnlyWithContent(t *testing.T) {
	a := fstest.MapFS{"AGENTS.md": &fstest.MapFile{Data: []byte("one")}}
	b := fstest.MapFS{"AGENTS.md": &fstest.MapFile{Data: []byte("one")}}
	c := fstest.MapFS{"AGENTS.md": &fstest.MapFile{Data: []byte("two")}}

	if SourceDigest(a, []string{"AGENTS.md"}) != SourceDigest(b, []string{"AGENTS.md"}) {
		t.Error("identical content produced different digests")
	}
	if SourceDigest(a, []string{"AGENTS.md"}) == SourceDigest(c, []string{"AGENTS.md"}) {
		t.Error("different content produced the same digest")
	}
}

func TestUnchangedSourcesDoNotDiverge(t *testing.T) {
	fsys := fstest.MapFS{"AGENTS.md": &fstest.MapFile{Data: []byte("rules")}}
	doc := "# DCODE\n\n" + RenderDigest(fsys, []string{"AGENTS.md"}) + "\n"

	if names, diverged := Diverged(doc, fsys); diverged {
		t.Fatalf("reported divergence with nothing changed: %v", names)
	}
}

// The warning has to say WHAT moved. "Something changed" sends a person
// looking; "AGENTS.md changed" is actionable.
func TestDivergenceNamesTheFileThatChanged(t *testing.T) {
	before := fstest.MapFS{
		"AGENTS.md":       &fstest.MapFile{Data: []byte("rules")},
		"CONTRIBUTING.md": &fstest.MapFile{Data: []byte("how to")},
	}
	doc := RenderDigest(before, []string{"AGENTS.md", "CONTRIBUTING.md"})

	after := fstest.MapFS{
		"AGENTS.md":       &fstest.MapFile{Data: []byte("rules, rewritten by another tool")},
		"CONTRIBUTING.md": &fstest.MapFile{Data: []byte("how to")},
	}
	names, diverged := Diverged(doc, after)
	if !diverged {
		t.Fatal("a rewritten source did not register")
	}
	if !reflect.DeepEqual(names, []string{"AGENTS.md"}) {
		t.Fatalf("named %v, want only the file that actually changed", names)
	}
}

func TestADeletedSourceIsNamedAsGone(t *testing.T) {
	before := fstest.MapFS{"AGENTS.md": &fstest.MapFile{Data: []byte("rules")}}
	doc := RenderDigest(before, []string{"AGENTS.md"})

	names, diverged := Diverged(doc, fstest.MapFS{})
	if !diverged || len(names) != 1 || !strings.Contains(names[0], "gone") {
		t.Fatalf("a deleted source reported %v / %v", names, diverged)
	}
}

func TestADocumentWithNoMarkerNeverDiverges(t *testing.T) {
	fsys := fstest.MapFS{"AGENTS.md": &fstest.MapFile{Data: []byte("rules")}}
	if _, diverged := Diverged("# hand written\n", fsys); diverged {
		t.Fatal("a hand-written DCODE.md was reported as diverged from sources it never had")
	}
}

// Scoping the detector to lines that are ABOUT tools is what separates a
// number worth reading from a number full of noise. Measured against dcode's
// own repository, taking every backticked identifier turned a list of commit
// types into findings.
func TestOnlyLinesAboutToolsAreScannedForNames(t *testing.T) {
	have := []string{"read", "write"}
	text := "Commit types: `feat`, `fix`, `chore`, `docs`.\n" +
		"Key tools: memory_store, agent_spawn, hooks_route.\n" +
		"Use the `Task` tool for sub-agents.\n" +
		"Run `make check` before pushing.\n"

	var names []string
	for _, f := range VerifyTools(text, have) {
		names = append(names, f.Subject)
	}
	for _, noise := range []string{"feat", "fix", "chore", "docs", "make"} {
		for _, n := range names {
			if strings.EqualFold(n, noise) {
				t.Errorf("%q was reported as a tool; it is a word on a line about something else", noise)
			}
		}
	}
	for _, real := range []string{"memory_store", "agent_spawn", "hooks_route", "Task"} {
		found := false
		for _, n := range names {
			if n == real {
				found = true
			}
		}
		if !found {
			t.Errorf("%q is named as a tool and was missed; findings were %v", real, names)
		}
	}
}
