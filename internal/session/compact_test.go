package session

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
)

// Compact does not run beside a turn, for the same reason Exec does not: its
// history has no mutex of its own, because nothing but the turn loop was
// ever meant to touch it while one runs.
func TestCompactDoesNotRunBesideATurn(t *testing.T) {
	s := New("s1", "/w", "m", "workspace-write", nil,
		NewEventLog("s1", 0, func() time.Time { return time.Unix(0, 0) }),
		func() time.Time { return time.Unix(0, 0) })

	// No engine here, so the refusal under test has to come first: the state
	// check is what this asserts, and it precedes the engine check.
	s.mu.Lock()
	s.state = protocol.SessionStateRunning
	s.mu.Unlock()

	_, err := s.Compact(context.Background())
	if err == nil {
		t.Fatal("compaction ran while a turn was running")
	}
	var perr *protocol.Error
	if !errors.As(err, &perr) || perr.Code != protocol.CodeTurnAlreadyActive {
		t.Errorf("the refusal does not say a turn is running: %v", err)
	}
}

// And a closed session says it is closed rather than that a turn is running.
func TestCompactOnAClosedSession(t *testing.T) {
	s := New("s1", "/w", "m", "workspace-write", nil,
		NewEventLog("s1", 0, func() time.Time { return time.Unix(0, 0) }),
		func() time.Time { return time.Unix(0, 0) })
	s.mu.Lock()
	s.state = protocol.SessionStateClosed
	s.mu.Unlock()

	_, err := s.Compact(context.Background())
	var perr *protocol.Error
	if !errors.As(err, &perr) || perr.Code != protocol.CodeSessionNotFound {
		t.Errorf("a closed session did not say so: %v", err)
	}
}

// minimalEngine is the smallest Engine Compact needs: a mode, a policy and a
// window. Compaction never touches the provider, the tools or the sandbox
// state, so none of them has to exist for this.
func minimalEngine(t *testing.T, s ce.Session) *loop.Engine {
	t.Helper()
	return loop.New(loop.Config{
		Mode: policy.ModeWorkspaceWrite, Policy: policy.PolicyOnRequest,
		CtxConfig: ce.Config{Window: 10_000_000},
	}, s)
}

// A fresh session has nothing worth summarising, and Compact says so rather
// than inventing a summary of nothing.
func TestCompactWithNothingWorthSummarisingReportsFalse(t *testing.T) {
	eng := minimalEngine(t, ce.Session{})
	s := New("s1", "/w", "m", "workspace-write", eng,
		NewEventLog("s1", 0, func() time.Time { return time.Unix(0, 0) }),
		func() time.Time { return time.Unix(0, 0) })

	compacted, err := s.Compact(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if compacted {
		t.Error("a fresh session with no history reported something was compacted")
	}
	if got := s.State(); got != protocol.SessionStateIdle {
		t.Errorf("session state = %q after compacting, want idle", got)
	}
}

// A long session has real work to summarise, and Compact does it — the
// /compact command's whole reason to exist: forced regardless of the
// 80%-threshold trigger, which a 10-million-token window would never cross
// on its own.
func TestCompactActuallyCompactsALongSession(t *testing.T) {
	long := ce.Session{Instructions: "You are dcode."}
	for i := 0; i < 30; i++ {
		long.History = append(long.History,
			ce.Message{Role: ce.RoleUser, Text: strings.Repeat("q ", 30)},
			ce.Message{Role: ce.RoleAssistant, Text: strings.Repeat("a ", 30)},
		)
	}
	eng := minimalEngine(t, long)
	s := New("s1", "/w", "m", "workspace-write", eng,
		NewEventLog("s1", 0, func() time.Time { return time.Unix(0, 0) }),
		func() time.Time { return time.Unix(0, 0) })

	compacted, err := s.Compact(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !compacted {
		t.Error("a long session with real work reported nothing to compact")
	}
}
