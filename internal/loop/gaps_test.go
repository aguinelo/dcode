package loop

import (
	"os"
	"path/filepath"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/tools"
)

// readFile adapts os.ReadFile to the shape the engine asks for.
func readFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	return string(b), err
}

func state(t *testing.T) (*tools.State, string) {
	t.Helper()
	r, err := policy.NewResolver(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return tools.NewState(r, tools.DefaultLimits(), nil), r.Workspace
}

// Undo belongs to the session, not to the model, so it is reached through the
// engine rather than through the registry. The engine owns the state that knows
// what changed, and that is the only reason it owns this at all.
func TestTheEngineUndoesThroughTheStateItOwns(t *testing.T) {
	st, dir := state(t)
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}

	st.BeginTurn()
	st.Snapshot(path)
	if err := os.WriteFile(path, []byte("after"), 0o600); err != nil {
		t.Fatal(err)
	}
	// What a writing tool does after it writes, and what makes the file part of
	// the turn: the hash recorded here is what undo compares against.
	st.MarkRead(path, "after", 0)
	st.MarkWritten(path)

	e := New(Config{State: st}, ce.Session{Instructions: "x"})
	restored, refused, err := e.Undo()
	if err != nil {
		t.Fatalf("undo failed: %v", err)
	}
	if len(refused) != 0 {
		t.Fatalf("a file nothing else touched was refused: %v", refused)
	}
	if len(restored) != 1 {
		t.Fatalf("restored %v, want the one file the turn changed", restored)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "before" {
		t.Errorf("the file holds %q, want the content from before the turn", b)
	}
}

// A session with no tool state has nothing to put back, and says so by
// answering rather than by panicking. The client asks the same way either way.
func TestUndoingWithoutStateIsAnAnswerNotACrash(t *testing.T) {
	e := New(Config{}, ce.Session{Instructions: "x"})
	restored, refused, err := e.Undo()
	if err != nil || restored != nil || refused != nil {
		t.Fatalf("got %v, %v, %v — want an empty answer", restored, refused, err)
	}
}

// Closing is what makes "a process dies with its session" a consequence of
// ownership. Both shapes have to work: the engine that owns state closes it,
// and the engine that owns none closes anyway.
func TestClosingTheEngineClosesTheStateAndToleratesNone(t *testing.T) {
	st, _ := state(t)
	New(Config{State: st}, ce.Session{Instructions: "x"}).Close()
	New(Config{}, ce.Session{Instructions: "x"}).Close()
}

// An instruction file in a directory the batch touched, that was not in the
// chain frozen at session start, is reported once — and only once, because the
// second report would be noise about a fact that has not changed.
func TestAnInstructionFoundOffTheChainIsReportedOnce(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "pkg")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	off := filepath.Join(sub, "AGENTS.md")
	if err := os.WriteFile(off, []byte("keep lines short"), 0o600); err != nil {
		t.Fatal(err)
	}

	e := New(Config{
		ReadFile:         readFile,
		InstructionChain: []string{filepath.Join(dir, "AGENTS.md")},
	}, ce.Session{Instructions: "x"})

	got := e.outOfChain([]string{sub})
	if len(got) != 1 {
		t.Fatalf("got %d instructions, want the one off the chain", len(got))
	}
	if got[0].Path != off {
		t.Errorf("reported %q, want %q", got[0].Path, off)
	}
	if got[0].Text != "keep lines short" {
		t.Errorf("reported text %q, want the file's contents", got[0].Text)
	}

	if again := e.outOfChain([]string{sub}); len(again) != 0 {
		t.Errorf("the same directory was reported twice: %v", again)
	}
}

// Three ways this correctly finds nothing, and none of them is an error: no
// reader configured, no directories touched, and a directory whose instruction
// file is already in the chain.
func TestOutOfChainFindsNothingWhenThereIsNothingToFind(t *testing.T) {
	dir := t.TempDir()
	onChain := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(onChain, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	noReader := New(Config{InstructionChain: []string{onChain}}, ce.Session{Instructions: "x"})
	if got := noReader.outOfChain([]string{dir}); got != nil {
		t.Errorf("with no reader configured: %v", got)
	}

	e := New(Config{ReadFile: readFile, InstructionChain: []string{onChain}}, ce.Session{Instructions: "x"})
	if got := e.outOfChain(nil); got != nil {
		t.Errorf("with no directories touched: %v", got)
	}
	if got := e.outOfChain([]string{dir}); len(got) != 0 {
		t.Errorf("a file already in the chain was reported as off it: %v", got)
	}
}

// A file that cannot be read is skipped rather than reported empty. An
// instruction with no text is worse than no instruction: it says a rule exists
// and does not say what it is.
func TestAnUnreadableInstructionIsSkippedRatherThanReportedEmpty(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "pkg")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	// A directory where the instruction file should be: reading it fails.
	if err := os.MkdirAll(filepath.Join(sub, "AGENTS.md"), 0o755); err != nil {
		t.Fatal(err)
	}

	e := New(Config{
		ReadFile:         readFile,
		InstructionChain: []string{filepath.Join(dir, "AGENTS.md")},
	}, ce.Session{Instructions: "x"})

	if got := e.outOfChain([]string{sub}); len(got) != 0 {
		t.Errorf("an unreadable instruction was reported: %v", got)
	}
}
