package provider

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

func pdf() ce.Document {
	return ce.Document{Name: "spec.pdf", MediaType: "application/pdf", Data: []byte("%PDF-1.7\n")}
}

// A PDF reaches Claude as a document block, by value, beside the question —
// the shape the Anthropic surface reads it in.
func TestClaudeCarriesAPDFAsADocumentBlock(t *testing.T) {
	wire, err := Claude{}.Encode(Request{
		Model: "claude-sonnet",
		Messages: []ce.Message{{
			Role: ce.RoleUser, Text: "what does section 2 say?", Documents: []ce.Document{pdf()},
		}},
	}, TransportAnthropic)
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		Messages []struct {
			Content []struct {
				Type   string `json:"type"`
				Text   string `json:"text"`
				Title  string `json:"title"`
				Source *struct {
					Type      string `json:"type"`
					MediaType string `json:"media_type"`
					Data      string `json:"data"`
				} `json:"source"`
			} `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(wire.Body, &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Messages) != 1 {
		t.Fatalf("messages = %d, want 1:\n%s", len(body.Messages), wire.Body)
	}
	var doc, question bool
	for _, c := range body.Messages[0].Content {
		switch c.Type {
		case "document":
			doc = true
			if c.Source == nil || c.Source.Type != "base64" || c.Source.MediaType != "application/pdf" {
				t.Errorf("the document's source is %+v, want base64 application/pdf", c.Source)
			}
			if c.Source != nil && c.Source.Data != base64.StdEncoding.EncodeToString(pdf().Data) {
				t.Errorf("the PDF's bytes did not survive: %q", c.Source.Data)
			}
			if c.Title != "spec.pdf" {
				t.Errorf("the document is titled %q, want the name it was attached as", c.Title)
			}
		case "text":
			question = question || c.Text == "what does section 2 say?"
		}
	}
	if !doc || !question {
		t.Errorf("document=%v question=%v, want both in one message:\n%s", doc, question, wire.Body)
	}
}

// A dialect that cannot carry a PDF says so. Encoding the message without it
// would send the question and not the file it is about, and the answer would
// read as the model's mistake.
func TestADialectThatCannotCarryAPDFRefusesRatherThanDroppingIt(t *testing.T) {
	req := Request{
		Model:    "m",
		Messages: []ce.Message{{Role: ce.RoleUser, Text: "read this", Documents: []ce.Document{pdf()}}},
	}
	cases := []struct {
		name      string
		family    Family
		transport string
	}{
		{"minimax over openai", MiniMaxM3{}, TransportOpenAI},
		{"minimax over anthropic", MiniMaxM3{}, TransportAnthropic},
		{"gemini", Gemini{}, TransportOpenAI},
		{"generic", Generic{}, TransportOpenAI},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := c.family.Encode(req, c.transport)
			if err == nil {
				t.Fatal("a PDF was encoded for a family that does not read one")
			}
			if !strings.Contains(err.Error(), "PDF") {
				t.Errorf("the error does not say what it refused: %v", err)
			}
		})
	}
}

// Each family declares whether it reads a PDF, and only Claude says yes.
// MiniMax and Gemini are not claimed to be unable: nothing here has measured
// them reading one, and a no said here is a refusal at submit time instead of
// a provider error after the upload.
func TestEachFamilySaysWhetherItReadsPDFs(t *testing.T) {
	want := map[string]bool{
		ClaudeName:   true,
		"minimax-m3": false,
		GeminiName:   false,
		GenericName:  false,
	}
	for _, f := range []Family{Claude{}, MiniMaxM3{}, Gemini{}, Generic{}} {
		if got := f.AcceptsPDF(); got != want[f.Name()] {
			t.Errorf("%s.AcceptsPDF() = %v, want %v", f.Name(), got, want[f.Name()])
		}
	}
}
