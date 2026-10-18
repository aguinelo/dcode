package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aguinelo/dcode/internal/policy"
)

// AskTool is how the orchestrator puts a design question to the person: it
// joins the project's queue of questions and the orchestrator goes on without
// waiting. The answer comes back as a report, at the start of a later turn.
// Only the person answers, and no tool of the orchestrator's does.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §5.1, §10.
type AskTool struct{ Conductor Conductor }

// AskOption is one answer a question offers.
type AskOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// AskInput is the argument shape.
type AskInput struct {
	Text        string      `json:"text"`
	Options     []AskOption `json:"options"`
	Recommended string      `json:"recommended"`
	Why         string      `json:"why"`
	ChildID     string      `json:"child_id,omitempty"`
}

func (AskTool) Name() string { return "ask" }

func (AskTool) Description() string {
	return "Ask the person a design question you cannot settle from the code and the task: the options, the one " +
		"you recommend, and why. It joins the queue of questions and returns its id at once; do not wait for it. " +
		"The answer reaches you as a report at the start of a later turn. Merges are not asked here: the pipeline " +
		"asks those itself."
}

func (AskTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{` +
		`"text":{"type":"string","description":"The question, complete: the person reads it without your conversation."},` +
		`"options":{"type":"array","minItems":2,"items":{"type":"object","properties":{"id":{"type":"string"},"label":{"type":"string"}},"required":["id","label"]}},` +
		`"recommended":{"type":"string","description":"The id of the option you recommend."},` +
		`"why":{"type":"string","description":"Why you recommend it; it goes to the decision log."},` +
		`"child_id":{"type":"string","description":"The child the question is about, if one."}},` +
		`"required":["text","options","recommended","why"]}`)
}

// Declare touches nothing: a question is a record, not a change to the tree.
func (t AskTool) Declare(json.RawMessage) (policy.Request, error) {
	return policy.Request{Tool: t.Name()}, nil
}

func (t AskTool) Execute(_ context.Context, input json.RawMessage, _ *State) (Result, error) {
	var in AskInput
	if err := decode(t.Name(), input, &in); err != nil {
		return err.(*ToolError).Result(), nil
	}
	if t.Conductor == nil {
		return noConductor(t.Name()), nil
	}
	id, err := t.Conductor.Ask(in)
	if err != nil {
		return refused(t.Name(), err), nil
	}
	return Result{Output: fmt.Sprintf("Asked as %s. Go on with what does not depend on it; the answer comes as a report.", id)}, nil
}
