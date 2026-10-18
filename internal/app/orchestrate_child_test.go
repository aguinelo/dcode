package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/orchestrate"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/tools"
)

// Spec: orchestration .i step 6 — the child's registry has no explore,
// delegate, children, steer_child, close_child nor ask; its shell refuses git
// that writes without asking; its writes outside owns are refused. The
// orchestrator's has the four tools that conduct.
func TestTheChildsRegistryHasNoToolThatConducts(t *testing.T) {
	ws := projectRepo(t)
	opts := baseOpts(t)
	opts.Workspace = ws
	requireSandbox(t, opts)
	(&Daemon{}).childMode("c", []string{"docs"})(&opts)
	child, err := New(opts, &ConsoleEmitter{W: io.Discard}, DenyAll{})
	if err != nil {
		t.Fatal(err)
	}
	names := strings.Join(child.Registry.Names(), " ")
	for _, banned := range []string{"explore", "delegate", "children", "steer_child", "close_child", "ask", "summarize", "propose_answers", "verdict"} {
		if strings.Contains(" "+names+" ", " "+banned+" ") {
			t.Errorf("the child's registry has %s: %s", banned, names)
		}
	}
	bash, _ := child.Registry.Get("bash")
	if _, err := bash.Declare(json.RawMessage(`{"command":"git push origin main"}`)); err == nil {
		t.Error("git push was declared in a child's shell")
	}
	write, _ := child.Registry.Get("write")
	_, err = write.Declare(json.RawMessage(`{"path":"a.txt","content":"x"}`))
	if te, ok := err.(*tools.ToolError); !ok || te.Code != protocol.CodeOutsideOwns {
		t.Errorf("a write outside owns declared %v, want outside_owns", err)
	}
	if _, err := write.Declare(json.RawMessage(`{"path":"docs/a.md","content":"x"}`)); err != nil {
		t.Errorf("a write inside owns was refused: %v", err)
	}

	orq := baseOpts(t)
	orq.Workspace = ws
	(&Daemon{}).orchestratorMode("p")(&orq)
	o, err := New(orq, &ConsoleEmitter{W: io.Discard}, DenyAll{})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"delegate", "children", "steer_child", "close_child", "ask", "summarize", "propose_answers"} {
		if _, ok := o.Registry.Get(want); !ok {
			t.Errorf("the orchestrator has no %s", want)
		}
	}
}

// routedModel answers the orchestrator and the children from their own
// scripts, told apart by what the child's first turn says. A child's request
// waits on hold until the test lets it go.
type routedModel struct {
	mu          sync.Mutex
	orq, child  [][]string
	seenO, seen int
	hold        chan struct{}
}

func (m *routedModel) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	isChild := strings.Contains(string(body), "You are a child of an orchestrator")
	if isChild && m.hold != nil {
		select {
		case <-m.hold:
		case <-r.Context().Done():
			return // the session went away while the child waited
		}
	}
	m.mu.Lock()
	script, n := m.orq, &m.seenO
	if isChild {
		script, n = m.child, &m.seen
	}
	turn := []string{frameText("nothing more")}
	if *n < len(script) {
		turn = script[*n]
	}
	*n++
	m.mu.Unlock()
	w.Header().Set("Content-Type", "text/event-stream")
	for _, f := range turn {
		_, _ = w.Write([]byte("data: " + f + "\n\n"))
	}
	_, _ = w.Write([]byte("data: [DONE]\n\n"))
}

func modelDaemon(t *testing.T, m *routedModel, timeout time.Duration, records, orch string) *Daemon {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	srv := httptest.NewServer(http.HandlerFunc(m.serve))
	t.Cleanup(srv.Close)
	base := baseOpts(t)
	base.BaseURL = srv.URL
	requireSandbox(t, base)
	d := NewDaemon(DaemonOptions{
		SocketPath: filepath.Join(t.TempDir(), "d.sock"), EventRetention: 10000, RecordDir: records,
		ApprovalTimeout: timeout, Base: base, Log: func(string) {},
		Orchestrator: OrchestratorOptions{Enabled: true, Dir: orch},
	})
	t.Cleanup(d.manager.CloseAll)
	t.Cleanup(d.closeProjects)
	return d
}

