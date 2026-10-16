package tui

import (
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// runLoop drives a whole /loop run the way the program actually runs one.
//
// The tests that existed injected `loopFinishedMsg` ready-made and asserted the
// notice draws what it is handed. That measures the rendering and nothing else:
// what the run HANDS it, after several specs with different outcomes, had never
// been asserted anywhere — which is the gap this file exists to close.
//
// Each entry in `outcomes` is one spec's completion, in queue order.
func runLoop(t *testing.T, outcomes []*protocol.Completion) *program {
	t.Helper()
	specs := make([]protocol.SpecFolder, 0, len(outcomes))
	completions := map[string]*protocol.Completion{}
	for i, c := range outcomes {
		specs = append(specs, protocol.SpecFolder{
			Path:     specPath(i),
			Criteria: 3,
			Pending:  true,
			Unmet:    1,
		})
		completions[specPath(i)] = c
	}
	p, tr := loopProgram(t, specs...)
	tr.completions = completions
	// The daemon works every spec and keeps where each was left; the TUI
	// follows it to the end and says it.
	typeLoop(t, p, "finish the backlog")
	followLoop(t, p, nil)
	return p
}

func specPath(i int) string {
	return "docs/specs/architecture/family-" + string(rune('a'+i))
}

// The verdict at the end of a run is about the RUN, not about the last spec.
//
// This is the report: a loop that "lets things through and justifies them". It
// does, and here is the mechanism — `loopStanding` is overwritten by every
// turn-completed event, so when the queue empties it holds whatever the LAST
// spec happened to say. A run whose first spec left coverage unmet and whose
// second spec was clean ends by announcing that everything is met.
//
// Nothing is lying. The number is true about the last spec and is printed under
// a sentence about the run, which is the worst kind of wrong: a person reading
// it has no way to tell, and the specs that failed are the ones they most need
// named.
func TestTheVerdictIsAboutTheRunAndNotTheLastSpec(t *testing.T) {
	p := runLoop(t, []*protocol.Completion{
		{Met: []string{"tests"}, Unmet: []string{"coverage"}, Unavailable: []string{"integration"}},
		{Met: []string{"tests", "vet", "build"}},
	})

	if len(p.model.Entries) == 0 {
		t.Fatal("the run drew nothing at all")
	}
	got := p.model.Entries[len(p.model.Entries)-1].Summary

	for _, want := range []string{"coverage", "integration"} {
		if !strings.Contains(got, want) {
			t.Errorf("the run ended without naming %q, which no spec met:\n%s", want, got)
		}
	}
	if strings.Contains(got, "4 of 4") {
		t.Errorf("the run reports every criterion met while a spec left one unmet:\n%s", got)
	}
}

// A spec that failed is named, so the person knows WHICH one to go back to.
//
// A count answers how much; only a name answers what to do next, and the end of
// a run is exactly when that is the question. The existing notice already
// applies that rule to criteria and cannot apply it to specs, because the run
// never kept which spec a standing came from.
func TestTheRunNamesTheSpecThatDidNotFinish(t *testing.T) {
	p := runLoop(t, []*protocol.Completion{
		{Met: []string{"tests"}, Unmet: []string{"coverage"}},
		{Met: []string{"tests", "vet"}},
	})
	got := p.model.Entries[len(p.model.Entries)-1].Summary
	if !strings.Contains(got, specPath(0)) {
		t.Errorf("the run does not name the spec that was left unmet:\n%s", got)
	}
}

// A run that worked specs and met nothing still ends out loud.
//
// The silence rule is right for its own case — the queue empties on every
// proposal commit, including the ones where there was never a queue — but it is
// keyed on `worked == 0`, and a run of one spec that ended with everything
// unmet is the run most worth announcing.
func TestARunThatMetNothingStillSaysSo(t *testing.T) {
	p := runLoop(t, []*protocol.Completion{
		{Unmet: []string{"coverage", "tests"}},
	})
	got := p.model.Entries[len(p.model.Entries)-1].Summary
	for _, want := range []string{"coverage", "tests"} {
		if !strings.Contains(got, want) {
			t.Errorf("a run that met nothing did not name %q:\n%s", want, got)
		}
	}
}

// The run ends from the idle branch even when nothing is queued.
//
// It did not: the advance was gated on "there is more queued", so when the last
// spec finished the queue was already empty, nothing was called and nothing was
// said. An ordinary run never announced its own end; the only path that ever
// produced the notice was a proposal commit. This asserts the branch fires by
// its one observable side effect — the run is cleared — without executing the
// batch it returns, which waits on the event channel.
func TestAnIdleSessionWithAnEmptyQueueEndsTheRun(t *testing.T) {
	p, tr := loopProgram(t, protocol.SpecFolder{Path: specPath(0), Criteria: 2, Pending: true, Unmet: 2})
	tr.completions = map[string]*protocol.Completion{specPath(0): {Unmet: []string{"tests"}}}
	typeLoop(t, p, "one spec")
	followLoop(t, p, nil)
	if run := tr.lastLoop(t); run.State != protocol.LoopFinished || run.Verdict == nil || run.Verdict.Worked != 1 {
		t.Fatalf("the first spec is not counted as worked, or the run did not end: %+v", run)
	}
	if !sawEntry(p, "finished", "tests") {
		t.Errorf("the session went idle with an empty queue and the run did not end: %q", summaries(p))
	}
}

// The end of a run says how many specs it worked once, in either language.
//
// It said "the loop is finished — 2 2 specs worked": the sentence put the
// count in front of a noun that plural() had already counted.
func TestTheEndOfARunCountsTheSpecsOnce(t *testing.T) {
	for _, c := range []struct {
		lang   Lang
		worked int
		want   string
	}{
		{En, 2, "the loop is finished — 2 specs worked"},
		{En, 1, "the loop is finished — 1 spec worked"},
		{PtBR, 2, "o laço terminou — 2 specs trabalhada(s)"},
		{PtBR, 1, "o laço terminou — 1 spec trabalhada(s)"},
	} {
		p, _ := loopProgram(t)
		p.model.Lang = c.lang
		p.loopEnded(protocol.LoopRun{State: protocol.LoopFinished, Goal: "the backlog",
			Verdict: &protocol.LoopVerdict{Worked: c.worked}})
		if got := summaries(p); len(got) != 1 || got[0] != c.want {
			t.Errorf("%s, %d worked: the end reads %q, want %q", c.lang, c.worked, got, c.want)
		}
	}
}
