package loopcommand

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// Transport is what a Runner needs of the daemon that hosts it.
//
// Declared here, by the one package that consumes it: the runner decides the
// sequence, the daemon owns sessions and the disk. A fake of this is how the
// sequence is tested without a model — it is deterministic.
type Transport interface {
	// ListSpecs says what each spec folder declares, and with measure where
	// it stands.
	ListSpecs(ctx context.Context, workspace string, measure bool) ([]protocol.SpecFolder, error)
	// Open creates a session that belongs to this run.
	Open(ctx context.Context, req protocol.CreateSessionRequest) (protocol.Session, error)
	// RunTurn submits text as a turn and blocks until that turn completes.
	RunTurn(ctx context.Context, sessionID, text string) (protocol.TurnCompleted, error)
	// CommitDone measures and writes what a qualifying session proposed.
	CommitDone(ctx context.Context, sessionID string) (protocol.CommitDoneResponse, error)
	// Interrupt stops the turn running in a session, if any.
	Interrupt(sessionID string)
}

// Runner conducts one `/loop`: the sequence the TUI ran in its own process,
// moved into the daemon so every client sees the same run.
//
// The sequence is the TUI's, unchanged: a folder is surveyed before anything
// opens, one that declares nothing is qualified first, a written proposal
// stops the run for a person, a goal works its pending folders one session
// each and in order, and the end is said with a verdict about the whole run.
type Runner struct {
	t       Transport
	req     protocol.StartLoopRequest
	args    LoopArgs
	publish func(protocol.LoopRun)
	now     func() time.Time

	mu sync.Mutex
	// run is the state every client sees; changed only through change.
	run protocol.LoopRun
	// worked counts the folders whose work started, as the TUI counted them.
	worked int
	// stopping is a stop that has been asked for: now, or after the folder.
	stopping string
	cancel   context.CancelFunc
	// resume carries the answer to a written proposal: true to work it,
	// false to stop. Buffered, so answering never waits on the runner.
	resume chan bool
}

// NewRunner parses what was typed and builds a run that has not started.
//
// A bad argument is refused here, before anything exists, with the sentence
// the TUI already gives.
func NewRunner(id string, req protocol.StartLoopRequest, t Transport, publish func(protocol.LoopRun), now func() time.Time) (*Runner, error) {
	args, err := ParseLoopArgs(req.Args)
	if err != nil {
		return nil, protocol.Errorf(protocol.CodeInvalidInput, "%v", err)
	}
	if now == nil {
		now = time.Now
	}
	if publish == nil {
		publish = func(protocol.LoopRun) {}
	}
	run := protocol.LoopRun{
		ID: id, Workspace: req.Workspace, Protect: args.Protect,
		State: protocol.LoopSurveying, Specs: []protocol.LoopSpecRun{}, Current: -1,
		Started: now(),
	}
	if args.Goal {
		run.Goal = args.Task
	}
	return &Runner{t: t, req: req, args: args, publish: publish, now: now, run: run, resume: make(chan bool, 1)}, nil
}

// Snapshot is the run as it stands.
func (r *Runner) Snapshot() protocol.LoopRun {
	r.mu.Lock()
	defer r.mu.Unlock()
	return copyRun(r.run)
}

// Run conducts the loop until it ends. The daemon calls it once, in its own
// goroutine, with a context that lives as long as the daemon — not the client
// that asked.
func (r *Runner) Run(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	r.mu.Lock()
	r.cancel = cancel
	stopped := r.stopping == protocol.StopNow
	r.mu.Unlock()

	var err error
	switch {
	case stopped:
	case r.args.Goal:
		err = r.goal(ctx)
	default:
		err = r.one(ctx)
	}
	r.end(err)
}

// Continue takes a run out of awaiting_review and into the work.
//
// The first answer wins: the state moves before anything else happens, so a
// second client pressing the same button is refused rather than starting the
// work twice.
func (r *Runner) Continue() (protocol.LoopRun, error) {
	r.mu.Lock()
	if r.run.State != protocol.LoopAwaitingReview || r.stopping != "" {
		state := r.run.State
		r.mu.Unlock()
		return r.Snapshot(), protocol.Errorf(protocol.CodeLoopNotAwaitingReview,
			"loop %s is %s, not waiting for a review", r.run.ID, state)
	}
	r.run.State = protocol.LoopWorking
	snap := copyRun(r.run)
	r.mu.Unlock()
	r.publish(snap)
	r.resume <- true
	return snap, nil
}

