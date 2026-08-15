package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/tools"
)

func clockAt(t time.Time) Clock { return func() time.Time { return t } }

// Undo reaches the files through the engine that owns the state. The refusal
// path is already claimed; this is the path where it works, which is the one
// the person actually asks for.
func TestUndoingASettledSessionPutsTheFilesBack(t *testing.T) {
	r, err := policy.NewResolver(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(r.Workspace, "a.txt")
	if err := os.WriteFile(path, []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}

	st := tools.NewState(r, tools.DefaultLimits(), nil)
	st.BeginTurn()
	st.Snapshot(path)
	if err := os.WriteFile(path, []byte("after"), 0o600); err != nil {
		t.Fatal(err)
	}
	st.MarkRead(path, "after", 0)
	st.MarkWritten(path)

	e := loop.New(loop.Config{State: st}, ce.Session{Instructions: "x"})
	s := New("s1", r.Workspace, "m", "workspace-write", e,
		NewEventLog("s1", 10, nil), clockAt(time.Unix(0, 0)))

	got, err := s.Undo()
	if err != nil {
		t.Fatalf("undo failed: %v", err)
	}
	if len(got.Restored) != 1 || len(got.Refused) != 0 {
		t.Fatalf("restored %v, refused %v — want the one file back", got.Restored, got.Refused)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "before" {
		t.Errorf("the file holds %q, want what it held before the turn", b)
	}
}

// A session with no engine has nothing to undo and answers emptily. The client
// asks the same way, and gets an answer rather than an error about internals.
func TestUndoingASessionWithNoEngineIsAnEmptyAnswer(t *testing.T) {
	s := New("s1", "/w", "m", "read-only", nil, NewEventLog("s1", 10, nil), clockAt(time.Unix(0, 0)))
	got, err := s.Undo()
	if err != nil {
		t.Fatalf("undo failed: %v", err)
	}
	if len(got.Restored) != 0 || len(got.Refused) != 0 {
		t.Errorf("got %+v, want nothing", got)
	}
}

// Pruning needs to know which records are still being written, and deciding
// that from the file alone would mean guessing from a timestamp that changes as
// you read it. LiveIDs is the answer, and nothing claimed it.
func TestLiveIDsNamesEverySessionTheManagerHolds(t *testing.T) {
	m := NewManager(4)
	for _, id := range []string{"a", "b"} {
		if err := m.Add(New(id, "/w", "m", "read-only", nil, NewEventLog(id, 4, nil), clockAt(time.Unix(0, 0)))); err != nil {
			t.Fatal(err)
		}
	}

	live := m.LiveIDs()
	if len(live) != 2 || !live["a"] || !live["b"] {
		t.Fatalf("live = %v, want both sessions", live)
	}

	if err := m.Remove("a"); err != nil {
		t.Fatal(err)
	}
	if live := m.LiveIDs(); len(live) != 1 || live["a"] {
		t.Errorf("live = %v after removing a", live)
	}

	// The map is a copy: a caller that edits it must not edit the manager.
	live["c"] = true
	if again := m.LiveIDs(); again["c"] {
		t.Error("editing the returned map changed what the manager holds")
	}
}

// A title is what a person picks a session by in the list, so it has to fit a
// column and it has to be the first thing they asked, not the whole paragraph.
func TestATitleIsOneLineAndFitsAColumn(t *testing.T) {
	for _, tc := range []struct {
		name, in, want string
	}{
		{"trimmed", "  hello  ", "hello"},
		{"first line only", "add the flag\n\nand then the test", "add the flag"},
		{"already short", "hi", "hi"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := firstLineOf(tc.in); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}

	long := strings.Repeat("a", titleLimit+40)
	got := firstLineOf(long)
	if !strings.HasSuffix(got, "…") {
		t.Errorf("a title over the limit was not marked as cut: %q", got)
	}
	if n := len([]rune(got)); n != titleLimit+1 {
		t.Errorf("the cut title is %d runes, want %d plus the ellipsis", n, titleLimit)
	}
}

// Non-ASCII must be cut by rune, not by byte: cutting mid-character puts a
// replacement glyph in the list, which reads as corruption rather than as a
// long title.
func TestATitleIsCutByCharacterNotByByte(t *testing.T) {
	got := firstLineOf(strings.Repeat("ç", titleLimit+10))
	if strings.Contains(got, "�") {
		t.Errorf("the title was cut mid-character: %q", got)
	}
}

// Replaying a record that is not there is not an error: recording can be off,
// or the record can have been pruned, and neither is a failure of the ask.
func TestReplayingARecordThatIsNotThereAnswersEmpty(t *testing.T) {
	r := &Record{path: filepath.Join(t.TempDir(), "gone.jsonl")}
	got, err := r.Replay(1)
	if err != nil {
		t.Fatalf("replaying a missing record failed: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d events from a record that does not exist", len(got))
	}
}

// A file in the record directory that is not a record — anything a person or
// another tool dropped there — is skipped rather than listed as a session with
// no title and no turns.
func TestSomethingThatIsNotARecordIsNotListedAsASession(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.jsonl"), []byte("{\"seq\":1}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	rec, err := NewRecord(dir, "real")
	if err != nil {
		t.Fatal(err)
	}
	if err := rec.Append([]protocol.Event{{
		Seq: 1, Type: protocol.EventSessionCreated,
		Payload: []byte(`{"id":"real","workspace":"/w","model":"m"}`),
	}}); err != nil {
		t.Fatal(err)
	}
	if err := rec.Close(); err != nil {
		t.Fatal(err)
	}

	found, err := Browse(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].ID != "real" {
		t.Fatalf("browse returned %+v, want only the actual record", found)
	}
}
