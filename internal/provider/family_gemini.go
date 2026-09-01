package provider

import (
	"fmt"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

// Gemini speaks the OpenAI dialect, through Google's compatibility surface.
//
// It embeds MiniMaxM3 for the encoding and the decoder, the same way Generic
// does and for the same reason: the wire format is the wire format, and a
// second copy of it would be a second thing to keep in step. What Gemini
// overrides is everything the FAMILY axis is for — the name, the models, the
// window, the limits and what it can read.
//
// The native surface (`:streamGenerateContent`) is deliberately not this. It
// puts the model in the URL, authenticates with a header of its own, frames its
// stream differently, and encodes calls as `functionCall`/`functionResponse`
// with `user`/`model` roles and a separate `systemInstruction`. That is a
// TRANSPORT, and writing one before anybody has run this family against a real
// key would be building the harder half first on a guess.
type Gemini struct{ MiniMaxM3 }

func (Gemini) Name() string { return GeminiName }

// GeminiName is the family, and it is also the credential store: a key for
// gemini-2.5-pro reaches the same account as one for a later model.
const GeminiName = "gemini"

func (Gemini) Models() []string     { return []string{"gemini-"} }
func (Gemini) Transports() []string { return []string{TransportOpenAI} }

// AcceptsImages is true. Gemini is natively multimodal and the compatibility
// surface takes a picture the same way the OpenAI dialect does, as a data URL,
// so the encoding this family inherits already produces the right bytes.
//
// Declared rather than attempted, which is the rule for this field — and
// declared without having been run against a key here, which is the honest
// caveat. Saying false would be the larger error: it would make the one
// property this model family is best known for unavailable, silently.
func (Gemini) AcceptsImages() bool { return true }

// Window is one conservative number rather than a table.
//
// The current line is documented at 1,048,576 input tokens; this returns a
// million flat. Under-guessing compacts early and costs a summary, over-guessing
// overruns the window and loses the turn, and the asymmetry decides which side
// to be wrong on. A per-model table would be a list of numbers nobody here can
// check, going stale on Google's release schedule rather than on ours.
func (Gemini) Window(string) (int, error) { return 1_000_000, nil }

// DefaultLimits is the cautious ceiling, because the horizon is unmeasured.
//
// MiniMax's 2000 is justified by a cited long-horizon run and by a real session
// this ceiling truncated. Nothing of the sort exists for Gemini, and copying a
// number across families is how a limit stops meaning anything. Fifty is what
// Claude uses and what Generic uses, sized from a refactor across ten files.
func (Gemini) DefaultLimits() Limits { return Limits{MaxIterations: 50} }

// Encode refuses a transport this family does not declare.
//
// Inherited from MiniMaxM3 it would happily serialize the Anthropic dialect,
// for a family whose Transports() says it speaks one. The registry would not
// compose that pair, so this is not reachable today — and an unreachable
// disagreement between two methods of the same type is exactly the kind that
// becomes reachable later without anyone noticing.
func (f Gemini) Encode(req Request, transport string) (WireRequest, error) {
	if transport != TransportOpenAI {
		return WireRequest{}, fmt.Errorf("family %s: unsupported transport %q", f.Name(), transport)
	}
	return encodeOpenAI(req)
}

func (Gemini) NewDecoder(tools []ce.ToolDef) Decoder {
	return &openAIDecoder{tools: tools}
}
