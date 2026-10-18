package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/sandbox"
)

// The harness of the orchestrator's scripted scenarios: a real daemon,
// in-process, with orchestrator.enabled, configured through the real chain —
// environment and project — and talking to the scripted model over HTTP; a
// toy repository per scenario, with a bare origin and the fake gh when the
// scenario has a forge. Deterministic and offline: no scenario reaches a
// network, a real model, or the machine's git configuration.
//
// A scenario that cannot run fails, never skips: no git, no sandbox, is a
// scenario that proved nothing, and a skip reads as a pass.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.i.spec.md, step 12.

// scenarioWait bounds every wait of a scenario. Generous for a race build on
// a loaded runner; a scenario that needs it is one that is stuck.
const scenarioWait = 45 * time.Second

// gateScript is the toy project's gate: red while a file says BROKEN.
const gateScript = `#!/bin/sh
if grep -rqx BROKEN --exclude=gate.sh --exclude-dir=.git . ; then
	echo "gate: a file says BROKEN"
	exit 1
fi
echo "gate: green"
`

type scenario struct {
	t     *testing.T
	model *scenarioModel
	d     *Daemon
	env   map[string]string
	// root is the project, origin its bare remote ("" without a forge),
	// gh the fake gh's folder.
	root, origin, gh string
	records, orch    string
	project          protocol.Project
}

// scenarioOptions shape a scenario's project.
type scenarioOptions struct {
	// forge is "github" (a bare origin and the fake gh) or "local" (no
	// origin; the person's tree off main, as a local merge needs).
	forge string
	// env adds or overrides configuration, by variable.
	env map[string]string
	// after runs on the repository once it is made, before the project opens.
	after func(root string)
}

// newScenario builds the repository and the daemon; the project is not open
// yet, so a scenario can change the tree first. Every requirement it does
// not find fails the scenario.
func newScenario(t *testing.T, o scenarioOptions) *scenario {
	t.Helper()
	for _, bin := range []string{"git", "sh", "grep"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Fatalf("a scripted scenario needs %s, and this machine has none: %v", bin, err)
		}
	}
	if err := sandboxHere(); err != nil {
		t.Fatalf("a scripted scenario runs its children in the sandbox, and this machine has none: %v", err)
	}
	for k, v := range map[string]string{"GIT_CONFIG_GLOBAL": "/dev/null", "GIT_CONFIG_NOSYSTEM": "1",
		"GIT_AUTHOR_NAME": "scenario", "GIT_AUTHOR_EMAIL": "s@example.com",
		"GIT_COMMITTER_NAME": "scenario", "GIT_COMMITTER_EMAIL": "s@example.com"} {
		t.Setenv(k, v)
	}
	s := &scenario{t: t, model: newScenarioModel(t), records: t.TempDir(), orch: t.TempDir()}
	s.root = s.repo(o)
	if o.forge == "github" {
		s.withForge()
	}

	srv := httptest.NewServer(http.HandlerFunc(s.model.serve))
	t.Cleanup(srv.Close)
	t.Cleanup(s.model.check)
	t.Cleanup(s.model.stop)
	home := t.TempDir()
	s.env = map[string]string{
		"HOME": home, "DCODE_HOME": home,
		"DCODE_API_KEY": "sk-test-key-abcdefghijklmnop", "DCODE_BASE_URL": srv.URL,
		"DCODE_MEMORY_ENABLED":               "false",
		"DCODE_ORCHESTRATOR_GATE_COMMAND":    "sh gate.sh",
		"DCODE_ORCHESTRATOR_GATE_TIMEOUT":    "1m",
		"DCODE_ORCHESTRATOR_CI_TIMEOUT":      "1m",
		"DCODE_ORCHESTRATOR_REPORT_DEBOUNCE": "10ms",
		"DCODE_ORCHESTRATOR_FORGE":           "local",
	}
	if o.forge == "github" {
		s.env["DCODE_ORCHESTRATOR_FORGE"] = "github"
	}
	for k, v := range o.env {
		s.env[k] = v
	}
	s.start()
	return s
}

// sandboxHere is whether this machine can confine a child's commands.
func sandboxHere() error {
	_, err := sandbox.New(sandbox.Config{Backend: sandbox.BackendAuto, AllowNetwork: func() bool { return true }},
		policy.ModeWorkspaceWrite)
	return err
}

