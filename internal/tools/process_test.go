package tools

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeHandle is a background command under the test's control.
type fakeHandle struct {
	mu      sync.Mutex
	out     string
	code    int
	done    bool
	stopped bool
}

func (h *fakeHandle) Output() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.out
}

func (h *fakeHandle) Exited() (int, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.code, h.done
}

func (h *fakeHandle) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.stopped, h.done = true, true
}

func (h *fakeHandle) write(s string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.out += s
}

func (h *fakeHandle) exit(code int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.code, h.done = code, true
}

func (h *fakeHandle) wasStopped() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.stopped
}

// fakeBackground hands out handles, newest last.
//
// Locked because a test that watches a process while it starts is running two
// goroutines over this by definition — which is the shape of the real thing,
// and the reason the first version of this file raced.
type fakeBackground struct {
	mu      sync.Mutex
	seed    []*fakeHandle
	handles []*fakeHandle
	err     error
	started []string
}

func (b *fakeBackground) Start(_ context.Context, _, command string) (Handle, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.err != nil {
		return nil, b.err
	}
	b.started = append(b.started, command)
	var h *fakeHandle
	if len(b.seed) > 0 {
		h, b.seed = b.seed[0], b.seed[1:]
	} else {
		h = &fakeHandle{}
	}
	b.handles = append(b.handles, h)
	return h, nil
}

// handle is the nth handle handed out.
func (b *fakeBackground) handle(n int) *fakeHandle {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.handles[n]
}

// commands is what reached the runner.
func (b *fakeBackground) commands() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.started...)
}

// backgroundBash is a Bash wired to run in the background, with a settle window
// short enough that a test does not wait on a real one.
func backgroundBash(b *fakeBackground) Bash {
	return Bash{Background: b, Workdir: ".", Settle: 20 * time.Millisecond}
}

