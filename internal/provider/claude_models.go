package provider

import "strings"

// claudeModel is what the Claude family knows about one line of models.
type claudeModel struct {
	prefix string
	// window is the context window, and output the largest max_tokens the
	// API accepts, both as Anthropic publishes them.
	window int
	output int
}

// claudeModels is matched by the longest prefix, so a dated id
// (claude-haiku-4-5-20251001) finds its alias's row, and a row for a line
// (claude-opus-5) covers its point releases (claude-opus-5-5).
//
// Published limits as of 2026-10. A model that is not here gets
// claudeDefaultOutput, never the largest number in the table: a ceiling above
// what the model allows is a request the API refuses outright, and a smaller
// one is only an answer that stops early and says so.
var claudeModels = []claudeModel{
	{"claude-fable-", 1_000_000, 128_000},
	{"claude-mythos-", 1_000_000, 128_000},
	{"claude-opus-5", 1_000_000, 128_000},
	{"claude-opus-4-8", 1_000_000, 128_000},
	{"claude-opus-4-7", 1_000_000, 128_000},
	{"claude-opus-4-6", 1_000_000, 128_000},
	{"claude-opus-4-5", 200_000, 64_000},
	{"claude-opus-4-1", 200_000, 32_000},
	{"claude-opus-4-0", 200_000, 32_000},
	{"claude-opus-4-2025", 200_000, 32_000},
	{"claude-sonnet-5", 1_000_000, 128_000},
	{"claude-sonnet-4-6", 1_000_000, 128_000},
	{"claude-sonnet-4-5", 200_000, 64_000},
	{"claude-sonnet-4-0", 200_000, 64_000},
	{"claude-sonnet-4-2025", 200_000, 64_000},
	{"claude-haiku-5", 1_000_000, 128_000},
	{"claude-haiku-4-5", 200_000, 64_000},
	{"claude-3-haiku", 200_000, 4_096},
}

// claudeDefaultOutput is the ceiling for a claude-* model the table does not
// name: the smallest any current Opus allows, which every model released
// since claude-opus-4 accepts.
const claudeDefaultOutput = 32_000

// claudeDefaultWindow is the window for a claude-* model the table does not
// name. Too small compacts early; too large is a request the provider
// refuses in the middle of a turn, which is the worse of the two.
const claudeDefaultWindow = 200_000

// lookupClaude finds the row for model by longest prefix.
func lookupClaude(model string) (claudeModel, bool) {
	var best claudeModel
	found := false
	for _, m := range claudeModels {
		if strings.HasPrefix(model, m.prefix) && len(m.prefix) > len(best.prefix) {
			best, found = m, true
		}
	}
	return best, found
}

// Window is the context window of model, from the same table.
//
// It was a fixed 200000. Every model since claude-opus-4-6 holds a million,
// so a session was compacted at a fifth of what it could keep.
func (Claude) Window(model string) (int, error) {
	if m, ok := lookupClaude(model); ok {
		return m.window, nil
	}
	return claudeDefaultWindow, nil
}

// OutputCeiling is the max_tokens a request for model carries when the
// configuration names none.
//
// The Anthropic dialect requires the field, and the number that used to fill
// it was a fixed 8192 — a sixteenth of what most current models allow, so a
// long answer was cut at a length nobody chose.
func (Claude) OutputCeiling(model string) int {
	if m, ok := lookupClaude(model); ok {
		return m.output
	}
	return claudeDefaultOutput
}
