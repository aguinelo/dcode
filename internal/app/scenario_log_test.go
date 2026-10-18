package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aguinelo/dcode/internal/orchestrate"
	"github.com/aguinelo/dcode/internal/protocol"
)

// What a scenario reads to decide whether it passed: the decision log,
// through the reader `dcode decisions` uses, and the questions the person is
// asked, through the routes the person's client uses. Never what a model said.

// decisions is every line of the project's log. A line that does not read,
// or a seq that does not follow, fails the scenario: the log is the record,
// and one with a hole is not.
func (s *scenario) decisions() []orchestrate.Decision {
	s.t.Helper()
	f, err := os.Open(filepath.Join(s.orch, s.project.ID, "decisions.jsonl"))
	if err != nil {
		s.t.Fatalf("the decision log does not open: %v", err)
	}
	defer f.Close()
	r := orchestrate.NewLogReader(f)
	var out []orchestrate.Decision
	for {
		line, err := r.Next()
		if errors.Is(err, io.EOF) {
			return out
		}
		if err != nil {
			s.t.Fatalf("the decision log does not read: %v", err)
		}
		if line.Err != nil || line.Gap {
			s.t.Fatalf("line %d of the decision log: %v (gap %v after seq %d): %s", line.N, line.Err, line.Gap, line.Prev, line.Raw)
		}
		out = append(out, line.Decision)
	}
}

// logText is the log, one decision a line, for a failure to show.
func (s *scenario) logText() string {
	var b strings.Builder
	f, err := os.Open(filepath.Join(s.orch, s.project.ID, "decisions.jsonl"))
	if err != nil {
		return "(no log: " + err.Error() + ")"
	}
	defer f.Close()
	r := orchestrate.NewLogReader(f)
	for {
		line, err := r.Next()
		if err != nil {
			return b.String()
		}
		d := line.Decision
		fmt.Fprintf(&b, "  #%d %-18s %s  %s\n", d.Seq, d.Kind, tail(d.Child), d.Why)
	}
}

func tail(id string) string {
	if len(id) > 6 {
		return id[len(id)-6:]
	}
	return id
}

// of is the decisions about child id, every one when id is empty.
func of(ds []orchestrate.Decision, id string) []orchestrate.Decision {
	var out []orchestrate.Decision
	for _, d := range ds {
		if id == "" || d.Child == id {
			out = append(out, d)
		}
	}
	return out
}

// kinds is the kind of each decision about child id, in order.
func (s *scenario) kinds(id string) []string {
	var out []string
	for _, d := range of(s.decisions(), id) {
		out = append(out, d.Kind)
	}
	return out
}

// inOrder checks the log holds want, in that order, about child id — other
// decisions may come between. Every scenario ends with one.
func (s *scenario) inOrder(id string, want ...string) {
	s.t.Helper()
	got := s.kinds(id)
	i := 0
	for _, k := range got {
		if i < len(want) && k == want[i] {
			i++
		}
	}
	if i < len(want) {
		s.t.Errorf("the log about %q does not hold %v in order: it stops matching at %q\n%s",
			tail(id), want, want[i], s.logText())
	}
}

// count is how many decisions of kind are about child id.
func (s *scenario) count(id, kind string) int {
	n := 0
	for _, d := range of(s.decisions(), id) {
		if d.Kind == kind {
			n++
		}
	}
	return n
}

// all is every decision of kind about child id, in order.
func (s *scenario) all(id, kind string) []orchestrate.Decision {
	var out []orchestrate.Decision
	for _, d := range of(s.decisions(), id) {
		if d.Kind == kind {
			out = append(out, d)
		}
	}
	return out
}

// gates is every gate the harness measured on child id, in order.
func (s *scenario) gates(id string) []protocol.GateResult {
	s.t.Helper()
	var out []protocol.GateResult
	for _, d := range s.all(id, "gate.measured") {
		out = append(out, decode[protocol.GateResult](s.t, d.Measured))
	}
	return out
}

// returns is every hand-back of child id, in order.
func (s *scenario) returns(id string) []protocol.Return {
	s.t.Helper()
	var out []protocol.Return
	for _, d := range s.all(id, "child.returned") {
		out = append(out, decode[struct {
			Return protocol.Return `json:"return"`
		}](s.t, d.Data).Return)
	}
	return out
}

// questions is the orchestrator's question queue, as the person's client
// reads it.
func (s *scenario) questions() []protocol.Question {
	s.t.Helper()
	code, body := route(s.t, s.d, http.MethodGet, "/sessions/"+s.project.OrchestratorID+"/questions", nil)
	if code != http.StatusOK {
		s.t.Fatalf("GET questions = %d %s", code, body)
	}
	var out protocol.ListQuestionsResponse
	if err := json.Unmarshal(body, &out); err != nil {
		s.t.Fatal(err)
	}
	return out.Questions
}

// pending is the unanswered question of kind about child id.
func (s *scenario) pending(id, kind string) (protocol.Question, bool) {
	s.t.Helper()
	for _, q := range s.questions() {
		if q.ChildID == id && q.Kind == kind && q.Answer == "" {
			return q, true
		}
	}
	return protocol.Question{}, false
}

// waitQuestion waits for the merge question about child id.
func (s *scenario) waitQuestion(id string) protocol.Question {
	s.t.Helper()
	var q protocol.Question
	s.waitFor("no merge question was asked about "+tail(id), func() bool {
		var ok bool
		q, ok = s.pending(id, "merge")
		return ok
	})
	return q
}

// answer is the person's answer, through the route; it returns the status.
func (s *scenario) answer(q protocol.Question, answer string) (int, []byte) {
	s.t.Helper()
	return route(s.t, s.d, http.MethodPost, "/sessions/"+s.project.OrchestratorID+"/questions/"+q.ID,
		protocol.AnswerQuestionRequest{Answer: answer, Note: "the scenario's person"})
}

// merge says yes to child id's merge question and wants it taken.
func (s *scenario) merge(id string) {
	s.t.Helper()
	q := s.waitQuestion(id)
	if code, body := s.answer(q, "merge"); code != http.StatusOK {
		s.t.Fatalf("the person's merge was refused: %d %s", code, body)
	}
}
