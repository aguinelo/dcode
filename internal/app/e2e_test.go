package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
)

// scriptedModel is a provider that answers with what a test tells it to.
//
// Over real HTTP, in the real streaming format, reached through the real
// transport. A fake at the Provider interface would skip the encoder, the SSE
// parsing and the whole configuration chain — which is where three of this
// project's bugs have been.
type scriptedModel struct {
	mu    sync.Mutex
	turns [][]string
	seen  int
	// bodies is what the model was actually sent, which is the other half of
	// what an end-to-end run is worth: not only did the file change, but the
	// model saw the tools and the doctrine it was supposed to.
	bodies []string
}

func (m *scriptedModel) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	m.mu.Lock()
	m.bodies = append(m.bodies, string(body))
	turn := []string{}
	if m.seen < len(m.turns) {
		turn = m.turns[m.seen]
	}
	m.seen++
	m.mu.Unlock()

	w.Header().Set("Content-Type", "text/event-stream")
	for _, f := range turn {
		_, _ = w.Write([]byte("data: " + f + "\n\n"))
	}
	_, _ = w.Write([]byte("data: [DONE]\n\n"))
}

// A whole run, from configuration to a changed file.
//
// Everything below the command line is real: the chain that resolves options,
// the prompt the behaviour package builds, the registry, the policy, the
// resolver, the tools, the loop, the encoder and the SSE parser. Only the model
// is a fake, because the model is the one part that costs money and answers
// differently every time.
//
// This is the layer the suite did not have. Unit tests hold each piece; the
// eval harness measures the model. Nothing asserted that the pieces, wired the
// way the product wires them, change a file on disk when told to.
func TestARunChangesAFileOnDisk(t *testing.T) {
	ws := t.TempDir()
	target := filepath.Join(ws, "stats.go")
	if err := os.WriteFile(target, []byte("package stats\n\nfunc Rows() int { return 1 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	model := &scriptedModel{turns: [][]string{
		// Read it first, because the product refuses an edit to a file it has
		// not read — and an end-to-end run that skipped that would be testing
		// a product this one is not.
		{frameToolCall("c1", "read", `{"path":"stats.go"}`)},
		{frameToolCall("c2", "edit", `{"path":"stats.go","old_string":"return 1","new_string":"return 2"}`)},
		{frameText("Changed it.")},
	}}
	srv := httptest.NewServer(http.HandlerFunc(model.serve))
	defer srv.Close()

	opts := baseOpts(t)
	opts.Workspace = ws
	opts.BaseURL = srv.URL
	opts.Policy = policy.PolicyNever
	// A session that cannot confine its own commands refuses to start, by
	// design, and a CI runner without a usable namespace is the environment
	// talking rather than the behaviour under test. Every sibling test here
	// skips on it; failing instead is what turned this green on macOS and red
	// on Linux.
	requireSandbox(t, opts)
	sess, err := New(opts, &ConsoleEmitter{W: io.Discard}, DenyAll{})
	if err != nil {
		t.Fatalf("wiring a session failed: %v", err)
	}
	defer sess.Engine.Close()

	out, err := sess.Engine.Run(context.Background(), "make Rows return two")
	if err != nil {
		t.Fatalf("the run failed: %v", err)
	}
	if out.Reason != protocol.StopDone {
		t.Errorf("stopped as %q", out.Reason)
	}

	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "return 2") {
		t.Errorf("the file on disk was not changed:\n%s", body)
	}

	// And the model was sent a real prompt, not an empty one.
	if len(model.bodies) == 0 {
		t.Fatal("the model was never called")
	}
	first := model.bodies[0]
	for _, want := range []string{"make Rows return two", `"name":"edit"`, "dcode"} {
		if !strings.Contains(first, want) {
			t.Errorf("the request does not carry %q", want)
		}
	}
}

// The boundary holds in a real run, not only in a unit test of the evaluator.
//
// A model asking for a file outside the workspace is the case every layer of
// this is built around, and the only assertion worth making is that the bytes
// did not come back.
func TestARunRefusesToLeaveTheWorkspace(t *testing.T) {
	ws := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("not for the model"), 0o600); err != nil {
		t.Fatal(err)
	}

	model := &scriptedModel{turns: [][]string{
		{frameToolCall("c1", "read", `{"path":"`+outside+`"}`)},
		{frameText("I could not read that.")},
	}}
	srv := httptest.NewServer(http.HandlerFunc(model.serve))
	defer srv.Close()

	opts := baseOpts(t)
	opts.Workspace = ws
	opts.BaseURL = srv.URL
	opts.Policy = policy.PolicyNever
	// A session that cannot confine its own commands refuses to start, by
	// design, and a CI runner without a usable namespace is the environment
	// talking rather than the behaviour under test. Every sibling test here
	// skips on it; failing instead is what turned this green on macOS and red
	// on Linux.
	requireSandbox(t, opts)
	sess, err := New(opts, &ConsoleEmitter{W: io.Discard}, DenyAll{})
	if err != nil {
		t.Fatalf("wiring failed: %v", err)
	}
	defer sess.Engine.Close()

	if _, err := sess.Engine.Run(context.Background(), "read that file"); err != nil {
		t.Fatalf("the run failed: %v", err)
	}

	// The second request carries the tool result. The secret must not be in it.
	if len(model.bodies) < 2 {
		t.Fatal("the tool result never went back to the model")
	}
	for i, b := range model.bodies {
		if strings.Contains(b, "not for the model") {
			t.Fatalf("request %d carried the contents of a file outside the workspace", i)
		}
	}
}

