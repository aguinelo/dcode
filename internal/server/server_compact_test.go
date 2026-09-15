package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
)

func postCompact(t *testing.T, srv *Server, id string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost,
		"/"+protocol.Version+"/sessions/"+id+"/compact", nil)
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)
	return rec
}

// A fresh session has nothing worth summarising, and the response says so
// directly — the one thing EventSessionCompacted never arrives to say.
func TestCompactOnAFreshSessionSaysThereWasNothingToDo(t *testing.T) {
	srv, _, _ := modeServer(t, policy.ModeWorkspaceWrite, policy.PolicyOnRequest)

	rec := postCompact(t, srv, "live")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body)
	}
	var out protocol.CompactResult
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Compacted {
		t.Error("a fresh session with no history reported something was compacted")
	}
}

// The id is checked before anything else, so a request aimed at nothing
// never reaches a session — the same order every other route in this file
// already uses.
func TestCompactOnAnUnknownSessionIs404(t *testing.T) {
	srv, _, _ := modeServer(t, policy.ModeWorkspaceWrite, policy.PolicyOnRequest)

	if rec := postCompact(t, srv, "nope"); rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
