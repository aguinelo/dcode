package loop

import (
	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
)

// refuse keeps a call the provider refused before it ran (tool_schema: a tool
// this session does not declare, or arguments that are not JSON) and says so.
//
// It went into the history as a tool result, with no call id and no call above
// it: the refused call never reached the answer, so there was nothing for the
// result to answer. MiniMax-M3 rejects that request outright — `tool result's
// tool id() not found` — and the session could not take another turn. A session
// continuing an orchestrator's conversation is where it showed: the model reads
// calls to `delegate` and makes one, and this session declares no `delegate`.
//
// So the refusal reaches the model as a note after its answer instead, which
// every provider accepts mid-conversation. And it is said on the stream: the
// call the model asked for did not run, and nothing on the screen said so.
func (e *Engine) refuse(msg string) {
	e.refused = append(e.refused, msg)
	e.emit(protocol.EventSessionError, protocol.Error{
		Code: string(provider.ErrClassToolSchema), Message: msg,
	})
}

// takeRefused returns the refusals of the stream just read, as the notes the
// model is sent, and forgets them.
func (e *Engine) takeRefused() []ce.Message {
	var out []ce.Message
	for _, msg := range e.refused {
		out = append(out, ce.Message{
			Role: ce.RoleUser, Reminder: true,
			Text: "Your tool call was refused before it ran, so it did nothing: " + msg,
		})
	}
	e.refused = nil
	return out
}
