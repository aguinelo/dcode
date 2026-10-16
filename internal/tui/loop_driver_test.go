package tui

import (
	"context"
	"reflect"
	goruntime "runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// How the loop tests drive a program: its commands run as the real runtime
// runs them, and a wait ends on what the program did — never on the clock,
// which only bounds the wait (loopWait), so a hang fails saying where it was.

// cmdRuntime runs the commands Update returns, as the program would: each in
// its own goroutine, their messages back through Update in arrival order. The
// loop is followed through waitForLoop, the real path, never by reading the
// follow's channel. One per program.
//
// It knows when the program is at rest: every command that does work has
// delivered what it returned, and what is left only waits — on the session's
// events, on the loop's stream, on the clock. That is how a test knows
// nothing more is coming, rather than inferring it from a quiet spell.
type cmdRuntime struct {
	msgs chan tea.Msg
	all  sync.WaitGroup
	mu   sync.Mutex
	// working is every command doing work that has not delivered yet, by
	// name, and events the session events the program has applied.
	working map[string]int
	events  map[uint64]bool
}

var (
	runtimesMu sync.Mutex
	runtimes   = map[*program]*cmdRuntime{}
)

func runtimeOf(p *program) *cmdRuntime {
	runtimesMu.Lock()
	defer runtimesMu.Unlock()
	rt, ok := runtimes[p]
	if !ok {
		rt = &cmdRuntime{msgs: make(chan tea.Msg, 1024), working: map[string]int{}, events: map[uint64]bool{}}
		runtimes[p] = rt
	}
	return rt
}

// waiters are the commands that wait for something from outside instead of
// doing work: they are not what a test waits for.
var waiters = []string{".waitForLoop.", ".waitForEvent.", ".Tick."}

func cmdName(cmd tea.Cmd) string {
	if fn := goruntime.FuncForPC(reflect.ValueOf(cmd).Pointer()); fn != nil {
		return fn.Name()
	}
	return "an unnamed command"
}

func (rt *cmdRuntime) spawn(p *program, cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	name, works := cmdName(cmd), true
	for _, w := range waiters {
		works = works && !strings.Contains(name, w)
	}
	if works {
		rt.mu.Lock()
		rt.working[name]++
		rt.mu.Unlock()
	}
	rt.all.Add(1)
	go func() {
		defer rt.all.Done()
		if works {
			// After the message is in msgs, or its children are spawned: at
			// no point is the work in neither place.
			defer func() {
				rt.mu.Lock()
				if rt.working[name]--; rt.working[name] == 0 {
					delete(rt.working, name)
				}
				rt.mu.Unlock()
			}()
		}
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

// settled is the program at rest. Work is read before the queue: work that
// finished has already put its message there, and only this goroutine takes
// messages out.
func (rt *cmdRuntime) settled() bool {
	rt.mu.Lock()
	idle := len(rt.working) == 0
	rt.mu.Unlock()
	return idle && len(rt.msgs) == 0
}

func (rt *cmdRuntime) inFlight() []string {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	out := make([]string, 0, len(rt.working))
	for name := range rt.working {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
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
// follows no loop and is at rest: the run it followed ended, and everything
// the end set off has been done.
//
// The daemon's state moves without a message when nothing follows it, so done
// is asked again every few milliseconds as well as after every message. That
// asks sooner; it decides nothing.
func followLoop(t *testing.T, p *program, done func() bool) {
	t.Helper()
	if done == nil {
		rt := runtimeOf(p)
		done = func() bool { return p.loop == nil && rt.settled() }
	}
	drive(t, p, "the loop never got there", done)
}

// settle runs the program until it is at rest, whatever it is doing.
func settle(t *testing.T, p *program) {
	t.Helper()
	rt := runtimeOf(p)
	drive(t, p, "the program never came to rest", rt.settled)
}

// applied reports the program having applied the session event with that
// sequence, on the stream it is attached to.
func applied(p *program, seq uint64) bool {
	rt := runtimeOf(p)
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.events[seq]
}

func drive(t *testing.T, p *program, failure string, done func() bool) {
	t.Helper()
	rt := runtimeOf(p)
	ctx, cancel := context.WithTimeout(t.Context(), loopWait)
	defer cancel()
	ask := time.NewTicker(2 * time.Millisecond)
	defer ask.Stop()
	for !done() {
		select {
		case m := <-rt.msgs:
			if em, ok := m.(eventMsg); ok && p.fromCurrentStream(em.gen) {
				rt.mu.Lock()
				rt.events[em.ev.Seq] = true
				rt.mu.Unlock()
			}
			_, cmd := p.Update(m)
			rt.spawn(p, cmd)
		case <-ask.C:
		case <-ctx.Done():
			t.Fatalf("%s in %s; still working: %v; entries: %q", failure, loopWait, rt.inFlight(), summaries(p))
		}
	}
}
