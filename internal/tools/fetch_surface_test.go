package tools

import (
	"encoding/json"
	"strings"
	"testing"
)

// A tool's description and schema are the only things the model reads before
// deciding to call it, which makes them behaviour rather than documentation.
// Both were unclaimed by any test, so nothing would have noticed a `fetch` that
// declared no URL parameter and could therefore never be called correctly.
func TestFetchTellsTheModelWhatItIsAndWhatItTakes(t *testing.T) {
	f := Fetch{}
	if f.Name() != "fetch" {
		t.Fatalf("name = %q", f.Name())
	}

	desc := f.Description()
	// The crossing has to be in the description: a model that does not know
	// this asks the user for permission it did not need, or reaches out when a
	// file in the workspace already had the answer.
	for _, want := range []string{"URL", "network", "binary"} {
		if !strings.Contains(strings.ToLower(desc), strings.ToLower(want)) {
			t.Errorf("the description never mentions %q: %s", want, desc)
		}
	}

	var schema struct {
		Type       string `json:"type"`
		Properties struct {
			URL *struct {
				Type string `json:"type"`
			} `json:"url"`
		} `json:"properties"`
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(f.Schema(), &schema); err != nil {
		t.Fatalf("the schema is not valid JSON: %v", err)
	}
	if schema.Type != "object" || schema.Properties.URL == nil {
		t.Fatalf("the schema does not declare a url property: %s", f.Schema())
	}
	if len(schema.Required) != 1 || schema.Required[0] != "url" {
		t.Errorf("required = %v, want url — a fetch with no URL cannot be answered", schema.Required)
	}
}

// The content type decides whether a body reaches the context window. Getting
// it wrong in either direction is costly: too strict refuses the small static
// pages most worth reading, too loose decodes a binary into the conversation.
func TestWhatCountsAsReadable(t *testing.T) {
	for _, tc := range []struct {
		ct   string
		want bool
		why  string
	}{
		{"", true, "plenty of servers send no type at all"},
		{"text/plain", true, "text"},
		{"text/html; charset=utf-8", true, "parameters do not change the type"},
		{"application/json", true, "json"},
		{"application/xml", true, "xml"},
		{"application/xhtml+xml", true, "xhtml"},
		{"application/problem+json", true, "a +json suffix is json"},
		{"image/svg+xml", true, "a +xml suffix is xml"},
		{"application/pdf", false, "a document is not text"},
		{"image/png", false, "an image is not text"},
		{"application/octet-stream", false, "bytes are not text"},
	} {
		if got := isReadable(tc.ct); got != tc.want {
			t.Errorf("isReadable(%q) = %v, want %v — %s", tc.ct, got, tc.want, tc.why)
		}
	}
}

// Input that is not the declared shape is refused as a tool error the model can
// correct, never as a crash and never as a request that goes out anyway.
func TestFetchRefusesInputThatIsNotTheDeclaredShape(t *testing.T) {
	if _, err := (Fetch{}).Declare(json.RawMessage(`{"url":42}`)); err == nil {
		t.Fatal("a url that is not a string was declared as a request")
	}
}
