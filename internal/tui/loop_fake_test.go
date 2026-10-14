package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/aguinelo/dcode/internal/loop/loopcommand"
	"github.com/aguinelo/dcode/internal/protocol"
)

// The daemon's side of `/loop`, for the TUI's tests: the REAL
// loopcommand.Runner, run over this fake's sessions, specs and commits. The
// sequence under the TUI is the daemon's own, so what these tests assert is
// what the person sees when the daemon runs the loop — never a copy of it.

func (f *fakeTransport) loopd() *fakeLoopDaemon {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.daemon == nil {
		f.daemon = &fakeLoopDaemon{frames: make(chan protocol.LoopChange, 1024),
			runs: map[string]*loopcommand.Runner{}}
	}
	return f.daemon
}

type fakeLoopDaemon struct {
	frames chan protocol.LoopChange
	runs   map[string]*loopcommand.Runner
	order  []string
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
	r, err := loopcommand.NewRunner(id, req, loopSide{f}, func(run protocol.LoopRun) {
		d.frames <- protocol.LoopChange{Kind: protocol.LoopChanged, Loop: &run}
	}, nil)
	if err != nil {
		return protocol.LoopRun{}, err
	}
	f.mu.Lock()
	d.runs[id] = r
	d.order = append(d.order, id)
	f.mu.Unlock()
	go r.Run(context.Background())
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

func (f *fakeTransport) WatchLoops(context.Context) (<-chan protocol.LoopChange, <-chan error) {
	f.mu.Lock()
	errs, frames := f.watchErrs, f.watchFrames
	f.mu.Unlock()
	if errs == nil {
		errs = make(chan error)
	}
	if frames != nil {
		return frames, errs
	}
	return f.loopd().frames, errs
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

// runtime runs the commands Update returns, as the program would: each in its
// own goroutine, their messages back through Update in arrival order. The
// loop is followed through waitForLoop, the real path, never by reading the
// follow's channel. One per program.
type runtime struct {
	msgs chan tea.Msg
}

var (
	runtimesMu sync.Mutex
	runtimes   = map[*program]*runtime{}
)

func runtimeOf(p *program) *runtime {
	runtimesMu.Lock()
	defer runtimesMu.Unlock()
	rt, ok := runtimes[p]
	if !ok {
		rt = &runtime{msgs: make(chan tea.Msg, 1024)}
		runtimes[p] = rt
	}
	return rt
}

func (rt *runtime) spawn(p *program, cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	go func() {
		m := cmd()
		if batch, ok := m.(tea.BatchMsg); ok {
			for _, c := range batch {
				rt.spawn(p, c)
			}
			return
		}
		if m == nil {
			return
		}
		select {
		case rt.msgs <- m:
		case <-p.ctx.Done():
		}
	}()
}

// typeLoop is the person typing `/loop <args>`.
func typeLoop(t *testing.T, p *program, args string) {
	t.Helper()
	_, cmd := p.runBuiltin(Resolved{Kind: CmdBuiltin, Name: "loop", Args: args})
	runtimeOf(p).spawn(p, cmd)
}

// deliver runs one command's messages through the program, as the runtime
// would, without waiting for them to arrive.
func deliver(p *program, cmd tea.Cmd) {
	runtimeOf(p).spawn(p, cmd)
}

// followLoop runs the program until done says so or, with done nil, until it
// no longer follows a loop and nothing more is arriving.
func followLoop(t *testing.T, p *program, done func() bool) {
	t.Helper()
	rt := runtimeOf(p)
	deadline := time.After(5 * time.Second)
	seen := p.loop != nil
	for {
		if done != nil && done() {
			return
		}
		select {
		case m := <-rt.msgs:
			_, cmd := p.Update(m)
			rt.spawn(p, cmd)
			seen = seen || p.loop != nil
		case <-time.After(150 * time.Millisecond):
			if done == nil && seen && p.loop == nil {
				return
			}
		case <-deadline:
			t.Fatalf("the loop never got there; entries: %q", summaries(p))
		}
	}
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
func loopProgram(t *testing.T, specs ...protocol.SpecFolder) (*program, *fakeTransport) {
	t.Helper()
	p, tr := newProgram(t)
	tr.specs = specs
	return p, tr
}
