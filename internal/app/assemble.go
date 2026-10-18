package app

import (
	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/session"
)

// assemble builds the engine opts describe and the session wrap puts around
// it, bound to each other: a new session and a reopened one are put together
// the same way, and differ only in how the session itself is made.
func (d *Daemon) assemble(opts Options, id string, wrap func(*loop.Engine) *session.Session) (*session.Session, error) {
	// The session is its own emitter and its own approver: the loop announces
	// a crossing through the log and blocks until a client answers, which is
	// what lets any attached client resolve it.
	var sess *session.Session
	// Bound late, like the emitter and the approver above it: the engine is
	// built before the session that holds the queue exists.
	opts.Steer = func() string {
		if sess == nil {
			return ""
		}
		return sess.TakeSteering()
	}

	timeout := d.opts.ApprovalTimeout
	if opts.Child {
		// A child's question waits for the person, however long: nobody
		// else may answer it, and it must not lapse into a denial (ADR-02).
		timeout = 0
	}
	appSession, err := New(opts,
		emitterFunc(func(t protocol.EventType, payload any) {
			if sess != nil {
				sess.Emit(t, payload)
			}
		}),
		// The daemon's timeout, but for a child of an orchestrator, above.
		session.NewApprover(func() *session.Session { return sess }, timeout),
	)
	if err != nil {
		return nil, err
	}

	sess = wrap(appSession.Engine)
	sess.OnState(d.conversations.StateChanged)
	sess.ContextWindow = appSession.ContextWindow
	sess.Family, sess.Transport, sess.BaseURL = opts.Family, opts.Transport, opts.BaseURL
	sess.Branch = appSession.Branch
	// What makes `/mode` reach the model rather than only the sandbox.
	sess.Reprompt = appSession.Reprompt

	// The same record the sandbox is asking, not a second copy: two would
	// answer differently the moment one of them is granted.
	sess.Standing = appSession.Standing
	if appSession.Proposals != nil {
		// Kept beside the session rather than inside it, because a proposal is
		// the LOOP's to collect: the qualifying turn records one and ends, and
		// the loop asks for it afterwards.
		d.holdProposals(id, appSession.Proposals)
	}
	return sess, nil
}