// Stop ends the run now, interrupting the turn under way, or after the folder
// it is in. Stopping a run that already ended answers where it ended.
func (r *Runner) Stop(when string) (protocol.LoopRun, error) {
	if when != protocol.StopNow && when != protocol.StopAfterSpec {
		return r.Snapshot(), protocol.Errorf(protocol.CodeInvalidInput,
			"when must be %q or %q, got %q", protocol.StopNow, protocol.StopAfterSpec, when)
	}
	r.mu.Lock()
	if !r.run.Live() {
		r.mu.Unlock()
		return r.Snapshot(), nil
	}
	if r.stopping != protocol.StopNow {
		r.stopping = when
	}
	waiting := r.run.State == protocol.LoopAwaitingReview
	session, cancel := r.run.SessionID, r.cancel
	r.mu.Unlock()

	switch {
	case waiting:
		// Nothing is running: either way, the run ends here.
		select {
		case r.resume <- false:
		default:
		}
	case when == protocol.StopNow:
		// Cancel first: whatever the runner does next sees the stop, so a turn
		// that ends on its own right now is not followed by a commit or by the
		// next folder.
		if cancel != nil {
			cancel()
		}
		if session != "" {
			r.t.Interrupt(session)
		}
	}
	return r.Snapshot(), nil
}

// one is `/loop <folder>`: survey, then qualify or work.
func (r *Runner) one(ctx context.Context) error {
	a := r.args
	found, err := r.t.ListSpecs(ctx, r.req.Workspace, false)
	if ctx.Err() != nil {
		// Cancelled is the stop, not a survey that failed.
		return ctx.Err()
	}
	if err != nil {
		r.change(func(run *protocol.LoopRun) {
			run.Warnings = append(run.Warnings,
				fmt.Sprintf("could not ask what %s declares (%v); the session opened anyway", a.Spec, err))
		})
	}
	if err == nil {
		// Not fatal when it fails: if the daemon cannot say, the session opens
		// and answers. Refusing the work because the survey failed would be the
		// survey holding the work hostage.
		if q, ok := ArgumentToQualify(a, found); ok {
			return r.qualify(ctx, q, LoopArgs{Spec: q.Spec, Task: q.Task, Protect: q.Protect})
		}
		for _, f := range found {
			if f.Path != a.Spec {
				continue
			}
			if f.Criteria == 0 && f.Error == "" {
				q := a
				q.Qualify, q.Task = true, ""
				return r.qualify(ctx, q, a)
			}
			break
		}
	}
	i := r.addSpec(a.Spec, protocol.SpecPending)
	return r.work(ctx, i, a)
}

// goal is `/loop <sentence>`: every folder measured, the pending ones worked
// one session each, in the order they sort.
func (r *Runner) goal(ctx context.Context) error {
	a := r.args
	found, err := r.t.ListSpecs(ctx, r.req.Workspace, true)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		return fmt.Errorf("could not look for specs: %w", err)
	}
	if q, ok := GoalToQualify(a, found); ok {
		return r.qualify(ctx, q, LoopArgs{Spec: q.Spec, Task: a.Task, Protect: a.Protect})
	}
	var pending, empty []int
	for _, f := range found {
		phase := protocol.SpecDone
		switch {
		case f.Error != "":
			phase = protocol.SpecSkipped
		case f.Pending:
			phase = protocol.SpecPending
		}
		i := r.addSpec(f.Path, phase)
		if f.Pending {
			pending = append(pending, i)
		}
		if f.Pending && f.Criteria == 0 && f.Error == "" {
			empty = append(empty, i)
		}
	}
	// Every pending folder that declares nothing is qualified first, in one
	// pass, and the run stops with the files written: one sitting to review
	// them all, before any work runs against a ruler nobody read.
	if len(empty) > 0 {
		for _, i := range empty {
			if r.stopAsked() {
				return nil
			}
			path := r.Snapshot().Specs[i].Path
			if err := r.propose(ctx, i, LoopArgs{Spec: path, Protect: a.Protect, Qualify: true}); err != nil {
				return err
			}
		}
		if goOn, err := r.review(ctx); !goOn || err != nil {
			return err
		}
	}
	for _, i := range pending {
		if r.stopAsked() {
			return nil
		}
		path := r.Snapshot().Specs[i].Path
		if err := r.work(ctx, i, LoopArgs{Spec: path, Task: a.Task, Protect: a.Protect}); err != nil {
			return err
		}
	}
	return nil
}

