package orchestrate

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The questions that wait for the person: the orchestrator's design
// questions, the ceiling's, and the merge step's, one queue per project that
// the routes read and answer, shown in cards (cards.go). Only the person
// answers (ADR-02).
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §5.1, §5.2, §5.5, §10.

// Ask is the orchestrator's ask: a design question for the person, with the
// option it recommends and why. It returns the question's id at once; the
// answer comes back as a report.
func (o *Orchestrator) Ask(text string, options []protocol.QuestionOption, recommended, why, childID string) (string, error) {
	return o.AskWithHeader("", text, options, recommended, why, childID)
}

// AskWithHeader is Ask with the short tag the card shows above the question;
// empty is "Design".
func (o *Orchestrator) AskWithHeader(header, text string, options []protocol.QuestionOption, recommended, why, childID string) (string, error) {
	text, why, recommended = strings.TrimSpace(text), strings.TrimSpace(why), strings.TrimSpace(recommended)
	switch {
	case text == "" || why == "":
		return "", protocol.Errorf(protocol.CodeInvalidInput, "a question needs its text and the why of its recommendation")
	case len(options) < 2:
		return "", protocol.Errorf(protocol.CodeInvalidInput, "a question offers at least two options; one is no choice")
	}
	seen, known := map[string]bool{}, false
	for _, opt := range options {
		if strings.TrimSpace(opt.ID) == "" || strings.TrimSpace(opt.Label) == "" || seen[opt.ID] {
			return "", protocol.Errorf(protocol.CodeInvalidInput, "each option needs an id of its own and a label")
		}
		if opt.ID == protocol.AnswerOther {
			return "", protocol.Errorf(protocol.CodeInvalidInput, "%q is the person's own words, not an option to offer", protocol.AnswerOther)
		}
		seen[opt.ID], known = true, known || opt.ID == recommended
	}
	if !known {
		return "", protocol.Errorf(protocol.CodeInvalidInput, "the recommended %q is not one of the options", recommended)
	}
	if childID != "" && o.child(childID).ID == "" {
		return "", protocol.Errorf(protocol.CodeSessionNotFound, "this orchestrator has no child %s", childID)
	}
	if header = strings.TrimSpace(header); header == "" {
		header = "Design"
	}
	q, err := o.ask(protocol.Question{Kind: "design", Header: header, Text: text, Recommended: recommended, Why: why,
		ChildID: childID, Options: append([]protocol.QuestionOption(nil), options...)})
	if err != nil {
		return "", protocol.Errorf(protocol.CodeInternal, "the question did not reach the log: %v", err)
	}
	return q.ID, nil
}

// QuestionsOf answers GET /sessions/{id}/questions: an orchestrator's
// questions, answered or not; held false for a session that conducts nothing
// here.
func (r *Registry) QuestionsOf(sessionID string) (questions []protocol.Question, held bool) {
	if o := r.orchestratorOf(sessionID); o != nil {
		return o.Questions(), true
	}
	return nil, false
}

// AnswerOf is the person's POST of an answer to question qid of the
// orchestrator sessionID. The first answer wins. The person's own words on a
// question the harness acts on are relayed, and the question keeps waiting.
func (r *Registry) AnswerOf(sessionID, qid, answer, note string) (protocol.Question, error) {
	o := r.orchestratorOf(sessionID)
	if o == nil {
		return protocol.Question{}, protocol.Errorf(protocol.CodeSessionNotFound,
			"session %s conducts no project in this daemon, so it has no questions", sessionID)
	}
	if strings.TrimSpace(answer) == "" {
		return protocol.Question{}, protocol.Errorf(protocol.CodeInvalidInput, "an answer names one of the question's options")
	}
	q, status, err := o.Respond(qid, answer, note)
	if err != nil {
		return q, err
	}
	if status == protocol.CardItemAnswered && q.Answer == "" {
		return q, fmt.Errorf("orchestrate: question %s came back unanswered", qid)
	}
	return q, nil
}

// Questions are the questions of the project, answered or not.
func (o *Orchestrator) Questions() []protocol.Question { return o.snapshot().Questions }

