package session

import (
	"encoding/json"
	"strings"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/protocol"
)

// Rebuild turns a record back into the conversation the model was sent.
//
// The record is the only copy: nothing else survives the session, and storing
// the history a second time would be a second copy to drift from the first —
// which this codebase has now found four separate times.
//
// It does not compact. The engine checks for compaction at the top of its first
// iteration, before any request, so a seeded history that is too large is
// handled by the same code that handles one that grew — and re-running it here
// would be a second implementation of the thing most worth having only one of.
//
// What cannot be rebuilt is left out rather than guessed: reminders the harness
// appended, approvals granted in a moment that has passed, and processes that
// died with their session. All three are re-asked or re-derived by the turn
// that follows.
func Rebuild(path string) ([]ce.Message, error) {
	// Through Carry, which FOLLOWS THE CHAIN.
	//
	// A record holds the marker naming the conversation it continues and not
	// that conversation's events, so reading one means walking back. Reading
	// the file directly would rebuild only the most recent leg and hand the
	// model a conversation that starts in the middle of itself.
	evs, _, err := Carry(path)
	if err != nil {
		return nil, err
	}
	return rebuildFrom(evs)
}

func rebuildFrom(evs []protocol.Event) ([]ce.Message, error) {
	var out []ce.Message

	// The assistant's turn is one message carrying its text and its calls, so
	// it is held open until something forces it out: the model heard from
	// again after its calls were answered, or the next question.
	var text strings.Builder
	var calls []ce.ToolCall
	// answered holds the result each open call got. A call without one is a
	// turn that was interrupted, and sending it would be a conversation the
	// model cannot answer — most providers reject it outright.
	//
	// Held until the batch is over, never written out at its first result. The
	// loop runs independent calls together and they finish in any order, so
	// the first result is not the last: writing on it sent the message out
	// with only the calls answered so far, and every later result answered a
	// call no message carried — which providers reject just the same.
	//
	// Scoped to the open message, not to the record: a call id is unique within
	// one reply and not across a chain. A typed command's numbering starts
	// again in every leg, and an OpenAI-compatible server mints its own.
	answered := map[string]ce.ToolResult{}

	flush := func() {
		var kept []ce.ToolCall
		for _, c := range calls {
			if _, ok := answered[c.ID]; ok {
				kept = append(kept, c)
			}
		}
		if text.Len() > 0 || len(kept) > 0 {
			out = append(out, ce.Message{
				Role:      ce.RoleAssistant,
				Text:      strings.TrimSpace(text.String()),
				ToolCalls: kept,
			})
		}
		// In the order the calls were made, not the order they finished. The
		// live session appends a batch's results at the index the model
		// emitted each call at (RN-3 of the agent loop), while the record
		// holds them as they completed. Following the calls rebuilds the
		// conversation the model was sent, and keeps it from depending on
		// which call won a race.
		for _, c := range kept {
			r := answered[c.ID]
			out = append(out, ce.Message{Role: ce.RoleTool, ToolResult: &r})
		}
		text.Reset()
		calls = nil
		clear(answered)
	}
	// Text or a call arriving after results is the model's next reply, and
	// that is what closes the batch before it.
	replying := func() {
		if len(answered) > 0 {
			flush()
		}
	}

	for _, ev := range evs {
		switch ev.Type {
		case protocol.EventTurnStarted:
			var d protocol.TurnStarted
			if json.Unmarshal(ev.Payload, &d) == nil && d.Text != "" {
				flush()
				out = append(out, ce.Message{Role: ce.RoleUser, Text: d.Text})
			}

		case protocol.EventMessageDelta:
			var d protocol.MessageDelta
			if json.Unmarshal(ev.Payload, &d) == nil {
				replying()
				text.WriteString(d.Text)
			}

		case protocol.EventToolRequested:
			var d protocol.ToolRequested
			if json.Unmarshal(ev.Payload, &d) == nil {
				replying()
				calls = append(calls, ce.ToolCall{ID: d.ToolCallID, Name: d.Name, Input: d.Input})
			}

		case protocol.EventToolCompleted:
			var d protocol.ToolCompleted
			if json.Unmarshal(ev.Payload, &d) == nil {
				answered[d.ToolCallID] = ce.ToolResult{
					ToolCallID: d.ToolCallID,
					Output:     d.Output,
					IsError:    !d.OK,
				}
			}
		}
	}
	flush()
	return out, nil
}
