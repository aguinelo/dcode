package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/config"
)

func ws(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

var toolSet = []string{"read", "write", "edit", "glob", "grep", "symbol", "bash", "plan"}

func TestNothingIsSaidWhenThereIsNothingToSay(t *testing.T) {
	if got := InstructionNotice(ws(t, nil), ForeignDefault, toolSet); got != "" {
		t.Fatalf("said %q with no instruction files present", got)
	}
}

// The measurement that started all of this, reproduced as a test.
func TestAForeignInstructionFileIsAnnouncedWithItsSize(t *testing.T) {
	dir := ws(t, map[string]string{
		"AGENTS.md": "Use the `Task` tool to spawn sub-agents.\n" +
			"Build with `npm run build`.\n" +
			"Call the memory_store tool for coordination.\n",
	})
	got := InstructionNotice(dir, ForeignDefault, toolSet)
	if got == "" {
		t.Fatal("said nothing about a file written for another tool")
	}
	for _, want := range []string{"AGENTS.md", "bytes", "/init"} {
		if !strings.Contains(got, want) {
			t.Errorf("the notice does not carry %q: %s", want, got)
		}
	}
	if !strings.Contains(got, "3 things") {
		t.Errorf("the notice does not count what does not exist here: %s", got)
	}
}

// A gate that stops you before you have asked anything is a gate you learn to
// walk through without reading.
func TestTheNoticeIsAStringAndNothingElse(t *testing.T) {
	dir := ws(t, map[string]string{"AGENTS.md": "Use the `Task` tool.\n"})
	if got := InstructionNotice(dir, ForeignDefault, toolSet); got == "" {
		t.Fatal("expected a notice")
	}
	// Nothing was created, nothing was moved, nothing was rewritten.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("the notice touched the workspace: %v", entries)
	}
}

func TestWithADcodeFileTheNoticeIsAboutDivergenceOnly(t *testing.T) {
	dir := ws(t, map[string]string{
		"AGENTS.md": "original rules",
		"DCODE.md":  "# dcode\n",
	})
	fsys := os.DirFS(dir)
	doc := "# dcode\n\n" + config.RenderDigest(fsys, []string{"AGENTS.md"}) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "DCODE.md"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := InstructionNotice(dir, ForeignDefault, toolSet); got != "" {
		t.Fatalf("said %q with a current DCODE.md", got)
	}

	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"),
		[]byte("rewritten by another tool"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := InstructionNotice(dir, ForeignDefault, toolSet)
	if !strings.Contains(got, "AGENTS.md") {
		t.Errorf("divergence does not name the file that moved: %s", got)
	}
	if !strings.Contains(got, "has not been touched") {
		t.Errorf("the notice does not say the file was left alone: %s", got)
	}
}

// DCODE.md is the output of the translation, never an input to it.
func TestDcodeMdIsNeverTreatedAsAForeignFile(t *testing.T) {
	got := foreignFiles("AGENTS.md, DCODE.md, CLAUDE.md")
	for _, f := range got {
		if f == "DCODE.md" {
			t.Fatalf("DCODE.md was accepted as a translation source: %v", got)
		}
	}
	if len(got) != 2 {
		t.Fatalf("foreign files = %v, want the two that are not DCODE.md", got)
	}
}

func TestAnEmptySettingFallsBackToTheDefaults(t *testing.T) {
	if got := foreignFiles("  "); len(got) != len(ForeignDefault) {
		t.Fatalf("foreign files = %v, want the defaults", got)
	}
}

// The measurement that started the whole exercise, run against this very
// repository. AGENTS.md here describes claude-flow: sub-agents through a Task
// tool dcode does not have, MCP it does not speak, and npm in a Go repository.
func TestThisRepositoryIsTheCaseThatMotivatedThis(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err != nil {
		t.Skip("no AGENTS.md at the repository root")
	}
	got := InstructionNotice(root, ForeignDefault, toolSet)
	if got == "" {
		t.Fatal("dcode's own repository carries instructions for another tool and the session says nothing")
	}
	t.Logf("session-start notice for this repository:\n  %s", got)
}
