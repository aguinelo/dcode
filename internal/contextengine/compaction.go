package contextengine

import "strings"

// Estimate approximates the token count of msgs.
//
// Deliberately a character heuristic and not a real tokenizer: the trigger is a
// fraction of the window with a safety margin, so precision is not needed. What
// is needed is determinism — an estimate that drifts between runs makes the
// compaction golden tests flap.
func Estimate(msgs []Message, cfg Config) int {
	cfg = withDefaults(cfg)
	chars := 0
	for _, m := range msgs {
		chars += len(m.Role) + len(m.Text)
		for _, c := range m.ToolCalls {
			chars += len(c.ID) + len(c.Name) + len(c.Input)
		}
		if m.ToolResult != nil {
			chars += len(m.ToolResult.ToolCallID) + len(m.ToolResult.Output)
		}
		// An image costs context, and a budget that does not count it drifts
		// silently: the band reads 60% while the window is nearly full, and
		// compaction arrives after the model has lost the thread.
		//
		// A flat cost per image, not a share of its bytes. A model prices an
		// image by what it sees; how well the PNG compressed is the one thing
		// that has nothing to do with it.
		chars += len(m.Images) * charsPerImage
	}
	base := float64(chars) / cfg.CharsPerToken
	return int(base * (1 + cfg.Margin))
}

// charsPerImage is what one image is counted as, in characters, before the
// same divisor every other input goes through.
//
// Deliberately coarse. Providers price images by tiles and by detail setting,
// and reproducing that arithmetic here would be a second implementation of
// somebody else's billing — wrong in a different way each time it changed. What
// this has to get right is the ORDER: that an image is worth a page of text
// rather than a word, so a session full of screenshots compacts before it
// overflows instead of after.
const charsPerImage = 6000

// CompactionPlan describes the span to replace with a single summary.
// FromIdx is inclusive, ToIdx exclusive, both indices into Session.History.
type CompactionPlan struct {
	FromIdx int
	ToIdx   int
}

// Plan decides whether to compact and where to cut. Pure: it only decides, and
// the caller generates the summary text, because that needs a model call and
// would drag I/O into this package.
//
// Two cuts are never made:
//
//   - inside a turn, splitting an assistant message from its tool results —
//     that produces history no provider will accept;
//   - at or after the most recent user message — the current task always
//     survives by construction, not by summary quality.
func Plan(s Session, cfg Config) (CompactionPlan, bool) {
	cfg = withDefaults(cfg)
	if cfg.Window <= 0 {
		return CompactionPlan{}, false
	}

	msgs, err := Assemble(s)
	if err != nil {
		return CompactionPlan{}, false
	}
	if float64(Estimate(msgs, cfg)) < cfg.CompactAt*float64(cfg.Window) {
		return CompactionPlan{}, false
	}

	from := 0
	if s.Summary != nil {
		from = s.Summary.UpToIdx
	}

	limit := protectedFrom(s.History, cfg.KeepTurns)
	to := turnBoundaryAtOrBefore(s.History, limit)
	if to <= from {
		return CompactionPlan{}, false
	}
	return CompactionPlan{FromIdx: from, ToIdx: to}, true
}

// protectedFrom returns the first index that must not be compacted: the most
// recent user message, walked back a further KeepTurns user messages.
func protectedFrom(h []Message, keepTurns int) int {
	if keepTurns < 0 {
		keepTurns = 0
	}
	seen := 0
	for i := len(h) - 1; i >= 0; i-- {
		if h[i].Role != RoleUser {
			continue
		}
		if seen == keepTurns {
			return i
		}
		seen++
	}
	return 0
}

// turnBoundaryAtOrBefore walks back from idx to the first index that is a clean
// turn boundary — one where no assistant message above it is still waiting for
// tool results at or after it.
func turnBoundaryAtOrBefore(h []Message, idx int) int {
	if idx > len(h) {
		idx = len(h)
	}
	for i := idx; i > 0; i-- {
		if isCleanBoundary(h, i) {
			return i
		}
	}
	return 0
}

// isCleanBoundary reports whether cutting before index i leaves every assistant
// tool call in h[:i] paired with its results in h[:i].
func isCleanBoundary(h []Message, i int) bool {
	pending := map[string]bool{}
	for _, m := range h[:i] {
		for _, c := range m.ToolCalls {
			pending[c.ID] = true
		}
		if m.ToolResult != nil {
			delete(pending, m.ToolResult.ToolCallID)
		}
	}
	return len(pending) == 0
}

// Apply returns the session with plan applied and text as the new summary. The
// original History slice is not mutated: append-only means the caller's view
// stays valid.
func Apply(s Session, plan CompactionPlan, text string) Session {
	prev := ""
	if s.Summary != nil {
		prev = s.Summary.Text
	}
	merged := text
	if prev != "" && strings.TrimSpace(text) != "" {
		merged = prev + "\n\n" + text
	} else if prev != "" {
		merged = prev
	}
	s.Summary = &Summary{Text: merged, UpToIdx: plan.ToIdx}
	return s
}

func withDefaults(cfg Config) Config {
	d := DefaultConfig()
	if cfg.CompactAt <= 0 {
		cfg.CompactAt = d.CompactAt
	}
	if cfg.CharsPerToken <= 0 {
		cfg.CharsPerToken = d.CharsPerToken
	}
	// Zero is treated as unset rather than as a deliberate choice. A zero
	// margin is not expressible on purpose: the margin exists to absorb the
	// heuristic's error, and removing it trades "compacts slightly early" for
	// "overruns the window", which is the worse failure of the two.
	if cfg.Margin <= 0 {
		cfg.Margin = d.Margin
	}
	return cfg
}