func childrenOf(t *testing.T, d *Daemon, orq string) []protocol.Child {
	t.Helper()
	code, body := route(t, d, http.MethodGet, "/sessions/"+orq+"/children", nil)
	if code != http.StatusOK {
		t.Fatalf("GET children = %d %s", code, body)
	}
	var out protocol.ListChildrenResponse
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatal(err)
	}
	return out.Children
}

// Spec: orchestration .i step 6 — creating children comes back with their ids,
// and the orchestrator is idle while they run; the person's DELETE closes a
// child, its worktree removed.
func TestCreatingChildrenLeavesTheOrchestratorIdleWhileTheyRun(t *testing.T) {
	ws := projectRepo(t)
	m := &routedModel{hold: make(chan struct{}), orq: [][]string{
		{frameToolCall("c1", "delegate", `{"items":[{"title":"docs","task":"write the docs","owns":["docs"],"why":"its own theme"}]}`)},
		{frameText("Dispatched.")},
	}, child: [][]string{{frameText("Done.")}}}
	d := modelDaemon(t, m, time.Minute, t.TempDir(), t.TempDir())
	p := openProject(t, d, ws, http.StatusCreated)
	orq, err := d.manager.Get(p.OrchestratorID)
	if err != nil {
		t.Fatal(err)
	}
	if err := orq.Submit("split the work"); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, "the orchestrator did not come back from dispatching", func() bool {
		return orq.State() == protocol.SessionStateIdle && len(childrenOf(t, d, orq.ID)) == 1
	})
	kid := childrenOf(t, d, orq.ID)[0]
	if kid.State != "working" || kid.Branch == "" {
		t.Fatalf("while it runs the child is %+v", kid)
	}
	cs, err := d.manager.Get(kid.ID)
	if err != nil || cs.Role != protocol.RoleChild || cs.ParentID != orq.ID || cs.ProjectID != p.ID || cs.Workspace != kid.Worktree {
		t.Fatalf("child session = %+v, %v", cs, err)
	}
	close(m.hold)
	waitUntil(t, "the child's turn did not end", func() bool { return childrenOf(t, d, orq.ID)[0].Reason != "" })

	if code, body := route(t, d, http.MethodDelete, "/sessions/"+kid.ID, nil); code != http.StatusNoContent {
		t.Fatalf("DELETE child = %d %s", code, body)
	}
	if _, err := os.Stat(kid.Worktree); !os.IsNotExist(err) {
		t.Errorf("the closed child's worktree stays: %v", err)
	}
	if kids := childrenOf(t, d, orq.ID); len(kids) != 0 {
		t.Errorf("with no retention the closed child is still listed: %+v", kids)
	}
}

