package tools

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/aguinelo/dcode/internal/policy"
)

// The verdicts a reviewer can give. Missing is not one of them: it is what the
// harness records when the review ended without a verdict, and a model cannot
// say it of itself.
const (
	VerdictApprove = "approve"
	VerdictChanges = "changes_requested"
)

// Verdict is how a reviewer ends a review: approve, or changes_requested with
// the findings. A tool, and not prose, so that "looks good" in a sentence is
// never read as an approval — a review that ends without this call is
// missing, which the harness never reads as approve.
//
// It is in the reviewer's registry alone, beside the tools that only read.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §7.2, §10.
type Verdict struct {
	// Submit records the verdict for the review this session is running. It
	// refuses a second one and one that does not hold together; the refusal
	// comes back to the model to correct. Injected so this package does not
	// import the orchestrator, which imports this one.
	Submit func(ctx context.Context, in VerdictInput) (string, error)
}

// VerdictInput is the argument shape.
type VerdictInput struct {
	Verdict  string           `json:"verdict"`
	Findings []VerdictFinding `json:"findings,omitempty"`
	// Why is the reviewer's reason, which goes to the decision log as is.
	Why string `json:"why"`
}

// VerdictFinding is one defect, where it is and why it is one.
type VerdictFinding struct {
	Path string `json:"path"`
	Line int    `json:"line,omitempty"`
	What string `json:"what"`
	Why  string `json:"why"`
}

func (Verdict) Name() string { return "verdict" }

func (Verdict) Description() string {
	return "End the review with your verdict on the commit you were given. This is the only way a review " +
		"ends: a review that finishes without calling it counts as no review, never as an approval.\n\n" +
		"`approve` when the diff does what the task asks, inside what the task owns, and the specs it names " +
		"still hold. `changes_requested` when it does not, with one finding per defect: the path, the line " +
		"when there is one, what is wrong, and why it is wrong. Each finding goes back to the child, who " +
		"first writes a test that fails because of it, so say it so a test can be written from it. " +
		"Call it once, after you have read what you need."
}

func (Verdict) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{` +
		`"verdict":{"type":"string","enum":["approve","changes_requested"],"description":"approve, or changes_requested with findings."},` +
		`"findings":{"type":"array","description":"One per defect; required with changes_requested, absent with approve.",` +
		`"items":{"type":"object","properties":{` +
		`"path":{"type":"string","description":"The file, relative to the worktree."},` +
		`"line":{"type":"integer","description":"The line, when the defect has one."},` +
		`"what":{"type":"string","description":"What is wrong."},` +
		`"why":{"type":"string","description":"Why it is wrong: the task, the spec or the behaviour it breaks."}},` +
		`"required":["path","what","why"]}},` +
		`"why":{"type":"string","description":"Your reason for the verdict, in one or two sentences."}},` +
		`"required":["verdict","why"]}`)
}

// Declare touches nothing: a verdict is a record for the harness, not a change
// to the tree.
func (v Verdict) Declare(input json.RawMessage) (policy.Request, error) {
	var in VerdictInput
	if err := decode(v.Name(), input, &in); err != nil {
		return policy.Request{}, err
	}
	return policy.Request{Tool: v.Name()}, nil
}

func (v Verdict) Execute(ctx context.Context, input json.RawMessage, _ *State) (Result, error) {
	var in VerdictInput
	if err := decode(v.Name(), input, &in); err != nil {
		return err.(*ToolError).Result(), nil
	}
	if v.Submit == nil {
		return errf(v.Name(), CodeDenied, "", "this session runs no review, so there is no verdict to give").Result(), nil
	}
	in.Verdict = strings.TrimSpace(in.Verdict)
	out, err := v.Submit(ctx, in)
	if err != nil {
		return errf(v.Name(), CodeBadInput, "", "%s", err.Error()).Result(), nil
	}
	return Result{Output: out}, nil
}
