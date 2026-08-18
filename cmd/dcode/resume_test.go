package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/session"
	"github.com/aguinelo/dcode/internal/tui"
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

// The list to pick from holds conversations, not the empty records that opening
// the interface leaves behind. Offering those would bury the few real ones.
func TestThePickListLeavesOutSessionsNobodyAskedAnythingIn(t *testing.T) {
	dir, ws := t.TempDir(), "/w"
	record(t, dir, "real", "2026-08-18T14:46:28-03:00", ws, "the thing I was doing")
	record(t, dir, "empty-one", "2026-08-18T14:47:12-03:00", ws)
	record(t, dir, "empty-two", "2026-08-18T14:47:53-03:00", ws)

	found, err := session.Browse(dir, ws)
	if err != nil {
		t.Fatal(err)
	}
	got := choicesFrom(found)
	if len(got) != 1 {
		t.Fatalf("the list offers %d sessions, want only the one with a conversation", len(got))
	}
	if got[0].ID != "real" || got[0].Title != "the thing I was doing" {
		t.Errorf("the row is %+v", got[0])
	}
}

// The two flags mean different things, and both open the interface.
//
// `-c` takes the last conversation; `-r` asks which. Asking is the safer
// default for a workspace somebody has used all week; taking the last is the
// faster one for the session they just closed. Collapsing them into one flag is
// what made `-r` mean neither.
func TestResumeAsksAndContinueTakesTheLast(t *testing.T) {
	for _, arg := range []string{"-c", "--continue", "-r", "--resume"} {
		if !opensTheInterface([]string{arg}) {
			t.Errorf("%s did not open the interface", arg)
		}
	}
	if opensTheInterface([]string{"-r is not a flag here"}) {
		t.Error("a task that starts with a dash was read as a flag")
	}
}

// The geometry is read at the edge, so the picker and the interface get the
// same one. Two readings would draw two different windows for one terminal.
func TestTheGeometryIsBuiltOnceForBothScreens(t *testing.T) {
	if g := geometry(false); g.Width <= 0 || g.Height <= 0 {
		t.Fatalf("the geometry is %dx%d", g.Width, g.Height)
	}
	if g := geometry(true); g.PanelMode != tui.PanelHidden {
		t.Errorf("--no-panel did not hide the panel: %v", g.PanelMode)
	}
}

// The language comes from the configuration when it is set there, and from the
// environment otherwise. The client never reads either itself.
func TestTheLanguageIsResolvedAtTheEdge(t *testing.T) {
	t.Setenv("DCODE_LANG", "en")
	if got := langOf(config.Resolved{}); got != tui.En {
		t.Errorf("the environment gave %q", got)
	}
}
