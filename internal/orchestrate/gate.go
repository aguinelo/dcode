package orchestrate

// The gate, measured by the orchestrator itself. What a child says of its own
// tests is a claim; this is the measurement a merge is decided on (RN-5).
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §7.4, §7.6;
// .config.spec.md §3.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/protocol"
)

const (
	// gateTail is how much of the output a GateResult carries (.config §4);
	// the whole of it goes to the gate's file.
	gateTail = 4 << 10
	// The bounds and default of orchestrator.gate_timeout (.config §3).
	gateTimeoutDefault = 30 * time.Minute
	gateTimeoutMin     = time.Minute
	gateTimeoutMax     = 6 * time.Hour
	// gateWaitDelay is how long Wait waits, after the kill, for output pipes
	// something outside the process group still holds.
	gateWaitDelay = 5 * time.Second
	// gitDeadline bounds each git the gate runs around the command: resolving
	// the commit, adding the throwaway tree, removing it.
	gitDeadline = 2 * time.Minute
)

// The four results of .p §5.2.
const (
	GateGreen      = "green"
	GateRed        = "red"
	GateTimeout    = "timeout"
	GateUndeclared = "undeclared"
)

// exactCommit is a full object name. A ref or an abbreviation names whatever
// it points at when git reads it, and the measurement has to be of one commit.
var exactCommit = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

// Gate is the project's declared gate: orchestrator.gate_command, or
// verify.command when that is empty.
type Gate struct {
	// Command runs under /bin/sh -c at the root of the throwaway tree. Empty
	// is gate_undeclared: nothing is measured, and nothing merges.
	Command string
	// Timeout bounds one run; a gate past it is timeout, which is not green.
	Timeout time.Duration
	// Logs is the directory the whole output is written to, as
	// <commit>.log — the gates/ of .p §8. Empty keeps only the tail.
	Logs string
}

// GateFrom reads the gate from configuration. An empty command is not an error
// here: gate_undeclared is said when a measurement is asked for (.config §3).
func GateFrom(r config.Resolved) (Gate, error) {
	g := Gate{Command: strings.TrimSpace(r.String("orchestrator.gate_command", "")), Timeout: gateTimeoutDefault}
	if g.Command == "" {
		g.Command = strings.TrimSpace(r.String("verify.command", ""))
	}
	if v := r.String("orchestrator.gate_timeout", ""); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d < gateTimeoutMin || d > gateTimeoutMax {
			return Gate{}, fmt.Errorf("orchestrator.gate_timeout must be a duration from 1m to 6h, got %q", v)
		}
		g.Timeout = d
	}
	return g, nil
}

// runGate runs the command in tree. A variable so a test can make it panic and
// see the tree removed anyway.
var runGate = runCommand

// Measure runs the gate on commit, an exact SHA the repository at dir has.
//
// It runs in a throwaway worktree made at that commit for this run alone —
// never the person's tree, never a child's worktree, whose files may not be
// the commit's — with every GIT_* variable out of the environment, so a gate
// started from a hook still finds the tree it stands in. The worktree is
// removed when Measure returns, on a timeout and on a panic too.
//
// Base, the main under the commit, is the caller's to fill: the pipeline made
// the commit by merging main and knows which.
//
// An error means nothing counts — with one exception, gate_undeclared, whose
// result ("undeclared") comes back with it to be recorded. A throwaway tree
// that could not be removed is an error too, beside the result it measured. A gate the caller stopped did not finish, and only a
// finished gate counts (.p §8).
func (g Gate) Measure(ctx context.Context, dir, commit string) (_ protocol.GateResult, err error) {
	if !exactCommit.MatchString(commit) {
		return protocol.GateResult{}, &protocol.Error{Code: protocol.CodeInvalidInput,
			Message: fmt.Sprintf("the gate measures one exact commit, a full SHA; %q is not one", commit)}
	}
	gctx, cancel := context.WithTimeout(ctx, gitDeadline)
	tree, err := gateGit(gctx, dir, "rev-parse", "--verify", "--quiet", commit+"^{commit}^{tree}")
	cancel()
	if err != nil {
		return protocol.GateResult{}, fmt.Errorf("orchestrate: the gate cannot measure %s: %w", commit, err)
	}
	r := protocol.GateResult{Commit: commit, Tree: tree, Command: g.Command}
	if strings.TrimSpace(g.Command) == "" {
		r.Result, r.At = GateUndeclared, time.Now().UTC()
		return r, &protocol.Error{Code: protocol.CodeGateUndeclared, Message: "neither orchestrator.gate_command " +
			"nor verify.command declares a gate: nothing was measured, and nothing merges without a measurement"}
	}
	if g.Timeout <= 0 {
		return protocol.GateResult{}, fmt.Errorf("orchestrate: a gate without a timeout would wait forever; got %v", g.Timeout)
	}

	out, err := newGateOutput(g.Logs, commit)
	if err != nil {
		return protocol.GateResult{}, err
	}
	scratch, err := os.MkdirTemp("", "dcode-gate-")
	if err != nil {
		_ = out.close()
		return protocol.GateResult{}, fmt.Errorf("orchestrate: make the gate's throwaway directory: %w", err)
	}
	place := filepath.Join(scratch, "tree")
	defer func() {
		if cerr := removeTree(dir, place, scratch); cerr != nil {
			err = errors.Join(err, cerr)
		}
	}()

	gctx, cancel = context.WithTimeout(ctx, gitDeadline)
	_, err = gateGit(gctx, dir, "worktree", "add", "--quiet", "--detach", place, commit)
	cancel()
	if err != nil {
		_ = out.close()
		return protocol.GateResult{}, fmt.Errorf("orchestrate: make the gate's worktree at %s: %w", commit, err)
	}

	start := time.Now()
	exit, timedOut, runErr := runGate(ctx, g, place, out)
	r.DurationMS = int(time.Since(start).Milliseconds())
	r.At = time.Now().UTC()
	if cerr := out.close(); cerr != nil {
		runErr = errors.Join(runErr, cerr)
	}
	if runErr != nil {
		return protocol.GateResult{}, runErr
	}
	r.ExitCode, r.OutputTail, r.OutputPath = exit, out.tailText(), out.path
	switch {
	case timedOut:
		r.Result = GateTimeout
	case exit == 0:
		r.Result = GateGreen
	default:
		r.Result = GateRed
	}
	return r, nil
}

