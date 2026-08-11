package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/config"

	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
)

func netReq() protocol.ApprovalRequest {
	return protocol.ApprovalRequest{
		ApprovalID: "a1", Tool: "bash", Command: "go test ./...",
		BoundaryCrossed: string(policy.BoundaryNetwork),
	}
}

// The question is asked once and the answer survives the session, which is the
// whole difference between a boundary someone maintains and one they switch off.
func TestAnAnswerForThisProjectSurvivesIntoTheNextSession(t *testing.T) {
	root, ws := t.TempDir(), t.TempDir()

	first, err := NewStandingGrants(root, ws)
	if err != nil {
		t.Fatal(err)
	}
	if d := first.Granted(netReq()); d.Grants() {
		t.Fatalf("a fresh install already permitted the network: %q", d)
	}
	if err := first.Remember(netReq(), protocol.ApprovalAllowProject); err != nil {
		t.Fatal(err)
	}

	// A new session, reading the record from disk.
	next, err := NewStandingGrants(root, ws)
	if err != nil {
		t.Fatal(err)
	}
	if d := next.Granted(netReq()); !d.Grants() {
		t.Error("the user answered and is being asked again")
	}

	// And a different project is still its own decision.
	other, err := NewStandingGrants(root, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if d := other.Granted(netReq()); d.Grants() {
		t.Error("answering for one project answered for another")
	}
}

// "Always" is the answer that covers projects that do not exist yet, and it is
// the one the user has to choose deliberately.
func TestAlwaysCoversAProjectCreatedAfterwards(t *testing.T) {
	root := t.TempDir()
	s, _ := NewStandingGrants(root, t.TempDir())
	if err := s.Remember(netReq(), protocol.ApprovalAllowAlways); err != nil {
		t.Fatal(err)
	}

	later, _ := NewStandingGrants(root, t.TempDir())
	if d := later.Granted(netReq()); !d.Grants() {
		t.Error("always did not cover a project made after the answer")
	}
}

// Only the network is remembered, and the restriction is the point. A standing
// answer to "write outside the workspace" would be a grant over the whole
// filesystem, given once, for a reason nobody records — and the path in that
// question is what makes it answerable at all.
func TestOnlyTheNetworkIsRememberedAcrossSessions(t *testing.T) {
	root, ws := t.TempDir(), t.TempDir()
	s, _ := NewStandingGrants(root, ws)

	for _, boundary := range []policy.Boundary{
		policy.BoundaryFilesystemWrit, policy.BoundaryFilesystemRead,
		policy.BoundaryWorkspaceWrite, policy.BoundaryPathRuleWrite,
		policy.BoundaryCommandRule,
	} {
		req := netReq()
		req.BoundaryCrossed = string(boundary)
		if err := s.Remember(req, protocol.ApprovalAllowAlways); err != nil {
			t.Fatal(err)
		}
		if d := s.Granted(req); d.Grants() {
			t.Errorf("%s became a standing grant", boundary)
		}
	}

	// And remembering nothing must not have granted the network either.
	fresh, _ := NewStandingGrants(root, ws)
	if d := fresh.Granted(netReq()); d.Grants() {
		t.Error("answering about another boundary permitted the network")
	}
}

// An answer that is not a standing one is not written down, however it arrives.
func TestAMomentaryAnswerIsNotWrittenDown(t *testing.T) {
	root, ws := t.TempDir(), t.TempDir()
	s, _ := NewStandingGrants(root, ws)

	for _, d := range []protocol.ApprovalDecision{
		protocol.ApprovalAllow, protocol.ApprovalAllowSession, protocol.ApprovalDeny,
	} {
		if err := s.Remember(netReq(), d); err != nil {
			t.Fatal(err)
		}
	}
	next, _ := NewStandingGrants(root, ws)
	if got := next.Granted(netReq()); got.Grants() {
		t.Error("a momentary answer was turned into a permanent one")
	}
}

// A record that cannot be read must stop the session rather than start one that
// silently permits more than the user agreed to.
func TestAnUnreadableRecordRefusesToStart(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, config.GrantsFile), []byte("]["), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewStandingGrants(root, t.TempDir()); err == nil {
		t.Error("a corrupt record started a session anyway")
	}
}
