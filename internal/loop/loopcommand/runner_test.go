package loopcommand

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// fakeDaemon is the runner's transport with no model behind it: the sequence
// is deterministic, so it is asserted on what was asked of the daemon.
type fakeDaemon struct {
	mu sync.Mutex

	found     []protocol.SpecFolder
	surveyErr error
	openErr   error
	commitErr error
	// standing is the completion a work session on a folder ends with.
	standing map[string]*protocol.Completion
	// hold makes the work turns of these folders wait until the channel
	// closes, or until they are cancelled.
	hold map[string]chan struct{}
	// reason is how a work turn on a folder ends; "done" when unset.
	reason map[string]string
	// surveyHold makes a survey wait for its context, as a slow one would.
	surveyHold bool
	// openHold makes Open wait until it closes, ignoring the context, as the
	// daemon's does; openEntered says Open was reached.
	openHold    chan struct{}
	openEntered chan struct{}
	// qualifyHold makes a qualifying turn wait until it closes, ignoring the
	// context: a turn that ends on its own just as it is stopped.
	qualifyHold chan struct{}

	surveys    []bool // measure, per survey
	opened     []protocol.CreateSessionRequest
	turns      []string // "<spec>: <text>"
	committed  []string
	interrupts []string
	started    chan string // the spec of each turn, as it starts
}

func newFake(found ...protocol.SpecFolder) *fakeDaemon {
	return &fakeDaemon{found: found, standing: map[string]*protocol.Completion{},
		hold: map[string]chan struct{}{}, reason: map[string]string{}, started: make(chan string, 16)}
}

func (f *fakeDaemon) ListSpecs(ctx context.Context, ws string, measure bool) ([]protocol.SpecFolder, error) {
	f.mu.Lock()
	f.surveys = append(f.surveys, measure)
	hold := f.surveyHold
	f.mu.Unlock()
	if hold {
		f.started <- "survey"
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return f.found, f.surveyErr
}

func (f *fakeDaemon) Open(_ context.Context, req protocol.CreateSessionRequest) (protocol.Session, error) {
	if f.openHold != nil {
		f.openEntered <- struct{}{}
		<-f.openHold
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.openErr != nil {
		return protocol.Session{}, f.openErr
	}
	f.opened = append(f.opened, req)
	return protocol.Session{ID: fmt.Sprintf("s%d", len(f.opened))}, nil
}

func (f *fakeDaemon) spec(id string) protocol.CreateSessionRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	var n int
	fmt.Sscanf(id, "s%d", &n)
	return f.opened[n-1]
}

func (f *fakeDaemon) RunTurn(ctx context.Context, id, text string) (protocol.TurnCompleted, error) {
	req := f.spec(id)
	f.mu.Lock()
	f.turns = append(f.turns, req.LoopSpec+": "+text)
	hold := f.hold[req.LoopSpec]
	if req.Qualify {
		hold = nil
	}
	done := f.standing[req.LoopSpec]
	reason := f.reason[req.LoopSpec]
	qhold := f.qualifyHold
	f.mu.Unlock()
	if reason == "" {
		reason = protocol.StopDone
	}
	f.started <- req.LoopSpec
	if req.Qualify && qhold != nil {
		<-qhold
	}
	if hold != nil {
		select {
		case <-hold:
		case <-ctx.Done():
			return protocol.TurnCompleted{}, ctx.Err()
		}
	}
	if req.Qualify {
		return protocol.TurnCompleted{Reason: "done"}, nil
	}
	return protocol.TurnCompleted{Reason: reason, Completion: done}, nil
}

func (f *fakeDaemon) CommitDone(_ context.Context, id string) (protocol.CommitDoneResponse, error) {
	req := f.spec(id)
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.commitErr != nil {
		return protocol.CommitDoneResponse{}, f.commitErr
	}
	f.committed = append(f.committed, id)
	return protocol.CommitDoneResponse{Path: filepath.Join(req.LoopSpec, "done.toml"), Criteria: 2}, nil
}

func (f *fakeDaemon) Interrupt(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.interrupts = append(f.interrupts, id)
}

// watch records every run the runner announces, so a test can wait for one.
type watch struct {
	mu   sync.Mutex
	seen []protocol.LoopRun
	ch   chan protocol.LoopRun
}

func newWatch() *watch { return &watch{ch: make(chan protocol.LoopRun, 256)} }

func (w *watch) publish(r protocol.LoopRun) {
	w.mu.Lock()
	w.seen = append(w.seen, r)
	w.mu.Unlock()
	w.ch <- r
}

func (w *watch) until(t *testing.T, state string) protocol.LoopRun {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case r := <-w.ch:
			if r.State == state {
				return r
			}
		case <-deadline:
			t.Fatalf("the run never reached %s", state)
		}
	}
}