// qualify runs the qualifying session for a folder, writes its proposal, and
// waits for a person. work is what runs once they say so.
func (r *Runner) qualify(ctx context.Context, q, work LoopArgs) error {
	i := r.addSpec(q.Spec, protocol.SpecQualifying)
	if err := r.propose(ctx, i, q); err != nil {
		return err
	}
	if goOn, err := r.review(ctx); !goOn || err != nil {
		return err
	}
	return r.work(ctx, i, work)
}

// propose runs the qualifying session for folder i and writes its proposal.
func (r *Runner) propose(ctx context.Context, i int, q LoopArgs) error {
	r.change(func(run *protocol.LoopRun) {
		run.Specs[i].Phase = protocol.SpecQualifying
		run.State, run.Current = protocol.LoopQualifying, i
	})
	id, err := r.open(ctx, i, q)
	if err != nil {
		return err
	}
	done, err := r.t.RunTurn(ctx, id, LoopTask(q))
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		// Stopped while the turn ended on its own: nothing is written.
		return ctx.Err()
	}
	if !endedNormally(done.Reason) {
		r.change(func(run *protocol.LoopRun) {
			run.Specs[i].Phase, run.Specs[i].Reason = protocol.SpecUnfinished, done.Reason
		})
		return fmt.Errorf("the qualifying turn on %s ended %s", q.Spec, done.Reason)
	}
	out, err := r.t.CommitDone(ctx, id)
	if err != nil {
		return fmt.Errorf("could not write the proposed definition of done: %w", err)
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	r.change(func(run *protocol.LoopRun) {
		run.Specs[i].Phase, run.Specs[i].Proposal = protocol.SpecProposed, out.Path
	})
	return nil
}

