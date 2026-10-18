package orchestrate

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = gitEnv(cmd.Environ())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func event(t *testing.T, typ protocol.EventType, payload any) protocol.Event {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return protocol.Event{Type: typ, Payload: raw}
}

func (p *project) one(t *testing.T, it Item) protocol.Child {
	t.Helper()
	kids, err := p.o.Delegate([]Item{it})
	if err != nil {
		t.Fatal(err)
	}
	return kids[0]
}

// Spec: orchestration .i step 6 — a child's approval leaves it
// awaiting_person, waiting on an approval, until the person answers.
func TestAChildsApprovalLeavesItAwaitingThePerson(t *testing.T) {
	p := openWith(t, nil)
	c := p.one(t, item("index", "internal/index"))
	p.reg.Observe(c.ID, event(t, protocol.EventApprovalRequired, protocol.ApprovalRequest{ApprovalID: "a1"}))
	if got := p.o.child(c.ID); got.State != stateAwaiting || got.Waiting != "approval" {
		t.Fatalf("after the approval the child is %s/%s", got.State, got.Waiting)
	}
	p.reg.Observe(c.ID, event(t, protocol.EventApprovalResolved, protocol.ApprovalResolved{ApprovalID: "a1"}))
	if got := p.o.child(c.ID); got.State != stateWorking || got.Waiting != "" {
		t.Errorf("after the answer the child is %s/%s", got.State, got.Waiting)
	}
}

// Spec: orchestration .i step 6 — the harness commits at the end of each of
// the child's turns, naming the paths; child.completed carries what the child
// said as a claim, and the turn's tokens count against the ceiling.
func TestTheHarnessCommitsTheEndOfEachChildTurnByName(t *testing.T) {
	p := openWith(t, nil)
	c := p.one(t, item("index", "internal/index"))
	if err := os.MkdirAll(filepath.Join(c.Worktree, "internal", "index"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(c.Worktree, "internal", "index", "i.go"), []byte("package index\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p.reg.Observe(c.ID, event(t, protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t1"}))
	p.reg.Observe(c.ID, event(t, protocol.EventMessageDelta, protocol.MessageDelta{Text: "Wrote it; make check passed."}))
	p.reg.Observe(c.ID, event(t, protocol.EventTurnCompleted, protocol.TurnCompleted{TurnID: "t1", Reason: protocol.StopDone,
		Usage: &protocol.Usage{InputTokens: 10, OutputTokens: 5}}))

	got := p.o.child(c.ID)
	if got.Head == "" || got.State != stateWorking || got.Reason != protocol.StopDone || got.Usage.InputTokens != 10 {
		t.Fatalf("child after its turn = %+v", got)
	}
	if files := strings.TrimSpace(gitOut(t, c.Worktree, "show", "--name-only", "--format=", got.Head)); files != "internal/index/i.go" {
		t.Errorf("the commit holds %q", files)
	}
	if msg := strings.TrimSpace(gitOut(t, c.Worktree, "log", "-1", "--format=%s")); msg != "index" {
		t.Errorf("commit message %q", msg)
	}
	if p.o.snapshot().Spent != 15 {
		t.Errorf("Spent = %d, want the child's 15", p.o.snapshot().Spent)
	}
	cc, ok := p.w.last[protocol.EventChildCompleted].(protocol.ChildCompleted)
	if !ok || cc.Conclusion != "Wrote it; make check passed." || len(cc.Wrote) != 1 {
		t.Errorf("child.completed = %+v", p.w.last[protocol.EventChildCompleted])
	}
	data, _ := os.ReadFile(filepath.Join(p.dir, p.o.Project().ID, decisionsFile))
	if !strings.Contains(string(data), `"claim":"Wrote it; make check passed."`) {
		t.Errorf("the child's claim is not kept in the log as a claim:\n%s", data)
	}

	p.reg.Observe(c.ID, event(t, protocol.EventTurnStarted, protocol.TurnStarted{TurnID: "t2"}))
	p.reg.Observe(c.ID, event(t, protocol.EventTurnCompleted, protocol.TurnCompleted{TurnID: "t2", Reason: protocol.StopMaxIterations}))
	if got := p.o.child(c.ID); got.State != stateFailed || got.Reason != protocol.StopMaxIterations {
		t.Errorf("a turn ending max_iterations left the child %s/%s", got.State, got.Reason)
	}
}

