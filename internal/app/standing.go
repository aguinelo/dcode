package app

import (
	"sync"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
)

// StandingGrants answers, for one workspace, which crossings the user has
// already permitted — and writes down the ones they permit from now on.
//
// This is the layer that knows both sides: the session knows a crossing was
// declared, the config knows what the user has recorded, and neither should
// learn the other's vocabulary. Which boundaries are worth remembering is a
// product decision, and it lives here.
type StandingGrants struct {
	// Root is the USER's config root. Never the workspace: a record inside a
	// project would let a repository arrive pre-approved.
	Root      string
	Workspace string

	mu     sync.Mutex
	grants config.Grants
	// open is a grant that lasts only as long as this process. An "allow once"
	// has to open the boundary for the command being asked about — approving
	// something that then fails anyway is the worst of both answers — and it
	// must not survive a restart, because that is what "once" means.
	open bool
	// closed is a refusal that lasts as long as this process.
	//
	// A shell command is opaque, so the question is true of nearly every one of
	// them: without this, saying no would mean being asked again on the next
	// command, and the one after. Answering the same question twenty times is
	// the pattern that teaches people to stop reading it.
	//
	// Not written to disk, deliberately. A refusal that outlived the session
	// would be a boundary the user closed once and then could not find — and
	// the way out of it would be editing a file they do not know exists.
	closed bool
}

// NetworkNow reports whether the network is permitted at this moment.
//
// Asked per command rather than read once, because the answer can arrive
// mid-session: the user is asked at the first crossing and says yes. A boundary
// fixed at startup would leave that answer with no effect until a restart.
func (s *StandingGrants) NetworkNow() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.open || s.grants.Network(s.Workspace)
}

// NewStandingGrants loads what the user has already permitted.
//
// A record that cannot be read grants nothing and says so. Starting a session
// that silently permits more than the user agreed to is worse than refusing to
// start, and worse than asking again.
func NewStandingGrants(root, workspace string) (*StandingGrants, error) {
	g, err := config.LoadGrants(root)
	if err != nil {
		return nil, err
	}
	return &StandingGrants{Root: root, Workspace: workspace, grants: g}, nil
}

// rememberable names the crossings a standing answer applies to.
//
// Only the network, for now, and the restriction is deliberate rather than
// incidental. A network grant is one fact about one project: this workspace may
// reach out. A standing answer to "write outside the workspace" would be a
// grant over the whole filesystem given once, months ago, for a reason nobody
// records — and the path in that question is what makes it answerable.
func rememberable(req protocol.ApprovalRequest) bool {
	return req.BoundaryCrossed == string(policy.BoundaryNetwork)
}

// Granted reports a decision the user already made for this crossing.
func (s *StandingGrants) Granted(req protocol.ApprovalRequest) protocol.ApprovalDecision {
	if !rememberable(req) {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// A refusal answers too. The record is what the user decided, and "no" is a
	// decision — asking again until they say yes is not a boundary, it is
	// nagging.
	if s.closed {
		return protocol.ApprovalDeny
	}
	if s.open || s.grants.Network(s.Workspace) {
		return protocol.ApprovalAllowProject
	}
	return ""
}

// Remember writes down an answer meant to outlive the session.
//
// It is applied in memory first and persisted second, so a session whose disk
// write fails still honours the decision for the rest of its life. The user
// answered; losing the file costs them being asked again next time, which is
// the safe direction to fail in.
func (s *StandingGrants) Remember(req protocol.ApprovalRequest, d protocol.ApprovalDecision) error {
	if !rememberable(req) {
		return nil
	}
	if d == protocol.ApprovalDeny {
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()
		return nil
	}
	if !d.Grants() {
		return nil
	}

	s.mu.Lock()
	// Every granting answer opens the boundary now. Approving a command that
	// then fails anyway is the worst of both answers: the user consented and
	// got a refusal, with nothing explaining which of the two applied.
	s.open = true

	if !d.Remembered() {
		s.mu.Unlock()
		return nil
	}
	switch d {
	case protocol.ApprovalAllowAlways:
		s.grants = s.grants.GrantNetworkAlways()
	default:
		s.grants = s.grants.GrantNetwork(s.Workspace)
	}
	snapshot := s.grants
	root := s.Root
	s.mu.Unlock()

	return snapshot.Save(root)
}
