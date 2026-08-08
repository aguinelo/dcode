// Package policy decides what an agent may do, and what needs consent first.
//
// Two orthogonal axes, and keeping them apart is what makes the model usable:
//
//   - SandboxMode is the technical boundary the operating system enforces.
//   - ApprovalPolicy is the authorization question, independent of that
//     boundary.
//
// Conflating them is how harnesses end up asking about everything, users turn
// prompting off entirely, and the security model becomes decoration. That is
// the real failure mode — not the sophisticated attack, but the exhausted user.
//
// Evaluate is pure. Security decided by a model is not security: a prompt could
// argue its way past it.
//
// Spec: docs/specs/architecture/sandbox-policy/202608072336-*.
package policy

import "fmt"

// SandboxMode is the technical boundary.
type SandboxMode string

const (
	ModeReadOnly       SandboxMode = "read-only"
	ModeWorkspaceWrite SandboxMode = "workspace-write"
	ModeFullAccess     SandboxMode = "full-access"
)

// ParseMode validates a configured mode. An unknown value is an error rather
// than a silent fallback: falling back to a default would either surprise the
// user with less access than asked, or — far worse — with more.
func ParseMode(s string) (SandboxMode, error) {
	switch SandboxMode(s) {
	case ModeReadOnly, ModeWorkspaceWrite, ModeFullAccess:
		return SandboxMode(s), nil
	}
	return "", fmt.Errorf("policy: unknown sandbox mode %q; valid: %s, %s, %s",
		s, ModeReadOnly, ModeWorkspaceWrite, ModeFullAccess)
}

// ApprovalPolicy is the authorization axis.
type ApprovalPolicy string

const (
	// PolicyUntrusted escalates everything that is not a workspace read.
	PolicyUntrusted ApprovalPolicy = "untrusted"
	// PolicyOnRequest escalates only at a boundary crossing.
	PolicyOnRequest ApprovalPolicy = "on-request"
	// PolicyNever never asks, and denies what would have been asked.
	PolicyNever ApprovalPolicy = "never"
)

// ParsePolicy validates a configured policy.
func ParsePolicy(s string) (ApprovalPolicy, error) {
	switch ApprovalPolicy(s) {
	case PolicyUntrusted, PolicyOnRequest, PolicyNever:
		return ApprovalPolicy(s), nil
	}
	return "", fmt.Errorf("policy: unknown approval policy %q; valid: %s, %s, %s",
		s, PolicyUntrusted, PolicyOnRequest, PolicyNever)
}

// Access is one path a call would touch, already resolved.
type Access struct {
	Path  string
	Write bool
}

// Request is what a tool declares it would do, before doing any of it.
type Request struct {
	Tool    string
	Paths   []Access
	Network bool
	Command string
}

// Decision is the verdict.
type Decision string

const (
	DecisionAllow    Decision = "allow"
	DecisionEscalate Decision = "escalate"
	DecisionDeny     Decision = "deny"
)

// Boundary names what a request would cross.
type Boundary string

const (
	BoundaryNone           Boundary = ""
	BoundaryNetwork        Boundary = "network"
	BoundaryWorkspaceWrite Boundary = "workspace_write"
	BoundaryFilesystemRead Boundary = "filesystem_read"
	BoundaryFilesystemWrit Boundary = "filesystem_write"
)

// Verdict is the outcome of Evaluate.
type Verdict struct {
	Decision Decision
	Boundary Boundary
	Reason   string
}

// Evaluate decides on a request. Pure: no I/O, no clock. Path resolution
// happens earlier in Resolve, precisely so this stays exactly testable — every
// cell of the decision table gets an assertion, because the cell nobody tested
// is the one that will be wrong.
//
// inWorkspace reports containment; the caller supplies it from Resolve.
func Evaluate(r Request, mode SandboxMode, pol ApprovalPolicy, inWorkspace func(Access) bool) Verdict {
	v := evaluateMode(r, mode, inWorkspace)
	return applyPolicy(v, pol)
}

// evaluateMode applies the sandbox axis: what is physically possible.
func evaluateMode(r Request, mode SandboxMode, inWorkspace func(Access) bool) Verdict {
	if mode == ModeFullAccess {
		return Verdict{Decision: DecisionAllow, Reason: "full access"}
	}

	if r.Network {
		if mode == ModeReadOnly {
			return Verdict{DecisionDeny, BoundaryNetwork, "network is unavailable in read-only mode"}
		}
		return Verdict{DecisionEscalate, BoundaryNetwork, "this would reach the network"}
	}

	// Writes are checked before reads: a call that both reads and writes is a
	// write as far as the boundary is concerned.
	for _, a := range r.Paths {
		if !a.Write {
			continue
		}
		if mode == ModeReadOnly {
			return Verdict{DecisionDeny, BoundaryFilesystemWrit,
				"writing is unavailable in read-only mode"}
		}
		if !inWorkspace(a) {
			return Verdict{DecisionEscalate, BoundaryFilesystemWrit,
				"this would write outside the workspace"}
		}
	}

	for _, a := range r.Paths {
		if a.Write || inWorkspace(a) {
			continue
		}
		return Verdict{DecisionEscalate, BoundaryFilesystemRead,
			"this would read outside the workspace"}
	}

	if hasWrite(r.Paths) {
		return Verdict{DecisionAllow, BoundaryWorkspaceWrite, "writes inside the workspace"}
	}
	return Verdict{Decision: DecisionAllow, Reason: "reads inside the workspace"}
}

// applyPolicy applies the authorization axis on top of the boundary verdict.
func applyPolicy(v Verdict, pol ApprovalPolicy) Verdict {
	switch pol {
	case PolicyUntrusted:
		// The one place the two policies differ in practice: untrusted asks
		// before writing even inside the workspace.
		if v.Decision == DecisionAllow && v.Boundary == BoundaryWorkspaceWrite {
			v.Decision = DecisionEscalate
			v.Reason = "untrusted policy asks before any write"
		}
	case PolicyNever:
		if v.Decision == DecisionEscalate {
			// Denying is the only safe reading. With nobody to ask, the
			// alternative would be granting in silence.
			v.Decision = DecisionDeny
			v.Reason = v.Reason + " (denied: policy never asks)"
		}
	}
	return v
}

func hasWrite(paths []Access) bool {
	for _, a := range paths {
		if a.Write {
			return true
		}
	}
	return false
}
