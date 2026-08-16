package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/tools"
)

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

// A long-lived process that announces itself, in nothing but shell builtins
// and sleep.
//
// It used to be `python3 -m http.server`, and answering real HTTP was a nice
// property that cost the test its portability: on the macOS CI runner python
// ran and printed nothing for sixty seconds, alive and silent, and no amount of
// waiting fixed it. Serving HTTP was never the claim. The claim is that N
// long-lived processes run at once, stay followable, and die with the session —
// and a shell loop models that on every platform with a shell.
func serverCmd(marker string) string {
	return fmt.Sprintf("echo up > %s; while true; do sleep 1; done", marker)
}

// up reports whether the process got far enough to announce itself. A file
// rather than a port: the question is "did it come up", and the answer must not
// depend on a runtime the machine may not have.
func up(dir, marker string) bool {
	_, err := os.Stat(filepath.Join(dir, marker))
	return err == nil
}

// waitAllUp polls every marker against ONE deadline rather than giving each its
// own in turn.
//
// Sequential waits hid a slow machine as a per-server failure: the first two
// spent their windows and the third answered because it had inherited theirs.
// A shared deadline costs slow startup once.
func waitAllUp(dir string, markers []string, within time.Duration) []string {
	deadline := time.Now().Add(within)
	for {
		var missing []string
		for _, m := range markers {
			if !up(dir, m) {
				missing = append(missing, m)
			}
		}
		if len(missing) == 0 {
			return nil
		}
		if !time.Now().Before(deadline) {
			return missing
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// Three servers at once, which is the shape people actually debug in: an API,
// a worker and a front end, brought up together to see how they behave against
// each other.
//
// The whole claim under test is that starting them does not cost the loop its
// ability to keep working, and that none of them outlives the session.
func TestThreeServersRunAtOnceAndDieWithTheSession(t *testing.T) {
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

	markers := []string{"up1", "up2", "up3"}
	var ids []string

	started := time.Now()
	for i, m := range markers {
		args, err := json.Marshal(map[string]any{
			"command":    serverCmd(m),
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
		ids = append(ids, strings.Fields(res.Output)[2]) // "started as bgN · ..."
	}
	t.Logf("three servers started in %s: %v", time.Since(started).Round(time.Millisecond), ids)

	// All three really came up. Reading the tool's word for it would test the
	// tool's optimism, not the processes. One shared deadline, generous: what
	// is measured is that three run at once, not how fast a loaded runner is.
	if missing := waitAllUp(ws, markers, 60*time.Second); len(missing) > 0 {
		// What the product says about them, so a failure here is diagnosable
		// rather than just "never came up".
		t.Fatalf("%v never came up; process reports:\n%s",
			missing, call(t, sess, state, "process", `{}`).Output)
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
	// its purpose, or one holding a resource the next start needs.
	if res := call(t, sess, state, "process",
		fmt.Sprintf(`{"id":%q,"stop":true}`, ids[0])); res.IsError {
		t.Fatalf("stopping %s failed: %s", ids[0], res.Output)
	}
	// Polled, not read once. "running" straight after a stop is the truth: the
	// signal is sent and the process has not been reaped yet, and reporting
	// "stopped" before it is would be the product claiming something it does
	// not know. What has to hold is that it gets there, and that it takes only
	// the one process with it.
	stopDeadline := time.Now().Add(10 * time.Second)
	for {
		list = call(t, sess, state, "process", `{}`)
		running := strings.Count(list.Output, "running")
		if strings.Contains(list.Output, "stopped") && running == 2 {
			break
		}
		if !time.Now().Before(stopDeadline) {
			t.Fatalf("stopping %s left the listing at %d running:\n%s", ids[0], running, list.Output)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Nothing survives what created it. Closing the session ends the two that
	// are left, as a consequence of ownership rather than of cleanup somebody
	// remembered to write.
	//
	// Asserted against the operating system, not against the product's own
	// bookkeeping: a session that merely forgot its processes would pass a
	// check that asked the session.
	alive := livePIDs(t, ws)
	// Without this the assertion below passes by finding nothing, which is the
	// exact shape of defect this codebase keeps turning up: a check that reads
	// something no side wrote. Two are still running at this point.
	if len(alive) < 2 {
		t.Fatalf("found %d live process(es) before closing; the check below would prove nothing", len(alive))
	}
	sess.Engine.Close()
	deadline := time.Now().Add(15 * time.Second)
	for {
		still := stillRunning(alive)
		if len(still) == 0 {
			break
		}
		if !time.Now().Before(deadline) {
			t.Fatalf("%d process(es) outlived the session: %v", len(still), still)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// livePIDs finds the shell loops this test started, by the marker each one
// writes. Reading the process table is the only way to tell "the session
// stopped tracking it" from "the process is gone".
func livePIDs(t *testing.T, ws string) []int {
	t.Helper()
	out, err := exec.Command("ps", "-Ao", "pid=,args=").Output()
	if err != nil {
		t.Skipf("cannot read the process table here: %v", err)
	}
	var pids []int
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "up1") && !strings.Contains(line, "up2") && !strings.Contains(line, "up3") {
			continue
		}
		if !strings.Contains(line, "while true") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if pid, err := strconv.Atoi(fields[0]); err == nil {
			pids = append(pids, pid)
		}
	}
	return pids
}

func stillRunning(pids []int) []int {
	var alive []int
	for _, pid := range pids {
		p, err := os.FindProcess(pid)
		if err != nil {
			continue
		}
		// Signal 0 asks "is it there" without touching it.
		if err := p.Signal(syscall.Signal(0)); err == nil {
			alive = append(alive, pid)
		}
	}
	return alive
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
