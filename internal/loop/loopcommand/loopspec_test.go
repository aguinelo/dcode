package loopcommand

import (
	"os"
	"testing"

	"github.com/aguinelo/dcode/internal/loop"
)

// A spec with two criteria and a frontmatter protected verifies the golden
// shape: name, command, exit code, and protected from the frontmatter.
func TestLoadSpecHappyPath(t *testing.T) {
	dir := t.TempDir()
	markdown := `---
protected = ["**/*_test.go"]
---

# Tasks — Demo

- [ ] 1. ` + "`path/to.ts`" + ` — desc. verify: ` + "`pnpm test`" + `
- [ ] 2. ` + "`path/to2.ts`" + ` — desc. verify: ` + "`make lint`" + `, exit: 0
`
	if err := os.WriteFile(dir+"/tasks.md", []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}

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
	dir := t.TempDir()
	markdown := `# Tasks — Demo

- [ ] 1. ` + "`a.ts`" + ` — desc. verify: ` + "`make test`" + `
- [ ] 2. ` + "`b.ts`" + ` — smoke manual com o usuário.
`
	if err := os.WriteFile(dir+"/tasks.md", []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}

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

// Zero explicit criteria is LoopSpec with nil Criteria, not an error. The
// empty list IS the spec — erroring on it would invent a missing-prerequisite
// story that does not exist.
func TestLoadSpecZeroCriteriaIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	markdown := `# Tasks — Empty

Nenhuma tarefa aqui ainda.
`
	if err := os.WriteFile(dir+"/tasks.md", []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadSpec(dir)
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	if got.Criteria != nil {
		t.Fatalf("expected nil Criteria, got %+v", got.Criteria)
	}
}

// Malformed input returns an error — never a silently-empty DoneSet that the
// agent would interpret as "nothing to verify" and report done.
func TestLoadSpecMalformedReturnsError(t *testing.T) {
	dir := t.TempDir()
	// Frontmatter declared but never closed.
	markdown := `---
protected = ["**/*_test.go"]
# Tasks — Broken
`
	if err := os.WriteFile(dir+"/tasks.md", []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadSpec(dir); err == nil {
		t.Fatal("expected error on unclosed frontmatter, got nil")
	}
}

// Missing tasks.md is an error — the parser promises to read what the user
// named, and "no such file" is not what they named.
func TestLoadSpecMissingFileIsAnError(t *testing.T) {
	dir := t.TempDir()
	if _, err := LoadSpec(dir); err == nil {
		t.Fatal("expected error on missing tasks.md, got nil")
	}
}

// Frontmatter `protected` plus an argument layer on top: both end up in the
// final Protected list. When they conflict, the file-declared one is what the
// user wrote; the argument layers on top.
func TestLoadSpecWithProtectLayersBoth(t *testing.T) {
	dir := t.TempDir()
	markdown := `---
protected = ["**/*_test.go"]
---

- [ ] 1. ` + "`a.ts`" + ` — desc. verify: ` + "`true`" + `
`
	if err := os.WriteFile(dir+"/tasks.md", []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}

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

// `exit: K` overrides the default zero exit code.
func TestLoadSpecExitCodeOverride(t *testing.T) {
	dir := t.TempDir()
	markdown := `- [ ] 1. ` + "`a.ts`" + ` — desc. verify: ` + "`grep -q TODO .`" + `, exit: 1
`
	if err := os.WriteFile(dir+"/tasks.md", []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadSpec(dir)
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	if len(got.Criteria) != 1 || got.Criteria[0].ExitCode != 1 {
		t.Fatalf("expected ExitCode 1, got %+v", got.Criteria)
	}
}

// Order of appearance in tasks.md is the order in Criteria — humans read in
// order, and Progressed is insensitive to it but the printed report is not.
func TestLoadSpecPreservesOrder(t *testing.T) {
	dir := t.TempDir()
	markdown := `- [ ] 3. ` + "`c.ts`" + ` — desc. verify: ` + "`true`" + `
- [ ] 1. ` + "`a.ts`" + ` — desc. verify: ` + "`true`" + `
- [ ] 2. ` + "`b.ts`" + ` — desc. verify: ` + "`true`" + `
`
	if err := os.WriteFile(dir+"/tasks.md", []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}

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
	dir := t.TempDir()
	markdown := `- [x] 1. ` + "`a.ts`" + ` — desc. verify: ` + "`true`" + `
`
	if err := os.WriteFile(dir+"/tasks.md", []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadSpec(dir)
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	if len(got.Criteria) != 1 {
		t.Fatalf("- [x] deveria virar critério: %+v", got.Criteria)
	}
}

func specsEqual(a, b LoopSpec) bool {
	if a.Path != b.Path {
		return false
	}
	if len(a.Criteria) != len(b.Criteria) {
		return false
	}
	for i := range a.Criteria {
		if a.Criteria[i] != b.Criteria[i] {
			return false
		}
	}
	if len(a.Protected) != len(b.Protected) {
		return false
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
