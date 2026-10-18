package orchestrate

import (
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// What the cards refuse, each with its code: a question or a card that is not
// there, one already answered, words where none are taken.
func TestTheCardsRefuseWhatIsNotThereWithItsCode(t *testing.T) {
	p := openWith(t, cardKeys)
	q1, q2 := design(t, p, "One?"), design(t, p, "Two?")
	card := openCards(p, "design")[0]
	if _, held := p.reg.CardsOf("nobody"); held {
		t.Error("a session that conducts nothing has cards")
	}

	_, _, err := p.o.Respond("q-nobody", protocol.AnswerOther, "words")
	wantCode(t, err, protocol.CodeQuestionNotFound)
	if _, err := p.o.Answer(q1, "a", ""); err != nil {
		t.Fatal(err)
	}
	_, _, err = p.o.Respond(q1, protocol.AnswerOther, "words")
	wantCode(t, err, protocol.CodeQuestionAlreadyAnswered)

	// A question kept from before the cards takes no words.
	if err := p.o.apply(func(s *State) {
		for i := range s.Questions {
			if s.Questions[i].ID == q2 {
				s.Questions[i].AllowOther = false
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	_, _, err = p.o.Respond(q2, protocol.AnswerOther, "words")
	wantCode(t, err, protocol.CodeInvalidInput)

	got, err := p.reg.AnswersOf(p.o.orchestratorID(), protocol.AnswerQuestionsRequest{CardID: card.ID, Skipped: []string{q1, "q-nobody"}})
	if err != nil {
		t.Fatal(err)
	}
	if r := got.Results; r[0].Code != protocol.CodeQuestionAlreadyAnswered || r[1].Code != protocol.CodeQuestionNotFound {
		t.Errorf("skips = %+v", r)
	}

	if err := p.o.Summarize(q2, "Duas linhas.\nSobre a escolha."); err != nil {
		t.Fatal(err)
	}
	if it := itemOf(openCards(p, "design")[0], q2); it.Summary != "Duas linhas.\nSobre a escolha." {
		t.Errorf("a design question's summary is not in its card: %+v", it)
	}
}

// A log without the measurement, or without the review, gives no evidence,
// and says which is missing: a merge question with no evidence is not asked.
func TestEvidenceWithoutTheLogsLinesIsRefusedSayingWhich(t *testing.T) {
	p := openWith(t, cardKeys)
	sha := strings.Repeat("a", 40)
	if _, err := p.o.log.Evidence("c1", sha); err == nil || !strings.Contains(err.Error(), "gate.measured") {
		t.Errorf("no gate line: %v", err)
	}
	if _, err := RecordGate(p.o.log, "c1", "it is green", protocol.GateResult{Commit: sha, Result: GateGreen, Command: "true"}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.o.log.Evidence("c1", sha); err == nil || !strings.Contains(err.Error(), "review.verdict") {
		t.Errorf("no review line: %v", err)
	}
	if _, err := p.o.log.Evidence("c1", strings.Repeat("b", 40)); err == nil {
		t.Error("a measurement of another commit was taken as the evidence")
	}
}
