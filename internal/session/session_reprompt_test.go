package session

import (
	"errors"
	"strings"
	"testing"
	"time"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
)

// repromptSession is a session whose prompt records the boundary it was built
// for, which is the only thing these tests need to see.
func repromptSession(t *testing.T, m policy.SandboxMode, p policy.ApprovalPolicy) (*Session, *loop.Engine, *[]string) {
	t.Helper()
	eng := loop.New(loop.Config{Mode: m, Policy: p}, ce.Session{Instructions: "prompt for " + string(m)})
	s := New("s1", "/w", "model", string(m), eng, NewEventLog("s1", 100, time.Now), time.Now)
	var built []string
	s.Reprompt = func(mode policy.SandboxMode, pol policy.ApprovalPolicy) (string, error) {
		// What the ENGINE says, not what the argument says. The two agreeing is
		// the ordering guarantee: the boundary is installed before anything is
		// said about it, so there is no moment where the model has been
		// promised a freedom the sandbox has not granted yet.
		live, _ := eng.Mode()
		built = append(built, string(mode)+"/"+string(live))
		return "prompt for " + string(mode), nil
	}
	return s, eng, &built
}

// Switching the mode rebuilds the prompt.
//
// This is the whole of the report behind this change: the person switched to
// full-access, said so, and the model kept refusing a few turns later. It kept
// refusing because nothing in what it reads every turn had changed — only what
// the sandbox would allow had. The boundary was enforced and never stated.
func TestAModeChangeRebuildsTheSystemPrompt(t *testing.T) {
	s, _, built := repromptSession(t, policy.ModeWorkspaceWrite, policy.PolicyOnRequest)
	if err := s.SetMode(protocol.ModeAuto); err != nil {
		t.Fatalf("SetMode(auto): %v", err)
	}
	if len(*built) != 1 {
		t.Fatalf("the prompt was rebuilt %d times, want once", len(*built))
	}
	// Both halves: built for full-access, and built while the engine was
	// ALREADY enforcing full-access.
	if got := (*built)[0]; got != "full-access/full-access" {
		t.Errorf("rebuilt as %q, want full-access/full-access", got)
	}

	// Switching to the mode already in force rebuilds nothing: the no-op is a
	// no-op all the way down, and a prompt rebuilt for a boundary that did not
	// move is a cache thrown away for nothing.
	if err := s.SetMode(protocol.ModeAuto); err != nil {
		t.Fatalf("SetMode(auto) again: %v", err)
	}
	if len(*built) != 1 {
		t.Errorf("a no-op switch rebuilt the prompt %d times", len(*built)-1)
	}
}

// A session with no rebuilder still switches.
//
// The field is optional, and a session with no app behind it — every test in
// this package, and any embedder that wires its own — must not lose the ability
// to change mode because it cannot rebuild a prompt it never had.
func TestAModeChangeWithoutARebuilderStillSwitches(t *testing.T) {
	s := modeSession(t, policy.ModeWorkspaceWrite, policy.PolicyOnRequest)
	if err := s.SetMode(protocol.ModeAuto); err != nil {
		t.Fatalf("SetMode(auto): %v", err)
	}
	if got := s.Describe().SandboxMode; got != string(policy.ModeFullAccess) {
		t.Errorf("sandbox = %q, want full-access", got)
	}
}

// A rebuild that fails is said out loud, and the mode still changes.
//
// The boundary has already moved by the time the prompt is built: the engine is
// told what to enforce first, deliberately, so there is no window where the
// model is promised a freedom it does not have. That ordering means a failed
// rebuild leaves a session that is usable and a model that was not told, which
// is a thing to report rather than a thing to hide or to roll back into a
// half-applied switch.
func TestAFailedRebuildIsReportedAndDoesNotBlockTheSwitch(t *testing.T) {
	s, _, _ := repromptSession(t, policy.ModeWorkspaceWrite, policy.PolicyOnRequest)
	s.Reprompt = func(policy.SandboxMode, policy.ApprovalPolicy) (string, error) {
		return "", errors.New("instructions unreadable")
	}
	if err := s.SetMode(protocol.ModeAuto); err != nil {
		t.Fatalf("SetMode(auto): %v", err)
	}
	if got := s.Describe().SandboxMode; got != string(policy.ModeFullAccess) {
		t.Errorf("sandbox = %q, want full-access: the switch must not be half-applied", got)
	}
	evs, err := s.Log.Replay(1)
	if err != nil {
		t.Fatal(err)
	}
	var saw bool
	for _, ev := range evs {
		if ev.Type == protocol.EventSessionError {
			saw = true
			if !strings.Contains(string(ev.Payload), "was not told") {
				t.Errorf("the error does not say what was lost: %s", ev.Payload)
			}
		}
	}
	if !saw {
		t.Error("a failed rebuild was silent; the model is not being told and nobody can see it")
	}
}
