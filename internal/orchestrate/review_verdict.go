package orchestrate

// The reviewer's session and its verdict: what it may use, how the verdict is
// taken, and how it reaches the decision log.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §5.3, §7.2, §9.1, §10.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/tools"
)

// VerdictMissing is what a review that ended without a verdict records. The
// harness says it; the verdict tool refuses it from a model.
const VerdictMissing = "missing"

// ErrNoVerdict is a review that ended without calling verdict: a failure the
// pipeline acts on (run once more, then leave the child in_review), never an
// approval.
var ErrNoVerdict = errors.New("orchestrate: the review ended without a verdict, which is no review and never an approval")

// ReviewerRegistry is the reviewer's tool set: the four that only read, and
// verdict. Built from nothing rather than filtered from a session's, so a tool
// added to sessions later is not the reviewer's until someone puts it here —
// no write, no edit, no bash.
func ReviewerRegistry(verdict tools.Verdict) *tools.Registry {
	return tools.NewRegistry(tools.Read{}, tools.Glob{}, tools.Grep{}, tools.Symbol{}, verdict)
}

// Review is one reviewer turn on one commit. Submit is the verdict tool's;
// Finish, called when the turn ends, records what was said or that nothing
// was. Safe for concurrent use.
type Review struct {
	mu       sync.Mutex
	req      ReviewRequest
	session  string
	verdict  *protocol.Verdict
	why      string
	finished bool
}

// NewReview starts the review of req by the reviewer session session.
func NewReview(req ReviewRequest, session string) *Review {
	return &Review{req: req, session: session}
}

// Submit takes the reviewer's verdict. One per review, and one that holds
// together: approve without findings, changes_requested with at least one,
// each with a path, what and why, and a why for the verdict itself. The
// refusal goes back to the model to correct.
func (r *Review) Submit(_ context.Context, in tools.VerdictInput) (string, error) {
	v := protocol.Verdict{SessionID: r.session, Commit: r.req.Commit, Verdict: strings.TrimSpace(in.Verdict)}
	why := strings.TrimSpace(in.Why)
	if why == "" {
		return "", errors.New("a verdict needs its why: one or two sentences the decision log keeps")
	}
	switch v.Verdict {
	case tools.VerdictApprove:
		if len(in.Findings) > 0 {
			return "", errors.New("approve carries no findings; a defect is changes_requested")
		}
	case tools.VerdictChanges:
		if len(in.Findings) == 0 {
			return "", errors.New("changes_requested needs at least one finding: the path, what is wrong and why")
		}
		for i, f := range in.Findings {
			ff := protocol.Finding{Path: strings.TrimSpace(f.Path), Line: f.Line, What: strings.TrimSpace(f.What), Why: strings.TrimSpace(f.Why)}
			if ff.Path == "" || ff.What == "" || ff.Why == "" || ff.Line < 0 {
				return "", fmt.Errorf("finding %d needs a path, what is wrong and why it is wrong", i+1)
			}
			v.Findings = append(v.Findings, ff)
		}
	default:
		return "", fmt.Errorf("%q is not a verdict; it is approve or changes_requested", v.Verdict)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	switch {
	case r.finished:
		return "", errors.New("this review is over; the verdict was already recorded")
	case r.verdict != nil:
		return "", fmt.Errorf("the verdict on %s is already %s; a review gives one", short(r.req.Commit), r.verdict.Verdict)
	}
	r.verdict, r.why = &v, why
	return fmt.Sprintf("Verdict %s on %s recorded. The review is over: end your turn.", v.Verdict, short(r.req.Commit)), nil
}

// Finish records review.verdict and returns the verdict. A review that ended
// without one records missing and returns it with ErrNoVerdict. A line the
// log could not write is an error too: the verdict did not happen.
func (r *Review) Finish(l *DecisionLog) (protocol.Verdict, error) {
	if l == nil {
		return protocol.Verdict{}, errors.New("orchestrate: no decision log to record the verdict in")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.finished {
		return protocol.Verdict{}, fmt.Errorf("orchestrate: the review of %s by %s is already finished", short(r.req.Commit), r.session)
	}
	r.finished = true
	v, why, missing := r.verdict, r.why, r.verdict == nil
	if missing {
		v = &protocol.Verdict{SessionID: r.session, Commit: r.req.Commit, Verdict: VerdictMissing}
		why = fmt.Sprintf("the reviewer's turn on %s ended without calling verdict: no review, never an approval", short(r.req.Commit))
	}
	measured, err := json.Marshal(v)
	if err != nil {
		return protocol.Verdict{}, fmt.Errorf("orchestrate: encode the verdict: %w", err)
	}
	if _, err := l.Record(Decision{Kind: "review.verdict", Child: r.req.ChildID, Session: r.session, Why: why, Measured: measured}); err != nil {
		return protocol.Verdict{}, err
	}
	if missing {
		return *v, ErrNoVerdict
	}
	return *v, nil
}
