package tui

import (
	"context"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/pkg/client"
)

// Orchestration is what the client reads and answers under an orchestrator.
type Orchestration interface {
	Children(ctx context.Context, sessionID string) ([]protocol.Child, error)
	Questions(ctx context.Context, sessionID string) ([]protocol.Question, error)
	AnswerQuestion(ctx context.Context, sessionID, questionID string, req protocol.AnswerQuestionRequest) (protocol.Question, error)
}

var _ Orchestration = (*client.Client)(nil)

type agentsLoadedMsg struct {
	session   string
	children  []protocol.Child
	questions []protocol.Question
	err       error
}

// AskView is the question ^Q opened.
type AskView struct {
	Question protocol.Question
	Cursor   int
}

// AgentPick is the list ^R opens under an orchestrator.
type AgentPick struct {
	Active bool
	Cursor int
}
