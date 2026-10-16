package loop

import (
	"context"
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
	"github.com/aguinelo/dcode/internal/tools"
)

// readsPDFs is a scripted provider whose family says it reads PDFs.
type readsPDFs struct{ *scriptedProvider }

func (readsPDFs) Family() provider.Family { return provider.Claude{} }

// A turn with documents puts them in the question's message — the PDF apart,
// the text written in — and says in turn.started which files they were,
// without their bytes.
func TestATurnCarriesItsDocumentsAndSaysWhichTheyWere(t *testing.T) {
	p := readsPDFs{&scriptedProvider{turns: [][]provider.StreamEvent{{text("read"), done()}}}}
	e, rec := newEngine(t, p, tools.NewRegistry())

	att := Attachments{Documents: []ce.Document{
		{Name: "spec.pdf", MediaType: protocol.DocumentPDF, Data: []byte("%PDF-1.7")},
		{Name: "notes.txt", MediaType: protocol.DocumentText, Data: []byte("remember the tabs")},
	}}
	if _, err := e.RunAttached(context.Background(), "compare them", att); err != nil {
		t.Fatal(err)
	}

	if len(p.seen) == 0 {
		t.Fatal("the model was never asked")
	}
	var asked *ce.Message
	for i, m := range p.seen[0].Messages {
		if m.Role == ce.RoleUser && strings.Contains(m.Text, "compare them") {
			asked = &p.seen[0].Messages[i]
		}
	}
	if asked == nil {
		t.Fatalf("the question did not reach the model: %+v", p.seen[0].Messages)
	}
	if len(asked.Documents) != 1 || asked.Documents[0].Name != "spec.pdf" {
		t.Errorf("the message carries %+v, want the PDF", asked.Documents)
	}
	if !strings.Contains(asked.Text, "remember the tabs") {
		t.Errorf("the text file was not written into the message:\n%s", asked.Text)
	}

	started, ok := rec.last[protocol.EventTurnStarted].(protocol.TurnStarted)
	if !ok {
		t.Fatalf("turn.started is %T", rec.last[protocol.EventTurnStarted])
	}
	if started.Text != "compare them" {
		t.Errorf("turn.started text = %q, want what was typed and not the files", started.Text)
	}
	want := []protocol.AttachedDocument{
		{Name: "spec.pdf", MediaType: protocol.DocumentPDF, Size: 8},
		{Name: "notes.txt", MediaType: protocol.DocumentText, Size: 17},
	}
	if len(started.Documents) != len(want) {
		t.Fatalf("turn.started documents = %+v, want %+v", started.Documents, want)
	}
	for i := range want {
		if started.Documents[i] != want[i] {
			t.Errorf("document %d = %+v, want %+v", i, started.Documents[i], want[i])
		}
	}
}

// The engine answers for its family: no family, no PDF.
func TestAnEngineSaysWhetherItsModelReadsPDFs(t *testing.T) {
	yes, _ := newEngine(t, readsPDFs{&scriptedProvider{}}, tools.NewRegistry())
	no, _ := newEngine(t, &scriptedProvider{}, tools.NewRegistry())
	if !yes.AcceptsPDF() {
		t.Error("an engine on Claude says it cannot read a PDF")
	}
	if no.AcceptsPDF() {
		t.Error("an engine with no family says it can read a PDF")
	}
}
