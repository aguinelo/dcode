package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
)

// The orchestrator's tools: it dispatches children, lists them, steers and
// closes them, and none of them waits for a child. They touch no path of the
// tree — the children write, in their worktrees — so they run in plan mode.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §10.

// Conductor is the project an orchestrator conducts, as its tools reach it.
// An interface for the reason Delegator is one: the daemon holds the project,
// and this package must not import what conducts it.
type Conductor interface {
	Delegate(items []DelegateItem) ([]protocol.Child, error)
	Children() ([]protocol.Child, error)
	SteerChild(childID, text, why string) error
	CloseChild(childID, why string) error
}

// DelegateItem is one child of a batch.
type DelegateItem struct {
	Title     string   `json:"title"`
	Task      string   `json:"task"`
	Owns      []string `json:"owns"`
	Model     string   `json:"model,omitempty"`
	PRTitle   string   `json:"pr_title,omitempty"`
	PRBody    string   `json:"pr_body,omitempty"`
	FixesMain bool     `json:"fixes_main,omitempty"`
	Why       string   `json:"why"`
}

// DelegateTool dispatches a batch of children, all or nothing.
type DelegateTool struct{ Conductor Conductor }

// ChildrenTool lists the orchestrator's children.
type ChildrenTool struct{ Conductor Conductor }

// SteerChildTool hands a child more work.
type SteerChildTool struct{ Conductor Conductor }

// CloseChildTool closes a child.
type CloseChildTool struct{ Conductor Conductor }

func (DelegateTool) Name() string   { return "delegate" }
func (ChildrenTool) Name() string   { return "children" }
func (SteerChildTool) Name() string { return "steer_child" }
func (CloseChildTool) Name() string { return "close_child" }

func (DelegateTool) Description() string {
	return "Dispatch children: each item becomes a child session working in its own git worktree, on its own branch " +
		"from main's tip, writing only the paths it owns. The batch is all or nothing: owns that cross each other or " +
		"a live child's, more children than allowed, a model outside the allowed list, or the token ceiling refuse the " +
		"whole batch before anything is made. It returns the children as soon as their first turns start; it never " +
		"waits for one to finish. Say why for each item: it goes to the decision log."
}

func (ChildrenTool) Description() string {
	return "List your children: their id, state, how their last turn ended, branch, owns and tokens."
}

func (SteerChildTool) Description() string {
	return "Give a child more to do: a new turn when it is idle or failed, or steering of the turn it is in. Say why."
}

func (CloseChildTool) Description() string {
	return "Close a child: its session ends, its worktree is removed and its owns are freed; its branch stays. Say why."
}

func (DelegateTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"items":{"type":"array","minItems":1,"items":{"type":"object","properties":{` +
		`"title":{"type":"string","description":"A short name for the child's theme"},` +
		`"task":{"type":"string","description":"What the child is to do, complete: it does not see your conversation"},` +
		`"owns":{"type":"array","items":{"type":"string"},"description":"Paths relative to the project's root the child alone may write; a folder owns what is under it, * and ? are globs"},` +
		`"model":{"type":"string","description":"A profile of the child role's list; empty is the first of it"},` +
		`"pr_title":{"type":"string"},"pr_body":{"type":"string"},` +
		`"fixes_main":{"type":"boolean","description":"The child repairs a red main"},` +
		`"why":{"type":"string","description":"Why this child, and why this model if you named one"}},` +
		`"required":["title","task","owns","why"]}}},"required":["items"]}`)
}

func (ChildrenTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{}}`)
}

func (SteerChildTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"child_id":{"type":"string"},"text":{"type":"string"},"why":{"type":"string"}},"required":["child_id","text","why"]}`)
}

func (CloseChildTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"child_id":{"type":"string"},"why":{"type":"string"}},"required":["child_id","why"]}`)
}

