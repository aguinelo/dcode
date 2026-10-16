package tui

import (
	"context"
	"fmt"
	goruntime "runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/loop/loopcommand"
	"github.com/aguinelo/dcode/internal/protocol"
)

// The daemon's side of `/loop`, for the TUI's tests: the REAL
// loopcommand.Runner, run over this fake's sessions, specs and commits. The
// sequence under the TUI is the daemon's own, so what these tests assert is
// what the person sees when the daemon runs the loop — never a copy of it.
//
// Nothing here waits on the clock to decide anything. A wait ends on what the
// fake daemon or the program did, and the clock only bounds it, so a hang is a
// failure that says where it stood, never a test that blocks for good.

// loopWait bounds every wait in these tests. It is a ceiling, not a guess at
// how long the work takes: nothing here finishes by its expiry.
const loopWait = 30 * time.Second

func (f *fakeTransport) loopd() *fakeLoopDaemon {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.daemon == nil {
		ctx, cancel := context.WithCancel(context.Background())
		f.daemon = &fakeLoopDaemon{runs: map[string]*loopcommand.Runner{},
			watchers: map[int]*loopWatcher{}, ctx: ctx, cancel: cancel}
	}
	return f.daemon
}

type fakeLoopDaemon struct {
	runs  map[string]*loopcommand.Runner
	order []string
	// watchers is every open loop stream. Each gets every frame, as each
	// client of the daemon does: one channel shared by every stream let a
	// stream being replaced take the frame the new one was waiting for.
	watchers map[int]*loopWatcher
	nextW    int
	// ctx is the daemon's life: the runs and the streams end with it, and
	// running is what has yet to.
	ctx     context.Context
	cancel  context.CancelFunc
	running sync.WaitGroup
}

type loopWatcher struct {
	ctx    context.Context
	frames chan protocol.LoopChange
	errs   chan error
}

// publish is the daemon telling every stream that a run moved. A stream too
// far behind to take it is a broken test, said, not a frame quietly lost.
func (f *fakeTransport) publish(run protocol.LoopRun) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, w := range f.daemon.watchers {
		select {
		case w.frames <- protocol.LoopChange{Kind: protocol.LoopChanged, Loop: &run}:
		default:
			panic(fmt.Sprintf("a loop stream fell %d frames behind: the fake daemon cannot deliver %s", cap(w.frames), run.State))
		}
	}
}

func (f *fakeTransport) StartLoop(_ context.Context, req protocol.StartLoopRequest) (protocol.LoopRun, error) {
	d := f.loopd()
	f.mu.Lock()
	id := fmt.Sprintf("L%d", len(d.order)+1)
	for _, other := range d.order {
		// One live loop per workspace, as the daemon's registry has it.
		if run := d.runs[other].Snapshot(); run.Live() && run.Workspace == req.Workspace {
			f.mu.Unlock()
			return protocol.LoopRun{}, protocol.Errorf(protocol.CodeLoopAlreadyRunning, "loop %s is already running", other)
		}
	}
	f.mu.Unlock()
	r, err := loopcommand.NewRunner(id, req, loopSide{f}, f.publish, nil)
	if err != nil {
		return protocol.LoopRun{}, err
	}
	f.mu.Lock()
	if d.ctx.Err() != nil {
		f.mu.Unlock()
		return protocol.LoopRun{}, fmt.Errorf("the fake daemon has stopped: %w", d.ctx.Err())
	}
	d.runs[id] = r
	d.order = append(d.order, id)
	d.running.Add(1)
	f.mu.Unlock()
	go func() {
		defer d.running.Done()
		r.Run(d.ctx)
	}()
	return r.Snapshot(), nil
}

func (f *fakeTransport) ListLoops(context.Context, string) ([]protocol.LoopRun, error) {
	d := f.loopd()
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []protocol.LoopRun{}
	for _, id := range d.order {
		out = append(out, d.runs[id].Snapshot())
	}
	return out, nil
}

