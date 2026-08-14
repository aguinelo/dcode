package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

func recordFile(t *testing.T, dir, id string, events ...protocol.Event) {
	t.Helper()
	var b strings.Builder
	for _, ev := range events {
		line, err := json.Marshal(ev)
		if err != nil {
			t.Fatal(err)
		}
		b.Write(line)
		b.WriteString("\n")
	}
	if err := os.WriteFile(filepath.Join(dir, id+".jsonl"), []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
}

func event(t *testing.T, seq uint64, id string, kind protocol.EventType, payload any) protocol.Event {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return protocol.Event{Seq: seq, SessionID: id, Type: kind, At: time.Unix(int64(seq), 0), Payload: raw}
}

func session(t *testing.T, dir, id, workspace, question string) {
	t.Helper()
	recordFile(t, dir, id,
		event(t, 1, id, protocol.EventSessionCreated, protocol.Session{ID: id, Workspace: workspace, Model: "MiniMax-M3"}),
		event(t, 2, id, protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1", Text: question}),
		event(t, 3, id, protocol.EventTurnCompleted, protocol.TurnCompleted{TurnID: "t1"}),
	)
}

// A list of hex ids is a list nobody picks from. The title is what makes the
// listing usable at all, and it comes from the first thing the person asked.
func TestASessionIsTitledByWhatWasAsked(t *testing.T) {
	dir := t.TempDir()
	session(t, dir, "s1", "/w", "rename Summary to Tally everywhere")

	got, err := Browse(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("found %d sessions", len(got))
	}
	if got[0].Title != "rename Summary to Tally everywhere" {
		t.Errorf("title is %q", got[0].Title)
	}
	if got[0].ID != "s1" || got[0].Workspace != "/w" {
		t.Errorf("got %+v", got[0])
	}
	if got[0].Turns != 1 {
		t.Errorf("counted %d turns", got[0].Turns)
	}
}

// "What was I doing here" is the question being asked almost every time.
func TestBrowsingFiltersByWorkspace(t *testing.T) {
	dir := t.TempDir()
	session(t, dir, "here", "/here", "one")
	session(t, dir, "elsewhere", "/elsewhere", "two")

	got, err := Browse(dir, "/here")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "here" {
		t.Errorf("got %+v", got)
	}

	all, err := Browse(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Errorf("unfiltered found %d", len(all))
	}
}

// Newest first: the one you want is almost always the last one you had.
func TestBrowsingPutsTheNewestFirst(t *testing.T) {
	dir := t.TempDir()
	session(t, dir, "older", "/w", "first")
	session(t, dir, "newer", "/w", "second")
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(filepath.Join(dir, "older.jsonl"), old, old); err != nil {
		t.Fatal(err)
	}

	got, err := Browse(dir, "/w")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "newer" {
		t.Errorf("order is %+v", got)
	}
}

// A record being written has no completed turn and possibly no question yet.
// It is still a session, and hiding it would hide the one you are in.
func TestASessionWithNoTurnYetStillAppears(t *testing.T) {
	dir := t.TempDir()
	recordFile(t, dir, "fresh",
		event(t, 1, "fresh", protocol.EventSessionCreated, protocol.Session{ID: "fresh", Workspace: "/w"}))

	got, err := Browse(dir, "/w")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("found %d", len(got))
	}
	if got[0].Title == "" {
		t.Error("a session with nothing asked yet still needs something to show")
	}
}

// A file that is not a record, or a line that is not an event, must not take
// the listing down with it.
func TestRubbishInTheDirectoryIsSkipped(t *testing.T) {
	dir := t.TempDir()
	session(t, dir, "good", "/w", "fine")
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "broken.jsonl"), []byte("{not json\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := Browse(dir, "")
	if err != nil {
		t.Fatalf("one bad file took the whole listing down: %v", err)
	}
	if len(got) != 1 || got[0].ID != "good" {
		t.Errorf("got %+v", got)
	}
}

func TestBrowsingSomewhereThatIsNotThereIsQuiet(t *testing.T) {
	got, err := Browse(filepath.Join(t.TempDir(), "absent"), "")
	if err != nil || len(got) != 0 {
		t.Errorf("got %v, %v", got, err)
	}
}

// Reading a session back is the audit case, and it is pure reading: no model,
// no reconstruction, no cost. It is why this comes before continuing.
func TestATranscriptReadsLikeTheConversation(t *testing.T) {
	dir := t.TempDir()
	recordFile(t, dir, "s1",
		event(t, 1, "s1", protocol.EventSessionCreated, protocol.Session{ID: "s1", Workspace: "/w", Model: "MiniMax-M3"}),
		event(t, 2, "s1", protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1", Text: "what does Rows do?"}),
		event(t, 3, "s1", protocol.EventToolRequested, protocol.ToolRequested{TurnID: "t1", ToolCallID: "c1", Name: "read", Input: json.RawMessage(`{"path":"stats.go"}`)}),
		event(t, 4, "s1", protocol.EventToolCompleted, protocol.ToolCompleted{ToolCallID: "c1", OK: true, Lines: 40}),
		event(t, 5, "s1", protocol.EventMessageDelta, protocol.MessageDelta{TurnID: "t1", Text: "It returns "}),
		event(t, 6, "s1", protocol.EventMessageDelta, protocol.MessageDelta{TurnID: "t1", Text: "count minus one."}),
		event(t, 7, "s1", protocol.EventTurnCompleted, protocol.TurnCompleted{TurnID: "t1"}),
	)

	got, err := Transcript(filepath.Join(dir, "s1.jsonl"))
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"what does Rows do?", "read", "stats.go", "It returns count minus one."} {
		if !strings.Contains(got, want) {
			t.Errorf("the transcript does not carry %q:\n%s", want, got)
		}
	}
	// The deltas are one answer, not sixteen fragments on sixteen lines.
	if strings.Count(got, "It returns") != 1 {
		t.Errorf("the answer was not joined:\n%s", got)
	}
}

// A record cut off mid-turn — the process died — still reads. Refusing to show
// it is refusing exactly when someone most wants to look.
func TestAnUnfinishedTranscriptStillReads(t *testing.T) {
	dir := t.TempDir()
	recordFile(t, dir, "s2",
		event(t, 1, "s2", protocol.EventSessionCreated, protocol.Session{ID: "s2", Workspace: "/w"}),
		event(t, 2, "s2", protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1", Text: "start something long"}),
		event(t, 3, "s2", protocol.EventMessageDelta, protocol.MessageDelta{TurnID: "t1", Text: "working on"}),
	)

	got, err := Transcript(filepath.Join(dir, "s2.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "start something long") || !strings.Contains(got, "working on") {
		t.Errorf("an interrupted session lost its content:\n%s", got)
	}
}

// The answer needs air around it, or it runs straight on from the last tool
// line and the two read as one thing.
func TestTheAnswerIsSetApartFromTheToolLines(t *testing.T) {
	dir := t.TempDir()
	recordFile(t, dir, "s3",
		event(t, 1, "s3", protocol.EventSessionCreated, protocol.Session{ID: "s3", Workspace: "/w"}),
		event(t, 2, "s3", protocol.EventToolRequested, protocol.ToolRequested{ToolCallID: "c1", Name: "read"}),
		event(t, 3, "s3", protocol.EventMessageDelta, protocol.MessageDelta{Text: "the answer"}),
	)
	got, err := Transcript(filepath.Join(dir, "s3.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "\n\nthe answer") {
		t.Errorf("the answer is not set apart:\n%q", got)
	}
}

// Compaction is part of the story: a transcript that silently skips it reads
// as a conversation with a hole nobody can see.
func TestCompactionIsVisibleInTheTranscript(t *testing.T) {
	dir := t.TempDir()
	recordFile(t, dir, "s4",
		event(t, 1, "s4", protocol.EventSessionCreated, protocol.Session{ID: "s4", Workspace: "/w"}),
		event(t, 2, "s4", protocol.EventSessionCompacted, map[string]int{"kept": 4}),
	)
	got, err := Transcript(filepath.Join(dir, "s4.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "summarised") {
		t.Errorf("compaction left no trace:\n%s", got)
	}
}

// A refused tool call is the part of a session most worth reading later.
func TestAFailedCallAndAnApprovalAreShown(t *testing.T) {
	dir := t.TempDir()
	recordFile(t, dir, "s5",
		event(t, 1, "s5", protocol.EventSessionCreated, protocol.Session{ID: "s5", Workspace: "/w"}),
		event(t, 2, "s5", protocol.EventApprovalResolved, protocol.ApprovalResolved{ApprovalID: "a1", Decision: protocol.ApprovalDeny}),
		event(t, 3, "s5", protocol.EventToolCompleted, protocol.ToolCompleted{ToolCallID: "c1", OK: false, Output: "denied by the user"}),
	)
	got, err := Transcript(filepath.Join(dir, "s5.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "deny") || !strings.Contains(got, "denied by the user") {
		t.Errorf("the refusal is not in the transcript:\n%s", got)
	}
}

func TestReadingASessionThatIsNotThereSaysSo(t *testing.T) {
	if _, err := Transcript(filepath.Join(t.TempDir(), "absent.jsonl")); !os.IsNotExist(err) {
		t.Errorf("got %v, want a not-exist error the caller can recognise", err)
	}
}

func TestBrowsingWithNoDirectoryConfiguredIsQuiet(t *testing.T) {
	got, err := Browse("", "")
	if err != nil || got != nil {
		t.Errorf("got %v, %v", got, err)
	}
}
