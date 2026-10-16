package session

import (
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
)

// claudeProvider is the idle provider under a family that reads PDFs.
type claudeProvider struct{ *idleProvider }

func (claudeProvider) Family() provider.Family { return provider.Claude{} }

func attachedPDF() loop.Attachments {
	return loop.Attachments{Documents: []ce.Document{
		{Name: "spec.pdf", MediaType: protocol.DocumentPDF, Data: []byte("%PDF-1.7")},
	}}
}

// A PDF for a model that does not read one refuses the turn when it is
// submitted, naming the model — not thirty seconds later as a provider error,
// and never by sending the question without the file.
func TestAPDFForAModelThatCannotReadItIsRefusedNamingTheModel(t *testing.T) {
	s := sessionWithEngine(t, &idleProvider{})
	t.Cleanup(s.Close)

	err := s.SubmitAttached("read it", attachedPDF())
	if err == nil {
		t.Fatal("a PDF was accepted for a family that does not read one")
	}
	pe, ok := protocol.AsError(err)
	if !ok || pe.Code != protocol.CodeInvalidInput {
		t.Fatalf("err = %v, want invalid_input", err)
	}
	if !strings.Contains(pe.Message, s.Model) {
		t.Errorf("the refusal does not name the model %q: %s", s.Model, pe.Message)
	}
	if s.State() != protocol.SessionStateIdle {
		t.Errorf("a refused turn left the session %s", s.State())
	}
}

// Text is read by every family, so a text file is never the reason a turn
// is refused.
func TestATextFileIsAcceptedByEveryFamily(t *testing.T) {
	s := sessionWithEngine(t, &idleProvider{})
	t.Cleanup(s.Close)

	err := s.SubmitAttached("read it", loop.Attachments{Documents: []ce.Document{
		{Name: "notes.txt", MediaType: protocol.DocumentText, Data: []byte("hi")},
	}})
	if err != nil {
		t.Fatalf("a text file was refused: %v", err)
	}
	waitState(t, s, protocol.SessionStateIdle)
}

// A family that reads PDFs takes the turn.
func TestAPDFForAModelThatReadsItStartsTheTurn(t *testing.T) {
	s := sessionWithEngine(t, claudeProvider{&idleProvider{}})
	t.Cleanup(s.Close)

	if err := s.SubmitAttached("read it", attachedPDF()); err != nil {
		t.Fatalf("a PDF for Claude was refused: %v", err)
	}
	waitState(t, s, protocol.SessionStateIdle)
}
