package provider

import "fmt"

// GenericName is the escape hatch a user names explicitly.
const GenericName = "generic"

// Generic is the family for a model nobody has measured.
//
// An unknown model does NOT resolve to this by accident. It fails at session
// creation listing the families that exist, because silently treating an
// unrecognised name as generic is how someone runs for a week against
// thresholds that were never measured for what they are using, and reads every
// oddity as a bug in dcode.
//
// Reaching it requires typing `--family generic`, and every session that does
// carries a warning saying what is not known. The escape hatch exists because
// refusing outright would make dcode unusable against a local model or a new
// release on the day it appears — which is a real cost, paid by the people most
// likely to be trying it.
//
// It speaks the OpenAI dialect, which is what a new endpoint almost always
// implements first, and it borrows MiniMax's encoding for exactly that reason.
type Generic struct{ MiniMaxM3 }

func (Generic) Name() string { return GenericName }

// Models is empty on purpose: nothing resolves to generic by prefix. The list
// being empty is what makes the escape hatch explicit rather than a fallback.
func (Generic) Models() []string { return nil }

func (Generic) Transports() []string { return []string{TransportOpenAI} }

// Encode refuses a transport this family does not declare, as Gemini's does.
//
// Inherited from MiniMaxM3 it serialised the Anthropic dialect for a family
// whose Transports() names one, and its PDF refusal spoke as minimax-m3 to a
// user who had typed `--family generic`. Inheriting the encoding is not
// inheriting the declaration.
func (f Generic) Encode(req Request, transport string) (WireRequest, error) {
	if transport != TransportOpenAI {
		return WireRequest{}, fmt.Errorf("family %s: unsupported transport %q", f.Name(), transport)
	}
	if err := refusePDF(f.Name(), req); err != nil {
		return WireRequest{}, err
	}
	return encodeOpenAI(req)
}

// AcceptsImages is false, and the reason is that nothing here can know.
//
// Generic points at whatever OpenAI-compatible endpoint somebody configured,
// and half of those serve text-only models. Saying no is not a claim that the
// endpoint cannot; it is a refusal to guess on the user's behalf, made where
// they can read it instead of thirty seconds later in a provider error.
//
// A generic endpoint that does read images is a reason to add a family for it,
// which is a decision with a name on it rather than a hope.
func (Generic) AcceptsImages() bool { return false }

// AcceptsPDF is false, for the reason AcceptsImages is.
func (Generic) AcceptsPDF() bool { return false }

// Window is the conservative guess.
//
// Under-guessing compacts early and costs a summary; over-guessing overruns the
// window and loses the turn. The asymmetry decides the number.
func (Generic) Window(string) (int, error) { return 128_000, nil }

// DefaultLimits errs toward the long horizon, not the short one.
//
// It used to be 50 — the same figure as Claude, sized for a ten-file
// refactor. Wrong for what this family actually serves: an endpoint dcode
// has never measured is, more often than not, a local model somebody is
// running specifically to avoid a cloud ceiling, and a low cap truncated
// real coding work silently — reported in the field as a turn that "just
// stops", with the round counter frozen and nothing on screen saying why.
// The repeat detector (MaxIdenticalCalls) is the real defence against a
// pathological loop, unrelated to this number; this is backstop against
// cost and wall-clock time, and a local model pays neither in API dollars.
func (Generic) DefaultLimits() Limits {
	return Limits{MaxIterations: 5000}
}

// Warning is what a session using this family has to say.
//
// Every behavioural threshold in the specs was measured against a named family.
// None of them applies here, and a user who is not told that will read a
// difference in behaviour as a defect.
const GenericWarning = "using --family generic: the behavioural thresholds in this " +
	"product were measured against named model families and none of them applies " +
	"to this model. It will work; how well is not something dcode knows."
