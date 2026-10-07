package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/session"
)

// What a session has to say as it opens comes after the conversation it
// continues, not above it: somebody continuing is looking at the end of what
// was carried, and a notice above eighteen thousand events is one nobody reads.
//
// Before the response, so last_seq counts the notices and a client reading up
// to it reads them. And recorded, unlike the carried events: they are this
// session's own, and the record is where somebody reads it afterwards.
func TestOpeningNoticesComeAfterTheContinuedConversation(t *testing.T) {
	notices := []protocol.Notice{
		{Code: protocol.NoticeFamilyUnmeasured, Message: "nobody measured it"},
		{Code: protocol.NoticeInstructionsUntranslated, Message: "written for another tool"},
	}
	var (
		built    *session.Session
		recorded []protocol.EventType
	)
	srv := New(Config{
		Manager: session.NewManager(10),
		Build: func(req protocol.CreateSessionRequest) (*session.Session, error) {
			log := session.NewEventLog("s1", 100, fixedClock())
			log.OnRecorded(func(ev protocol.Event) { recorded = append(recorded, ev.Type) })
			s := session.New("s1", req.Workspace, "m", "workspace-write", nil, log, fixedClock())
			for range 3 {
				s.Carried = append(s.Carried, protocol.Event{
					Type: protocol.EventMessageDelta, Payload: json.RawMessage(`{}`),
				})
			}
			s.CarriedFrom = "older"
			s.Notices = notices
			built = s
			return s, nil
		},
	})

	body, _ := json.Marshal(protocol.CreateSessionRequest{Workspace: t.TempDir()})
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/"+protocol.Version+"/sessions",
		strings.NewReader(string(body))))
	if rec.Code != http.StatusCreated {
		t.Fatalf("creating the session failed: %d %s", rec.Code, rec.Body.String())
	}

	events, err := built.Log.Replay(1)
	if err != nil {
		t.Fatal(err)
	}
	var types []protocol.EventType
	for _, ev := range events {
		types = append(types, ev.Type)
	}
	want := []protocol.EventType{
		protocol.EventSessionCreated, protocol.EventSessionResumed,
		protocol.EventMessageDelta, protocol.EventMessageDelta, protocol.EventMessageDelta,
		protocol.EventSessionNotice, protocol.EventSessionNotice,
	}
	if !slices.Equal(types, want) {
		t.Fatalf("the session opened as\n  %v\nwant\n  %v", types, want)
	}
	for i, n := range notices {
		var got protocol.Notice
		if err := json.Unmarshal(events[len(events)-2+i].Payload, &got); err != nil || got != n {
			t.Errorf("notice %d said %+v (%v), want %+v", i, got, err, n)
		}
	}

	var described protocol.Session
	if err := json.Unmarshal(rec.Body.Bytes(), &described); err != nil {
		t.Fatal(err)
	}
	if described.LastSeq != uint64(len(want)) {
		t.Errorf("last_seq is %d; a client reading up to it misses the notices after it", described.LastSeq)
	}
	if n := countOf(recorded, protocol.EventSessionNotice); n != len(notices) {
		t.Errorf("%d of %d notices reached the record: %v", n, len(notices), recorded)
	}
}

func countOf(types []protocol.EventType, t protocol.EventType) int {
	n := 0
	for _, x := range types {
		if x == t {
			n++
		}
	}
	return n
}
