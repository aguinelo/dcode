package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// edit refuses the wrong edit — unread file, ambiguous match, changed hash. It
// cannot refuse the right edit under a wrong decision: seven files changed
// competently and the third should never have been touched.
//
// Today the only way back is git, and only if you committed first.
func TestATurnCanBeUndone(t *testing.T) {
	s, ws := setup(t)
	path := writeFileT(t, ws, "stats.go", "package stats\n")

	s.BeginTurn()
	s.Snapshot(path)
	if err := os.WriteFile(path, []byte("package tally\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s.MarkRead(path, "package tally\n", 0)
	s.MarkWritten(path)

	done, refused, err := s.Undo()
	if err != nil {
		t.Fatal(err)
	}
	if len(refused) != 0 {
		t.Errorf("refused %v with nothing to refuse", refused)
	}
	if len(done) != 1 || done[0] != path {
		t.Errorf("restored %v", done)
	}
	if got := read(t, path); got != "package stats\n" {
		t.Errorf("the file is %q", got)
	}
}

// A file the turn created did not exist before, and putting the old content
// back means removing it.
func TestUndoingACreationRemovesTheFile(t *testing.T) {
	s, ws := setup(t)
	path := filepath.Join(ws, "new.go")

	s.BeginTurn()
	s.Snapshot(path)
	if err := os.WriteFile(path, []byte("package new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s.MarkRead(path, "package new\n", 0)
	s.MarkWritten(path)

	if _, _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("a file the turn created survived the undo: %v", err)
	}
}

// The one refusal that matters. Undoing after the person edited the file
// themselves would throw away their work to put back something older, which is
// the opposite of what undo is for.
func TestUndoRefusesAFileChangedSinceTheTurn(t *testing.T) {
	s, ws := setup(t)
	path := writeFileT(t, ws, "stats.go", "original\n")

	s.BeginTurn()
	s.Snapshot(path)
	if err := os.WriteFile(path, []byte("by the agent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s.MarkRead(path, "by the agent\n", 0)
	s.MarkWritten(path)

	// And then a person edits it.
	if err := os.WriteFile(path, []byte("by the person\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	done, refused, err := s.Undo()
	if err != nil {
		t.Fatal(err)
	}
	if len(done) != 0 {
		t.Errorf("restored %v over somebody's work", done)
	}
	if len(refused) != 1 || refused[0] != path {
		t.Errorf("refused %v", refused)
	}
	if got := read(t, path); got != "by the person\n" {
		t.Errorf("their work is gone: %q", got)
	}
}

// One snapshot per path per turn: the first is what the turn started from, and
// a later one would record a state the turn itself produced.
func TestOnlyTheFirstStateOfATurnIsKept(t *testing.T) {
	s, ws := setup(t)
	path := writeFileT(t, ws, "a.go", "first\n")

	s.BeginTurn()
	s.Snapshot(path)
	_ = os.WriteFile(path, []byte("second\n"), 0o644)
	s.Snapshot(path)
	_ = os.WriteFile(path, []byte("third\n"), 0o644)
	s.MarkRead(path, "third\n", 0)
	s.MarkWritten(path)

	if _, _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if got := read(t, path); got != "first\n" {
		t.Errorf("undo went back to %q, want what the turn started from", got)
	}
}

// A new turn is a new thing to undo. Keeping the old snapshots would let one
// undo reach back through work the person had already accepted.
func TestANewTurnReplacesWhatCanBeUndone(t *testing.T) {
	s, ws := setup(t)
	a := writeFileT(t, ws, "a.go", "a1\n")
	b := writeFileT(t, ws, "b.go", "b1\n")

	s.BeginTurn()
	s.Snapshot(a)
	_ = os.WriteFile(a, []byte("a2\n"), 0o644)
	s.MarkRead(a, "a2\n", 0)

	s.BeginTurn()
	s.Snapshot(b)
	_ = os.WriteFile(b, []byte("b2\n"), 0o644)
	s.MarkRead(b, "b2\n", 0)

	done, _, err := s.Undo()
	if err != nil {
		t.Fatal(err)
	}
	if len(done) != 1 || done[0] != b {
		t.Errorf("restored %v, want only the last turn's file", done)
	}
	if got := read(t, a); got != "a2\n" {
		t.Errorf("the earlier turn was reached back into: %q", got)
	}
}

// Nothing to undo is not an error, and must not report success either.
func TestUndoingATurnThatChangedNothing(t *testing.T) {
	s, _ := setup(t)
	s.BeginTurn()
	done, refused, err := s.Undo()
	if err != nil || len(done) != 0 || len(refused) != 0 {
		t.Errorf("got %v %v %v", done, refused, err)
	}
}

// A turn holds many cycles, and undoing at turn scope after one bad cycle
// would throw away every good cycle before it.
func TestUndoCycleLeavesTheEarlierCyclesAlone(t *testing.T) {
	s, ws := setup(t)
	good := writeFileT(t, ws, "good.go", "original\n")
	bad := filepath.Join(ws, "bad.go")
	s.BeginTurn()

	// Cycle one writes good.go and is fine.
	s.Snapshot(good)
	put(t, s, good, "cycle one\n")

	// Cycle two writes both, and regresses.
	s.BeginCycle()
	s.Snapshot(good)
	put(t, s, good, "cycle two\n")
	s.Snapshot(bad)
	put(t, s, bad, "created by cycle two\n")

	restored, refused, err := s.UndoCycle()
	if err != nil {
		t.Fatal(err)
	}
	if len(refused) != 0 {
		t.Errorf("nothing moved on disk and yet %v was refused", refused)
	}
	if body := read(t, good); body != "cycle one\n" {
		t.Errorf("the earlier cycle's work was thrown away: %q", body)
	}
	if _, err := os.Stat(bad); !os.IsNotExist(err) {
		t.Error("the file the bad cycle created is still there")
	}
	// Both: the bad cycle wrote both, so both go back — good.go to what cycle
	// one left, not to what the turn started from.
	if len(restored) != 2 {
		t.Errorf("restored %v, want the two the bad cycle wrote", restored)
	}
}

// The person's /undo still reaches the whole turn after a cycle was undone.
func TestUndoCycleKeepsTheTurnUndoable(t *testing.T) {
	s, ws := setup(t)
	f := writeFileT(t, ws, "f.go", "before the turn\n")
	s.BeginTurn()
	s.Snapshot(f)
	put(t, s, f, "cycle one\n")

	s.BeginCycle()
	other := filepath.Join(ws, "other.go")
	s.Snapshot(other)
	put(t, s, other, "cycle two\n")
	if _, _, err := s.UndoCycle(); err != nil {
		t.Fatal(err)
	}

	if _, _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if body := read(t, f); body != "before the turn\n" {
		t.Errorf("the turn is no longer undoable after a cycle was undone: %q", body)
	}
}

// A cycle boundary nobody marked is not a boundary, and guessing one would
// undo the whole turn under a name that promises less.
func TestUndoCycleWithoutABoundaryUndoesNothing(t *testing.T) {
	s, ws := setup(t)
	f := writeFileT(t, ws, "f.go", "start\n")
	s.BeginTurn()
	s.Snapshot(f)
	put(t, s, f, "written\n")

	restored, refused, err := s.UndoCycle()
	if err != nil || len(restored) != 0 || len(refused) != 0 {
		t.Errorf("undid %v / refused %v with no boundary marked", restored, refused)
	}
	if body := read(t, f); body != "written\n" {
		t.Errorf("a file was restored with no cycle boundary: %q", body)
	}
}

// put writes through the state the way a tool would, so Undo sees it as the
// turn's own work.
func put(t *testing.T, s *State, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	s.MarkRead(path, body, 0)
	s.MarkWritten(path)
}