// A tool that fails hands the model something to recover from rather than
// ending the turn, which is the property the whole error-message layer rests on.
func TestARunSurvivesAToolFailure(t *testing.T) {
	ws := t.TempDir()
	model := &scriptedModel{turns: [][]string{
		{frameToolCall("c1", "read", `{"path":"nowhere.go"}`)},
		{frameText("That file does not exist.")},
	}}
	srv := httptest.NewServer(http.HandlerFunc(model.serve))
	defer srv.Close()

	opts := baseOpts(t)
	opts.Workspace = ws
	opts.BaseURL = srv.URL
	opts.Policy = policy.PolicyNever
	// A session that cannot confine its own commands refuses to start, by
	// design, and a CI runner without a usable namespace is the environment
	// talking rather than the behaviour under test. Every sibling test here
	// skips on it; failing instead is what turned this green on macOS and red
	// on Linux.
	requireSandbox(t, opts)
	sess, err := New(opts, &ConsoleEmitter{W: io.Discard}, DenyAll{})
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Engine.Close()

	out, err := sess.Engine.Run(context.Background(), "read nowhere.go")
	if err != nil {
		t.Fatalf("a failing tool ended the run: %v", err)
	}
	if out.Reason != protocol.StopDone {
		t.Errorf("stopped as %q", out.Reason)
	}
	if len(model.bodies) < 2 || !strings.Contains(model.bodies[1], "does not exist") {
		t.Error("the model was not shown why the call failed")
	}
}

// A question put through the daemon is announced carrying the instant it
// lapses, and that is the instant the session holds it to.
//
// The loop announced the crossing before the session had set a deadline, so
// tool.approval_required went out with expires_at as the zero time. The
// deadline existed only on the copy the session kept, which no route exposes:
// a client could not show how long an approval had left, and a session in the
// background was denied two minutes later without anyone seeing it coming.
//
// Through the daemon's own wiring, because the defect lived between three
// parts — the loop that announces, the session that enforces, the daemon that
// holds the timeout — and each of them was right on its own.
func TestAnApprovalIsAnnouncedWithTheDeadlineTheSessionEnforces(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "elsewhere.txt")
	if err := os.WriteFile(outside, []byte("outside the workspace"), 0o600); err != nil {
		t.Fatal(err)
	}
	model := &scriptedModel{turns: [][]string{
		{frameToolCall("c1", "read", `{"path":"`+outside+`"}`)},
		{frameText("Understood.")},
	}}
	srv := httptest.NewServer(http.HandlerFunc(model.serve))
	defer srv.Close()

	base := baseOpts(t)
	base.BaseURL = srv.URL
	requireSandbox(t, base)
	const timeout = time.Minute
	d := NewDaemon(DaemonOptions{
		SocketPath:      filepath.Join(t.TempDir(), "d.sock"),
		ApprovalTimeout: timeout,
		Base:            base,
	})
	sess, err := d.build(protocol.CreateSessionRequest{Workspace: t.TempDir()})
	if err != nil {
		t.Fatalf("wiring a session through the daemon failed: %v", err)
	}
	defer sess.Close()

	asked := time.Now()
	if err := sess.Submit("read that file"); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, "the crossing was never put to anyone", func() bool { return len(sess.Pending()) > 0 })
	held := sess.Pending()[0]
	answered := time.Now()

	events, err := sess.Log.Replay(1)
	if err != nil {
		t.Fatal(err)
	}
	var announced []protocol.ApprovalRequest
	for _, ev := range events {
		if ev.Type != protocol.EventApprovalRequired {
			continue
		}
		var req protocol.ApprovalRequest
		if err := json.Unmarshal(ev.Payload, &req); err != nil {
			t.Fatal(err)
		}
		announced = append(announced, req)
	}
	if len(announced) != 1 {
		t.Fatalf("%d announcements for one crossing, want 1", len(announced))
	}
	got := announced[0]
	if got.ExpiresAt.IsZero() {
		t.Fatal("tool.approval_required went out with expires_at as the zero time; " +
			"a client has nothing to count down to")
	}
	if !got.ExpiresAt.Equal(held.ExpiresAt) {
		t.Errorf("announced expiring at %v, and the session holds the question to %v",
			got.ExpiresAt, held.ExpiresAt)
	}
	// The daemon's timeout, counted from when the question was put.
	if got.ExpiresAt.Before(asked.Add(timeout)) || got.ExpiresAt.After(answered.Add(timeout)) {
		t.Errorf("expires_at %v is not the %v timeout from when the question was put (%v to %v)",
			got.ExpiresAt, timeout, asked, answered)
	}

	// Answered, so the turn ends on the denial rather than being cut off.
	if err := sess.Resolve(held.ApprovalID, protocol.ApprovalDeny); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, "the turn did not end after the question was answered", func() bool {
		return sess.State() == protocol.SessionStateIdle
	})
}

// waitUntil polls cond, and fails the test with why if it never holds.
func waitUntil(t *testing.T, why string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal(why)
		}
		time.Sleep(time.Millisecond)
	}
}