// start opens a daemon on the scenario's directories: the first, or the one
// after a restart.
func (s *scenario) start() {
	s.t.Helper()
	env := s.env
	s.d = NewDaemon(DaemonOptions{
		SocketPath: filepath.Join(s.t.TempDir(), "d.sock"), EventRetention: 100000, RecordDir: s.records,
		ApprovalTimeout: 50 * time.Millisecond, Log: func(string) {},
		Base:         Options{Env: func(k string) string { return env[k] }},
		Orchestrator: OrchestratorOptions{Enabled: true, Dir: s.orch, ClosedRetention: time.Hour},
	})
	d := s.d
	s.t.Cleanup(d.closeProjects)
	s.t.Cleanup(d.manager.CloseAll)
}

// restart stops the daemon — every session closed in the middle of whatever
// it was doing, as a crash leaves them — and starts another on the same
// directories, which reloads the projects.
func (s *scenario) restart() {
	s.t.Helper()
	s.d.manager.CloseAll()
	s.d.closeProjects()
	s.start()
	if errs := s.d.reloadProjects(context.Background()); len(errs) != 0 {
		s.t.Fatalf("the projects did not reload: %v", errs)
	}
}

// repo is the toy project: a gate, a shared file, and main.
func (s *scenario) repo(o scenarioOptions) string {
	s.t.Helper()
	root, err := filepath.EvalSymlinks(s.t.TempDir())
	if err != nil {
		s.t.Fatal(err)
	}
	s.git(root, "init", "-q", "-b", "main")
	s.git(root, "config", "commit.gpgsign", "false")
	s.write(root, "gate.sh", gateScript)
	s.write(root, "shared.txt", "base\n")
	s.git(root, "add", "-A")
	s.git(root, "commit", "-qm", "the toy project")
	if o.forge != "github" {
		// A local merge moves main by a conditional update-ref, refused while
		// any worktree has main checked out (.p §7.6): the person's is off it.
		s.git(root, "checkout", "-q", "--detach")
	}
	if o.after != nil {
		o.after(root)
	}
	return root
}

