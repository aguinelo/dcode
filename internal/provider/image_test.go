package provider

import (
	"encoding/json"
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

func shot() ce.Image {
	return ce.Image{MediaType: "image/png", Data: []byte{0x89, 0x50, 0x4e, 0x47}}
}

// The user sends a screenshot of the interface misbehaving, which is the case
// this exists for. It has to reach the model in the shape its API expects.
func TestMiniMaxCarriesAnImage(t *testing.T) {
	wire, err := MiniMaxM3{}.Encode(Request{
		Model:    "MiniMax-M3",
		Messages: []ce.Message{{Role: ce.RoleUser, Text: "why is this cut off?", Images: []ce.Image{shot()}}},
	}, "openai")
	if err != nil {
		t.Fatal(err)
	}
	body := string(wire.Body)

	// The OpenAI-compatible shape: content becomes an array of parts.
	if !strings.Contains(body, `"type":"image_url"`) {
		t.Errorf("no image part in the body:\n%s", body)
	}
	if !strings.Contains(body, "data:image/png;base64,") {
		t.Errorf("the image is not a data URL:\n%s", body)
	}
	if !strings.Contains(body, "why is this cut off?") {
		t.Errorf("the question did not survive:\n%s", body)
	}
}

// A message with no image keeps the plain string form. Sending every message as
// an array would be a wire change for every request to buy nothing.
func TestAMessageWithoutAnImageStaysAString(t *testing.T) {
	wire, err := MiniMaxM3{}.Encode(Request{
		Model:    "MiniMax-M3",
		Messages: []ce.Message{{Role: ce.RoleUser, Text: "plain question"}},
	}, "openai")
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		Messages []struct {
			Content json.RawMessage `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(wire.Body, &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Messages) != 1 || !strings.HasPrefix(string(body.Messages[0].Content), `"`) {
		t.Errorf("content is %s, want a plain string", body.Messages[0].Content)
	}
}

// Claude's blocks are a different shape, and the family abstraction is what
// keeps that difference out of everything above it.
func TestClaudeCarriesAnImage(t *testing.T) {
	wire, err := Claude{}.Encode(Request{
		Model:    "claude-sonnet",
		Messages: []ce.Message{{Role: ce.RoleUser, Text: "look", Images: []ce.Image{shot()}}},
	}, "anthropic")
	if err != nil {
		t.Fatal(err)
	}
	body := string(wire.Body)

	if !strings.Contains(body, `"type":"image"`) {
		t.Errorf("no image block:\n%s", body)
	}
	if !strings.Contains(body, `"media_type":"image/png"`) {
		t.Errorf("the media type is missing:\n%s", body)
	}
	if strings.Contains(body, "data:image/png") {
		t.Error("Claude takes raw base64, not a data URL")
	}
}

// dcode speaks to several providers. A capability some have and others do not
// is exactly the kind that works on the machine it was written on, so each
// family says plainly whether it reads pictures.
func TestEachFamilySaysWhetherItReadsPictures(t *testing.T) {
	if !(MiniMaxM3{}).AcceptsImages() {
		t.Error("M3 is natively multimodal and says it is not")
	}
	if !(Claude{}).AcceptsImages() {
		t.Error("Claude reads images and says it does not")
	}
	// Generic points at whatever endpoint somebody configured, and half of
	// those serve text-only models. Saying no is a refusal to guess, not a
	// claim about the endpoint.
	if (Generic{}).AcceptsImages() {
		t.Error("generic claimed a capability nothing here can know")
	}
}

// Every family answers. A new one that forgets would inherit whatever it
// embedded, which is how generic would have claimed M3's multimodality.
func TestEveryRegisteredFamilyAnswersTheQuestion(t *testing.T) {
	for _, f := range []Family{MiniMaxM3{}, Claude{}, Generic{}} {
		// Calling it is the assertion: the interface requires it, so a family
		// that does not implement it does not compile.
		_ = f.AcceptsImages()
		if f.Name() == "" {
			t.Error("a family with no name")
		}
	}
}