// WatchLoops opens a stream the way the daemon does: every run as it stands
// first, then every change. A stream opened after a run moved still learns
// where it is.
func (f *fakeTransport) WatchLoops(ctx context.Context) (<-chan protocol.LoopChange, <-chan error) {
	d := f.loopd()
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.watchFrames != nil {
		return f.watchFrames, make(chan error)
	}
	if d.ctx.Err() != nil {
		// The test is over: the stream ends, saying why.
		frames, errs := make(chan protocol.LoopChange), make(chan error, 1)
		errs <- d.ctx.Err()
		close(frames)
		return frames, errs
	}
	w := &loopWatcher{ctx: ctx, frames: make(chan protocol.LoopChange, 1024), errs: make(chan error, 1)}
	snap := protocol.LoopChange{Kind: protocol.LoopSnapshot, Loops: []protocol.LoopRun{}}
	for _, id := range d.order {
		snap.Loops = append(snap.Loops, d.runs[id].Snapshot())
	}
	w.frames <- snap
	n := d.nextW
	d.nextW++
	d.watchers[n] = w
	d.running.Add(1)
	go func() {
		defer d.running.Done()
		select {
		case <-ctx.Done():
		case <-d.ctx.Done():
		}
		f.mu.Lock()
		delete(d.watchers, n)
		f.mu.Unlock()
	}()
	return w.frames, w.errs
}

