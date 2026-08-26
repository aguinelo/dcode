package loopcommand

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/policy"
)

// SessionOptions configures the dedicated session /loop will create.
//
// The session is born with the DoneSet already in place: this is what
// guarantees the engine sees the same definition of done the user typed
// /loop for, regardless of any later session-level mutation.
type SessionOptions struct {
	Spec          LoopSpec
	Limits        loop.Limits
	MaxStall      int
	DoneTimeout   time.Duration
	SandboxMode   policy.SandboxMode
	Workspace     string
	Model         string
	SessionPrefix string
}

// SessionConfig builds the loop.Config a /loop session is born with, and the
// name that session will carry.
//
// It creates nothing. The name says so, because the name that was here before
// — NewSession — promised a session, and the `.p` it was written against
// declared it as `NewSession(ctx, srv, opts) (SessionHandle, error)`: a call
// that reaches the server, hands back a handle with ID() and SubmitTurn(), and
// owns the session's lifetime. None of that exists. A constructor that
// constructs nothing is the kind of name a reader trusts once and then stops
// trusting the package.
//
// Creating the session, running the turn and closing it afterwards belong to
// the caller that does not exist yet — Step 3 of the family's `.i`. What can
// be built without a client is the Config that caller will pass, and this is
// it.
func SessionConfig(opts SessionOptions) (loop.Config, string) {
	cfg := loop.Config{
		Done:        opts.Spec.DoneSet(),
		DoneEnabled: len(opts.Spec.Criteria) > 0,
		Limits:      opts.Limits,
		// MaxStallCycles and DoneTimeout carry through untouched: /loop is a
		// façade and a façade has no budget of its own.
		MaxStallCycles: opts.MaxStall,
		DoneTimeout:    opts.DoneTimeout,
		Mode:           opts.SandboxMode,
		Model:          opts.Model,
	}
	return cfg, sessionName(opts.SessionPrefix, opts.Spec.Path)
}

// sessionName is the public-facing identifier a /loop session carries.
//
// Format: <prefix><basename>-<YYYYMMDDHHMMSS>. Two /loop calls on the
// same spec within the same second collide; that is acceptable for now
// (sub-second repetition is not a real use case) and the timestamp is
// what makes the names distinguishable when a user opens the rail.
func sessionName(prefix, specPath string) string {
	base := filepath.Base(specPath)
	if base == "." || base == string(filepath.Separator) {
		base = "loop"
	}
	return fmt.Sprintf("%s%s-%s", prefix, base, time.Now().UTC().Format("20060102150405"))
}
