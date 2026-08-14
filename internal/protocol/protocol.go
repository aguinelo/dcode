// Package protocol is the shared vocabulary between the dcode daemon and its
// clients. It holds no logic and performs no I/O: both sides import it so the
// wire contract has exactly one definition.
//
// Spec: docs/specs/architecture/client-server-protocol/202608072240-*.
package protocol

import (
	"encoding/json"
	"time"
)

// Version is the protocol version carried in the URL prefix.
const Version = "v1"

// EventType identifies the payload shape of an Event.
type EventType string

// Event types. Payload shapes are defined in section 5.1 of the planning spec.
const (
	EventSessionCreated EventType = "session.created"
	EventTurnStarted    EventType = "turn.started"
	EventMessageDelta   EventType = "message.delta"
	// EventMessageReasoning carries the model's thinking, which is not its
	// answer. It is an event so a client can show it and so replaying a session
	// looks like having watched it — but it never enters the history the model
	// is sent, because a model that reads its own reasoning back as something
	// it said out loud starts defending it.
	EventMessageReasoning EventType = "message.reasoning"
	EventToolRequested    EventType = "tool.requested"
	EventApprovalRequired EventType = "tool.approval_required"
	EventApprovalResolved EventType = "tool.approval_resolved"
	EventToolCompleted    EventType = "tool.completed"
	EventTurnCompleted    EventType = "turn.completed"
	EventPlanUpdated      EventType = "plan.updated"
	EventSessionCompacted EventType = "session.compacted"
	EventSessionError     EventType = "session.error"
)

// Event is the envelope every observable fact travels in. Seq is per session,
// monotonic from 1, never reused and never gapped.
//
// At is the only non-deterministic field; golden comparisons zero it.
type Event struct {
	Seq       uint64          `json:"seq"`
	SessionID string          `json:"session_id"`
	Type      EventType       `json:"type"`
	At        time.Time       `json:"at"`
	Payload   json.RawMessage `json:"payload"`
}

// SessionState is the lifecycle state of a session.
type SessionState string

const (
	SessionStateIdle    SessionState = "idle"
	SessionStateRunning SessionState = "running"
	SessionStateBlocked SessionState = "blocked"
	SessionStateClosed  SessionState = "closed"
)

// Valid reports whether s is a state the server can be in.
func (s SessionState) Valid() bool {
	switch s {
	case SessionStateIdle, SessionStateRunning, SessionStateBlocked, SessionStateClosed:
		return true
	}
	return false
}

// Session is the server-owned session record. Clients hold no session state of
// their own beyond scroll position, panel visibility and their input queue.
type Session struct {
	ID          string       `json:"id"`
	State       SessionState `json:"state"`
	Workspace   string       `json:"workspace"`
	Model       string       `json:"model"`
	SandboxMode string       `json:"sandbox_mode"`
	CreatedAt   time.Time    `json:"created_at"`
	LastSeq     uint64       `json:"last_seq"`
	// ContextWindow is the model's window in tokens. The client needs it to
	// turn a token count into the percentage a person can act on; without it,
	// "12400 tokens" answers nothing.
	ContextWindow int `json:"context_window,omitempty"`
}

// CreateSessionRequest opens a session. Workspace must be absolute.
type CreateSessionRequest struct {
	Workspace   string `json:"workspace"`
	Model       string `json:"model,omitempty"`
	SandboxMode string `json:"sandbox_mode,omitempty"`
	// Resume names a recorded session whose conversation this one continues.
	//
	// A new session either way: the old one ended with the client that ran it,
	// and nothing survives what created it. What carries over is the history,
	// rebuilt from the record — not the approvals, which were consent given in
	// a moment that has passed, and not the background processes, which died
	// with the session that started them.
	Resume string `json:"resume,omitempty"`
}

// SubmitTurnRequest submits user input. Rejected with turn_already_active if a
// turn is already running: one turn per session.
type SubmitTurnRequest struct {
	Text string `json:"text"`
}

// UndoResult is what putting the last turn back changed, and what it would not
// touch.
//
// Refused is per file rather than all-or-nothing: seven files changed and one
// edited by hand should still give six back, and naming the one that stayed is
// more useful than refusing everything on account of it.
type UndoResult struct {
	Restored []string `json:"restored"`
	Refused  []string `json:"refused"`
}

// ApprovalDecision is the user's answer to a boundary crossing.
type ApprovalDecision string

