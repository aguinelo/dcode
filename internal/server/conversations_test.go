package server

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// fakeIndex stands in for the daemon's list, so the routes are asserted on
// their own: the folding is the session package's, and is tested there.
type fakeIndex struct {
	list      []protocol.Conversation
	listErr   error
	subErr    error
	changes   chan protocol.ConversationChange
	askedFor  string
	subscribe int
}

func (f *fakeIndex) List(ws string) ([]protocol.Conversation, error) {
	f.askedFor = ws
	return f.list, f.listErr
}

func (f *fakeIndex) Subscribe(context.Context) ([]protocol.Conversation, <-chan protocol.ConversationChange, error) {
	f.subscribe++
	if f.subErr != nil {
		return nil, nil, f.subErr
	}
	return f.list, f.changes, nil
}

func get(srv *Server, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/"+protocol.Version+path, nil))
	return rec
}

func TestListConversationsAnswersWhatTheIndexHolds(t *testing.T) {
	srv, _ := newServer(t, 4)
	idx := &fakeIndex{list: []protocol.Conversation{{ID: "a", Title: "conserte o parser", State: protocol.ConversationRecorded}}}
	srv.cfg.Conversations = idx

	rec := get(srv, "/conversations?workspace=/w")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var out protocol.ListConversationsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Conversations) != 1 || out.Conversations[0].ID != "a" || idx.askedFor != "/w" {
		t.Errorf("got %+v, filtered by %q", out, idx.askedFor)
	}

	if rec := get(srv, "/conversations?workspace=relative"); rec.Code != http.StatusBadRequest {
		t.Errorf("a relative workspace gave %d, want 400", rec.Code)
	}

	idx.listErr = protocol.Errorf(protocol.CodeInternal, "the record directory cannot be read")
	if rec := get(srv, "/conversations"); rec.Code == http.StatusOK || !strings.Contains(rec.Body.String(), "cannot be read") {
		t.Errorf("a failure was answered as a list: %d %s", rec.Code, rec.Body.String())
	}
}

// With no list kept, the list is empty — an array, not null, so a client can
// iterate it without a special case — and the stream is refused, saying why.
func TestADaemonWithNoListAnswersEmptyAndRefusesTheStream(t *testing.T) {
	srv, _ := newServer(t, 4)
	rec := get(srv, "/conversations")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"conversations":[]`) {
		t.Errorf("got %d %s, want an empty array", rec.Code, rec.Body.String())
	}
	if rec := get(srv, "/conversations/events"); rec.Code == http.StatusOK || !strings.Contains(rec.Body.String(), "no list") {
		t.Errorf("the stream of no list gave %d %s", rec.Code, rec.Body.String())
	}
}

func TestTheListStreamSendsTheSnapshotThenEachChange(t *testing.T) {
	srv, _ := newServer(t, 4)
	idx := &fakeIndex{
		list:    []protocol.Conversation{{ID: "a"}},
		changes: make(chan protocol.ConversationChange, 2),
	}
	srv.cfg.Conversations = idx
	hs := httptest.NewServer(srv.mux)
	defer hs.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, hs.URL+"/"+protocol.Version+"/conversations/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	frames := readFrames(resp)

	first := <-frames
	if first.Kind != protocol.ConversationSnapshot || len(first.Conversations) != 1 {
		t.Fatalf("the stream opened with %+v", first)
	}
	idx.changes <- protocol.ConversationChange{Kind: protocol.ConversationChanged, Conversation: &protocol.Conversation{ID: "b"}}
	if ch := <-frames; ch.Kind != protocol.ConversationChanged || ch.Conversation == nil || ch.Conversation.ID != "b" {
		t.Fatalf("the change came as %+v", ch)
	}
	// A list that drops this subscriber closes its channel, and the stream
	// ends, so the client reconnects to a fresh snapshot.
	close(idx.changes)
	if _, open := <-frames; open {
		t.Error("the stream went on after the list let it go")
	}
}

func TestTheListStreamSaysWhyItCannotOpen(t *testing.T) {
	srv, _ := newServer(t, 4)
	srv.cfg.Conversations = &fakeIndex{subErr: errors.New("the record directory cannot be read")}
	if rec := get(srv, "/conversations/events"); rec.Code == http.StatusOK || !strings.Contains(rec.Body.String(), "cannot be read") {
		t.Errorf("got %d %s", rec.Code, rec.Body.String())
	}
}

// readFrames decodes the data lines of a stream until it ends.
func readFrames(resp *http.Response) <-chan protocol.ConversationChange {
	out := make(chan protocol.ConversationChange, 8)
	go func() {
		defer close(out)
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			line := sc.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			var ch protocol.ConversationChange
			if json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &ch) == nil {
				out <- ch
			}
		}
	}()
	return out
}
