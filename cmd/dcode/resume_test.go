package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// record writes a session record: one creation, then one completed turn per
// title. No titles is a session somebody opened and never asked anything in.
func record(t *testing.T, dir, id, at, ws string, titles ...string) {
	t.Helper()
	lines := `{"seq":1,"session_id":"` + id + `","type":"session.created",` +
		`"at":"` + at + `","payload":{"id":"` + id + `",` +
		`"workspace":"` + ws + `","model":"m"}}` + "\n"
	for _, title := range titles {
		lines += `{"seq":2,"session_id":"` + id + `","type":"turn.started",` +
			`"payload":{"text":"` + title + `"}}` + "\n"
		lines += `{"seq":3,"session_id":"` + id + `","type":"turn.completed",` +
			`"payload":{}}` + "\n"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, id+".jsonl"), []byte(lines), 0o600); err != nil {
		t.Fatal(err)
	}
}

// Continuing skips a session nobody asked anything in.
//
// Opening dcode records a session before the first question, so an interface
// closed without a question — or one that failed to open at all — leaves an
// empty record behind. Taking the newest without looking makes `-r` continue
// that emptiness, and the run of `-r` that did it leaves another one: the flag
// destroys its own target.
func TestContinuingSkipsASessionNobodyAskedAnythingIn(t *testing.T) {
	dir, ws := t.TempDir(), "/w"
	record(t, dir, "conversation", "2026-08-18T14:46:28-03:00", ws, "the thing I was doing")
	record(t, dir, "empty", "2026-08-18T14:47:12-03:00", ws)

	got, err := latestSession(dir, ws)
	if err != nil {
		t.Fatal(err)
	}
	if got != "conversation" {
		t.Errorf("continued %q, want the session that has a conversation", got)
	}
}

// With nothing but empty records, it says that rather than claiming there is
// no session here. The two are different problems and only one of them is the
// user's fault.
func TestNothingButEmptyRecordsSaysSo(t *testing.T) {
	dir, ws := t.TempDir(), "/w"
	record(t, dir, "empty", "2026-08-18T14:47:12-03:00", ws)

	_, err := latestSession(dir, ws)
	if err == nil {
		t.Fatal("continued an empty session")
	}
	if !strings.Contains(err.Error(), "nothing was asked") {
		t.Errorf("the refusal does not say why: %v", err)
	}
}

// A workspace nobody has used says so, and keeps saying so.
func TestAWorkspaceWithNoRecordSaysSo(t *testing.T) {
	if _, err := latestSession(t.TempDir(), "/w"); err == nil {
		t.Fatal("continued a session that does not exist")
	}
}
