package server

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

func doc(name, mediaType, body string) protocol.TurnDocument {
	return protocol.TurnDocument{Name: name, MediaType: mediaType, Data: b64(body)}
}

// A document is checked where it arrives: what it says it is, what it is
// called, how large and how many. Each refusal is invalid_input and names what
// was wrong, before the session sees a byte of it.
func TestADocumentIsCheckedAtTheBoundary(t *testing.T) {
	c, _ := newDaemon(t)
	ctx := context.Background()
	s, err := c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: "/tmp"})
	if err != nil {
		t.Fatal(err)
	}

	tooMany := make([]protocol.TurnDocument, protocol.MaxDocumentsPerTurn+1)
	for i := range tooMany {
		tooMany[i] = doc("n.txt", protocol.DocumentText, "x")
	}
	cases := []struct {
		name string
		docs []protocol.TurnDocument
		says string
	}{
		{"a PDF that is not one", []protocol.TurnDocument{doc("a.pdf", protocol.DocumentPDF, "hello")}, "a.pdf"},
		{"text that is not UTF-8", []protocol.TurnDocument{doc("a.txt", protocol.DocumentText, "\xff\xfe")}, "UTF-8"},
		{"text with a NUL in it", []protocol.TurnDocument{doc("a.txt", protocol.DocumentText, "a\x00b")}, "a.txt"},
		{"a kind nobody reads", []protocol.TurnDocument{doc("a.docx", "application/msword", "x")}, "application/msword"},
		{"no name", []protocol.TurnDocument{doc("", protocol.DocumentText, "x")}, "name"},
		{"a path for a name", []protocol.TurnDocument{doc("/etc/passwd", protocol.DocumentText, "x")}, "name"},
		{"a control character in the name", []protocol.TurnDocument{doc("a\nb.txt", protocol.DocumentText, "x")}, "name"},
		{"too many", tooMany, "5"},
		{"text too large", []protocol.TurnDocument{doc("big.txt", protocol.DocumentText,
			strings.Repeat("a", protocol.MaxTextDocumentBytes+1))}, "big.txt"},
		{"a PDF too large", []protocol.TurnDocument{doc("big.pdf", protocol.DocumentPDF,
			"%PDF-"+strings.Repeat("a", protocol.MaxPDFBytes))}, "big.pdf"},
		{"base64 that is not", []protocol.TurnDocument{{Name: "a.txt", MediaType: protocol.DocumentText,
			Data: "not base64!!"}}, "a.txt"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := c.SubmitTurn(ctx, s.ID, protocol.SubmitTurnRequest{Text: "read", Documents: tc.docs})
			pe, ok := protocol.AsError(err)
			if !ok || pe.Code != protocol.CodeInvalidInput {
				t.Fatalf("err = %v, want invalid_input", err)
			}
			if !strings.Contains(pe.Message, tc.says) {
				t.Errorf("the refusal does not say %q: %s", tc.says, pe.Message)
			}
		})
	}
}

// What is well formed gets past the boundary. The session here has no engine,
// so the turn is refused there — with an answer that is not about the files.
func TestAWellFormedDocumentGetsPastTheBoundary(t *testing.T) {
	c, _ := newDaemon(t)
	ctx := context.Background()
	s, err := c.CreateSession(ctx, protocol.CreateSessionRequest{Workspace: "/tmp"})
	if err != nil {
		t.Fatal(err)
	}
	err = c.SubmitTurn(ctx, s.ID, protocol.SubmitTurnRequest{Text: "read", Documents: []protocol.TurnDocument{
		doc("notes.md", protocol.DocumentMarkdown, "# título\n"),
		doc("data.csv", protocol.DocumentCSV, "a,b\n1,2\n"),
		doc("spec.pdf", protocol.DocumentPDF, "%PDF-1.7\n"),
	}})
	if pe, ok := protocol.AsError(err); ok && pe.Code == protocol.CodeInvalidInput {
		t.Fatalf("well-formed documents were refused at the boundary: %v", err)
	}
}
