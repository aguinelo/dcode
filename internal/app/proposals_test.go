package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/loop/qualifier"
	"github.com/aguinelo/dcode/internal/tools"
)

// ticking is a clock that moves one minute every time it is read, so each
// proposal is stamped with a time the test can name.
func ticking(start time.Time) func() time.Time {
	n := 0
	return func() time.Time {
		n++
		return start.Add(time.Duration(n) * time.Minute)
	}
}

// Every proposal of a qualifying turn is kept, in order, each with its place
// and the time it arrived — and the last one is the one written.
//
// The history is the defence against a manipulated re-proposal: narrowing a
// criterion after seeing it pass is right, swapping it for a trivial one is
// wrong, and only the sequence tells the two apart.
func TestEveryProposalIsKeptInOrder(t *testing.T) {
	start := time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC)
	held := &Proposals{now: ticking(start)}
	for _, spec := range []string{"a", "b", "c"} {
		if err := held.put(&Proposal{Spec: spec}); err != nil {
			t.Fatalf("proposing %s: %v", spec, err)
		}
	}

	every, err := held.Take()
	if err != nil {
		t.Fatal(err)
	}
	if len(every) != 3 {
		t.Fatalf("kept %d proposals, want all 3", len(every))
	}
	for i, p := range every {
		want := string(rune('a' + i))
		if p.Spec != want || p.Sequence != i+1 || !p.At.Equal(start.Add(time.Duration(i+1)*time.Minute)) {
			t.Errorf("proposal %d is %q #%d at %v; want %q #%d at %v",
				i, p.Spec, p.Sequence, p.At, want, i+1, start.Add(time.Duration(i+1)*time.Minute))
		}
	}
	if again, _ := held.Take(); again != nil {
		t.Errorf("the history survived being taken: %+v", again)
	}
}

// The ceiling is declared, and reaching it fails the qualifying turn out loud:
// the proposal over it is refused to the model with the reason, and the commit
// that follows writes nothing and says why. Never a silent drop of the oldest.
func TestTheProposalCeilingFailsTheQualifyingTurn(t *testing.T) {
	d, sess, spec := committing(t, qualifier.Proposed{Name: "a", Command: "false", Expects: qualifier.ExpectFail})
	held := d.proposals("s1")
	for i := 1; i < MaxProposals; i++ {
		if err := held.put(&Proposal{Spec: spec, Criteria: []qualifier.Proposed{{Name: "a", Command: "false"}}}); err != nil {
			t.Fatalf("proposal %d of %d refused: %v", i+1, MaxProposals, err)
		}
	}
	err := held.put(&Proposal{Spec: spec})
	if !errors.Is(err, ErrTooManyProposals) {
		t.Fatalf("proposal %d came back %v, want ErrTooManyProposals", MaxProposals+1, err)
	}
	if !strings.Contains(err.Error(), fmt.Sprint(MaxProposals)) {
		t.Errorf("the refusal does not name the ceiling: %v", err)
	}

	_, cerr := d.commitDone(context.Background(), "s1")
	if cerr == nil || !strings.Contains(cerr.Error(), fmt.Sprint(MaxProposals)) {
		t.Fatalf("a turn that crossed the ceiling was committed: %v", cerr)
	}
	if _, serr := os.Stat(filepath.Join(spec, "done.toml")); !os.IsNotExist(serr) {
		t.Errorf("a turn that crossed the ceiling wrote its file: %v", serr)
	}
	if got := announced(t, sess); len(got) != 0 {
		t.Errorf("a turn that crossed the ceiling was announced: %v", got)
	}
}

