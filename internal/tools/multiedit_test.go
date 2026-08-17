package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// editCall runs the edit tool with whatever shape the argument describes.
func editCall(t *testing.T, s *State, args any) Result {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Edit{}.Execute(context.Background(), raw, s)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func readBack(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// seed writes a file and marks it read, which is what the invariant requires
// before any edit.
func seed(t *testing.T, s *State, dir, name, body string) string {
	t.Helper()
	p := writeFileT(t, dir, name, body)
	s.MarkRead(p, body, 0)
	return p
}

// A rename across several files is one act, and it has to land as one.
//
// Twelve separate calls is twelve round trips, and each can fail on its own —
// half a rename is worse than no rename, because the code no longer compiles
// and the reason is spread across a conversation.
func TestOneCallRenamesAcrossSeveralFiles(t *testing.T) {
	s, ws := setup(t)
	a := seed(t, s, ws, "a.go", "func Old() {}\nvar x = Old\n")
	b := seed(t, s, ws, "b.go", "call(Old)\n")

	res := editCall(t, s, map[string]any{
		"edits": []map[string]any{
			{"path": a, "old_string": "Old", "new_string": "New", "replace_all": true},
			{"path": b, "old_string": "Old", "new_string": "New"},
		},
	})
	if res.IsError {
		t.Fatalf("the rename failed: %s", res.Output)
	}
	if got := readBack(t, a); strings.Contains(got, "Old") {
		t.Errorf("a.go still holds the old name:\n%s", got)
	}
	if got := readBack(t, b); !strings.Contains(got, "New") {
		t.Errorf("b.go was not changed:\n%s", got)
	}
}

// Nothing is written until everything has been checked.
//
// This is the whole reason the batch exists. One bad edit in a set of five must
// leave all five files as they were, so the model can fix the one and try again
// against a state it still understands.
func TestOneBadEditLeavesEveryFileUntouched(t *testing.T) {
	s, ws := setup(t)
	a := seed(t, s, ws, "a.go", "func Old() {}\n")
	b := seed(t, s, ws, "b.go", "nothing to see\n")
	before := readBack(t, a)

	res := editCall(t, s, map[string]any{
		"edits": []map[string]any{
			{"path": a, "old_string": "Old", "new_string": "New"},
			{"path": b, "old_string": "absent", "new_string": "x"},
		},
	})
	if !res.IsError {
		t.Fatal("a batch with an unmatchable edit reported success")
	}
	if got := readBack(t, a); got != before {
		t.Errorf("the good edit landed anyway:\n%s", got)
	}
	// And it says which one, or the model has to guess among five.
	if !strings.Contains(res.Output, "b.go") {
		t.Errorf("the failure does not name the edit that failed: %s", res.Output)
	}
}

// An ambiguous match is refused for the batch exactly as it is for one edit.
// Picking the first occurrence is right most of the time and, when it is wrong,
// edits the wrong place silently.
func TestAnAmbiguousEditStopsTheWholeBatch(t *testing.T) {
	s, ws := setup(t)
	a := seed(t, s, ws, "a.go", "x := 1\nx := 2\n")
	before := readBack(t, a)

	res := editCall(t, s, map[string]any{
		"edits": []map[string]any{{"path": a, "old_string": "x :=", "new_string": "y :="}},
	})
	if !res.IsError {
		t.Fatal("an ambiguous edit was applied")
	}
	if got := readBack(t, a); got != before {
		t.Error("an ambiguous edit changed the file")
	}
}

// Edits to the same file apply in order, and each sees what the one before it
// left. Without that a batch can only ever make independent changes.
func TestEditsToOneFileChainInOrder(t *testing.T) {
	s, ws := setup(t)
	a := seed(t, s, ws, "a.go", "one\n")

	res := editCall(t, s, map[string]any{
		"edits": []map[string]any{
			{"path": a, "old_string": "one", "new_string": "two"},
			{"path": a, "old_string": "two", "new_string": "three"},
		},
	})
	if res.IsError {
		t.Fatalf("chained edits failed: %s", res.Output)
	}
	if got := readBack(t, a); strings.TrimSpace(got) != "three" {
		t.Errorf("got %q, want the second edit applied on top of the first", got)
	}
}

// The read-before-edit invariant holds for every file in the batch. A batch is
// not a way around the rule that stops a file being edited from assumed
// content.
func TestAFileNobodyReadStopsTheBatch(t *testing.T) {
	s, ws := setup(t)
	a := seed(t, s, ws, "a.go", "func Old() {}\n")
	unread := writeFileT(t, ws, "c.go", "func Old() {}\n")
	before := readBack(t, a)

	res := editCall(t, s, map[string]any{
		"edits": []map[string]any{
			{"path": a, "old_string": "Old", "new_string": "New"},
			{"path": unread, "old_string": "Old", "new_string": "New"},
		},
	})
	if !res.IsError {
		t.Fatal("a batch edited a file nobody had read")
	}
	if got := readBack(t, a); got != before {
		t.Error("the other file was written anyway")
	}
}

// The single-edit shape keeps working, unchanged. The model has learned it, and
// a batch that broke it would break every session already in flight.
func TestTheSingleEditShapeIsUnchanged(t *testing.T) {
	s, ws := setup(t)
	a := seed(t, s, ws, "a.go", "one\n")

	res := editCall(t, s, map[string]any{
		"path": a, "old_string": "one", "new_string": "two",
	})
	if res.IsError {
		t.Fatalf("the single-edit shape failed: %s", res.Output)
	}
	if got := readBack(t, a); strings.TrimSpace(got) != "two" {
		t.Errorf("got %q", got)
	}
}

// Every path in the batch is declared, because the policy decides on paths and
// a path it never saw is a path nobody was asked about.
func TestEveryPathInTheBatchIsDeclared(t *testing.T) {
	raw, err := json.Marshal(map[string]any{
		"edits": []map[string]any{
			{"path": "a.go", "old_string": "x", "new_string": "y"},
			{"path": "b.go", "old_string": "x", "new_string": "y"},
			{"path": "a.go", "old_string": "p", "new_string": "q"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	req, err := Edit{}.Declare(raw)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, a := range req.Paths {
		if !a.Write {
			t.Errorf("%s was declared without a write", a.Path)
		}
		seen[a.Path] = true
	}
	if !seen["a.go"] || !seen["b.go"] {
		t.Errorf("declared %v, want both files", req.Paths)
	}
	// The same file twice is one path to ask about, not two.
	if len(req.Paths) != 2 {
		t.Errorf("declared %d paths for two distinct files: %v", len(req.Paths), req.Paths)
	}
}

// An empty batch is a call that would do nothing, and saying so beats reporting
// a success that changed no file.
func TestAnEmptyBatchIsRefused(t *testing.T) {
	s, _ := setup(t)
	if res := editCall(t, s, map[string]any{"edits": []map[string]any{}}); !res.IsError {
		t.Error("an empty batch reported success")
	}
}

// The result says what happened to each file rather than a single total. Which
// file moved how far is the thing a person scans for.
func TestTheResultAccountsForEveryFile(t *testing.T) {
	s, ws := setup(t)
	a := seed(t, s, ws, "a.go", "one\n")
	b := seed(t, s, ws, filepath.Join("sub", "b.go"), "one\n")

	res := editCall(t, s, map[string]any{
		"edits": []map[string]any{
			{"path": a, "old_string": "one", "new_string": "two"},
			{"path": b, "old_string": "one", "new_string": "two"},
		},
	})
	if res.IsError {
		t.Fatalf("%s", res.Output)
	}
	for _, want := range []string{"a.go", "b.go"} {
		if !strings.Contains(res.Output, want) {
			t.Errorf("the result does not account for %s:\n%s", want, res.Output)
		}
	}
}

// Undo puts back every file the batch touched, not the last one. A batch that
// could not be undone as a unit would be a new way to lose work.
func TestUndoPutsBackEveryFileTheBatchTouched(t *testing.T) {
	s, ws := setup(t)
	s.BeginTurn()
	a := seed(t, s, ws, "a.go", "one\n")
	b := seed(t, s, ws, "b.go", "one\n")

	if res := editCall(t, s, map[string]any{
		"edits": []map[string]any{
			{"path": a, "old_string": "one", "new_string": "two"},
			{"path": b, "old_string": "one", "new_string": "two"},
		},
	}); res.IsError {
		t.Fatalf("%s", res.Output)
	}

	restored, refused, err := s.Undo()
	if err != nil {
		t.Fatal(err)
	}
	if len(refused) != 0 {
		t.Fatalf("refused %v", refused)
	}
	if len(restored) != 2 {
		t.Fatalf("restored %v, want both files", restored)
	}
	for _, p := range []string{a, b} {
		if got := readBack(t, p); strings.TrimSpace(got) != "one" {
			t.Errorf("%s holds %q after undo", p, got)
		}
	}
}

// A path outside the workspace stops the batch before anything is written, and
// says which edit asked for it. The containment rule is not something a batch
// can be used to get around.
func TestAPathOutsideTheWorkspaceStopsTheBatch(t *testing.T) {
	s, ws := setup(t)
	a := seed(t, s, ws, "a.go", "one\n")
	before := readBack(t, a)

	res := editCall(t, s, map[string]any{
		"edits": []map[string]any{
			{"path": a, "old_string": "one", "new_string": "two"},
			{"path": "../escape.go", "old_string": "one", "new_string": "two"},
		},
	})
	if !res.IsError {
		t.Fatal("a batch reached outside the workspace")
	}
	if got := readBack(t, a); got != before {
		t.Error("the other file was written anyway")
	}
	if !strings.Contains(res.Output, "edit 2 of 2") {
		t.Errorf("the refusal does not say which edit: %s", res.Output)
	}
}

// A file that is not there is refused with the same message a single edit
// gives, and takes the batch with it.
func TestAMissingFileStopsTheBatch(t *testing.T) {
	s, ws := setup(t)
	a := seed(t, s, ws, "a.go", "one\n")

	res := editCall(t, s, map[string]any{
		"edits": []map[string]any{
			{"path": a, "old_string": "one", "new_string": "two"},
			{"path": filepath.Join(ws, "gone.go"), "old_string": "one", "new_string": "two"},
		},
	})
	if !res.IsError {
		t.Fatal("a batch edited a file that does not exist")
	}
	if strings.TrimSpace(readBack(t, a)) != "one" {
		t.Error("the first file was written before the second was checked")
	}
}

// A file changed on disk after it was read stops the batch, for the same reason
// it stops a single edit: editing from assumed content is how files get
// corrupted.
func TestAFileThatMovedUnderTheBatchStopsIt(t *testing.T) {
	s, ws := setup(t)
	a := seed(t, s, ws, "a.go", "one\n")
	b := seed(t, s, ws, "b.go", "one\n")
	if err := os.WriteFile(b, []byte("somebody else wrote this\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	res := editCall(t, s, map[string]any{
		"edits": []map[string]any{
			{"path": a, "old_string": "one", "new_string": "two"},
			{"path": b, "old_string": "one", "new_string": "two"},
		},
	})
	if !res.IsError {
		t.Fatal("a batch edited a file that had changed under it")
	}
	if strings.TrimSpace(readBack(t, a)) != "one" {
		t.Error("the first file was written anyway")
	}
	if strings.TrimSpace(readBack(t, b)) != "somebody else wrote this" {
		t.Error("the other writer's work was overwritten")
	}
}

// A write that fails reports which file, so the person knows how far it got.
// This is the one failure a batch cannot prevent, and undo covers the turn.
func TestAFailedWriteNamesTheFile(t *testing.T) {
	s, ws := setup(t)
	a := seed(t, s, ws, "a.go", "one\n")
	// Make the file's directory unwritable so the atomic temp file cannot be
	// created. Restored by t.TempDir cleanup.
	dir := filepath.Dir(a)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Skipf("cannot make the directory read-only here: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	res := editCall(t, s, map[string]any{"path": a, "old_string": "one", "new_string": "two"})
	if !res.IsError {
		t.Skip("this filesystem allowed the write anyway")
	}
	if !strings.Contains(res.Output, "a.go") {
		t.Errorf("the failure does not name the file: %s", res.Output)
	}
}