func runBash(t *testing.T, b Bash, s *State, input string) Result {
	t.Helper()
	res, err := b.Execute(context.Background(), json.RawMessage(input), s)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func runProcess(t *testing.T, s *State, input string) Result {
	t.Helper()
	res, err := Process{}.Execute(context.Background(), json.RawMessage(input), s)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// A server does not terminate — that is what makes it a server — so waiting for
// it is waiting forever, and the ceiling that saves the session from that is
// also what makes starting one impossible. The background call returns while
// the command is still running.
func TestABackgroundCommandReturnsWhileItIsStillRunning(t *testing.T) {
	s, _ := setup(t)
	bg := &fakeBackground{}

	res := runBash(t, backgroundBash(bg), s, `{"command":"npm start","background":true}`)

	if res.IsError {
		t.Fatalf("starting in the background failed: %s", res.Output)
	}
	if got := bg.commands(); len(got) != 1 || got[0] != "npm start" {
		t.Fatalf("the command did not reach the runner: %v", got)
	}
	if !strings.Contains(res.Output, "bg1") {
		t.Errorf("no identifier came back, so nothing can be read later: %q", res.Output)
	}
	if !strings.Contains(res.Output, "still running") {
		t.Errorf("the model cannot tell it is up: %q", res.Output)
	}
}

// "Did it come up" is a different question from "did it finish well", and the
// common failure is crashing during startup. The settle window answers that one
// without inventing a readiness protocol for a port nobody named.
func TestABackgroundCommandThatDiesDuringStartupSaysSo(t *testing.T) {
	s, _ := setup(t)
	dying := &fakeHandle{}
	bg := &fakeBackground{seed: []*fakeHandle{dying}}
	b := backgroundBash(bg)

	go func() {
		time.Sleep(2 * time.Millisecond)
		dying.write("Error: port 3000 is already in use\n")
		dying.exit(1)
	}()

	res := runBash(t, b, s, `{"command":"npm start","background":true}`)

	if !strings.Contains(res.Output, "exit 1") {
		t.Errorf("a startup crash was reported as a healthy start: %q", res.Output)
	}
	if !strings.Contains(res.Output, "port 3000") {
		t.Errorf("the reason it died was dropped: %q", res.Output)
	}
}

// The value of starting a server is reading the log when it breaks. Output
// nobody can collect is a server that started with nobody able to say whether
// it works.
func TestProcessReadsTheOutputOfARunningCommand(t *testing.T) {
	s, _ := setup(t)
	bg := &fakeBackground{}
	runBash(t, backgroundBash(bg), s, `{"command":"npm start","background":true}`)
	bg.handle(0).write("listening on http://localhost:3000\n")

	res := runProcess(t, s, `{"id":"bg1"}`)

	if res.IsError {
		t.Fatalf("reading a live process failed: %s", res.Output)
	}
	if !strings.Contains(res.Output, "localhost:3000") {
		t.Errorf("the output never reached the model: %q", res.Output)
	}
	if !strings.Contains(res.Output, "running") {
		t.Errorf("the state is missing, so the model cannot tell live from finished: %q", res.Output)
	}
}

// Without a listing, a model that lost the identifier has no way back to a
// process it started, and the only remaining move is starting a second one.
func TestProcessWithNoIdentifierListsWhatIsRunning(t *testing.T) {
	s, _ := setup(t)
	bg := &fakeBackground{}
	b := backgroundBash(bg)
	runBash(t, b, s, `{"command":"npm start","background":true}`)
	runBash(t, b, s, `{"command":"go run ./cmd/api","background":true}`)

	res := runProcess(t, s, `{}`)

	for _, want := range []string{"bg1", "bg2", "npm start", "go run ./cmd/api"} {
		if !strings.Contains(res.Output, want) {
			t.Errorf("the listing omits %q: %q", want, res.Output)
		}
	}
}

func TestProcessSaysSoWhenNothingIsRunning(t *testing.T) {
	s, _ := setup(t)

	res := runProcess(t, s, `{}`)

	if res.IsError {
		t.Errorf("an empty table is an answer, not a failure: %q", res.Output)
	}
	if !strings.Contains(res.Output, "no background") {
		t.Errorf("silence reads as a broken tool: %q", res.Output)
	}
}

// RN-3 makes tool error text a behaviour surface: the message is what the model
// reads to correct itself, so it names what does exist.
func TestProcessNamesTheIdentifiersItKnowsWhenGivenAnUnknownOne(t *testing.T) {
	s, _ := setup(t)
	bg := &fakeBackground{}
	runBash(t, backgroundBash(bg), s, `{"command":"npm start","background":true}`)

	res := runProcess(t, s, `{"id":"bg7"}`)

	if !res.IsError {
		t.Fatal("an unknown identifier returned success")
	}
	if !strings.Contains(res.Output, "bg1") {
		t.Errorf("the error does not say what does exist: %q", res.Output)
	}
}

// A model that started the wrong server needs the port back before it can start
// the right one.
func TestStoppingAProcessReturnsItsFinalOutput(t *testing.T) {
	s, _ := setup(t)
	bg := &fakeBackground{}
	runBash(t, backgroundBash(bg), s, `{"command":"npm start","background":true}`)
	bg.handle(0).write("shutting down\n")

	res := runProcess(t, s, `{"id":"bg1","stop":true}`)

	if res.IsError {
		t.Fatalf("stopping failed: %s", res.Output)
	}
	if !bg.handle(0).wasStopped() {
		t.Error("the process was reported stopped without being stopped")
	}
	if !strings.Contains(res.Output, "shutting down") {
		t.Errorf("the last output was discarded: %q", res.Output)
	}
}

// The decision recorded in docs/backlog/202608130300: a process dies with the
// session. It is the reason approving a long command needs no separate question
// about duration — the authorisation and the process have the same lifetime.
//
// This test is that decision. An orphan on someone's machine is worse than a
// frozen session, because freezing is visible.
func TestClosingTheStateStopsEveryProcess(t *testing.T) {
	s, _ := setup(t)
	bg := &fakeBackground{}
	b := backgroundBash(bg)
	runBash(t, b, s, `{"command":"npm start","background":true}`)
	runBash(t, b, s, `{"command":"go run ./cmd/api","background":true}`)

	s.Close()

	for i, h := range []*fakeHandle{bg.handle(0), bg.handle(1)} {
		if !h.wasStopped() {
			t.Errorf("process %d outlived the session that started it", i+1)
		}
	}
}

// Reading a buffer this process already owns crosses nothing. Were it folded
// into bash — which declares the network and a write to the workspace — every
// read of a log would queue an approval for a boundary it does not touch.
func TestProcessCrossesNoBoundaryAndSoNeverAsks(t *testing.T) {
	req, err := Process{}.Declare([]byte(`{"id":"bg1","stop":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if req.Network {
		t.Error("reading a local buffer was declared as reaching the network")
	}
	if len(req.Paths) != 0 {
		t.Errorf("no path is touched, but %v were declared", req.Paths)
	}
}

func TestBackgroundIsRefusedWhenNothingCanRunIt(t *testing.T) {
	s, _ := setup(t)

	res := runBash(t, Bash{Workdir: "."}, s, `{"command":"npm start","background":true}`)

	if !res.IsError {
		t.Fatal("a background command was accepted with no runner to start it")
	}
}

// An identifier is a sequence, not a clock. A timestamp here would ride into
// tool output and change the history byte-for-byte between otherwise identical
// sessions, which is what MarkRead already refuses for the same reason.
func TestAProcessIdentifierCarriesNoClock(t *testing.T) {
	s, _ := setup(t)
	bg := &fakeBackground{}
	b := backgroundBash(bg)

	first := runBash(t, b, s, `{"command":"a","background":true}`)
	second := runBash(t, b, s, `{"command":"b","background":true}`)

	if !strings.Contains(first.Output, "bg1") || !strings.Contains(second.Output, "bg2") {
		t.Errorf("identifiers are not a plain sequence: %q then %q", first.Output, second.Output)
	}
}

// A chatty server outruns any window. The tail is what says why it died; the
// head is startup banner nobody needs twice.
func TestProcessKeepsTheTailWhenTheOutputIsTooLong(t *testing.T) {
	s, _ := setup(t)
	s.Limits.MaxToolOutput = 200
	bg := &fakeBackground{}
	runBash(t, backgroundBash(bg), s, `{"command":"noisy","background":true}`)
	bg.handle(0).write(strings.Repeat("noise\n", 200) + "THE LAST LINE\n")

	res := runProcess(t, s, `{"id":"bg1"}`)

	if !strings.Contains(res.Output, "THE LAST LINE") {
		t.Error("the tail was dropped, which is the part that says why it died")
	}
	if !res.Truncated {
		t.Error("output was cut without saying so")
	}
}

// Background changes when the call returns, never what it is allowed to touch.
func TestBackgroundDeclaresExactlyWhatAForegroundCommandDeclares(t *testing.T) {
	fg, err := Bash{}.Declare([]byte(`{"command":"npm start"}`))
	if err != nil {
		t.Fatal(err)
	}
	bg, err := Bash{}.Declare([]byte(`{"command":"npm start","background":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if bg.Network != fg.Network || len(bg.Paths) != len(fg.Paths) || bg.Command != fg.Command {
		t.Errorf("background changed the declaration: %+v vs %+v", bg, fg)
	}
}
