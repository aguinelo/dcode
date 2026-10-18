package tui

import (
	"encoding/json"
	"fmt"
	"slices"

	tea "charm.land/bubbletea/v2"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The orchestrator's question card in its transcript (orchestration .p §5.5):
// the questions of one kind that wait together, drawn where the card was
// said and changed there in place, one item at a time. ^Q puts the keyboard
// on it; what the person picks is a draft until the last item confirms it,
// in one POST with every answer and the items skipped.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §5.5.

// CardFocus is the card ^Q put the keyboard on, and what the person did with
// it so far. Picks an item does not have keep its selected answer.
type CardFocus struct {
	CardID  string
	Index   int
	Picks   map[string]cardPick
	Skipped []string
	// Seen are the items the person was shown: only those are answered on
	// confirm, the rest go as skipped — nothing is answered unseen.
	Seen []string
	// Sending is true while the confirm is in flight.
	Sending bool
	// Results are what became of each answer of the last confirm.
	Results map[string]protocol.AnswerResult
}

// cardPick is an option's id, or AnswerOther with the person's words.
type cardPick struct{ answer, note string }

// cardAnsweredMsg is the daemon's answer to a confirm.
type cardAnsweredMsg struct {
	cardID string
	resp   protocol.AnswerQuestionsResponse
	err    error
}

// applyCard folds a question.card into the transcript: the card's entry
// changes in place unless it already holds a newer version, and a card not
// seen before is drawn where it is said. A payload that does not read is said.
func (m Model) applyCard(ev protocol.Event) Model {
	if ev.Type != protocol.EventQuestionCard {
		return m
	}
	var c protocol.QuestionCard
	if err := json.Unmarshal(ev.Payload, &c); err != nil || c.ID == "" {
		if err == nil {
			err = fmt.Errorf("a card with no id")
		}
		m.Entries = append(m.Entries, Entry{
			Kind: KindError, IsError: true, Expanded: true, Seq: ev.Seq,
			Summary: fmt.Sprintf(Text(m.Lang).AgentsUnreadable, ev.Type, err),
		})
		return m
	}
	return m.withCards([]protocol.QuestionCard{c}, ev.Seq)
}

// withCards puts cards in the transcript as the daemon holds them.
func (m Model) withCards(cards []protocol.QuestionCard, seq uint64) Model {
	for _, c := range cards {
		c := c
		i := slices.IndexFunc(m.Entries, func(e Entry) bool { return e.Kind == KindCard && e.Card != nil && e.Card.ID == c.ID })
		if i < 0 {
			m.Entries = append(m.Entries, Entry{Kind: KindCard, Card: &c, Seq: seq, Expanded: true})
			continue
		}
		if m.Entries[i].Card.Updated.After(c.Updated) {
			continue
		}
		entries := append([]Entry(nil), m.Entries...)
		entries[i].Card = &c
		m.Entries = entries
	}
	return m
}

// cardOf is the card of id as the transcript holds it now.
func (m Model) cardOf(id string) *protocol.QuestionCard {
	for i := len(m.Entries) - 1; i >= 0; i-- {
		if e := m.Entries[i]; e.Kind == KindCard && e.Card != nil && e.Card.ID == id {
			return e.Card
		}
	}
	return nil
}

func cardNeedsPerson(c *protocol.QuestionCard) bool { return c.Closed == nil && c.Pending > 0 }

func itemOpen(it protocol.CardItem) bool { return it.State != protocol.CardItemAnswered }

// openCard is the latest card that still needs the person, or nil.
func (m Model) openCard() *protocol.QuestionCard {
	for i := len(m.Entries) - 1; i >= 0; i-- {
		if e := m.Entries[i]; e.Kind == KindCard && e.Card != nil && cardNeedsPerson(e.Card) {
			return e.Card
		}
	}
	return nil
}

// focusOn opens a card on its first item that still waits.
func focusOn(c *protocol.QuestionCard) *CardFocus {
	f := &CardFocus{CardID: c.ID, Picks: map[string]cardPick{}}
	for i, it := range c.Items {
		if itemOpen(it) {
			f.Index = i
			break
		}
	}
	f.see(c)
	return f
}

func (f *CardFocus) see(c *protocol.QuestionCard) {
	if f.Index < len(c.Items) && !slices.Contains(f.Seen, c.Items[f.Index].ID) {
		f.Seen = append(f.Seen, c.Items[f.Index].ID)
	}
}

func (f *CardFocus) item(c *protocol.QuestionCard) (protocol.CardItem, bool) {
	if len(c.Items) == 0 {
		return protocol.CardItem{}, false
	}
	return c.Items[min(f.Index, len(c.Items)-1)], true
}

func (f *CardFocus) last(c *protocol.QuestionCard) bool { return f.Index >= len(c.Items)-1 }

// pick is an item's pick: what the person chose, or what the daemon selected
// — the recommendation, or the proposal with its note — or the words of an
// item that was relayed.
func (f *CardFocus) pick(it protocol.CardItem) cardPick {
	if p, ok := f.Picks[it.ID]; ok {
		return p
	}
	if it.State == protocol.CardItemRelayed && it.Other != "" {
		return cardPick{protocol.AnswerOther, it.Other}
	}
	sel := it.Selected
	if sel == "" {
		sel = it.Recommended
	}
	return cardPick{sel, it.SelectedNote}
}

// cardChoices are what ↑ and ↓ walk: the options, then "Outro" when the item takes words.
func cardChoices(it protocol.CardItem) []string {
	ids := make([]string, 0, len(it.Options)+1)
	for _, o := range it.Options {
		ids = append(ids, o.ID)
	}
	if it.AllowOther {
		ids = append(ids, protocol.AnswerOther)
	}
	return ids
}

func (f *CardFocus) goTo(c *protocol.QuestionCard, i int) {
	f.Index = max(0, min(len(c.Items)-1, i))
	f.see(c)
}

func (f *CardFocus) choose(c *protocol.QuestionCard, answer, note string) {
	it, ok := f.item(c)
	if !ok || !itemOpen(it) {
		return
	}
	f.Picks[it.ID] = cardPick{answer, note}
	f.Skipped = slices.DeleteFunc(f.Skipped, func(id string) bool { return id == it.ID })
}

func (f *CardFocus) step(c *protocol.QuestionCard, by int) {
	it, ok := f.item(c)
	if !ok || !itemOpen(it) {
		return
	}
	ch := cardChoices(it)
	at := max(0, min(len(ch)-1, slices.Index(ch, f.pick(it).answer)+by))
	if len(ch) > 0 && ch[at] != f.pick(it).answer {
		f.choose(c, ch[at], "")
	}
}

func (f *CardFocus) skip(c *protocol.QuestionCard) {
	if it, ok := f.item(c); ok && itemOpen(it) && !slices.Contains(f.Skipped, it.ID) {
		f.Skipped = append(f.Skipped, it.ID)
	}
	f.goTo(c, f.Index+1)
}

// blocked says why the card cannot be confirmed yet: "Outro" with no words.
func (f *CardFocus) blocked(c *protocol.QuestionCard) string {
	for _, it := range c.Items {
		if !itemOpen(it) || !slices.Contains(f.Seen, it.ID) || slices.Contains(f.Skipped, it.ID) {
			continue
		}
		if p := f.pick(it); p.answer == protocol.AnswerOther && p.note == "" {
			if it.Header != "" {
				return it.Header
			}
			return it.Text
		}
	}
	return ""
}

// batch is the confirm: every item seen and not skipped, with its pick; the
// rest in skipped, where they keep waiting.
func (f *CardFocus) batch(c *protocol.QuestionCard) protocol.AnswerQuestionsRequest {
	req := protocol.AnswerQuestionsRequest{CardID: c.ID, Answers: []protocol.AnswerItem{}}
	for _, it := range c.Items {
		if !itemOpen(it) {
			continue
		}
		if !slices.Contains(f.Seen, it.ID) || slices.Contains(f.Skipped, it.ID) {
			req.Skipped = append(req.Skipped, it.ID)
			continue
		}
		p := f.pick(it)
		req.Answers = append(req.Answers, protocol.AnswerItem{QuestionID: it.ID, Answer: p.answer, Note: p.note})
	}
	return req
}

// recommendAll is "apply the recommendation to all": every item still waiting.
func recommendAll(c *protocol.QuestionCard) protocol.AnswerQuestionsRequest {
	req := protocol.AnswerQuestionsRequest{CardID: c.ID, Answers: []protocol.AnswerItem{}}
	for _, it := range c.Items {
		if itemOpen(it) {
			req.Answers = append(req.Answers, protocol.AnswerItem{QuestionID: it.ID, Answer: it.Recommended})
		}
	}
	return req
}

// ---------- the keyboard ----------

// focusCard puts the keyboard on the latest open card and scrolls to it.
// False when there is none, and ^Q answers the oldest question as before.
func (p *program) focusCard() bool {
	c := p.model.openCard()
	if c == nil {
		return false
	}
	p.model.Card = focusOn(c)
	p.scrollToCard(c.ID)
	return true
}

// scrollToCard brings the card's first row into view: the rows of everything
// before it are where it starts.
func (p *program) scrollToCard(id string) {
	i := slices.IndexFunc(p.model.Entries, func(e Entry) bool { return e.Kind == KindCard && e.Card != nil && e.Card.ID == id })
	if i < 0 {
		return
	}
	before := p.model
	before.Entries = p.model.Entries[:i]
	top := len(StreamLines(before, p.geo))
	if top >= MaxScroll(len(StreamLines(p.model, p.geo)), BodyHeight(p.model, p.geo)) {
		p.model = p.model.ScrollToBottom(p.geo)
		return
	}
	p.model.ScrollTop, p.model.Follow = top, false
}

// onCardKey owns the keyboard while a card has it. Every other key is
// swallowed, as the question overlay does: half an answer must not leak into
// the composer. Esc gives the keyboard back; the card stays in the stream.
func (p *program) onCardKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	f := p.model.Card
	c := p.model.cardOf(f.CardID)
	if c == nil || !cardNeedsPerson(c) {
		p.model.Card = nil
		return p, nil
	}
	it, _ := f.item(c)
	typing := itemOpen(it) && it.AllowOther && f.pick(it).answer == protocol.AnswerOther
	switch s := k.String(); s {
	case "esc", "ctrl+c", "ctrl+q":
		p.model.Card = nil
	case "up":
		f.step(c, -1)
	case "down":
		f.step(c, 1)
	case "left":
		f.goTo(c, f.Index-1)
	case "right":
		f.goTo(c, f.Index+1)
	case "enter", "ctrl+enter", "super+enter":
		if !f.last(c) {
			f.goTo(c, f.Index+1)
			return p, nil
		}
		return p, p.confirmCard(c, f.batch(c))
	case "backspace":
		if typing {
			r := []rune(f.pick(it).note)
			if len(r) > 0 {
				f.choose(c, protocol.AnswerOther, string(r[:len(r)-1]))
			}
		}
	default:
		if typing && k.Text != "" {
			f.choose(c, protocol.AnswerOther, f.pick(it).note+k.Text)
			return p, nil
		}
		switch {
		case s == "s":
			last := f.last(c)
			f.skip(c)
			if last {
				return p, p.confirmCard(c, f.batch(c))
			}
		case s == "a":
			return p, p.sendCard(c, recommendAll(c))
		case len(s) == 1 && s[0] >= '1' && s[0] <= '9':
			if ch := cardChoices(it); int(s[0]-'1') < len(ch) && itemOpen(it) {
				f.choose(c, ch[s[0]-'1'], "")
			}
		}
	}
	return p, nil
}