func looping(t *testing.T, f *fakeDaemon, args string) (*Runner, *watch) {
	t.Helper()
	w := newWatch()
	r, err := NewRunner("L1", protocol.StartLoopRequest{Workspace: "/w", Args: args, Model: "m"}, f, w.publish, nil)
	if err != nil {
		t.Fatal(err)
	}
	return r, w
}

func standing(met, unmet, unavailable []string) *protocol.Completion {
	return &protocol.Completion{Met: met, Unmet: unmet, Unavailable: unavailable}
}

// A folder that declares criteria is surveyed without running anything, then
// worked in one session that carries the folder and the person's words, and
// the run ends with where it left the folder.
func TestALoopOnAFolderWorksItAndSaysWhereItEnded(t *testing.T) {
	f := newFake(protocol.SpecFolder{Path: "specs/a", Criteria: 2})
	f.standing["specs/a"] = standing([]string{"unit", "lint"}, nil, nil)
	r, _ := looping(t, f, "specs/a refaça o header --protect tests/**")

	r.Run(context.Background())

	got := r.Snapshot()
	if got.State != protocol.LoopFinished {
		t.Fatalf("state %s (%s), want finished", got.State, got.Error)
	}
	if len(f.surveys) != 1 || f.surveys[0] {
		t.Errorf("surveys %v, want one that runs nothing", f.surveys)
	}
	if len(f.opened) != 1 || f.opened[0].LoopSpec != "specs/a" || f.opened[0].Qualify ||
		f.opened[0].Protect[0] != "tests/**" || f.opened[0].Model != "m" {
		t.Fatalf("opened %+v, want one work session on specs/a", f.opened)
	}
	if f.turns[0] != "specs/a: refaça o header" {
		t.Errorf("turn %q, want the person's words", f.turns[0])
	}
	if s := got.Specs[0]; s.Phase != protocol.SpecDone || s.SessionID != "s1" || s.Standing == nil {
		t.Errorf("spec %+v, want done in s1 with its standing", s)
	}
	if v := got.Verdict; v == nil || v.Worked != 1 || v.Met != 2 || v.Total != 2 {
		t.Errorf("verdict %+v, want 1 worked, 2 of 2", got.Verdict)
	}
	if got.Current != -1 || got.Finished.IsZero() {
		t.Errorf("an ended run still points at a folder or has no end: %+v", got)
	}
}

// A folder that declares nothing is qualified first, the proposal is written,
// and the run waits for a person; continuing works it with what they typed.
func TestAFolderThatDeclaresNothingIsQualifiedAndWaitsForReview(t *testing.T) {
	f := newFake(protocol.SpecFolder{Path: "specs/x", Criteria: 0})
	f.standing["specs/x"] = standing([]string{"a"}, nil, nil)
	r, w := looping(t, f, "specs/x refaça o header")

	go r.Run(context.Background())
	waiting := w.until(t, protocol.LoopAwaitingReview)

	if len(f.opened) != 1 || !f.opened[0].Qualify {
		t.Fatalf("opened %+v, want one qualifying session", f.opened)
	}
	if strings.Contains(f.turns[0], "refaça o header") || !strings.Contains(f.turns[0], "done_propose") {
		t.Errorf("the qualifying turn is %q: it must ask for a proposal and not carry the work", f.turns[0])
	}
	if s := waiting.Specs[0]; s.Phase != protocol.SpecProposed || s.Proposal != "specs/x/done.toml" {
		t.Errorf("spec %+v, want proposed with the file", s)
	}

	if _, err := r.Continue(); err != nil {
		t.Fatal(err)
	}
	done := w.until(t, protocol.LoopFinished)
	if len(f.opened) != 2 || f.opened[1].Qualify || f.turns[1] != "specs/x: refaça o header" {
		t.Errorf("after continue: opened %+v, turns %q", f.opened, f.turns)
	}
	if done.Specs[0].Phase != protocol.SpecDone {
		t.Errorf("phase %s, want done", done.Specs[0].Phase)
	}
}

