package orchestrate

import (
	"fmt"
	"slices"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The cards the person decides in (owner's decision of 2026-10-18): the
// questions of one kind that wait together are one card, shown in the
// orchestrator's conversation, each item with its recommendation selected.
// Grouping is the harness's rule, not a decision of anyone's: it follows
// question.asked and needs no line of its own. A card is a view of the queue,
// and answering an item is answering its question.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §5.5.

// Card is a card as state.json keeps it: the questions by id, with where
// each one is in the card. The wire's QuestionCard joins it with the queue.
type Card struct {
	ID       string      `json:"id"`
	Kind     string      `json:"kind"`
	Title    string      `json:"title"`
	Proposed bool        `json:"proposed,omitempty"`
	From     string      `json:"from,omitempty"`
	Why      string      `json:"why,omitempty"`
	Items    []CardEntry `json:"items"`
	Created  time.Time   `json:"created"`
	Updated  time.Time   `json:"updated"`
	Closed   *time.Time  `json:"closed,omitempty"`
}

// CardEntry is one item of a Card.
type CardEntry struct {
	Question     string `json:"question"`
	State        string `json:"state"`
	Selected     string `json:"selected"`
	SelectedNote string `json:"selected_note,omitempty"`
	Other        string `json:"other,omitempty"`
}

// cardTitles are the harness's titles of a card, by kind; a client may say
// them in the person's language by the kind.
var cardTitles = map[string]string{
	"merge":   "Merges ready",
	"design":  "Design questions",
	"ceiling": "Token ceiling",
}

// proposedTitle is the title of a card the orchestrator proposed.
const proposedTitle = "Proposed answers"

func cloneCards(cards []Card) []Card {
	if cards == nil {
		return nil
	}
	out := make([]Card, len(cards))
	for i, c := range cards {
		c.Items = append([]CardEntry(nil), c.Items...)
		out[i] = c
	}
	return out
}

// joinCard puts q in the open card of its kind, or in a new one when none is
// open, and returns the card's id. Only a card the harness made takes a
// question; a proposed one holds what it was proposed with.
func joinCard(s *State, q protocol.Question) string {
	at := q.Asked
	entry := CardEntry{Question: q.ID, State: protocol.CardItemPending, Selected: q.Recommended}
	for i := len(s.Cards) - 1; i >= 0; i-- {
		c := &s.Cards[i]
		if c.Kind == q.Kind && !c.Proposed && c.Closed == nil {
			c.Items, c.Updated = append(c.Items, entry), at
			return c.ID
		}
	}
	title := cardTitles[q.Kind]
	if title == "" {
		title = "Questions"
	}
	id := fmt.Sprintf("c%d", len(s.Cards)+1)
	s.Cards = append(s.Cards, Card{ID: id, Kind: q.Kind, Title: title, Items: []CardEntry{entry}, Created: at, Updated: at})
	return id
}

// Cards are the project's cards as the wire shows them, open and closed.
func (o *Orchestrator) Cards() []protocol.QuestionCard {
	st := o.snapshot()
	out := make([]protocol.QuestionCard, 0, len(st.Cards))
	for _, c := range st.Cards {
		out = append(out, materialize(st, c))
	}
	return out
}

// CardsOf is the cards of the orchestrator sessionID; held false for a
// session that conducts nothing here.
func (r *Registry) CardsOf(sessionID string) (cards []protocol.QuestionCard, held bool) {
	if o := r.orchestratorOf(sessionID); o != nil {
		return o.Cards(), true
	}
	return nil, false
}

// materialize joins a card with the queue: each item is its question as it
// is now, and a question answered anywhere is answered in every card.
func materialize(st State, c Card) protocol.QuestionCard {
	out := protocol.QuestionCard{ID: c.ID, Kind: c.Kind, Title: c.Title, Proposed: c.Proposed, From: c.From, Why: c.Why,
		Items: make([]protocol.CardItem, 0, len(c.Items)), Created: c.Created, Updated: c.Updated, Closed: c.Closed}
	for _, e := range c.Items {
		q, _ := questionIn(st, e.Question)
		state := e.State
		if q.Answered != nil {
			state = protocol.CardItemAnswered
		}
		if state != protocol.CardItemAnswered {
			out.Pending++
		}
		out.Items = append(out.Items, protocol.CardItem{Question: q, State: state,
			Selected: e.Selected, SelectedNote: e.SelectedNote, Other: e.Other})
	}
	return out
}

func questionIn(st State, id string) (protocol.Question, bool) {
	for _, q := range st.Questions {
		if q.ID == id {
			return q, true
		}
	}
	return protocol.Question{}, false
}

func cardIn(st State, id string) (Card, bool) {
	for _, c := range st.Cards {
		if c.ID == id {
			return c, true
		}
	}
	return Card{}, false
}

// cardsWith are the ids of the cards question qid is in.
func cardsWith(st State, qid string) []string {
	var ids []string
	for _, c := range st.Cards {
		if slices.ContainsFunc(c.Items, func(e CardEntry) bool { return e.Question == qid }) {
			ids = append(ids, c.ID)
		}
	}
	return ids
}

// markItems sets, in every card qid is in — or only in card when it is not
// empty — the item of qid with set, and closes a card whose every question
// is answered. It returns the cards it changed.
func markItems(s *State, qid, card string, at time.Time, set func(*CardEntry)) []string {
	var changed []string
	for i := range s.Cards {
		c := &s.Cards[i]
		if card != "" && c.ID != card {
			continue
		}
		hit := false
		for j := range c.Items {
			if c.Items[j].Question == qid {
				set(&c.Items[j])
				hit = true
			}
		}
		if !hit {
			continue
		}
		c.Updated = at
		if c.Closed == nil && allAnswered(*s, *c) {
			closed := at
			c.Closed = &closed
		}
		changed = append(changed, c.ID)
	}
	return changed
}

func allAnswered(st State, c Card) bool {
	for _, e := range c.Items {
		if q, _ := questionIn(st, e.Question); q.Answered == nil {
			return false
		}
	}
	return true
}

// settle is a question that got its answer: answered in every card it is in,
// and the cards whose every item is answered close. Bookkeeping that follows
// question.answered, so no line of its own.
func (o *Orchestrator) settle(qid string) {
	var changed []string
	at := o.reg.now().UTC()
	if err := o.apply(func(s *State) {
		changed = markItems(s, qid, "", at, func(e *CardEntry) { e.State = protocol.CardItemAnswered })
	}); err != nil {
		o.sayError(protocol.CodeInternal, fmt.Sprintf("question %s is answered, and its cards could not be kept so: %v", qid, err))
		return
	}
	o.emitCards(changed...)
}

// emitCards sends question.card for each of the cards, as they are now.
func (o *Orchestrator) emitCards(ids ...string) {
	if len(ids) == 0 {
		return
	}
	st := o.snapshot()
	for _, id := range ids {
		if c, ok := cardIn(st, id); ok {
			o.emit(protocol.EventQuestionCard, materialize(st, c))
		}
	}
}

// cardFor puts a question the merge step asked in its card.
func (o *Orchestrator) cardFor(q protocol.Question) {
	var card string
	if err := o.apply(func(s *State) { card = joinCard(s, q) }); err != nil {
		o.sayError(protocol.CodeInternal, fmt.Sprintf("question %s is asked, and is in no card: %v", q.ID, err))
		return
	}
	o.emitCards(card)
}