// Spec: orchestration .i step 6 — a closed child leaves /children after
// closed_retention; a failed one stays. Closing removes the worktree and
// frees the owns.
func TestAClosedChildLeavesTheListAfterTheRetentionAndAFailedOneStays(t *testing.T) {
	p := openWith(t, nil)
	a := p.one(t, item("a", "a"))
	b := p.one(t, item("b", "b"))
	p.reg.Observe(b.ID, event(t, protocol.EventTurnCompleted, protocol.TurnCompleted{TurnID: "t1", Reason: protocol.StopError}))

	if held, err := p.reg.CloseChild(a.ID); !held || err != nil {
		t.Fatalf("CloseChild = %v, %v", held, err)
	}
	if _, err := os.Stat(a.Worktree); !os.IsNotExist(err) {
		t.Errorf("the closed child's worktree is still there: %v", err)
	}
	if got := p.o.child(a.ID); got.State != stateClosed || got.Reason != "cancelled" || got.Ended == nil {
		t.Errorf("closed child = %+v", got)
	}
	if len(p.o.Children()) != 2 {
		t.Errorf("inside the retention the closed child is listed: %d", len(p.o.Children()))
	}
	p.clk.add(2 * time.Hour)
	kids := p.o.Children()
	if len(kids) != 1 || kids[0].ID != b.ID || kids[0].State != stateFailed {
		t.Errorf("after the retention /children = %+v, want the failed one alone", kids)
	}
	if _, err := p.o.Delegate([]Item{item("a again", "a")}); err != nil {
		t.Errorf("the closed child's owns were not freed: %v", err)
	}
	kinds := strings.Join(kindsIn(t, p.dir, p.o.Project().ID), " ")
	if !strings.Contains(kinds, "child.closed worktree.removed") {
		t.Errorf("log = %s", kinds)
	}
}

// Spec: orchestration .p §8 — at the daemon's start each child that is not
// terminal is reopened under its own id; one whose turn was open fails with
// daemon_restart, said in the list and in the log.
func TestAChildIsReopenedAfterARestartAndAnOpenTurnFails(t *testing.T) {
	p := openWith(t, nil)
	c := p.one(t, item("index", "internal/index"))
	p.reg.Close()

	again := p.registry(t, nil)
	if errs := again.Reload(t.Context()); len(errs) != 0 {
		t.Fatalf("Reload: %v", errs)
	}
	if !strings.Contains(strings.Join(p.sp.reopened, " "), c.ID) {
		t.Fatalf("reopened %v, want the child %s", p.sp.reopened, c.ID)
	}
	kids, held := again.ChildrenOf(c.ParentID)
	if !held || len(kids) != 1 || kids[0].State != stateFailed || kids[0].Reason != protocol.StopDaemonRestart {
		t.Errorf("after the restart /children = %+v", kids)
	}
	kinds := strings.Join(kindsIn(t, p.dir, p.o.Project().ID), " ")
	if !strings.HasSuffix(kinds, "daemon.restart child.state") {
		t.Errorf("log = %s", kinds)
	}
}

// Spec: orchestration .p §9.1 — core.hooksPath read after making a worktree
// differs from the one read before: hooks.changed with both values.
func TestAWorktreeThatChangesTheHooksPathIsSaid(t *testing.T) {
	p := openWith(t, nil)
	hooks := filepath.Join(p.root, ".git", "hooks")
	script := "#!/bin/sh\ngit config core.hooksPath /nowhere\n"
	if err := os.WriteFile(filepath.Join(hooks, "post-checkout"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	p.one(t, item("index", "internal/index"))
	data, _ := os.ReadFile(filepath.Join(p.dir, p.o.Project().ID, decisionsFile))
	if !strings.Contains(string(data), `"kind":"hooks.changed"`) || !strings.Contains(string(data), `"after":"/nowhere"`) {
		t.Errorf("hooks.changed is not in the log:\n%s", data)
	}
}
