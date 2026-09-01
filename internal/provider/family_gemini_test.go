package provider

import (
	"encoding/json"
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

// The family resolves by prefix, and only by its own prefix. A model name that
// merely mentions the word is not this family: resolution decides which
// thresholds, window and limits a session runs under, and a loose prefix is how
// one model's numbers get applied to another.
func TestGeminiClaimsItsOwnModelsAndNoOthers(t *testing.T) {
	families := []Family{MiniMaxM3{}, Claude{}, Gemini{}}
	for model, want := range map[string]string{
		"gemini-2.5-pro":  "gemini",
		"gemini-3-pro":    "gemini",
		"claude-opus-4":   "claude",
		"MiniMax-M3":      "minimax-m3",
		"gemma-2-9b":      "",
		"my-gemini-clone": "",
		"gemini":          "",
	} {
		f, ok := FamilyFor(model, families)
		got := ""
		if ok {
			got = f.Name()
		}
		if got != want {
			t.Errorf("FamilyFor(%q) = %q, want %q", model, got, want)
		}
	}
}

// It speaks one dialect and refuses the other. Inherited from MiniMaxM3, Encode
// would have serialised Anthropic for a family whose Transports() names only
// OpenAI — two methods of one type disagreeing about what it speaks.
func TestGeminiEncodesOpenAIAndRefusesTheOther(t *testing.T) {
	f := Gemini{}
	if got := f.Transports(); len(got) != 1 || got[0] != TransportOpenAI {
		t.Fatalf("Transports() = %v", got)
	}
	req := Request{Model: "gemini-2.5-pro", Messages: []ce.Message{{Role: ce.RoleUser, Text: "hi"}}}

	wire, err := f.Encode(req, TransportOpenAI)
	if err != nil {
		t.Fatalf("the dialect it declares must encode: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(wire.Body, &body); err != nil {
		t.Fatalf("the body is not JSON: %v", err)
	}
	if body["model"] != "gemini-2.5-pro" {
		t.Errorf("the model did not survive encoding: %v", body["model"])
	}

	if _, err := f.Encode(req, TransportAnthropic); err == nil {
		t.Error("a transport this family does not declare must be refused, not encoded")
	} else if !strings.Contains(err.Error(), TransportAnthropic) {
		t.Errorf("the refusal must name the transport: %v", err)
	}
}

// The window is under the documented one, on purpose: under-guessing compacts
// early and costs a summary, over-guessing overruns and loses the turn.
func TestGeminiWindowErrsOnTheCheapSide(t *testing.T) {
	got, err := Gemini{}.Window("gemini-2.5-pro")
	if err != nil {
		t.Fatal(err)
	}
	if got > 1_048_576 {
		t.Errorf("window %d is above the documented 1,048,576; the wrong side to be wrong on", got)
	}
	if got < 100_000 {
		t.Errorf("window %d compacts a long-context model into a short one", got)
	}
}

// The limits are the cautious ones, and specifically not MiniMax's. Copying a
// ceiling across families is how a limit stops meaning anything: MiniMax's 2000
// is justified by a cited long-horizon run that says nothing about this model.
func TestGeminiDoesNotInheritMiniMaxsHorizon(t *testing.T) {
	got := Gemini{}.DefaultLimits()
	mini := MiniMaxM3{}.DefaultLimits()
	if got == mini {
		t.Errorf("gemini inherited MiniMax's ceiling of %d without a measurement to justify it",
			got.MaxIterations)
	}
	if got.MaxIterations <= 0 {
		t.Error("a family with no ceiling has no backstop at all")
	}
}
