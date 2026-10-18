package provider

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

// Wire names.
const (
	TransportOpenAI    = "openai"
	TransportAnthropic = "anthropic"
)

// MiniMaxM3 is the project's primary family.
//
// It speaks both dialects, which is the concrete reason transport and family
// are separate axes: with one axis, supporting both would mean two families
// with identical adaptation and identical thresholds, and the copies would
// diverge at the first maintenance.
type MiniMaxM3 struct{}

func (MiniMaxM3) Name() string { return "minimax-m3" }

// AcceptsImages is true: M3 is natively multimodal and its OpenAI-compatible
// surface takes an image as a data URL, up to ten megabytes each.
func (MiniMaxM3) AcceptsImages() bool { return true }

// AcceptsPDF is false. The OpenAI-compatible surface has no part for a file,
// and nothing here has been measured reading one through the Anthropic
// surface — the rule for this field is declared, not attempted.
func (MiniMaxM3) AcceptsPDF() bool     { return false }
func (MiniMaxM3) Transports() []string { return []string{TransportOpenAI, TransportAnthropic} }
func (MiniMaxM3) Models() []string     { return []string{"MiniMax-M3", "minimax-m3"} }

func (MiniMaxM3) Window(string) (int, error) { return 1_000_000, nil }

// DefaultLimits allows far more iterations than a short-task model would.
// M3 is trained for long-horizon agent loops — MiniMax demonstrated a run with
// 1,959 tool calls — and a cap sized for a ten-file refactor would truncate
// legitimate work. The repeat detector remains the real defence; this is the
// backstop, and a backstop tracks the model's horizon.
//
// 200 was below that horizon, and the citation above already said so: the run
// this number was justified by is ten times the number that was written down.
// It truncated a real one — an unattended session on this repository wrote the
// fix it was asked for, complete and passing the gate, and then hit the ceiling
// before it could say it was done. The work survived; the answer did not.
func (MiniMaxM3) DefaultLimits() Limits {
	return Limits{MaxIterations: 2000}
}

func (f MiniMaxM3) Encode(req Request, transport string) (WireRequest, error) {
	if err := refusePDF(f.Name(), req); err != nil {
		return WireRequest{}, err
	}
	switch transport {
	case TransportOpenAI:
		return encodeOpenAI(req)
	case TransportAnthropic:
		return encodeAnthropic(req, m3AnthropicOutput, false)
	}
	return WireRequest{}, fmt.Errorf("family %s: unsupported transport %q", f.Name(), transport)
}

func (f MiniMaxM3) NewDecoder(tools []ce.ToolDef) Decoder {
	return &openAIDecoder{tools: tools}
}

// Claude is the second family. It exists to prove the axes are orthogonal: one
// implementation never validates an abstraction.
type Claude struct{}

// ClaudeName is the family, and the credential store with it.
const ClaudeName = "claude"

func (Claude) Name() string { return ClaudeName }

// AcceptsImages is true, through a source block rather than a data URL.
func (Claude) AcceptsImages() bool { return true }

// AcceptsPDF is true: the Anthropic surface reads a PDF as a document block,
// its text and a picture of each page.
func (Claude) AcceptsPDF() bool     { return true }
func (Claude) Transports() []string { return []string{TransportAnthropic} }
func (Claude) Models() []string     { return []string{"claude-"} }

// DefaultLimits is sized from the case that shaped it: a refactor across ten
// files runs thirty to fifty tool calls.
func (Claude) DefaultLimits() Limits { return Limits{MaxIterations: 50} }

func (f Claude) Encode(req Request, transport string) (WireRequest, error) {
	if transport != TransportAnthropic {
		return WireRequest{}, fmt.Errorf("family %s: unsupported transport %q", f.Name(), transport)
	}
	return encodeAnthropic(req, f.OutputCeiling(req.Model), true)
}

func (Claude) NewDecoder(tools []ce.ToolDef) Decoder {
	return &anthropicDecoder{tools: tools}
}

// ---------- OpenAI dialect ----------

