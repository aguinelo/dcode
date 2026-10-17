package session

import (
	"context"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// Red: today's approver, which always names a deadline.
type Approver struct {
	session func() *Session
	timeout time.Duration
}

func NewApprover(get func() *Session, timeout time.Duration) Approver {
	return Approver{session: get, timeout: timeout}
}

func (a Approver) Deadline() time.Time { return time.Now().Add(a.timeout) }

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