// runCommand runs the gate's command in tree until it ends, the timeout
// passes, or ctx is done. The command gets a process group of its own, so the
// kill reaches everything it started.
func runCommand(ctx context.Context, g Gate, tree string, out *gateOutput) (exit int, timedOut bool, err error) {
	tctx, cancel := context.WithTimeout(ctx, g.Timeout)
	defer cancel()
	cmd := exec.CommandContext(tctx, "/bin/sh", "-c", g.Command)
	cmd.Dir = tree
	cmd.Env = withoutGit(cmd.Environ()) // Environ sets PWD to Dir
	cmd.Stdout, cmd.Stderr = out, out   // the same writer: one goroutine writes
	setGateGroup(cmd)
	cmd.Cancel = func() error { return killGateGroup(cmd) }
	cmd.WaitDelay = gateWaitDelay

	err = cmd.Run()
	if ctx.Err() != nil {
		return 0, false, fmt.Errorf("orchestrate: the gate was stopped before it finished, so it measured nothing: %w", ctx.Err())
	}
	if errors.Is(tctx.Err(), context.DeadlineExceeded) {
		return cmd.ProcessState.ExitCode(), true, nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode(), false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("orchestrate: run the gate %q: %w", g.Command, err)
	}
	return 0, false, nil
}

// removeTree takes the throwaway worktree out of git and off the disk. It runs
// on a context of its own: the caller's may be the reason the gate stopped.
func removeTree(repo, place, scratch string) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitDeadline)
	defer cancel()
	var errs []error
	if _, err := os.Stat(place); err == nil {
		if _, err := gateGit(ctx, repo, "worktree", "remove", "--force", place); err != nil {
			errs = append(errs, err)
		}
	}
	if err := os.RemoveAll(scratch); err != nil {
		errs = append(errs, err)
	}
	if _, err := gateGit(ctx, repo, "worktree", "prune"); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("orchestrate: the gate's throwaway worktree %s was not removed: %w", place, errors.Join(errs...))
	}
	return nil
}

func gateGit(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = gitEnv(cmd.Environ())
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s in %s: %s", strings.Join(args, " "), dir, msg)
	}
	return strings.TrimSpace(string(out)), nil
}

// withoutGit drops every GIT_* variable. A hook exports GIT_DIR and its kin,
// and a gate that runs git — or tests that make repositories of their own —
// would find the hook's repository instead of the tree it stands in.
func withoutGit(env []string) []string {
	out := env[:0:0]
	for _, kv := range env {
		if !strings.HasPrefix(kv, "GIT_") {
			out = append(out, kv)
		}
	}
	return out
}

