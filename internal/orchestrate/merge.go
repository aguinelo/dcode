package orchestrate

// A merge waits for the person. In step-by-step — the only level of this
// slice — a child measured green becomes a PR and a merge question; only the
// person's yes puts it in the queue, and the queue merges one at a time, each
// checked fresh over main as it is now, with main's health read after each
// merge. A red or unknown main stops the queue until it is green.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §7.5, §7.6;
// .config.spec.md §3.

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
)

// The answers a merge question offers (.p §7.5).
const (
	answerMerge  = "merge"
	answerReturn = "return"
	answerClose  = "close"
)

// The codes of merge.refused beside the protocol's: the person's no, and the
// forge's.
const (
	refusedByPerson = "person"
	refusedByForge  = "forge"
)

// healthPollDefault is how long apart main's CI is read after a merge.
const healthPollDefault = 30 * time.Second

// MergeOptions configure a project's merges.
type MergeOptions struct {
	Log    *DecisionLog
	Forge  Forge
	Config MergeConfig
	// Poll is how long apart main's CI is read after a merge; 0 is 30s.
	Poll time.Duration
	// Emit sends an event to the orchestrator's stream, after the log has the
	// decision (.p §8). Nil sends nothing.
	Emit func(protocol.EventType, any)
	Now  func() time.Time
	// LocalSaid is whether forge.local was already said for this project.
	LocalSaid bool
}

// Merges is one project's merge questions and its serial queue.
type Merges struct {
	o MergeOptions

	mu        sync.Mutex // the questions and the queue
	questions []protocol.Question
	queue     []string
	main      protocol.MainHealth

	serial sync.Mutex // one merge at a time (.config §4)
}

