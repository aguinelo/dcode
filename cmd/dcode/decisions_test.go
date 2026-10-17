package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/orchestrate"
	"github.com/aguinelo/dcode/internal/protocol"
)

// decisionsFixture is a git tree, a state root outside it, and the
// environment a reader of its log is run with. No daemon anywhere: the socket
// names a path nothing listens on.
type decisionsFixture struct {
	repo, path string
	env        func(string) string
}

func newDecisionsFixture(t *testing.T) decisionsFixture {
	t.Helper()
	repo, state, home := t.TempDir(), t.TempDir(), t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	vars := map[string]string{
		"DCODE_STATE_DIR": state, "HOME": home,
		"DCODE_SOCKET": filepath.Join(home, "nobody-listens.sock"),
	}
	env := func(k string) string { return vars[k] }
	path, err := decisionLogPath(context.Background(), repo, env)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(state, "orchestrate"); !strings.HasPrefix(path, want) {
		t.Fatalf("log path %s is not under orchestrator.dir %s", path, want)
	}
	return decisionsFixture{repo: repo, path: path, env: env}
}

// record writes decisions through the log the orchestrator writes with.
func (fx decisionsFixture) record(t *testing.T, ds ...orchestrate.Decision) {
	t.Helper()
	root, id, err := orchestrate.ResolveProject(context.Background(), fx.repo)
	if err != nil {
		t.Fatal(err)
	}
	l, err := orchestrate.OpenDecisionLog(filepath.Dir(filepath.Dir(fx.path)), protocol.Project{ID: id, Root: root})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	for _, d := range ds {
		if _, err := l.Record(d); err != nil {
			t.Fatal(err)
		}
	}
}

func (fx decisionsFixture) run(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	var o, e bytes.Buffer
	err = decisions(context.Background(), append([]string{"--project", fx.repo}, args...), fx.env, time.Now, &o, &e)
	return o.String(), e.String(), err
}

func gateMeasured(child string, exit int) orchestrate.Decision {
	m, _ := json.Marshal(protocol.GateResult{Command: "make check", ExitCode: exit, Result: "red"})
	return orchestrate.Decision{Kind: "gate.measured", Child: child, Why: "the gate ran on H",
		Claim: "make check passed", Measured: m}
}

// Spec: orchestration .p §9.2 — the log is read with the daemon off.
func TestDecisionsReadsTheLogWithTheDaemonOff(t *testing.T) {
	fx := newDecisionsFixture(t)
	fx.record(t,
		orchestrate.Decision{Kind: "dispatch", Why: "three themes, disjoint owns"},
		gateMeasured("c1", 2),
	)
	out, errOut, err := fx.run(t)
	if err != nil {
		t.Fatalf("err = %v\nstderr: %s", err, errOut)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("printed %d lines, want 2:\n%s", len(lines), out)
	}
	for _, want := range []string{"#2", "gate.measured", "c1", "red  make check (exit 2)",
		"claim: make check passed", "why: the gate ran on H"} {
		if !strings.Contains(lines[1], want) {
			t.Errorf("line %q lacks %q", lines[1], want)
		}
	}
	if !strings.Contains(lines[0], "#1  dispatch  why: three themes, disjoint owns") {
		t.Errorf("line %q", lines[0])
	}
	if errOut != "" {
		t.Errorf("stderr on a clean log: %s", errOut)
	}
}

// Spec: orchestration .p §9.2 — a line that does not read is said on stderr
// with its number, the rest is printed, and the command fails.
func TestAnUnreadableLineIsSaidWithItsNumberAndTheRestIsPrinted(t *testing.T) {
	fx := newDecisionsFixture(t)
	fx.record(t, orchestrate.Decision{Kind: "dispatch", Why: "first"})
	appendRaw(t, fx.path, "{this is not a decision\n")
	appendRaw(t, fx.path, `{"v":1,"seq":2,"at":"2026-10-17T21:04:05Z","project":"x","kind":"pr.opened","why":"after the bad one"}`+"\n")

	out, errOut, err := fx.run(t)
	if err == nil {
		t.Fatal("a log with an unreadable line was read as a clean one")
	}
	if !strings.Contains(errOut, fx.path+":2:") {
		t.Errorf("stderr does not name line 2:\n%s", errOut)
	}
	if !strings.Contains(out, "why: first") || !strings.Contains(out, "why: after the bad one") {
		t.Errorf("the readable lines were not all printed:\n%s", out)
	}
}

// Spec: orchestration .p §9.2 — a seq with a gap is said.
func TestAGapInTheSeqIsSaid(t *testing.T) {
	fx := newDecisionsFixture(t)
	fx.record(t, orchestrate.Decision{Kind: "dispatch", Why: "first"})
	appendRaw(t, fx.path, `{"v":1,"seq":5,"at":"2026-10-17T21:04:05Z","project":"x","kind":"pr.opened","why":"later"}`+"\n")

	out, errOut, err := fx.run(t)
	if err == nil {
		t.Fatal("a log with a gap was read as a clean one")
	}
	if !strings.Contains(errOut, ":2: seq 5 follows 1") {
		t.Errorf("stderr does not say the gap:\n%s", errOut)
	}
	if !strings.Contains(out, "why: later") {
		t.Errorf("the line after the gap was not printed:\n%s", out)
	}
}

