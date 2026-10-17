package loop

import (
	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/protocol"
)

// Attachments are what a turn carries besides its text.
//
// Documents holds every file, text and PDF alike, as it was attached: which
// of them is written into the question and which rides apart is decided when
// the message is built, in one place, rather than by every caller.
//
// Origin is not a file but rides here for the same reason the files do: it is
// part of what the turn starts with, and turn.started is where it is said.
// Empty is the person; protocol.OriginChildren is the harness.
type Attachments struct {
	Images    []ce.Image
	Documents []ce.Document
	Origin    string
}

// named is what turn.started says about the documents: name, kind and size,
// never the bytes.
func (a Attachments) named() []protocol.AttachedDocument {
	if len(a.Documents) == 0 {
		return nil
	}
	out := make([]protocol.AttachedDocument, 0, len(a.Documents))
	for _, d := range a.Documents {
		out = append(out, protocol.AttachedDocument{
			Name: d.Name, MediaType: d.MediaType, Size: len(d.Data),
		})
	}
	return out
}

// message is the question as the model reads it: text files written in
// ahead of what was typed, PDFs and pictures beside it.
func (a Attachments) message(input string) ce.Message {
	text, pdfs := ce.InlineDocuments(input, a.Documents)
	return ce.Message{Role: ce.RoleUser, Text: text, Images: a.Images, Documents: pdfs}
}

// AcceptsPDF reports whether this engine's model reads a PDF.
//
// Asked of the family, which declares it. An engine without one — a test
// double, or a provider nobody named — says no: a yes nobody declared is the
// guess the declaration exists to prevent.
func (e *Engine) AcceptsPDF() bool {
	if e.cfg.Provider == nil {
		return false
	}
	f := e.cfg.Provider.Family()
	return f != nil && f.AcceptsPDF()
}
