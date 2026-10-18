package provider

import (
	"fmt"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

// CheckToolPairs refuses a conversation whose tool calls and tool results do
// not pair the way the dialects require: every tool result answers, by a
// non-empty id, a call of the assistant message its run of results follows,
// and every call of that message is answered before anything else is said.
//
// It is the floor under the loop and the replay, which are where pairs are
// made. A request that breaks it is one the provider rejects anyway — MiniMax
// with `tool result's tool id() not found`, which names neither the message nor
// the call — so it fails here, before it is sent, naming the message that
// broke it. Classed as a bad request: the same history sent again is the same
// history refused, and a retry would only hide that.
func CheckToolPairs(msgs []ce.Message) error {
	open := map[string]bool{}
	inRun := false
	for i, m := range msgs {
		if m.Role == ce.RoleTool {
			if m.ToolResult == nil {
				// Carries nothing, and no dialect sends it.
				continue
			}
			if m.ToolResult.ToolCallID == "" {
				return unpaired(i, "is a tool result with no call id")
			}
			if !inRun || !open[m.ToolResult.ToolCallID] {
				return unpaired(i, fmt.Sprintf(
					"answers call %q, which the assistant message before it did not make", m.ToolResult.ToolCallID))
			}
			delete(open, m.ToolResult.ToolCallID)
			continue
		}
		if len(open) > 0 {
			return unpaired(i, fmt.Sprintf(
				"comes before %d call(s) of the assistant message above it were answered", len(open)))
		}
		inRun = false
		if m.Role != ce.RoleAssistant || len(m.ToolCalls) == 0 {
			continue
		}
		for _, c := range m.ToolCalls {
			if c.ID == "" {
				return unpaired(i, fmt.Sprintf("makes a call to %q with no id", c.Name))
			}
			if open[c.ID] {
				return unpaired(i, fmt.Sprintf("makes two calls with the id %q", c.ID))
			}
			open[c.ID] = true
		}
		inRun = true
	}
	if len(open) > 0 {
		return unpaired(len(msgs)-1, fmt.Sprintf(
			"ends the conversation with %d call(s) nobody answered", len(open)))
	}
	return nil
}

func unpaired(i int, what string) error {
	return &ProviderError{
		Class: ErrClassBadRequest,
		Message: fmt.Sprintf(
			"refusing to send a conversation the provider would reject: message %d %s", i, what),
	}
}
