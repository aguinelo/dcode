package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/session"
)

// postMode drives the real handler through the real mux. An earlier version of
// this file built a ServeMux of its own with an inline handler and asserted on
// that, which tested net/http and nothing in this repository.
func postMode(t *testing.T, srv *Server, id, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost,
		"/"+protocol.Version+"/sessions/"+id+"/mode", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)
	return rec
}

// modeServer registers a live session whose engine really is in mode/pol, so
// the switch has something to land on.
func modeServer(t *testing.T, mode policy.SandboxMode, pol policy.ApprovalPolicy) (*Server, *session.Session, *loop.Engine) {
	t.Helper()
	srv, mgr := newServer(t, 10)
	eng := loop.New(loop.Config{Mode: mode, Policy: pol}, ce.Session{})
	sess := session.New("live", t.TempDir(), "MiniMax-M3", string(mode),
		eng, session.NewEventLog("live", 0, fixedClock()), fixedClock())
	if err := mgr.Add(sess); err != nil {
		t.Fatal(err)
	}
	return srv, sess, eng
}

// TestSetModeSwitchesTheEngine is the end the route exists for: what the
// handler answers matters less than what the engine runs afterwards.
func TestSetModeSwitchesTheEngine(t *testing.T) {
	srv, sess, eng := modeServer(t, policy.ModeWorkspaceWrite, policy.PolicyOnRequest)

	if rec := postMode(t, srv, "live", `{"mode":"plan"}`); rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (body %s)", rec.Code, rec.Body)
	}
	if got := sess.Describe().Mode; got != protocol.ModePlan {
		t.Errorf("session mode = %q, want plan", got)
	}
	mode, pol := eng.Mode()
	if mode != policy.ModeReadOnly || pol != policy.PolicyNever {
		t.Errorf("engine runs (%q, %q), want (read-only, never)", mode, pol)
	}
}

// TestSetModeRefusesAnUnknownName keeps a typo off the boundary, with a 4xx
// rather than a 500: the client sent something wrong, the daemon did not break.
func TestSetModeRefusesAnUnknownName(t *testing.T) {
	srv, sess, _ := modeServer(t, policy.ModeWorkspaceWrite, policy.PolicyOnRequest)

	rec := postMode(t, srv, "live", `{"mode":"turbo"}`)
	if rec.Code < 400 || rec.Code >= 500 {
		t.Errorf("status = %d, want a 4xx", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "turbo") {
		t.Errorf("the refusal must name what was sent: %s", rec.Body)
	}
	if got := sess.Describe().Mode; got != protocol.ModeAssist {
		t.Errorf("a refused switch changed the mode to %q", got)
	}
}

// TestSetModeOnAnUnknownSessionIs404: the id is checked before the body, so a
// switch aimed at nothing never reaches a session.
func TestSetModeOnAnUnknownSessionIs404(t *testing.T) {
	srv, _, _ := modeServer(t, policy.ModeWorkspaceWrite, policy.PolicyOnRequest)

	if rec := postMode(t, srv, "nope", `{"mode":"plan"}`); rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

// TestSetModeAnnouncesOverTheLog is what makes the mode readable by a client
// that was not attached when it changed.
func TestSetModeAnnouncesOverTheLog(t *testing.T) {
	srv, sess, _ := modeServer(t, policy.ModeWorkspaceWrite, policy.PolicyOnRequest)
	before := sess.Log.LastSeq()

	if rec := postMode(t, srv, "live", `{"mode":"auto"}`); rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	events, err := sess.Log.Replay(before + 1)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, ev := range events {
		if ev.Type == protocol.EventSessionModeChanged {
			found = true
			if !strings.Contains(string(ev.Payload), `"previous":"assist"`) {
				t.Errorf("the announcement must carry where it came from: %s", ev.Payload)
			}
		}
	}
	if !found {
		t.Error("no session.mode_changed on the log")
	}
}

// TestValidModeRoundtrips pins the public mode constants. Drift here is
// silent: the footer would render "[weird]" while the server refuses "weird".
func TestValidModeRoundtrips(t *testing.T) {
	for _, m := range []string{protocol.ModePlan, protocol.ModeAssist, protocol.ModeAuto} {
		if !protocol.ValidMode(m) {
			t.Errorf("ValidMode(%q) = false, want true", m)
		}
	}
	if protocol.ValidMode("turbo") {
		t.Error("ValidMode(turbo) = true, want false")
	}
	if protocol.ValidMode("") {
		t.Error("ValidMode(\"\") = true, want false")
	}
}
