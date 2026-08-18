package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/memory"
)

func rememberCall(t *testing.T, s *State, args any) Result {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Remember{Commit: "a2c6e69", Today: "2026-08-18"}.
		Execute(context.Background(), raw, s)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func memoryOf(t *testing.T, ws string) string {
	t.Helper()
	b, err := os.ReadFile(memory.Path(ws))
	if err != nil {
		t.Fatalf("no memory was written: %v", err)
	}
	return string(b)
}

// Something learned is written where the next session will read it, in the
// grammar that reads it back.
func TestRememberingWritesSomethingTheNextSessionCanRead(t *testing.T) {
	s, ws := setup(t)

	res := rememberCall(t, s, map[string]any{
		"kind":    "gotcha",
		"subject": "make test precisa de go generate antes",
		"body":    "os arquivos gerados ficam velhos.",
	})
	if res.IsError {
		t.Fatalf("%s", res.Output)
	}

	got, err := memory.Read(ws)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 1 {
		t.Fatalf("read back %+v", got.Entries)
	}
	e := got.Entries[0]
	if e.Kind != memory.KindGotcha || !strings.Contains(e.Subject, "go generate") {
		t.Errorf("got %+v", e)
	}
	if e.Body != "os arquivos gerados ficam velhos." {
		t.Errorf("body = %q", e.Body)
	}
}

// Every memory carries when it was learned and at which commit. That is what
// makes staleness checkable rather than guessed at.
func TestEveryMemoryCarriesItsProvenance(t *testing.T) {
	s, ws := setup(t)
	rememberCall(t, s, map[string]any{"kind": "decision", "subject": "x", "body": "y"})

	raw := memoryOf(t, ws)
	if !strings.Contains(raw, "a2c6e69") {
		t.Errorf("the commit is missing:\n%s", raw)
	}
	if !strings.Contains(raw, "2026-08-18") {
		t.Errorf("the date is missing:\n%s", raw)
	}
	got, _ := memory.Read(ws)
	if got.Entries[0].Commit != "a2c6e69" || got.Entries[0].Learned != "2026-08-18" {
		t.Errorf("provenance did not survive the round trip: %+v", got.Entries[0])
	}
}

// It appends. Rewriting the file to sort, dedupe or tidy would turn a
// three-line change into a whole-file diff, and an unreadable diff is a review
// that does not happen — the only quality gate this design has.
func TestRememberingAppendsAndLeavesWhatWasThere(t *testing.T) {
	s, ws := setup(t)
	existing := "## convention: escrito à mão\n\nninguém mexe nisto.\n"
	if err := os.MkdirAll(filepath.Dir(memory.Path(ws)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(memory.Path(ws), []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}

	rememberCall(t, s, map[string]any{"kind": "gotcha", "subject": "novo", "body": "b"})

	raw := memoryOf(t, ws)
	if !strings.HasPrefix(raw, existing) {
		t.Errorf("what was already there was rewritten:\n%s", raw)
	}
	if !strings.Contains(raw, "novo") {
		t.Errorf("the new memory is missing:\n%s", raw)
	}
	got, _ := memory.Read(ws)
	if len(got.Entries) != 2 {
		t.Errorf("read back %d memories, want both", len(got.Entries))
	}
}

// A kind outside the three is refused, and the refusal names them. "invalid
// kind" leaves the model guessing at a list it cannot see.
func TestAKindOutsideTheThreeIsRefusedByName(t *testing.T) {
	s, _ := setup(t)
	res := rememberCall(t, s, map[string]any{"kind": "note", "subject": "x", "body": "y"})
	if !res.IsError {
		t.Fatal("an unknown kind was accepted")
	}
	for _, want := range []string{"gotcha", "decision", "convention"} {
		if !strings.Contains(res.Output, want) {
			t.Errorf("the refusal does not name %q:\n%s", want, res.Output)
		}
	}
}

// A memory with no subject is a memory nobody finds.
func TestAMemoryWithNoSubjectIsRefused(t *testing.T) {
	s, _ := setup(t)
	for _, subject := range []string{"", "   "} {
		if res := rememberCall(t, s, map[string]any{
			"kind": "gotcha", "subject": subject, "body": "y",
		}); !res.IsError {
			t.Errorf("subject %q was accepted", subject)
		}
	}
}

// The policy decides on paths, and the only path this touches is the memory.
// A tool that declared more than it writes would be asking about access it does
// not need.
func TestRememberDeclaresTheMemoryAndNothingElse(t *testing.T) {
	req, err := Remember{}.Declare(json.RawMessage(
		`{"kind":"gotcha","subject":"x","body":"y"}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(req.Paths) != 1 {
		t.Fatalf("declared %+v", req.Paths)
	}
	if req.Paths[0].Path != memory.FileName || !req.Paths[0].Write {
		t.Errorf("declared %+v, want a write to %s", req.Paths[0], memory.FileName)
	}
	if req.Network {
		t.Error("remembering declared the network")
	}
}

// The file is created on the first memory, directory and all. A workspace that
// never learned anything has neither.
func TestTheFirstMemoryCreatesTheFile(t *testing.T) {
	s, ws := setup(t)
	if _, err := os.Stat(memory.Path(ws)); err == nil {
		t.Fatal("the workspace already had a memory")
	}
	if res := rememberCall(t, s, map[string]any{
		"kind": "gotcha", "subject": "first", "body": "b",
	}); res.IsError {
		t.Fatalf("%s", res.Output)
	}
	if _, err := os.Stat(memory.Path(ws)); err != nil {
		t.Errorf("the file was not created: %v", err)
	}
}

// A memory written now does not reach this session's prefix, and cannot: the
// prefix was frozen when the session opened. The tool says so rather than
// letting the model wait for something that will not arrive.
func TestTheResultSaysItLandsNextSession(t *testing.T) {
	s, _ := setup(t)
	res := rememberCall(t, s, map[string]any{"kind": "gotcha", "subject": "x", "body": "y"})
	if !strings.Contains(strings.ToLower(res.Output), "next session") {
		t.Errorf("the result does not say when it takes effect:\n%s", res.Output)
	}
}

// Without provenance to record, it still records the memory. A build with no
// commit to name is a workspace that is not a repository, which is ordinary.
func TestAWorkspaceWithNoCommitStillRemembers(t *testing.T) {
	s, ws := setup(t)
	raw, _ := json.Marshal(map[string]any{"kind": "gotcha", "subject": "x", "body": "y"})
	res, err := Remember{Today: "2026-08-18"}.Execute(context.Background(), raw, s)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("%s", res.Output)
	}
	got, _ := memory.Read(ws)
	if len(got.Entries) != 1 {
		t.Fatalf("got %+v", got.Entries)
	}
	if got.Entries[0].Commit != "" {
		t.Errorf("invented a commit: %q", got.Entries[0].Commit)
	}
}