// confirmCard sends the draft, unless a confirm is in flight or "Outro" has
// no words yet, which is said.
func (p *program) confirmCard(c *protocol.QuestionCard, req protocol.AnswerQuestionsRequest) tea.Cmd {
	if missing := p.model.Card.blocked(c); missing != "" {
		p.model.Flash = fmt.Sprintf(cardText(p.model.Lang).OtherEmpty, missing)
		return nil
	}
	return p.sendCard(c, req)
}

// sendCard POSTs a batch, once at a time.
func (p *program) sendCard(c *protocol.QuestionCard, req protocol.AnswerQuestionsRequest) tea.Cmd {
	f := p.model.Card
	if f.Sending {
		return nil
	}
	f.Sending = true
	o, id := p.opts.Orchestration, p.orch.id
	return func() tea.Msg {
		resp, err := o.AnswerQuestions(p.ctx, id, req)
		return cardAnsweredMsg{cardID: c.ID, resp: resp, err: err}
	}
}

// onCardAnswered takes the daemon's cards into the transcript, takes the
// answered questions out of the list, and says what became of each answer
// that did not go through. A batch refused whole is said, and the card keeps
// the keyboard for another try.
func (p *program) onCardAnswered(msg cardAnsweredMsg) {
	t := cardText(p.model.Lang)
	if f := p.model.Card; f != nil && f.CardID == msg.cardID {
		f.Sending = false
	}
	if msg.err != nil {
		p.note(fmt.Sprintf(t.Failed, msg.err))
		return
	}
	p.model = p.model.withCards(msg.resp.Cards, 0)
	results := map[string]protocol.AnswerResult{}
	for _, r := range msg.resp.Results {
		results[r.QuestionID] = r
		switch r.Status {
		case protocol.CardItemAnswered:
			p.dropQuestion(r.QuestionID)
		case protocol.AnswerRefused:
			p.note(fmt.Sprintf(t.Refused, r.QuestionID, r.Code, r.Message))
		}
	}
	f := p.model.Card
	if f == nil || f.CardID != msg.cardID {
		return
	}
	c := p.model.cardOf(msg.cardID)
	if c == nil || !cardNeedsPerson(c) {
		p.model.Card = nil
		return
	}
	// What still waits opens again, the daemon's selections as the picks.
	next := focusOn(c)
	next.Results = results
	p.model.Card = next
}
