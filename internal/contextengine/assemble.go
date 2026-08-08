package contextengine

import (
	"errors"
	"strings"
)

// ErrNoInstructions is returned when a session carries no system prompt. An
// agent with no doctrine is not a degraded agent, it is an unpredictable one.
var ErrNoInstructions = errors.New("contextengine: session has no instructions")

// Assemble builds the message list for one model call.
//
// The order is fixed, most stable first, so the provider can match the longest
// possible cached prefix:
//
//  1. system prompt  — never changes within a session
//  2. tool defs      — frozen at creation
//  3. summary        — changes only on compaction
//  4. live history   — append-only
//
// Blocks 1 to 3 collapse into a single system message: they are one immutable
// unit, and splitting them would let a provider that keys the cache on message
// boundaries miss on a summary change alone.
//
// Assemble is pure. Calling it twice with the same Session yields byte-identical
// output, which is what makes golden testing exact.
func Assemble(s Session) ([]Message, error) {
	if strings.TrimSpace(s.Instructions) == "" {
		return nil, ErrNoInstructions
	}

	var b strings.Builder
	b.WriteString(s.Instructions)

	if len(s.Tools) > 0 {
		b.WriteString("\n\n")
		b.WriteString(renderTools(s.Tools))
	}
	// A nil summary contributes nothing at all — not an empty marker, which
	// would still be a byte difference against a session that never compacted.
	if s.Summary != nil {
		b.WriteString("\n\n")
		b.WriteString(renderSummary(*s.Summary))
	}

	live := s.liveHistory()
	out := make([]Message, 0, len(live)+1)
	out = append(out, Message{Role: RoleSystem, Text: b.String()})
	out = append(out, live...)
	return out, nil
}

// liveHistory returns the slice of History not covered by the summary.
func (s Session) liveHistory() []Message {
	from := 0
	if s.Summary != nil {
		from = s.Summary.UpToIdx
	}
	if from < 0 {
		from = 0
	}
	if from > len(s.History) {
		from = len(s.History)
	}
	return s.History[from:]
}

// renderTools writes the tool block. Order follows the slice, which the caller
// freezes at session creation; sorting here would hide an unstable caller.
func renderTools(tools []ToolDef) string {
	var b strings.Builder
	b.WriteString("## Tools\n")
	for _, t := range tools {
		b.WriteString("\n### ")
		b.WriteString(t.Name)
		b.WriteString("\n")
		if t.Description != "" {
			b.WriteString(t.Description)
			b.WriteString("\n")
		}
		if len(t.Schema) > 0 {
			b.WriteString("Schema: ")
			b.Write(t.Schema)
			b.WriteString("\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderSummary(s Summary) string {
	return "## Earlier in this session\n\n" + s.Text
}
