package app

import (
	"strings"
	"sync"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// sessionErrors keeps every session error a session emits, and nothing else.
type sessionErrors struct {
	mu  sync.Mutex
	got []protocol.Error
}

func (s *sessionErrors) Emit(t protocol.EventType, payload any) {
	if t != protocol.EventSessionError {
		return
	}
	if e, ok := payload.(protocol.Error); ok {
		s.mu.Lock()
		s.got = append(s.got, e)
		s.mu.Unlock()
	}
}

// A package.json that exists and does not parse is said, naming the file and
// why, at the moment the session opens.
//
// It used to read as "declares nothing": the prefix said nothing about gates,
// exactly as it does for a project that declares none, and nobody was told the
// probe had looked at a file it could not read. "I did not manage to read it"
// and "there is nothing there" are different facts and must not read the same.
func TestAnUnreadablePackageJSONIsSaid(t *testing.T) {
	opts := baseOpts(t)
	opts.WorkspaceGates = true
	writeIn(t, opts.Workspace, "package.json", "{not json at all")
	requireSandbox(t, opts)

	rec := &sessionErrors{}
	sess, err := New(opts, rec, DenyAll{})
	if err != nil {
		t.Fatalf("an unreadable package.json must not stop a session: %v", err)
	}
	defer sess.Engine.Close()

	for _, e := range rec.got {
		if e.Code == "manifest_unreadable" &&
			strings.Contains(e.Message, "package.json") &&
			strings.Contains(e.Message, "invalid character") {
			return
		}
	}
	t.Fatalf("no warning named package.json and its parse error; session errors: %+v", rec.got)
}

// A workspace with no manifest at all says nothing: absence is ordinary, and
// only a file that is there and cannot be read earns a warning.
func TestNoManifestIsNotAWarning(t *testing.T) {
	opts := baseOpts(t)
	opts.WorkspaceGates = true
	requireSandbox(t, opts)

	rec := &sessionErrors{}
	sess, err := New(opts, rec, DenyAll{})
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Engine.Close()

	for _, e := range rec.got {
		if e.Code == "manifest_unreadable" {
			t.Fatalf("an absent manifest was reported as unreadable: %+v", e)
		}
	}
}
