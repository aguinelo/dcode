package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// A turn whose context is already gone is not submitted: a stopped loop must
// not leave a session working behind it.
func TestATurnIsNotSubmittedOnACancelledContext(t *testing.T) {
	srv, _ := newServer(t, 4)
	sl := serverLoops{s: srv, loopID: "L1"}
	s, err := sl.Open(context.Background(), protocol.CreateSessionRequest{Workspace: "/w"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := sl.RunTurn(ctx, s.ID, "x"); !errors.Is(err, context.Canceled) {
		t.Errorf("a turn on a cancelled context gave %v, want it never submitted", err)
	}
}

// The stream misses no change: whatever happens while a client subscribes
// arrives either in its snapshot or after it — so a run's end is always seen.
func TestTheLoopStreamMissesNoChange(t *testing.T) {
	for i := 0; i < 300; i++ {
		h := &heldDaemon{found: []protocol.SpecFolder{{Path: "specs/a", Criteria: 1}}, release: make(chan struct{})}
		srv := loopServer(t, h)
		ws := t.TempDir()
		close(h.release)
		run, err := srv.loops.start(protocol.StartLoopRequest{Workspace: ws, Args: "specs/a"})
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		snapshot, changes := srv.loops.subscribe(ctx)
		last := ""
		for _, r := range snapshot {
			if r.ID == run.ID {
				last = r.State
			}
		}
		deadline := time.After(2 * time.Second)
	read:
		for last != protocol.LoopFinished {
			select {
			case ch := <-changes:
				if ch.Loop != nil && ch.Loop.ID == run.ID {
					last = ch.Loop.State
				}
			case <-deadline:
				break read
			}
		}
		cancel()
		if last != protocol.LoopFinished {
			t.Fatalf("run %d: a subscriber was left at %q, never told the run finished", i, last)
		}
	}
}

// A loop's workspace is a folder that exists, read through its links, and no
// loop starts inside or around the tree of a live one.
func TestALoopNeedsAFolderOfItsOwn(t *testing.T) {
	h := &heldDaemon{found: []protocol.SpecFolder{{Path: "specs/a", Criteria: 1}}, release: make(chan struct{})}
	srv := loopServer(t, h)
	root := t.TempDir()
	ws := filepath.Join(root, "proj")
	sub := filepath.Join(ws, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "a-file")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(ws, link); err != nil {
		t.Fatal(err)
	}

	for _, bad := range []string{filepath.Join(root, "missing"), file} {
		if _, err := srv.loops.start(protocol.StartLoopRequest{Workspace: bad, Args: "specs/a"}); !hasCode(err, protocol.CodeWorkspaceInvalid) {
			t.Errorf("%s gave %v, want workspace_invalid", bad, err)
		}
	}
	if _, err := srv.loops.start(protocol.StartLoopRequest{Workspace: ws, Args: "specs/a"}); err != nil {
		t.Fatal(err)
	}
	for _, overlapping := range []string{sub, root, link} {
		if _, err := srv.loops.start(protocol.StartLoopRequest{Workspace: overlapping, Args: "specs/a"}); !hasCode(err, protocol.CodeLoopAlreadyRunning) {
			t.Errorf("%s gave %v, want loop_already_running", overlapping, err)
		}
	}
	close(h.release)
}

// No loop starts on a daemon going away: it would run against a context that
// is already over, and nothing would wait for it.
func TestNoLoopStartsOnADaemonGoingAway(t *testing.T) {
	h := &heldDaemon{release: make(chan struct{})}
	srv := loopServer(t, h)
	srv.loops.close()
	if _, err := srv.loops.start(protocol.StartLoopRequest{Workspace: t.TempDir(), Args: "specs/a"}); err == nil ||
		!strings.Contains(err.Error(), "shutting down") {
		t.Errorf("a loop on a closed daemon gave %v", err)
	}
}

// An id that cannot be drawn is an error, never a run named by zeroes.
func TestALoopIDThatCannotBeDrawnIsAnError(t *testing.T) {
	h := &heldDaemon{release: make(chan struct{})}
	srv := loopServer(t, h)
	old := randRead
	randRead = func([]byte) (int, error) { return 0, errors.New("no entropy") }
	defer func() { randRead = old }()
	if _, err := srv.loops.start(protocol.StartLoopRequest{Workspace: t.TempDir(), Args: "specs/a"}); err == nil ||
		!strings.Contains(err.Error(), "no entropy") {
		t.Errorf("a failed draw gave %v", err)
	}
}

// A subscriber that stopped reading is dropped, and the drop is said.
func TestASlowLoopSubscriberIsDroppedAndSaid(t *testing.T) {
	h := &heldDaemon{found: []protocol.SpecFolder{{Path: "specs/a", Criteria: 1}}, release: make(chan struct{})}
	srv := loopServer(t, h)
	var mu sync.Mutex
	var said []string
	srv.loops.log = func(s string) {
		mu.Lock()
		said = append(said, s)
		mu.Unlock()
	}
	run, err := srv.loops.start(protocol.StartLoopRequest{Workspace: t.TempDir(), Args: "specs/a"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv.loops.subscribe(ctx) // never read
	for i := 0; i < 200; i++ {
		srv.loops.changed(run.ID)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(said) == 0 || !strings.Contains(strings.Join(said, "\n"), "dropped") {
		t.Errorf("a dropped subscriber was not said: %q", said)
	}
	close(h.release)
}

// Nesting is nesting at the root too: a loop on "/" holds every tree.
func TestOverlapsHoldsAtTheRoot(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"/", "/a", true},
		{"/a", "/", true},
		{"/", "/", true},
		{"/a", "/a/b", true},
		{"/a/b", "/a", true},
		{"/a", "/ab", false},
		{"/a/b", "/a/c", false},
	}
	for _, c := range cases {
		if got := overlaps(c.a, c.b); got != c.want {
			t.Errorf("overlaps(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

// The list keeps a workspace's loops by the rule that refuses a second one:
// from a folder inside the loop's tree, around it, or through a link to it, a
// client finds the loop that blocks it. It kept only the exact spelling, and a
// /loop typed again from a subfolder found nothing to continue and was refused
// with loop_already_running instead.
func TestTheListFindsTheLoopThatBlocksTheWorkspace(t *testing.T) {
	h := &heldDaemon{found: []protocol.SpecFolder{{Path: "specs/a", Criteria: 1}}, release: make(chan struct{})}
	srv := loopServer(t, h)
	root := t.TempDir()
	ws := filepath.Join(root, "proj")
	sub := filepath.Join(ws, "sub")
	elsewhere := filepath.Join(root, "other")
	for _, dir := range []string{sub, elsewhere} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(ws, link); err != nil {
		t.Fatal(err)
	}
	run, err := srv.loops.start(protocol.StartLoopRequest{Workspace: ws, Args: "specs/a"})
	if err != nil {
		t.Fatal(err)
	}
	defer close(h.release)

	listed := func(ws string) []protocol.LoopRun {
		t.Helper()
		rec := get(srv, "/loops?"+url.Values{"workspace": {ws}}.Encode())
		if rec.Code != http.StatusOK {
			t.Fatalf("listing %s gave %d %s", ws, rec.Code, rec.Body.String())
		}
		var list protocol.ListLoopsResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
			t.Fatal(err)
		}
		return list.Loops
	}
	for _, from := range []string{ws, sub, root, link, filepath.Join(link, "sub")} {
		if got := listed(from); len(got) != 1 || got[0].ID != run.ID {
			t.Errorf("listing from %s gave %+v, want the loop that blocks it", from, got)
		}
	}
	if got := listed(elsewhere); len(got) != 0 {
		t.Errorf("listing a tree of its own gave %+v, want nothing", got)
	}
}

// A workspace the list cannot read is refused with workspace_invalid, never
// answered with an empty list that reads as "no loop here".
func TestTheListRefusesAWorkspaceItCannotRead(t *testing.T) {
	h := &heldDaemon{release: make(chan struct{})}
	srv := loopServer(t, h)
	missing := filepath.Join(t.TempDir(), "missing")
	rec := get(srv, "/loops?"+url.Values{"workspace": {missing}}.Encode())
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), protocol.CodeWorkspaceInvalid) {
		t.Errorf("listing a missing workspace gave %d %s, want workspace_invalid", rec.Code, rec.Body.String())
	}
}
