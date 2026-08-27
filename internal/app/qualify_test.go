package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/loop/qualifier"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/session"
	"github.com/aguinelo/dcode/internal/tools"
)

func fixedRunner(answers map[string]int) loop.CriterionRunner {
	return func(_ context.Context, cmd string) (int, string, error) {
		return answers[cmd], "", nil
	}
}

func propose(t *testing.T, dir string, run loop.CriterionRunner, in tools.DoneProposeInput) (string, error) {
	t.Helper()
	held := &Proposals{}
	tool := QualifyingTool(dir, held)
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	res, err := tool.Execute(context.Background(), raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		return res.Output, errFromResult(res)
	}
	// The tool records; the LOOP measures and writes. Doing both here is what
	// the caller is testing, and doing them separately is the whole point of
	// the split: the turn cannot write, so the write happens after it.
	out, cerr := CommitProposal(context.Background(), held.Take(), run, 0)
	if cerr != nil {
		return res.Output, cerr
	}
	return out, nil
}

type toolErr struct{ msg string }

func (e toolErr) Error() string { return e.msg }

func errFromResult(r tools.Result) error { return toolErr{r.Output} }

// The proposal is measured before it is written, and the file carries what the
// measurement said.
//
// The model does not write it: it calls the tool, and the HARNESS runs each
// criterion and writes the result. That division is what makes the proposal
// measurable at all.
func TestAProposalIsMeasuredBeforeItIsWritten(t *testing.T) {
	dir := t.TempDir()
	out, err := propose(t, dir, fixedRunner(map[string]int{"make test": 1, "make lint": 0}),
		tools.DoneProposeInput{
			Criteria: []tools.DoneProposeCriterion{
				{Name: "unit", Command: "make test", Expects: "fail", Why: "o trabalho faz passar"},
				{Name: "lint", Command: "make lint", Expects: "pass"},
			},
			Protected: []string{"tests/**"},
		})
	if err != nil {
		t.Fatal(err)
	}

	body, rerr := os.ReadFile(filepath.Join(dir, tools.DoneProposeFile))
	if rerr != nil {
		t.Fatal(rerr)
	}
	for _, want := range []string{"[unit]", "acceptance", "[lint]", "regression", `protected = "tests/**"`} {
		if !strings.Contains(string(body), want) {
			t.Errorf("the file does not carry %q:\n%s", want, body)
		}
	}
	// And the model is told what happened, so it can fix a broken one itself.
	for _, want := range []string{"acceptance", "regression", "do not start the work"} {
		if !strings.Contains(out, want) {
			t.Errorf("the summary does not carry %q:\n%s", want, out)
		}
	}
}

// The file it writes is the file the loader reads. A proposal nobody can load
// back is a proposal that did nothing.
func TestAProposalRoundTripsIntoADefinitionOfDone(t *testing.T) {
	dir := t.TempDir()
	if _, err := propose(t, dir, fixedRunner(map[string]int{"a": 1, "b": 1}),
		tools.DoneProposeInput{Criteria: []tools.DoneProposeCriterion{
			{Name: "one", Command: "a", Expects: "fail"},
			{Name: "two", Command: "b", ExitCode: 3, Expects: "fail"},
		}}); err != nil {
		t.Fatal(err)
	}

	set, err := sessionDoneSet(Options{Workspace: t.TempDir(), LoopSpec: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Criteria) != 2 {
		t.Fatalf("the proposal did not load back: %+v", set.Criteria)
	}
	if set.Criteria[0].Command != "a" || set.Criteria[1].ExitCode != 3 {
		t.Errorf("what loaded back is not what was proposed: %+v", set.Criteria)
	}
}

// A qualifying session is measured against nothing: it is the one working out
// what the measurement will be, and giving it a definition of done would be
// asking it to satisfy the ruler it is drawing.
func TestAQualifyingSessionIsMeasuredAgainstNothing(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "done.toml"), []byte("[x]\ncommand = \"true\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	set, err := sessionDoneSet(Options{Workspace: t.TempDir(), LoopSpec: dir, Qualify: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Criteria) != 0 {
		t.Errorf("a qualifying session carries criteria: %+v", set.Criteria)
	}
}

// A proposal with nothing in it is refused: a definition of done with no
// criteria reports done.
func TestAnEmptyProposalIsRefused(t *testing.T) {
	if _, err := propose(t, t.TempDir(), fixedRunner(nil), tools.DoneProposeInput{}); err == nil {
		t.Fatal("an empty proposal was accepted")
	}
}

