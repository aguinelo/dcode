package orchestrate

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aguinelo/dcode/internal/protocol"
)

// What the orchestrator does for the person's decision, and nothing more: it
// writes what changed into a waiting question, and turns the person's words
// into a proposed card. It never answers — a proposal is a card whose
// selections are its reading, and only the person's POST of them answers.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §5.5, §10.

// The bounds of a summary: two or three lines, a few sentences.
const (
	summaryMaxLines = 3
	summaryMaxBytes = 800
)

// Summarize attaches the orchestrator's lines on what changed to a question
// that still waits, in every card it is in.
func (o *Orchestrator) Summarize(qid, summary string) error {
	summary = strings.TrimSpace(summary)
	switch {
	case summary == "":
		return protocol.Errorf(protocol.CodeInvalidInput, "a summary says what changed, in two or three lines")
	case strings.Count(summary, "\n")+1 > summaryMaxLines:
		return protocol.Errorf(protocol.CodeInvalidInput, "a summary is at most %d lines; the evidence beside it says the rest", summaryMaxLines)
	case len(summary) > summaryMaxBytes:
		return protocol.Errorf(protocol.CodeInvalidInput, "a summary is at most %d bytes, and this one is %d", summaryMaxBytes, len(summary))
	}
	q, ok := questionIn(o.snapshot(), qid)
	switch {
	case !ok:
		return protocol.Errorf(protocol.CodeQuestionNotFound, "this project has no question %s", qid)
	case q.Answered != nil:
		return protocol.Errorf(protocol.CodeQuestionAlreadyAnswered, "question %s was already answered %q; a summary is for one that waits", qid, q.Answer)
	}
	data, _ := json.Marshal(map[string]string{"question": qid, "summary": summary})
	d := Decision{Kind: "question.summarized", Child: q.ChildID, Data: data,
		Why: "the orchestrator wrote what changed, for the person to read beside the evidence"}
	if q.Kind == "merge" {
		mg, _, err := o.merges()
		if err != nil {
			return protocol.Errorf(protocol.CodeInternal, "the merge step could not open: %v", err)
		}
		if err := mg.summarize(qid, summary, d); err != nil {
			return err
		}
		o.saveMerges(mg)
	} else if err := o.commit(d, func(s *State) {
		for i := range s.Questions {
			if s.Questions[i].ID == qid {
				s.Questions[i].Summary = summary
			}
		}
	}); err != nil {
		return protocol.Errorf(protocol.CodeInternal, "the summary did not reach the log: %v", err)
	}
	o.emitCards(cardsWith(o.snapshot(), qid)...)
	return nil
}

// summarize is the merge step's half: its copy of the question takes the
// summary, after the line, so the next save of the merges keeps it.
func (m *Merges) summarize(qid, summary string, d Decision) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.questions {
		if m.questions[i].ID != qid {
			continue
		}
		if m.questions[i].Answered != nil {
			return protocol.Errorf(protocol.CodeQuestionAlreadyAnswered, "question %s was already answered %q", qid, m.questions[i].Answer)
		}
		if _, err := m.o.Log.Record(d); err != nil {
			return protocol.Errorf(protocol.CodeInternal, "the summary did not reach the log: %v", err)
		}
		m.questions[i].Summary = summary
		return nil
	}
	return protocol.Errorf(protocol.CodeQuestionNotFound, "the merge step has no question %s", qid)
}

// ProposeAnswers is the orchestrator's reading of the person's words as
// answers: a proposed card with them selected, for the person to confirm. It
// answers nothing. Every answer is one of its question's options, to a
// question that waits, one per question; from is the person's words.
func (o *Orchestrator) ProposeAnswers(from, why string, answers []protocol.AnswerItem) (protocol.QuestionCard, error) {
	from, why = strings.TrimSpace(from), strings.TrimSpace(why)
	switch {
	case from == "" || why == "":
		return protocol.QuestionCard{}, protocol.Errorf(protocol.CodeInvalidInput, "a proposal names the person's words it came from, and why it reads them so")
	case len(answers) == 0:
		return protocol.QuestionCard{}, protocol.Errorf(protocol.CodeInvalidInput, "a proposal proposes at least one answer")
	}
	st := o.snapshot()
	seen, kind := map[string]bool{}, ""
	entries := make([]CardEntry, 0, len(answers))
	for _, a := range answers {
		q, ok := questionIn(st, a.QuestionID)
		switch {
		case !ok:
			return protocol.QuestionCard{}, protocol.Errorf(protocol.CodeQuestionNotFound, "this project has no question %s", a.QuestionID)
		case q.Answered != nil:
			return protocol.QuestionCard{}, protocol.Errorf(protocol.CodeQuestionAlreadyAnswered, "question %s was already answered %q", q.ID, q.Answer)
		case seen[q.ID]:
			return protocol.QuestionCard{}, protocol.Errorf(protocol.CodeInvalidInput, "question %s is proposed twice; one answer each", q.ID)
		case !offers(q, a.Answer):
			return protocol.QuestionCard{}, protocol.Errorf(protocol.CodeInvalidInput,
				"%q is not one of the options of question %s; a proposal selects an option, and the person's own words stay theirs", a.Answer, q.ID)
		}
		seen[q.ID] = true
		if kind == "" {
			kind = q.Kind
		} else if kind != q.Kind {
			kind = "mixed"
		}
		entries = append(entries, CardEntry{Question: q.ID, State: protocol.CardItemPending, Selected: a.Answer, SelectedNote: strings.TrimSpace(a.Note)})
	}
	at := o.reg.now().UTC()
	card := Card{ID: fmt.Sprintf("c%d", len(st.Cards)+1), Kind: kind, Title: proposedTitle, Proposed: true,
		From: from, Why: why, Items: entries, Created: at, Updated: at}
	data, _ := json.Marshal(map[string]any{"card": card.ID, "from": from, "answers": answers})
	if err := o.commit(Decision{Kind: "answers.proposed", Data: data, Why: why}, func(s *State) {
		card.ID = fmt.Sprintf("c%d", len(s.Cards)+1)
		s.Cards = append(s.Cards, card)
	}); err != nil {
		return protocol.QuestionCard{}, protocol.Errorf(protocol.CodeInternal, "the proposal did not reach the log: %v", err)
	}
	o.emitCards(card.ID)
	got, _ := cardIn(o.snapshot(), card.ID)
	return materialize(o.snapshot(), got), nil
}