// Spec: orchestration .i step 6 — a child's approval stays pending past
// ApprovalTimeout, with no deadline, and the child is awaiting_person; after
// a restart the child comes back under its own id, failed daemon_restart.
func TestAChildsApprovalWaitsPastTheTimeoutAndSurvivesTheDaemon(t *testing.T) {
	ws := projectRepo(t)
	outside := filepath.Join(t.TempDir(), "elsewhere.txt")
	if err := os.WriteFile(outside, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := &routedModel{child: [][]string{{frameToolCall("c1", "read", `{"path":"`+outside+`"}`)}, {frameText("ok")}}}
	records, orch := t.TempDir(), t.TempDir()
	const timeout = 20 * time.Millisecond
	d := modelDaemon(t, m, timeout, records, orch)
	p := openProject(t, d, ws, http.StatusCreated)
	o, _ := d.projects.reg.Get(p.ID)
	kids, err := o.Delegate([]orchestrate.Item{{Title: "read", Task: "read that file", Owns: []string{"docs"}, Why: "to ask"}})
	if err != nil {
		t.Fatal(err)
	}
	cs, err := d.manager.Get(kids[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	waitUntil(t, "the child's crossing was never put to anyone", func() bool { return len(cs.Pending()) > 0 })
	time.Sleep(10 * timeout)
	pending := cs.Pending()
	if len(pending) != 1 || !pending[0].ExpiresAt.IsZero() {
		t.Fatalf("past the timeout the child's approval is %+v, want one with no deadline", pending)
	}
	waitUntil(t, "the child is not awaiting the person", func() bool {
		k := childrenOf(t, d, p.OrchestratorID)[0]
		return k.State == "awaiting_person" && k.Waiting == "approval"
	})

	d.manager.CloseAll()
	d.closeProjects()
	again := modelDaemon(t, m, timeout, records, orch)
	if errs := again.reloadProjects(context.Background()); len(errs) != 0 {
		t.Fatalf("reload: %v", errs)
	}
	back, err := again.manager.Get(kids[0].ID)
	if err != nil || back.Role != protocol.RoleChild || back.ParentID != p.OrchestratorID {
		t.Fatalf("after the restart the child is %+v, %v", back, err)
	}
	k := childrenOf(t, again, p.OrchestratorID)[0]
	if k.State != "failed" || k.Reason != protocol.StopDaemonRestart {
		t.Errorf("after the restart the child is %s/%s, want failed daemon_restart", k.State, k.Reason)
	}
}

// The orchestrator's tools reach its project through the daemon; a session
// whose project this daemon does not hold is told so, and a child the daemon
// cannot build is refused before it is a session.
func TestTheConductorReachesTheProjectThroughTheDaemon(t *testing.T) {
	ws := projectRepo(t)
	d := modelDaemon(t, &routedModel{hold: make(chan struct{})}, time.Minute, t.TempDir(), t.TempDir())
	p := openProject(t, d, ws, http.StatusCreated)
	c := conductor{d: d, project: p.ID}
	kids, err := c.Delegate([]tools.DelegateItem{{Title: "docs", Task: "t", Owns: []string{"docs"}, Why: "w"}})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := c.Children(); err != nil || len(got) != 1 {
		t.Errorf("Children = %v, %v", got, err)
	}
	if err := c.SteerChild(kids[0].ID, "more", "why"); err != nil {
		t.Errorf("SteerChild = %v", err)
	}
	if err := c.CloseChild(kids[0].ID, "enough"); err != nil {
		t.Errorf("CloseChild = %v", err)
	}
	lost := conductor{d: d, project: "ffffffffffffffffffffffff"}
	if _, err := lost.Children(); err == nil {
		t.Error("a project the daemon does not hold answered")
	}
	if err := lost.SteerChild("x", "y", "z"); err == nil {
		t.Error("a project the daemon does not hold was steered")
	}
	if err := lost.CloseChild("x", "y"); err == nil {
		t.Error("a project the daemon does not hold closed a child")
	}
	if _, err := lost.Delegate(nil); err == nil {
		t.Error("a project the daemon does not hold dispatched")
	}
	if err := d.Steer("nobody", "x"); err == nil {
		t.Error("a session that is not open was steered")
	}
	if err := d.Submit("nobody", "x", ""); err == nil {
		t.Error("a session that is not open was submitted to")
	}
	d.emitTo("nobody", protocol.EventChildState, nil)
	for name, req := range map[string]orchestrate.ChildRequest{
		"bad id":    {ID: "../x", ParentID: p.OrchestratorID, Worktree: ws},
		"taken id":  {ID: p.OrchestratorID, ParentID: p.OrchestratorID, Worktree: ws},
		"no parent": {ID: "c1", ParentID: "nobody", Worktree: ws},
		"profile":   {ID: "c2", ParentID: p.OrchestratorID, Worktree: ws, Profile: "nowhere"},
	} {
		if _, err := d.SpawnChild(req); err == nil {
			t.Errorf("%s: the child was built", name)
		}
	}
	off := &daemonProjects{off: "off"}
	if kids := off.Children("x"); kids != nil {
		t.Error("an off orchestrator listed children")
	}
	if held, _ := off.CloseChild("x"); held {
		t.Error("an off orchestrator closed a child")
	}
	if _, err := (conductor{d: &Daemon{}, project: "p"}).Children(); err == nil {
		t.Error("a daemon that conducts nothing answered")
	}
}
