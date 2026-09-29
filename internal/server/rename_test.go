package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/session"
)

func recordDirWith(t *testing.T, id string) string {
	t.Helper()
	dir := t.TempDir()
	line := `{"seq":1,"session_id":"` + id + `","type":"session.created",` +
		`"at":"2026-08-21T00:00:00Z","payload":{"id":"` + id + `","workspace":"/w","model":"m"}}`
	if err := os.WriteFile(filepath.Join(dir, id+".jsonl"), []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// liveAndRecorded registers a loaded conversation that writes its record where
// the daemon keeps transcripts, opened the way createSession opens one.
func liveAndRecorded(t *testing.T, srv *Server, mgr *session.Manager, id string) *session.Session {
	t.Helper()
	srv.cfg.RecordDir = t.TempDir()
	rec, err := session.NewRecord(srv.cfg.RecordDir, id)
	if err != nil {
		t.Fatal(err)
	}
	log := session.NewEventLog(id, 0, fixedClock())
	log.SetRecord(rec)
	sess := session.New(id, "/w", "m", "workspace-write", nil, log, fixedClock())
	if err := mgr.Add(sess); err != nil {
		t.Fatal(err)
	}
	sess.Emit(protocol.EventSessionCreated, sess.Describe())
	return sess
}

func postName(t *testing.T, srv *Server, id, name string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(protocol.RenameSessionRequest{Name: name})
	req := httptest.NewRequest(http.MethodPost,
		"/"+protocol.Version+"/sessions/"+id+"/name", strings.NewReader(string(body)))
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)
	return rec
}

// The name reaches the record of a conversation that is not loaded, which is
// the case the rail actually has: it lists what a workspace recorded, and
// almost none of it is live.
func TestNamingReachesAConversationThatIsNotLoaded(t *testing.T) {
	srv, _ := newServer(t, 4)
	srv.cfg.RecordDir = recordDirWith(t, "s1")

	if got := postName(t, srv, "s1", "reformulação visual").Code; got != http.StatusNoContent {
		t.Fatalf("got %d", got)
	}
	found, err := session.Browse(srv.cfg.RecordDir, "/w")
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Name != "reformulação visual" {
		t.Errorf("the name did not reach the record: %+v", found)
	}
}

// A loaded conversation is named through its own log, so whoever is watching
// it sees the name arrive. Written straight to the file, the name reached the
// record and nobody attached: the stream never carried it.
func TestNamingALiveConversationReachesTheClientsWatchingIt(t *testing.T) {
	srv, mgr := newServer(t, 4)
	liveAndRecorded(t, srv, mgr, "s1")
	c := serve(t, srv)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	events, errs := c.Subscribe(ctx, "s1", 1)
	next := func(waiting string) protocol.Event {
		t.Helper()
		select {
		case ev, open := <-events:
			if !open {
				t.Fatalf("the stream closed waiting for %s", waiting)
			}
			return ev
		case err := <-errs:
			t.Fatalf("the stream failed waiting for %s: %v", waiting, err)
		case <-time.After(3 * time.Second):
			t.Fatalf("%s never reached the client watching the conversation", waiting)
		}
		return protocol.Event{}
	}
	// The stream is attached before the name is sent, so what comes after
	// session.created comes live rather than from a replay.
	if ev := next("session.created"); ev.Type != protocol.EventSessionCreated {
		t.Fatalf("the stream opened with %s", ev.Type)
	}

	if err := c.RenameSession(ctx, "s1", "reformulação visual"); err != nil {
		t.Fatal(err)
	}
	ev := next("the name")
	if ev.Type != protocol.EventSessionRenamed {
		t.Fatalf("the client got %s, want %s", ev.Type, protocol.EventSessionRenamed)
	}
	var d protocol.SessionRenamed
	if err := json.Unmarshal(ev.Payload, &d); err != nil {
		t.Fatal(err)
	}
	if d.Name != "reformulação visual" {
		t.Errorf("the client was told %q", d.Name)
	}
}

// The record takes the name in its place in the sequence, like every other
// event. Appended behind the log, it took the number after the file's last —
// the number the log hands the next live event, so the record held it twice.
func TestNamingALiveConversationKeepsItsRecordInSequence(t *testing.T) {
	srv, mgr := newServer(t, 4)
	sess := liveAndRecorded(t, srv, mgr, "s1")

	if got := postName(t, srv, "s1", "reformulação visual").Code; got != http.StatusNoContent {
		t.Fatalf("got %d", got)
	}
	// The conversation goes on after it is named.
	sess.Emit(protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1", Text: "catalogar os contratos"})
	// Closing flushes the record, which is where the sequence is read back.
	if err := mgr.Remove("s1"); err != nil {
		t.Fatal(err)
	}

	body, err := os.ReadFile(filepath.Join(srv.cfg.RecordDir, "s1.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var seqs []uint64
	for _, line := range strings.Split(strings.TrimSpace(string(body)), "\n") {
		var ev protocol.Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("a line of the record is not an event: %v", err)
		}
		seqs = append(seqs, ev.Seq)
	}
	// Strictly increasing and unique, which with nothing carried is also
	// without a gap: session.created, the name, the turn.
	if want := []uint64{1, 2, 3}; !slices.Equal(seqs, want) {
		t.Errorf("the record holds sequences %v, want %v", seqs, want)
	}
	// Through the log and still on disk: the rail reads names from the record.
	found, err := session.Browse(srv.cfg.RecordDir, "/w")
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Name != "reformulação visual" {
		t.Errorf("the name did not reach the record: %+v", found)
	}
}

// A session can close between being found and being named: a DELETE from
// another client lands in between. Named through a log that has closed, the
// name went to memory nobody reads and to a record already shut, and the
// answer still said it worked.
func TestNamingASessionThatHasClosedSaysSo(t *testing.T) {
	srv, mgr := newServer(t, 4)
	sess := liveAndRecorded(t, srv, mgr, "s1")
	// Closed and still held: the moment after the handler found it.
	sess.Close()

	if rec := postName(t, srv, "s1", "reformulação visual"); rec.Code != http.StatusNotFound {
		t.Fatalf("got %d (%s), want 404: the name went nowhere", rec.Code, rec.Body)
	}
	found, err := session.Browse(srv.cfg.RecordDir, "/w")
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Name != "" {
		t.Errorf("a name reached the record of a session that had closed: %+v", found)
	}
}

// Naming something that was never recorded says so, and creates nothing.
func TestNamingAnUnknownConversationIsNotFound(t *testing.T) {
	srv, _ := newServer(t, 4)
	srv.cfg.RecordDir = recordDirWith(t, "s1")

	if got := postName(t, srv, "nope", "x").Code; got == http.StatusNoContent {
		t.Error("naming a conversation that does not exist succeeded")
	}
	entries, _ := os.ReadDir(srv.cfg.RecordDir)
	if len(entries) != 1 {
		t.Errorf("a record was created for a conversation that does not exist: %d files", len(entries))
	}
}

// A daemon that keeps no transcripts has nothing to name, and says that rather
// than failing in a way that reads as the name being rejected.
func TestADaemonWithoutTranscriptsSaysThereIsNothingToName(t *testing.T) {
	srv, _ := newServer(t, 4)
	rec := postName(t, srv, "s1", "x")
	if rec.Code == http.StatusNoContent {
		t.Fatal("a daemon with no record directory accepted a name")
	}
	if !strings.Contains(rec.Body.String(), "transcripts") {
		t.Errorf("the refusal does not say why: %s", rec.Body.String())
	}
}

// Refused at the edge rather than written and read back wrong.
func TestAnOverLongNameIsRefusedByTheDaemon(t *testing.T) {
	srv, _ := newServer(t, 4)
	srv.cfg.RecordDir = recordDirWith(t, "s1")

	if got := postName(t, srv, "s1", strings.Repeat("a", session.NameLimit+1)).Code; got == http.StatusNoContent {
		t.Fatal("an over-long name was accepted")
	}
	found, _ := session.Browse(srv.cfg.RecordDir, "/w")
	if len(found) == 1 && found[0].Name != "" {
		t.Errorf("a refused name reached the record: %q", found[0].Name)
	}
}