// NewMerges starts a project's merges. A local forge says, the first time,
// what is lost (forge.local).
func NewMerges(o MergeOptions) (*Merges, error) {
	if o.Log == nil || o.Forge == nil {
		return nil, errors.New("orchestrate: merges need the decision log and a forge")
	}
	if o.Config.CITimeout <= 0 {
		return nil, errors.New("orchestrate: merges need a ci_timeout; without one a pending main is waited on forever")
	}
	if o.Poll <= 0 {
		o.Poll = healthPollDefault
	}
	if o.Emit == nil {
		o.Emit = func(protocol.EventType, any) {}
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.Forge.Name() == ForgeLocal && !o.LocalSaid {
		if _, err := o.Log.Record(Decision{Kind: "forge.local", Why: localLoses}); err != nil {
			return nil, err
		}
	}
	return &Merges{o: o}, nil
}

// Proposal is a child the pipeline measured green, to be proposed.
type Proposal struct {
	Child       protocol.Child
	Title, Body string
	// Paths and Lines are the diff's, which say whether the merge is sensitive.
	Paths []string
	Lines int
}

// MergeOutcome is what one merge did: completed, or refused, and main's
// health as the queue now reads it.
type MergeOutcome struct {
	Completed *protocol.MergeCompleted
	Refused   *protocol.MergeRefused
	Main      protocol.MainHealth
}

// Propose opens the child's PR — the branch pushed, never forced, and a body
// with the evidence — and asks the person whether it merges. Only a green
// measurement of the child's head is proposed. It returns the child with its
// PR, and the question.
func (m *Merges) Propose(ctx context.Context, p Proposal) (protocol.Child, protocol.Question, error) {
	c := p.Child
	if c.Gate == nil || c.Gate.Result != GateGreen || c.Gate.Commit == "" || c.Gate.Commit != c.Head {
		return c, protocol.Question{}, &protocol.Error{Code: protocol.CodeInvalidInput, Message: fmt.Sprintf(
			"%s has no green measurement of its head; without a green gate there is no PR and no merge", c.ID)}
	}
	pr, err := m.o.Forge.OpenPR(ctx, c.Branch, p.Title, PRBody(p.Body, *c.Gate, c.Review))
	if err != nil {
		return c, protocol.Question{}, err
	}
	if m.o.Forge.Name() != ForgeLocal {
		c.PR = &pr
		data, _ := json.Marshal(pr)
		if _, err := m.o.Log.Record(Decision{Kind: "pr.opened", Child: c.ID, Data: data,
			Why: fmt.Sprintf("the gate was green on %s, so the harness pushed %s and opened #%d", short(c.Head), c.Branch, pr.Number)}); err != nil {
			return c, protocol.Question{}, err
		}
	}
	q := m.question(c, p)
	data, _ := json.Marshal(q)
	if _, err := m.o.Log.Record(Decision{Kind: "question.asked", Child: c.ID, Data: data,
		Why: "step by step: every merge waits for the person's answer"}); err != nil {
		return c, protocol.Question{}, err
	}
	m.mu.Lock()
	m.questions = append(m.questions, q)
	m.mu.Unlock()
	m.o.Emit(protocol.EventQuestionAsked, q)
	return c, q, nil
}

func (m *Merges) question(c protocol.Child, p Proposal) protocol.Question {
	text := fmt.Sprintf("Merge %q, measured green at %s?", p.Title, short(c.Head))
	var sensitive []string
	over := m.o.Config.SensitiveDiffLines > 0 && p.Lines > m.o.Config.SensitiveDiffLines
	for _, path := range p.Paths {
		if over || m.sensitive(path) {
			sensitive = append(sensitive, path)
		}
	}
	if over {
		text += fmt.Sprintf(" The diff is %d lines, over sensitive_diff_lines (%d).", p.Lines, m.o.Config.SensitiveDiffLines)
	}
	return protocol.Question{ID: "q-" + randomHex(), Kind: "merge", Text: text, Recommended: answerMerge,
		Why: "the reviewer approved and the harness measured the gate green on this commit", ChildID: c.ID,
		Options:   []protocol.QuestionOption{{ID: answerMerge, Label: "Merge"}, {ID: answerReturn, Label: "Return to the child"}, {ID: answerClose, Label: "Close the child"}},
		Sensitive: sensitive, Asked: m.o.Now().UTC()}
}

func (m *Merges) sensitive(path string) bool {
	for _, g := range m.o.Config.Sensitive {
		if policy.Glob(g, path) {
			return true
		}
	}
	return false
}

// Answer is the person's answer; the first one wins. merge queues the child;
// return and close are the person's refusal, said as merge.refused with the
// note. Only the person answers: the orchestrator has no tool for this.
func (m *Merges) Answer(id, answer, note string) (protocol.Question, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	i := -1
	for n := range m.questions {
		if m.questions[n].ID == id {
			i = n
		}
	}
	if i < 0 {
		return protocol.Question{}, &protocol.Error{Code: protocol.CodeQuestionNotFound, Message: fmt.Sprintf("no question %q", id)}
	}
	q := m.questions[i]
	if q.Answered != nil {
		return q, &protocol.Error{Code: protocol.CodeQuestionAlreadyAnswered,
			Message: fmt.Sprintf("%s was answered %q already; the first answer wins", id, q.Answer)}
	}
	if !offers(q, answer) {
		return q, &protocol.Error{Code: protocol.CodeInvalidInput, Message: fmt.Sprintf("%s offers no answer %q", id, answer)}
	}
	at := m.o.Now().UTC()
	q.Answer, q.Note, q.Answered = answer, note, &at
	data, _ := json.Marshal(q)
	if _, err := m.o.Log.Record(Decision{Kind: "question.answered", Child: q.ChildID, Data: data,
		Why: fmt.Sprintf("the person answered %q", answer)}); err != nil {
		return protocol.Question{}, err
	}
	m.questions[i] = q
	m.o.Emit(protocol.EventQuestionAnswered, q)
	if answer != answerMerge {
		reason := fmt.Sprintf("the person answered %s", answer)
		if note != "" {
			reason += ": " + note
		}
		r := &protocol.MergeRefused{ChildID: q.ChildID, Code: refusedByPerson, Reason: reason}
		return q, m.refuse(r)
	}
	if _, err := m.o.Log.Record(Decision{Kind: "merge.queued", Child: q.ChildID,
		Why: fmt.Sprintf("the person answered merge; %d ahead in the queue", len(m.queue))}); err != nil {
		return q, err
	}
	m.queue = append(m.queue, q.ChildID)
	return q, nil
}

func offers(q protocol.Question, answer string) bool {
	for _, o := range q.Options {
		if o.ID == answer {
			return true
		}
	}
	return false
}

// Questions is every question, answered or not, in the order asked.
func (m *Merges) Questions() []protocol.Question {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]protocol.Question(nil), m.questions...)
}