// RecordGate writes gate.measured: the measurement, with the child it is about
// and what the child said of its own gate. The claim is kept beside the
// measurement as data and never read as it: the result is the measured one.
func RecordGate(l *DecisionLog, child, claim string, r protocol.GateResult) (uint64, error) {
	if l == nil {
		return 0, errors.New("orchestrate: no decision log to record the gate in")
	}
	switch r.Result {
	case GateGreen, GateRed, GateTimeout, GateUndeclared:
	default:
		return 0, fmt.Errorf("orchestrate: %q is not a gate result; a measurement is green, red, timeout or undeclared", r.Result)
	}
	if !exactCommit.MatchString(r.Commit) {
		return 0, fmt.Errorf("orchestrate: a measurement of %q is not of an exact commit", r.Commit)
	}
	measured, err := json.Marshal(r)
	if err != nil {
		return 0, fmt.Errorf("orchestrate: encode the measurement: %w", err)
	}
	return l.Record(Decision{Kind: "gate.measured", Child: child, Why: gateWhy(r), Claim: claim, Measured: measured})
}

func gateWhy(r protocol.GateResult) string {
	switch r.Result {
	case GateUndeclared:
		return fmt.Sprintf("neither orchestrator.gate_command nor verify.command declares a gate, so %s was not "+
			"measured and cannot merge", short(r.Commit))
	case GateTimeout:
		return fmt.Sprintf("the harness ran %q on %s itself, and it passed gate_timeout after %d ms: timeout, "+
			"which is not green", r.Command, short(r.Commit), r.DurationMS)
	}
	return fmt.Sprintf("the harness ran %q on %s itself: %s (exit %d); what a child says of its own gate is a "+
		"claim, never this result", r.Command, short(r.Commit), r.Result, r.ExitCode)
}

// CheckFresh is the merge's lock on the measurement: the gate measured head,
// the commit about to merge, over main, main as it is now. Anything else is
// gate_stale, and the pipeline measures again (.p §7.6). It says nothing of
// green; that is the result's.
func CheckFresh(r protocol.GateResult, head, main string) error {
	stale := func(format string, a ...any) error {
		return &protocol.Error{Code: protocol.CodeGateStale, Message: fmt.Sprintf(format, a...)}
	}
	switch {
	case head == "" || main == "":
		return stale("no commit to merge or no main to compare the measurement with; nothing merges unchecked")
	case r.Commit == "":
		return stale("nothing was measured for %s", short(head))
	case r.Commit != head:
		return stale("the gate measured %s and the branch is at %s; it measures again", short(r.Commit), short(head))
	case r.Base == "":
		return stale("the measurement of %s names no main under it; it measures again", short(r.Commit))
	case r.Base != main:
		return stale("the gate measured over main at %s and main is at %s; it measures again", short(r.Base), short(main))
	}
	return nil
}

func short(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

// gateOutput keeps what the gate writes: the last gateTail bytes in memory,
// and the whole of it in the gate's file when there is one. Memory is bounded
// whatever the gate prints.
type gateOutput struct {
	tail []byte
	file *os.File
	path string
	err  error // the first write to the file that failed
}

func newGateOutput(logs, commit string) (*gateOutput, error) {
	o := &gateOutput{}
	if logs == "" {
		return o, nil
	}
	if err := os.MkdirAll(logs, 0o700); err != nil {
		return nil, fmt.Errorf("orchestrate: make the gates directory %s: %w", logs, err)
	}
	o.path = filepath.Join(logs, commit+".log")
	f, err := os.OpenFile(o.path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, fmt.Errorf("orchestrate: open the gate's output %s: %w", o.path, err)
	}
	o.file = f
	return o, nil
}

func (o *gateOutput) Write(p []byte) (int, error) {
	if o.file != nil && o.err == nil {
		_, o.err = o.file.Write(p)
	}
	o.tail = append(o.tail, p...)
	if len(o.tail) > 2*gateTail {
		o.tail = append(o.tail[:0], o.tail[len(o.tail)-gateTail:]...)
	}
	return len(p), nil
}

// tailText is the last gateTail bytes, starting on a whole UTF-8 character.
func (o *gateOutput) tailText() string {
	t := o.tail
	if len(t) > gateTail {
		t = t[len(t)-gateTail:]
	}
	for i := 0; i < 3 && len(t) > 0 && t[0]&0xC0 == 0x80; i++ {
		t = t[1:]
	}
	return string(t)
}

// close syncs and closes the file, and says the first write that failed: a
// gate whose whole output was promised to disk and is not there is not quiet.
func (o *gateOutput) close() error {
	if o.file == nil {
		return nil
	}
	err := o.err
	if err == nil {
		err = o.file.Sync()
	}
	if cerr := o.file.Close(); err == nil {
		err = cerr
	}
	o.file = nil
	if err != nil {
		return fmt.Errorf("orchestrate: write the gate's output %s: %w", o.path, err)
	}
	return nil
}
