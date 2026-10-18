package specguard

// The orchestrator's wire vocabulary is declared before the daemon speaks it:
// step 2 of the orchestration .i puts the types on the wire so the steps after
// it change behaviour only. Each name below is excused until the step that
// sends or reads it, and the guard's other half — an excuse for a name that now
// has a user is an error — removes it there, one line at a time.
const orchestrationAhead = "orchestration wire vocabulary (orchestration .p §5), declared by step 2 of its .i ahead of the step that produces it"

// The gate is built by step 9 of the orchestration .i ahead of the pipeline
// that calls it after review and before the merge queue (steps 6 to 8 and 10).
const orchestrationGate = "the gate (orchestration .p §7.4, §7.6), built by step 9 of its .i; the per-child pipeline calls it after review and before the merge queue once steps 6 to 8 land"

const orchestrationClient = "pkg/client is the public API; a consumer conducts the orchestrator's routes (orchestration .p §5.1), as the TUI and the desktop will"

func init() {
	for _, name := range []string{
		"Checked", "Ended", "Forge", "Merged", "Method",
		"Recommended", "Returned", "Sensitive", "ValidRole", "What",
	} {
		exportedWithoutUser[name] = orchestrationAhead
	}
	for _, name := range []string{"CheckFresh", "GateFrom", "RecordGate"} {
		exportedWithoutUser[name] = orchestrationGate
	}
	for _, name := range []string{"AnswerQuestion", "ListProjects", "OpenProject"} {
		exportedWithoutUser[name] = orchestrationClient
	}
}
