package evals

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every scenario the specs declare a fixture for must actually load. This is
// the test that catches a threshold whose material was never written, or was
// renamed and left the spec pointing at nothing.
func TestEveryShippedFixtureLoads(t *testing.T) {
	ids := []string{
		"toolcall-schema-valid", "toolcall-recover", "no-phantom-tool",
		"notices-wrong-replacement",
		"records-before-compaction", "warns-when-task-exceeds-budget",
		"runs-verification-after-change", "reports-failure-honestly",
		"states-what-was-not-verified", "no-verification-on-read-only",
		"fixes-cause-not-measure", "states-unmet-on-stall", "no-dod-on-read-only",
		"init-drops-absent-tool", "init-drops-absent-command",
		"init-keeps-real-convention", "init-does-not-execute",
	}
	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			f, err := LoadFixture(FixtureRoot, id)
			if err != nil {
				t.Fatalf("the spec names this fixture and it does not load: %v", err)
			}
			if f.Task == "" {
				t.Error("task is empty")
			}
			if len(f.Tools) == 0 {
				t.Error("no tools declared")
			}
			for _, tool := range f.Tools {
				if tool.Name == "" {
					t.Error("a declared tool has no name")
				}
				if len(tool.Schema) == 0 {
					t.Errorf("tool %q has no schema: a fixture measuring schema fidelity needs one", tool.Name)
				}
			}
		})
	}
}

func TestMissingFixtureIsAnError(t *testing.T) {
	if _, err := LoadFixture(FixtureRoot, "no-such-scenario"); err == nil {
		t.Fatal("a missing fixture loaded without error: a scenario with no material must fail loudly, never measure nothing")
	}
}

func TestEmptyTaskIsAnError(t *testing.T) {
	dir := t.TempDir()
	id := "blank"
	if err := os.MkdirAll(filepath.Join(dir, id), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, id, "task.md"), "   \n")
	write(t, filepath.Join(dir, id, "tools.json"), `[{"name":"read","description":"d","schema":{}}]`)

	_, err := LoadFixture(dir, id)
	if err == nil {
		t.Fatal("a blank task loaded: the model would be asked nothing and every run would pass")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error does not say what was wrong: %v", err)
	}
}

func TestToolsetWithNoToolsIsAnError(t *testing.T) {
	dir := t.TempDir()
	id := "toolless"
	if err := os.MkdirAll(filepath.Join(dir, id), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, id, "task.md"), "do something")
	write(t, filepath.Join(dir, id, "tools.json"), `[]`)

	if _, err := LoadFixture(dir, id); err == nil {
		t.Fatal("an empty tool set loaded: no-phantom-tool would pass by having nothing to call")
	}
}

func TestMalformedToolsetIsAnError(t *testing.T) {
	dir := t.TempDir()
	id := "broken"
	if err := os.MkdirAll(filepath.Join(dir, id), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, id, "task.md"), "do something")
	write(t, filepath.Join(dir, id, "tools.json"), `{ not json`)

	if _, err := LoadFixture(dir, id); err == nil {
		t.Fatal("malformed tools.json loaded without error")
	}
}

func TestDeclaresAnswersTheSetMembershipQuestion(t *testing.T) {
	f, err := LoadFixture(FixtureRoot, "no-phantom-tool")
	if err != nil {
		t.Fatal(err)
	}
	if !f.Declares("read") {
		t.Error("read is in the fixture and Declares said no")
	}
	if f.Declares("delete_file") {
		t.Error("delete_file is not in the fixture and Declares said yes: this is the exact question the phantom-tool judge asks")
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
