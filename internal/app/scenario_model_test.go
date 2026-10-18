package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"testing"
)

// The scripted model of the orchestrator's scenarios (orchestration .i step
// 12): one HTTP server, in the real streaming format, answering every session
// a project opens — the orchestrator, each child, each reviewer — from a
// script of its own. Nothing about what a scenario proves is read from what
// the model says: the scripts only drive the product, and the assertions read
// the decision log, the wire and git.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.i.spec.md, step 12.

// step is one answer of a script: the frames of one model response, after
// before has run, or a response that waits on hold first.
type step struct {
	frames []string
	before func()
	hold   chan struct{}
}

func say(text string) step { return step{frames: []string{frameText(text)}} }

// use is one tool call; args is encoded as the tool's JSON input.
func use(name string, args any) step {
	b, err := json.Marshal(args)
	if err != nil {
		panic(err)
	}
	return step{frames: []string{frameToolCall("call-"+name, name, string(b))}}
}

// then runs f before the step answers: what the world does while the model
// is thinking — a commit on main, a push.
func (s step) then(f func()) step { s.before = f; return s }

// heldUntil makes the step wait for release, or for its session to go.
func (s step) heldUntil(release chan struct{}) step { s.hold = release; return s }

// marker names whose script a request is: the task of a child, and the
// review of it, carry [[key]].
var marker = regexp.MustCompile(`\[\[([a-z0-9-]+)\]\]`)

// role is whose request this is, told apart by the tools it was offered:
// only an orchestrator has delegate, only a reviewer has verdict.
func role(body string) string {
	switch {
	case strings.Contains(body, `"name":"delegate"`):
		return "orchestrator"
	case strings.Contains(body, `"name":"verdict"`):
		return "reviewer"
	}
	return "child"
}

type scenarioModel struct {
	t *testing.T

	mu         sync.Mutex
	orq        []step
	scripts    map[string][]step // "child/key", "reviewer/key"
	seen       map[string]int
	bodies     map[string][]string
	unscripted []string
	gone       chan struct{}
	stopped    bool
}

func newScenarioModel(t *testing.T) *scenarioModel {
	return &scenarioModel{t: t, scripts: map[string][]step{}, seen: map[string]int{}, bodies: map[string][]string{}, gone: make(chan struct{})}
}

// orchestrator scripts the orchestrator; once it runs out, every turn of it —
// the reports — is answered with a plain acknowledgement.
func (m *scenarioModel) orchestrator(steps ...step) { m.orq = steps }

// child scripts the child whose task carries [[key]], every turn of it.
func (m *scenarioModel) child(key string, steps ...step) { m.scripts["child/"+key] = steps }

// reviewer scripts the reviews of that child, one after the other: each
// verdict, and the turn's last answer after it.
func (m *scenarioModel) reviewer(key string, verdicts ...step) {
	var steps []step
	for _, v := range verdicts {
		steps = append(steps, v, say("Review given."))
	}
	m.scripts["reviewer/"+key] = steps
}

// requests is how many answers a script gave.
func (m *scenarioModel) requests(script string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.seen[script]
}

// sentTo is every request body a script was sent, one after the other: what
// the product showed the model — a tool's result, a hand-back's text.
func (m *scenarioModel) sentTo(script string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return strings.Join(m.bodies[script], "\n")
}

// stop releases every request still waiting; called before the server closes.
func (m *scenarioModel) stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.stopped {
		m.stopped = true
		close(m.gone)
	}
}

// check fails the scenario for every request no script answered: a child or a
// reviewer asked for more than the scenario wrote is a scenario that went
// somewhere it did not mean to.
func (m *scenarioModel) check() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.unscripted {
		m.t.Errorf("the model was asked past its script: %s", u)
	}
}

func (m *scenarioModel) serve(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	body := string(raw)
	who := role(body)
	name := "orchestrator"
	if who != "orchestrator" {
		key := "?"
		if k := marker.FindStringSubmatch(body); k != nil {
			key = k[1]
		}
		name = who + "/" + key
	}

	m.mu.Lock()
	n := m.seen[name]
	m.bodies[name] = append(m.bodies[name], body)
	m.seen[name]++
	var s step
	var scripted bool
	if name == "orchestrator" {
		s, scripted = say("Noted; the pipeline goes on."), true
		if n < len(m.orq) {
			s = m.orq[n]
		}
	} else if script := m.scripts[name]; n < len(script) {
		s, scripted = script[n], true
	} else {
		m.unscripted = append(m.unscripted, fmt.Sprintf("%s, answer %d", name, n+1))
	}
	m.mu.Unlock()

	if !scripted {
		// Nothing goes on from here: the request waits until the session
		// or the scenario goes, and check says it.
		select {
		case <-r.Context().Done():
		case <-m.gone:
		}
		return
	}
	if s.hold != nil {
		select {
		case <-s.hold:
		case <-r.Context().Done():
			return
		case <-m.gone:
			return
		}
	}
	if s.before != nil {
		s.before()
	}
	w.Header().Set("Content-Type", "text/event-stream")
	for _, f := range s.frames {
		_, _ = w.Write([]byte("data: " + f + "\n\n"))
	}
	_, _ = w.Write([]byte("data: [DONE]\n\n"))
}
