package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func recordWith(t *testing.T, dir, id string, lines ...string) string {
	t.Helper()
	path := filepath.Join(dir, id+".jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func seeded(t *testing.T) (dir, id string) {
	t.Helper()
	dir, id = t.TempDir(), "s1"
	recordWith(t, dir, id,
		`{"seq":1,"session_id":"s1","type":"session.created","at":"2026-08-21T00:00:00Z","payload":{"id":"s1","workspace":"/w","model":"m"}}`,
		`{"seq":2,"session_id":"s1","type":"turn.started","at":"2026-08-21T00:00:01Z","payload":{"turn_id":"t1","text":"catalogar os contratos"}}`,
		`{"seq":3,"session_id":"s1","type":"turn.completed","at":"2026-08-21T00:00:09Z","payload":{"turn_id":"t1","reason":"done"}}`,
	)
	return dir, id
}

func at(_ *testing.T) func() time.Time {
	return func() time.Time { return time.Unix(0, 0).UTC() }
}

// The name goes into the conversation's own record, so it dies with what it
// named. Pruning removes transcripts, and a name stored anywhere else would
// outlive the conversation — a listing full of titles nobody can open.
func TestANameIsStoredInTheConversationsOwnRecord(t *testing.T) {
	dir, id := seeded(t)
	if err := Rename(dir, id, "reformulação visual", at(t)); err != nil {
		t.Fatal(err)
	}

	found, err := Browse(dir, "/w")
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 {
		t.Fatalf("got %d sessions", len(found))
	}
	if found[0].Name != "reformulação visual" {
		t.Errorf("the name did not come back: %q", found[0].Name)
	}
	// The derived title survives beside it: they are different claims, and a
	// listing has to be able to say which it is showing.
	if found[0].Title != "catalogar os contratos" {
		t.Errorf("the derived title was lost: %q", found[0].Title)
	}
}

// The sequence is read rather than assumed. Appending a number already in the
// file would put a duplicate in a log whose whole contract is that there are
// none.
func TestARenameContinuesTheSequenceRatherThanGuessingIt(t *testing.T) {
	dir, id := seeded(t)
	if err := Rename(dir, id, "first", at(t)); err != nil {
		t.Fatal(err)
	}
	if err := Rename(dir, id, "second", at(t)); err != nil {
		t.Fatal(err)
	}

	body, err := os.ReadFile(filepath.Join(dir, id+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[uint64]bool{}
	var last uint64
	for _, line := range strings.Split(strings.TrimSpace(string(body)), "\n") {
		var ev struct {
			Seq uint64 `json:"seq"`
		}
		if json.Unmarshal([]byte(line), &ev) != nil {
			continue
		}
		if seen[ev.Seq] {
			t.Errorf("sequence %d appears twice", ev.Seq)
		}
		if ev.Seq != last+1 {
			t.Errorf("a gap: %d follows %d", ev.Seq, last)
		}
		seen[ev.Seq], last = true, ev.Seq
	}
}

// Renaming twice is somebody changing their mind, not two names.
func TestTheLastNameWins(t *testing.T) {
	dir, id := seeded(t)
	for _, n := range []string{"first", "second", "third"} {
		if err := Rename(dir, id, n, at(t)); err != nil {
			t.Fatal(err)
		}
	}
	found, _ := Browse(dir, "/w")
	if found[0].Name != "third" {
		t.Errorf("got %q", found[0].Name)
	}
}

// An empty name is the way back, not an error. One operation with a meaningful
// zero value is one thing to get right.
func TestAnEmptyNameRestoresTheDerivedTitle(t *testing.T) {
	dir, id := seeded(t)
	if err := Rename(dir, id, "given", at(t)); err != nil {
		t.Fatal(err)
	}
	if err := Rename(dir, id, "", at(t)); err != nil {
		t.Fatal(err)
	}
	found, _ := Browse(dir, "/w")
	if found[0].Name != "" {
		t.Errorf("the name survived being cleared: %q", found[0].Name)
	}
	if found[0].Title != "catalogar os contratos" {
		t.Errorf("the derived title is gone: %q", found[0].Title)
	}
}

// A newline inside a name would make one line of the record look like two, and
// the record is read back line by line.
func TestControlCharactersDoNotReachTheRecord(t *testing.T) {
	dir, id := seeded(t)
	if err := Rename(dir, id, "one\ntwo\ttab", at(t)); err != nil {
		t.Fatal(err)
	}
	found, _ := Browse(dir, "/w")
	if strings.ContainsAny(found[0].Name, "\n\t") {
		t.Errorf("a control character survived: %q", found[0].Name)
	}
	if found[0].Name != "onetwotab" {
		t.Errorf("got %q", found[0].Name)
	}
}

// Too long is refused rather than trimmed. Silently keeping half of what was
// typed is how somebody ends up with a name they did not choose.
func TestANameTooLongIsRefusedAndNotTrimmed(t *testing.T) {
	dir, id := seeded(t)
	long := strings.Repeat("a", NameLimit+1)
	if err := Rename(dir, id, long, at(t)); err == nil {
		t.Fatal("an over-long name was accepted")
	}
	found, _ := Browse(dir, "/w")
	if found[0].Name != "" {
		t.Errorf("a refused name reached the record: %q", found[0].Name)
	}
}

// Naming something that is not there says so, rather than creating a record for
// a conversation that never happened.
func TestNamingAConversationThatDoesNotExistSaysSo(t *testing.T) {
	dir := t.TempDir()
	if err := Rename(dir, "nope", "x", at(t)); err == nil {
		t.Fatal("naming a missing conversation succeeded")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("a record was created for a conversation that does not exist")
	}
}
