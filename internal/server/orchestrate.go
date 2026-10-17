package server

import (
	"encoding/json"
	"net/http"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The project routes: one orchestrator per project (orchestration .p §5.1).
// The children's and the questions' routes arrive with the steps that make
// children and questions.

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
