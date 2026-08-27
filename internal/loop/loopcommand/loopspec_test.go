package loopcommand

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/loop"
)

// write puts a tasks.md in a fresh directory and hands back the directory.
func write(t *testing.T, markdown string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// bt is a backtick. Go has no escape for one inside a raw string, and the
// fixtures below are mostly backticks.
const bt = "`"

// A spec with two criteria and a frontmatter protected verifies the golden
// shape: name, command, exit code, and protected from the frontmatter.
func TestLoadSpecHappyPath(t *testing.T) {
	dir := write(t, `---
protected = ["**/*_test.go"]
---

# Tasks — Demo

- [ ] 1. `+bt+`path/to.ts`+bt+` — desc. verify: `+bt+`pnpm test`+bt+`
- [ ] 2. `+bt+`path/to2.ts`+bt+` — desc. verify: `+bt+`make lint`+bt+`, exit: 0
`)

	got, err := LoadSpec(dir)
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}

	want := LoopSpec{
		Path: dir,
		Criteria: []loop.Criterion{
			{Name: "1", Command: "pnpm test", ExitCode: 0},
			{Name: "2", Command: "make lint", ExitCode: 0},
		},
		Protected: []string{"**/*_test.go"},
	}
	if !specsEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// A task without `verify:` is ignored by the parser — it surfaces as
// CriterionUnavailable in the final report via the existing empty-command
// path in done.go.
func TestLoadSpecIgnoresProse(t *testing.T) {
	dir := write(t, `# Tasks — Demo

Um parágrafo de prosa que menciona verify: e não é tarefa nenhuma.

- [ ] 1. `+bt+`a.ts`+bt+` — desc. verify: `+bt+`make test`+bt+`
- [ ] 2. `+bt+`b.ts`+bt+` — smoke manual com o usuário.
`)

	got, err := LoadSpec(dir)
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}

	if len(got.Criteria) != 1 {
		t.Fatalf("prosa virou critério: %+v", got.Criteria)
	}
	if got.Criteria[0].Name != "1" {
		t.Fatalf("primeiro critério errado: %+v", got.Criteria[0])
	}
}

// A tasks.md whose tasks carry no `verify:` declares zero criteria — that is
// the file saying "nothing here can be checked", and it is not an error.
//
// This is the ONE way an empty DoneSet is legitimate. The other shape that
// used to produce one — a file with no tasks at all — is the error below.
func TestLoadSpecZeroCriteriaIsNotAnError(t *testing.T) {
	dir := write(t, `# Tasks — Manual only

- [ ] 1. `+bt+`a.ts`+bt+` — smoke manual, sem comando.
- [ ] 2. `+bt+`b.ts`+bt+` — validar com o usuário.
`)

	got, err := LoadSpec(dir)
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	if got.Criteria != nil {
		t.Fatalf("expected nil Criteria, got %+v", got.Criteria)
	}
}

// A file with no task list at all is an error, not an empty DoneSet.
//
// This is RN-6, and it is the defect that mattered most: the parser used to
// skip every line it did not recognise, so a file of pure prose — or a
// tasks.md written in a shape the parser does not know — came back as zero
// criteria and a nil error. Zero criteria is "no definition of done", which
// the agent loop reports as done. An unreadable file would have become a
// green report.
func TestLoadSpecWithoutTaskLinesIsAnError(t *testing.T) {
	dir := write(t, "isto aqui é lixo\n@@@@ não é uma lista de tarefas\n")

	_, err := LoadSpec(dir)
	if err == nil {
		t.Fatal("a file with no task line came back with no error")
	}
	if !strings.Contains(err.Error(), "no task line") {
		t.Fatalf("error does not say what is missing: %v", err)
	}
}

// The separator between the task number and its description is documentation,
// not syntax.
//
// The parser required a literal em dash. A tasks.md written with a plain
// hyphen — which is what a keyboard produces — yielded zero criteria and no
// error, so an entire file of real work read as "no definition of done". One
// keystroke away from a false green.
func TestLoadSpecSeparatorIsNotSyntax(t *testing.T) {
	for _, form := range []struct {
		name string
		line string
	}{
		{"em dash", "- [ ] 1. " + bt + "a.ts" + bt + " — desc. verify: " + bt + "true" + bt},
		{"en dash", "- [ ] 1. " + bt + "a.ts" + bt + " – desc. verify: " + bt + "true" + bt},
		{"hyphen", "- [ ] 1. " + bt + "a.ts" + bt + " - desc. verify: " + bt + "true" + bt},
		{"colon", "- [ ] 1. " + bt + "a.ts" + bt + ": desc. verify: " + bt + "true" + bt},
		{"no path at all", "- [ ] 1. desc. verify: " + bt + "true" + bt},
		{"tab indented", "\t- [ ] 1. desc. verify: " + bt + "true" + bt},
	} {
		t.Run(form.name, func(t *testing.T) {
			got, err := LoadSpec(write(t, form.line+"\n"))
			if err != nil {
				t.Fatalf("LoadSpec: %v", err)
			}
			if len(got.Criteria) != 1 || got.Criteria[0].Command != "true" {
				t.Fatalf("%s dropped the criterion: %+v", form.name, got.Criteria)
			}
		})
	}
}

