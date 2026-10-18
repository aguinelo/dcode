package orchestrate

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/aguinelo/dcode/internal/protocol"
)

// gateRepo is a repository with two commits: file.txt says "one" in the first
// and "two" in the second. It returns the repository and both full SHAs.
func gateRepo(t *testing.T) (repo, first, second string) {
	t.Helper()
	repo = filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "init", "-q")
	write := func(s string) {
		if err := os.WriteFile(filepath.Join(repo, "file.txt"), []byte(s+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		git(t, repo, "add", "file.txt")
		git(t, repo, "commit", "-q", "-m", s)
	}
	write("one")
	first = revParse(t, repo, "HEAD")
	write("two")
	second = revParse(t, repo, "HEAD")
	return repo, first, second
}

func revParse(t *testing.T, dir, rev string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", rev)
	cmd.Dir = dir
	cmd.Env = gitEnv(os.Environ())
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse %s: %v", rev, err)
	}
	return strings.TrimSpace(string(out))
}

// worktrees is how many worktrees git lists for repo, the main one included.
func worktrees(t *testing.T, repo string) int {
	t.Helper()
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repo
	cmd.Env = gitEnv(os.Environ())
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git worktree list: %v", err)
	}
	return strings.Count(string(out), "worktree ")
}

func measure(t *testing.T, g Gate, repo, commit string) protocol.GateResult {
	t.Helper()
	r, err := g.Measure(context.Background(), repo, commit)
	if err != nil {
		t.Fatalf("Measure: %v", err)
	}
	return r
}

// Spec: orchestration .p §7.4 — the gate runs on the exact commit, and
// gate.measured carries the commit and its tree.
func TestAGreenGateIsMeasuredOnTheExactCommit(t *testing.T) {
	repo, first, _ := gateRepo(t)
	g := Gate{Command: "grep -q one file.txt && echo measured-one", Timeout: time.Minute}

	r := measure(t, g, repo, first)

	if r.Result != "green" || r.ExitCode != 0 {
		t.Fatalf("result = %s (exit %d), want green; output:\n%s", r.Result, r.ExitCode, r.OutputTail)
	}
	if r.Commit != first {
		t.Errorf("commit = %q, want %q", r.Commit, first)
	}
	if want := revParse(t, repo, first+"^{tree}"); r.Tree != want {
		t.Errorf("tree = %q, want %q", r.Tree, want)
	}
	if r.Command != g.Command {
		t.Errorf("command = %q, want %q", r.Command, g.Command)
	}
	if !strings.Contains(r.OutputTail, "measured-one") {
		t.Errorf("output tail %q does not carry the gate's output", r.OutputTail)
	}
	if r.At.IsZero() || r.DurationMS < 0 {
		t.Errorf("at = %v, duration = %d: the measurement is not dated", r.At, r.DurationMS)
	}
	if n := worktrees(t, repo); n != 1 {
		t.Errorf("%d worktrees after the gate, want only the person's", n)
	}
}

