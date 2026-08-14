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

// Content, not modification time: a file rewritten with the same bytes has not
// changed, and a clock would make the digest differ between machines.
func TestDigestChangesOnlyWithContent(t *testing.T) {
	a := fstest.MapFS{"AGENTS.md": &fstest.MapFile{Data: []byte("one")}}
	b := fstest.MapFS{"AGENTS.md": &fstest.MapFile{Data: []byte("one")}}
	c := fstest.MapFS{"AGENTS.md": &fstest.MapFile{Data: []byte("two")}}

	if RenderDigest(a, []string{"AGENTS.md"}) != RenderDigest(b, []string{"AGENTS.md"}) {
		t.Error("identical content produced different digests")
	}
	if RenderDigest(a, []string{"AGENTS.md"}) == RenderDigest(c, []string{"AGENTS.md"}) {
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

// A tool name is a name. "the file tools" is prose.
//
// VerifyTools flagged the word before "tool" whatever it was, held back only by
// a list of common words that could never be complete — its own comment admits
// "without this every 'the tool' becomes a finding". So a DCODE.md saying
// "prefer the file tools over shell commands" reported `file` as a tool dcode
// does not have.
//
// That is not only a judge problem. This function also drives the session-start
// notice, which tells the user how many things the instructions name that do
// not exist here — a count that was wrong for any prose mentioning tools.
func TestOnlySomethingShapedLikeAToolNameIsAFinding(t *testing.T) {
	have := []string{"bash", "edit", "glob", "grep", "plan", "read", "write"}

	prose := []string{
		"Prefer the file tools over shell commands.",
		"There are no sub-agent tools here.",
		"These tools are available: read, write, edit.",
		"Use the dedicated tools rather than the shell.",
		"Every tool reports what it touched.",
		"Several tools can run at once.",
	}
	for _, s := range prose {
		if f := VerifyTools(s, have); len(f) > 0 {
			t.Errorf("prose read as a tool name: %q -> %v", s, f)
		}
	}

	named := map[string]string{
		"Use the Task tool to spawn subagents for anything big.": "Task",
		"Run it with the `Dispatcher` tool.":                     "Dispatcher",
		"The Dispatch tool handles fan-out.":                     "Dispatch",
	}
	for s, want := range named {
		f := VerifyTools(s, have)
		if len(f) != 1 || f[0].Subject != want {
			t.Errorf("%q -> %v, want one finding for %q", s, f, want)
		}
	}
}

// A tool this build has is never a finding, however it is written.
func TestAToolThatExistsIsNeverAFinding(t *testing.T) {
	have := []string{"bash", "edit", "glob", "read"}
	for _, s := range []string{
		"Use the Read tool for files.",
		"Use the `glob` tool to find them.",
		"tools: read, write, glob",
	} {
		if f := VerifyTools(s, have); len(f) > 0 {
			for _, x := range f {
				if strings.EqualFold(x.Subject, "read") || strings.EqualFold(x.Subject, "glob") {
					t.Errorf("%q flagged a tool that exists: %v", s, f)
				}
			}
		}
	}
}
