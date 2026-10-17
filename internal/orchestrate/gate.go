package orchestrate

import (
	"context"
	"time"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/protocol"
)

// Gate is the project's declared gate. Not built yet.
type Gate struct {
	Command string
	Timeout time.Duration
	Logs    string
}

type gateOutput struct{}

var runGate = func(context.Context, Gate, string, *gateOutput) (int, bool, error) { return 0, false, nil }

// GateFrom reads the gate from configuration. Not built yet.
func GateFrom(config.Resolved) (Gate, error) { return Gate{}, nil }

// Measure runs the gate on commit. Not built yet.
func (g Gate) Measure(context.Context, string, string) (protocol.GateResult, error) {
	return protocol.GateResult{}, nil
}

// RecordGate records a measurement. Not built yet.
func RecordGate(*DecisionLog, string, string, protocol.GateResult) (uint64, error) { return 0, nil }

// CheckFresh says whether a measurement is of what is about to merge. Not built yet.
func CheckFresh(protocol.GateResult, string, string) error { return nil }
