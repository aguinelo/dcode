// Package contextengine assembles the message list sent to the model.
//
// Everything here is a pure function of session state: no I/O, no clock, no
// randomness. That is not a style preference — it is what guards ADR-03. The
// context prefix must be byte-identical between turns or the provider's cache
// misses and every turn re-bills the full prompt.
//
// Named contextengine rather than context so it never shadows the standard
// library package, which every file in this project imports.
//
// Spec: docs/specs/architecture/context-engine/202608072333-*.
package contextengine

import "encoding/json"

// Role identifies who produced a message.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message is one entry of the model conversation. It is the neutral type: no
// provider-specific shape reaches beyond the provider package.
type Message struct {
	Role       Role        `json:"role"`
	Text       string      `json:"text,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolResult *ToolResult `json:"tool_result,omitempty"`
}

// ToolCall is a model request to run a tool.
type ToolCall struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ToolResult is the outcome fed back to the model. IsError is not a failure of
// the turn: the model reads it and recovers.
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Output     string `json:"output"`
	IsError    bool   `json:"is_error"`
	Truncated  bool   `json:"truncated"`
}

// ToolDef is a tool declaration. The set is frozen at session creation: a
// definition that appears mid-session invalidates the whole cached prefix.
type ToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Schema      json.RawMessage `json:"schema"`
}

// Summary replaces a compacted span of history. UpToIdx is exclusive.
type Summary struct {
	Text    string `json:"text"`
	UpToIdx int    `json:"up_to_idx"`
}

// Session is the complete state Assemble is a function of. Nothing else is
// read: no globals, no environment, no clock.
type Session struct {
	// Instructions is the already-built system prompt. The behavior package
	// composes it; this package only places it.
	Instructions string
	// Tools is frozen at session creation.
	Tools []ToolDef
	// Summary is nil until the first compaction.
	Summary *Summary
	// History is append-only. Nothing already in it is ever edited.
	History []Message
}

// Config carries the knobs Plan and Estimate need. Passed in rather than read
// from the environment, so both stay pure.
type Config struct {
	// CompactAt is the fraction of the model window that triggers compaction.
	CompactAt float64
	// KeepTurns is how many recent turns survive beyond the mandatory ones.
	KeepTurns int
	// CharsPerToken is the estimation heuristic.
	CharsPerToken float64
	// Margin is added to every estimate to absorb heuristic error.
	Margin float64
	// Window is the model's context window in tokens. Supplied by the provider.
	Window int
}

// DefaultConfig mirrors the defaults documented in the config spec.
func DefaultConfig() Config {
	return Config{
		CompactAt:     0.80,
		KeepTurns:     4,
		CharsPerToken: 3.5,
		Margin:        0.10,
	}
}