// Spec: orchestration .p §7.4 — a red gate is red, with its exit code, the
// last 4 KiB in the result and the whole output on disk.
func TestARedGateIsRedWithItsExitAndItsOutput(t *testing.T) {
	repo, _, second := gateRepo(t)
	logs := t.TempDir()
	g := Gate{Command: "head -c 9000 /dev/zero | tr '\\0' x; echo; echo boom; exit 3", Timeout: time.Minute, Logs: logs}

	r := measure(t, g, repo, second)

	if r.Result != "red" || r.ExitCode != 3 {
		t.Fatalf("result = %s (exit %d), want red with exit 3", r.Result, r.ExitCode)
	}
	if len(r.OutputTail) > 4<<10 || !strings.HasSuffix(r.OutputTail, "boom\n") {
		t.Errorf("tail is %d bytes ending %q, want at most 4 KiB ending in the last line", len(r.OutputTail),
			r.OutputTail[max(0, len(r.OutputTail)-10):])
	}
	if want := filepath.Join(logs, second+".log"); r.OutputPath != want {
		t.Fatalf("output path = %q, want %q", r.OutputPath, want)
	}
	whole, err := os.ReadFile(r.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(whole) != 9000+1+5 {
		t.Errorf("the file has %d bytes, want the whole output (%d)", len(whole), 9000+1+5)
	}
}

// Spec: orchestration .config §3 — a gate past gate_timeout is timeout, which
// is not green, and the throwaway worktree still goes.
func TestAGateThatOutlivesItsTimeoutIsTimeoutAndNotGreen(t *testing.T) {
	repo, _, second := gateRepo(t)
	g := Gate{Command: "sleep 30 & sleep 30; wait", Timeout: 300 * time.Millisecond}

	start := time.Now()
	r := measure(t, g, repo, second)

	if r.Result != "timeout" {
		t.Fatalf("result = %s, want timeout", r.Result)
	}
	if took := time.Since(start); took > 10*time.Second {
		t.Errorf("the gate took %v to stop: the timeout did not end its processes", took)
	}
	if n := worktrees(t, repo); n != 1 {
		t.Errorf("%d worktrees after a timeout, want only the person's", n)
	}
}

// Spec: orchestration .p §7.4 — the gate runs in a throwaway worktree, never in
// the person's tree, and without the GIT_* a hook exports.
func TestTheGateRunsInAThrowawayTreeWithoutGitEnvironment(t *testing.T) {
	repo, _, second := gateRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "dirty.txt"), []byte("not committed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_DIR", filepath.Join(t.TempDir(), "elsewhere"))
	t.Setenv("GIT_INDEX_FILE", filepath.Join(t.TempDir(), "index"))
	t.Setenv("GIT_ANYTHING", "set")
	g := Gate{
		Command: `test -z "$GIT_DIR" && test -z "$GIT_INDEX_FILE" && test -z "$GIT_ANYTHING" && ` +
			`test ! -e dirty.txt && touch made-by-gate && pwd -P`,
		Timeout: time.Minute,
	}

	r := measure(t, g, repo, second)

	if r.Result != "green" {
		t.Fatalf("result = %s (exit %d); output:\n%s", r.Result, r.ExitCode, r.OutputTail)
	}
	if _, err := os.Stat(filepath.Join(repo, "made-by-gate")); err == nil {
		t.Error("the gate wrote in the person's tree")
	}
	ran := strings.TrimSpace(r.OutputTail)
	person, _ := filepath.EvalSymlinks(repo)
	if ran == person || strings.HasPrefix(ran, person+string(filepath.Separator)) {
		t.Errorf("the gate ran in %s, inside the person's tree", ran)
	}
	if _, err := os.Stat(ran); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the throwaway tree %s is still there (%v)", ran, err)
	}
}

// Spec: orchestration .p §7.4 — the throwaway worktree is removed even when
// running the gate panics.
func TestTheThrowawayWorktreeIsRemovedEvenOnAPanic(t *testing.T) {
	repo, _, second := gateRepo(t)
	var tree string
	saved := runGate
	runGate = func(_ context.Context, _ Gate, dir string, _ *gateOutput) (int, bool, error) {
		tree = dir
		panic("the gate's runner broke")
	}
	t.Cleanup(func() { runGate = saved })

	func() {
		defer func() {
			if recover() == nil {
				t.Error("the panic did not reach the caller")
			}
		}()
		_, _ = Gate{Command: "true", Timeout: time.Minute}.Measure(context.Background(), repo, second)
	}()

	if tree == "" {
		t.Fatal("the runner was never reached")
	}
	if _, err := os.Stat(tree); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the throwaway tree %s is still there after the panic (%v)", tree, err)
	}
	if n := worktrees(t, repo); n != 1 {
		t.Errorf("%d worktrees after a panic, want only the person's", n)
	}
}

// Spec: orchestration .p §7.4 — without gate_command or verify.command,
// gate_undeclared, and nothing runs.
func TestAnUndeclaredGateIsSaidAndNothingRuns(t *testing.T) {
	repo, _, second := gateRepo(t)
	dir := t.TempDir()
	l := openLog(t, dir, testProject(t))

	r, err := Gate{Command: "  ", Timeout: time.Minute}.Measure(context.Background(), repo, second)

	var pe *protocol.Error
	if !errors.As(err, &pe) || pe.Code != protocol.CodeGateUndeclared {
		t.Fatalf("err = %v, want %s", err, protocol.CodeGateUndeclared)
	}
	if r.Result != "undeclared" || r.Commit != second {
		t.Errorf("result = %s on %q, want undeclared on %q", r.Result, r.Commit, second)
	}
	if n := worktrees(t, repo); n != 1 {
		t.Errorf("%d worktrees for an undeclared gate, want none made", n)
	}
	if _, err := RecordGate(l, "child-1", "", r); err != nil {
		t.Fatalf("RecordGate: %v", err)
	}
	lines := readLines(t, dir)
	if len(lines) == 0 {
		t.Fatal("the undeclared gate was not recorded")
	}
	if got := measuredResult(t, lines[len(lines)-1]); got != "undeclared" {
		t.Errorf("recorded result = %q, want undeclared", got)
	}
}

