package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/protocol"
)

// ExecName is the tool a typed command runs through.
//
// Through the tool and not around it, deliberately. `!` is a shortcut past the
// model, never past the sandbox: the command declares its paths, the policy
// evaluates it, and a crossing is put to the person exactly as it would be had
// the model asked. A shell escape that skipped that would be a hole in the one
// boundary this product is built around.
const ExecName = "bash"

// Exec runs a command the PERSON typed, outside any turn.
//
// The output goes two places, and both matter. It reaches the screen as the
// tool events the stream already knows how to draw, and it reaches the history
// as one user message — because the user did run it, and the next turn has to
// know what they saw. Anything else makes the model answer about a workspace it
// cannot see the state of.
//
// It is a user message rather than a tool result on purpose: a tool result with
// no tool call before it is a shape no provider accepts, and inventing an
// assistant call the model never made would put words in its mouth to hold an
// output it did not ask for.
func (e *Engine) Exec(ctx context.Context, command string) (string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", protocol.Errorf(protocol.CodeInvalidInput, "no command to run")
	}
	if e.cfg.Tools == nil {
		return "", protocol.Errorf(protocol.CodeInternal, "this session runs no tools")
	}
	if _, ok := e.cfg.Tools.Get(ExecName); !ok {
		return "", protocol.Errorf(protocol.CodeInvalidInput,
			"this session has no %s tool, so there is nothing to run a command with", ExecName)
	}

	input, err := json.Marshal(map[string]string{"command": command})
	if err != nil {
		return "", err
	}

	// Its own turn id. The events are the ones the stream already draws, and
	// they have to belong somewhere; borrowing the last turn's id would attach
	// this to work it had no part in, and undo reads turns.
	e.turnSeq++
	turnID := fmt.Sprintf("x%d", e.turnSeq)

	msgs, _ := e.execute(ctx, turnID, []ce.ToolCall{{
		ID: turnID + "-1", Name: ExecName, Input: input,
	}})

	out := ""
	for _, m := range msgs {
		if m.ToolResult != nil {
			out += m.ToolResult.Output
		}
	}
	e.session.History = append(e.session.History, ce.Message{
		Role: ce.RoleUser,
		Text: "I ran `" + command + "` myself. It printed:\n\n" + out,
	})
	return out, nil
}