// withForge gives the project a bare origin holding main, and the fake gh
// first on PATH.
func (s *scenario) withForge() {
	s.t.Helper()
	s.origin = filepath.Join(s.t.TempDir(), "origin.git")
	s.git(s.root, "init", "-q", "--bare", "-b", "main", s.origin)
	s.git(s.root, "remote", "add", "origin", s.origin)
	s.git(s.root, "push", "-q", "-u", "origin", "main")
	bin, err := filepath.Abs(filepath.Join("testdata", "scenariogh"))
	if err != nil {
		s.t.Fatal(err)
	}
	s.gh = s.t.TempDir()
	s.t.Setenv("SCENARIO_GH", s.gh)
	s.t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// open opens the project, as the person's client does.
func (s *scenario) open() {
	s.t.Helper()
	s.project = openProject(s.t, s.d, s.root, http.StatusCreated)
}

// ask is the person's first message to the orchestrator.
func (s *scenario) ask(text string) {
	s.t.Helper()
	code, body := route(s.t, s.d, http.MethodPost, "/sessions/"+s.project.OrchestratorID+"/turns",
		protocol.SubmitTurnRequest{Text: text})
	if code != http.StatusAccepted && code != http.StatusOK && code != http.StatusCreated {
		s.t.Fatalf("the person's turn was refused: %d %s", code, body)
	}
}

// children is /children of the orchestrator, as the person's client reads it.
func (s *scenario) children() []protocol.Child {
	s.t.Helper()
	return childrenOf(s.t, s.d, s.project.OrchestratorID)
}

// kid is the child with that title; the zero Child when there is none.
func (s *scenario) kid(title string) protocol.Child {
	s.t.Helper()
	for _, c := range s.children() {
		if c.Title == title {
			return c
		}
	}
	return protocol.Child{}
}

// waitFor waits for cond, failing with why and the log so far.
func (s *scenario) waitFor(why string, cond func() bool) {
	s.t.Helper()
	deadline := time.Now().Add(scenarioWait)
	for !cond() {
		if time.Now().After(deadline) {
			s.t.Fatalf("%s\nthe decision log so far:\n%s", why, s.logText())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// waitState waits for the child with that title to be in state.
func (s *scenario) waitState(title, state string) protocol.Child {
	s.t.Helper()
	var c protocol.Child
	s.waitFor("child "+title+" never reached "+state, func() bool {
		c = s.kid(title)
		return c.State == state
	})
	return c
}

// quiet waits until the orchestrator is idle and the log has stopped
// growing: the pipeline has nothing in flight, and the scenario can end
// without a step still writing into its folders.
func (s *scenario) quiet() {
	s.t.Helper()
	last, still := -1, 0
	s.waitFor("the project never went quiet", func() bool {
		orq, err := s.d.manager.Get(s.project.OrchestratorID)
		n := len(s.decisions())
		if err != nil || orq.State() != protocol.SessionStateIdle || n != last {
			last, still = n, 0
			return false
		}
		still++
		return still > 60 // ~300ms with nothing new
	})
}

func (s *scenario) git(dir string, args ...string) string {
	s.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		s.t.Fatalf("git %s in %s: %v\n%s", strings.Join(args, " "), dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

func (s *scenario) write(dir, name, body string) {
	s.t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		s.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		s.t.Fatal(err)
	}
}

// show is a file at a commit of the project, "" when it is not there.
func (s *scenario) show(commit, path string) string {
	s.t.Helper()
	cmd := exec.Command("git", "show", commit+":"+path)
	cmd.Dir = s.root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// mainTip is main where it merges: origin's with a forge, the project's without.
func (s *scenario) mainTip() string {
	s.t.Helper()
	if s.origin != "" {
		// Fetched, so the project has the commit to show: a squash is made
		// on the forge, never in the project.
		s.git(s.root, "fetch", "-q", "origin")
		return s.git(s.root, "--git-dir="+s.origin, "rev-parse", "refs/heads/main")
	}
	return s.git(s.root, "rev-parse", "refs/heads/main")
}

// commitOnMain is somebody else's change landing on main while the
// orchestrator works: pushed to origin with a forge, a commit made without
// a checkout without one. It returns main's new tip.
func (s *scenario) commitOnMain(path, body string) string {
	s.t.Helper()
	if s.origin != "" {
		clone := filepath.Join(s.t.TempDir(), "other")
		s.git(s.root, "clone", "-q", s.origin, clone)
		s.write(clone, path, body)
		s.git(clone, "add", path)
		s.git(clone, "commit", "-qm", "somebody else's change to "+path)
		s.git(clone, "push", "-q", "origin", "HEAD:refs/heads/main")
		return s.mainTip()
	}
	// An index of its own, so the person's tree and its index stay as they
	// are; set on these commands alone, never on the process the daemon's
	// own git inherits from.
	index := []string{"GIT_INDEX_FILE=" + filepath.Join(s.t.TempDir(), "index")}
	blob := s.gitWith(nil, body, "hash-object", "-w", "--stdin")
	s.gitWith(index, "", "read-tree", "refs/heads/main")
	s.gitWith(index, "", "update-index", "--add", "--cacheinfo", "100644,"+blob+","+path)
	tree := s.gitWith(index, "", "write-tree")
	commit := s.git(s.root, "commit-tree", tree, "-p", "refs/heads/main", "-m", "somebody else's change to "+path)
	s.git(s.root, "update-ref", "refs/heads/main", commit)
	return commit
}

// gitWith runs git in the project with more environment and stdin.
func (s *scenario) gitWith(env []string, stdin string, args ...string) string {
	s.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir, cmd.Stdin = s.root, strings.NewReader(stdin)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.Output()
	if err != nil {
		s.t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out))
}

// ghCalls is every call the fake gh got, as one text.
func (s *scenario) ghCalls() string {
	b, err := os.ReadFile(filepath.Join(s.gh, "calls"))
	if err != nil && !os.IsNotExist(err) {
		s.t.Fatal(err)
	}
	return string(b)
}

// closeChild is the person's DELETE of a child.
func (s *scenario) closeChild(id string) {
	s.t.Helper()
	if code, body := route(s.t, s.d, http.MethodDelete, "/sessions/"+id, nil); code != http.StatusNoContent {
		s.t.Fatalf("the person's close of %s = %d %s", id, code, body)
	}
}

// approve answers a child's pending approval as the person, through the route.
func (s *scenario) approve(childID, approvalID string) {
	s.t.Helper()
	code, body := route(s.t, s.d, http.MethodPost, "/sessions/"+childID+"/approvals/"+approvalID,
		protocol.ResolveApprovalRequest{Decision: protocol.ApprovalAllow})
	if code >= 300 {
		s.t.Fatalf("the person's approval was refused: %d %s", code, body)
	}
}

// decode reads a JSON object of the log, failing the scenario on one that
// does not read.
func decode[T any](t *testing.T, raw json.RawMessage) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("%s does not read: %v", raw, err)
	}
	return v
}
