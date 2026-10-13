package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// recordOf writes a record of n questions, one event each.
func recordOf(t *testing.T, dir, id string, n int) {
	t.Helper()
	var b strings.Builder
	for i := 1; i <= n; i++ {
		raw, _ := json.Marshal(protocol.TurnStarted{TurnID: "t" + strconv.Itoa(i), Text: "pergunta " + strconv.Itoa(i)})
		line, _ := json.Marshal(protocol.Event{Seq: uint64(i), SessionID: id, Type: protocol.EventTurnStarted, Payload: raw})
		b.Write(line)
		b.WriteByte('\n')
	}
	if err := os.WriteFile(filepath.Join(dir, id+".jsonl"), []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
}

// A recorded conversation's events are read without loading it: nothing is
// added to the manager, and the page says how much there is and where to
// read on.
func TestARecordedConversationIsReadPageByPage(t *testing.T) {
	srv, mgr := newServer(t, 4)
	root := t.TempDir()
	srv.cfg.RecordDir = filepath.Join(root, "sessions")
	if err := os.Mkdir(srv.cfg.RecordDir, 0o700); err != nil {
		t.Fatal(err)
	}
	recordOf(t, srv.cfg.RecordDir, "s1", 5)
	// A record beside the directory, which no id may reach.
	recordOf(t, root, "secret", 1)

	cases := []struct {
		name   string
		path   string
		status int
		code   string
		seqs   []uint64
		total  int
		next   uint64
	}{
		{name: "whole", path: "/conversations/s1/events", status: 200, seqs: []uint64{1, 2, 3, 4, 5}, total: 5},
		{name: "first page", path: "/conversations/s1/events?limit=2", status: 200, seqs: []uint64{1, 2}, total: 5, next: 3},
		{name: "middle page", path: "/conversations/s1/events?from=3&limit=2", status: 200, seqs: []uint64{3, 4}, total: 5, next: 5},
		{name: "last page", path: "/conversations/s1/events?from=5&limit=2", status: 200, seqs: []uint64{5}, total: 5},
		{name: "past the end", path: "/conversations/s1/events?from=9", status: 200, seqs: nil, total: 5},
		{name: "missing record", path: "/conversations/nope/events", status: 404, code: protocol.CodeSessionNotFound},
		{name: "dot dot", path: "/conversations/../events", status: 0},
		{name: "escaped climb", path: "/conversations/..%2Fsecret/events", status: 400, code: protocol.CodeInvalidInput},
		{name: "escaped dot dot", path: "/conversations/%2E%2E/events", status: 400, code: protocol.CodeInvalidInput},
		{name: "escaped slash", path: "/conversations/a%2Fb/events", status: 400, code: protocol.CodeInvalidInput},
		{name: "from zero", path: "/conversations/s1/events?from=0", status: 400, code: protocol.CodeInvalidInput},
		{name: "from not a number", path: "/conversations/s1/events?from=x", status: 400, code: protocol.CodeInvalidInput},
		{name: "limit zero", path: "/conversations/s1/events?limit=0", status: 400, code: protocol.CodeInvalidInput},
		{name: "limit over the cap", path: "/conversations/s1/events?limit=" + strconv.Itoa(protocol.ConversationEventsLimit+1), status: 400, code: protocol.CodeInvalidInput},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := get(srv, c.path)
			body := rec.Body.String()
			if strings.Contains(body, "pergunta 1\"") && strings.Contains(c.path, "secret") {
				t.Fatalf("an id climbed out of the record directory: %s", body)
			}
			if c.status == 0 {
				// The mux cleans the path and redirects before any handler
				// runs; what matters is that it never answers events.
				if rec.Code == http.StatusOK {
					t.Fatalf("a climbing path was answered: %s", body)
				}
				return
			}
			if rec.Code != c.status {
				t.Fatalf("status %d, want %d: %s", rec.Code, c.status, body)
			}
			if c.code != "" {
				var e protocol.Error
				if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil || e.Code != c.code || e.Message == "" {
					t.Fatalf("error %+v, want %s with a reason", e, c.code)
				}
				return
			}
			var out protocol.ConversationEventsResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
				t.Fatal(err)
			}
			var seqs []uint64
			for _, e := range out.Events {
				seqs = append(seqs, e.Seq)
			}
			if !slices.Equal(seqs, c.seqs) || out.Total != c.total || out.Next != c.next {
				t.Errorf("got seqs %v total %d next %d, want %v %d %d", seqs, out.Total, out.Next, c.seqs, c.total, c.next)
			}
		})
	}
	if mgr.Count() != 0 {
		t.Errorf("reading a recorded conversation loaded %d sessions", mgr.Count())
	}
}

// A daemon that keeps no transcripts says so, rather than answering every
// conversation as empty.
func TestADaemonWithoutRecordsRefusesToReadOne(t *testing.T) {
	srv, _ := newServer(t, 4)
	rec := get(srv, "/conversations/s1/events")
	var e protocol.Error
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil || rec.Code != http.StatusBadRequest ||
		e.Code != protocol.CodeInvalidInput || !strings.Contains(e.Message, "transcripts") {
		t.Fatalf("status %d, error %+v: want invalid_input naming transcripts", rec.Code, e)
	}
}
