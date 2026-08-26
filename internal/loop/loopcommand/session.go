package loopcommand

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/policy"
)

// SessionOptions configures the dedicated session /loop creates.
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

// NewSession builds the loop.Config the engine consumes for a /loop turn.
// Returns the config and the public-facing session ID.
//
// The ID is derived from the spec path basename plus a timestamp, so two
// /loop invocations on the same spec at different moments produce
// distinguishable IDs.
func NewSession(opts SessionOptions) (loop.Config, string) {
	cfg := loop.Config{
		Done:           opts.Spec.DoneSet(),
		DoneEnabled:    len(opts.Spec.Criteria) > 0,
		Limits:         opts.Limits,
		MaxStallCycles: opts.MaxStall,
		DoneTimeout:    opts.DoneTimeout,
		Mode:           opts.SandboxMode,
		Model:          opts.Model,
	}
	return cfg, sessionID(opts.SessionPrefix, opts.Spec.Path)
}

// sessionID is the public-facing identifier of a /loop session.
//
// Format: <prefix><basename>-<YYYYMMDDHHMMSS>. Two /loop calls on the
// same spec within the same second collide; that is acceptable for now
// (sub-second repetition is not a real use case) and the timestamp is
// what makes the IDs distinguishable when a user opens the rail.
func sessionID(prefix, specPath string) string {
	base := filepath.Base(specPath)
	if base == "." || base == "/" {
		base = "loop"
	}
	return fmt.Sprintf("%s%s-%s", prefix, base, time.Now().UTC().Format("20060102150405"))
}
