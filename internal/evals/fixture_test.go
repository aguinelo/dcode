package evals

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

// Every scenario the specs declare a fixture for must actually load. This is
// the test that catches a threshold whose material was never written, or was
// renamed and left the spec pointing at nothing.
func TestEveryShippedFixtureLoads(t *testing.T) {
	ids := []string{
		"toolcall-schema-valid", "toolcall-recover", "no-phantom-tool",
		"notices-wrong-replacement",
		"records-before-compaction", "warns-when-task-exceeds-budget",
		"no-budget-noise-when-low",
		"runs-verification-after-change", "reports-failure-honestly",
		"states-what-was-not-verified", "no-verification-on-read-only",
		"fixes-cause-not-measure", "states-unmet-on-stall", "no-dod-on-read-only",
		"init-drops-absent-tool", "init-drops-absent-command",
		"init-keeps-real-convention", "init-does-not-execute",
		"delegates-wide-reads", "does-not-delegate-trivial", "reports-unread-paths",
		// The contracts that predate the harness. behavior-definition declared
		// ten and agent-loop five before there was anything to run them with.
		"tool-over-shell", "safety-not-overridable", "reminder-acted-upon",
		"reminder-not-user", "follows-project-instruction", "directory-over-project",
		"skill-loaded-on-trigger", "plan-depth-trivial", "plan-depth-complex",
		"plan-stays-live",
		"tool-error-recover", "tool-error-giveup", "no-blind-retry",
		"turn-ends-clean", "parallel-no-order-assumption",
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

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The defect that made six contracts score a flat zero: the prompt the model
// received was the task and nothing else. Every contract about behaviour the
// doctrine produces was being measured against a model that had never been
// told anything.
func TestTheAssembledCallCarriesTheDoctrine(t *testing.T) {
	f, err := LoadFixture(FixtureRoot, "tool-over-shell")
	if err != nil {
		t.Fatal(err)
	}
	msgs, err := f.Messages(context.Background(), "", t.TempDir(), []ce.Message{{Role: ce.RoleUser, Text: f.Task}})
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) < 2 {
		t.Fatalf("got %d messages, want a system prompt and the task", len(msgs))
	}
	if msgs[0].Role != ce.RoleSystem {
		t.Errorf("the first message is %v, want the system prompt", msgs[0].Role)
	}
	// The shipped doctrine names the tools and states the safety section; both
	// are refused by behavior.Build when absent, so their presence here is
	// what proves the real doctrine was built rather than a stand-in.
	for _, want := range []string{"Safety", "bash"} {
		if !strings.Contains(msgs[0].Text, want) {
			t.Errorf("the system prompt does not mention %q:\n%s", want, msgs[0].Text)
		}
	}
	if last := msgs[len(msgs)-1]; last.Text != f.Task {
		t.Errorf("the task is not the last message: %q", last.Text)
	}
}

// A scenario about which instruction wins has to be able to carry both, and
// the source has to survive loading — the ordering is the whole contract.
func TestAFixtureCarriesItsInstructionsInAuthorityOrder(t *testing.T) {
	f, err := LoadFixture(FixtureRoot, "directory-over-project")
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Instructions) != 2 {
		t.Fatalf("got %d instructions, want the project one and the directory one", len(f.Instructions))
	}
	prompt, err := f.Prompt("")
	if err != nil {
		t.Fatal(err)
	}
	project, directory := strings.Index(prompt, "Must"), strings.Index(prompt, "legacyTrimSuffix")
	if project < 0 || directory < 0 {
		t.Fatalf("the prompt is missing one of the two conventions:\n%s", prompt)
	}
	// Most specific last, which is the position of greatest weight (RN-4).
	if directory < project {
		t.Errorf("the directory instruction came before the project one, so the weaker rule reads as final:\n%s", prompt)
	}
}

// The skill index reaches the prompt as one line per skill, never a body.
func TestAFixtureCarriesItsSkillIndex(t *testing.T) {
	f, err := LoadFixture(FixtureRoot, "skill-loaded-on-trigger")
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Skills) == 0 {
		t.Fatal("the scenario is about a skill index and carries none")
	}
	prompt, err := f.Prompt("")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt, "release") {
		t.Errorf("the skill index is not in the prompt:\n%s", prompt)
	}
}

// A source that is not a source must not load. A file named `porject.md` would
// otherwise become an instruction with no authority ranking, sort to the front,
// and measure the opposite of what the scenario says.
func TestAnUnknownInstructionSourceIsRefused(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "bad")
	if err := os.MkdirAll(filepath.Join(dir, "bad", "instructions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad", "instructions", "porject.md"),
		[]byte("something"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFixture(dir, "bad"); err == nil {
		t.Fatal("a misspelled source loaded as though it were one")
	}
}

// An empty instruction file is material that looks present and contributes
// nothing — the exact shape of the defect this whole package exists to catch.
func TestAnEmptyInstructionIsRefused(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "bad")
	if err := os.MkdirAll(filepath.Join(dir, "bad", "instructions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad", "instructions", "project.md"),
		[]byte("  \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFixture(dir, "bad"); err == nil {
		t.Fatal("an empty instruction loaded")
	}
}

// A skill index entry with no when-to-use tells the model nothing about when
// to load it, which makes the index material that cannot do its job.
func TestASkillWithNoTriggerIsRefused(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "bad")
	if err := os.WriteFile(filepath.Join(dir, "bad", "skills.json"),
		[]byte(`[{"name":"release"}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFixture(dir, "bad"); err == nil {
		t.Fatal("a skill with no trigger loaded")
	}
}

func TestMalformedSkillsAreRefused(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "bad")
	if err := os.WriteFile(filepath.Join(dir, "bad", "skills.json"),
		[]byte(`{"name":"release"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFixture(dir, "bad"); err == nil {
		t.Fatal("a skills.json that is not a list loaded")
	}
}

// writeFixture lays down the minimum a fixture needs to load.
func writeFixture(t *testing.T, root, id string) {
	t.Helper()
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "task.md"), []byte("do a thing"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tools.json"),
		[]byte(`[{"name":"read","description":"d","schema":{"type":"object"}}]`), 0o644); err != nil {
		t.Fatal(err)
	}
}