// reachCeiling says the ceiling in the log and asks its question, unless one
// is already waiting.
func (o *Orchestrator) reachCeiling(why string) {
	for _, q := range o.snapshot().Questions {
		if q.Kind == "ceiling" && q.Answer == "" {
			return
		}
	}
	st := o.snapshot()
	data, _ := json.Marshal(map[string]int64{"spent": st.Spent, "ceiling": o.limits.TokenCeiling})
	o.note(Decision{Kind: "ceiling.reached", Session: st.Project.OrchestratorID, Why: why, Data: data})
	_, _ = o.ask(protocol.Question{Kind: "ceiling", Header: "Tokens", Recommended: "stop",
		Text: fmt.Sprintf("The project has spent %d tokens, past orchestrator.token_ceiling (%d). Continue with a new window, or stop dispatching?", st.Spent, o.limits.TokenCeiling),
		Why:  "children already running finish what they do; no new child is dispatched until this is answered",
		Options: []protocol.QuestionOption{
			{ID: "continue", Label: "Continue: start a new window of tokens", Description: "The count goes back to zero and dispatching goes on."},
			{ID: "stop", Label: "Stop dispatching", Description: "No new child until you say otherwise; the running ones finish."},
		}})
}

// ask puts a question in the queue and in the open card of its kind:
// question.asked, the state, the wire. Every question takes the person's own
// words besides its options.
func (o *Orchestrator) ask(q protocol.Question) (protocol.Question, error) {
	st := o.snapshot()
	q.ID = fmt.Sprintf("q%d", len(st.Questions)+1)
	q.Asked = o.reg.now().UTC()
	q.AllowOther = true
	data, _ := json.Marshal(q)
	var card string
	if err := o.commit(Decision{Kind: "question.asked", Child: q.ChildID, Session: st.Project.OrchestratorID, Why: q.Why, Data: data},
		func(s *State) {
			s.Questions = append(s.Questions, q)
			card = joinCard(s, q)
		}); err != nil {
		o.sayError(protocol.CodeInternal, "a question did not reach the log, and is not asked: "+err.Error())
		return q, err
	}
	o.emit(protocol.EventQuestionAsked, q)
	o.emitCards(card)
	return q, nil
}

// Answer answers a question; the first answer wins. continue on the ceiling
// question opens a new window: Spent goes back to zero. The person's own
// words are an answer only where the harness acts on none (Respond).
func (o *Orchestrator) Answer(id, answer, note string) (protocol.Question, error) {
	st := o.snapshot()
	i := -1
	for j, q := range st.Questions {
		if q.ID == id {
			i = j
		}
	}
	if i < 0 {
		return protocol.Question{}, protocol.Errorf(protocol.CodeQuestionNotFound, "this project has no question %s", id)
	}
	q := st.Questions[i]
	if q.Kind == "merge" {
		got, err := o.answerMerge(q, answer, note)
		if err == nil {
			o.settle(got.ID)
		}
		return got, err
	}
	if q.Answer != "" {
		return q, protocol.Errorf(protocol.CodeQuestionAlreadyAnswered, "question %s was already answered %q", id, q.Answer)
	}
	known := q.Kind == "design" && answer == protocol.AnswerOther && q.AllowOther && strings.TrimSpace(note) != ""
	for _, opt := range q.Options {
		known = known || opt.ID == answer
	}
	if !known {
		return q, protocol.Errorf(protocol.CodeInvalidInput, "%q is not one of the options of question %s", answer, id)
	}
	now := o.reg.now().UTC()
	q.Answer, q.Note, q.Answered = answer, note, &now
	why := "the person answered " + answer
	if q.Kind == "ceiling" && answer == "continue" {
		why += ": the token window starts again from zero"
	}
	data, _ := json.Marshal(q)
	if err := o.commit(Decision{Kind: "question.answered", Session: st.Project.OrchestratorID, Child: q.ChildID, Why: why, Data: data},
		func(s *State) {
			for j := range s.Questions {
				if s.Questions[j].ID == id {
					s.Questions[j] = q
				}
			}
			if q.Kind == "ceiling" && answer == "continue" {
				s.Spent = 0
			}
		}); err != nil {
		return protocol.Question{}, protocol.Errorf(protocol.CodeInternal, "the answer did not reach the log: %v", err)
	}
	o.emit(protocol.EventQuestionAnswered, q)
	o.settle(q.ID)
	said := fmt.Sprintf("The person answered question %s (%s) with %q.", q.ID, q.Kind, answer)
	if answer == protocol.AnswerOther {
		said = fmt.Sprintf("The person answered question %s (%s) in their own words: %q.", q.ID, q.Kind, note)
	}
	o.Report(said)
	return q, nil
}
