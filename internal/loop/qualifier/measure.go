package qualifier

import (
	"context"
	"errors"
	"time"

	"github.com/aguinelo/dcode/internal/loop"
)

// MaxOutput is how much of a criterion's output reaches the operator.
//
// It is a constant rather than a setting for now: the configuration key the
// spec declares arrives with the caller that could pass it, and a key nothing
// reads is a promise of a control that is not there.
const MaxOutput = 2000

// ErrEmptyProposal is a proposal with no criteria at all.
//
// An error, never an empty DoneSet. The absence of a definition is not a
// permissive definition — an empty DoneSet means "nothing to verify", which
// the agent loop reports as done, and that is how a proposal empty of content
// would become a green report.
var ErrEmptyProposal = errors.New("qualifier: the proposal declares no criteria")

// Class is what the run before any work says the criterion IS, whatever the
// proposer said it would be.
type Class string

const (
	// ClassAcceptance failed: it can testify that the work happened.
	//
	// This is the rule the whole family is named for. A criterion that already
	// passed cannot testify that the work met it — it would have passed with
	// nothing done, so the green at the end is coincidence rather than
	// evidence. The red-to-green transition is the proof.
	ClassAcceptance Class = "acceptance"
	// ClassRegression passed: it can testify that nothing else broke.
	//
	// Legitimate for the opposite reason. A green suite before any work is
	// exactly what is wanted from a regression guard: its job is to stay green.
	ClassRegression Class = "regression"
	// ClassBroken failed because there was nothing to run.
	//
	// It testifies to nothing and would stay red forever. Without this class it
	// disguises itself as acceptance while measuring the absence of a tool
	// rather than the absence of work.
	ClassBroken Class = "broken"
)

// brokenExits are the shell's answers for "there was nothing to run": 127 is
// command not found, 126 is found and not executable.
var brokenExits = map[int]bool{126: true, 127: true}

// Measured is one criterion after the run that happens before any work.
type Measured struct {
	Proposed
	Class Class
	// Exit is what the command actually exited with.
	Exit int
	// Output is capped at MaxOutput, and says so when it was cut. It is the
	// only thing that separates a criterion red because the work is missing
	// from one red because the world is.
	Output string
	// Truncated marks output that did not fit.
	Truncated bool
	// Mismatch is Expects disagreeing with Class. Not an error and not a
	// rejection: it is where the operator's eye should land.
	Mismatch bool
}

// Conditions are what the whole measured set says about itself.
type Conditions struct {
	// NoAcceptance is a set where nothing is red.
	//
	// A warning the operator signs, never a refusal. Such a set will report
	// done without anything having to change, and almost always that is a
	// defect — but not always: a genuine refactor has nothing new to prove and
	// everything to preserve. The harness cannot tell those apart, so it names
	// the condition and does not decide. Deciding here would be the harness
	// choosing what counts as the measurement.
	NoAcceptance bool
}

// Measure runs every proposed criterion once against the workspace as it
// stands, before any work.
//
// It never writes and it never retries. The runner is injected and is the same
// one the loop uses for a criterion, so what runs here goes through the sandbox
// exactly as it will later.
func Measure(ctx context.Context, p Proposal, run loop.CriterionRunner, timeout time.Duration) ([]Measured, Conditions, error) {
	if len(p.Criteria) == 0 {
		return nil, Conditions{}, ErrEmptyProposal
	}
	if run == nil {
		return nil, Conditions{}, errors.New("qualifier: no runner; a proposal nobody can run is a proposal nobody can classify")
	}

	out := make([]Measured, 0, len(p.Criteria))
	acceptance := 0
	for _, c := range p.Criteria {
		m := measureOne(ctx, c, run, timeout)
		if m.Class == ClassAcceptance {
			acceptance++
		}
		out = append(out, m)
	}
	return out, Conditions{NoAcceptance: acceptance == 0}, nil
}

func measureOne(ctx context.Context, c Proposed, run loop.CriterionRunner, timeout time.Duration) Measured {
	m := Measured{Proposed: c}

	runCtx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	exit, output, err := run(runCtx, c.Command)
	m.Exit = exit
	m.Output, m.Truncated = cut(output)

	switch {
	case err != nil || brokenExits[exit]:
		// Could not be started, or the shell said there was nothing to start.
		m.Class = ClassBroken
	case exit == c.ExitCode:
		// Compared against ExitCode and never against zero. A criterion
		// declared with exit 1 is met by exiting 1, and comparing to zero would
		// classify an already-green criterion as acceptance — precisely the
		// error this package exists to avoid.
		m.Class = ClassRegression
	default:
		m.Class = ClassAcceptance
	}

	// Broken is its own alarm and not a disagreement: it holds whatever the
	// proposer expected.
	switch m.Class {
	case ClassAcceptance:
		m.Mismatch = c.Expects == ExpectPass
	case ClassRegression:
		m.Mismatch = c.Expects == ExpectFail
	}
	return m
}

func cut(s string) (string, bool) {
	if len(s) <= MaxOutput {
		return s, false
	}
	return s[:MaxOutput], true
}
