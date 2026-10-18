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
// Only the person answers, and no tool of the orchestrator's does: summarize
// and propose_answers help the person decide, and neither answers.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §5.1, §5.5, §10.
type AskTool struct{ Conductor Conductor }

// SummarizeTool writes what changed into a question that waits.
type SummarizeTool struct{ Conductor Conductor }

// ProposeAnswersTool turns the person's words into a proposed card.
type ProposeAnswersTool struct{ Conductor Conductor }

// AskOption is one answer a question offers, and what choosing it does.
type AskOption struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// AskInput is the argument shape.
type AskInput struct {
	Header      string      `json:"header,omitempty"`
	Text        string      `json:"text"`
	Options     []AskOption `json:"options"`
	Recommended string      `json:"recommended"`
	Why         string      `json:"why"`
	ChildID     string      `json:"child_id,omitempty"`
}

// ProposedAnswer is one answer the orchestrator read in the person's words.
type ProposedAnswer struct {
	QuestionID string `json:"question_id"`
	Answer     string `json:"answer"`
	Note       string `json:"note,omitempty"`
}

// ProposeInput is propose_answers' argument shape.
type ProposeInput struct {
	From    string           `json:"from"`
	Why     string           `json:"why"`
	Answers []ProposedAnswer `json:"answers"`
}

func (AskTool) Name() string            { return "ask" }
func (SummarizeTool) Name() string      { return "summarize" }
func (ProposeAnswersTool) Name() string { return "propose_answers" }

func (AskTool) Description() string {
	return "Ask the person a design question you cannot settle from the code and the task: the options, the one " +
		"you recommend, and why. It joins the queue of questions and returns its id at once; do not wait for it. " +
		"The answer reaches you as a report at the start of a later turn. Merges are not asked here: the pipeline " +
		"asks those itself."
}

func (SummarizeTool) Description() string {
	return "Write what changed into a question that waits for the person — a merge the pipeline asked, above all: " +
		"two or three lines, in the person's language, of what the child's change does. The evidence beside it is " +
		"the harness's measurement; do not restate it or the child's claims. It answers nothing."
}

func (ProposeAnswersTool) Description() string {
	return "When the person tells you in the conversation how to answer waiting questions — \"merge them all\", " +
		"\"send the lists one back asking for more tests\" — put what you understood in a proposed card: one answer " +
		"per question, each one of its options, with a note where they gave one. The person confirms it with one " +
		"click. This tool never answers: nothing is answered until the person confirms, and you never answer for them."
}

func (AskTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{` +
		`"header":{"type":"string","description":"A short tag of a few words shown above the question."},` +
		`"text":{"type":"string","description":"The question, complete: the person reads it without your conversation."},` +
		`"options":{"type":"array","minItems":2,"items":{"type":"object","properties":{"id":{"type":"string"},"label":{"type":"string"},` +
		`"description":{"type":"string","description":"What choosing it does."}},"required":["id","label"]}},` +
		`"recommended":{"type":"string","description":"The id of the option you recommend."},` +
		`"why":{"type":"string","description":"Why you recommend it; it goes to the decision log."},` +
		`"child_id":{"type":"string","description":"The child the question is about, if one."}},` +
		`"required":["text","options","recommended","why"]}`)
}

func (SummarizeTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{` +
		`"question_id":{"type":"string","description":"The waiting question, as the report named it."},` +
		`"summary":{"type":"string","description":"Two or three lines, in the person's language."}},` +
		`"required":["question_id","summary"]}`)
}

func (ProposeAnswersTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{` +
		`"from":{"type":"string","description":"The person's words, as they wrote them."},` +
		`"why":{"type":"string","description":"How you read them; it goes to the decision log."},` +
		`"answers":{"type":"array","minItems":1,"items":{"type":"object","properties":{` +
		`"question_id":{"type":"string"},"answer":{"type":"string","description":"One of the question's option ids."},` +
		`"note":{"type":"string","description":"What the person said to go with it, such as what to change."}},` +
		`"required":["question_id","answer"]}}},"required":["from","why","answers"]}`)
}

// Declare touches nothing: a question is a record, not a change to the tree.
func (t AskTool) Declare(json.RawMessage) (policy.Request, error) {
	return policy.Request{Tool: t.Name()}, nil
}
func (t SummarizeTool) Declare(json.RawMessage) (policy.Request, error) {
	return policy.Request{Tool: t.Name()}, nil
}
func (t ProposeAnswersTool) Declare(json.RawMessage) (policy.Request, error) {
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

func (t SummarizeTool) Execute(_ context.Context, input json.RawMessage, _ *State) (Result, error) {
	var in struct {
		QuestionID string `json:"question_id"`
		Summary    string `json:"summary"`
	}
	if err := decode(t.Name(), input, &in); err != nil {
		return err.(*ToolError).Result(), nil
	}
	if t.Conductor == nil {
		return noConductor(t.Name()), nil
	}
	if err := t.Conductor.Summarize(in.QuestionID, in.Summary); err != nil {
		return refused(t.Name(), err), nil
	}
	return Result{Output: fmt.Sprintf("The summary is on %s, for the person to read; it is still theirs to answer.", in.QuestionID)}, nil
}

func (t ProposeAnswersTool) Execute(_ context.Context, input json.RawMessage, _ *State) (Result, error) {
	var in ProposeInput
	if err := decode(t.Name(), input, &in); err != nil {
		return err.(*ToolError).Result(), nil
	}
	if t.Conductor == nil {
		return noConductor(t.Name()), nil
	}
	card, err := t.Conductor.ProposeAnswers(in)
	if err != nil {
		return refused(t.Name(), err), nil
	}
	return Result{Output: fmt.Sprintf("Proposed as card %s; nothing is answered until the person confirms it. "+
		"The confirmed answers reach you as reports.", card)}, nil
}
