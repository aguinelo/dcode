package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/loop/qualifier"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/session"
)

// committing builds a daemon holding one proposal for session s1 in a fresh
// workspace, and returns the daemon, the session and the spec folder.
func committing(t *testing.T, criteria ...qualifier.Proposed) (*Daemon, *session.Session, string) {
	t.Helper()
	d := NewDaemon(DaemonOptions{Base: Options{SandboxMode: policy.ModeFullAccess}})
	ws := t.TempDir()
	spec := filepath.Join(ws, "specs", "x")
	if err := os.MkdirAll(spec, 0o755); err != nil {
		t.Fatal(err)
	}
	sess := newBareSession(t, "s1", ws)
	if err := d.manager.Add(sess); err != nil {
		t.Fatal(err)
	}
	held := &Proposals{}
	held.put(&Proposal{Spec: spec, Criteria: criteria, Protected: []string{"tests/**"}})
	d.holdProposals("s1", held)
	return d, sess, spec
}

// announced is every done.proposed in the session's log, decoded by name so
// the test reads the wire and not the Go type.
func announced(t *testing.T, sess *session.Session) []map[string]any {
	t.Helper()
	events, err := sess.Log.Replay(1)
	if err != nil {
		t.Fatal(err)
	}
	var out []map[string]any
	for _, ev := range events {
		if ev.Type != protocol.EventDoneProposed {
			continue
		}
		var p map[string]any
		if err := json.Unmarshal(ev.Payload, &p); err != nil {
			t.Fatal(err)
		}
		out = append(out, p)
	}
	return out
}

// A written proposal is announced in the qualifying session's log, carrying
// what was measured and the file it was written to — so a client can draw the
// review without reading the summary's prose.
//
// It is a notice: the qualifying turn has already ended, nothing waits for an
// answer, and nothing in it expires.
func TestAWrittenProposalIsAnnounced(t *testing.T) {
	d, sess, spec := committing(t,
		qualifier.Proposed{Name: "red", Command: "false", Expects: qualifier.ExpectFail},
		qualifier.Proposed{Name: "green", Command: "true", Expects: qualifier.ExpectFail, Why: "a claim the run contradicts"},
	)

	out, err := d.commitDone(context.Background(), "s1")
	if err != nil {
		t.Fatal(err)
	}

	got := announced(t, sess)
	if len(got) != 1 {
		t.Fatalf("writing a proposal announced it %d times, want once", len(got))
	}
	p := got[0]
	want := filepath.Join(spec, "done.toml")
	if p["path"] != want || out.Path != want {
		t.Errorf("announced path %v, response path %q; want both %q", p["path"], out.Path, want)
	}
	if id, _ := p["proposal_id"].(string); id == "" {
		t.Error("the announcement has no proposal_id")
	}
	for _, gone := range []string{"turn_id", "round", "expires_at"} {
		if _, ok := p[gone]; ok {
			t.Errorf("the announcement still carries %q: nothing waits for an answer", gone)
		}
	}
	criteria, _ := p["criteria"].([]any)
	if len(criteria) != 2 {
		t.Fatalf("announced %d criteria, want 2: %v", len(criteria), p["criteria"])
	}
	red, _ := criteria[0].(map[string]any)
	green, _ := criteria[1].(map[string]any)
	if red["class"] != string(qualifier.ClassAcceptance) || green["class"] != string(qualifier.ClassRegression) {
		t.Errorf("classes %v and %v, want acceptance and regression", red["class"], green["class"])
	}
	if green["mismatch"] != true {
		t.Errorf("a criterion said to fail that passed is not flagged: %v", green)
	}
	if prot, _ := p["protected"].([]any); len(prot) != 1 || prot[0] != "tests/**" {
		t.Errorf("protected %v, want [tests/**]", p["protected"])
	}
}

// A proposal that could not be written is not announced: the notice says a
// file exists for review, and a notice about a file that does not exist sends
// the person to read nothing.
func TestAProposalThatWasNotWrittenIsNotAnnounced(t *testing.T) {
	d, sess, spec := committing(t, qualifier.Proposed{Name: "a", Command: "false", Expects: qualifier.ExpectFail})
	if err := os.RemoveAll(spec); err != nil {
		t.Fatal(err)
	}

	if _, err := d.commitDone(context.Background(), "s1"); err == nil {
		t.Fatal("writing into a folder that is gone came back with no error")
	}
	if got := announced(t, sess); len(got) != 0 {
		t.Errorf("a proposal that was never written was announced: %v", got)
	}
}