// review stops the run for a person, and says whether to go on to the work.
//
// A proposal nobody looked at is a ruler nobody read; the independence is that
// qualifying happened by itself, not that the signature was skipped.
func (r *Runner) review(ctx context.Context) (bool, error) {
	if r.stopAsked() {
		return false, nil
	}
	r.change(func(run *protocol.LoopRun) {
		run.State, run.Current = protocol.LoopAwaitingReview, -1
	})
	select {
	case goOn := <-r.resume:
		return goOn, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

// work runs one folder's work session and keeps where it was left.
func (r *Runner) work(ctx context.Context, i int, a LoopArgs) error {
	r.mu.Lock()
	r.worked++
	r.mu.Unlock()
	r.change(func(run *protocol.LoopRun) {
		run.Specs[i].Phase = protocol.SpecWorking
		run.State, run.Current = protocol.LoopWorking, i
	})
	id, err := r.open(ctx, i, a)
	if err != nil {
		return err
	}
	done, err := r.t.RunTurn(ctx, id, LoopTask(a))
	if err != nil {
		return err
	}
	normal := endedNormally(done.Reason)
	r.change(func(run *protocol.LoopRun) {
		// The last word for a folder wins: what matters is where it was left.
		run.Specs[i].Standing, run.Specs[i].Reason = done.Completion, done.Reason
		run.Specs[i].Phase = protocol.SpecDone
		if !normal || unfinished(done.Completion) {
			run.Specs[i].Phase = protocol.SpecUnfinished
		}
	})
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if !normal {
		// Not a folder done, and not one to walk past: the run stops here with
		// the reason in sight rather than opening the next folder on top of it.
		return fmt.Errorf("the turn on %s ended %s", a.Spec, done.Reason)
	}
	return nil
}

// endedNormally reports a turn that ended as a result rather than a failure.
// Unverified and incomplete are results: the criteria say where things stand.
func endedNormally(reason string) bool {
	switch reason {
	case protocol.StopDone, protocol.StopUnverified, protocol.StopIncomplete:
		return true
	}
	return false
}

func (r *Runner) open(ctx context.Context, i int, a LoopArgs) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s, err := r.t.Open(ctx, protocol.CreateSessionRequest{
		Workspace: r.req.Workspace, Model: r.req.Model, SandboxMode: r.req.SandboxMode,
		LoopSpec: a.Spec, Protect: a.Protect, Qualify: a.Qualify,
	})
	if err != nil {
		return "", fmt.Errorf("could not open a session: %w", err)
	}
	r.change(func(run *protocol.LoopRun) {
		run.Specs[i].SessionID, run.SessionID = s.ID, s.ID
	})
	// Opening does not watch the context: a stop that arrived meanwhile is
	// honoured here, before any turn is submitted.
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return s.ID, nil
}

// end says how the run ended, with the verdict of the whole run.
func (r *Runner) end(err error) {
	r.change(func(run *protocol.LoopRun) {
		switch {
		case r.stopping != "":
			run.State = protocol.LoopStopped
		case err != nil:
			run.State, run.Error = protocol.LoopFailed, err.Error()
		default:
			run.State = protocol.LoopFinished
		}
		if run.State == protocol.LoopStopped && err != nil && !errors.Is(err, context.Canceled) {
			// Stopping does not hide a failure that happened on its own.
			run.Error = err.Error()
		}
		run.Current = -1
		run.Verdict = verdict(r.worked, run.Specs)
		run.Finished = r.now()
	})
}

func (r *Runner) addSpec(path, phase string) int {
	var i int
	r.change(func(run *protocol.LoopRun) {
		run.Specs = append(run.Specs, protocol.LoopSpecRun{Path: path, Phase: phase})
		i = len(run.Specs) - 1
	})
	return i
}

func (r *Runner) stopAsked() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stopping != ""
}

// change applies f under the lock and announces the result outside it.
func (r *Runner) change(f func(*protocol.LoopRun)) {
	r.mu.Lock()
	f(&r.run)
	snap := copyRun(r.run)
	r.mu.Unlock()
	r.publish(snap)
}

// verdict folds the folders worked into the numbers and names a run reports.
//
// Across every folder, not the last one: criteria are summed because each
// folder carries its own set, names are kept once because the same criterion
// failing twice is one thing to fix, and folders are named because that says
// where to go back to.
func verdict(worked int, specs []protocol.LoopSpecRun) *protocol.LoopVerdict {
	v := &protocol.LoopVerdict{Worked: worked}
	seenUnmet, seenUnchecked := map[string]bool{}, map[string]bool{}
	add := func(dst *[]string, seen map[string]bool, names []string) {
		for _, n := range names {
			if !seen[n] {
				seen[n] = true
				*dst = append(*dst, n)
			}
		}
	}
	for _, s := range specs {
		c := s.Standing
		if c == nil {
			continue
		}
		v.Met += len(c.Met)
		v.Total += len(c.Met) + len(c.Unmet) + len(c.Unavailable)
		add(&v.Unmet, seenUnmet, c.Unmet)
		add(&v.Unchecked, seenUnchecked, c.Unavailable)
		if unfinished(c) {
			v.Unfinished = append(v.Unfinished, s.Path)
		}
	}
	return v
}

// unfinished reports a completion with anything outstanding. Unavailable
// counts: a criterion nobody could check is not one that passed.
func unfinished(c *protocol.Completion) bool {
	return c != nil && (len(c.Unmet) > 0 || len(c.Unavailable) > 0)
}

func copyRun(r protocol.LoopRun) protocol.LoopRun {
	r.Specs = append(make([]protocol.LoopSpecRun, 0, len(r.Specs)), r.Specs...)
	r.Warnings = append([]string(nil), r.Warnings...)
	if r.Verdict != nil {
		v := *r.Verdict
		r.Verdict = &v
	}
	return r
}
