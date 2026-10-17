package protocol

import "time"

// The orchestrator's vocabulary on the wire: one orchestrator per project, its
// children as sessions with a parent, and what the pipeline measures of each.
// Nothing here answers yet; the routes and the events arrive with the steps
// that produce them.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §5.

// Roles a session can have under an orchestrator (Session.Role). Empty is a
// session nobody orchestrates.
const (
	RoleOrchestrator = "orchestrator"
	RoleChild        = "child"
	RoleReviewer     = "reviewer"
)

// ValidRole reports whether role is one of the three. Empty is not a role: a
// session without one is read by its absence, never by a value.
func ValidRole(role string) bool {
	switch role {
	case RoleOrchestrator, RoleChild, RoleReviewer:
		return true
	}
	return false
}

// Events of an orchestrator's log. A child's own progress stays in the child's
// log; these are what the live list and the conversation need.
const (
	EventChildStarted     EventType = "child.started"
	EventChildState       EventType = "child.state"
	EventChildCompleted   EventType = "child.completed"
	EventReviewVerdict    EventType = "review.verdict"
	EventGateMeasured     EventType = "gate.measured"
	EventQuestionAsked    EventType = "question.asked"
	EventQuestionAnswered EventType = "question.answered"
	EventMergeCompleted   EventType = "merge.completed"
	EventMergeRefused     EventType = "merge.refused"
	EventMainHealth       EventType = "main.health"
)

// Project is one tree conducted by one orchestrator.
type Project struct {
	// ID is 24 hex characters of the sha256 of Root.
	ID string `json:"id"`
	// Root is the main worktree's root, links resolved: every worktree of the
	// same tree is the same project.
	Root string `json:"root"`
	Main string `json:"main"`
	// Forge is "github" or "local".
	Forge string `json:"forge"`
	// Autonomy is "step", always, in this slice.
	Autonomy       string    `json:"autonomy"`
	OrchestratorID string    `json:"orchestrator_id"`
	Created        time.Time `json:"created"`
}

// Child is one child of an orchestrator, as the live list shows it. ID is the
// child's session id.
type Child struct {
	ID       string `json:"id"`
	ParentID string `json:"parent_id"`
	Title    string `json:"title"`
	Profile  string `json:"profile"`
	Model    string `json:"model"`
	// State is one of the eight of the state machine (.p §6).
	State string `json:"state"`
	// Reason is how the last turn ended; Waiting is "approval", "question" or
	// "merge" while the child waits for the person.
	Reason   string   `json:"reason,omitempty"`
	Waiting  string   `json:"waiting,omitempty"`
	Owns     []string `json:"owns"`
	Branch   string   `json:"branch"`
	Worktree string   `json:"worktree"`
	// Base is the main commit the branch left from; Head the last commit the
	// harness made on it.
	Base     string      `json:"base"`
	Head     string      `json:"head,omitempty"`
	Returned int         `json:"returned,omitempty"`
	Return   *Return     `json:"return,omitempty"`
	Review   *Verdict    `json:"review,omitempty"`
	Gate     *GateResult `json:"gate,omitempty"`
	PR       *ChildPR    `json:"pr,omitempty"`
	// Usage is summed over the child's turns.
	Usage   Usage      `json:"usage"`
	Started time.Time  `json:"started"`
	Ended   *time.Time `json:"ended,omitempty"`
}

// Return is why a child was handed its work back.
type Return struct {
	// Why is outside_owns, review, conflict, gate_red or person.
	Why string `json:"why"`
	// Phase is red or fix, for a return that asks for a red test first.
	Phase string   `json:"phase,omitempty"`
	Paths []string `json:"paths,omitempty"`
}

// Verdict is what the reviewer said of one commit: approve,
// changes_requested, or missing when it ended without saying.
type Verdict struct {
	SessionID string    `json:"session_id"`
	Commit    string    `json:"commit"`
	Verdict   string    `json:"verdict"`
	Findings  []Finding `json:"findings,omitempty"`
}

// Finding is one defect the reviewer pointed at, with why it is one.
type Finding struct {
	Path string `json:"path"`
	Line int    `json:"line,omitempty"`
	What string `json:"what"`
	Why  string `json:"why"`
}

