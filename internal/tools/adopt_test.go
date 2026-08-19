package tools

import (
	"os"
	"path/filepath"
	"testing"
)

// A delegated child's changes are undone by the turn that asked for them.
//
// Undo is per turn, and delegation happens inside one. A child keeps its own
// State, so without adoption the parent's undo would reach everything except
// the part it delegated — and undoing half of a division of work leaves a tree
// nobody designed.
func TestAdoptGivesTheParentWhatTheChildChanged(t *testing.T) {
	parent, ws := setup(t)
	child, _ := setup(t)
	child.Resolver = parent.Resolver

	path := filepath.Join(ws, "made.md")
	parent.BeginTurn()
	child.BeginTurn()
	child.Snapshot(path)
	if err := os.WriteFile(path, []byte("by the child"), 0o644); err != nil {
		t.Fatal(err)
	}
	child.MarkRead(path, "by the child", 0)
	child.MarkWritten(path)

	parent.Adopt(child)

	if got := parent.Written(); len(got) != 1 || got[0] != "made.md" {
		t.Fatalf("the parent does not know the child wrote: %v", got)
	}
	restored, refused, err := parent.Undo()
	if err != nil {
		t.Fatal(err)
	}
	if len(refused) != 0 {
		t.Fatalf("refused %v", refused)
	}
	if len(restored) != 1 {
		t.Fatalf("restored %v, want the child's file", restored)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("undoing a creation must remove it")
	}
}

// The first snapshot of the turn is what the turn started from, and the parent
// took it first. A child's later one records a state the turn itself produced.
func TestAdoptKeepsTheSnapshotTheParentTookFirst(t *testing.T) {
	parent, ws := setup(t)
	child, _ := setup(t)
	child.Resolver = parent.Resolver

	path := filepath.Join(ws, "shared.md")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	parent.BeginTurn()
	parent.Snapshot(path)
	if err := os.WriteFile(path, []byte("by the parent"), 0o644); err != nil {
		t.Fatal(err)
	}
	parent.MarkRead(path, "by the parent", 0)
	parent.MarkWritten(path)

	child.BeginTurn()
	child.Snapshot(path)
	if err := os.WriteFile(path, []byte("by the child"), 0o644); err != nil {
		t.Fatal(err)
	}
	child.MarkRead(path, "by the child", 0)
	child.MarkWritten(path)

	parent.Adopt(child)
	if _, _, err := parent.Undo(); err != nil {
		t.Fatal(err)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "original" {
		t.Errorf("undo restored %q; the turn started from \"original\"", body)
	}
}

// Adopting nothing is not an error, and not a panic. A child that wrote nothing
// is the ordinary case for a read-only one.
func TestAdoptingAChildThatChangedNothingIsQuiet(t *testing.T) {
	parent, _ := setup(t)
	child, _ := setup(t)
	parent.BeginTurn()
	parent.Adopt(child)
	if got := parent.Written(); len(got) != 0 {
		t.Errorf("nothing was written, got %v", got)
	}
	parent.Adopt(nil)
}
