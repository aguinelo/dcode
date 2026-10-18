package orchestrate

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/protocol"
)

var errNotBuilt = errors.New("orchestrate: the merge is not built yet")

const (
	ForgeAuto   = "auto"
	ForgeGitHub = "github"
	ForgeLocal  = "local"
)

var defaultSensitive = []string{}

type MergeConfig struct {
	Forge              string
	CITimeout          time.Duration
	Sensitive          []string
	SensitiveDiffLines int
}

func MergeFrom(config.Resolved) (MergeConfig, error) { return MergeConfig{}, errNotBuilt }

type Forge interface {
	Name() string
	Tips(ctx context.Context, branch string) (main, head string, err error)
	OpenPR(ctx context.Context, branch, title, body string) (protocol.ChildPR, error)
	Merge(ctx context.Context, c protocol.Child, measured string) (string, error)
	MainHealth(ctx context.Context, commit string) (protocol.MainHealth, error)
}

type ForgeSetup struct {
	Declared, Root, Main string
	Integration          *Integration
	Gate                 Gate
}

func NewForge(context.Context, ForgeSetup) (Forge, error) { return nil, errNotBuilt }

type Integration struct {
	repo, place string
	mu          sync.Mutex
}

func OpenIntegration(context.Context, string, string) (*Integration, error) { return nil, errNotBuilt }

type Conflict struct{ Paths []string }

func (c *Conflict) Error() string            { return "conflict" }
func (c *Conflict) Return() protocol.Return { return protocol.Return{} }

func (i *Integration) Update(context.Context, *DecisionLog, string, string, string, string) (string, error) {
	return "", errNotBuilt
}

type MergeOptions struct {
	Log    *DecisionLog
	Forge  Forge
	Config MergeConfig
	Poll   time.Duration
	Emit   func(protocol.EventType, any)
	Now    func() time.Time
}

type Merges struct{}

func NewMerges(MergeOptions) (*Merges, error) { return nil, errNotBuilt }

type Proposal struct {
	Child       protocol.Child
	Title, Body string
	Paths       []string
	Lines       int
}

type MergeOutcome struct {
	Completed *protocol.MergeCompleted
	Refused   *protocol.MergeRefused
	Main      protocol.MainHealth
}

func (m *Merges) Propose(context.Context, Proposal) (protocol.Child, protocol.Question, error) {
	return protocol.Child{}, protocol.Question{}, errNotBuilt
}
func (m *Merges) Answer(string, string, string) (protocol.Question, error) {
	return protocol.Question{}, errNotBuilt
}
func (m *Merges) Questions() []protocol.Question { return nil }
func (m *Merges) Merge(context.Context, protocol.Child, bool) (MergeOutcome, error) {
	return MergeOutcome{}, errNotBuilt
}
func (m *Merges) queued(string) bool           { return false }
func (m *Merges) health() protocol.MainHealth { return protocol.MainHealth{} }
