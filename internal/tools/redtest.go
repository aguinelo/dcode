package tools

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/aguinelo/dcode/internal/policy"
)

// RedTest is how a child in its red phase names the test that shows the
// defect a reviewer found. The harness commits the turn, runs the command on
// that commit — before any fix — and takes the fix only once the test fails
// there (red_test.confirmed). Outside the red phase it refuses, and says why.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §7.3, §10.
type RedTest struct {
	// Submit hands the command to the pipeline of this child. Injected so
	// this package does not import the orchestrator, which imports this one.
	Submit func(ctx context.Context, command string) (string, error)
}

// RedTestInput is the argument shape.
type RedTestInput struct {
	Command string `json:"command"`
}

func (RedTest) Name() string { return "red_test" }

func (RedTest) Description() string {
	return "Only in the red phase, after a reviewer asked for changes: name the shell command that runs the test you " +
		"just wrote to show the defect. Call it once you have written the test and before fixing anything, then end " +
		"your turn. The harness commits your test, runs this command on that commit, and accepts it only if it " +
		"fails there; a test that passes does not show the defect. Outside the red phase it is refused."
}

func (RedTest) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{` +
		`"command":{"type":"string","description":"The command that runs the failing test, from the worktree's root, e.g. go test ./internal/x -run TestY."}},` +
		`"required":["command"]}`)
}

// Declare touches nothing: the command runs in the harness's throwaway tree,
// never in this session's shell.
func (t RedTest) Declare(input json.RawMessage) (policy.Request, error) {
	var in RedTestInput
	if err := decode(t.Name(), input, &in); err != nil {
		return policy.Request{}, err
	}
	return policy.Request{Tool: t.Name()}, nil
}

func (t RedTest) Execute(ctx context.Context, input json.RawMessage, _ *State) (Result, error) {
	var in RedTestInput
	if err := decode(t.Name(), input, &in); err != nil {
		return err.(*ToolError).Result(), nil
	}
	if t.Submit == nil {
		return errf(t.Name(), CodeDenied, "", "this session is no child of an orchestrator, so it has no red phase").Result(), nil
	}
	out, err := t.Submit(ctx, strings.TrimSpace(in.Command))
	if err != nil {
		return errf(t.Name(), CodeBadInput, "", "%s", err.Error()).Result(), nil
	}
	return Result{Output: out}, nil
}
