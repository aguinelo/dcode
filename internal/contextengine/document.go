package contextengine

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"html"
	"strings"
)

// InlineDocuments writes the text files into the question and hands back the
// PDFs, which a family has to declare it reads.
//
// Text goes ahead of the question, each file between markers that name it
// and carry a boundary derived from its bytes, so the file cannot close its own
// block — a body holding </document> is still inside the file:
// every model reads text, so a text file needs no family to say yes, and
// putting it first is the order a reader meets a document and then the
// question about it. Pure, like Assemble — the same files and the same
// question give the same bytes.
func InlineDocuments(text string, docs []Document) (string, []Document) {
	var b strings.Builder
	var pdfs []Document
	for _, d := range docs {
		if !strings.HasPrefix(d.MediaType, "text/") {
			pdfs = append(pdfs, d)
			continue
		}
		// Escaped, so a name with a quote in it is a name and not the start
		// of an attribute of its own.
		bound := boundary(d.Data)
		b.WriteString(`<document name="` + html.EscapeString(d.Name) +
			`" media_type="` + html.EscapeString(d.MediaType) +
			`" boundary="` + bound + `">` + "\n")
		b.Write(d.Data)
		if len(d.Data) > 0 && d.Data[len(d.Data)-1] != '\n' {
			b.WriteByte('\n')
		}
		b.WriteString(`</document boundary="` + bound + `">` + "\n\n")
	}
	if b.Len() == 0 {
		return text, pdfs
	}
	return b.String() + text, pdfs
}

// boundary is what closes one document's block and nothing else.
//
// From the content rather than random, so the same file gives the same
// message — the prefix is a pure function of history. A body cannot contain
// the closing marker without containing its own digest, which is the property
// a boundary needs: the file is left whole, nothing in it escaped or altered,
// and still nothing in it can end the block early.
func boundary(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:12])
}

// PDFPages estimates how many pages a PDF has, from its page objects.
//
// A heuristic, and said to be one: a page object is `/Type /Page`, the tree
// above them is `/Type /Pages`, and a PDF that packs its objects into a
// compressed stream hides both. Hidden pages count as one rather than none, so
// an opaque PDF still costs something — the error the budget can afford is
// counting a document as short, not counting it as free.
func PDFPages(data []byte) int {
	n := 0
	for _, marker := range [][]byte{[]byte("/Type /Page"), []byte("/Type/Page")} {
		rest := data
		for {
			i := bytes.Index(rest, marker)
			if i < 0 {
				break
			}
			rest = rest[i+len(marker):]
			if len(rest) > 0 && rest[0] == 's' {
				continue
			}
			n++
		}
	}
	if n == 0 {
		return 1
	}
	return n
}

// charsPerPDFPage is what one page is counted as, in characters.
//
// A page reaches the model as its text and as a picture of it, and the
// Anthropic documentation puts that at fifteen hundred to three thousand
// tokens. The high end, at four characters a token: a budget that guesses
// long compacts a summary early, one that guesses short loses the turn.
const charsPerPDFPage = 12000

// documentChars is what a message's PDFs cost the budget.
func documentChars(docs []Document) int {
	chars := 0
	for _, d := range docs {
		chars += len(d.Name) + PDFPages(d.Data)*charsPerPDFPage
	}
	return chars
}