// Outside a qualifying turn the tool refuses rather than working: it is the
// loop that decides there is a qualification, not the model.
func TestTheToolRefusesOutsideAQualifyingTurn(t *testing.T) {
	raw, _ := json.Marshal(tools.DoneProposeInput{
		Criteria: []tools.DoneProposeCriterion{{Name: "a", Command: "b", Expects: "fail"}},
	})
	res, err := tools.DonePropose{}.Execute(context.Background(), raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError || !strings.Contains(res.Output, "qualifying turn") {
		t.Errorf("the tool worked outside a qualifying turn: %+v", res)
	}
}

// done_propose touches nothing, and plan mode therefore allows it.
//
// This is the whole of design A, and it was found by running: the first
// version declared a write to the spec folder — honest about the consequence,
// wrong about the actor — and read-only denied it with no exception, so the
// model correctly reported it could not propose.
//
// Nothing here writes. The proposal is recorded, the turn ends, and the loop
// measures and writes afterwards under the boundary the work will run under.
// The alternative was an exception in read-only, and an exception in a
// guarantee is the guarantee the next person widens.
func TestProposingIsAllowedInPlanModeBecauseItTouchesNothing(t *testing.T) {
	req, err := tools.DonePropose{Spec: "/w/specs/x"}.Declare(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(req.Paths) != 0 {
		t.Fatalf("done_propose declares %+v; a write here is denied by plan mode", req.Paths)
	}
	if req.Network {
		t.Error("done_propose declares the network")
	}

	// And the boundary agrees: read-only allows it.
	v := policy.Evaluate(req, policy.ModeReadOnly, policy.PolicyNever, policy.Rules{}, nil,
		func(policy.Access) bool { return true })
	if v.Decision != policy.DecisionAllow {
		t.Errorf("plan mode answers %q to a proposal: %s", v.Decision, v.Reason)
	}
}

// The proposal is measured under the boundary the WORK will run under, never
// the read-only one the proposing turn ran in.
//
// Measuring there would call a criterion broken because the sandbox refused it
// a cache directory, and a proposal born with a false measurement is worse
// than none.
func TestAProposalIsMeasuredWhereItsCriteriaCanRun(t *testing.T) {
	dir := t.TempDir()
	held := &Proposals{}
	raw, _ := json.Marshal(tools.DoneProposeInput{Criteria: []tools.DoneProposeCriterion{
		{Name: "writes", Command: "touch scratch", Expects: "fail"},
	}})
	if _, err := QualifyingTool(dir, held).Execute(context.Background(), raw, nil); err != nil {
		t.Fatal(err)
	}

	// A runner that refuses everything stands in for read-only; one that lets
	// it through stands in for the workspace boundary. The same proposal
	// measures differently, which is exactly why the loop chooses.
	p := held.Take()
	if p == nil {
		t.Fatal("nothing was recorded")
	}
	blocked, err := CommitProposal(context.Background(), p,
		func(context.Context, string) (int, string, error) { return 127, "denied", nil }, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(blocked, "broken") {
		t.Errorf("a criterion the boundary refused did not come back broken:\n%s", blocked)
	}
}

// The daemon holds a qualifying session's proposal until the loop takes it,
// and taking it is the end of its life.
//
// Take rather than Get: committing a proposal is what it was for, and one that
// survived being written would be written again by the next commit.
func TestTheDaemonHoldsAProposalUntilTheLoopTakesIt(t *testing.T) {
	d := &Daemon{}
	if got := d.proposals("s1"); got != nil {
		t.Fatalf("a session that proposed nothing has %+v", got)
	}

	held := &Proposals{}
	d.holdProposals("s1", held)
	if d.proposals("s1") != held {
		t.Fatal("the proposal was not held against its session")
	}
	if d.proposals("s2") != nil {
		t.Error("another session sees it")
	}

	held.put(&Proposal{Spec: "/w/specs/x"})
	if got := held.Take(); got == nil || got.Spec != "/w/specs/x" {
		t.Fatalf("took %+v", got)
	}
	if got := held.Take(); got != nil {
		t.Errorf("a proposal survived being taken: %+v", got)
	}
}

// A second proposal replaces the first. A model that proposes twice has
// changed its mind, and keeping both would leave the loop choosing between
// them.
func TestASecondProposalReplacesTheFirst(t *testing.T) {
	held := &Proposals{}
	held.put(&Proposal{Spec: "a"})
	held.put(&Proposal{Spec: "b"})
	got := held.Take()
	if got == nil || got.Spec != "b" {
		t.Fatalf("took %+v, want the later one", got)
	}
}

// Committing nothing is an error, not an empty file: a definition of done with
// nothing in it reports done.
func TestCommittingNothingIsAnError(t *testing.T) {
	if _, err := CommitProposal(context.Background(), nil, fixedRunner(nil), 0); err == nil {
		t.Fatal("committing nothing came back with no error")
	}
}

// Qualifying forces plan mode and nothing else does; a request that is not
// qualifying keeps the boundary it asked for.
func TestQualifyModeOnlyAppliesToAQualifyingSession(t *testing.T) {
	base := Options{SandboxMode: policy.ModeFullAccess, Policy: policy.PolicyOnRequest}
	if got := qualifyMode(base, false); got.SandboxMode != policy.ModeFullAccess || got.Qualify {
		t.Errorf("a non-qualifying session was changed: %+v", got)
	}
	if got := qualifyMode(base, true); got.SandboxMode != policy.ModeReadOnly || !got.Qualify {
		t.Errorf("a qualifying session was not forced to plan mode: %+v", got)
	}
}

// A spec path that climbs out is refused before any of this begins.
func TestQualifyOptionsRefusesAPathOutsideTheWorkspace(t *testing.T) {
	d := &Daemon{opts: DaemonOptions{Base: Options{SandboxMode: policy.ModeReadOnly}}}
	if _, err := d.qualifyOptions(t.TempDir(), protocol.CreateSessionRequest{
		LoopSpec: "../../elsewhere", Qualify: true,
	}); err == nil {
		t.Fatal("a spec outside the workspace was accepted")
	}
	if _, err := d.qualifyOptions("relative", protocol.CreateSessionRequest{Qualify: true}); err == nil {
		t.Fatal("a relative workspace was accepted")
	}
}

// The daemon's commit refuses a session that is not qualifying, and one that
// proposed nothing — and writes the file when there is one.
func TestTheDaemonCommitRefusesWhatItCannotWrite(t *testing.T) {
	d := NewDaemon(DaemonOptions{Base: Options{SandboxMode: policy.ModeReadOnly}})
	ws := t.TempDir()
	spec := filepath.Join(ws, "specs", "x")
	if err := os.MkdirAll(spec, 0o755); err != nil {
		t.Fatal(err)
	}
	sess := newBareSession(t, "s1", ws)
	if err := d.manager.Add(sess); err != nil {
		t.Fatal(err)
	}

	// Not a qualifying session at all.
	if _, err := d.commitDone(context.Background(), "s1"); err == nil {
		t.Error("a session that never qualified was committed")
	}
	// A session that does not exist.
	if _, err := d.commitDone(context.Background(), "nope"); err == nil {
		t.Error("an unknown session was committed")
	}
	// Qualifying, but nothing was proposed.
	d.holdProposals("s1", &Proposals{})
	if _, err := d.commitDone(context.Background(), "s1"); err == nil {
		t.Error("a session that proposed nothing was committed")
	}
}

// A criterion whose output is enormous is cut before it reaches the file: the
// proposal is something a person reads.
func TestACommittedProposalCutsAHugeOutput(t *testing.T) {
	dir := t.TempDir()
	huge := strings.Repeat("y", 40_000)
	out, err := CommitProposal(context.Background(), &Proposal{
		Spec:     dir,
		Criteria: []qualifier.Proposed{{Name: "loud", Command: "shout", Expects: qualifier.ExpectFail}},
	}, func(context.Context, string) (int, string, error) { return 1, huge, nil }, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) > 10_000 {
		t.Errorf("the summary is %d bytes; it is meant to be read", len(out))
	}
	body, rerr := os.ReadFile(filepath.Join(dir, "done.toml"))
	if rerr != nil {
		t.Fatal(rerr)
	}
	if len(body) > 10_000 {
		t.Errorf("the file is %d bytes; it is meant to be read", len(body))
	}
}

// Writing where nothing can be written is an error that names the path.
func TestACommitThatCannotWriteSaysWhere(t *testing.T) {
	_, err := CommitProposal(context.Background(), &Proposal{
		Spec:     filepath.Join(t.TempDir(), "not", "there"),
		Criteria: []qualifier.Proposed{{Name: "a", Command: "b", Expects: qualifier.ExpectFail}},
	}, fixedRunner(map[string]int{"b": 1}), 0)
	if err == nil {
		t.Fatal("writing into a directory that does not exist came back with no error")
	}
	if !strings.Contains(err.Error(), "done.toml") {
		t.Errorf("the error does not name the file: %v", err)
	}
}

func newBareSession(t *testing.T, id, ws string) *session.Session {
	t.Helper()
	return session.New(id, ws, "m", "read-only", nil,
		session.NewEventLog(id, 0, time.Now), time.Now)
}