// Continue is for a run waiting for a review, and the first answer wins.
func TestContinueOutsideAReviewIsRefused(t *testing.T) {
	f := newFake(protocol.SpecFolder{Path: "specs/x"})
	r, w := looping(t, f, "specs/x")
	if _, err := r.Continue(); !isCode(err, protocol.CodeLoopNotAwaitingReview) {
		t.Errorf("continuing a run that is surveying gave %v", err)
	}
	f.mu.Lock()
	f.hold["specs/x"] = make(chan struct{})
	f.mu.Unlock()
	go r.Run(context.Background())
	w.until(t, protocol.LoopAwaitingReview)
	if _, err := r.Continue(); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Continue(); !isCode(err, protocol.CodeLoopNotAwaitingReview) {
		t.Errorf("a second continue gave %v, want loop_not_awaiting_review", err)
	}
	turnStarted(t, f)
	turnStarted(t, f)
	r.Stop(protocol.StopNow)
	w.until(t, protocol.LoopStopped)
}

// If the daemon cannot say what a folder declares, the work runs anyway.
func TestASurveyThatFailsStillOpensTheWork(t *testing.T) {
	f := newFake()
	f.surveyErr = errors.New("disk on fire")
	r, _ := looping(t, f, "specs/a")
	r.Run(context.Background())
	got := r.Snapshot()
	if len(f.opened) != 1 || f.opened[0].Qualify || got.State != protocol.LoopFinished {
		t.Errorf("opened %+v, state %s", f.opened, got.State)
	}
	// Going on is not the same as staying quiet about it.
	if len(got.Warnings) != 1 || !strings.Contains(got.Warnings[0], "disk on fire") {
		t.Errorf("warnings %q, want the survey's failure said", got.Warnings)
	}
}

// A bare word that names no folder is a goal, qualified at .dcode.
func TestABareWordThatNamesNothingIsQualifiedAsAGoal(t *testing.T) {
	f := newFake(protocol.SpecFolder{Path: "specs/a", Criteria: 1})
	r, w := looping(t, f, "oi")
	go r.Run(context.Background())
	w.until(t, protocol.LoopAwaitingReview)
	if f.opened[0].LoopSpec != protocol.WorkspaceDoneDir || !f.opened[0].Qualify {
		t.Errorf("opened %+v, want a qualifying session at %s", f.opened[0], protocol.WorkspaceDoneDir)
	}
	r.Stop(protocol.StopAfterSpec)
	if got := w.until(t, protocol.LoopStopped); len(f.opened) != 1 {
		t.Errorf("stopping a run waiting for review opened work: %+v / %+v", f.opened, got)
	}
}