type oaMessage struct {
	Role string `json:"role"`
	// Content is a string when the message is only text, and an array of parts
	// when it carries an image. Both are valid on the wire, and sending every
	// message as an array would change every request to buy nothing.
	Content    any          `json:"content,omitempty"`
	ToolCalls  []oaToolCall `json:"tool_calls,omitempty"`
	ToolCallID string       `json:"tool_call_id,omitempty"`
}

// oaPart is one piece of a multimodal message.
type oaPart struct {
	Type     string      `json:"type"`
	Text     string      `json:"text,omitempty"`
	ImageURL *oaImageURL `json:"image_url,omitempty"`
}

// oaImageURL carries the picture as a data URL, which is what the
// OpenAI-compatible surface takes.
type oaImageURL struct {
	URL string `json:"url"`
}

type oaToolCall struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	// Index is how fragments of the same call are matched across frames. It is
	// the only thing that keeps two parallel calls from merging into one
	// unparseable blob.
	Index    int `json:"index"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type oaTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}

type oaRequest struct {
	Model     string      `json:"model"`
	Messages  []oaMessage `json:"messages"`
	Tools     []oaTool    `json:"tools,omitempty"`
	Stream    bool        `json:"stream"`
	MaxTokens int         `json:"max_tokens,omitempty"`
	// StreamOptions asks for the final usage frame.
	//
	// Without it the dialect reports `"usage": null` on every frame and never
	// sends a total, so the context meter has no numerator and the cache
	// saving is invisible. It is opt-in in the dialect, which means every
	// provider speaking it is silent by default.
	StreamOptions *oaStreamOptions `json:"stream_options,omitempty"`
}

type oaStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

func encodeOpenAI(req Request) (WireRequest, error) {
	// This dialect has no part for a file. Encoding the message without the
	// PDF would ask about a document the model never saw.
	if err := refusePDF("the openai dialect", req); err != nil {
		return WireRequest{}, err
	}
	if err := CheckToolPairs(req.Messages); err != nil {
		return WireRequest{}, err
	}
	body := oaRequest{
		Model: req.Model, Stream: true, MaxTokens: req.MaxTokens,
		StreamOptions: &oaStreamOptions{IncludeUsage: true},
	}

	for _, m := range req.Messages {
		if m.Role == ce.RoleTool && m.ToolResult == nil {
			// A tool message carrying nothing has no call to name, and a tool
			// message naming none is what the provider rejects. Dropped, as
			// the Anthropic dialect drops it (CheckToolPairs).
			continue
		}
		om := oaMessage{Role: string(m.Role), Content: m.Text}
		if m.Role == ce.RoleTool && m.ToolResult != nil {
			om.ToolCallID = m.ToolResult.ToolCallID
			om.Content = m.ToolResult.Output
		}
		if len(m.Images) > 0 {
			parts := make([]oaPart, 0, len(m.Images)+1)
			if m.Text != "" {
				parts = append(parts, oaPart{Type: "text", Text: m.Text})
			}
			for _, img := range m.Images {
				parts = append(parts, oaPart{
					Type: "image_url",
					ImageURL: &oaImageURL{
						URL: "data:" + img.MediaType + ";base64," +
							base64.StdEncoding.EncodeToString(img.Data),
					},
				})
			}
			om.Content = parts
		}
		for _, c := range m.ToolCalls {
			var tc oaToolCall
			tc.ID = c.ID
			tc.Type = "function"
			tc.Function.Name = c.Name
			tc.Function.Arguments = string(c.Input)
			om.ToolCalls = append(om.ToolCalls, tc)
		}
		body.Messages = append(body.Messages, om)
	}

	for _, t := range req.Tools {
		var ot oaTool
		ot.Type = "function"
		ot.Function.Name = t.Name
		ot.Function.Description = t.Description
		ot.Function.Parameters = t.Schema
		body.Tools = append(body.Tools, ot)
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return WireRequest{}, err
	}
	return WireRequest{Model: req.Model, Body: raw, Stream: true}, nil
}

type oaChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
			// Reasoning is the model's thinking. MiniMax-M3 sends it here AND
			// repeats it in Content wrapped in <think> markers, so a frame
			// carrying Reasoning is a thinking frame and its Content is not an
			// answer.
			Reasoning string       `json:"reasoning"`
			ToolCalls []oaToolCall `json:"tool_calls"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens        int `json:"prompt_tokens"`
		CompletionTokens    int `json:"completion_tokens"`
		PromptTokensDetails struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"prompt_tokens_details"`
	} `json:"usage"`
}

// openAIDecoder assembles one stream in the OpenAI dialect.
//
// It exists because a frame is not a unit of meaning here: a tool call's name
// arrives in one frame and its arguments across several more, and the call is
// only whole once the stream reports it finished.
type openAIDecoder struct {
	tools []ce.ToolDef

	// pending holds partial calls keyed by the wire index, and order records
	// the indices as they first appeared so the calls are emitted in the order
	// the model asked for them rather than in map order.
	pending map[int]*partialCall
	order   []int

	flushed bool
	// finished records a finish_reason seen without usage attached, so the
	// terminal event can wait for the frame that carries it.
	finished bool
	// terminated records that the terminal event has already gone out.
	terminated bool
	// truncated records a finish_reason of length: the answer reached the
	// request's output ceiling, and the terminal event has to say so.
	truncated bool
}

type partialCall struct {
	id   string
	name string
	args strings.Builder
}

func (d *openAIDecoder) Decode(ev WireEvent) ([]StreamEvent, error) {
	data := strings.TrimSpace(string(ev.Data))
	if data == "" {
		return nil, nil
	}
	if data == "[DONE]" {
		out, err := d.flush()
		return append(out, d.terminal(nil)...), err
	}

	var c oaChunk
	if err := json.Unmarshal([]byte(data), &c); err != nil {
		return nil, &ProviderError{
			Class:   ErrClassProvider,
			Message: "malformed stream frame: " + sanitize(err.Error()),
		}
	}

	// The CONTENT of the frame first, and the usage after it.
	//
	// This was the other way round, and a frame that carried both was a frame
	// whose content was thrown away: the usage branch flushed a decoder that
	// had absorbed nothing yet, emitted the terminal event and returned. Every
	// tool call in that frame vanished, silently, with the stream reporting a
	// clean end.
	//
	// It is not a hypothetical shape. OpenAI and MiniMax send usage on a frame
	// of its own, after the one that finishes — which is why this held for as
	// long as it did. Gemini sends ONE frame carrying the tool call, the finish
	// reason and the usage together, so the whole family could not make a
	// single tool call, and the failure looked like a model that answered
	// nothing rather than like a decoder that dropped it.
	var out []StreamEvent
	if len(c.Choices) > 0 {
		ch := c.Choices[0]

		for _, tc := range ch.Delta.ToolCalls {
			d.absorb(tc)
		}

		// A frame carrying reasoning is a thinking frame. Its Content repeats
		// the same text with <think> markers around it, so reading Content here
		// would put the model's thinking in its own mouth.
		if ch.Delta.Reasoning != "" {
			out = append(out, StreamEvent{Type: EventReasoningDelta, Text: ch.Delta.Reasoning})
		} else if text, ok := answerText(ch.Delta.Content); ok {
			out = append(out, StreamEvent{Type: EventTextDelta, Text: text})
		}

		if ch.FinishReason != nil {
			flushed, err := d.flush()
			out = append(out, flushed...)
			if err != nil {
				return out, err
			}
			// Finished, but not necessarily over: the usage may ride on a later
			// frame that repeats this one. Terminating here is what threw the
			// token accounting away.
			d.finished = true
			if *ch.FinishReason == "length" {
				d.truncated = true
			}
		}
	}

	if c.Usage != nil {
		// Flushed again rather than assumed flushed: a provider may put usage
		// on a frame with no finish_reason at all, and the calls absorbed
		// before it still have to come out. flush is once-only, so a second
		// call after the branch above costs nothing.
		flushed, err := d.flush()
		out = append(out, flushed...)
		if err != nil {
			return out, err
		}
		return append(out, d.terminal(&Usage{
			InputTokens:     c.Usage.PromptTokens,
			OutputTokens:    c.Usage.CompletionTokens,
			CacheReadTokens: c.Usage.PromptTokensDetails.CachedTokens,
		})...), nil
	}
	return out, nil
}

// terminal emits the single end-of-stream event, at most once.
func (d *openAIDecoder) terminal(u *Usage) []StreamEvent {
	if d.terminated {
		return nil
	}
	d.terminated = true
	return []StreamEvent{{Type: EventDone, Usage: u, Truncated: d.truncated}}
}

// Close ends a stream that simply stopped. Only a stream that reported a finish
// counts as complete; anything else is a truncation, and the pump says so.
func (d *openAIDecoder) Close() []StreamEvent {
	if !d.finished {
		return nil
	}
	return d.terminal(nil)
}

// absorb folds one fragment into the call it belongs to.
func (d *openAIDecoder) absorb(tc oaToolCall) {
	if d.pending == nil {
		d.pending = map[int]*partialCall{}
	}
	p, seen := d.pending[tc.Index]
	if !seen {
		p = &partialCall{}
		d.pending[tc.Index] = p
		d.order = append(d.order, tc.Index)
	}
	// Later frames carry only the fragment, so nothing already known is
	// overwritten with an empty string.
	if tc.ID != "" {
		p.id = tc.ID
	}
	if tc.Function.Name != "" {
		p.name = tc.Function.Name
	}
	p.args.WriteString(tc.Function.Arguments)
}

// flush emits the assembled calls, once.
//
// Once, because a provider may repeat finish_reason — MiniMax does — and a
// second emission would run every tool twice.
func (d *openAIDecoder) flush() ([]StreamEvent, error) {
	if d.flushed || len(d.order) == 0 {
		return nil, nil
	}
	d.flushed = true

	var out []StreamEvent
	for _, idx := range d.order {
		p := d.pending[idx]
		args := p.args.String()
		if strings.TrimSpace(args) == "" {
			// An empty object is what a no-argument tool looks like on the
			// wire; validation still has to agree with the schema.
			args = "{}"
		}
		call, err := validateToolCall(p.id, p.name, args, d.tools)
		if err != nil {
			// Returning what was already assembled keeps the earlier calls
			// visible; the error ends the turn either way.
			return out, err
		}
		out = append(out, StreamEvent{Type: EventToolCall, ToolCall: call})
	}
	return out, nil
}

// answerText strips the framing MiniMax leaves in Content and reports whether
// anything worth showing is left.
//
// The model closes its reasoning with a frame of pure markers and newlines —
// `\n</think>\n\n</think>`, twice over in practice. Stripping alone would turn
// that into three blank lines at the top of every answer, so a frame that was
// nothing but framing is dropped entirely.
func answerText(s string) (string, bool) {
	stripped := s
	for _, marker := range []string{"<think>", "</think>"} {
		stripped = strings.ReplaceAll(stripped, marker, "")
	}
	if stripped == s {
		// No markers, so the content is the model's own — including any
		// whitespace it meant to send.
		return s, s != ""
	}
	if strings.TrimSpace(stripped) == "" {
		return "", false
	}
	return stripped, true
}

// ---------- Anthropic dialect ----------

type anRequest struct {
	Model string `json:"model"`
	// System is a string, or a list of blocks when it carries a cache
	// breakpoint: a block is what carries one. Both are valid on the wire, and
	// a family that sends no breakpoint keeps the bytes it always sent.
	System    any         `json:"system,omitempty"`
	Messages  []anMessage `json:"messages"`
	Tools     []anTool    `json:"tools,omitempty"`
	Stream    bool        `json:"stream"`
	MaxTokens int         `json:"max_tokens"`
}

type anMessage struct {
	Role    string      `json:"role"`
	Content []anContent `json:"content"`
}

type anContent struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
	Source    *anSource       `json:"source,omitempty"`
	// Title names a document block: the file the person attached.
	Title string `json:"title,omitempty"`
	// CacheControl marks the end of a prefix the provider may cache.
	CacheControl *anCacheControl `json:"cache_control,omitempty"`
}

// anSource is how the Anthropic surface carries a picture: raw base64 in a
// block of its own, with the media type named beside it.
type anSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type anTool struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	InputSchema  json.RawMessage `json:"input_schema"`
	CacheControl *anCacheControl `json:"cache_control,omitempty"`
}

// m3AnthropicOutput is what MiniMax M3 asks for on the Anthropic dialect when
// nothing is configured. It is the number this dialect always sent, kept
// because no limit was published that would justify another; an answer that
// reaches it now says so (Truncated) instead of passing for a finished one.
const m3AnthropicOutput = 8192

// encodeAnthropic hoists the system message out of the message list, which is
// where this dialect expects it. Same rules, different placement — exactly the
// difference the two-axis design exists to absorb.
//
// ceiling fills max_tokens when the request names none: the dialect requires
// the field, and what it may be is the family's to say, per model.
//
// cache marks the prompt-cache breakpoints (markCache). It is the family's to
// ask for: the dialect carries the marker, and only a family that knows its
// endpoint reads it should send one.
func encodeAnthropic(req Request, ceiling int, cache bool) (WireRequest, error) {
	if err := CheckToolPairs(req.Messages); err != nil {
		return WireRequest{}, err
	}
	body := anRequest{Model: req.Model, Stream: true, MaxTokens: req.MaxTokens}
	var system string
	if body.MaxTokens <= 0 {
		body.MaxTokens = ceiling
	}

	for _, m := range req.Messages {
		switch m.Role {
		case ce.RoleSystem:
			system = m.Text
			continue
		case ce.RoleTool:
			if m.ToolResult == nil {
				continue
			}
			body.Messages = append(body.Messages, anMessage{Role: "user", Content: []anContent{{
				Type: "tool_result", ToolUseID: m.ToolResult.ToolCallID,
				Content: m.ToolResult.Output, IsError: m.ToolResult.IsError,
			}}})
			continue
		}

		am := anMessage{Role: string(m.Role)}
		if m.Text != "" {
			am.Content = append(am.Content, anContent{Type: "text", Text: m.Text})
		}
		// Raw base64 in a source block, not a data URL: the two surfaces spell
		// the same picture differently, and keeping that difference here is
		// what the family abstraction is for.
		for _, img := range m.Images {
			am.Content = append(am.Content, anContent{
				Type: "image",
				Source: &anSource{
					Type:      "base64",
					MediaType: img.MediaType,
					Data:      base64.StdEncoding.EncodeToString(img.Data),
				},
			})
		}
		// A PDF is a document block with the same base64 source a picture
		// uses, titled with the name it was attached as.
		for _, d := range m.Documents {
			am.Content = append(am.Content, anContent{
				Type:  "document",
				Title: d.Name,
				Source: &anSource{
					Type:      "base64",
					MediaType: d.MediaType,
					Data:      base64.StdEncoding.EncodeToString(d.Data),
				},
			})
		}
		for _, c := range m.ToolCalls {
			am.Content = append(am.Content, anContent{
				Type: "tool_use", ID: c.ID, Name: c.Name, Input: c.Input,
			})
		}
		if len(am.Content) == 0 {
			continue
		}
		body.Messages = append(body.Messages, am)
	}

	for _, t := range req.Tools {
		body.Tools = append(body.Tools, anTool{
			Name: t.Name, Description: t.Description, InputSchema: t.Schema,
		})
	}
	switch {
	case cache:
		markCache(&body, system)
	case system != "":
		body.System = system
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return WireRequest{}, err
	}
	return WireRequest{Model: req.Model, Body: raw, Stream: true}, nil
}

type anEvent struct {
	Type string `json:"type"`
	// Index identifies the content block a delta belongs to. Tool calls and
	// text share one numbering, which is why the decoder keys on it.
	Index        int `json:"index"`
	ContentBlock *struct {
		Type  string          `json:"type"`
		ID    string          `json:"id"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	} `json:"content_block"`
	Delta *struct {
		Type string `json:"type"`
		Text string `json:"text"`
		// PartialJSON carries a tool call's arguments, a fragment per frame.
		// The input on content_block_start is always an empty object.
		PartialJSON string `json:"partial_json"`
		// Thinking is the reasoning channel of this dialect.
		Thinking string `json:"thinking"`
		// StopReason arrives on message_delta. Two matter here: max_tokens,
		// the answer stopped at the ceiling, unfinished; and refusal, the
		// model declined, with StopDetails saying why when it says anything.
		StopReason  string         `json:"stop_reason"`
		StopDetails *anStopDetails `json:"stop_details"`
	} `json:"delta"`
	Usage *anUsage `json:"usage"`
	// Message is set on message_start, the frame that carries the prompt's
	// usage.
	Message *struct {
		Usage *anUsage `json:"usage"`
	} `json:"message"`
	// Error is set on an error frame: a failure that arrives inside a stream
	// that already answered 200.
	Error *anError `json:"error"`
}

type anUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

// anthropicDecoder assembles one stream in the Anthropic dialect.
//
// Same reason as the OpenAI one, different shape: a tool call opens with an
// empty input object and its arguments follow as input_json_delta fragments,
// so the call only exists once its block closes.
type anthropicDecoder struct {
	tools []ce.ToolDef

	pending    map[int]*partialCall
	order      []int
	flushed    bool
	terminated bool
	// started is the usage message_start reported, kept for the terminal
	// event.
	started anUsage
}

// Close is a no-op for this dialect: message_stop is explicit, so a stream that
// ends without one really was truncated.
func (d *anthropicDecoder) Close() []StreamEvent { return nil }

func (d *anthropicDecoder) Decode(ev WireEvent) ([]StreamEvent, error) {
	data := strings.TrimSpace(string(ev.Data))
	if data == "" {
		return nil, nil
	}

	var e anEvent
	if err := json.Unmarshal([]byte(data), &e); err != nil {
		return nil, &ProviderError{
			Class:   ErrClassProvider,
			Message: "malformed stream frame: " + sanitize(err.Error()),
		}
	}

	switch e.Type {
	case "message_start":
		// The prompt side of the usage — what was read and written by the
		// cache among it — is reported here and only here. message_delta
		// carries the output.
		if e.Message != nil && e.Message.Usage != nil {
			d.started = *e.Message.Usage
		}
		return nil, nil

	case "content_block_start":
		if e.ContentBlock != nil && e.ContentBlock.Type == "tool_use" {
			// Opened, not complete: the arguments are still coming.
			//
			// Said out loud, because the name is knowable HERE and the
			// arguments can take seconds. Keeping it to ourselves is what left
			// a screen with nothing on it while a model wrote a file.
			d.open(e.Index, e.ContentBlock.ID, e.ContentBlock.Name)
			return []StreamEvent{{
				Type:   EventToolCallOpened,
				CallID: e.ContentBlock.ID, CallName: e.ContentBlock.Name,
			}}, nil
		}

	case "content_block_delta":
		if e.Delta == nil {
			return nil, nil
		}
		if e.Delta.PartialJSON != "" {
			d.append(e.Index, e.Delta.PartialJSON)
			// Bytes rather than lines: what has arrived is a fragment of JSON,
			// and counting lines in half an escaped string would be counting
			// something that is not there yet.
			if p, ok := d.pending[e.Index]; ok {
				return []StreamEvent{{
					Type:   EventToolCallProgress,
					CallID: p.id, CallName: p.name, Bytes: p.args.Len(),
				}}, nil
			}
			return nil, nil
		}
		if e.Delta.Thinking != "" {
			return []StreamEvent{{Type: EventReasoningDelta, Text: e.Delta.Thinking}}, nil
		}
		if e.Delta.Text != "" {
			return []StreamEvent{{Type: EventTextDelta, Text: e.Delta.Text}}, nil
		}

	case "error":
		return nil, classifyStreamError(e.Error)

	case "message_delta", "message_stop":
		if !d.terminated && e.Delta != nil && e.Delta.StopReason == "refusal" {
			return d.refuse(e), nil
		}
		out, err := d.flush()
		if err != nil {
			return out, err
		}
		if d.terminated {
			return out, nil
		}
		d.terminated = true
		se := StreamEvent{Type: EventDone}
		se.Truncated = e.Delta != nil && e.Delta.StopReason == "max_tokens"
		if u, ok := d.usage(e.Usage); ok {
			se.Usage = &u
		}
		return append(out, se), nil
	}
	return nil, nil
}

