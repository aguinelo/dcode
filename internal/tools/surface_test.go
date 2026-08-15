package tools

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A tool's description and schema are what the model reads before deciding to
// call it, which makes them behaviour and not documentation. `process` had
// neither claimed by a test, so a schema that declared no identifier — leaving
// the tool uncallable for the one thing it is for — would have shipped.
func TestProcessTellsTheModelWhatItIsAndWhatItTakes(t *testing.T) {
	p := Process{}
	if p.Name() != "process" {
		t.Fatalf("name = %q", p.Name())
	}

	desc := strings.ToLower(p.Description())
	// The lifetime is the part a model gets wrong: a background process that
	// outlives its session is the guarantee this codebase decided to make, and
	// the description is where the model learns it.
	for _, want := range []string{"background", "stop", "session"} {
		if !strings.Contains(desc, want) {
			t.Errorf("the description never mentions %q: %s", want, desc)
		}
	}

	var schema struct {
		Type       string `json:"type"`
		Properties struct {
			ID   *struct{ Type string } `json:"id"`
			Stop *struct{ Type string } `json:"stop"`
		} `json:"properties"`
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(p.Schema(), &schema); err != nil {
		t.Fatalf("the schema is not valid JSON: %v", err)
	}
	if schema.Properties.ID == nil || schema.Properties.Stop == nil {
		t.Fatalf("the schema is missing id or stop: %s", p.Schema())
	}
	// Nothing is required: listing what is running is a call with no arguments,
	// and requiring an identifier would make that impossible to ask for.
	if len(schema.Required) != 0 {
		t.Errorf("required = %v, want nothing — listing takes no arguments", schema.Required)
	}
}

// Declaring reports nothing touched, so the verdict is always allow. Reading
// output from a process that already ran is not a second crossing of anything.
func TestReadingAProcessDeclaresNothing(t *testing.T) {
	req, err := (Process{}).Declare(json.RawMessage(`{"id":"p1","stop":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if req.Network || len(req.Paths) != 0 || req.Command != "" {
		t.Errorf("declared %+v, want nothing touched", req)
	}
}

// ---------- writing ----------

// Writing is atomic by default: a crash between truncating and writing leaves a
// half-written source file, which is worse than the original state. Both modes
// have to land the same content, and neither may leave the temp file behind.
func TestWritingLandsTheContentAndLeavesNoLitter(t *testing.T) {
	for _, atomic := range []bool{true, false} {
		dir := t.TempDir()
		// A directory that does not exist yet is created rather than refused:
		// writing a new file in a new package is ordinary work.
		path := filepath.Join(dir, "pkg", "a.txt")
		if err := writeFile(path, "hello", atomic); err != nil {
			t.Fatalf("atomic=%v: %v", atomic, err)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("atomic=%v: %v", atomic, err)
		}
		if string(b) != "hello" {
			t.Errorf("atomic=%v: file holds %q", atomic, b)
		}

		entries, err := os.ReadDir(filepath.Dir(path))
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 {
			t.Errorf("atomic=%v: the directory holds %d files, want only the one written", atomic, len(entries))
		}
	}
}

// Overwriting keeps the permissions the file is expected to have rather than
// inheriting whatever the temp file was created with — 0600 on a source file is
// a change nobody asked for.
func TestOverwritingLeavesReadablePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.txt")
	if err := writeFile(path, "one", true); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(path, "two", true); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o644 {
		t.Errorf("mode = %v, want 0644", fi.Mode().Perm())
	}
}

// A filesystem error carries the absolute path, and an absolute path in a tool
// result tells the model where the workspace sits on this machine. The meaning
// is kept and the path is replaced with the one the model already knows.
func TestAFilesystemErrorKeepsItsMeaningAndDropsTheAbsolutePath(t *testing.T) {
	err := scrub(&fs.PathError{
		Op: "open", Path: "/Users/somebody/private/ws/a.txt", Err: fs.ErrNotExist,
	}, "a.txt")

	msg := err.Error()
	if strings.Contains(msg, "/Users/somebody") {
		t.Errorf("the absolute path survived: %q", msg)
	}
	if !strings.Contains(msg, "a.txt") {
		t.Errorf("the workspace path was dropped too: %q", msg)
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the meaning was lost: %v", err)
	}

	// An error that is not a path error passes through untouched: inventing a
	// path for it would be worse than saying what actually happened.
	plain := errors.New("disk is full")
	if got := scrub(plain, "a.txt"); got != plain {
		t.Errorf("a plain error was rewritten to %v", got)
	}
}

// The line count reported next to an edit is what a person scans to see how big
// it was. Reporting growth as shrinkage, or the reverse, makes that number
// worse than absent.
func TestTheLineCountReportsTheDirectionTheEditWent(t *testing.T) {
	for _, tc := range []struct {
		name           string
		before, after  string
		added, removed int
	}{
		{"grew", "a\n", "a\nb\nc\n", 2, 0},
		{"shrank", "a\nb\nc\n", "a\n", 0, 2},
		{"same size", "a\nb\n", "x\ny\n", 0, 0},
		{"from nothing", "", "a\n", 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			added, removed := lineDelta(tc.before, tc.after)
			if added != tc.added || removed != tc.removed {
				t.Errorf("got +%d/-%d, want +%d/-%d", added, removed, tc.added, tc.removed)
			}
		})
	}
}

// ---------- undo, at its edges ----------

// A file the turn snapshotted but never actually wrote is left alone. A
// snapshot is taken before a write is attempted, and an attempt can fail — a
// denied path, an ambiguous match. Rewriting it would count work as undone that
// was never done.
func TestUndoLeavesAFileTheTurnNeverActuallyWrote(t *testing.T) {
	s, ws := setup(t)
	path := writeFileT(t, ws, "a.txt", "original")

	s.BeginTurn()
	s.Snapshot(path)
	// No MarkRead, no MarkWritten: the write never happened.

	restored, refused, err := s.Undo()
	if err != nil {
		t.Fatal(err)
	}
	if len(restored) != 0 || len(refused) != 0 {
		t.Errorf("restored %v, refused %v — want neither", restored, refused)
	}
}

// A file the turn created and somebody then deleted is already undone, and
// restoring over an absence clobbers nothing.
func TestUndoingACreationSomebodyAlreadyDeletedIsNotARefusal(t *testing.T) {
	s, ws := setup(t)
	path := filepath.Join(ws, "new.txt")

	s.BeginTurn()
	s.Snapshot(path) // Not there: undoing this means removing it.
	if err := os.WriteFile(path, []byte("made"), 0o600); err != nil {
		t.Fatal(err)
	}
	s.MarkRead(path, "made", 0)
	s.MarkWritten(path)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}

	restored, refused, err := s.Undo()
	if err != nil {
		t.Fatal(err)
	}
	if len(refused) != 0 {
		t.Errorf("refused %v — an absence cannot be clobbered", refused)
	}
	if len(restored) != 1 {
		t.Errorf("restored %v, want the creation reported as undone", restored)
	}
}

// A file changed on disk after the turn left it is refused, never overwritten.
// Undoing over somebody's own edit throws away their work to restore something
// older, which is the opposite of what undo is for.
func TestUndoRefusesAFileThePersonChangedThemselves(t *testing.T) {
	s, ws := setup(t)
	path := writeFileT(t, ws, "a.txt", "original")

	s.BeginTurn()
	s.Snapshot(path)
	if err := os.WriteFile(path, []byte("the turn wrote this"), 0o600); err != nil {
		t.Fatal(err)
	}
	s.MarkRead(path, "the turn wrote this", 0)
	s.MarkWritten(path)

	// And then a person edited it.
	if err := os.WriteFile(path, []byte("and then I did"), 0o600); err != nil {
		t.Fatal(err)
	}

	restored, refused, err := s.Undo()
	if err != nil {
		t.Fatal(err)
	}
	if len(restored) != 0 || len(refused) != 1 {
		t.Fatalf("restored %v, refused %v — want the changed file refused", restored, refused)
	}
	b, _ := os.ReadFile(path)
	if string(b) != "and then I did" {
		t.Errorf("the person's edit was overwritten with %q", b)
	}
}
