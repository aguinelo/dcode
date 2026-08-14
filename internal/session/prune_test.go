package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeRecord(t *testing.T, dir, name string, size int, age time.Duration) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, make([]byte, size), 0o600); err != nil {
		t.Fatal(err)
	}
	when := time.Now().Add(-age)
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatal(err)
	}
	return path
}

func kept(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

// Age is the first budget, and on its own it is not enough: a busy fortnight
// fills a disk inside the window.
func TestOldRecordsAreDropped(t *testing.T) {
	dir := t.TempDir()
	writeRecord(t, dir, "old.jsonl", 10, 40*24*time.Hour)
	writeRecord(t, dir, "new.jsonl", 10, time.Hour)

	if _, err := Prune(dir, PruneBudget{MaxAge: 30 * 24 * time.Hour, KeepAtLeast: 0}); err != nil {
		t.Fatal(err)
	}
	if got := kept(t, dir); len(got) != 1 || got[0] != "new.jsonl" {
		t.Errorf("kept %v, want only new.jsonl", got)
	}
}

// Size is the second, and on its own it deletes this morning's work on a busy
// day. Both have to hold.
func TestTheOldestGoFirstWhenTheBudgetIsExceeded(t *testing.T) {
	dir := t.TempDir()
	writeRecord(t, dir, "a.jsonl", 100, 3*time.Hour)
	writeRecord(t, dir, "b.jsonl", 100, 2*time.Hour)
	writeRecord(t, dir, "c.jsonl", 100, time.Hour)

	if _, err := Prune(dir, PruneBudget{MaxBytes: 250, KeepAtLeast: 0}); err != nil {
		t.Fatal(err)
	}
	got := kept(t, dir)
	if len(got) != 2 {
		t.Fatalf("kept %v, want the two newest", got)
	}
	for _, n := range got {
		if n == "a.jsonl" {
			t.Error("the oldest survived while newer ones were considered")
		}
	}
}

// The floor beats both budgets. Someone who used dcode twice last year should
// still find those two sessions, and a rule that empties the directory the
// first time it runs is a rule nobody trusts again.
func TestTheMostRecentAreKeptWhateverTheBudgetSays(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 5; i++ {
		writeRecord(t, dir, string(rune('a'+i))+".jsonl", 1000, time.Duration(400+i)*24*time.Hour)
	}

	if _, err := Prune(dir, PruneBudget{MaxAge: time.Hour, MaxBytes: 1, KeepAtLeast: 3}); err != nil {
		t.Fatal(err)
	}
	if got := kept(t, dir); len(got) != 3 {
		t.Errorf("kept %v, want the three most recent", got)
	}
}

// A session being written is not garbage, however old its first line is.
func TestALiveSessionIsNeverPruned(t *testing.T) {
	dir := t.TempDir()
	writeRecord(t, dir, "live.jsonl", 10, 400*24*time.Hour)
	writeRecord(t, dir, "dead.jsonl", 10, 400*24*time.Hour)

	if _, err := Prune(dir, PruneBudget{MaxAge: time.Hour, KeepAtLeast: 0, Live: map[string]bool{"live": true}}); err != nil {
		t.Fatal(err)
	}
	got := kept(t, dir)
	if len(got) != 1 || got[0] != "live.jsonl" {
		t.Errorf("kept %v, wanted the open session and nothing else", got)
	}
}

// It reports what it removed. A cleanup nobody can see is one that gets blamed
// for every missing file.
func TestPruneReportsWhatItRemoved(t *testing.T) {
	dir := t.TempDir()
	writeRecord(t, dir, "old.jsonl", 42, 400*24*time.Hour)

	n, err := Prune(dir, PruneBudget{MaxAge: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("reported %d removed, want 1", n)
	}
}

// Nothing configured means nothing removed. A zero budget is "no policy", not
// "delete everything", because the second reading loses data on a typo.
func TestAnEmptyBudgetRemovesNothing(t *testing.T) {
	dir := t.TempDir()
	writeRecord(t, dir, "old.jsonl", 10, 400*24*time.Hour)

	if n, err := Prune(dir, PruneBudget{}); err != nil || n != 0 {
		t.Errorf("removed %d with no policy set (err %v)", n, err)
	}
	if len(kept(t, dir)) != 1 {
		t.Error("a zero budget emptied the directory")
	}
}

// A directory that is not there yet is not an error: the first run of a fresh
// install prunes nothing and says so quietly.
func TestPruningWhatDoesNotExistIsQuiet(t *testing.T) {
	if n, err := Prune(filepath.Join(t.TempDir(), "absent"), PruneBudget{MaxAge: time.Hour}); err != nil || n != 0 {
		t.Errorf("n=%d err=%v", n, err)
	}
}
