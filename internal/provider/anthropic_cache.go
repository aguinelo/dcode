package provider

// anText is a text block on its own, which is how the system prompt travels
// once it carries a cache breakpoint.
type anText struct {
	Type         string          `json:"type"`
	Text         string          `json:"text"`
	CacheControl *anCacheControl `json:"cache_control,omitempty"`
}

// anCacheControl is the dialect's prompt-cache breakpoint.
type anCacheControl struct {
	Type string `json:"type"`
}

// ephemeral is the only kind of breakpoint dcode sends: the default lifetime,
// which a loop running round after round keeps warm by reading it.
func ephemeral() *anCacheControl { return &anCacheControl{Type: "ephemeral"} }

// markCache places the prompt-cache breakpoints on a request: at most three,
// under the four the API accepts.
//
// The provider renders tools, then system, then messages, and caches the
// prefix up to each marker. So the markers sit where the prefix stops being
// stable:
//
//   - the last tool, which caches every tool definition — the set is fixed at
//     session start, so this one is read on every round;
//   - the last system block, which caches tools and system together;
//   - the last block of the last message, which caches the whole conversation
//     so far. The next round re-sends it unchanged, append-only context being
//     what it is, and reads it back instead of paying for it.
//
// Nothing here edits a block that was already sent: a marker is not part of
// the cached bytes, so moving it to the new last block each round keeps every
// earlier prefix readable.
func markCache(body *anRequest, system string) {
	if n := len(body.Tools); n > 0 {
		body.Tools[n-1].CacheControl = ephemeral()
	}
	if system != "" {
		body.System = []anText{{Type: "text", Text: system, CacheControl: ephemeral()}}
	}
	if n := len(body.Messages); n > 0 {
		last := &body.Messages[n-1]
		if k := len(last.Content); k > 0 {
			last.Content[k-1].CacheControl = ephemeral()
		}
	}
}

// usage combines what message_start said about the prompt with what the
// closing frame says.
//
// The closing frame's numbers win where it has them — it reports the final
// output count, and a provider that repeats the prompt counts there reports
// the same totals — and message_start fills the rest. Reading the closing
// frame alone was what kept every cache number at zero.
func (d *anthropicDecoder) usage(closing *anUsage) (Usage, bool) {
	u := d.started
	seen := u != (anUsage{})
	if closing != nil {
		seen = true
		pick := func(dst *int, v int) {
			if v != 0 {
				*dst = v
			}
		}
		pick(&u.InputTokens, closing.InputTokens)
		pick(&u.OutputTokens, closing.OutputTokens)
		pick(&u.CacheReadInputTokens, closing.CacheReadInputTokens)
		pick(&u.CacheCreationInputTokens, closing.CacheCreationInputTokens)
	}
	return Usage{
		InputTokens:      u.InputTokens,
		OutputTokens:     u.OutputTokens,
		CacheReadTokens:  u.CacheReadInputTokens,
		CacheWriteTokens: u.CacheCreationInputTokens,
	}, seen
}
