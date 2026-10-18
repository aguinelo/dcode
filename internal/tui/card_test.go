package tui

import (
	"reflect"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The orchestrator's question card in the transcript (orchestration .p §5.5):
// drawn where it was said, one item at a time with "1 de N", the options
// numbered with the recommended one said, and ^Q putting the keyboard on it
// to move, select, skip, go to the next item and confirm every answer in one
// POST.

func cardItem(id, header string) protocol.CardItem {
	q := mergeQuestion(id, "c-"+id, time.Unix(100, 0))
	q.Header = header
	q.Text = `Merge "` + header + `"?`
	q.AllowOther = true
	q.Summary = "Muda " + header + ".\nCom teste."
	q.Options[0].Description = "Entra na main."
	q.Evidence = &protocol.QuestionEvidence{Verdict: "approve", Gate: "green", Command: "make check", Commit: "1f2e3d4c5b"}
	return protocol.CardItem{Question: q, State: protocol.CardItemPending, Selected: "merge"}
}

func threeMerges(updated int64) protocol.QuestionCard {
	return protocol.QuestionCard{
		ID: "card-1", Kind: "merge", Title: "Merges prontos",
		Items:   []protocol.CardItem{cardItem("q1", "índice"), cardItem("q2", "notas"), cardItem("q3", "docs")},
		Pending: 3, Created: time.Unix(100, 0), Updated: time.Unix(updated, 0),
	}
}

func streamText(m Model) string {
	g := DefaultGeometry(140, 40)
	g.Palette, g.Unicode = Palette{}, true
	return strings.Join(StreamLines(m, g), "\n")
}

// withCard is an orchestrator whose transcript has the three-merge card, its
// questions in the list.
func withCard(t *testing.T, lang Lang, f *fakeOrchestration) *program {
	t.Helper()
	p, _ := orchestrating(t, lang, f)
	c := threeMerges(100)
	for _, it := range c.Items {
		f.questions = append(f.questions, it.Question)
	}
	loadedWith(t, p, f)
	p.model = apply(t, p.model,
		ev(t, 2, protocol.EventMessageDelta, protocol.MessageDelta{TurnID: "t", Text: "três prontos"}),
		ev(t, 3, protocol.EventQuestionCard, c),
	)
	return p
}

func TestTheQuestionCardIsDrawnInTheTranscriptWhereItWasSaid(t *testing.T) {
	p := withCard(t, PtBR, &fakeOrchestration{})
	s := streamText(p.model)
	for _, want := range []string{"índice", `Merge "índice"?`, "1 de 3", "1. (•) fazer o merge (Recomendado)", "Entra na main.",
		"Muda índice.", "revisão approve", "gate green", "make check", "1f2e3d4", "4. ( ) Outro", "Precisa da sua contribuição · 3 pendentes"} {
		if !strings.Contains(s, want) {
			t.Errorf("the card does not show %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "1f2e3d4c5b") {
		t.Errorf("the commit is not shortened:\n%s", s)
	}
	if !strings.Contains(s, "três prontos") || strings.Index(s, "três prontos") > strings.Index(s, "1 de 3") {
		t.Errorf("the card is not drawn after what was said before it:\n%s", s)
	}

	// A newer version of the card changes it in place; an older one does not come back.
	updated := threeMerges(200)
	updated.Pending = 2
	updated.Items[0].State, updated.Items[0].Answer = protocol.CardItemAnswered, "merge"
	p.model = apply(t, p.model, ev(t, 4, protocol.EventQuestionCard, updated), ev(t, 5, protocol.EventQuestionCard, threeMerges(150)))
	cards := 0
	for _, e := range p.model.Entries {
		if e.Kind == KindCard {
			cards++
			if e.Card.Pending != 2 {
				t.Errorf("the card holds pending %d, want the newest version's 2", e.Card.Pending)
			}
		}
	}
	if cards != 1 {
		t.Fatalf("the card is drawn %d times, want once, changed in place", cards)
	}
}

func TestCtrlQFocusesTheCardAndItsKeysConfirmEveryAnswerInOnePost(t *testing.T) {
	f := &fakeOrchestration{}
	p := withCard(t, PtBR, f)
	f.batchResp = protocol.AnswerQuestionsResponse{
		Results: []protocol.AnswerResult{{QuestionID: "q1", Status: protocol.CardItemAnswered}, {QuestionID: "q3", Status: protocol.CardItemAnswered}, {QuestionID: "q2", Status: protocol.CardItemSkipped}},
	}

	p.Update(ctrl('q'))
	if p.model.Card == nil || p.model.Ask != nil {
		t.Fatalf("^Q did not put the keyboard on the card: card %+v, ask %+v", p.model.Card, p.model.Ask)
	}
	p.Update(special(tea.KeyDown)) // q1: return
	p.Update(special(tea.KeyEnter))
	if p.model.Card.Index != 1 {
		t.Fatalf("enter did not go to the next item: %d", p.model.Card.Index)
	}
	p.Update(key("s")) // q2: skipped
	if !strings.Contains(streamText(p.model), "3 de 3") {
		t.Fatalf("skip did not show the last item:\n%s", streamText(p.model))
	}
	p.Update(key("3")) // q3: close
	_, cmd := p.Update(special(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("enter on the last item sent nothing")
	}
	p.Update(one(t, cmd))

	want := protocol.AnswerQuestionsRequest{CardID: "card-1", Answers: []protocol.AnswerItem{
		{QuestionID: "q1", Answer: "return"}, {QuestionID: "q3", Answer: "close"},
	}, Skipped: []string{"q2"}}
	if got := f.confirmed(); len(got) != 1 || !reflect.DeepEqual(got[0], want) {
		t.Fatalf("the confirm was not one POST of every answer:\n got %+v\nwant %+v", got, want)
	}
	if len(p.model.Agents.Questions) != 1 || p.model.Agents.Questions[0].ID != "q2" {
		t.Fatalf("the answered questions are still counted: %+v", p.model.Agents.Questions)
	}
}

func TestTheCardAppliesTheRecommendationToAll(t *testing.T) {
	f := &fakeOrchestration{}
	p := withCard(t, En, f)
	p.Update(ctrl('q'))
	_, cmd := p.Update(key("a"))
	p.Update(one(t, cmd))
	got := f.confirmed()
	if len(got) != 1 || len(got[0].Answers) != 3 || got[0].Answers[2] != (protocol.AnswerItem{QuestionID: "q3", Answer: "merge"}) {
		t.Fatalf("apply-all did not send each recommendation: %+v", got)
	}
}

func TestTheCardTakesThePersonsWordsInOther(t *testing.T) {
	f := &fakeOrchestration{}
	p := withCard(t, En, f)
	p.Update(ctrl('q'))
	p.Update(key("4"))
	_, cmd := p.Update(special(tea.KeyEnter))
	_ = cmd
	p.Update(special(tea.KeyRight))
	p.Update(special(tea.KeyRight))
	if _, cmd := p.Update(special(tea.KeyEnter)); cmd != nil {
		t.Fatal("Other with no words was sent")
	}
	if !strings.Contains(p.model.Flash, "Other") {
		t.Fatalf("Other with no words was not said: %q", p.model.Flash)
	}
	p.Update(special(tea.KeyLeft))
	p.Update(special(tea.KeyLeft))
	for _, r := range "wait CI" {
		p.Update(key(string(r)))
	}
	p.Update(special(tea.KeyBackspace))
	p.Update(special(tea.KeyRight))
	p.Update(special(tea.KeyRight))
	_, cmd = p.Update(special(tea.KeyEnter))
	p.Update(one(t, cmd))
	got := f.confirmed()
	if len(got) != 1 || got[0].Answers[0] != (protocol.AnswerItem{QuestionID: "q1", Answer: protocol.AnswerOther, Note: "wait C"}) {
		t.Fatalf("Other did not go with the words as the note: %+v", got)
	}
}

func TestARefusedAnswerOfTheCardIsSaid(t *testing.T) {
	f := &fakeOrchestration{batchResp: protocol.AnswerQuestionsResponse{Results: []protocol.AnswerResult{
		{QuestionID: "q1", Status: protocol.AnswerRefused, Code: protocol.CodeQuestionAlreadyAnswered, Message: "q1 was answered merge"},
	}}}
	p := withCard(t, En, f)
	p.Update(ctrl('q'))
	_, cmd := p.Update(key("a"))
	p.Update(one(t, cmd))
	s := streamText(p.model)
	if !strings.Contains(s, "question_already_answered") || !strings.Contains(s, "q1 was answered merge") {
		t.Fatalf("the refusal was not said:\n%s", s)
	}
}

func TestAProposedCardShowsThePersonsWords(t *testing.T) {
	p, _ := orchestrating(t, PtBR, &fakeOrchestration{})
	c := threeMerges(100)
	c.ID, c.Proposed, c.From, c.Why = "card-2", true, "manda os dois e devolve o terceiro", "li duas aprovações"
	c.Items[0].Selected, c.Items[0].SelectedNote = "return", "falta teste"
	p.model = apply(t, p.model, ev(t, 2, protocol.EventQuestionCard, c))
	s := streamText(p.model)
	for _, want := range []string{"Você pediu: manda os dois e devolve o terceiro", "li duas aprovações", "(•) " + Text(PtBR).AnswerReturn, "nota: falta teste"} {
		if !strings.Contains(s, want) {
			t.Errorf("the proposed card does not show %q:\n%s", want, s)
		}
	}
}

func TestEscLetsGoOfTheCardAndItStaysInTheTranscript(t *testing.T) {
	p := withCard(t, En, &fakeOrchestration{})
	p.Update(ctrl('q'))
	p.Update(special(tea.KeyEscape))
	if p.model.Card != nil {
		t.Fatal("esc did not give the keyboard back")
	}
	if !strings.Contains(streamText(p.model), "1 of 3") {
		t.Fatalf("the card left the transcript:\n%s", streamText(p.model))
	}
}

func TestAnUnreadableCardIsSaid(t *testing.T) {
	m := apply(t, NewModel("orq", "/w", "m", "workspace-write", En), protocol.Event{Seq: 1, SessionID: "orq", Type: protocol.EventQuestionCard, Payload: []byte(`{"id":7}`)})
	if len(m.Entries) != 1 || m.Entries[0].Kind != KindError {
		t.Fatalf("an unreadable card was not said: %+v", m.Entries)
	}
}
