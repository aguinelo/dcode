package specguard

// The orchestrator's wire vocabulary is declared before the daemon speaks it:
// step 2 of the orchestration .i puts the types on the wire so the steps after
// it change behaviour only. Each name below is excused until the step that
// sends or reads it, and the guard's other half — an excuse for a name that now
// has a user is an error — removes it there, one line at a time.
const orchestrationAhead = "orchestration wire vocabulary (orchestration .p §5), declared by step 2 of its .i ahead of the step that produces it"

// The merge's lock on the measurement is built by step 9 of the orchestration
// .i; the merge step (step 10) calls it before a merge.
const orchestrationGate = "the gate's freshness check (orchestration .p §7.6), built by step 9 of its .i; the merge step of step 10 calls it before each merge"

const orchestrationClient = "pkg/client is the public API; a consumer conducts the orchestrator's routes (orchestration .p §5.1), as the TUI and the desktop will"

func init() {
	for _, name := range []string{
		"Checked", "Forge", "Merged", "Method",
		"Sensitive", "ValidRole",
	} {
		exportedWithoutUser[name] = orchestrationAhead
	}
	exportedWithoutUser["CheckFresh"] = orchestrationGate
	for _, name := range []string{"AnswerQuestion", "ListProjects", "OpenProject"} {
		exportedWithoutUser[name] = orchestrationClient
	}
}
