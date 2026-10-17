package session

import (
	"context"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// Approver is what the loop asks a boundary question through: the session that
// holds the question, and the instant it lapses (loop.Deadliner).
//
// The timeout is named here and stamped on the question by the loop before it
// is announced, so every client sees the deadline the session holds it to.
//
// A timeout of zero is a question with NO deadline: ExpiresAt stays zero and
// the session waits for a person however long that takes. That is the child of
// an orchestrator, whose approval nobody else may answer and which must not
// lapse into a denial while the person is away (ADR-02).
type Approver struct {
	session func() *Session
	timeout time.Duration
}

// NewApprover asks through the session get returns — bound late, because the
// engine that asks is built before the session that answers.
func NewApprover(get func() *Session, timeout time.Duration) Approver {
	return Approver{session: get, timeout: timeout}
}

// Deadline is when a question put now lapses; zero is never.
func (a Approver) Deadline() time.Time {
	if a.timeout <= 0 {
		return time.Time{}
	}
	return time.Now().Add(a.timeout)
}

// Approve hands the question to the session. Without one there is nobody to
// ask, and denying is the only safe reading.
func (a Approver) Approve(ctx context.Context, req protocol.ApprovalRequest) (protocol.ApprovalDecision, error) {
	var s *Session
	if a.session != nil {
		s = a.session()
	}
	if s == nil {
		return protocol.ApprovalDeny, nil
	}
	return s.Approve(ctx, req)
}
