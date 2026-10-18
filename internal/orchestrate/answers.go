package orchestrate

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The person's answers, one or a card's worth in one POST. Only the person
// answers: these are reached from the routes alone, and no tool of the
// orchestrator's reaches them (ADR-02).
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §5.1, §5.5.

// Respond is one answer of the person's: an option, or their own words. The
// words answer a design question; on a merge or a ceiling question, whose
// answers the harness acts on, they go to the orchestrator and the question
// keeps waiting. status is the item's new state.
func (o *Orchestrator) Respond(qid, answer, note string) (q protocol.Question, status string, err error) {
	if answer != protocol.AnswerOther {
		q, err = o.Answer(qid, answer, note)
		return q, protocol.CardItemAnswered, err
	}
	q, ok := questionIn(o.snapshot(), qid)
	switch {
	case !ok:
		return q, "", protocol.Errorf(protocol.CodeQuestionNotFound, "this project has no question %s", qid)
	case q.Answered != nil:
		return q, "", protocol.Errorf(protocol.CodeQuestionAlreadyAnswered, "question %s was already answered %q", qid, q.Answer)
	case !q.AllowOther:
		return q, "", protocol.Errorf(protocol.CodeInvalidInput, "question %s takes one of its options only", qid)
	case strings.TrimSpace(note) == "":
		return q, "", protocol.Errorf(protocol.CodeInvalidInput, "an answer in the person's own words needs the words, as the note")
	case q.Kind == "design":
		q, err = o.Answer(qid, answer, note)
		return q, protocol.CardItemAnswered, err
	}
	return q, protocol.CardItemRelayed, o.relay(q, strings.TrimSpace(note))
}

// relay hands the person's words on a question the harness acts on to the
// orchestrator, which proposes the answers they mean; the question waits.
func (o *Orchestrator) relay(q protocol.Question, words string) error {
	data, _ := json.Marshal(map[string]string{"question": q.ID, "words": words})
	var changed []string
	at := o.reg.now().UTC()
	if err := o.commit(Decision{Kind: "question.relayed", Child: q.ChildID, Data: data,
		Why: "the person answered " + q.ID + " in their own words; the harness acts only on an option, so the words go to the orchestrator and the question waits"},
		func(s *State) {
			changed = markItems(s, q.ID, "", at, func(e *CardEntry) { e.State, e.Other = protocol.CardItemRelayed, words })
		}); err != nil {
		return protocol.Errorf(protocol.CodeInternal, "the person's words did not reach the log: %v", err)
	}
	o.emitCards(changed...)
	o.Report(fmt.Sprintf("The person answered question %s (%s) in their own words instead of an option: %q. "+
		"It still waits. Call propose_answers with the answers they mean; only the person's confirmation answers it.", q.ID, q.Kind, words))
	return nil
}

// skip is an item the person passed over when confirming card: it is said
// skipped there, and its question keeps waiting.
func (o *Orchestrator) skip(card, qid string) error {
	st := o.snapshot()
	c, _ := cardIn(st, card)
	q, ok := questionIn(st, qid)
	switch {
	case !ok || !containsEntry(c, qid):
		return protocol.Errorf(protocol.CodeQuestionNotFound, "card %s has no question %s", card, qid)
	case q.Answered != nil:
		return protocol.Errorf(protocol.CodeQuestionAlreadyAnswered, "question %s was already answered %q", qid, q.Answer)
	}
	data, _ := json.Marshal(map[string]string{"card": card, "question": qid})
	var changed []string
	at := o.reg.now().UTC()
	if err := o.commit(Decision{Kind: "question.skipped", Child: q.ChildID, Data: data,
		Why: "the person confirmed card " + card + " and skipped " + qid + ", which still waits"},
		func(s *State) {
			changed = markItems(s, qid, card, at, func(e *CardEntry) { e.State = protocol.CardItemSkipped })
		}); err != nil {
		return protocol.Errorf(protocol.CodeInternal, "the skip did not reach the log: %v", err)
	}
	o.emitCards(changed...)
	return nil
}

func containsEntry(c Card, qid string) bool {
	for _, e := range c.Items {
		if e.Question == qid {
			return true
		}
	}
	return false
}

// AnswersOf is the person's batch POST to the orchestrator sessionID.
func (r *Registry) AnswersOf(sessionID string, req protocol.AnswerQuestionsRequest) (protocol.AnswerQuestionsResponse, error) {
	o := r.orchestratorOf(sessionID)
	if o == nil {
		return protocol.AnswerQuestionsResponse{}, protocol.Errorf(protocol.CodeSessionNotFound,
			"session %s conducts no project in this daemon, so it has no questions", sessionID)
	}
	return o.AnswerAll(req)
}

// AnswerAll takes a card's worth of the person's answers at once: each one on
// its own, in the order sent, the first answer to a question winning inside
// the batch too; then the skips. A batch with nothing in it, skips without
// their card, or a card that is not there is refused whole, before anything
// is taken.
func (o *Orchestrator) AnswerAll(req protocol.AnswerQuestionsRequest) (protocol.AnswerQuestionsResponse, error) {
	resp := protocol.AnswerQuestionsResponse{Results: []protocol.AnswerResult{}, Cards: []protocol.QuestionCard{}}
	switch {
	case len(req.Answers) == 0 && len(req.Skipped) == 0:
		return resp, protocol.Errorf(protocol.CodeInvalidInput, "a batch carries at least one answer or one skip")
	case len(req.Skipped) > 0 && req.CardID == "":
		return resp, protocol.Errorf(protocol.CodeInvalidInput, "a skip is made in a card: name it with card_id")
	}
	if req.CardID != "" {
		if _, ok := cardIn(o.snapshot(), req.CardID); !ok {
			return resp, protocol.Errorf(protocol.CodeInvalidInput, "this project has no card %s", req.CardID)
		}
	}
	touched := map[string]bool{}
	for _, a := range req.Answers {
		q, status, err := o.Respond(a.QuestionID, a.Answer, a.Note)
		resp.Results = append(resp.Results, result(a.QuestionID, status, q, err))
		touched[a.QuestionID] = true
	}
	for _, qid := range req.Skipped {
		err := o.skip(req.CardID, qid)
		q, _ := questionIn(o.snapshot(), qid)
		resp.Results = append(resp.Results, result(qid, protocol.CardItemSkipped, q, err))
		touched[qid] = true
	}
	st := o.snapshot()
	for _, c := range st.Cards {
		if c.ID == req.CardID || slicesAny(c.Items, touched) {
			resp.Cards = append(resp.Cards, materialize(st, c))
		}
	}
	return resp, nil
}

func slicesAny(items []CardEntry, ids map[string]bool) bool {
	for _, e := range items {
		if ids[e.Question] {
			return true
		}
	}
	return false
}

// result is one answer's outcome: its status, or refused with the code.
func result(qid, status string, q protocol.Question, err error) protocol.AnswerResult {
	r := protocol.AnswerResult{QuestionID: qid, Status: status}
	if q.ID != "" {
		r.Question = &q
	}
	if err != nil {
		r.Status, r.Code, r.Message = protocol.AnswerRefused, protocol.CodeInternal, err.Error()
		var e *protocol.Error
		if errors.As(err, &e) {
			r.Code, r.Message = e.Code, e.Message
		}
	}
	return r
}
