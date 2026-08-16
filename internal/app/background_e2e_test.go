package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/tools"
)

// freePort asks the kernel for a port nobody is using and gives it straight
// back. A hard-coded port makes the test fail on a machine that happens to be
// running something, which reads as a defect in the code under test.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// call runs a tool through the registry the session actually built, which is
// what makes this an end-to-end test rather than a test of tools.Bash.
func call(t *testing.T, sess *Session, state *tools.State, name, args string) tools.Result {
	t.Helper()
	tool, ok := sess.Registry.Get(name)
	if !ok {
		t.Fatalf("the session has no %q tool", name)
	}
	res, err := tool.Execute(context.Background(), json.RawMessage(args), state)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return res
}

func waitServing(port int, within time.Duration) bool {
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
		if err == nil {
			resp.Body.Close()
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// Three servers at once, which is the shape people actually debug in: an API,
// a worker and a front end, brought up together to see how they behave against
// each other.
//
// The whole claim under test is that starting them does not cost the loop its
// ability to keep working, and that none of them outlives the session.
func TestThreeServersRunAtOnceAndDieWithTheSession(t *testing.T) {
	if _, err := os.Stat("/usr/bin/python3"); err != nil {
		t.Skip("no python3 to serve with")
	}

	ws := t.TempDir()
	// Something for the loop to do while the servers run, so "the loop stayed
	// free" is asserted rather than assumed.
	if err := os.WriteFile(filepath.Join(ws, "notes.txt"), []byte("hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	opts := baseOpts(t)
	opts.Workspace = ws
	// The permission model is not what this measures, and a prompt nobody can
	// answer would hang the test.
	opts.Policy = policy.PolicyNever
	opts.SandboxMode = policy.ModeFullAccess
	requireSandbox(t, opts)

	sess, err := New(opts, &ConsoleEmitter{W: io.Discard}, DenyAll{})
	if err != nil {
		t.Fatalf("wiring a session failed: %v", err)
	}
	state := sess.State
	if state == nil {
		t.Fatal("the session has no tool state, so nothing owns the processes")
	}
	// The guarantee under test at the end. Deferred here so a failure midway
	// still does not leave three servers on the machine.
	defer sess.Engine.Close()

	ports := []int{freePort(t), freePort(t), freePort(t)}
	var ids []string

	started := time.Now()
	for i, port := range ports {
		// No cd: bash already runs in the workspace, and building the path into
		// the command is how the quoting went wrong the first time.
		args, err := json.Marshal(map[string]any{
			"command":    fmt.Sprintf("python3 -m http.server %d", port),
			"background": true,
		})
		if err != nil {
			t.Fatal(err)
		}
		res := call(t, sess, state, "bash", string(args))
		if res.IsError {
			t.Fatalf("server %d did not start: %s", i+1, res.Output)
		}
		// The settle window answers "did it come up", and saying so is the
		// point: a start that reported success on a process that had already
		// died would be the tool claiming a success the command did not have.
		if strings.Contains(res.Output, "during startup") {
			t.Fatalf("server %d died on startup: %s", i+1, res.Output)
		}
		id := strings.Fields(res.Output)[2] // "started as bgN · ..."
		ids = append(ids, id)
	}
	t.Logf("three servers started in %s: %v", time.Since(started).Round(time.Millisecond), ids)

	// All three are actually serving. Reading the tool's word for it would
	// test the tool's optimism, not the processes.
	for i, port := range ports {
		if !waitServing(port, 5*time.Second) {
			t.Errorf("server %d (%s, port %d) never answered", i+1, ids[i], port)
		}
	}

	// The loop is free: an ordinary tool call works while all three run, and
	// it is not queued behind them.
	readArgs, err := json.Marshal(map[string]string{"path": filepath.Join(ws, "notes.txt")})
	if err != nil {
		t.Fatal(err)
	}
	read := call(t, sess, state, "read", string(readArgs))
	if read.IsError || !strings.Contains(read.Output, "hello") {
		t.Errorf("the loop could not work while the servers ran: %s", read.Output)
	}

	// And they are all visible as running, which is what makes them followable
	// rather than merely started.
	list := call(t, sess, state, "process", `{}`)
	for _, id := range ids {
		if !strings.Contains(list.Output, id) {
			t.Errorf("%s is missing from the listing:\n%s", id, list.Output)
		}
	}
	if n := strings.Count(list.Output, "running"); n != 3 {
		t.Errorf("%d of 3 are running:\n%s", n, list.Output)
	}

	// One can be stopped without touching the others: a server that has served
	// its purpose, or one holding a port the next start needs.
	if res := call(t, sess, state, "process",
		fmt.Sprintf(`{"id":%q,"stop":true}`, ids[0])); res.IsError {
		t.Fatalf("stopping %s failed: %s", ids[0], res.Output)
	}
	if waitServing(ports[0], 2*time.Second) {
		t.Errorf("%s kept serving after it was stopped", ids[0])
	}
	if !waitServing(ports[1], 2*time.Second) || !waitServing(ports[2], 2*time.Second) {
		t.Error("stopping one server took the others with it")
	}

	// Nothing survives what created it. Closing the session ends the two that
	// are left, as a consequence of ownership rather than of cleanup somebody
	// remembered to write.
	sess.Engine.Close()
	for i := 1; i < 3; i++ {
		if waitServing(ports[i], 3*time.Second) {
			t.Errorf("server %d (%s, port %d) outlived its session", i+1, ids[i], ports[i])
		}
	}
}

// A process that dies on startup is reported as dead, not as started. A tool
// that says "started" about a command that already exited hands the model a
// success it can only discover was false much later.
func TestAServerThatDiesOnStartupIsNotReportedAsStarted(t *testing.T) {
	opts := baseOpts(t)
	opts.Workspace = t.TempDir()
	opts.Policy = policy.PolicyNever
	opts.SandboxMode = policy.ModeFullAccess
	requireSandbox(t, opts)

	sess, err := New(opts, &ConsoleEmitter{W: io.Discard}, DenyAll{})
	if err != nil {
		t.Fatalf("wiring a session failed: %v", err)
	}
	defer sess.Engine.Close()

	res := call(t, sess, sess.State, "bash",
		`{"command":"echo boom >&2; exit 3","background":true}`)
	if !strings.Contains(res.Output, "during startup") {
		t.Errorf("a command that exited immediately was reported as running:\n%s", res.Output)
	}
	if !res.Meta.HasExit || res.Meta.ExitCode != 3 {
		t.Errorf("exit code = %d (present %v), want 3", res.Meta.ExitCode, res.Meta.HasExit)
	}
}
