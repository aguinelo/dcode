package provider

import "strings"

// claudeModel is what the Claude family knows about one line of models.
type claudeModel struct {
	prefix string
	// output is the largest max_tokens the API accepts for the model, as
	// Anthropic publishes it.
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
	{"claude-fable-", 128_000},
	{"claude-mythos-", 128_000},
	{"claude-opus-5", 128_000},
	{"claude-opus-4-8", 128_000},
	{"claude-opus-4-7", 128_000},
	{"claude-opus-4-6", 128_000},
	{"claude-opus-4-5", 64_000},
	{"claude-opus-4-1", 32_000},
	{"claude-opus-4-0", 32_000},
	{"claude-opus-4-2025", 32_000},
	{"claude-sonnet-5", 128_000},
	{"claude-sonnet-4-6", 128_000},
	{"claude-sonnet-4-5", 64_000},
	{"claude-sonnet-4-0", 64_000},
	{"claude-sonnet-4-2025", 64_000},
	{"claude-haiku-5", 128_000},
	{"claude-haiku-4-5", 64_000},
	{"claude-3-haiku", 4_096},
}

// claudeDefaultOutput is the ceiling for a claude-* model the table does not
// name: the smallest any current Opus allows, which every model released
// since claude-opus-4 accepts.
const claudeDefaultOutput = 32_000

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
