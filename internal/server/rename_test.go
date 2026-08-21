package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
