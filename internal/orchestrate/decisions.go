package orchestrate

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

const maxLine = 64 << 10

var (
	errNoReason    = errors.New("not implemented")
	errUnknownKind = errors.New("not implemented")
)

var kinds = map[string]bool{}

type Decision struct {
	V        int             `json:"v"`
	Seq      uint64          `json:"seq"`
	At       time.Time       `json:"at"`
	Project  string          `json:"project"`
	Kind     string          `json:"kind"`
	Child    string          `json:"child,omitempty"`
	Session  string          `json:"session,omitempty"`
	Why      string          `json:"why"`
	Claim    string          `json:"claim,omitempty"`
	Measured json.RawMessage `json:"measured,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
}

type DecisionLog struct {
	mu  sync.Mutex
	f   *os.File
	now func() time.Time
}

func OpenDecisionLog(dir string, p protocol.Project) (*DecisionLog, error) {
	return &DecisionLog{}, nil
}

func (l *DecisionLog) Record(d Decision) (uint64, error) { return 0, nil }

func (l *DecisionLog) Torn() []byte { return nil }

func (l *DecisionLog) Close() error { return nil }
