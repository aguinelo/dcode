package server

import (
	"bytes"
	"encoding/base64"
	"strings"
	"unicode"
	"unicode/utf8"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/protocol"
)

// decodeDocuments turns the wire form back into bytes, checking each file on
// the way: the boundary is where a caller can still be told which part of the
// request was wrong.
//
// Every check here is about the file, not the model. Whether the session's
// model reads a PDF is the session's question, asked when the turn starts.
func decodeDocuments(in []protocol.TurnDocument) ([]ce.Document, error) {
	if len(in) > protocol.MaxDocumentsPerTurn {
		return nil, protocol.Errorf(protocol.CodeInvalidInput,
			"%d documents in one turn, and the limit is %d", len(in), protocol.MaxDocumentsPerTurn)
	}
	out := make([]ce.Document, 0, len(in))
	for _, d := range in {
		if err := checkDocumentName(d.Name); err != nil {
			return nil, err
		}
		limit, ok := documentLimit(d.MediaType)
		if !ok {
			return nil, protocol.Errorf(protocol.CodeInvalidInput,
				"%s is %s, and a turn carries a PDF or plain text", d.Name, d.MediaType)
		}
		// Measured before decoding, so an oversized file is refused without
		// first being held whole in memory a second time.
		if base64.StdEncoding.DecodedLen(len(d.Data)) > limit+2 {
			return nil, tooLarge(d.Name, limit)
		}
		body, err := base64.StdEncoding.DecodeString(d.Data)
		if err != nil {
			return nil, protocol.Errorf(protocol.CodeInvalidInput,
				"%s could not be decoded: %v", d.Name, err)
		}
		if len(body) > limit {
			return nil, tooLarge(d.Name, limit)
		}
		if err := checkDocumentBody(d.Name, d.MediaType, body); err != nil {
			return nil, err
		}
		out = append(out, ce.Document{Name: d.Name, MediaType: d.MediaType, Data: body})
	}
	return out, nil
}

// documentLimit is the size a kind of document may have, and whether it is a
// kind a turn carries at all.
func documentLimit(mediaType string) (int, bool) {
	switch mediaType {
	case protocol.DocumentPDF:
		return protocol.MaxPDFBytes, true
	case protocol.DocumentText, protocol.DocumentMarkdown, protocol.DocumentCSV:
		return protocol.MaxTextDocumentBytes, true
	}
	return 0, false
}

// checkDocumentName refuses a name that is not one: empty, too long, carrying
// a control character, or a path. The name is shown on the message and to the
// model, and a path would tell both something about a machine they are not on.
func checkDocumentName(name string) error {
	bad := name == "" || len(name) > protocol.MaxDocumentNameBytes ||
		!utf8.ValidString(name) || strings.ContainsAny(name, `/\`)
	for _, r := range name {
		if unicode.IsControl(r) {
			bad = true
		}
	}
	if bad {
		return protocol.Errorf(protocol.CodeInvalidInput,
			"document name %q is not a file name: it must be 1 to %d bytes, without a path or a control character",
			name, protocol.MaxDocumentNameBytes)
	}
	return nil
}

// pdfMagic is how a PDF begins, whatever it is called.
var pdfMagic = []byte("%PDF-")

// checkDocumentBody holds a file to what it says it is. A PDF is known by its
// first bytes, never by its name; text has nothing to sniff, so it has to be
// valid UTF-8 and carry no NUL — what a binary file renamed .txt would not.
func checkDocumentBody(name, mediaType string, body []byte) error {
	if mediaType == protocol.DocumentPDF {
		if !bytes.HasPrefix(body, pdfMagic) {
			return protocol.Errorf(protocol.CodeInvalidInput,
				"%s is not a PDF: it does not begin with %s", name, pdfMagic)
		}
		return nil
	}
	if !utf8.Valid(body) {
		return protocol.Errorf(protocol.CodeInvalidInput, "%s is not UTF-8 text", name)
	}
	if bytes.IndexByte(body, 0) >= 0 {
		return protocol.Errorf(protocol.CodeInvalidInput,
			"%s has a NUL byte in it, which text does not", name)
	}
	return nil
}

func tooLarge(name string, limit int) error {
	return protocol.Errorf(protocol.CodeInvalidInput,
		"%s is larger than the %d bytes a turn carries for it", name, limit)
}