// GateResult is one run of the gate on one commit.
type GateResult struct {
	// Commit is the commit measured, Tree its tree, Base the main under it.
	Commit     string `json:"commit"`
	Tree       string `json:"tree"`
	Base       string `json:"base"`
	Command    string `json:"command"`
	ExitCode   int    `json:"exit_code"`
	DurationMS int    `json:"duration_ms"`
	// Result is green, red, timeout or undeclared.
	Result string `json:"result"`
	// OutputTail is the last 4 KiB; OutputPath the whole output on disk.
	OutputTail string    `json:"output_tail,omitempty"`
	OutputPath string    `json:"output_path,omitempty"`
	At         time.Time `json:"at"`
}

// ChildPR is a child's pull request. Number is 0 for a local merge; CI is in
// GitChecks' grammar.
type ChildPR struct {
	Number int    `json:"number,omitempty"`
	URL    string `json:"url,omitempty"`
	CI     string `json:"ci,omitempty"`
}

// MainHealth is what the forge says of main at one commit: green, red, pending
// or unknown.
type MainHealth struct {
	Commit  string    `json:"commit"`
	Health  string    `json:"health"`
	Checked time.Time `json:"checked"`
}

// Question is one thing waiting for the person. Kind is merge, design or
// ceiling; Recommended is the ID of one of Options.
type Question struct {
	ID          string           `json:"id"`
	Kind        string           `json:"kind"`
	Text        string           `json:"text"`
	Recommended string           `json:"recommended"`
	Why         string           `json:"why"`
	ChildID     string           `json:"child_id,omitempty"`
	Options     []QuestionOption `json:"options"`
	// Sensitive are the paths of the diff that make a merge ask with care.
	Sensitive []string   `json:"sensitive,omitempty"`
	Asked     time.Time  `json:"asked"`
	Answer    string     `json:"answer,omitempty"`
	Note      string     `json:"note,omitempty"`
	Answered  *time.Time `json:"answered,omitempty"`
}

// QuestionOption is one answer a question offers.
type QuestionOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// OpenProjectRequest opens, or finds, the project of a workspace.
type OpenProjectRequest struct {
	Workspace string `json:"workspace"`
}

// ListProjectsResponse answers GET /projects.
type ListProjectsResponse struct {
	Projects []Project `json:"projects"`
}

// ListChildrenResponse answers GET /sessions/{id}/children: the live ones, and
// the closed ones still inside their retention.
type ListChildrenResponse struct {
	Children []Child `json:"children"`
}

// ListQuestionsResponse answers GET /sessions/{id}/questions.
type ListQuestionsResponse struct {
	Questions []Question `json:"questions"`
}

// AnswerQuestionRequest answers one question; the first answer wins.
type AnswerQuestionRequest struct {
	Answer string `json:"answer"`
	Note   string `json:"note,omitempty"`
}

// ChildChanged is the payload of child.started and child.state.
type ChildChanged struct {
	Child Child `json:"child"`
}

// ChildCompleted is the payload of child.completed, sent for every child turn
// that ends. Conclusion is cut at DelegateMaxResultBytes, and Truncated says so.
type ChildCompleted struct {
	ChildID    string   `json:"child_id"`
	TurnID     string   `json:"turn_id"`
	Reason     string   `json:"reason"`
	Conclusion string   `json:"conclusion"`
	Truncated  bool     `json:"truncated,omitempty"`
	Wrote      []string `json:"wrote,omitempty"`
	Unwritten  []string `json:"unwritten,omitempty"`
	Usage      Usage    `json:"usage"`
}

// ReviewVerdict is the payload of review.verdict: the verdict, flat, with the
// child it is about.
type ReviewVerdict struct {
	ChildID string `json:"child_id"`
	Verdict `tstype:",extends"`
}

// GateMeasured is the payload of gate.measured: the result, flat, with the
// child it is about. Sent for every measurement, green or not.
type GateMeasured struct {
	ChildID    string `json:"child_id"`
	GateResult `tstype:",extends"`
}

// MergeCompleted is the payload of merge.completed. Merged is the commit on
// main; Measured the one the gate ran on. PR is 0 for a local merge.
type MergeCompleted struct {
	ChildID  string `json:"child_id"`
	Method   string `json:"method"`
	PR       int    `json:"pr,omitempty"`
	Merged   string `json:"merged"`
	Measured string `json:"measured"`
}

// MergeRefused is the payload of merge.refused: main_red, gate_stale,
// gate_undeclared, or the forge's own refusal, with the reason.
type MergeRefused struct {
	ChildID string `json:"child_id"`
	Code    string `json:"code"`
	Reason  string `json:"reason"`
}
