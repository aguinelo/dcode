package orchestrate

import (
	"github.com/aguinelo/dcode/internal/sandbox"
)

// GateSandbox makes the boundary one measurement runs in.
type GateSandbox func() (sandbox.Sandbox, error)
