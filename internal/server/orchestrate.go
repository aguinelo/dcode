package server

import (
	"encoding/json"
	"net/http"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The project routes: one orchestrator per project, its children and the
// questions that wait for the person (orchestration .p §5.1).

// Projects is what the daemon conducts. Injected because opening a project
// builds a session, which is the app's business. An implementation that
// cannot conduct — the orchestrator off, or sessions not recorded — answers
// orchestrator_unavailable saying which, and lists nothing.
type Projects interface {
	// Open opens the project of a workspace or finds the one it already is;
	// created says which.
	Open(workspace string) (p protocol.Project, created bool, err error)
	List() []protocol.Project
	Get(id string) (protocol.Project, error)
	// Children are an orchestrator's children, the closed ones inside their
	// retention among them; none for any other session.
	Children(sessionID string) []protocol.Child
	// CloseChild is the person's close of a child: held is false when the
	// session is no child of a project here.
	CloseChild(sessionID string) (held bool, err error)
	// Questions are an orchestrator's questions, answered or not; held is
	// false for any other session.
	Questions(sessionID string) (questions []protocol.Question, held bool)
	// Answer is the person's answer to one; the first answer wins.
	Answer(sessionID, questionID string, req protocol.AnswerQuestionRequest) (protocol.Question, error)
}

// listQuestions answers GET /sessions/{id}/questions. A session that is no
// orchestrator has none: an empty list, as /children answers.
func (s *Server) listQuestions(w http.ResponseWriter, r *http.Request) {
	sess, err := s.cfg.Manager.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	out := protocol.ListQuestionsResponse{Questions: []protocol.Question{}}
	if s.cfg.Projects != nil {
		if qs, held := s.cfg.Projects.Questions(sess.ID); held && qs != nil {
			out.Questions = qs
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// answerQuestion is the person's POST of an answer. Only the person answers:
// no tool of the orchestrator's reaches this.
func (s *Server) answerQuestion(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Projects == nil {
		writeErr(w, noProjects())
		return
	}
	var req protocol.AnswerQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, protocol.Errorf(protocol.CodeInvalidInput, "malformed request: %v", err))
		return
	}
	q, err := s.cfg.Projects.Answer(r.PathValue("id"), r.PathValue("qid"), req)
	if err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	writeJSON(w, http.StatusOK, q)
}

// listChildren answers GET /sessions/{id}/children. A session with no
// children answers an empty list; one that does not exist, session_not_found.
func (s *Server) listChildren(w http.ResponseWriter, r *http.Request) {
	sess, err := s.cfg.Manager.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	out := protocol.ListChildrenResponse{Children: []protocol.Child{}}
	if s.cfg.Projects != nil {
		if kids := s.cfg.Projects.Children(sess.ID); kids != nil {
			out.Children = kids
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// closeChild is DELETE of a child: the person's close. handled is false when
// the session is no child, and the ordinary delete follows.
func (s *Server) closeChild(w http.ResponseWriter, id string) (handled bool) {
	if s.cfg.Projects == nil {
		return false
	}
	held, err := s.cfg.Projects.CloseChild(id)
	if !held {
		return false
	}
	if err != nil {
		writeErr(w, wrapErr(err))
		return true
	}
	w.WriteHeader(http.StatusNoContent)
	return true
}

func (s *Server) openProject(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Projects == nil {
		writeErr(w, noProjects())
		return
	}
	var req protocol.OpenProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, protocol.Errorf(protocol.CodeInvalidInput, "malformed request: %v", err))
		return
	}
	p, created, err := s.cfg.Projects.Open(req.Workspace)
	if err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, p)
}

func (s *Server) listProjects(w http.ResponseWriter, _ *http.Request) {
	out := protocol.ListProjectsResponse{Projects: []protocol.Project{}}
	if s.cfg.Projects != nil {
		if ps := s.cfg.Projects.List(); ps != nil {
			out.Projects = ps
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) getProject(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Projects == nil {
		writeErr(w, noProjects())
		return
	}
	p, err := s.cfg.Projects.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// projectOrchestrator answers the session that conducts a project. One the
// project names and this daemon does not hold is said, not answered with
// whatever session happens to be open.
func (s *Server) projectOrchestrator(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Projects == nil {
		writeErr(w, noProjects())
		return
	}
	p, err := s.cfg.Projects.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, wrapErr(err))
		return
	}
	sess, err := s.cfg.Manager.Get(p.OrchestratorID)
	if err != nil {
		writeErr(w, protocol.Errorf(protocol.CodeInternal,
			"the orchestrator %s of project %s is not open in this daemon", p.OrchestratorID, p.ID))
		return
	}
	writeJSON(w, http.StatusOK, sess.Describe())
}

func noProjects() *protocol.Error {
	return protocol.Errorf(protocol.CodeOrchestratorUnavailable, "this daemon conducts no projects")
}
