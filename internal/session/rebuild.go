package session

import (
	"bufio"
	"encoding/json"
	"os"
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
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []ce.Message

	// The assistant's turn is one message carrying its text and its calls, so
	// it is held open until something forces it out: a result to report, or the
	// next question.
	var text strings.Builder
	var calls []ce.ToolCall
	// answered marks a call that got a result. A call without one is a turn
	// that was interrupted, and sending it would be a conversation the model
	// cannot answer — most providers reject it outright.
	answered := map[string]bool{}

	flush := func() {
		kept := calls[:0]
		for _, c := range calls {
			if answered[c.ID] {
				kept = append(kept, c)
			}
		}
		if text.Len() == 0 && len(kept) == 0 {
			text.Reset()
			calls = nil
			return
		}
		out = append(out, ce.Message{
			Role:      ce.RoleAssistant,
			Text:      strings.TrimSpace(text.String()),
			ToolCalls: append([]ce.ToolCall(nil), kept...),
		})
		text.Reset()
		calls = nil
	}

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		var ev protocol.Event
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue
		}
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
				text.WriteString(d.Text)
			}

		case protocol.EventToolRequested:
			var d protocol.ToolRequested
			if json.Unmarshal(ev.Payload, &d) == nil {
				calls = append(calls, ce.ToolCall{ID: d.ToolCallID, Name: d.Name, Input: d.Input})
			}

		case protocol.EventToolCompleted:
			var d protocol.ToolCompleted
			if json.Unmarshal(ev.Payload, &d) != nil {
				continue
			}
			answered[d.ToolCallID] = true
			// The message that made the calls goes out first: a result
			// referring to a call the model has not seen is out of order.
			flush()
			out = append(out, ce.Message{Role: ce.RoleTool, ToolResult: &ce.ToolResult{
				ToolCallID: d.ToolCallID,
				Output:     d.Output,
				IsError:    !d.OK,
			}})
		}
	}
	flush()
	return out, sc.Err()
}
