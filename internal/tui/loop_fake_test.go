package tui

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

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
	return f.loopd().frames, make(chan error)
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
		t.Fatal("no loop was started")
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

// typeLoop is the person typing `/loop <args>`: the command, and what it
// answered — the run started, or a note.
func typeLoop(t *testing.T, p *program, args string) {
	t.Helper()
	_, cmd := p.runBuiltin(Resolved{Kind: CmdBuiltin, Name: "loop", Args: args})
	if cmd == nil {
		return
	}
	p.Update(cmd())
}

// followLoop feeds the program every message about the run it follows, as
// the update loop would, until done says so or the run is no longer followed.
func followLoop(t *testing.T, p *program, done func() bool) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for done == nil || !done() {
		f := p.loop
		if f == nil {
			return
		}
		select {
		case m := <-f.msgs:
			p.Update(m)
		case <-deadline:
			t.Fatalf("the loop never got there; entries: %+v", summaries(p))
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
