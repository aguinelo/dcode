package behavior

// Roles of an orchestrated project, as the protocol spells them. Duplicated as
// strings rather than imported, for the reason Boundary's modes are, and
// compared against the protocol's in internal/app.
const (
	RoleOrchestrator = "orchestrator"
	RoleChild        = "child"
	RoleReviewer     = "reviewer"
)