// `verify:` announced and nothing runnable behind it is an error that names
// the line — the author meant to declare a check and the file does not carry
// one, which is the difference between "no check here" and "a check I could
// not read".
func TestLoadSpecBadVerifyNamesTheLine(t *testing.T) {
	dir := write(t, `# Tasks

- [ ] 1. `+bt+`a.ts`+bt+` — desc. verify: `+bt+`true`+bt+`
- [ ] 2. `+bt+`b.ts`+bt+` — desc. verify: pnpm test
`)

	_, err := LoadSpec(dir)
	if err == nil {
		t.Fatal("a verify: with no command in backticks came back with no error")
	}
	if !strings.Contains(err.Error(), "line 4") {
		t.Fatalf("error does not name the line: %v", err)
	}
}

// An empty command between backticks is the same defect wearing the right
// punctuation.
func TestLoadSpecEmptyVerifyCommandIsAnError(t *testing.T) {
	if _, err := LoadSpec(write(t, "- [ ] 1. desc. verify: "+bt+bt+"\n")); err == nil {
		t.Fatal("an empty verify command came back with no error")
	}
}

// `exit: K` overrides the default zero exit code.
func TestLoadSpecExitCodeOverride(t *testing.T) {
	got, err := LoadSpec(write(t, "- [ ] 1. `a.ts` — desc. verify: "+bt+"grep -q TODO ."+bt+", exit: 1\n"))
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	if len(got.Criteria) != 1 || got.Criteria[0].ExitCode != 1 {
		t.Fatalf("expected ExitCode 1, got %+v", got.Criteria)
	}
}

// An exit code that does not read as a number is an error, not a silent zero.
//
// Silently defaulting here would give the author the opposite of what they
// asked for: they declared that a non-zero exit means met, and the criterion
// would be checked for zero.
func TestLoadSpecUnreadableExitCodeIsAnError(t *testing.T) {
	_, err := LoadSpec(write(t, "- [ ] 1. desc. verify: "+bt+"true"+bt+", exit: um\n"))
	if err == nil {
		t.Fatal("an unreadable exit code came back with no error")
	}
	if !strings.Contains(err.Error(), "exit") {
		t.Fatalf("error does not name what it could not read: %v", err)
	}
}

// Prose after the command is for the human and does not change what runs.
func TestLoadSpecTrailingProseAfterTheCommandIsKept(t *testing.T) {
	got, err := LoadSpec(write(t, "- [ ] 1. desc. verify: "+bt+"pnpm test"+bt+" (roda no CI também)\n"))
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	if len(got.Criteria) != 1 || got.Criteria[0].Command != "pnpm test" {
		t.Fatalf("trailing prose changed the command: %+v", got.Criteria)
	}
}

// The same task number twice is an error. Name is what the report prints and
// what Progressed compares, so two criteria called "3" are two rows nobody can
// tell apart — and one of them silently standing in for the other.
func TestLoadSpecDuplicateTaskNumberIsAnError(t *testing.T) {
	dir := write(t, `- [ ] 1. desc. verify: `+bt+`true`+bt+`
- [ ] 2. desc. verify: `+bt+`true`+bt+`
- [ ] 1. desc. verify: `+bt+`false`+bt+`
`)

	_, err := LoadSpec(dir)
	if err == nil {
		t.Fatal("a duplicated task number came back with no error")
	}
	if !strings.Contains(err.Error(), "line 1") {
		t.Fatalf("error does not name where the first one was: %v", err)
	}
}

// Malformed input returns an error — never a silently-empty DoneSet that the
// agent would interpret as "nothing to verify" and report done.
func TestLoadSpecMalformedReturnsError(t *testing.T) {
	dir := write(t, `---
protected = ["**/*_test.go"]
# Tasks — Broken
`)

	if _, err := LoadSpec(dir); err == nil {
		t.Fatal("expected error on unclosed frontmatter, got nil")
	}
}

