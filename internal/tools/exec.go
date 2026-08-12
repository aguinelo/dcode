package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
)

// Runner executes a command inside the sandbox. The tool never reaches for
// os/exec directly: every execution goes through the boundary, and an
// interface is what makes that structural instead of a rule to remember.
type Runner interface {
	Run(ctx context.Context, workdir, command string) (stdout string, exitCode int, err error)
}

// Bash runs a shell command.
type Bash struct {
	Runner  Runner
	Workdir string
	Timeout time.Duration
}

// BashInput is the argument shape.
type BashInput struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout_seconds,omitempty"`
}

func (Bash) Name() string { return "bash" }

func (Bash) Description() string {
	return "Run a shell command in the workspace. " +
		"Use the dedicated tools for reading, searching and editing files, and `glob` for " +
		"listing or locating them — they are cheaper and need fewer permissions. " +
		"A non-zero exit is a result, not a failure; read the output and decide."
}

func (Bash) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{` +
		`"command":{"type":"string"},` +
		`"timeout_seconds":{"type":"integer"}},` +
		`"required":["command"]}`)
}

func (b Bash) Declare(input json.RawMessage) (policy.Request, error) {
	var in BashInput
	if err := decode(b.Name(), input, &in); err != nil {
		return policy.Request{}, err
	}
	// A shell command is opaque, so the worst case is what gets declared. A
	// build resolves dependencies, a test suite pulls an image, a formatter
	// checks for a newer version: there is no reading of the string that
	// answers whether this one reaches out.
	//
	// This used to be declared only when the sandbox already permitted the
	// network, and the reasoning was sound at the time: approving granted
	// nothing, because the OS blocked it either way, so the answer did not do
	// what the question said. That premise is gone. The sandbox now asks per
	// command and a grant opens it, so consent means what it appears to mean —
	// and the question is asked once per project rather than once per command.
	return policy.Request{
		Tool:    b.Name(),
		Command: in.Command,
		Network: true,
		Paths:   []policy.Access{{Path: ".", Write: true}},
	}, nil
}

func (b Bash) Execute(ctx context.Context, input json.RawMessage, s *State) (Result, error) {
	var in BashInput
	if err := decode(b.Name(), input, &in); err != nil {
		return err.(*ToolError).Result(), nil
	}
	if strings.TrimSpace(in.Command) == "" {
		return errf(b.Name(), CodeBadInput, "", "command is required").Result(), nil
	}
	if b.Runner == nil {
		return errf(b.Name(), CodeDenied, "",
			"no sandbox is available, so no command can run").Result(), nil
	}

	timeout := b.Timeout
	if in.Timeout > 0 {
		timeout = time.Duration(in.Timeout) * time.Second
	}
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	out, code, err := b.Runner.Run(runCtx, b.Workdir, in.Command)
	out = clamp(out, s.Limits.MaxToolOutput)

	if err != nil && errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		// Whatever was produced before the cut still helps the model decide,
		// so it is returned rather than discarded.
		return Result{
			Output:    fmt.Sprintf("%s\n\n(timed out after %s)", out, timeout),
			IsError:   true,
			Code:      CodeTimeout,
			Truncated: true,
		}, nil
	}
	if err != nil && code < 0 {
		return errf(b.Name(), CodeDenied, "", "could not run the command: %v", err).Result(), nil
	}

	// A non-zero exit is data. Turning it into a tool error would hide the
	// exact signal the model needs to decide whether to retry, change approach
	// or report back.
	body := out
	if strings.TrimSpace(body) == "" {
		body = "(no output)"
	}
	return Result{
		Output: fmt.Sprintf("exit %d\n%s", code, body),
		Meta:   Meta{ExitCode: code, HasExit: true, Lines: countLines(out)},
	}, nil
}

// countLines counts non-empty output lines, which is what a reader means by
// "how much did that print".
func countLines(s string) int {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func clamp(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + fmt.Sprintf("\n… output truncated at %d bytes", max)
}

// ---------- plan ----------

// Plan maintains the session plan. It touches no file and runs nothing: it
// changes session state only, which is why its policy verdict is always allow.
type Plan struct{}

// PlanInput is the argument shape.
type PlanInput struct {
	Items []protocol.PlanItem `json:"items"`
}

func (Plan) Name() string { return "plan" }

func (Plan) Description() string {
	return "Record the execution plan, before you start work that touches more than one file. " +
		"Replaces the plan entirely on each call. " +
		"Keep it proportional: a one-line fix needs one item, a cross-file change needs several. " +
		"Exactly one item may be active. Mark an item blocked, with a reason, " +
		"rather than done when it could not be finished."
}

func (Plan) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"items":{"type":"array","items":{` +
		`"type":"object","properties":{` +
		`"id":{"type":"integer"},"text":{"type":"string"},` +
		`"status":{"type":"string","enum":["pending","active","done","blocked"]},` +
		`"blocked":{"type":"string","description":"Why it is blocked; required when status is blocked."}},` +
		`"required":["id","text","status"]}}},"required":["items"]}`)
}

func (p Plan) Declare(json.RawMessage) (policy.Request, error) {
	// No paths, no network. The verdict is always allow.
	return policy.Request{Tool: p.Name()}, nil
}

func (p Plan) Execute(_ context.Context, input json.RawMessage, s *State) (Result, error) {
	var in PlanInput
	if err := decode(p.Name(), input, &in); err != nil {
		return err.(*ToolError).Result(), nil
	}

	active := 0
	for _, it := range in.Items {
		switch it.Status {
		case protocol.PlanPending, protocol.PlanDone:
		case protocol.PlanActive:
			active++
		case protocol.PlanBlocked:
			if strings.TrimSpace(it.Blocked) == "" {
				// A blocked item with no reason is worse than no block at all:
				// the panel shows a stop with nothing to act on.
				return errf(p.Name(), CodePlanNoReason,
					"Say what is blocking it, in a few words.",
					"item %d is blocked but gives no reason", it.ID).Result(), nil
			}
		default:
			return errf(p.Name(), CodeBadInput,
				"Use pending, active, done or blocked.",
				"item %d has unknown status %q", it.ID, it.Status).Result(), nil
		}
	}
	if active > 1 {
		return errf(p.Name(), CodePlanMultiActive,
			"Mark one item active and leave the rest pending.",
			"%d items are active; only one may be", active).Result(), nil
	}

	// The plan is only replaced once it validates, so a rejected call leaves
	// the previous plan exactly as it was.
	s.setPlan(in.Items)

	done := 0
	blocked := 0
	for _, it := range in.Items {
		switch it.Status {
		case protocol.PlanDone:
			done++
		case protocol.PlanBlocked:
			blocked++
		}
	}
	msg := fmt.Sprintf("plan updated: %d item(s), %d done", len(in.Items), done)
	if blocked > 0 {
		msg += fmt.Sprintf(", %d blocked", blocked)
	}
	return Result{Output: msg}, nil
}

// LocalRunner executes through a plain shell. It exists for the development
// entry point only; the sandbox package supplies the real one.
type LocalRunner struct{}

// Run executes command with sh.
func (LocalRunner) Run(ctx context.Context, workdir, command string) (string, int, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return string(out), ee.ExitCode(), nil
		}
		return string(out), -1, err
	}
	return string(out), 0, nil
}