// Merge merges one child the person said yes to. In order: main as the forge
// has it now; a red, pending or unknown main is main_red and the child waits
// in the queue, unless it fixes main; a measurement of another head or over
// another main is gate_stale, and the child leaves the queue to be measured
// again; then the forge's merge of the measured commit, and main's health
// read until CI ends or ci_timeout passes.
//
// A refusal is an outcome, not an error: it is in the log and on the wire.
// An error is a merge that could not be tried or told.
func (m *Merges) Merge(ctx context.Context, c protocol.Child, fixesMain bool) (MergeOutcome, error) {
	m.serial.Lock()
	defer m.serial.Unlock()
	if !m.queued(c.ID) {
		return MergeOutcome{}, &protocol.Error{Code: protocol.CodeInvalidInput,
			Message: fmt.Sprintf("%s is not in the merge queue: in step by step nothing merges without the person's yes", c.ID)}
	}
	if c.Gate == nil || c.Gate.Result != GateGreen {
		return m.refused(c.ID, protocol.CodeGateUndeclared, "nothing merges without a green measurement", true)
	}
	tip, head, err := m.o.Forge.Tips(ctx, c.Branch)
	if err != nil {
		return MergeOutcome{}, err
	}
	main := m.health()
	if main.Commit != tip {
		if main, err = m.readMain(ctx, tip); err != nil {
			return MergeOutcome{Main: main}, err
		}
	}
	if main.Health != healthGreen && !fixesMain {
		return m.refused(c.ID, protocol.CodeMainRed, fmt.Sprintf("main at %s is %s; the queue waits for it to be green, "+
			"and a fix comes before any other merge", short(tip), main.Health), false)
	}
	if err := CheckFresh(*c.Gate, head, tip); err != nil {
		return m.refused(c.ID, protocol.CodeGateStale, err.(*protocol.Error).Message, true)
	}
	merged, err := m.o.Forge.Merge(ctx, c, c.Gate.Commit)
	var left *branchLeft
	var no *forgeRefusal
	switch {
	case errors.As(err, &no):
		return m.refused(c.ID, refusedByForge, no.reason, true)
	case err != nil && !errors.As(err, &left):
		return MergeOutcome{}, err
	}
	done := &protocol.MergeCompleted{ChildID: c.ID, Method: "squash", Merged: merged, Measured: c.Gate.Commit}
	if c.PR != nil {
		done.PR = c.PR.Number
	}
	why := fmt.Sprintf("the person said merge, main was %s, and the gate measured %s over main at %s: %s squashed to %s",
		main.Health, short(c.Gate.Commit), short(tip), m.o.Forge.Name(), short(merged))
	if left != nil {
		why += "; " + left.Error()
	}
	data, _ := json.Marshal(done)
	if _, err := m.o.Log.Record(Decision{Kind: "merge.completed", Child: c.ID, Why: why, Data: data}); err != nil {
		return MergeOutcome{}, err
	}
	m.dequeue(c.ID)
	m.o.Emit(protocol.EventMergeCompleted, *done)
	h, err := m.waitMain(ctx, merged)
	return MergeOutcome{Completed: done, Main: h}, err
}

// refused records and sends a refusal; leave takes the child out of the queue.
func (m *Merges) refused(child, code, reason string, leave bool) (MergeOutcome, error) {
	r := &protocol.MergeRefused{ChildID: child, Code: code, Reason: reason}
	if err := m.refuse(r); err != nil {
		return MergeOutcome{}, err
	}
	if leave {
		m.dequeue(child)
	}
	return MergeOutcome{Refused: r, Main: m.health()}, nil
}

