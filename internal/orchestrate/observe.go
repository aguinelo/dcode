package orchestrate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/aguinelo/dcode/internal/protocol"
)

// What the harness reads of a project's sessions: every turn's tokens, for
// the ceiling, and of a child's, its questions to the person and the end of
// each turn, where the harness commits what it changed.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §5.3, §6, §7.1, §10.

// defaultConclusionMax is DelegateMaxResultBytes' default, where a child's
// conclusion is cut in child.completed.
const defaultConclusionMax = 32768

// commitDeadline bounds the harness's commit of one turn.
const commitDeadline = 2 * dispatchDeadline

// Observe hands the registry one event of a session: an orchestrator's or a
// child's of a project it holds. Anything else is not the registry's, and is
// let go. Called in order, per session, from one goroutine each.
func (r *Registry) Observe(sessionID string, ev protocol.Event) {
	r.mu.Lock()
	var held *Orchestrator
	isChild, isReviewer := false, false
	for _, o := range r.projects {
		if o.orchestratorID() == sessionID {
			held = o
			break
		}
		if o.child(sessionID).ID != "" {
			held, isChild = o, true
			break
		}
		if o.reviewer(sessionID) {
			held, isReviewer = o, true
			break
		}
	}
	r.mu.Unlock()
	if held == nil {
		return
	}
	if !isChild {
		if ev.Type != protocol.EventTurnCompleted {
			return
		}
		var tc protocol.TurnCompleted
		if json.Unmarshal(ev.Payload, &tc) == nil {
			held.spend(tc.Usage)
		}
		if isReviewer {
			held.reviewerEnded(sessionID)
			return
		}
		// The orchestrator's turn ended: what reached its inbox meanwhile
		// starts the next one, never this one's middle (ADR-03).
		held.scheduleReports()
		return
	}
	held.observeChild(sessionID, ev)
}

// observeChild reads one event of child id.
func (o *Orchestrator) observeChild(id string, ev protocol.Event) {
	switch ev.Type {
	case protocol.EventTurnStarted:
		o.said(id, "", true)
		if c := o.child(id); c.State == stateFailed || (c.State == stateWorking && c.Reason != "") {
			o.move(id, stateWorking, "", "a turn started in the child")
		}
	case protocol.EventMessageDelta:
		var d protocol.MessageDelta
		if json.Unmarshal(ev.Payload, &d) == nil {
			o.said(id, d.Text, false)
		}
	case protocol.EventToolRequested:
		o.said(id, "", true) // what came before a tool call is not the conclusion
	case protocol.EventApprovalRequired:
		if c := o.child(id); c.State == stateWorking {
			o.commitWaiting(id, "approval")
		}
	case protocol.EventApprovalResolved:
		if c := o.child(id); c.State == stateAwaiting {
			o.move(id, stateWorking, "", "the person answered the child's approval")
		}
	case protocol.EventTurnCompleted:
		var tc protocol.TurnCompleted
		if err := json.Unmarshal(ev.Payload, &tc); err != nil {
			o.sayError(protocol.CodeInternal, fmt.Sprintf("the end of a turn of child %s does not read: %v", id, err))
			return
		}
		o.turnEnded(id, tc)
	}
}

// said keeps the text of the child's turn since its last tool call.
func (o *Orchestrator) said(id, text string, reset bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.conclusions == nil {
		o.conclusions = map[string]*strings.Builder{}
	}
	b := o.conclusions[id]
	if b == nil || reset {
		b = &strings.Builder{}
		o.conclusions[id] = b
	}
	if b.Len() < o.reg.conclusionMax*2 {
		b.WriteString(text)
	}
}

func (o *Orchestrator) conclusion(id string) (string, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	var s string
	if b := o.conclusions[id]; b != nil {
		s = strings.TrimSpace(b.String())
	}
	delete(o.conclusions, id)
	max := o.reg.conclusionMax
	if len(s) <= max {
		return s, false
	}
	cut := max
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut], true
}

