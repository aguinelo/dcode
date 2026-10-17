package provider

import (
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

// An unknown model must NOT land here by accident. Silently treating an
// unrecognised name as generic is how someone runs for a week against
// thresholds never measured for what they are using, and reads every oddity as
// a bug in dcode.
func TestAnUnknownModelDoesNotResolveToGeneric(t *testing.T) {
	families := []Family{MiniMaxM3{}, Claude{}, Generic{}}
	if f, ok := FamilyFor("some-model-nobody-has", families); ok {
		t.Fatalf("an unknown model resolved to %q; it must fail and list what exists", f.Name())
	}
	// The escape hatch is reached by naming it, and only by naming it.
	var probe Generic
	if len(probe.Models()) != 0 {
		t.Error("generic claims model prefixes, so something could resolve to it by accident")
	}
}

func TestGenericResolvesWhenNamed(t *testing.T) {
	r := NewRegistry()
	r.RegisterTransport(NewReplayTransport(TransportOpenAI, Transcript{}))
	for _, f := range []Family{MiniMaxM3{}, Claude{}, Generic{}} {
		if err := r.RegisterFamily(f); err != nil {
			t.Fatal(err)
		}
	}
	p, err := r.ResolveFamily(GenericName, "")
	if err != nil {
		t.Fatalf("the escape hatch the spec declares does not resolve: %v", err)
	}
	if p.Family().Name() != GenericName {
		t.Errorf("resolved %q", p.Family().Name())
	}
}

// Under-guessing the window compacts early and costs a summary; over-guessing
// overruns it and loses the turn. The asymmetry decides the number.
func TestGenericGuessesConservatively(t *testing.T) {
	var g Generic
	var m MiniMaxM3
	w, err := g.Window("anything")
	if err != nil {
		t.Fatal(err)
	}
	mini, _ := m.Window("MiniMax-M3")
	if w >= mini {
		t.Errorf("generic guesses %d, which is not more cautious than the measured %d", w, mini)
	}
}

// The window guess errs toward the safe side of the asymmetry; the round
// ceiling errs the other way, on purpose. Generic is, more often than not,
// a local model somebody is running specifically to avoid a cloud ceiling —
// a low cap truncated real coding work silently, reported in the field as a
// turn that "just stops". The repeat detector is what actually defends
// against a pathological loop; this number only bounds cost and wall-clock
// time, and a local model pays neither in API dollars.
func TestGenericErrsTowardTheLongHorizonNotTheShortOne(t *testing.T) {
	var g Generic
	if got := g.DefaultLimits().MaxIterations; got < 1000 {
		t.Errorf("generic's ceiling is %d, too close to a cloud family's cost-bounded default for what this family actually serves", got)
	}
}

func TestTheWarningSaysWhatIsNotKnown(t *testing.T) {
	for _, want := range []string{"generic", "measured", "will work"} {
		if !strings.Contains(GenericWarning, want) {
			t.Errorf("the warning does not carry %q: %s", want, GenericWarning)
		}
	}
}

// It speaks one dialect and refuses the other, as Gemini does. Inherited from
// MiniMaxM3, Encode serialised Anthropic for a family whose Transports() names
// only OpenAI, and the refusal it did make spoke in MiniMax's name rather than
// the one the user typed.
func TestGenericEncodesOpenAIAndRefusesTheOther(t *testing.T) {
	f := Generic{}
	if got := f.Transports(); len(got) != 1 || got[0] != TransportOpenAI {
		t.Fatalf("Transports() = %v", got)
	}
	req := Request{Model: "local-model", Messages: []ce.Message{{Role: ce.RoleUser, Text: "hi"}}}

	if _, err := f.Encode(req, TransportOpenAI); err != nil {
		t.Fatalf("the dialect it declares must encode: %v", err)
	}
	if _, err := f.Encode(req, TransportAnthropic); err == nil {
		t.Error("a transport this family does not declare must be refused, not encoded")
	} else if !strings.Contains(err.Error(), TransportAnthropic) || !strings.Contains(err.Error(), GenericName) {
		t.Errorf("the refusal must name the transport and the family: %v", err)
	}

	pdf := Request{Model: "local-model", Messages: []ce.Message{{
		Role: ce.RoleUser, Text: "read this",
		Documents: []ce.Document{{Name: "spec.pdf", MediaType: "application/pdf", Data: []byte("%PDF-1.7")}},
	}}}
	_, err := f.Encode(pdf, TransportOpenAI)
	if err == nil {
		t.Fatal("a PDF must be refused by a family that does not read one")
	}
	if !strings.HasPrefix(err.Error(), GenericName+" ") || strings.Contains(err.Error(), "minimax") {
		t.Errorf("the PDF refusal must speak for the family the user named: %v", err)
	}
}
