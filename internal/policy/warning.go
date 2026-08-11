package policy

// BoundaryWarning reports the combination that removes every boundary, or ""
// when there is one left.
//
// full-access lets the process touch anything the user can, and an approval
// policy of never means nobody is asked before it does. Either alone is a
// deliberate, defensible choice. Together there is no mechanism and no consent
// — the agent simply acts, and the first anyone knows is the result.
//
// It warns rather than refuses, and that is the right side of the line: someone
// running a container they will throw away wants exactly this, and a product
// that argues with them there is a product they route around. What it must not
// do is let it happen quietly.
//
// A pure function on purpose. The server logs it, the client shows it, and
// neither has to agree with the other about when.
func BoundaryWarning(mode SandboxMode, pol ApprovalPolicy) string {
	if mode != ModeFullAccess || pol != PolicyNever {
		return ""
	}
	return "sandbox.mode is full-access and sandbox.approval_policy is never: " +
		"there is no boundary and nobody will be asked. Every command runs with " +
		"your permissions, and the first you will know of one is its result."
}