// `---` mid-file is markdown's horizontal rule, not a frontmatter delimiter.
//
// Treating one as an opening delimiter swallowed every line after it, so a
// tasks.md with a section break lost the tasks below it — or failed whole as
// "frontmatter never closed" when there was no second rule.
func TestLoadSpecHorizontalRuleIsNotFrontmatter(t *testing.T) {
	dir := write(t, `# Tasks — Demo

- [ ] 1. desc. verify: `+bt+`true`+bt+`

---

## Bloco 2

- [ ] 2. desc. verify: `+bt+`false`+bt+`
`)

	got, err := LoadSpec(dir)
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	if len(got.Criteria) != 2 {
		t.Fatalf("a section break ate the tasks below it: %+v", got.Criteria)
	}
}

// Missing tasks.md is an error — the parser promises to read what the user
// named, and "no such file" is not what they named.
func TestLoadSpecMissingFileIsAnError(t *testing.T) {
	if _, err := LoadSpec(t.TempDir()); err == nil {
		t.Fatal("expected error on missing tasks.md, got nil")
	}
}

// Frontmatter `protected` and the argument are a UNION, both present in the
// result.
//
// Not a precedence. Protected is what gets surfaced when touched, so "the file
// wins over the argument" would let an argument remove a protection the file
// asked for — the one direction that must never be reachable by accident.
func TestLoadSpecWithProtectLayersBoth(t *testing.T) {
	dir := write(t, `---
protected = ["**/*_test.go"]
---

- [ ] 1. desc. verify: `+bt+`true`+bt+`
`)

	got, err := LoadSpecWithProtect(dir, []string{"**/*.go"})
	if err != nil {
		t.Fatalf("LoadSpecWithProtect: %v", err)
	}

	if !contains(got.Protected, "**/*_test.go") {
		t.Fatalf("file-declared protected dropped: %+v", got.Protected)
	}
	if !contains(got.Protected, "**/*.go") {
		t.Fatalf("argument protected dropped: %+v", got.Protected)
	}
}

// The same glob from both sources appears once. A report that lists a path
// twice reads as two different protections.
func TestLoadSpecProtectIsNotDuplicated(t *testing.T) {
	dir := write(t, `---
protected = ["**/*_test.go"]
---

- [ ] 1. desc. verify: `+bt+`true`+bt+`
`)

	got, err := LoadSpecWithProtect(dir, []string{"**/*_test.go", ""})
	if err != nil {
		t.Fatalf("LoadSpecWithProtect: %v", err)
	}
	if len(got.Protected) != 1 {
		t.Fatalf("expected one glob, got %+v", got.Protected)
	}
}

// Nothing declared means nothing protected. RN-4: the harness does not decide
// what counts as the measurement, and the position of the spec is not a
// declaration.
func TestLoadSpecWithoutProtectDeclaresNothing(t *testing.T) {
	got, err := LoadSpec(write(t, "- [ ] 1. desc. verify: "+bt+"true"+bt+"\n"))
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	if got.Protected != nil {
		t.Fatalf("nothing was declared and something is protected: %+v", got.Protected)
	}
}

// An empty list declared is still nothing, and a frontmatter carrying other
// keys does not invent behaviour for them.
func TestLoadSpecFrontmatterEdges(t *testing.T) {
	for _, tc := range []struct {
		name        string
		frontmatter string
	}{
		{"empty list", `protected = []`},
		{"list of empties", `protected = ["", ""]`},
		{"no protected key", `title = "whatever"`},
		{"single quotes", `protected = ['a.go']`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := write(t, "---\n"+tc.frontmatter+"\n---\n\n- [ ] 1. desc. verify: "+bt+"true"+bt+"\n")
			got, err := LoadSpec(dir)
			if err != nil {
				t.Fatalf("LoadSpec: %v", err)
			}
			if tc.name == "single quotes" {
				if len(got.Protected) != 1 || got.Protected[0] != "a.go" {
					t.Fatalf("single-quoted glob not read: %+v", got.Protected)
				}
				return
			}
			if got.Protected != nil {
				t.Fatalf("%s produced a protection: %+v", tc.name, got.Protected)
			}
		})
	}
}