// failLoopStream fails the stream of the follow on screen — that one, not a
// stream being replaced — once it is open.
func (f *fakeTransport) failLoopStream(t *testing.T, p *program, err error) {
	t.Helper()
	if p.loop == nil {
		t.Fatal("no loop is followed, so there is no stream to fail")
	}
	d, ctx := f.loopd(), p.loop.ctx
	deadline := time.Now().Add(loopWait)
	for time.Now().Before(deadline) {
		f.mu.Lock()
		for _, w := range d.watchers {
			if w.ctx == ctx {
				w.errs <- err
				f.mu.Unlock()
				return
			}
		}
		f.mu.Unlock()
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("the follow on screen never opened its loop stream in %s", loopWait)
}

func (f *fakeTransport) runner(id string) *loopcommand.Runner {
	d := f.loopd()
	f.mu.Lock()
	defer f.mu.Unlock()
	return d.runs[id]
}

func (f *fakeTransport) ContinueLoop(_ context.Context, id string) (protocol.LoopRun, error) {
	return f.runner(id).Continue()
}

func (f *fakeTransport) StopLoop(_ context.Context, id, when string) (protocol.LoopRun, error) {
	return f.runner(id).Stop(when)
}

// lastLoop is the last run the fake daemon started, as it stands.
func (f *fakeTransport) lastLoop(t *testing.T) protocol.LoopRun {
	t.Helper()
	d := f.loopd()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(d.order) == 0 {
		return protocol.LoopRun{}
	}
	return d.runs[d.order[len(d.order)-1]].Snapshot()
}

// loopSide is the daemon as the runner sees it, over this fake.
type loopSide struct{ f *fakeTransport }

func (l loopSide) ListSpecs(context.Context, string, bool) ([]protocol.SpecFolder, error) {
	l.f.mu.Lock()
	defer l.f.mu.Unlock()
	return l.f.specs, l.f.specsErr
}

func (l loopSide) Open(_ context.Context, req protocol.CreateSessionRequest) (protocol.Session, error) {
	f := l.f
	f.mu.Lock()
	defer f.mu.Unlock()
	f.created = append(f.created, req)
	if f.createErr != nil {
		return protocol.Session{}, f.createErr
	}
	criteria := 0
	for _, s := range f.specs {
		if s.Path == req.LoopSpec {
			criteria = s.Criteria
		}
	}
	s := protocol.Session{
		ID: fmt.Sprintf("loop-s%d", len(f.created)), Workspace: req.Workspace, Model: req.Model,
		SandboxMode: req.SandboxMode, State: protocol.SessionStateIdle, DoneCriteria: criteria,
	}
	if req.Qualify {
		s.DoneCriteria = 0
	}
	f.sessions = append(f.sessions, s)
	return s, nil
}

func (l loopSide) RunTurn(ctx context.Context, id, text string) (protocol.TurnCompleted, error) {
	f := l.f
	f.mu.Lock()
	f.submitted = append(f.submitted, text)
	var n int
	fmt.Sscanf(id, "loop-s%d", &n)
	req := f.created[n-1]
	hold, done := f.holdTurns, f.completions[req.LoopSpec]
	f.mu.Unlock()
	if req.Qualify {
		return protocol.TurnCompleted{Reason: protocol.StopDone}, nil
	}
	if hold != nil {
		select {
		case <-hold:
		case <-ctx.Done():
			return protocol.TurnCompleted{}, ctx.Err()
		}
	}
	return protocol.TurnCompleted{Reason: protocol.StopDone, Completion: done}, nil
}

func (l loopSide) CommitDone(_ context.Context, id string) (protocol.CommitDoneResponse, error) {
	f := l.f
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.committed, f.commitErr
}

func (l loopSide) Interrupt(string) {
	f := l.f
	f.mu.Lock()
	defer f.mu.Unlock()
	f.interrupts++
}

// sawEntry reports an entry carrying every one of the strings.
func sawEntry(p *program, want ...string) bool {
	for _, e := range p.model.Entries {
		ok := true
		for _, w := range want {
			if !strings.Contains(e.Summary, w) {
				ok = false
			}
		}
		if ok {
			return true
		}
	}
	return false
}

func summaries(p *program) []string {
	out := make([]string, 0, len(p.model.Entries))
	for _, e := range p.model.Entries {
		out = append(out, e.Summary)
	}
	return out
}

// loopProgram is a program on a fake daemon, in a workspace, idle.
//
// When the test ends, everything it set going ends with it — the program's
// commands and follows, the daemon's runs and streams — or the test fails
// naming what was left. A goroutine left behind is the next test's flake.
func loopProgram(t *testing.T, specs ...protocol.SpecFolder) (*program, *fakeTransport) {
	t.Helper()
	if _, checked := followChecked.LoadOrStore(t, true); !checked {
		// Once per test, and first, so it runs after every program the test
		// made has been ended.
		t.Cleanup(func() {
			followsEnded(t)
			followChecked.Delete(t)
		})
	}
	p, tr := newProgram(t)
	tr.specs = specs
	d, rt := tr.loopd(), runtimeOf(p)
	t.Cleanup(func() {
		p.cancel()
		// Under the lock, so nothing starts a run or a stream once it is
		// over: what is running is all the wait below has to see end.
		tr.mu.Lock()
		d.cancel()
		tr.mu.Unlock()
		ended("the fake daemon's runs and streams", &d.running, t)
		ended("the program's commands", &rt.all, t)
		runtimesMu.Lock()
		delete(runtimes, p)
		runtimesMu.Unlock()
	})
	return p, tr
}

func ended(what string, wg *sync.WaitGroup, t *testing.T) {
	t.Helper()
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(loopWait):
		t.Errorf("%s were still running %s after the test ended", what, loopWait)
	}
}

var followChecked sync.Map

// followsEnded fails the test if a follow it started is still reading the
// loop stream. The follow is the program's own goroutine, so it is looked for
// where it runs; the tests in this package do not run in parallel, so any
// follow there is this test's.
func followsEnded(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(loopWait)
	buf := make([]byte, 1<<20)
	for {
		stacks := buf[:goruntime.Stack(buf, true)]
		if !strings.Contains(string(stacks), "tui.(*program).followLoop(") {
			return
		}
		if time.Now().After(deadline) {
			t.Errorf("a loop follow was still running %s after the test ended", loopWait)
			return
		}
		time.Sleep(time.Millisecond)
	}
}