func measuredResult(t *testing.T, d Decision) string {
	t.Helper()
	var r protocol.GateResult
	if err := json.Unmarshal(d.Measured, &r); err != nil {
		t.Fatalf("measured %s: %v", d.Measured, err)
	}
	return r.Result
}

// Spec: orchestration .p §7.4 and §9.1 — what a child said is a claim beside
// the measurement, never the result.
func TestAChildThatClaimsGreenIsRecordedRedBesideItsClaim(t *testing.T) {
	repo, _, second := gateRepo(t)
	dir := t.TempDir()
	l := openLog(t, dir, testProject(t))
	r := measure(t, Gate{Command: "echo FAIL; exit 2", Timeout: time.Minute}, repo, second)
	claim := "make check passed, all green"

	seq, err := RecordGate(l, "child-1", claim, r)
	if err != nil {
		t.Fatalf("RecordGate: %v", err)
	}

	lines := readLines(t, dir)
	if len(lines) == 0 {
		t.Fatal("nothing was recorded")
	}
	d := lines[len(lines)-1]
	if d.Seq != seq || d.Kind != "gate.measured" || d.Child != "child-1" {
		t.Fatalf("line = seq %d kind %q child %q, want seq %d gate.measured child-1", d.Seq, d.Kind, d.Child, seq)
	}
	if d.Claim != claim {
		t.Errorf("claim = %q, want the child's words %q", d.Claim, claim)
	}
	if got := measuredResult(t, d); got != "red" {
		t.Errorf("measured result = %q, want red whatever the child said", got)
	}
	if !strings.Contains(d.Why, "red") || !strings.Contains(d.Why, second[:12]) {
		t.Errorf("why = %q, want the harness's reason naming the result and the commit", d.Why)
	}
}

// Spec: orchestration .p §7.6 — a measurement of another commit, or over
// another main, than the one about to merge is gate_stale.
func TestAMeasurementOfAnotherCommitOrMainIsStale(t *testing.T) {
	r := protocol.GateResult{Commit: "aaaa", Base: "bbbb", Result: "green"}
	if err := CheckFresh(r, "aaaa", "bbbb"); err != nil {
		t.Fatalf("the same commit over the same main: %v", err)
	}
	for name, c := range map[string]struct {
		r          protocol.GateResult
		head, main string
	}{
		"the branch moved":   {r, "cccc", "bbbb"},
		"main moved":         {r, "aaaa", "dddd"},
		"no base measured":   {protocol.GateResult{Commit: "aaaa", Result: "green"}, "aaaa", "bbbb"},
		"nothing measured":   {protocol.GateResult{}, "aaaa", "bbbb"},
		"nothing to merge":   {r, "", "bbbb"},
		"no main to compare": {r, "aaaa", ""},
	} {
		err := CheckFresh(c.r, c.head, c.main)
		var pe *protocol.Error
		if !errors.As(err, &pe) || pe.Code != protocol.CodeGateStale {
			t.Errorf("%s: err = %v, want %s", name, err, protocol.CodeGateStale)
		}
	}
}

// Spec: orchestration .p §7.4 — the commit is an exact SHA the repository has,
// never a ref that moves under the gate.
func TestAGateRefusesWhatIsNotAnExactCommit(t *testing.T) {
	repo, _, second := gateRepo(t)
	g := Gate{Command: "true", Timeout: time.Minute}
	for _, commit := range []string{"HEAD", "main", second[:12], "", strings.Repeat("0", 40)} {
		if _, err := g.Measure(context.Background(), repo, commit); err == nil {
			t.Errorf("%q was measured; want an error", commit)
		}
	}
	if _, err := g.Measure(context.Background(), t.TempDir(), second); err == nil {
		t.Error("a folder in no repository was measured; want an error")
	}
	if _, err := (Gate{Command: "true"}).Measure(context.Background(), repo, second); err == nil {
		t.Error("a gate without a timeout was measured; want an error")
	}
	if n := worktrees(t, repo); n != 1 {
		t.Errorf("%d worktrees after refusals, want only the person's", n)
	}
}

