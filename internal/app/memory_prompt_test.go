package app

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
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

// A memory nobody can read is said to whoever attaches, and said after the
// session announces itself.
//
// New says it while the session is being built, and through the daemon that is
// before the session it is building exists: the emitter New is handed forwarded
// only once there was a session to forward to, so the one thing New had to say
// went nowhere, and the session opened as though its memory had been read.
//
// A directory where the file should be: os.Open succeeds and the read does not,
// which holds for any user, root included, where a mode bit would not.
func TestAMemoryThatCannotBeReadIsSaidToWhoeverAttaches(t *testing.T) {
	ws := t.TempDir()
	if err := os.MkdirAll(filepath.Join(ws, ".dcode", "memory.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, c, _ := daemonFor(t, ws)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sess, err := c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: ws})
	if err != nil {
		t.Fatalf("a memory that cannot be read refused the session: %v", err)
	}
	events, errs := c.Subscribe(ctx, sess.ID, 1)
	var types []protocol.EventType
	var said *protocol.Error
	for done := false; !done; {
		select {
		case ev, ok := <-events:
			if !ok {
				t.Fatalf("the stream ended before seq %d", sess.LastSeq)
			}
			types = append(types, ev.Type)
			if ev.Type == protocol.EventSessionError {
				var e protocol.Error
				if err := json.Unmarshal(ev.Payload, &e); err != nil {
					t.Fatalf("a session.error that does not decode: %v", err)
				}
				if e.Code == "memory_unreadable" {
					said = &e
				}
			}
			done = ev.Seq >= sess.LastSeq
		case err, ok := <-errs:
			if ok && err != nil {
				t.Fatal(err)
			}
			errs = nil
		case <-ctx.Done():
			t.Fatalf("the events up to seq %d never arrived", sess.LastSeq)
		}
	}
	if said == nil {
		t.Fatalf("the session opened without saying its memory could not be read; "+
			"an attached client read %v", types)
	}
	if said.Message == "" {
		t.Error("memory_unreadable said nothing about why")
	}
	if types[0] != protocol.EventSessionCreated {
		t.Errorf("the log opens with %s, want %s: %v", types[0], protocol.EventSessionCreated, types)
	}
}