// Order of appearance in tasks.md is the order in Criteria — humans read in
// order, and Progressed is insensitive to it but the printed report is not.
func TestLoadSpecPreservesOrder(t *testing.T) {
	dir := write(t, `- [ ] 3. `+bt+`c.ts`+bt+` — desc. verify: `+bt+`true`+bt+`
- [ ] 1. `+bt+`a.ts`+bt+` — desc. verify: `+bt+`true`+bt+`
- [ ] 2. `+bt+`b.ts`+bt+` — desc. verify: `+bt+`true`+bt+`
`)

	got, err := LoadSpec(dir)
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	if len(got.Criteria) != 3 ||
		got.Criteria[0].Name != "3" ||
		got.Criteria[1].Name != "1" ||
		got.Criteria[2].Name != "2" {
		t.Fatalf("ordem não preservada: %+v", got.Criteria)
	}
}

// Tarefa - [x] é tratada igual a - [ ]: a marcação é trabalho do agente,
// não do parser. A verificação ainda precisa rodar.
func TestLoadSpecCheckedTaskIsStillACriterion(t *testing.T) {
	got, err := LoadSpec(write(t, "- [x] 1. `a.ts` — desc. verify: "+bt+"true"+bt+"\n"))
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	if len(got.Criteria) != 1 {
		t.Fatalf("- [x] deveria virar critério: %+v", got.Criteria)
	}
}

func specsEqual(a, b LoopSpec) bool {
	if a.Path != b.Path || len(a.Criteria) != len(b.Criteria) || len(a.Protected) != len(b.Protected) {
		return false
	}
	for i := range a.Criteria {
		if a.Criteria[i] != b.Criteria[i] {
			return false
		}
	}
	for i := range a.Protected {
		if a.Protected[i] != b.Protected[i] {
			return false
		}
	}
	return true
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

// A spec folder can declare its own definition of done, in commands.
//
// Because the criteria a project can actually run are often nowhere in its
// tasks. The specs this family was built for declare their acceptance criteria
// as SENTENCES — "Lighthouse >= 95", "loads in under a second on 4G" — which is
// what a person writes and what no parser may turn into a command without
// inventing one. The folder gets a place to say it in commands instead.
func TestASpecFolderCanDeclareItsOwnDoneFile(t *testing.T) {
	dir := write(t, "# Tasks\n\n- [ ] 1. `a.ts` — smoke manual, sem comando.\n")
	if err := os.WriteFile(filepath.Join(dir, "done.toml"), []byte(
		"protected = \"**/*_test.ts\"\n\n[coverage]\ncommand = \"pnpm test:coverage\"\n\n[no-todo]\ncommand = \"grep -q TODO .\"\nexit_code = \"1\"\n",
	), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadSpecWithProtect(dir, []string{"docs/**"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Criteria) != 2 {
		t.Fatalf("got %+v", got.Criteria)
	}
	if got.Criteria[0].Name != "coverage" || got.Criteria[1].ExitCode != 1 {
		t.Fatalf("criteria wrong: %+v", got.Criteria)
	}
	// The protect argument unions with what the file declared, same as it does
	// for a tasks.md.
	if !contains(got.Protected, "**/*_test.ts") || !contains(got.Protected, "docs/**") {
		t.Errorf("protected = %+v", got.Protected)
	}
	if got.Path != dir {
		t.Errorf("path = %q", got.Path)
	}
}

// It wins over tasks.md, and tasks.md is not consulted at all.
//
// Two sources for one folder would be two answers to "what is this measured
// against", and the one nobody reads is the one that drifts.
func TestTheSpecDoneFileWinsOverTasks(t *testing.T) {
	dir := write(t, "- [ ] 1. desc. verify: "+bt+"from-tasks"+bt+"\n")
	if err := os.WriteFile(filepath.Join(dir, "done.toml"),
		[]byte("[from-file]\ncommand = \"true\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSpec(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Criteria) != 1 || got.Criteria[0].Name != "from-file" {
		t.Fatalf("tasks.md was consulted anyway: %+v", got.Criteria)
	}
}

// A done.toml that declares nothing is an error, not an empty DoneSet: a
// definition of done with nothing in it reports done.
func TestAnEmptySpecDoneFileIsAnError(t *testing.T) {
	dir := write(t, "- [ ] 1. desc. verify: "+bt+"true"+bt+"\n")
	if err := os.WriteFile(filepath.Join(dir, "done.toml"), []byte("# nothing here\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSpec(dir); err == nil {
		t.Fatal("an empty done.toml fell through to tasks.md instead of failing")
	}
}

// No done.toml is the ordinary case and not an error: the folder declares its
// criteria in tasks.md, or not at all.
func TestNoSpecDoneFileFallsToTasks(t *testing.T) {
	got, err := LoadSpec(write(t, "- [ ] 1. desc. verify: "+bt+"true"+bt+"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Criteria) != 1 || got.Criteria[0].Command != "true" {
		t.Fatalf("got %+v", got.Criteria)
	}
}