// commitWaiting moves the child to awaiting_person, saying on what it waits.
// Its question has no deadline, and only the person answers it (ADR-02).
func (o *Orchestrator) commitWaiting(id, on string) {
	data, _ := json.Marshal(map[string]string{"from": stateWorking, "to": stateAwaiting, "waiting": on})
	if err := o.commit(Decision{Kind: "child.state", Child: id, Data: data,
		Why: "the child asked the person for an " + on + ", which waits for the person without a deadline"},
		edit(id, func(c *protocol.Child) { c.State, c.Waiting = stateAwaiting, on })); err != nil {
		o.sayError(protocol.CodeInternal, fmt.Sprintf("child %s waits for the person and the log did not take it: %v", id, err))
		return
	}
	o.emit(protocol.EventChildState, protocol.ChildChanged{Child: o.child(id)})
}

// turnEnded is the end of a child's turn. The owns check comes first: a
// change outside owns is committed by nobody and goes back. Inside, the
// harness commits by name, and child.completed says the turn with the child's
// claim beside it. A turn that did not end done fails the child; one that did
// enters the pipeline (.p §7).
func (o *Orchestrator) turnEnded(id string, tc protocol.TurnCompleted) {
	c := o.child(id)
	if c.ID == "" || terminal(c.State) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), commitDeadline)
	defer cancel()
	said, cut := o.conclusion(id)
	changed, outside, cerr := o.checkOwns(ctx, c)
	var head string
	var paths, unwritten []string
	if cerr == nil && len(outside) == 0 {
		head, paths, cerr = commitTurn(ctx, c.Worktree, c.Title)
	} else if cerr == nil {
		unwritten = changed
	}
	u := protocol.Usage{}
	if tc.Usage != nil {
		u = *tc.Usage
	}
	data := map[string]any{"turn_id": tc.TurnID, "reason": tc.Reason, "head": head, "wrote": paths}
	why := fmt.Sprintf("the child's turn ended %s; the harness committed %d path(s) by name", tc.Reason, len(paths))
	switch {
	case cerr != nil:
		data["commit_error"] = cerr.Error()
		why = fmt.Sprintf("the child's turn ended %s, and the harness could not commit it", tc.Reason)
	case len(outside) > 0:
		data["outside_owns"] = outside
		why = fmt.Sprintf("the child's turn ended %s with changes outside owns, so the harness committed nothing", tc.Reason)
	}
	raw, _ := json.Marshal(data)
	err := o.commit(Decision{Kind: "child.completed", Child: id, Session: id, Why: why, Claim: said, Data: raw},
		func(st *State) {
			st.Spent += tokens(tc.Usage)
			setBrief(st, id, func(b *Brief) { b.Claim = said })
			edit(id, func(ch *protocol.Child) {
				ch.Usage = addUsage(ch.Usage, u)
				if head != "" {
					ch.Head = head
				}
				ch.Reason = tc.Reason
			})(st)
		})
	if err != nil {
		o.sayError(protocol.CodeInternal, fmt.Sprintf("the end of a turn of child %s did not reach the log: %v", id, err))
		return
	}
	o.emit(protocol.EventChildCompleted, protocol.ChildCompleted{ChildID: id, TurnID: tc.TurnID, Reason: tc.Reason,
		Conclusion: said, Truncated: cut, Wrote: paths, Unwritten: unwritten, Usage: u})
	switch {
	case cerr != nil:
		o.fail(id, fmt.Errorf("the harness could not commit the turn in %s: %w", c.Worktree, cerr))
		o.reportChild(id, "the harness could not commit its turn: "+cerr.Error())
	case tc.Reason != protocol.StopDone:
		o.move(id, stateFailed, tc.Reason, "the child's turn ended "+tc.Reason+", which is not done")
		o.reportChild(id, "its turn ended "+tc.Reason+", which is not done, so it failed. It said: "+quote(said))
	case len(outside) > 0:
		o.move(id, stateInReview, tc.Reason, "the child's turn ended done; the owns check comes first")
		o.giveBack(id, protocol.Return{Why: protocol.CodeOutsideOwns, Paths: outside}, true, ownsText(outside, c.Owns))
	default:
		o.afterDone(id)
	}
}

// quote is what a child said, for a report: never empty.
func quote(said string) string {
	if strings.TrimSpace(said) == "" {
		return "(nothing)"
	}
	return said
}

func addUsage(a, b protocol.Usage) protocol.Usage {
	a.InputTokens += b.InputTokens
	a.OutputTokens += b.OutputTokens
	a.CacheReadTokens += b.CacheReadTokens
	a.CacheWriteTokens += b.CacheWriteTokens
	return a
}
