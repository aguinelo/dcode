package client

import (
	"context"
	"net/http"
	"net/url"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The orchestration routes (orchestration .p §5.1). The daemon answers them
// from the steps that build the orchestrator; until then each is a refusal the
// caller reads like any other.

// OpenProject opens the project of a workspace, or finds the one it already
// is: every worktree of one tree is the same project.
func (c *Client) OpenProject(ctx context.Context, workspace string) (protocol.Project, error) {
	var out protocol.Project
	err := c.do(ctx, http.MethodPost, "/projects", protocol.OpenProjectRequest{Workspace: workspace}, &out)
	return out, err
}

// ListProjects lists the projects the daemon conducts.
func (c *Client) ListProjects(ctx context.Context) ([]protocol.Project, error) {
	var out protocol.ListProjectsResponse
	err := c.do(ctx, http.MethodGet, "/projects", nil, &out)
	return out.Projects, err
}

// Project returns one project, or project_not_found.
func (c *Client) Project(ctx context.Context, id string) (protocol.Project, error) {
	var out protocol.Project
	err := c.do(ctx, http.MethodGet, "/projects/"+url.PathEscape(id), nil, &out)
	return out, err
}

// Orchestrator returns the session that conducts a project.
func (c *Client) Orchestrator(ctx context.Context, projectID string) (protocol.Session, error) {
	var out protocol.Session
	err := c.do(ctx, http.MethodGet, "/projects/"+url.PathEscape(projectID)+"/orchestrator", nil, &out)
	return out, err
}

// Children lists a session's children: the live ones, and the closed ones
// still inside their retention. A session with none answers an empty list.
func (c *Client) Children(ctx context.Context, sessionID string) ([]protocol.Child, error) {
	var out protocol.ListChildrenResponse
	err := c.do(ctx, http.MethodGet, "/sessions/"+url.PathEscape(sessionID)+"/children", nil, &out)
	return out.Children, err
}

// Questions lists what an orchestrator is waiting for the person to answer.
func (c *Client) Questions(ctx context.Context, sessionID string) ([]protocol.Question, error) {
	var out protocol.ListQuestionsResponse
	err := c.do(ctx, http.MethodGet, "/sessions/"+url.PathEscape(sessionID)+"/questions", nil, &out)
	return out.Questions, err
}

// AnswerQuestion answers one question. The first answer wins; a second is
// refused with question_already_answered.
func (c *Client) AnswerQuestion(ctx context.Context, sessionID, questionID string, req protocol.AnswerQuestionRequest) (protocol.Question, error) {
	var out protocol.Question
	path := "/sessions/" + url.PathEscape(sessionID) + "/questions/" + url.PathEscape(questionID)
	err := c.do(ctx, http.MethodPost, path, req, &out)
	return out, err
}

// AnswerQuestions confirms a card: every answer in one POST, each with its
// own result, the first answer to a question winning.
func (c *Client) AnswerQuestions(ctx context.Context, sessionID string, req protocol.AnswerQuestionsRequest) (protocol.AnswerQuestionsResponse, error) {
	var out protocol.AnswerQuestionsResponse
	err := c.do(ctx, http.MethodPost, "/sessions/"+url.PathEscape(sessionID)+"/questions/answers", req, &out)
	return out, err
}