const (
	ApprovalAllow        ApprovalDecision = "allow"
	ApprovalAllowSession ApprovalDecision = "allow_session"
	ApprovalDeny         ApprovalDecision = "deny"

	// The two answers that outlive the session, because some questions are
	// about the project rather than about the moment. Asking again next week
	// about a decision already made is how a prompt becomes something people
	// dismiss without reading — and a prompt nobody reads protects nobody.
	//
	// They are recorded in the USER's config root, never in the workspace: a
	// grant living inside a project would let a repository arrive
	// pre-approved, so cloning something would permit it before anyone read a
	// line of it.
	ApprovalAllowProject ApprovalDecision = "allow_project"
	ApprovalAllowAlways  ApprovalDecision = "allow_always"
)

// Valid reports whether d is a decision the server accepts.
func (d ApprovalDecision) Valid() bool {
	switch d {
	case ApprovalAllow, ApprovalAllowSession, ApprovalDeny,
		ApprovalAllowProject, ApprovalAllowAlways:
		return true
	}
	return false
}

// Remembered reports whether a decision outlives the session, and how widely.
//
// Named rather than compared inline, so the two places that persist a grant and
// the one that renders it cannot drift into disagreeing about which answers are
// standing ones.
func (d ApprovalDecision) Remembered() bool {
	return d == ApprovalAllowProject || d == ApprovalAllowAlways
}

// Grants reports whether a decision permits the crossing at all.
func (d ApprovalDecision) Grants() bool {
	return d != ApprovalDeny && d.Valid()
}

// ResolveApprovalRequest answers a pending approval. First writer wins.
type ResolveApprovalRequest struct {
	Decision ApprovalDecision `json:"decision"`
}

// ApprovalRequest is emitted when execution crosses the sandbox boundary. The
// turn blocks until it is resolved or ExpiresAt passes, which denies.
type ApprovalRequest struct {
	ApprovalID      string    `json:"approval_id"`
	TurnID          string    `json:"turn_id"`
	ToolCallID      string    `json:"tool_call_id"`
	Tool            string    `json:"tool"`
	Command         string    `json:"command,omitempty"`
	BoundaryCrossed string    `json:"boundary_crossed"`
	ExpiresAt       time.Time `json:"expires_at"`
	// Reason says why in a sentence, and Rule names the pattern that raised the
	// question when one did. Consent to a rule nobody can see is consent to
	// nothing, and Rule is also what an "allow for the session" answer is
	// remembered against.
	Reason string `json:"reason,omitempty"`
	Rule   string `json:"rule,omitempty"`
}

// PlanItem is one entry of the session plan, maintained by the plan tool.
type PlanItem struct {
	ID      int    `json:"id"`
	Text    string `json:"text"`
	Status  string `json:"status"`
	Blocked string `json:"blocked,omitempty"`
}

// Plan item statuses.
const (
	PlanPending = "pending"
	PlanActive  = "active"
	PlanDone    = "done"
	PlanBlocked = "blocked"
)

// PlanUpdated is the payload of EventPlanUpdated.
type PlanUpdated struct {
	Items []PlanItem `json:"items"`
}

// Turn stop reasons, carried in TurnCompleted.Reason.
const (
	StopDone          = "done"
	StopInterrupted   = "interrupted"
	StopMaxIterations = "max_iterations"
	StopRepeatLoop    = "repeat_loop"
	StopMaxTokens     = "max_tokens"
	StopError         = "error"
	// StopUnverified and StopIncomplete are RESULTS, not errors. They are the
	// honest state of work delivered without a check having passed.
	//
	// Treating them as failure would create exactly the wrong incentive: the
	// easy way out becomes switching the checking off.
	StopUnverified = "unverified"
	StopIncomplete = "incomplete"
)