// The model is told, in the tool's answer, that the ceiling was reached.
func TestTheToolSaysTheCeilingWasReached(t *testing.T) {
	held := &Proposals{}
	tool := QualifyingTool(t.TempDir(), held)
	raw, _ := json.Marshal(tools.DoneProposeInput{Criteria: []tools.DoneProposeCriterion{{Name: "a", Command: "false", Expects: "fail"}}})
	for i := 0; i < MaxProposals; i++ {
		if res, err := tool.Execute(context.Background(), raw, nil); err != nil || res.IsError {
			t.Fatalf("proposal %d: %v %+v", i+1, err, res)
		}
	}
	res, err := tool.Execute(context.Background(), raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError || !strings.Contains(res.Output, fmt.Sprint(MaxProposals)) {
		t.Errorf("proposal %d was not refused with the ceiling: %+v", MaxProposals+1, res)
	}
}

// The announcement carries the history: the written proposal says which one it
// was and when, and every earlier one comes with it, oldest first, as the model
// sent it. The summary a terminal reads says the same.
func TestTheAnnouncementCarriesEveryEarlierProposal(t *testing.T) {
	d, sess, spec := committing(t, qualifier.Proposed{Name: "first", Command: "go test ./a", Expects: qualifier.ExpectFail, Why: "too loose"})
	held := d.proposals("s1")
	if err := held.put(&Proposal{Spec: spec, Criteria: []qualifier.Proposed{
		{Name: "second", Command: "false", Expects: qualifier.ExpectFail},
	}}); err != nil {
		t.Fatal(err)
	}

	out, err := d.commitDone(context.Background(), "s1")
	if err != nil {
		t.Fatal(err)
	}
	if out.Proposals != 2 {
		t.Errorf("the response says %d proposals, want 2", out.Proposals)
	}
	if !strings.Contains(out.Summary, "proposed 2 times") || !strings.Contains(out.Summary, "#1") ||
		!strings.Contains(out.Summary, "go test ./a") {
		t.Errorf("the summary does not show the earlier proposal:\n%s", out.Summary)
	}

	got := announced(t, sess)
	if len(got) != 1 {
		t.Fatalf("announced %d times, want once", len(got))
	}
	p := got[0]
	if p["sequence"] != float64(2) {
		t.Errorf("the written proposal is #%v, want #2", p["sequence"])
	}
	if at, _ := p["proposed_at"].(string); at == "" {
		t.Error("the written proposal has no proposed_at")
	}
	criteria, _ := p["criteria"].([]any)
	if c, _ := criteria[0].(map[string]any); len(criteria) != 1 || c["name"] != "second" {
		t.Errorf("the written criteria are %v, want the last proposal's", p["criteria"])
	}
	earlier, _ := p["earlier"].([]any)
	if len(earlier) != 1 {
		t.Fatalf("announced %d earlier proposals, want 1: %v", len(earlier), p["earlier"])
	}
	e, _ := earlier[0].(map[string]any)
	if e["sequence"] != float64(1) {
		t.Errorf("the earlier proposal is #%v, want #1", e["sequence"])
	}
	if at, _ := e["proposed_at"].(string); at == "" {
		t.Error("the earlier proposal has no proposed_at")
	}
	ec, _ := e["criteria"].([]any)
	first, _ := ec[0].(map[string]any)
	if len(ec) != 1 || first["name"] != "first" || first["command"] != "go test ./a" ||
		first["expects"] != "fail" || first["why"] != "too loose" {
		t.Errorf("the earlier criteria are %v, want the first proposal as sent", e["criteria"])
	}
	if _, measured := first["class"]; measured {
		t.Errorf("an earlier proposal claims a measurement it never had: %v", first)
	}
}

// One proposal is the common case, and it says nothing about earlier ones.
func TestASingleProposalHasNoEarlierOnes(t *testing.T) {
	d, sess, _ := committing(t, qualifier.Proposed{Name: "a", Command: "false", Expects: qualifier.ExpectFail})
	out, err := d.commitDone(context.Background(), "s1")
	if err != nil {
		t.Fatal(err)
	}
	if out.Proposals != 1 || strings.Contains(out.Summary, "proposed 1 times") {
		t.Errorf("a single proposal: %d, summary\n%s", out.Proposals, out.Summary)
	}
	p := announced(t, sess)[0]
	if _, ok := p["earlier"]; ok || p["sequence"] != float64(1) {
		t.Errorf("a single proposal announced %v", p)
	}
}