func (d *anthropicDecoder) open(index int, id, name string) {
	if d.pending == nil {
		d.pending = map[int]*partialCall{}
	}
	if _, seen := d.pending[index]; seen {
		return
	}
	d.pending[index] = &partialCall{id: id, name: name}
	d.order = append(d.order, index)
}

func (d *anthropicDecoder) append(index int, fragment string) {
	if p, ok := d.pending[index]; ok {
		p.args.WriteString(fragment)
	}
}

func (d *anthropicDecoder) flush() ([]StreamEvent, error) {
	if d.flushed || len(d.order) == 0 {
		return nil, nil
	}
	d.flushed = true

	var out []StreamEvent
	for _, idx := range d.order {
		p := d.pending[idx]
		args := p.args.String()
		if strings.TrimSpace(args) == "" {
			args = "{}"
		}
		call, err := validateToolCall(p.id, p.name, args, d.tools)
		if err != nil {
			return out, err
		}
		out = append(out, StreamEvent{Type: EventToolCall, ToolCall: call})
	}
	return out, nil
}

// ---------- shared validation ----------

// validateToolCall enforces two rules before a call ever reaches the loop:
// the tool must be one we declared, and the arguments must be valid JSON.
//
// Executing a guessed call is how an agent corrupts a file. There is no case
// where running it anyway is worth the risk, so a failure here becomes a
// tool_schema error the model reads and corrects.
func validateToolCall(id, name, args string, tools []ce.ToolDef) (*ce.ToolCall, error) {
	if name == "" {
		return nil, &ProviderError{Class: ErrClassToolSchema, Message: "tool call has no name"}
	}
	declared := false
	for _, t := range tools {
		if t.Name == name {
			declared = true
			break
		}
	}
	if !declared {
		return nil, &ProviderError{
			Class:   ErrClassToolSchema,
			Message: fmt.Sprintf("tool %q was not declared; available: %s", name, toolNames(tools)),
		}
	}

	if strings.TrimSpace(args) == "" {
		args = "{}"
	}
	if !json.Valid([]byte(args)) {
		return nil, &ProviderError{
			Class:   ErrClassToolSchema,
			Message: fmt.Sprintf("arguments for %q are not valid JSON", name),
		}
	}
	return &ce.ToolCall{ID: id, Name: name, Input: json.RawMessage(args)}, nil
}

func toolNames(tools []ce.ToolDef) string {
	names := make([]string, 0, len(tools))
	for _, t := range tools {
		names = append(names, t.Name)
	}
	return strings.Join(names, ", ")
}

// refusePDF fails an encoding that would have to leave a PDF out.
//
// The submit-time check is what a person meets; this is the floor under it, so
// a path that reaches a family without asking it still cannot send the
// question without the file it is about.
func refusePDF(who string, req Request) error {
	for _, m := range req.Messages {
		if len(m.Documents) > 0 {
			return fmt.Errorf("%s cannot carry a PDF (%s); only a family that reads one is sent one",
				who, m.Documents[0].Name)
		}
	}
	return nil
}