// A goal measures every folder, works the pending ones in order, and says the
// verdict of the whole run — naming the folders that did not finish.
func TestAGoalWorksEveryPendingFolderAndSumsTheRun(t *testing.T) {
	f := newFake(
		protocol.SpecFolder{Path: "specs/a", Criteria: 2, Unmet: 1, Pending: true},
		protocol.SpecFolder{Path: "specs/b", Criteria: 1},
		protocol.SpecFolder{Path: "specs/c", Criteria: 2, Unmet: 2, Pending: true},
		protocol.SpecFolder{Path: "specs/d", Error: "no tasks.md"},
	)
	f.standing["specs/a"] = standing([]string{"unit"}, []string{"cover"}, nil)
	f.standing["specs/c"] = standing([]string{"unit", "e2e"}, nil, nil)
	r, _ := looping(t, f, "implemente as specs pendentes")

	r.Run(context.Background())

	got := r.Snapshot()
	if got.Goal != "implemente as specs pendentes" || got.State != protocol.LoopFinished {
		t.Fatalf("run %+v", got)
	}
	if len(f.surveys) != 1 || !f.surveys[0] {
		t.Errorf("surveys %v, want one that measures", f.surveys)
	}
	if len(f.opened) != 2 || f.opened[0].LoopSpec != "specs/a" || f.opened[1].LoopSpec != "specs/c" {
		t.Errorf("opened %+v, want specs/a then specs/c", f.opened)
	}
	phases := []string{protocol.SpecUnfinished, protocol.SpecDone, protocol.SpecDone, protocol.SpecSkipped}
	for i, want := range phases {
		if got.Specs[i].Phase != want {
			t.Errorf("%s is %s, want %s", got.Specs[i].Path, got.Specs[i].Phase, want)
		}
	}
	v := got.Verdict
	if v.Worked != 2 || v.Met != 3 || v.Total != 4 || len(v.Unmet) != 1 || v.Unmet[0] != "cover" ||
		len(v.Unfinished) != 1 || v.Unfinished[0] != "specs/a" {
		t.Errorf("verdict %+v, want 2 worked, 3 of 4, cover unmet in specs/a", v)
	}
}

// A criterion nobody could check counts as left, never as met.
func TestAnUncheckedCriterionLeavesTheFolderUnfinished(t *testing.T) {
	f := newFake(protocol.SpecFolder{Path: "specs/a", Criteria: 1})
	f.standing["specs/a"] = standing(nil, nil, []string{"e2e"})
	r, _ := looping(t, f, "specs/a")
	r.Run(context.Background())
	got := r.Snapshot()
	if len(got.Specs) != 1 || got.Verdict == nil {
		t.Fatalf("run %+v, want one folder and a verdict", got)
	}
	if got.Specs[0].Phase != protocol.SpecUnfinished || got.Verdict.Unchecked[0] != "e2e" || got.Verdict.Met != 0 {
		t.Errorf("run %+v / %+v", got.Specs[0], got.Verdict)
	}
}

// A goal with nothing left opens nothing and ends with nothing worked.
func TestAGoalWithNothingPendingOpensNothing(t *testing.T) {
	f := newFake(protocol.SpecFolder{Path: "specs/a", Criteria: 1})
	r, _ := looping(t, f, "implemente tudo")
	r.Run(context.Background())
	if got := r.Snapshot(); len(f.opened) != 0 || got.State != protocol.LoopFinished || got.Verdict.Worked != 0 {
		t.Errorf("opened %+v, run %+v", f.opened, got)
	}
}

// A goal that finds no folder at all is qualified, never refused.
func TestAGoalWithNoFolderIsQualified(t *testing.T) {
	f := newFake()
	r, w := looping(t, f, "revise o projeto")
	go r.Run(context.Background())
	w.until(t, protocol.LoopAwaitingReview)
	if !f.opened[0].Qualify || !strings.Contains(f.turns[0], `"revise o projeto"`) {
		t.Errorf("opened %+v, turn %q", f.opened[0], f.turns[0])
	}
	if _, err := r.Continue(); err != nil {
		t.Fatal(err)
	}
	w.until(t, protocol.LoopFinished)
	if f.turns[1] != protocol.WorkspaceDoneDir+": revise o projeto" {
		t.Errorf("the work turn is %q, want the goal itself", f.turns[1])
	}
}