// None of them touches a path: the verdict is always allow, in plan mode too.
func (t DelegateTool) Declare(json.RawMessage) (policy.Request, error) {
	return policy.Request{Tool: t.Name()}, nil
}
func (t ChildrenTool) Declare(json.RawMessage) (policy.Request, error) {
	return policy.Request{Tool: t.Name()}, nil
}
func (t SteerChildTool) Declare(json.RawMessage) (policy.Request, error) {
	return policy.Request{Tool: t.Name()}, nil
}
func (t CloseChildTool) Declare(json.RawMessage) (policy.Request, error) {
	return policy.Request{Tool: t.Name()}, nil
}

func (t DelegateTool) Execute(_ context.Context, input json.RawMessage, _ *State) (Result, error) {
	var in struct {
		Items []DelegateItem `json:"items"`
	}
	if err := decode(t.Name(), input, &in); err != nil {
		return err.(*ToolError).Result(), nil
	}
	if t.Conductor == nil {
		return noConductor(t.Name()), nil
	}
	kids, err := t.Conductor.Delegate(in.Items)
	if err != nil {
		return refused(t.Name(), err), nil
	}
	return Result{Output: fmt.Sprintf("Dispatched %d child(ren); they run on their own, and you are told when each turn of theirs ends.\n%s",
		len(kids), RenderChildren(kids))}, nil
}

func (t ChildrenTool) Execute(_ context.Context, _ json.RawMessage, _ *State) (Result, error) {
	if t.Conductor == nil {
		return noConductor(t.Name()), nil
	}
	kids, err := t.Conductor.Children()
	if err != nil {
		return refused(t.Name(), err), nil
	}
	if len(kids) == 0 {
		return Result{Output: "No children."}, nil
	}
	return Result{Output: RenderChildren(kids)}, nil
}

func (t SteerChildTool) Execute(_ context.Context, input json.RawMessage, _ *State) (Result, error) {
	var in struct {
		ChildID string `json:"child_id"`
		Text    string `json:"text"`
		Why     string `json:"why"`
	}
	if err := decode(t.Name(), input, &in); err != nil {
		return err.(*ToolError).Result(), nil
	}
	if t.Conductor == nil {
		return noConductor(t.Name()), nil
	}
	if err := t.Conductor.SteerChild(in.ChildID, in.Text, in.Why); err != nil {
		return refused(t.Name(), err), nil
	}
	return Result{Output: "ok"}, nil
}

func (t CloseChildTool) Execute(_ context.Context, input json.RawMessage, _ *State) (Result, error) {
	var in struct {
		ChildID string `json:"child_id"`
		Why     string `json:"why"`
	}
	if err := decode(t.Name(), input, &in); err != nil {
		return err.(*ToolError).Result(), nil
	}
	if t.Conductor == nil {
		return noConductor(t.Name()), nil
	}
	if err := t.Conductor.CloseChild(in.ChildID, in.Why); err != nil {
		return refused(t.Name(), err), nil
	}
	return Result{Output: "ok"}, nil
}

// RenderChildren is the children as text, one line each, in a fixed order and
// shape: what the model reads is the same for the same children.
func RenderChildren(kids []protocol.Child) string {
	var b strings.Builder
	for _, c := range kids {
		state := c.State
		if c.Reason != "" {
			state += " (" + c.Reason + ")"
		}
		if c.Waiting != "" {
			state += " waiting on " + c.Waiting
		}
		u := c.Usage
		fmt.Fprintf(&b, "%s  %s  %q  branch %s  owns %s  tokens %d\n", c.ID, state, c.Title, c.Branch,
			strings.Join(c.Owns, ", "), u.InputTokens+u.OutputTokens+u.CacheReadTokens)
	}
	return strings.TrimRight(b.String(), "\n")
}

// refused is a refusal of the project, with its code in the text.
func refused(tool string, err error) Result {
	code := CodeDenied
	var e *protocol.Error
	if errors.As(err, &e) {
		code = e.Code
	}
	return Result{Output: fmt.Sprintf("%s refused (%s): %v", tool, code, err), IsError: true, Code: code}
}

func noConductor(tool string) Result {
	return errf(tool, CodeDenied, "", "%s has no project to conduct in this session", tool).Result()
}
