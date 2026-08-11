package behavior

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden files")

// The assembled prefix is the one string in the product that must be
// byte-identical between turns, because the provider's cache is keyed on it
// (ADR-03). A test that asserts "contains the safety text" cannot see a change
// in spacing, in ordering, or in a heading — and each of those costs the whole
// cached prompt on every turn of every session, silently, as a bill.
func goldenPrompt(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".txt")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s: %v\nRun `go test ./internal/behavior -update` to record it.", path, err)
	}
	if got != string(want) {
		t.Errorf("%s changed.\n\nwant:\n%s\n\ngot:\n%s\n\n"+
			"Every byte here is paid for on every turn of every session, and a change "+
			"invalidates the provider cache. If it is intended, re-record with -update.",
			path, want, got)
	}
}

func goldenFixture() Prompt {
	return Prompt{
		Doctrine: DefaultDoctrine([]string{"bash", "edit", "explore", "glob", "grep", "plan", "read", "symbol", "write"}),
		Tools:    []string{"read", "write"},
		SkillIndex: []SkillIndexEntry{
			{Name: "release", WhenToUse: "preparing a release"},
			{Name: "debug", WhenToUse: "chasing a failing test"},
		},
		Instructions: []Instruction{
			{Source: SourceUser, Scope: "~/.config/dcode/DCODE.md", Text: "Prefer small commits."},
			{Source: SourceProject, Scope: "DCODE.md", Text: "Run make check before pushing."},
			{Source: SourceLocked, Locked: true, Scope: "requirements.toml", Text: "Never disable the sandbox."},
		},
	}
}

func TestGoldenPromptMarkdown(t *testing.T) {
	got, err := Build(goldenFixture(), FormulationFor("minimax-m3"))
	if err != nil {
		t.Fatal(err)
	}
	goldenPrompt(t, "prompt_markdown", got)
}

// The other formulation, recorded separately, so RN-8 has a visible artefact
// rather than only an inequality assertion.
func TestGoldenPromptTagged(t *testing.T) {
	got, err := Build(goldenFixture(), FormulationFor("claude"))
	if err != nil {
		t.Fatal(err)
	}
	goldenPrompt(t, "prompt_tagged", got)
}

func TestGoldenPromptBare(t *testing.T) {
	got, err := Build(Prompt{Doctrine: DefaultDoctrine([]string{"read"})}, FormulationFor("minimax-m3"))
	if err != nil {
		t.Fatal(err)
	}
	goldenPrompt(t, "prompt_bare", got)
}