// Stopping now interrupts the turn under way and ends the run.
func TestStopNowInterruptsTheTurn(t *testing.T) {
	f := newFake(protocol.SpecFolder{Path: "specs/a", Criteria: 1})
	f.hold["specs/a"] = make(chan struct{})
	r, w := looping(t, f, "specs/a")
	go r.Run(context.Background())
	turnStarted(t, f)
	if _, err := r.Stop(protocol.StopNow); err != nil {
		t.Fatal(err)
	}
	got := w.until(t, protocol.LoopStopped)
	if len(f.interrupts) != 1 || f.interrupts[0] != "s1" || got.Error != "" {
		t.Errorf("interrupts %v, run %+v", f.interrupts, got)
	}
}

// Stopping after the folder lets it finish and opens no other.
func TestStopAfterTheFolderOpensNoOther(t *testing.T) {
	f := newFake(
		protocol.SpecFolder{Path: "specs/a", Criteria: 1, Pending: true},
		protocol.SpecFolder{Path: "specs/b", Criteria: 1, Pending: true},
	)
	release := make(chan struct{})
	f.hold["specs/a"] = release
	f.standing["specs/a"] = standing([]string{"unit"}, nil, nil)
	r, w := looping(t, f, "implemente tudo")
	go r.Run(context.Background())
	turnStarted(t, f)
	if _, err := r.Stop(protocol.StopAfterSpec); err != nil {
		t.Fatal(err)
	}
	// The folder's turn ends on its own.
	close(release)
	got := w.until(t, protocol.LoopStopped)
	if len(f.interrupts) != 0 {
		t.Errorf("stopping after the folder interrupted it: %v", f.interrupts)
	}
	if len(f.opened) != 1 || got.Specs[0].Phase != protocol.SpecDone || got.Specs[1].Phase != protocol.SpecPending {
		t.Errorf("opened %+v, specs %+v; want specs/a finished and specs/b never opened", f.opened, got.Specs)
	}
}

// What cannot go on fails, and says why.
func TestARunThatCannotGoOnSaysWhy(t *testing.T) {
	cases := []struct {
		name string
		set  func(*fakeDaemon)
		args string
		want string
	}{
		{"open", func(f *fakeDaemon) { f.openErr = errors.New("max sessions") }, "specs/a", "could not open a session"},
		{"commit", func(f *fakeDaemon) { f.commitErr = errors.New("nothing was proposed") }, "specs/x", "could not write"},
		{"goal survey", func(f *fakeDaemon) { f.surveyErr = errors.New("gone") }, "faça tudo", "could not look for specs"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFake(protocol.SpecFolder{Path: "specs/a", Criteria: 1}, protocol.SpecFolder{Path: "specs/x"})
			c.set(f)
			r, _ := looping(t, f, c.args)
			r.Run(context.Background())
			got := r.Snapshot()
			if got.State != protocol.LoopFailed || !strings.Contains(got.Error, c.want) {
				t.Errorf("state %s, error %q; want failed naming %q", got.State, got.Error, c.want)
			}
		})
	}
}

// What was typed is refused before anything exists when it cannot be read.
func TestABadArgumentIsRefusedBeforeAnythingRuns(t *testing.T) {
	for _, args := range []string{"", "   ", "specs/a --protcet x", "--protect"} {
		_, err := NewRunner("L", protocol.StartLoopRequest{Workspace: "/w", Args: args}, newFake(), nil, nil)
		if !isCode(err, protocol.CodeInvalidInput) {
			t.Errorf("%q gave %v, want invalid_input", args, err)
		}
	}
	_, err := NewRunner("L", protocol.StartLoopRequest{Workspace: "/w", Args: "specs/a --bogus"}, newFake(), nil, nil)
	if err == nil || !strings.Contains(err.Error(), "--bogus is not a flag here") {
		t.Errorf("an unknown flag gave %v", err)
	}
}

