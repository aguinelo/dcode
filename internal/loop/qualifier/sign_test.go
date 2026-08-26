package qualifier

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/loop"
)

// asks answers each round from a script, and records what it was shown.
func asks(answers ...SignedAnswer) (Asker, *[][]Measured, *[]Conditions) {
	var shown [][]Measured
	var conds []Conditions
	i := 0
	return func(_ context.Context, m []Measured, c Conditions) (SignedAnswer, error) {
		shown = append(shown, m)
		conds = append(conds, c)
		if i >= len(answers) {
			return SignedAnswer{}, errors.New("the asker ran out of scripted answers")
		}
		a := answers[i]
		i++
		return a, nil
	}, &shown, &conds
}

func fixedRun(answers map[string]answer) loop.CriterionRunner {
	return func(_ context.Context, cmd string) (int, string, error) {
		a := answers[cmd]
		return a.exit, a.output, a.err
	}
}

func measured(t *testing.T, run loop.CriterionRunner, cs ...Proposed) []Measured {
	t.Helper()
	m, _, err := Measure(context.Background(), Proposal{Criteria: cs}, run, 0)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// Signing without touching anything freezes exactly what was shown.
func TestSigningUntouchedFreezesWhatWasShown(t *testing.T) {
	run := fixedRun(map[string]answer{"a": {exit: 1}, "b": {exit: 0}})
	in := measured(t, run, Proposed{Name: "1", Command: "a"}, Proposed{Name: "2", Command: "b"})

	ask, shown, _ := asks(SignedAnswer{Signed: true, Criteria: []Proposed{
		{Name: "1", Command: "a"}, {Name: "2", Command: "b"},
	}, Protected: []string{"**/*_test.go"}})

	set, err := Sign(context.Background(), in, ask, run, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(*shown) != 1 {
		t.Fatalf("asked %d times, want once", len(*shown))
	}
	if len(set.Criteria) != 2 || set.Criteria[0].Name != "1" || set.Criteria[1].Command != "b" {
		t.Fatalf("frozen set is %+v", set.Criteria)
	}
	if len(set.Protected) != 1 || set.Protected[0] != "**/*_test.go" {
		t.Errorf("protected did not survive the signature: %+v", set.Protected)
	}
}

// The operator's answer carries the set AS THEY LEFT IT, not a verdict on the
// one proposed. Changing item 3 must not mean redoing everything.
func TestTheAnswerIsTheSetTheOperatorLeft(t *testing.T) {
	run := fixedRun(map[string]answer{"a": {exit: 1}, "c": {exit: 1}})
	in := measured(t, run, Proposed{Name: "1", Command: "a"}, Proposed{Name: "2", Command: "a"})

	// They dropped 2, kept 1, and the class of 1 does not change.
	ask, _, _ := asks(SignedAnswer{Signed: true, Criteria: []Proposed{{Name: "1", Command: "a"}}})
	set, err := Sign(context.Background(), in, ask, run, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Criteria) != 1 || set.Criteria[0].Name != "1" {
		t.Fatalf("the operator's deletion did not survive: %+v", set.Criteria)
	}
}

// An edited criterion is measured AGAIN before anything freezes.
//
// Without it the operator's own edit escapes the rule the package exists to
// apply: a hand-written, already-green command would enter as acceptance
// without ever having been red.
func TestAnEditedCriterionIsMeasuredAgain(t *testing.T) {
	run := fixedRun(map[string]answer{
		"pnpm test:new": {exit: 1}, // red, acceptance
		"test -f x":     {exit: 0}, // green — the edit changes what it is
	})
	in := measured(t, run, Proposed{Name: "1", Command: "pnpm test:new", Expects: ExpectFail})

	edited := SignedAnswer{Signed: true, Criteria: []Proposed{{Name: "1", Command: "test -f x", Expects: ExpectFail}}}
	// Round two: shown the new class, they sign it again unchanged.
	ask, shown, _ := asks(edited, SignedAnswer{Signed: true, Criteria: edited.Criteria})

	set, err := Sign(context.Background(), in, ask, run, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(*shown) != 2 {
		t.Fatalf("asked %d times; an edit that changed the class must go back once", len(*shown))
	}
	if got := (*shown)[1][0].Class; got != ClassRegression {
		t.Errorf("the second round showed class %q; the edit was not re-measured", got)
	}
	if len(set.Criteria) != 1 || set.Criteria[0].Command != "test -f x" {
		t.Fatalf("frozen set is %+v", set.Criteria)
	}
}

// An edit that does not change what the criterion measures settles at once.
// Going back for a class the operator already saw is asking them twice.
func TestAnEditThatKeepsTheClassSettlesAtOnce(t *testing.T) {
	run := fixedRun(map[string]answer{"a": {exit: 1}, "a --verbose": {exit: 1}})
	in := measured(t, run, Proposed{Name: "1", Command: "a"})

	ask, shown, _ := asks(SignedAnswer{Signed: true, Criteria: []Proposed{{Name: "1", Command: "a --verbose"}}})
	if _, err := Sign(context.Background(), in, ask, run, 0); err != nil {
		t.Fatal(err)
	}
	if len(*shown) != 1 {
		t.Fatalf("asked %d times; the class did not change and the operator saw it already", len(*shown))
	}
}

// A criterion the operator ADDED has never been measured, so the set cannot
// settle on the round it appears. Signing a class nobody has seen is the blind
// spot the re-measurement exists to close.
func TestAnAddedCriterionGoesBackOnce(t *testing.T) {
	run := fixedRun(map[string]answer{"a": {exit: 1}, "b": {exit: 1}})
	in := measured(t, run, Proposed{Name: "1", Command: "a"})

	added := []Proposed{{Name: "1", Command: "a"}, {Name: "2", Command: "b"}}
	ask, shown, _ := asks(SignedAnswer{Signed: true, Criteria: added}, SignedAnswer{Signed: true, Criteria: added})
	if _, err := Sign(context.Background(), in, ask, run, 0); err != nil {
		t.Fatal(err)
	}
	if len(*shown) != 2 {
		t.Fatalf("asked %d times; an unmeasured criterion must be shown before it freezes", len(*shown))
	}
}

// Refusing ends the turn. There is no third state and no default.
func TestRefusingEndsIt(t *testing.T) {
	run := fixedRun(map[string]answer{"a": {exit: 1}})
	in := measured(t, run, Proposed{Name: "1", Command: "a"})

	ask, _, _ := asks(SignedAnswer{Signed: false})
	if _, err := Sign(context.Background(), in, ask, run, 0); !errors.Is(err, ErrRefused) {
		t.Fatalf("refusing gave %v, want ErrRefused", err)
	}
}

// The deadline passing is a refusal, never an approval. A deadline that
// approved would be the quietest way to start a turn against a ruler nobody
// read.
func TestTheDeadlineRefusesAndNeverApproves(t *testing.T) {
	run := fixedRun(map[string]answer{"a": {exit: 1}})
	in := measured(t, run, Proposed{Name: "1", Command: "a"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ask, shown, _ := asks(SignedAnswer{Signed: true, Criteria: []Proposed{{Name: "1", Command: "a"}}})
	if _, err := Sign(ctx, in, ask, run, 0); !errors.Is(err, ErrRefused) {
		t.Fatalf("an expired deadline gave %v, want ErrRefused", err)
	}
	if len(*shown) != 0 {
		t.Error("an expired deadline still asked")
	}
}

// Running out of rounds refuses too. A ceiling that accepted the last state
// would be the deadline defect wearing another name.
func TestTheRoundLimitRefusesAndNeverApproves(t *testing.T) {
	// Every round the operator edits into something with a different class.
	flip := []string{"red", "green", "red", "green", "red", "green", "red"}
	run := fixedRun(map[string]answer{"red": {exit: 1}, "green": {exit: 0}})
	in := measured(t, run, Proposed{Name: "1", Command: "red"})

	i := 0
	ask := Asker(func(_ context.Context, _ []Measured, _ Conditions) (SignedAnswer, error) {
		i++
		return SignedAnswer{Signed: true, Criteria: []Proposed{{Name: "1", Command: flip[i%len(flip)]}}}, nil
	})
	_, err := Sign(context.Background(), in, ask, run, 0)
	if !errors.Is(err, ErrRefused) {
		t.Fatalf("running out of rounds gave %v, want ErrRefused", err)
	}
	if i != MaxSignRounds {
		t.Errorf("asked %d times, want %d", i, MaxSignRounds)
	}
}

// Signing an empty set is signing "nothing to verify", which the loop reports
// as done. The absence of a definition is not a permissive definition,
// whoever produced it.
func TestSigningAnEmptySetIsRefused(t *testing.T) {
	run := fixedRun(map[string]answer{"a": {exit: 1}})
	in := measured(t, run, Proposed{Name: "1", Command: "a"})

	ask, _, _ := asks(SignedAnswer{Signed: true, Criteria: nil})
	if _, err := Sign(context.Background(), in, ask, run, 0); !errors.Is(err, ErrEmptyProposal) {
		t.Fatalf("signing an empty set gave %v, want ErrEmptyProposal", err)
	}
	if _, err := Sign(context.Background(), nil, ask, run, 0); !errors.Is(err, ErrEmptyProposal) {
		t.Fatalf("signing nothing at all gave %v, want ErrEmptyProposal", err)
	}
}

// A broken criterion does not reach the frozen set. It measures nothing about
// the work and would sit unmet for the whole turn, holding the loop open
// against a failure that has nothing to do with the code.
func TestABrokenCriterionDoesNotReachTheFrozenSet(t *testing.T) {
	run := fixedRun(map[string]answer{"a": {exit: 1}, "pnmp test": {exit: 127}})
	in := measured(t, run, Proposed{Name: "1", Command: "a"}, Proposed{Name: "2", Command: "pnmp test"})

	same := []Proposed{{Name: "1", Command: "a"}, {Name: "2", Command: "pnmp test"}}
	ask, _, _ := asks(SignedAnswer{Signed: true, Criteria: same})
	set, err := Sign(context.Background(), in, ask, run, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Criteria) != 1 || set.Criteria[0].Name != "1" {
		t.Fatalf("a broken criterion was frozen into the set: %+v", set.Criteria)
	}
}

// The operator is shown the condition alongside the set, so "nothing here is
// red" is something they sign knowingly rather than discover afterwards.
func TestTheConditionIsShownWithTheSet(t *testing.T) {
	run := fixedRun(map[string]answer{"a": {exit: 0}})
	in := measured(t, run, Proposed{Name: "1", Command: "a"})

	ask, _, conds := asks(SignedAnswer{Signed: true, Criteria: []Proposed{{Name: "1", Command: "a"}}})
	if _, err := Sign(context.Background(), in, ask, run, 0); err != nil {
		t.Fatal(err)
	}
	if !(*conds)[0].NoAcceptance {
		t.Error("a set with nothing red was shown without the condition")
	}
}

// No asker is an error rather than a silent approval: a proposal nobody can
// put in front of anyone cannot be signed.
func TestNoAskerIsAnError(t *testing.T) {
	run := fixedRun(map[string]answer{"a": {exit: 1}})
	in := measured(t, run, Proposed{Name: "1", Command: "a"})
	if _, err := Sign(context.Background(), in, nil, run, 0); err == nil {
		t.Fatal("signing with no asker came back with no error")
	}
}

// Signing with nothing able to re-measure is refused up front.
//
// The whole point of the round trip is measuring what the operator edited. A
// signature that cannot re-measure would freeze a criterion whose class nobody
// knows, which is the blind spot this exists to close — so it is refused
// before anyone is asked anything, rather than discovered halfway.
func TestSigningWithNoRunnerIsRefusedUpFront(t *testing.T) {
	run := fixedRun(map[string]answer{"a": {exit: 1}})
	in := measured(t, run, Proposed{Name: "1", Command: "a"})

	ask, shown, _ := asks(SignedAnswer{Signed: true, Criteria: []Proposed{{Name: "1", Command: "a"}}})
	if _, err := Sign(context.Background(), in, ask, nil, time.Second); err == nil {
		t.Fatal("signing with no runner came back with no error")
	}
	if len(*shown) != 0 {
		t.Error("the operator was asked to sign something that could not be re-measured")
	}
}

// A set that becomes empty on the way out is refused.
//
// Broken criteria are dropped because they cannot run, and dropping them can
// leave nothing — which the loop would read as "nothing to verify" and report
// as done. The proposal was not empty; it became empty, and that is the one
// door this defect had left.
func TestASetThatEmptiesItselfIsRefused(t *testing.T) {
	run := fixedRun(map[string]answer{"pnmp test": {exit: 127}, "yran lint": {exit: 127}})
	in := measured(t, run,
		Proposed{Name: "1", Command: "pnmp test"},
		Proposed{Name: "2", Command: "yran lint"})

	same := []Proposed{{Name: "1", Command: "pnmp test"}, {Name: "2", Command: "yran lint"}}
	ask, _, _ := asks(SignedAnswer{Signed: true, Criteria: same})

	set, err := Sign(context.Background(), in, ask, run, 0)
	if !errors.Is(err, ErrEmptyProposal) {
		t.Fatalf("a set of nothing but broken criteria gave %v and %+v, want ErrEmptyProposal", err, set)
	}
}

// The channel failing is a refusal too.
//
// Nobody said yes. A transport error that fell through to a signature would be
// the deadline defect a third time: the turn starting against a ruler nobody
// read, on the strength of nobody having said no.
func TestAFailedAskIsARefusal(t *testing.T) {
	run := fixedRun(map[string]answer{"a": {exit: 1}})
	in := measured(t, run, Proposed{Name: "1", Command: "a"})

	ask := Asker(func(context.Context, []Measured, Conditions) (SignedAnswer, error) {
		return SignedAnswer{Signed: true}, errors.New("the client went away")
	})
	if _, err := Sign(context.Background(), in, ask, run, 0); !errors.Is(err, ErrRefused) {
		t.Fatalf("a failed ask gave %v, want ErrRefused", err)
	}
}