// Payloads for the remaining event types.
type (
	// TurnStarted announces an accepted input.
	TurnStarted struct {
		TurnID string `json:"turn_id"`
		// Text is what the user asked for.
		//
		// It rides here because a turn starts BECAUSE of it, and because
		// nothing else carried it: the log held the model's side of a
		// conversation and none of the questions. A transcript could not be
		// read, a session could not be titled, and a client attaching to a
		// session already under way saw answers to questions it never saw.
		Text string `json:"text,omitempty"`
	}
	// MessageReasoning is a fragment of the model's thinking.
	MessageReasoning struct {
		TurnID string `json:"turn_id"`
		Text   string `json:"text"`
	}
	// MessageDelta is a fragment of model text.
	MessageDelta struct {
		TurnID string `json:"turn_id"`
		Text   string `json:"text"`
	}
	// ToolRequested announces a tool call before policy evaluation.
	ToolRequested struct {
		TurnID     string          `json:"turn_id"`
		ToolCallID string          `json:"tool_call_id"`
		Name       string          `json:"name"`
		Input      json.RawMessage `json:"input"`
	}
	// ApprovalResolved records the decision that unblocked a turn.
	ApprovalResolved struct {
		ApprovalID string           `json:"approval_id"`
		Decision   ApprovalDecision `json:"decision"`
	}
	// ToolCompleted carries the result of an execution.
	ToolCompleted struct {
		ToolCallID string `json:"tool_call_id"`
		OK         bool   `json:"ok"`
		Output     string `json:"output"`
		Truncated  bool   `json:"truncated"`

		// What the call did, stated once by the tool that knows.
		//
		// A client that parses Output to rebuild these numbers breaks silently
		// the day the wording changes, and every client has to reimplement the
		// same parsing. All optional: a tool reports what applies to it.
		Lines      int  `json:"lines,omitempty"`
		Files      int  `json:"files,omitempty"`
		Added      int  `json:"added,omitempty"`
		Removed    int  `json:"removed,omitempty"`
		ExitCode   int  `json:"exit_code,omitempty"`
		HasExit    bool `json:"has_exit,omitempty"`
		DurationMS int  `json:"duration_ms,omitempty"`

		// StartedAt and FinishedAt are the REAL execution order (RN-3.3).
		//
		// A duration cannot answer which of two concurrent calls went first,
		// and results arrive in emission order rather than the order they ran.
		// A client showing a batch has a person looking at it, and "which of
		// these happened first" is a reasonable thing for them to ask.
		//
		// They live HERE and never in the context sent to the model: a
		// timestamp in the prefix would make two runs of the same session
		// differ, which ADR-03 forbids. The event is a person's window; the
		// context is not.
		StartedAt  time.Time `json:"started_at,omitempty"`
		FinishedAt time.Time `json:"finished_at,omitempty"`

		// Diff is the unified diff of a change. Present only for tools that
		// modify a file, and never part of what the model was sent.
		Diff string `json:"diff,omitempty"`
	}
	// TurnCompleted ends a turn with one of the Stop* reasons.
	TurnCompleted struct {
		TurnID string `json:"turn_id"`
		Reason string `json:"reason"`
		// Usage is what the turn cost. Absent when the provider did not report
		// it, which is why it is a pointer: zero tokens and unknown tokens are
		// different facts and a client shows them differently.
		Usage *Usage `json:"usage,omitempty"`
		// Completion is what was and was not checked. Absent when the turn had
		// no definition of done, which is why it is a pointer: "nothing to
		// check" and "checked, all met" are different facts.
		//
		// It travels on the wire because it is the guarantee that survives a
		// model claiming success in prose. The text can lie; this cannot.
		Completion *Completion `json:"completion,omitempty"`
	}

	// Completion is the state of the done criteria when a turn ended.
	Completion struct {
		// Verification is the single-criterion seal: clean, passed, failed,
		// stale or unavailable.
		Verification string `json:"verification"`
		// Met and Unmet name the criteria, so a client can show which.
		Met   []string `json:"met,omitempty"`
		Unmet []string `json:"unmet,omitempty"`
		// Unavailable are criteria that could not be run at all. Different from
		// unmet, and shown differently: nothing was learned about them.
		Unavailable []string `json:"unavailable,omitempty"`
		// TouchedProtected are paths that are part of how the work is measured
		// and were written this turn. Never omitted when present.
		TouchedProtected []string `json:"touched_protected,omitempty"`
	}

	// Usage is the token accounting for a turn.
	Usage struct {
		InputTokens      int `json:"input_tokens"`
		OutputTokens     int `json:"output_tokens"`
		CacheReadTokens  int `json:"cache_read_tokens,omitempty"`
		CacheWriteTokens int `json:"cache_write_tokens,omitempty"`
	}
	// SessionCompacted records a context compaction.
	SessionCompacted struct {
		FromSeq uint64 `json:"from_seq"`
		ToSeq   uint64 `json:"to_seq"`
	}
)
