package specguard

// The orchestrator's wire vocabulary is declared before the daemon speaks it:
// step 2 of the orchestration .i puts the types on the wire so the steps after
// it change behaviour only. Each name below is excused until the step that
// sends or reads it, and the guard's other half — an excuse for a name that now
// has a user is an error — removes it there, one line at a time.
const orchestrationAhead = "orchestration wire vocabulary (orchestration .p §5), declared by step 2 of its .i ahead of the step that produces it"

// The decision log is built by step 4 of the orchestration .i, before the
// orchestrator that opens it (step 5) and the command that reads it (step 11).
const orchestrationLog = "the decision log (orchestration .p §9.1), built by step 4 of its .i; step 5 opens it for each project and says what a reopen cut"

const orchestrationClient = "pkg/client is the public API; a consumer conducts the orchestrator's routes (orchestration .p §5.1), as the TUI and the desktop will"

func init() {
	for _, name := range []string{
		"Answered", "Autonomy", "CI", "Checked", "ChildChanged", "ChildCompleted", "Created", "Ended",
		"EventChildCompleted", "EventChildStarted", "EventChildState", "EventGateMeasured",
		"EventMainHealth", "EventMergeCompleted", "EventMergeRefused", "EventQuestionAnswered",
		"EventQuestionAsked", "EventReviewVerdict", "Forge", "GateMeasured", "Main", "MainHealth",
		"MergeCompleted", "MergeRefused", "Merged", "Method", "OrchestratorID", "OutputPath",
		"OutputTail", "Recommended", "Returned", "Review", "ReviewVerdict", "Sensitive", "ValidRole",
		"Waiting", "What",
	} {
		exportedWithoutUser[name] = orchestrationAhead
	}
	for _, name := range []string{"OpenDecisionLog", "Torn"} {
		exportedWithoutUser[name] = orchestrationLog
	}
	for _, name := range []string{"AnswerQuestion", "ListProjects", "OpenProject", "Orchestrator"} {
		exportedWithoutUser[name] = orchestrationClient
	}
}
