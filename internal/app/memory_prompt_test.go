package app

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func withMemory(t *testing.T, body string) Options {
	t.Helper()
	ws := t.TempDir()
	if err := os.MkdirAll(filepath.Join(ws, ".dcode"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, ".dcode", "memory.md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	opts := baseOpts(t)
	opts.Workspace = ws
	opts.Memory = true
	return opts
}

func prompt(t *testing.T, opts Options) string {
	t.Helper()
	requireSandbox(t, opts)
	sess, err := New(opts, &ConsoleEmitter{W: io.Discard}, DenyAll{})
	if err != nil {
		t.Fatalf("wiring a session failed: %v", err)
	}
	t.Cleanup(sess.Engine.Close)
	return sess.Prompt
}

// What an earlier session learned reaches the model, and reaches it marked as
// something the agent noted rather than something a person required.
//
// Asserted end to end because the two halves were separately correct and
// unconnected is the shape this codebase keeps finding: a renderer that knows
// how to say it and nothing reading it.
func TestASessionReadsWhatEarlierSessionsLearned(t *testing.T) {
	got := prompt(t, withMemory(t,
		"## gotcha: make test precisa de go generate antes\n\nos gerados ficam velhos.\n"))

	if !strings.Contains(got, "go generate") {
		t.Errorf("the memory did not reach the prompt:\n%s", got)
	}
	if !strings.Contains(got, "learned") {
		t.Errorf("the prompt does not mark it as learned:\n%s", got)
	}
	if !strings.Contains(got, ".dcode/memory.md") {
		t.Errorf("the prompt does not say where it came from:\n%s", got)
	}
}

// Nothing learned outranks anything a person wrote, end to end: the memory
// appears before the project's own instructions, which is the weaker position.
func TestWhatWasLearnedIsWeighedBelowWhatAPersonWrote(t *testing.T) {
	opts := withMemory(t, "## gotcha: LEARNED-NOTE\n\nbody.\n")
	// Options carries no defaults of its own — the configuration chain supplies
	// them — so the switch has to be set explicitly here.
	opts.Instructions = true
	if err := os.WriteFile(filepath.Join(opts.Workspace, "AGENTS.md"),
		[]byte("PROJECT-RULE: keep files short.\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got := prompt(t, opts)
	learned := strings.Index(got, "LEARNED-NOTE")
	rule := strings.Index(got, "PROJECT-RULE")
	if learned < 0 || rule < 0 {
		t.Fatalf("learned at %d, rule at %d — both must be present", learned, rule)
	}
	if learned > rule {
		t.Error("the learned note is weighed above the project's own rule")
	}
}

// A workspace that never learned anything opens exactly as it always did.
func TestAWorkspaceWithNoMemoryIsUnchanged(t *testing.T) {
	opts := baseOpts(t)
	opts.Memory = true
	if strings.Contains(prompt(t, opts), "earlier sessions") {
		t.Error("a workspace with no memory got a memory block")
	}
}

// Off is the product from before this existed.
func TestMemoryTurnedOffLeavesThePromptAlone(t *testing.T) {
	opts := withMemory(t, "## gotcha: SHOULD-NOT-APPEAR\n\nbody.\n")
	opts.Memory = false
	if strings.Contains(prompt(t, opts), "SHOULD-NOT-APPEAR") {
		t.Error("memory was read with the feature off")
	}
}

// A memory from a commit the repository no longer has is marked in the prompt
// and still there. The model reads that and weighs it; nothing decides for it.
func TestAMemoryFromAVanishedCommitIsMarkedInThePrompt(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git here")
	}
	opts := withMemory(t, `## gotcha: still here
<!-- learned 2026-08-18 · commit HEAD_SHA -->

body.

## gotcha: from a rebased commit
<!-- learned 2026-08-18 · commit 0000000000000000000000000000000000000000 -->

body.
`)
	ws := opts.Workspace
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.email", "t@e.com"},
		{"config", "user.name", "T"},
		{"config", "commit.gpgsign", "false"},
		{"add", "-A"},
		{"commit", "-qm", "first"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = ws
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git %v: %v\n%s", args, err, out)
		}
	}
	head, err := exec.Command("git", "-C", ws, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	// Rewrite the placeholder now that there is a commit to name.
	path := filepath.Join(ws, ".dcode", "memory.md")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path,
		[]byte(strings.ReplaceAll(string(body), "HEAD_SHA", strings.TrimSpace(string(head)))), 0o600); err != nil {
		t.Fatal(err)
	}

	got := prompt(t, opts)
	if !strings.Contains(got, "from a rebased commit") {
		t.Fatal("a memory from a vanished commit was dropped from the prompt")
	}
	if strings.Count(got, "no longer in this repository") != 1 {
		t.Errorf("expected exactly the one mark:\n%s", got)
	}
}
