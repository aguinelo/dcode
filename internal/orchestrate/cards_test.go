package orchestrate

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/tools"
)

// The person decides in the orchestrator's conversation, through a card
// (owner's decision of 2026-10-18, after a live run where three merges were
// answered agent by agent): the questions of one kind that wait together are
// one card, the recommendation selected in each; one POST confirms them all;
// the orchestrator writes what changed and proposes answers from the person's
// words, and never answers.

// cardKeys open a project whose gate is green and whose reports wait.
var cardKeys = map[string]string{"orchestrator.gate_command": "true", "orchestrator.report_debounce": "1m"}

var designOptions = []protocol.QuestionOption{{ID: "a", Label: "A", Description: "the first"}, {ID: "b", Label: "B"}}

// design asks a design question and returns its id.
func design(t *testing.T, p *project, text string) string {
	t.Helper()
	id, err := p.o.Ask(text, designOptions, "a", "fewer moving parts", "")
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// proposedAs takes a child of its own theme through the pipeline to its merge
// question, as proposed does for one.
func proposedAs(t *testing.T, p *project, title, owns string) (protocol.Child, protocol.Question) {
	t.Helper()
	gitOut(t, p.root, "checkout", "-q", "--detach")
	c := p.one(t, item(title, owns))
	put(t, c, owns+"/f.go", "package x\n")
	p.end(t, c.ID, protocol.StopDone, "Done. I ran make check and it is green.")
	p.review(t, approve)
	for _, q := range p.o.Questions() {
		if q.Kind == "merge" && q.ChildID == c.ID {
			return p.o.child(c.ID), q
		}
	}
	t.Fatalf("no merge question for %s: %+v; log %s", c.ID, p.o.Questions(), p.kinds(t))
	return protocol.Child{}, protocol.Question{}
}

func openCards(p *project, kind string) []protocol.QuestionCard {
	var out []protocol.QuestionCard
	for _, c := range p.o.Cards() {
		if c.Kind == kind && c.Closed == nil {
			out = append(out, c)
		}
	}
	return out
}

func itemOf(c protocol.QuestionCard, qid string) protocol.CardItem {
	for _, it := range c.Items {
		if it.ID == qid {
			return it
		}
	}
	return protocol.CardItem{}
}

// Spec: orchestration .p §5.5 — the questions of one kind pending together
// are one card, in the order asked, each with its recommendation selected; a
// card closes when every item is answered, and the next question opens a new
// one.
func TestPendingQuestionsOfOneKindShareOneCard(t *testing.T) {
	p := openWith(t, cardKeys)
	_, m1 := proposedAs(t, p, "index", "internal/index")
	_, m2 := proposedAs(t, p, "lists", "internal/lists")
	d1 := design(t, p, "One store or two?")

	merges := openCards(p, "merge")
	if len(merges) != 1 || len(merges[0].Items) != 2 || merges[0].Items[0].ID != m1.ID || merges[0].Items[1].ID != m2.ID {
		t.Fatalf("merge cards = %+v, want one with both merges in order", merges)
	}
	card := merges[0]
	if card.Pending != 2 || card.Proposed || card.Title == "" {
		t.Errorf("card = pending %d, proposed %v, title %q", card.Pending, card.Proposed, card.Title)
	}
	for _, it := range card.Items {
		if it.State != protocol.CardItemPending || it.Selected != it.Recommended || it.Selected != "merge" ||
			it.Header == "" || !it.AllowOther || it.Options[0].Description == "" {
			t.Errorf("item %s = %+v, want pending, the recommendation selected, a header, other allowed and described options", it.ID, it)
		}
	}
	if designs := openCards(p, "design"); len(designs) != 1 || len(designs[0].Items) != 1 || designs[0].Items[0].ID != d1 {
		t.Fatalf("design cards = %+v, want one of its own", designs)
	}
	if p.w.count(protocol.EventQuestionCard) < 3 {
		t.Errorf("question.card sent %d times, want one per change", p.w.count(protocol.EventQuestionCard))
	}

	if _, err := p.o.Answer(d1, "b", ""); err != nil {
		t.Fatal(err)
	}
	if len(openCards(p, "design")) != 0 {
		t.Fatalf("a card whose every item is answered stays open: %+v", p.o.Cards())
	}
	d2 := design(t, p, "Tabs or spaces?")
	if designs := openCards(p, "design"); len(designs) != 1 || len(designs[0].Items) != 1 || designs[0].Items[0].ID != d2 {
		t.Errorf("after the first card closed, the next question is in %+v", designs)
	}
	orq := p.o.orchestratorID()
	if cards, held := p.reg.CardsOf(orq); !held || len(cards) != 3 {
		t.Errorf("CardsOf = %d cards, %v; want the closed one and the two open", len(cards), held)
	}
}

// Spec: orchestration .p §5.1, §5.5 — the batch answers each item on its own
// and the first answer to a question wins, inside the batch too; skipped
// items keep waiting; a malformed batch is refused whole.
func TestTheBatchAnswersEachItemAndTheFirstAnswerWins(t *testing.T) {
	p := openWith(t, cardKeys)
	q1, q2, q3 := design(t, p, "One?"), design(t, p, "Two?"), design(t, p, "Three?")
	card := openCards(p, "design")[0]
	orq := p.o.orchestratorID()

	for name, req := range map[string]protocol.AnswerQuestionsRequest{
		"nothing":            {},
		"a skip and no card": {Skipped: []string{q3}},
		"a card not there":   {CardID: "c-nobody", Answers: []protocol.AnswerItem{{QuestionID: q1, Answer: "a"}}},
	} {
		if _, err := p.reg.AnswersOf(orq, req); err == nil {
			t.Errorf("%s: the batch was taken", name)
		}
	}
	if _, err := p.reg.AnswersOf("nobody", protocol.AnswerQuestionsRequest{Answers: []protocol.AnswerItem{{QuestionID: q1, Answer: "a"}}}); err == nil {
		t.Error("a batch to a session that conducts nothing was taken")
	}
	if q := p.o.Questions(); q[0].Answer != "" {
		t.Fatalf("a refused batch answered: %+v", q[0])
	}

	got, err := p.reg.AnswersOf(orq, protocol.AnswerQuestionsRequest{CardID: card.ID, Skipped: []string{q3},
		Answers: []protocol.AnswerItem{{QuestionID: q1, Answer: "a"}, {QuestionID: q1, Answer: "b"},
			{QuestionID: q2, Answer: "b", Note: "simpler"}, {QuestionID: "q-nobody", Answer: "a"}, {QuestionID: q2, Answer: "zzz"}}})
	if err != nil {
		t.Fatal(err)
	}
	want := []struct{ status, code string }{
		{protocol.CardItemAnswered, ""}, {protocol.AnswerRefused, protocol.CodeQuestionAlreadyAnswered},
		{protocol.CardItemAnswered, ""}, {protocol.AnswerRefused, protocol.CodeQuestionNotFound},
		{protocol.AnswerRefused, protocol.CodeQuestionAlreadyAnswered}, {protocol.CardItemSkipped, ""},
	}
	if len(got.Results) != len(want) {
		t.Fatalf("results = %+v", got.Results)
	}
	for i, w := range want {
		if r := got.Results[i]; r.Status != w.status || r.Code != w.code {
			t.Errorf("result %d = %+v, want %s %s", i, r, w.status, w.code)
		}
	}
	qs := map[string]protocol.Question{}
	for _, q := range p.o.Questions() {
		qs[q.ID] = q
	}
	if qs[q1].Answer != "a" || qs[q2].Answer != "b" || qs[q2].Note != "simpler" || qs[q3].Answer != "" {
		t.Errorf("questions after the batch: %+v", qs)
	}
	after := openCards(p, "design")
	if len(after) != 1 || after[0].Pending != 1 || itemOf(after[0], q3).State != protocol.CardItemSkipped ||
		itemOf(after[0], q1).State != protocol.CardItemAnswered {
		t.Fatalf("the card after the batch = %+v", after)
	}
	if len(got.Cards) != 1 || got.Cards[0].ID != card.ID {
		t.Errorf("the batch's cards = %+v", got.Cards)
	}
	if n := len(p.lines(t, "question.skipped")); n != 1 {
		t.Errorf("question.skipped lines = %d", n)
	}
}

// Spec: orchestration .p §5.5, §10 — propose_answers makes a proposed card
// with the answers selected, and answers nothing: no question.answered, no
// merge queued; only the person's POST of it answers.
func TestProposingAnswersMakesAProposedCardAndAnswersNothing(t *testing.T) {
	p := openWith(t, cardKeys)
	c, m := proposedAs(t, p, "index", "internal/index")
	d := design(t, p, "One store or two?")
	from := "merge em todas, e na pergunta do store fica com dois"

	for name, answers := range map[string][]protocol.AnswerItem{
		"nothing":        nil,
		"off the list":   {{QuestionID: d, Answer: "zzz"}},
		"other":          {{QuestionID: d, Answer: protocol.AnswerOther, Note: "neither"}},
		"no question":    {{QuestionID: "q-nobody", Answer: "a"}},
		"twice the same": {{QuestionID: d, Answer: "a"}, {QuestionID: d, Answer: "b"}},
	} {
		if _, err := p.o.ProposeAnswers(from, "the person said so", answers); err == nil {
			t.Errorf("%s: proposed", name)
		}
	}
	if _, err := p.o.ProposeAnswers("", "w", []protocol.AnswerItem{{QuestionID: d, Answer: "b"}}); err == nil {
		t.Error("a proposal without the person's words was made")
	}

	card, err := p.o.ProposeAnswers(from, "the person said merge all, and two stores",
		[]protocol.AnswerItem{{QuestionID: m.ID, Answer: "merge"}, {QuestionID: d, Answer: "b", Note: "two stores"}})
	if err != nil {
		t.Fatal(err)
	}
	if !card.Proposed || card.From != from || len(card.Items) != 2 || card.Items[0].Selected != "merge" ||
		card.Items[1].Selected != "b" || card.Items[1].SelectedNote != "two stores" || card.Pending != 2 {
		t.Fatalf("proposed card = %+v", card)
	}
	for _, q := range p.o.Questions() {
		if q.Answer != "" || q.Answered != nil {
			t.Errorf("a proposal answered %s: %+v", q.ID, q)
		}
	}
	if k := p.kinds(t); strings.Contains(k, "question.answered") || strings.Contains(k, "merge.queued") || !strings.Contains(k, "answers.proposed") {
		t.Errorf("log after a proposal: %s", k)
	}
	if got := p.o.child(c.ID); got.State != statePR || len(p.o.snapshot().Queue) != 0 {
		t.Errorf("after a proposal the child is %s, the queue %v", got.State, p.o.snapshot().Queue)
	}

	// The person confirms it: now, and only now, the answers are taken.
	if _, err := p.reg.AnswersOf(p.o.orchestratorID(), protocol.AnswerQuestionsRequest{CardID: card.ID,
		Answers: []protocol.AnswerItem{{QuestionID: d, Answer: "b", Note: "two stores"}}}); err != nil {
		t.Fatal(err)
	}
	for _, c := range p.o.Cards() {
		if it := itemOf(c, d); it.ID != "" && it.State != protocol.CardItemAnswered {
			t.Errorf("card %s still shows %s as %s", c.ID, d, it.State)
		}
	}
}

// Spec: orchestration .p §5.5, §10 — summarize attaches the orchestrator's
// lines to a waiting question, in every card it is in; it is no answer, and
// it survives the merges' own save.
func TestASummaryIsAttachedToAWaitingMergeQuestion(t *testing.T) {
	p := openWith(t, cardKeys)
	_, m := proposedAs(t, p, "index", "internal/index")
	for name, text := range map[string]string{
		"empty":      " ",
		"four lines": "um\ndois\ntrês\nquatro",
		"too long":   strings.Repeat("x", 801),
	} {
		if err := p.o.Summarize(m.ID, text); err == nil {
			t.Errorf("%s: summarized", name)
		}
	}
	if err := p.o.Summarize("q-nobody", "x"); err == nil {
		t.Error("a summary of no question was taken")
	}
	sum := "Conta as palavras de cada arquivo do índice.\nCom teste do caso vazio."
	if err := p.o.Summarize(m.ID, sum); err != nil {
		t.Fatal(err)
	}
	_, _ = proposedAs(t, p, "lists", "internal/lists") // the merges save their questions again
	var got protocol.Question
	for _, q := range p.o.Questions() {
		if q.ID == m.ID {
			got = q
		}
	}
	if got.Summary != sum || got.Answer != "" {
		t.Fatalf("question after summarize = %+v", got)
	}
	if it := itemOf(openCards(p, "merge")[0], m.ID); it.Summary != sum {
		t.Errorf("the card's item = %+v", it)
	}
	if n := len(p.lines(t, "question.summarized")); n != 1 {
		t.Errorf("question.summarized lines = %d", n)
	}
	if _, err := p.o.Answer(m.ID, "close", ""); err != nil {
		t.Fatal(err)
	}
	if err := p.o.Summarize(m.ID, "late"); err == nil {
		t.Error("an answered question was summarized")
	}
}

// Spec: orchestration .p §5.5, RN-5 — the evidence of a merge item is what
// the log measured — the review.verdict and gate.measured lines, with their
// seq — and never what the child said of its own gate.
func TestTheMergeEvidenceIsWhatTheLogMeasuredNeverTheClaim(t *testing.T) {
	p := openWith(t, cardKeys)
	c, q := proposedAs(t, p, "index", "internal/index")
	e := q.Evidence
	if e == nil {
		t.Fatal("a merge question without evidence")
	}
	gates, reviews := p.lines(t, "gate.measured"), p.lines(t, "review.verdict")
	gate, review := gates[len(gates)-1], reviews[len(reviews)-1]
	var measured protocol.GateResult
	if err := json.Unmarshal(gate.Measured, &measured); err != nil {
		t.Fatal(err)
	}
	if e.GateSeq != gate.Seq || e.ReviewSeq != review.Seq || e.Gate != measured.Result || e.Command != measured.Command ||
		e.Commit != c.Head || e.Commit != measured.Commit || e.Base != measured.Base || e.Verdict != tools.VerdictApprove {
		t.Errorf("evidence = %+v, want the lines %d and %d: %+v", e, gate.Seq, review.Seq, measured)
	}
	if gate.Claim == "" {
		t.Fatal("the child's claim is not in the log, so the test proves nothing")
	}
	raw, _ := json.Marshal(e)
	if strings.Contains(string(raw), "make check") {
		t.Errorf("the evidence carries the child's claim: %s", raw)
	}
}

// Spec: orchestration .p §5.5 — the person's own words: on a design question
// they are the answer; on a merge they go to the orchestrator, which
// proposes, and the question keeps waiting; words are needed.
func TestThePersonsOwnWordsAnswerADesignAndRelayAMerge(t *testing.T) {
	p := openWith(t, cardKeys)
	_, m := proposedAs(t, p, "lists", "internal/lists")
	d := design(t, p, "One store or two?")
	orq := p.o.orchestratorID()

	got, err := p.reg.AnswersOf(orq, protocol.AnswerQuestionsRequest{Answers: []protocol.AnswerItem{
		{QuestionID: d, Answer: protocol.AnswerOther},
		{QuestionID: d, Answer: protocol.AnswerOther, Note: "three, one per tenant"},
		{QuestionID: m.ID, Answer: protocol.AnswerOther, Note: "devolve pedindo mais testes"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if r := got.Results; r[0].Status != protocol.AnswerRefused || r[0].Code != protocol.CodeInvalidInput ||
		r[1].Status != protocol.CardItemAnswered || r[2].Status != protocol.CardItemRelayed {
		t.Fatalf("results = %+v", r)
	}
	for _, q := range p.o.Questions() {
		switch q.ID {
		case d:
			if q.Answer != protocol.AnswerOther || q.Note != "three, one per tenant" {
				t.Errorf("design = %+v", q)
			}
		case m.ID:
			if q.Answer != "" {
				t.Errorf("a merge was answered by the person's words: %+v", q)
			}
		}
	}
	if it := itemOf(openCards(p, "merge")[0], m.ID); it.State != protocol.CardItemRelayed || it.Other != "devolve pedindo mais testes" {
		t.Errorf("merge item = %+v", it)
	}
	p.reported(t, "devolve pedindo mais testes")
	if !strings.Contains(p.kinds(t), "question.relayed") {
		t.Errorf("log = %s", p.kinds(t))
	}
}