func (m *Merges) refuse(r *protocol.MergeRefused) error {
	data, _ := json.Marshal(r)
	if _, err := m.o.Log.Record(Decision{Kind: "merge.refused", Child: r.ChildID, Why: r.Reason, Data: data}); err != nil {
		return err
	}
	m.o.Emit(protocol.EventMergeRefused, *r)
	return nil
}

// waitMain reads main's health at commit until it is green or red, or
// ci_timeout passes; pending then is unknown, which stops the queue too.
func (m *Merges) waitMain(ctx context.Context, commit string) (protocol.MainHealth, error) {
	deadline := m.o.Now().Add(m.o.Config.CITimeout)
	for {
		h, err := m.o.Forge.MainHealth(ctx, commit)
		if err == nil && h.Health == healthPending && !m.o.Now().Before(deadline) {
			h.Health = healthUnknown
		}
		if err != nil || h.Health != healthPending {
			return m.setMain(h, err, fmt.Sprintf("after the merge of %s", short(commit)))
		}
		m.o.Emit(protocol.EventMainHealth, h)
		select {
		case <-ctx.Done():
			h.Health = healthUnknown
			return m.setMain(h, ctx.Err(), "the wait for main's CI was stopped")
		case <-time.After(m.o.Poll):
		}
	}
}

// readMain is one reading of main at a tip the queue has not read yet.
func (m *Merges) readMain(ctx context.Context, tip string) (protocol.MainHealth, error) {
	h, err := m.o.Forge.MainHealth(ctx, tip)
	return m.setMain(h, err, "before a merge, at a main the queue had not read")
}

// setMain records main.health and keeps it. A reading that failed is
// unknown, said with why, and returned with the error.
func (m *Merges) setMain(h protocol.MainHealth, readErr error, when string) (protocol.MainHealth, error) {
	why := fmt.Sprintf("%s, %s says main at %s is %s", when, m.o.Forge.Name(), short(h.Commit), h.Health)
	if readErr != nil {
		h.Health = healthUnknown
		why = fmt.Sprintf("%s, main at %s could not be read and is unknown: %v", when, short(h.Commit), readErr)
	}
	if h.Checked.IsZero() {
		h.Checked = m.o.Now().UTC()
	}
	measured, _ := json.Marshal(h)
	if _, err := m.o.Log.Record(Decision{Kind: "main.health", Why: why, Measured: measured}); err != nil {
		return h, errors.Join(readErr, err)
	}
	m.mu.Lock()
	m.main = h
	m.mu.Unlock()
	m.o.Emit(protocol.EventMainHealth, h)
	return h, readErr
}

func (m *Merges) queued(child string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range m.queue {
		if id == child {
			return true
		}
	}
	return false
}

func (m *Merges) dequeue(child string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := m.queue[:0]
	for _, id := range m.queue {
		if id != child {
			out = append(out, id)
		}
	}
	m.queue = out
}

func (m *Merges) health() protocol.MainHealth {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.main
}

// PRBody is the child's own body and a fixed block of evidence: the commit
// measured, the main under it, the command, the result and the verdict. The
// gate's output never goes in: what a gate prints may hold a secret, and the
// whole of it stays in gates/ on this machine.
func PRBody(body string, g protocol.GateResult, v *protocol.Verdict) string {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(body))
	b.WriteString("\n\n---\n\n**Evidence, measured by the dcode harness**\n\n")
	fmt.Fprintf(&b, "- commit measured: `%s`\n- over main at: `%s`\n- gate: `%s` — %s (exit %d, %d ms)\n",
		g.Commit, g.Base, g.Command, g.Result, g.ExitCode, g.DurationMS)
	if v != nil {
		fmt.Fprintf(&b, "- review: %s, by session `%s`, of `%s`\n", v.Verdict, v.SessionID, v.Commit)
	} else {
		b.WriteString("- review: none recorded\n")
	}
	return b.String()
}

func randomHex() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
