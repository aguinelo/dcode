// Package tools implements what turns dcode into a coding agent rather than a
// chat: reading, searching, editing and running.
//
// This is the package with the most consequence per line. A wrong edit here
// corrupts a user's work silently, so several rules are enforced structurally
// rather than trusted to the model:
//
//   - edit only operates on a file read in this session and unchanged since;
//   - an ambiguous match fails instead of guessing;
//   - Declare reports what a call would touch without doing any of it, so
//     policy decides before there is anything to undo.
//
// Spec: docs/specs/architecture/tool-suite/202608072337-*.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/policy"
)

// Result is what a tool hands back to the loop.
type Result struct {
	Output    string
	IsError   bool
	Code      string
	Truncated bool
	Remaining int

	// Meta is what a client needs to render the call in one line without
	// parsing Output.
	//
	// Parsing prose to rebuild a number the tool already knew is how a UI
	// silently breaks when the wording changes. The tool states it once.
	Meta Meta
}

// Meta is the structured account of what a tool call did. Every field is
// optional: a tool reports what applies to it and leaves the rest zero.
type Meta struct {
	// Lines read, or matched, or written.
	Lines int
	// Files touched or matched.
	Files int
	// Added and Removed are line counts for an edit or a write.
	Added   int
	Removed int
	// ExitCode of a command. Meaningful only when HasExit is set, because a
	// successful command exits zero and zero is also the empty value.
	ExitCode int
	HasExit  bool
	// Diff is the unified diff of a change, for a client to render.
	//
	// It never reaches the model: the model wrote the edit and already knows
	// what it changed, so putting the diff in the history would pay tokens for
	// something nobody reads. It rides on the event instead.
	Diff string
}

// Tool is one capability.
type Tool interface {
	Name() string
	Description() string
	Schema() json.RawMessage
	// Declare reports what this call would touch, with no side effect at all.
	// Splitting it from Execute is what makes "policy decides first" a
	// structural property rather than a rule someone has to remember.
	Declare(input json.RawMessage) (policy.Request, error)
	Execute(ctx context.Context, input json.RawMessage, s *State) (Result, error)
}

// Error codes. Everything but denied is recoverable: the model reads the
// error and corrects itself, which is the whole reason the messages are
// written for a reader rather than for a log.
const (
	CodeFileNotRead     = "file_not_read"
	CodeFileChanged     = "file_changed"
	CodeNoMatch         = "no_match"
	CodeAmbiguousMatch  = "ambiguous_match"
	CodeNoOpEdit        = "no_op_edit"
	CodeNotFound        = "not_found"
	CodeInvalidPattern  = "invalid_pattern"
	CodeTimeout         = "timeout"
	CodePlanMultiActive = "plan_multiple_active"
	CodePlanNoReason    = "plan_blocked_no_reason"
	CodeDenied          = "denied"
	CodeBadInput        = "bad_input"
)

// ToolError is a failure the model is expected to recover from.
//
// Reason says what failed; Hint suggests a way forward without prescribing
// one. A generic message forces the model to guess, and guessing is how an
// agent corrupts a file.
type ToolError struct {
	Tool   string
	Code   string
	Reason string
	Detail string
	Hint   string
}

func (e *ToolError) Error() string { return e.Reason }

// Result renders the error as the model will read it.
func (e *ToolError) Result() Result {
	out := e.Reason
	if e.Detail != "" {
		out += "\n\n" + e.Detail
	}
	if e.Hint != "" {
		out += "\n\n" + e.Hint
	}
	return Result{Output: out, IsError: true, Code: e.Code}
}

func errf(tool, code, hint, format string, args ...any) *ToolError {
	return &ToolError{Tool: tool, Code: code, Hint: hint, Reason: fmt.Sprintf(format, args...)}
}

// Registry holds the enabled tools.
type Registry struct {
	byName map[string]Tool
	order  []string
}

// NewRegistry builds a registry from tools.
func NewRegistry(ts ...Tool) *Registry {
	r := &Registry{byName: map[string]Tool{}}
	for _, t := range ts {
		r.byName[t.Name()] = t
		r.order = append(r.order, t.Name())
	}
	sort.Strings(r.order)
	return r
}

// Get returns a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.byName[name]
	return t, ok
}

// Defs renders the declarations that go into the context prefix.
//
// Sorted by name so the output is byte-identical between runs: this lands in
// the cached prefix, and an unstable order would invalidate the cache of every
// session on every restart.
func (r *Registry) Defs() []ce.ToolDef {
	out := make([]ce.ToolDef, 0, len(r.order))
	for _, n := range r.order {
		t := r.byName[n]
		out = append(out, ce.ToolDef{
			Name:        t.Name(),
			Description: t.Description(),
			Schema:      t.Schema(),
		})
	}
	return out
}

// Names lists registered tool names, sorted.
func (r *Registry) Names() []string {
	out := make([]string, len(r.order))
	copy(out, r.order)
	return out
}

func decode(tool string, input json.RawMessage, v any) error {
	if len(input) == 0 {
		input = json.RawMessage("{}")
	}
	if err := json.Unmarshal(input, v); err != nil {
		return errf(tool, CodeBadInput,
			"Check the argument names and types against the tool schema.",
			"could not read the arguments for %s: %v", tool, err)
	}
	return nil
}
