package qualifier

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aguinelo/dcode/internal/loop"
)

// MaxSignRounds is how many times a proposal goes back to the operator when
// their own edit changed what a criterion measures.
//
// Exhausting it REFUSES. A ceiling that accepted the last state on running out
// is the same defect as a deadline that approves, wearing another name.
//
// A constant rather than a setting for now: the configuration key the spec
// declares arrives with the caller that could pass it, and a key nothing reads
// is a promise of a control that is not there.
const MaxSignRounds = 3

// ErrRefused is the operator declining, the deadline passing, or the round
// limit running out.
//
// All three end the turn the same way and none of them starts a loop. The cost
// of refusing is the operator typing again; the cost of approving on their
// behalf is an agent working against a ruler nobody read, and a report at the
// end that says done.
var ErrRefused = errors.New("qualifier: the definition of done was not signed")

// SignedAnswer is what came back from the operator.
type SignedAnswer struct {
	Signed    bool
	Criteria  []Proposed
	Protected []string
}

// Asker puts a measured proposal in front of the operator and returns what came
// back.
//
// The transport is the server's. This is the seam that keeps the qualifier
// testable without one, and it is the same shape as the criterion runner: the
// package decides what is asked, not how it travels.
type Asker func(ctx context.Context, measured []Measured, c Conditions) (SignedAnswer, error)

// Sign runs the operator round trip and returns the frozen definition of done.
//
// Criteria the operator edited are measured AGAIN before anything freezes.
// Without that the edit escapes the rule this package exists to apply: a
// hand-written, already-green command would enter as acceptance without ever
// having been red. The fail-first rule is about what will be measured, not
// about who wrote it.
func Sign(ctx context.Context, measured []Measured, ask Asker, run loop.CriterionRunner, timeout time.Duration) (loop.DoneSet, error) {
	if ask == nil {
		return loop.DoneSet{}, fmt.Errorf("qualifier: no asker; a proposal nobody can put in front of anyone cannot be signed")
	}
	if run == nil {
		// Refused up front rather than discovered halfway. The whole point of
		// the round trip is measuring what the operator edited, and a signature
		// that cannot re-measure would freeze a criterion whose class nobody
		// knows — which is the blind spot this exists to close.
		return loop.DoneSet{}, fmt.Errorf("qualifier: no runner; an edit nobody can measure cannot be signed")
	}
	if len(measured) == 0 {
		return loop.DoneSet{}, ErrEmptyProposal
	}

	current := measured
	for round := 0; round < MaxSignRounds; round++ {
		if err := ctx.Err(); err != nil {
			// The deadline passing is a refusal, never an approval.
			return loop.DoneSet{}, fmt.Errorf("%w: %v", ErrRefused, err)
		}

		answer, err := ask(ctx, current, conditionsOf(current))
		if err != nil {
			return loop.DoneSet{}, fmt.Errorf("%w: %v", ErrRefused, err)
		}
		if !answer.Signed {
			return loop.DoneSet{}, ErrRefused
		}
		if len(answer.Criteria) == 0 {
			// Signing an empty set is signing "nothing to verify", which the
			// loop reports as done. The absence of a definition is not a
			// permissive definition, whoever produced it.
			return loop.DoneSet{}, ErrEmptyProposal
		}

		next, settled := remeasure(ctx, current, answer, run, timeout)
		if settled {
			return freeze(next, answer.Protected)
		}
		current = next
	}
	// Out of rounds. Accepting the last state here would be the ceiling
	// approving, which is the thing this refuses to do.
	return loop.DoneSet{}, fmt.Errorf("%w: %d rounds of edits and the measurement never settled", ErrRefused, MaxSignRounds)
}

// remeasure runs whatever the operator changed and says whether the set is
// settled — that is, whether every criterion in it carries a class the operator
// has already seen.
//
// A criterion the operator added is never settled on the round it appears: they
// have not seen what it does yet, and a set signed on an unseen class is
// exactly the blind spot the re-measurement exists to close.
func remeasure(ctx context.Context, seen []Measured, answer SignedAnswer, run loop.CriterionRunner, timeout time.Duration) ([]Measured, bool) {
	before := make(map[string]Measured, len(seen))
	for _, m := range seen {
		before[m.Name] = m
	}

	out := make([]Measured, 0, len(answer.Criteria))
	settled := true
	for _, c := range answer.Criteria {
		old, known := before[c.Name]
		if known && old.Command == c.Command && old.ExitCode == c.ExitCode {
			// Untouched: it keeps the measurement the operator was looking at
			// when they signed. Re-running it would be a second measurement of
			// the same thing, and the two can disagree.
			old.Proposed = c
			out = append(out, old)
			continue
		}
		m := measureOne(ctx, c, run, timeout)
		if !known || m.Class != old.Class || m.Class == ClassBroken {
			settled = false
		}
		out = append(out, m)
	}
	return out, settled
}

// freeze turns the signed set into the DoneSet the session is born with.
//
// Broken criteria do not reach it. A command that could not be run measures
// nothing about the work and would sit unmet for the whole turn, holding the
// loop open against a failure that has nothing to do with the code.
//
// And dropping them can empty the set, which is why this returns an error.
// A DoneSet with no criteria means "nothing to verify", which the loop reports
// as done — so a proposal where everything was broken would end as a green
// report on work nobody defined. That is the defect this package exists to
// prevent, arriving through the one door left open: not the proposal being
// empty, but becoming empty on the way out.
func freeze(measured []Measured, protected []string) (loop.DoneSet, error) {
	set := loop.DoneSet{Protected: protected}
	broken := 0
	for _, m := range measured {
		if m.Class == ClassBroken {
			broken++
			continue
		}
		set.Criteria = append(set.Criteria, loop.Criterion{
			Name:     m.Name,
			Command:  m.Command,
			ExitCode: m.ExitCode,
		})
	}
	if len(set.Criteria) == 0 {
		return loop.DoneSet{}, fmt.Errorf(
			"%w: all %d were broken, and a set with nothing runnable in it reports done", ErrEmptyProposal, broken)
	}
	return set, nil
}

func conditionsOf(measured []Measured) Conditions {
	for _, m := range measured {
		if m.Class == ClassAcceptance {
			return Conditions{}
		}
	}
	return Conditions{NoAcceptance: true}
}
