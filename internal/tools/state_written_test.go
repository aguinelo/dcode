package tools

import (
	"reflect"
	"testing"
)

// The definition of done needs to know whether this session changed anything.
// MarkRead cannot answer it: write and edit both call MarkRead immediately
// afterwards, to keep the next edit from failing as file_changed.
func TestWrittenReportsWhatTheSessionChanged(t *testing.T) {
	s, ws := setup(t)
	writeFileT(t, ws, "a.go", "one\n")
	writeFileT(t, ws, "b.go", "two\n")

	run(t, Write{}, s, WriteInput{Path: "new.go", Content: "x\n"})
	if got := s.Written(); !reflect.DeepEqual(got, []string{"new.go"}) {
		t.Fatalf("after a write, Written() = %v", got)
	}

	before := "one\n"
	s.MarkRead(writeFileT(t, ws, "a.go", before), before, 0)
	run(t, Edit{}, s, EditInput{Path: "a.go", OldString: "one", NewString: "ONE"})
	if got := s.Written(); !reflect.DeepEqual(got, []string{"a.go", "new.go"}) {
		t.Fatalf("after an edit, Written() = %v", got)
	}
}

func TestReadingAloneWritesNothing(t *testing.T) {
	s, ws := setup(t)
	writeFileT(t, ws, "a.go", "one\n")
	run(t, Read{}, s, ReadInput{Path: "a.go"})
	if got := s.Written(); len(got) != 0 {
		t.Fatalf("reading reported writes: %v — a read-only turn would run the verification", got)
	}
}

// The list a delegated turn hands back beside its conclusion.
func TestReadPathsReportsWhatWasOpened(t *testing.T) {
	s, ws := setup(t)
	writeFileT(t, ws, "a.go", "one\n")
	writeFileT(t, ws, "b/c.go", "two\n")

	if got := s.ReadPaths(); len(got) != 0 {
		t.Fatalf("a fresh session reported %v", got)
	}
	run(t, Read{}, s, ReadInput{Path: "a.go"})
	run(t, Read{}, s, ReadInput{Path: "b/c.go"})

	got := s.ReadPaths()
	if len(got) != 2 || got[0] != "a.go" || got[1] != "b/c.go" {
		t.Fatalf("read paths = %v, want both, workspace-relative and sorted", got)
	}
}

// The fact behind the file-changed reminder.
func TestChangedSinceReadNoticesAFileThatMovedUnderneath(t *testing.T) {
	s, ws := setup(t)
	p := writeFileT(t, ws, "a.go", "one\n")
	run(t, Read{}, s, ReadInput{Path: "a.go"})

	current := map[string]string{p: "one\n"}
	read := func(path string) (string, error) { return current[path], nil }

	if got := s.ChangedSinceRead(read); len(got) != 0 {
		t.Fatalf("nothing changed and it reported %v", got)
	}
	current[p] = "one\ntwo\n"
	got := s.ChangedSinceRead(read)
	if len(got) != 1 {
		t.Fatalf("a file changed on disk and it reported %v", got)
	}
}
