package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// Each call goes to the route of the orchestration .p §5.1, with its method,
// and reads the answer the route gives.
func TestTheOrchestrationCallsAskTheirRoutes(t *testing.T) {
	var asked []string
	c := loopDaemon(t, func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, r.Method+" "+r.URL.Path)
		v := "/" + protocol.Version
		switch r.Method + " " + r.URL.Path {
		case "POST " + v + "/projects":
			var req protocol.OpenProjectRequest
			if json.NewDecoder(r.Body).Decode(&req) != nil || req.Workspace != "/w" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"id":"p1","orchestrator_id":"o"}`)
		case "GET " + v + "/projects":
			fmt.Fprint(w, `{"projects":[{"id":"p1"}]}`)
		case "GET " + v + "/projects/p1":
			fmt.Fprint(w, `{"id":"p1"}`)
		case "GET " + v + "/projects/p1/orchestrator":
			fmt.Fprint(w, `{"id":"o","role":"orchestrator"}`)
		case "GET " + v + "/sessions/o/children":
			fmt.Fprint(w, `{"children":[{"id":"c","state":"working"}]}`)
		case "GET " + v + "/sessions/o/questions":
			fmt.Fprint(w, `{"questions":[{"id":"q1","kind":"merge"}]}`)
		case "POST " + v + "/sessions/o/questions/answers":
			var req protocol.AnswerQuestionsRequest
			if json.NewDecoder(r.Body).Decode(&req) != nil || req.CardID != "c1" || len(req.Answers) != 1 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			fmt.Fprint(w, `{"results":[{"question_id":"q1","status":"answered"}],"cards":[]}`)
		case "POST " + v + "/sessions/o/questions/q1":
			w.WriteHeader(http.StatusConflict)
			fmt.Fprint(w, `{"code":"question_already_answered","message":"answered at 12:00"}`)
		default:
			http.NotFound(w, r)
		}
	})
	ctx := context.Background()

	if p, err := c.OpenProject(ctx, "/w"); err != nil || p.ID != "p1" || p.OrchestratorID != "o" {
		t.Errorf("OpenProject: %+v %v", p, err)
	}
	if ps, err := c.ListProjects(ctx); err != nil || len(ps) != 1 {
		t.Errorf("ListProjects: %+v %v", ps, err)
	}
	if p, err := c.Project(ctx, "p1"); err != nil || p.ID != "p1" {
		t.Errorf("Project: %+v %v", p, err)
	}
	if s, err := c.Orchestrator(ctx, "p1"); err != nil || s.Role != protocol.RoleOrchestrator {
		t.Errorf("Orchestrator: %+v %v", s, err)
	}
	if cs, err := c.Children(ctx, "o"); err != nil || len(cs) != 1 || cs[0].State != "working" {
		t.Errorf("Children: %+v %v", cs, err)
	}
	if qs, err := c.Questions(ctx, "o"); err != nil || len(qs) != 1 {
		t.Errorf("Questions: %+v %v", qs, err)
	}
	_, err := c.AnswerQuestion(ctx, "o", "q1", protocol.AnswerQuestionRequest{Answer: "merge"})
	if pe, ok := protocol.AsError(err); !ok || pe.Code != protocol.CodeQuestionAlreadyAnswered {
		t.Errorf("AnswerQuestion: the refusal reads %v, want question_already_answered", err)
	}
	got, err := c.AnswerQuestions(ctx, "o", protocol.AnswerQuestionsRequest{CardID: "c1",
		Answers: []protocol.AnswerItem{{QuestionID: "q1", Answer: "merge"}}})
	if err != nil || len(got.Results) != 1 || got.Results[0].Status != protocol.CardItemAnswered {
		t.Errorf("AnswerQuestions: %+v %v", got, err)
	}
	if len(asked) != 8 {
		t.Errorf("asked %v", asked)
	}
}
