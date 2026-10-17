package tui

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The orchestrator's live list (orchestration RN-9, .p §11): its children,
// what each is doing and costing, and what waits for the person.
//
// A fold over the wire and nothing else (D8): `child.*`, `question.*`,
// `review.verdict`, `gate.measured`, `merge.*` and `main.health` from the
// orchestrator's own log, and the daemon's `/children` and `/questions`
// answers through loadAgents. Never the text of a turn.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §6, §11.

// The eight states of a child, as the wire spells them (.p §6).
const (
	childDispatched = "dispatched"
	childWorking    = "working"
	childAwaiting   = "awaiting_person"
	childInReview   = "in_review"
	childPR         = "pr"
	childMerged     = "merged"
	childClosed     = "closed"
	childFailed     = "failed"
)

// Agent is one row of the list: the child as the daemon last said it, and the
// last refused merge, which the child itself does not carry.
type Agent struct {
	protocol.Child
	Refused *protocol.MergeRefused
}

// Agents is the list. Children in the order they started; Questions are the
// ones still waiting, oldest first.
type Agents struct {
	Children  []Agent
	Questions []protocol.Question
	Main      *protocol.MainHealth
}

// isAgentEvent reports whether t belongs to the list.
func isAgentEvent(t protocol.EventType) bool {
	switch t {
	case protocol.EventChildStarted, protocol.EventChildState, protocol.EventChildCompleted,
		protocol.EventReviewVerdict, protocol.EventGateMeasured,
		protocol.EventQuestionAsked, protocol.EventQuestionAnswered,
		protocol.EventMergeCompleted, protocol.EventMergeRefused, protocol.EventMainHealth:
		return true
	}
	return false
}

// applyAgents folds one orchestrator event into the list. A payload that does
// not read is said in the stream: a list that silently skipped an event would
// show a state the daemon has already left.
func (m Model) applyAgents(ev protocol.Event) Model {
	if !isAgentEvent(ev.Type) {
		return m
	}
	a, err := m.Agents.apply(ev)
	if err != nil {
		m.Entries = append(m.Entries, Entry{
			Kind: KindError, IsError: true, Expanded: true, Seq: ev.Seq,
			Summary: fmt.Sprintf(Text(m.Lang).AgentsUnreadable, ev.Type, err),
		})
		return m
	}
	m.Agents = a
	return m
}

func (a Agents) apply(ev protocol.Event) (Agents, error) {
	read := func(v any) error { return json.Unmarshal(ev.Payload, v) }
	switch ev.Type {
	case protocol.EventChildStarted, protocol.EventChildState:
		var d protocol.ChildChanged
		if err := read(&d); err != nil {
			return a, err
		}
		if d.Child.ID == "" {
			return a, fmt.Errorf("a child with no id")
		}
		return a.with(d.Child.ID, func(r *Agent) {
			refused := r.Refused
			*r = Agent{Child: d.Child, Refused: refused}
		}), nil

	case protocol.EventChildCompleted:
		var d protocol.ChildCompleted
		if err := read(&d); err != nil {
			return a, err
		}
		return a.with(d.ChildID, func(r *Agent) { r.Reason = d.Reason }), nil

	case protocol.EventReviewVerdict:
		var d protocol.ReviewVerdict
		if err := read(&d); err != nil {
			return a, err
		}
		return a.with(d.ChildID, func(r *Agent) { v := d.Verdict; r.Review = &v }), nil

	case protocol.EventGateMeasured:
		var d protocol.GateMeasured
		if err := read(&d); err != nil {
			return a, err
		}
		return a.with(d.ChildID, func(r *Agent) { g := d.GateResult; r.Gate = &g }), nil

	case protocol.EventMergeCompleted:
		var d protocol.MergeCompleted
		if err := read(&d); err != nil {
			return a, err
		}
		return a.with(d.ChildID, func(r *Agent) { r.Refused = nil }), nil

	case protocol.EventMergeRefused:
		var d protocol.MergeRefused
		if err := read(&d); err != nil {
			return a, err
		}
		return a.with(d.ChildID, func(r *Agent) { r.Refused = &d }), nil

	case protocol.EventQuestionAsked, protocol.EventQuestionAnswered:
		var q protocol.Question
		if err := read(&q); err != nil {
			return a, err
		}
		qs := make([]protocol.Question, 0, len(a.Questions)+1)
		for _, old := range a.Questions {
			if old.ID != q.ID {
				qs = append(qs, old)
			}
		}
		if ev.Type == protocol.EventQuestionAsked && q.Answered == nil {
			qs = append(qs, q)
		}
		a.Questions = qs
		return a, nil

	case protocol.EventMainHealth:
		var h protocol.MainHealth
		if err := read(&h); err != nil {
			return a, err
		}
		a.Main = &h
		return a, nil
	}
	return a, nil
}

// with copies the list and changes the row of id, adding one when the list
// has not heard of it. An event about an unseen child is a row with no state
// yet, drawn as such, rather than an event dropped.
//
// A copy every time: Apply is pure, and a Model kept by the caller must not
// see a row change under it.
func (a Agents) with(id string, change func(*Agent)) Agents {
	rows := append([]Agent(nil), a.Children...)
	i := -1
	for j := range rows {
		if rows[j].ID == id {
			i = j
			break
		}
	}
	if i < 0 {
		rows = append(rows, Agent{Child: protocol.Child{ID: id}})
		i = len(rows) - 1
	}
	change(&rows[i])
	a.Children = rows
	return a
}

// loadAgents takes the daemon's answers to `/children` and `/questions` as the
// list. A closed or merged child leaves when the daemon stops bringing it —
// its retention is the daemon's (`orchestrator.closed_retention`), never a
// clock of this client's. A failed child the client already knows stays even
// when the answer leaves it out: failed waits for someone (.p §6), and does
// not leave on its own.
func (m Model) loadAgents(children []protocol.Child, questions []protocol.Question) Model {
	old := map[string]Agent{}
	for _, r := range m.Agents.Children {
		old[r.ID] = r
	}
	seen := map[string]bool{}
	rows := make([]Agent, 0, len(children))
	for _, c := range children {
		seen[c.ID] = true
		rows = append(rows, Agent{Child: c, Refused: old[c.ID].Refused})
	}
	for _, r := range m.Agents.Children {
		if r.State == childFailed && !seen[r.ID] {
			rows = append(rows, r)
		}
	}
	m.Agents.Children = rows

	var waiting []protocol.Question
	for _, q := range questions {
		if q.Answered == nil {
			waiting = append(waiting, q)
		}
	}
	sort.SliceStable(waiting, func(i, j int) bool { return waiting[i].Asked.Before(waiting[j].Asked) })
	m.Agents.Questions = waiting
	return m
}