// Stop answers only "now" and "after_spec", and an ended run stays as it was.
func TestStopNamesWhen(t *testing.T) {
	f := newFake(protocol.SpecFolder{Path: "specs/a", Criteria: 1})
	r, _ := looping(t, f, "specs/a")
	if _, err := r.Stop("later"); !isCode(err, protocol.CodeInvalidInput) {
		t.Errorf("stop later gave %v", err)
	}
	r.Run(context.Background())
	got, err := r.Stop(protocol.StopNow)
	if err != nil || got.State != protocol.LoopFinished {
		t.Errorf("stopping an ended run gave %s, %v", got.State, err)
	}
}

func isCode(err error, code string) bool {
	pe, ok := protocol.AsError(err)
	return ok && pe.Code == code
}

// turnStarted waits for the next turn to start, and fails rather than hangs.
func turnStarted(t *testing.T, f *fakeDaemon) {
	t.Helper()
	select {
	case <-f.started:
	case <-time.After(5 * time.Second):
		t.Fatal("no turn started")
	}
}

// Stopping while the survey runs opens nothing: a cancelled survey is the stop,
// not a survey that failed and lets the work go ahead.
func TestStopDuringTheSurveyOpensNothing(t *testing.T) {
	f := newFake(protocol.SpecFolder{Path: "specs/a", Criteria: 1})
	f.surveyHold = true
	r, w := looping(t, f, "specs/a")
	go r.Run(context.Background())
	turnStarted(t, f) // the survey
	if _, err := r.Stop(protocol.StopNow); err != nil {
		t.Fatal(err)
	}
	got := w.until(t, protocol.LoopStopped)
	if len(f.opened) != 0 || len(f.turns) != 0 || len(got.Warnings) != 0 {
		t.Errorf("a stopped survey opened %+v, ran %q, warned %q", f.opened, f.turns, got.Warnings)
	}
}

// A stop that arrives while a session opens submits nothing to it: the run
// says stopped, and nothing is working behind it.
func TestStopWhileASessionOpensSubmitsNothing(t *testing.T) {
	f := newFake(protocol.SpecFolder{Path: "specs/a", Criteria: 1})
	f.openHold, f.openEntered = make(chan struct{}), make(chan struct{}, 1)
	r, w := looping(t, f, "specs/a")
	go r.Run(context.Background())
	select {
	case <-f.openEntered:
	case <-time.After(5 * time.Second):
		t.Fatal("no session was opened")
	}
	if _, err := r.Stop(protocol.StopNow); err != nil {
		t.Fatal(err)
	}
	close(f.openHold)
	w.until(t, protocol.LoopStopped)
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.turns) != 0 {
		t.Errorf("a turn was submitted after the stop: %q", f.turns)
	}
}

// A stop while a qualifying turn ends on its own writes no proposal and never
// says it is waiting for a review.
func TestStopWhileQualifyingWritesNoProposal(t *testing.T) {
	f := newFake(protocol.SpecFolder{Path: "specs/x"})
	f.qualifyHold = make(chan struct{})
	r, w := looping(t, f, "specs/x")
	go r.Run(context.Background())
	turnStarted(t, f)
	if _, err := r.Stop(protocol.StopNow); err != nil {
		t.Fatal(err)
	}
	close(f.qualifyHold)
	got := w.until(t, protocol.LoopStopped)
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, seen := range w.seen {
		if seen.State == protocol.LoopAwaitingReview {
			t.Error("a stopped run said it was waiting for a review")
		}
	}
	if len(f.committed) != 0 || got.Error != "" {
		t.Errorf("committed %v, error %q; want nothing written and nothing to report", f.committed, got.Error)
	}
}

