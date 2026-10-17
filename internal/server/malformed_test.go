package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// A request the daemon cannot read is the caller's mistake, and every route
// says so the same way: 400 invalid_input, with a message naming what was
// wrong. It used to answer 500 internal on most session routes, which tells a
// client the daemon broke and that retrying might help — neither is true.
func TestAMalformedRequestIsInvalidInputOnEveryRoute(t *testing.T) {
	srv, _ := newServer(t, 10)
	srv.cfg.RecordDir = t.TempDir()
	h := srv.Handler()
	p := "/" + protocol.Version
	created := httptest.NewRecorder()
	h.ServeHTTP(created, httptest.NewRequest(http.MethodPost, p+"/sessions",
		strings.NewReader(`{"workspace":"/w"}`)))
	if created.Code != http.StatusCreated {
		t.Fatalf("setup failed: %d %s", created.Code, created.Body.String())
	}
	s := p + "/sessions/s1"

	cases := []struct {
		name, method, path, body, names string
	}{
		{"create", http.MethodPost, p + "/sessions", `{{`, "malformed request"},
		{"rename", http.MethodPost, s + "/name", `{{`, "malformed request"},
		{"submit", http.MethodPost, s + "/turns", `{{`, "malformed request"},
		{"submit with a bad image", http.MethodPost, s + "/turns",
			`{"text":"look","images":[{"media_type":"image/png","data":"not base64!!"}]}`, "image"},
		{"exec", http.MethodPost, s + "/exec", `{{`, "malformed request"},
		{"steer", http.MethodPost, s + "/steer", `{{`, "malformed request"},
		{"mode", http.MethodPost, s + "/mode", `{{`, "malformed request"},
		{"approve", http.MethodPost, s + "/approvals/a1", `{{`, "malformed request"},
		{"events from", http.MethodGet, s + "/events?from=x", "", "from"},
		{"create pr", http.MethodPost, p + "/git/pr", `{{`, "malformed request"},
		{"start loop", http.MethodPost, p + "/loops", `{{`, "malformed request"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(c.method, c.path, strings.NewReader(c.body)))
			var e protocol.Error
			if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil ||
				rec.Code != http.StatusBadRequest || e.Code != protocol.CodeInvalidInput ||
				!strings.Contains(e.Message, c.names) {
				t.Errorf("status %d, body %s: want 400 %s naming %q",
					rec.Code, rec.Body.String(), protocol.CodeInvalidInput, c.names)
			}
		})
	}
}