// Spec: orchestration .p §9.2 — --kind and --child filter; --json returns the
// lines byte for byte.
func TestKindAndChildFilterAndJSONIsTheLinesByteForByte(t *testing.T) {
	fx := newDecisionsFixture(t)
	fx.record(t,
		orchestrate.Decision{Kind: "dispatch", Why: "two themes"},
		gateMeasured("c1", 2),
		gateMeasured("c2", 0),
		orchestrate.Decision{Kind: "pr.opened", Child: "c1", Why: "green <at last> & said"},
	)
	written, err := os.ReadFile(fx.path)
	if err != nil {
		t.Fatal(err)
	}
	all := strings.SplitAfter(string(written), "\n")

	out, _, err := fx.run(t, "--json")
	if err != nil || out != string(written) {
		t.Fatalf("--json = %q (err %v), want the file byte for byte %q", out, err, written)
	}
	out, _, err = fx.run(t, "--json", "--child", "c1")
	if err != nil || out != all[1]+all[3] {
		t.Errorf("--child c1 = %q (err %v), want lines 2 and 4", out, err)
	}
	out, _, err = fx.run(t, "--json", "--kind", "gate.measured,dispatch")
	if err != nil || out != all[0]+all[1]+all[2] {
		t.Errorf("--kind = %q (err %v), want lines 1 to 3", out, err)
	}
	out, _, err = fx.run(t, "--kind", "gate.measured", "--child", "c2")
	if err != nil || strings.Count(out, "\n") != 1 || !strings.Contains(out, "#3") {
		t.Errorf("--kind with --child = %q (err %v), want only #3", out, err)
	}
}

func TestSinceKeepsOnlyTheRecentDecisions(t *testing.T) {
	fx := newDecisionsFixture(t)
	appendRaw(t, fx.path, `{"v":1,"seq":1,"at":"2020-01-01T00:00:00Z","project":"x","kind":"dispatch","why":"long ago"}`+"\n")
	fx.record(t, orchestrate.Decision{Kind: "pr.opened", Why: "just now"})

	out, errOut, err := fx.run(t, "--since", "1h")
	if err != nil {
		t.Fatalf("err = %v\n%s", err, errOut)
	}
	if strings.Contains(out, "long ago") || !strings.Contains(out, "just now") {
		t.Errorf("--since 1h = %q", out)
	}
	if _, _, err := fx.run(t, "--since", "-1h"); err == nil {
		t.Error("a negative --since was accepted")
	}
}

// No log is an error that says where it looked, never an empty answer that
// reads as "no decisions".
func TestNoLogIsAnErrorThatSaysWhere(t *testing.T) {
	fx := newDecisionsFixture(t)
	_, _, err := fx.run(t)
	if err == nil || !strings.Contains(err.Error(), fx.path) {
		t.Fatalf("err = %v, want one naming %s", err, fx.path)
	}
}

// A torn last line is said and not printed, and the reader does not cut it:
// the log is the orchestrator's to change.
func TestATornLastLineIsSaidAndLeftAsItIs(t *testing.T) {
	fx := newDecisionsFixture(t)
	fx.record(t, orchestrate.Decision{Kind: "dispatch", Why: "first"})
	appendRaw(t, fx.path, `{"v":1,"seq":2,"at`)
	before, _ := os.ReadFile(fx.path)

	out, errOut, err := fx.run(t)
	if err == nil || !strings.Contains(errOut, ":2: no newline at the end") {
		t.Fatalf("err = %v, stderr = %s; want the torn line said", err, errOut)
	}
	if strings.Count(out, "\n") != 1 {
		t.Errorf("printed %q, want only the complete line", out)
	}
	if after, _ := os.ReadFile(fx.path); !bytes.Equal(before, after) {
		t.Error("reading the log changed it")
	}
}

func TestFollowPrintsDecisionsAsTheyAreWritten(t *testing.T) {
	fx := newDecisionsFixture(t)
	fx.record(t, orchestrate.Decision{Kind: "dispatch", Why: "first"})

	ctx, cancel := context.WithCancel(context.Background())
	out := &syncBuffer{}
	done := make(chan error, 1)
	go func() {
		done <- decisions(ctx, []string{"--project", fx.repo, "--follow"}, fx.env, time.Now, out, &syncBuffer{})
	}()
	fx.record(t, orchestrate.Decision{Kind: "pr.opened", Why: "second"})
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(out.String(), "why: second") && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(out.String(), "why: first") || !strings.Contains(out.String(), "why: second") {
		t.Fatalf("--follow printed %q", out.String())
	}
}

type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func appendRaw(t *testing.T, path, s string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(s); err != nil {
		t.Fatal(err)
	}
}