// A turn that did not end normally is not a folder done: the folder keeps the
// reason, the run fails saying it, and a goal opens nothing after it.
func TestATurnThatDidNotEndNormallyFailsTheRun(t *testing.T) {
	for _, reason := range []string{protocol.StopError, protocol.StopMaxIterations, protocol.StopRepeatLoop,
		protocol.StopMaxTokens, protocol.StopInterrupted} {
		t.Run(reason, func(t *testing.T) {
			f := newFake(
				protocol.SpecFolder{Path: "specs/a", Criteria: 1, Pending: true},
				protocol.SpecFolder{Path: "specs/b", Criteria: 1, Pending: true},
			)
			f.reason["specs/a"] = reason
			r, _ := looping(t, f, "implemente tudo")
			r.Run(context.Background())
			got := r.Snapshot()
			if got.State != protocol.LoopFailed || !strings.Contains(got.Error, reason) || !strings.Contains(got.Error, "specs/a") {
				t.Errorf("state %s, error %q; want failed naming specs/a and %s", got.State, got.Error, reason)
			}
			if len(got.Specs) < 2 || got.Specs[0].Phase != protocol.SpecUnfinished || got.Specs[0].Reason != reason {
				t.Errorf("specs %+v; want specs/a unfinished with its reason", got.Specs)
			}
			if len(f.opened) != 1 {
				t.Errorf("opened %+v; a failed folder must stop the goal", f.opened)
			}
		})
	}
}

// Results are not failures: a turn that ends unverified or incomplete leaves
// the folder where its criteria say, and the run goes on.
func TestATurnThatEndsWithAResultIsNotAFailure(t *testing.T) {
	f := newFake(
		protocol.SpecFolder{Path: "specs/a", Criteria: 1, Pending: true},
		protocol.SpecFolder{Path: "specs/b", Criteria: 1, Pending: true},
	)
	f.reason["specs/a"] = protocol.StopIncomplete
	f.standing["specs/a"] = standing(nil, []string{"unit"}, nil)
	f.reason["specs/b"] = protocol.StopUnverified
	r, _ := looping(t, f, "implemente tudo")
	r.Run(context.Background())
	got := r.Snapshot()
	if got.State != protocol.LoopFinished || len(f.opened) != 2 || got.Specs[0].Phase != protocol.SpecUnfinished {
		t.Errorf("run %+v, opened %d", got, len(f.opened))
	}
}

// A goal over a backlog qualifies every pending folder that declares nothing,
// in one pass, and waits for one review of all the files before any work;
// continuing works every pending folder in order (proposal §3.3).
func TestAGoalQualifiesEveryFolderThatDeclaresNothingBeforeAnyWork(t *testing.T) {
	f := newFake(
		protocol.SpecFolder{Path: "specs/a", Criteria: 2, Unmet: 1, Pending: true},
		protocol.SpecFolder{Path: "specs/b", Criteria: 0, Pending: true},
		protocol.SpecFolder{Path: "specs/c", Criteria: 0, Pending: true},
	)
	r, w := looping(t, f, "implemente o que falta")
	go r.Run(context.Background())
	waiting := w.until(t, protocol.LoopAwaitingReview)

	f.mu.Lock()
	opened := append([]protocol.CreateSessionRequest(nil), f.opened...)
	f.mu.Unlock()
	if len(opened) != 2 || opened[0].LoopSpec != "specs/b" || !opened[0].Qualify ||
		opened[1].LoopSpec != "specs/c" || !opened[1].Qualify {
		t.Fatalf("opened %+v; want qualifying sessions on specs/b and specs/c and no work", opened)
	}
	phases := []string{protocol.SpecPending, protocol.SpecProposed, protocol.SpecProposed}
	for i, want := range phases {
		if waiting.Specs[i].Phase != want {
			t.Errorf("%s is %s, want %s", waiting.Specs[i].Path, waiting.Specs[i].Phase, want)
		}
	}

	if _, err := r.Continue(); err != nil {
		t.Fatal(err)
	}
	done := w.until(t, protocol.LoopFinished)
	f.mu.Lock()
	defer f.mu.Unlock()
	var worked []string
	for _, o := range f.opened[2:] {
		if o.Qualify {
			t.Errorf("a folder was qualified again after the review: %+v", o)
		}
		worked = append(worked, o.LoopSpec)
	}
	if strings.Join(worked, ",") != "specs/a,specs/b,specs/c" {
		t.Errorf("worked %v, want every pending folder in order", worked)
	}
	if done.Verdict.Worked != 3 {
		t.Errorf("verdict %+v, want three folders worked", done.Verdict)
	}
}