// Spec: orchestration .p §8 — a gate stopped by the caller did not finish, and
// only a finished gate counts.
func TestAGateTheCallerStopsIsAnErrorAndNotAResult(t *testing.T) {
	repo, _, second := gateRepo(t)
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(200*time.Millisecond, cancel)

	r, err := Gate{Command: "sleep 30", Timeout: time.Minute}.Measure(ctx, repo, second)

	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want the cancellation", err)
	}
	if r.Result != "" {
		t.Errorf("result = %q for a gate that never finished", r.Result)
	}
	if n := worktrees(t, repo); n != 1 {
		t.Errorf("%d worktrees after a cancel, want only the person's", n)
	}
}

// Spec: orchestration .config §3 — gate_command, verify.command as its
// fallback, gate_timeout from 1m to 6h.
func TestTheGateIsReadFromConfigurationWithVerifyAsFallback(t *testing.T) {
	g, err := GateFrom(resolved(map[string]string{"verify.command": "go test ./..."}))
	if err != nil {
		t.Fatal(err)
	}
	if g.Command != "go test ./..." || g.Timeout != 30*time.Minute {
		t.Errorf("fallback = %+v, want verify.command and 30m", g)
	}
	g, err = GateFrom(resolved(map[string]string{
		"verify.command": "go test ./...", "orchestrator.gate_command": "make check",
		"orchestrator.gate_timeout": "2h",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if g.Command != "make check" || g.Timeout != 2*time.Hour {
		t.Errorf("declared = %+v, want make check and 2h", g)
	}
	g, err = GateFrom(resolved(nil))
	if err != nil || g.Command != "" {
		t.Errorf("nothing declared = %+v, %v; want an empty command and no error (gate_undeclared is the merge's)", g, err)
	}
	for _, bad := range []string{"30s", "7h", "soon", "-1m"} {
		_, err := GateFrom(resolved(map[string]string{"orchestrator.gate_timeout": bad}))
		if err == nil || !strings.Contains(err.Error(), "orchestrator.gate_timeout") || !strings.Contains(err.Error(), bad) {
			t.Errorf("gate_timeout %q: err = %v, want one naming the key and the value", bad, err)
		}
	}
}

// Spec: orchestration .p §9.1 — only a measurement is recorded as one, and a
// timeout says it is not green.
func TestOnlyAMeasurementIsRecordedAsOne(t *testing.T) {
	dir := t.TempDir()
	l := openLog(t, dir, testProject(t))
	sha := strings.Repeat("a", 40)
	for name, c := range map[string]struct {
		l *DecisionLog
		r protocol.GateResult
	}{
		"no log":           {nil, protocol.GateResult{Commit: sha, Result: GateGreen}},
		"no result":        {l, protocol.GateResult{Commit: sha}},
		"a made-up result": {l, protocol.GateResult{Commit: sha, Result: "passed"}},
		"no exact commit":  {l, protocol.GateResult{Commit: "HEAD", Result: GateGreen}},
	} {
		if _, err := RecordGate(c.l, "child-1", "", c.r); err == nil {
			t.Errorf("%s: recorded; want an error", name)
		}
	}
	if _, err := RecordGate(l, "child-1", "", protocol.GateResult{Commit: sha, Result: GateTimeout, Command: "make check"}); err != nil {
		t.Fatal(err)
	}
	lines := readLines(t, dir)
	if len(lines) != 1 || !strings.Contains(lines[0].Why, "not green") {
		t.Errorf("lines = %+v, want one timeout saying it is not green", lines)
	}
}

// Spec: orchestration .config §4 — the tail is the last 4 KiB, starting on a
// whole character, and a gates directory that cannot be made is said.
func TestTheTailStartsOnAWholeCharacterAndAnUnwritableGatesDirIsSaid(t *testing.T) {
	o := &gateOutput{}
	_, _ = o.Write([]byte(strings.Repeat("é", gateTail)))
	tail := o.tailText()
	if len(tail) > gateTail || !utf8.ValidString(tail) {
		t.Errorf("tail is %d bytes, valid UTF-8 %v; want at most %d and valid", len(tail), utf8.ValidString(tail), gateTail)
	}

	repo, _, second := gateRepo(t)
	file := filepath.Join(t.TempDir(), "a-file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (Gate{Command: "true", Timeout: time.Minute, Logs: file}).Measure(context.Background(), repo, second); err == nil {
		t.Error("a gates directory that is a file was accepted")
	}
	if n := worktrees(t, repo); n != 1 {
		t.Errorf("%d worktrees, want only the person's", n)
	}
}
