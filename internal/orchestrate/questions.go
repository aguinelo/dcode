package orchestrate

import (
	"fmt"
	"strings"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The questions that wait for the person: the orchestrator's design
// questions, the ceiling's, and the merge step's, one queue per project that
// the routes read and answer. Only the person answers (ADR-02).
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §5.1, §5.2, §10.

// Ask is the orchestrator's ask: a design question for the person, with the
// option it recommends and why. It returns the question's id at once; the
// answer comes back as a report.
func (o *Orchestrator) Ask(text string, options []protocol.QuestionOption, recommended, why, childID string) (string, error) {
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
		seen[opt.ID], known = true, known || opt.ID == recommended
	}
	if !known {
		return "", protocol.Errorf(protocol.CodeInvalidInput, "the recommended %q is not one of the options", recommended)
	}
	if childID != "" && o.child(childID).ID == "" {
		return "", protocol.Errorf(protocol.CodeSessionNotFound, "this orchestrator has no child %s", childID)
	}
	q, err := o.ask(protocol.Question{Kind: "design", Text: text, Recommended: recommended, Why: why,
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
// orchestrator sessionID. The first answer wins.
func (r *Registry) AnswerOf(sessionID, qid, answer, note string) (protocol.Question, error) {
	o := r.orchestratorOf(sessionID)
	if o == nil {
		return protocol.Question{}, protocol.Errorf(protocol.CodeSessionNotFound,
			"session %s conducts no project in this daemon, so it has no questions", sessionID)
	}
	if strings.TrimSpace(answer) == "" {
		return protocol.Question{}, protocol.Errorf(protocol.CodeInvalidInput, "an answer names one of the question's options")
	}
	q, err := o.Answer(qid, answer, note)
	if err != nil {
		return q, err
	}
	if q.Answer == "" {
		return q, fmt.Errorf("orchestrate: question %s came back unanswered", qid)
	}
	return q, nil
}
