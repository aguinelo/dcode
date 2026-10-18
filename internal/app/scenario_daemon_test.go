package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The scenarios of what happens around a child's work rather than to it: main
// moving under its branch, the daemon going down in the middle of a turn, and
// a child that waits for the person while the others go on.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.i.spec.md, step 12;
// .p §6, §7.4, §8, §10.

// TestScenarioAConflictReturnsToTheChildThatOwnsTheBranch: main changes the
// file the child changed while it works; updating the branch conflicts, the
// work goes back to the child that owns the branch with the paths, it
// resolves them, and the gate measures a commit that contains main.
func TestScenarioAConflictReturnsToTheChildThatOwnsTheBranch(t *testing.T) {
	s := newScenario(t, scenarioOptions{forge: "github"})
	var moved string
	s.model.orchestrator(dispatch(item("shared", "shared.txt")), say("Dispatched."))
	s.model.child("shared",
		use("read", map[string]string{"path": "shared.txt"}),
		writes("shared.txt", "the child's line\n"),
		say("Done.").then(func() { moved = s.commitOnMain("shared.txt", "main's line\n") }),
		use("read", map[string]string{"path": "shared.txt"}),
		writes("shared.txt", "main's line\nthe child's line\n"),
		say("Resolved: both lines kept."))
	s.model.reviewer("shared", approve("fine"), approve("still fine"))
	s.open()
	s.ask("Change the shared file.")

	c := s.waitState("shared", "working")
	s.waitQuestion(c.ID)

	var conflict *protocol.Return
	for _, r := range s.returns(c.ID) {
		if r.Why == "conflict" {
			r := r
			conflict = &r
		}
	}
	if conflict == nil || len(conflict.Paths) != 1 || conflict.Paths[0] != "shared.txt" {
		t.Fatalf("the hand-backs were %+v, want a conflict on shared.txt", s.returns(c.ID))
	}
	if sent := s.model.sentTo("child/shared"); !strings.Contains(sent, "<<<<<<<") {
		t.Error("the child's worktree held no conflict markers when it read the file")
	}
	g := s.lastGate(c.ID)
	if g.Result != "green" || g.Base != moved || !s.ancestor(moved, g.Commit) {
		t.Errorf("the gate measured %s over %s (%s), want a green commit containing main's tip %s", g.Commit, g.Base, g.Result, moved)
	}
	if got := s.show(g.Commit, "shared.txt"); got != "main's line\nthe child's line\n" {
		t.Errorf("the measured commit holds shared.txt = %q, want the child's resolution", got)
	}
	integration := filepath.Join(s.orch, s.project.ID, "integration")
	if _, err := os.Stat(integration); err == nil {
		if st := s.git(integration, "status", "--porcelain"); st != "" {
			t.Errorf("the integration tree is not clean after the conflict:\n%s", st)
		}
	}
	s.inOrder(c.ID, "child.completed", "review.verdict", "child.returned", "child.completed", "gate.measured", "question.asked")
	s.quiet()
}

// TestScenarioADaemonRestartEndsTheTurnWithItsReason: the daemon goes down
// in the middle of a child's turn; the next one reopens the orchestrator and
// the child under their own ids, ends the open turn daemon_restart — on the
// child's log, in the list and in the decision log, whose seq goes on with
// no hole — and reports it to the orchestrator.
func TestScenarioADaemonRestartEndsTheTurnWithItsReason(t *testing.T) {
	s := newScenario(t, scenarioOptions{forge: "local"})
	never := make(chan struct{})
	s.model.orchestrator(dispatch(item("docs", "docs")), say("Dispatched."))
	s.model.child("docs", writes("docs/a.md", "half\n"), say("unreachable").heldUntil(never))
	s.open()
	s.ask("Write the docs.")
	c := s.waitState("docs", "working")
	s.waitFor("the child's turn never reached the model the second time", func() bool {
		return s.model.requests("child/docs") == 2
	})
	before := len(s.decisions())

	s.restart()

	orq := orchestratorOf(t, s.d, s.project.ID)
	if orq.ID != s.project.OrchestratorID {
		t.Errorf("after the restart the orchestrator is %s, want %s", orq.ID, s.project.OrchestratorID)
	}
	cs, err := s.d.manager.Get(c.ID)
	if err != nil || cs.Role != protocol.RoleChild || cs.ParentID != s.project.OrchestratorID {
		t.Fatalf("after the restart the child is %+v, %v", cs, err)
	}
	if reason := lastTurnEnd(t, cs.Log.Replay); reason != protocol.StopDaemonRestart {
		t.Errorf("the child's open turn ended %q, want daemon_restart", reason)
	}
	k := s.waitState("docs", "failed")
	if k.Reason != protocol.StopDaemonRestart {
		t.Errorf("after the restart the child is failed %q, want daemon_restart", k.Reason)
	}
	all := s.decisions() // the reader fails on any hole in seq
	if len(all) <= before {
		t.Fatal("the restart wrote nothing to the decision log")
	}
	restart := s.all("", "daemon.restart")
	if len(restart) != 1 || !strings.Contains(string(restart[0].Data), c.ID) {
		t.Errorf("daemon.restart is %v, want one naming the child it reopened", restart)
	}
	s.waitFor("the restart was never reported to the orchestrator", func() bool {
		o, err := s.d.manager.Get(s.project.OrchestratorID)
		return err == nil && hasOrigin(o.Log.Replay, protocol.OriginChildren)
	})
	s.inOrder("", "dispatch", "child.state", "daemon.restart", "child.state")
	close(never)
	s.quiet()
}

// replay is a session log's Replay.
type replay func(from uint64) ([]protocol.Event, error)

// lastTurnEnd is the reason of the last turn.completed of a session's log.
func lastTurnEnd(t *testing.T, r replay) string {
	t.Helper()
	events, err := r(0)
	if err != nil {
		t.Fatal(err)
	}
	reason := ""
	for _, ev := range events {
		if ev.Type == protocol.EventTurnCompleted {
			var tc protocol.TurnCompleted
			if err := json.Unmarshal(ev.Payload, &tc); err != nil {
				t.Fatal(err)
			}
			reason = tc.Reason
		}
	}
	return reason
}

// hasOrigin is whether a session's log has a turn started by origin.
func hasOrigin(r replay, origin string) bool {
	events, err := r(0)
	if err != nil {
		return false
	}
	for _, ev := range events {
		var ts protocol.TurnStarted
		if ev.Type == protocol.EventTurnStarted && json.Unmarshal(ev.Payload, &ts) == nil && ts.Origin == origin {
			return true
		}
	}
	return false
}

// TestScenarioAChildApprovalWaitsForThePerson: one child asks the person to
// cross its boundary; the approval has no deadline and outlives the daemon's
// approval timeout, nothing but the person answers it, and the orchestrator
// carries the other child to its merge question meanwhile.
func TestScenarioAChildApprovalWaitsForThePerson(t *testing.T) {
	s := newScenario(t, scenarioOptions{forge: "local"})
	outside := filepath.Join(t.TempDir(), "elsewhere.txt")
	if err := os.WriteFile(outside, []byte("the person's notes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	s.model.orchestrator(dispatch(item("notes", "notes"), item("docs", "docs")), say("Two children."))
	s.model.child("notes",
		use("read", map[string]string{"path": outside}),
		writes("notes/n.md", "read them\n"),
		say("Done."))
	s.model.child("docs", writes("docs/a.md", "docs\n"), say("Done."))
	s.model.reviewer("notes", approve("fine"))
	s.model.reviewer("docs", approve("fine"))
	s.open()
	s.ask("Notes and docs.")

	notes := s.waitState("notes", "awaiting_person")
	if notes.Waiting != "approval" {
		t.Errorf("the child waits on %q, want approval", notes.Waiting)
	}
	cs, err := s.d.manager.Get(notes.ID)
	if err != nil {
		t.Fatal(err)
	}
	docs := s.kid("docs")
	s.waitQuestion(docs.ID)
	time.Sleep(4 * s.d.opts.ApprovalTimeout)
	pending := cs.Pending()
	if len(pending) != 1 || !pending[0].ExpiresAt.IsZero() {
		t.Fatalf("past the approval timeout the child's approval is %+v, want one, with no deadline", pending)
	}
	if k := s.kid("notes"); k.State != "awaiting_person" {
		t.Errorf("while the person has not answered the child is %s", k.State)
	}
	if orq, _ := s.d.manager.Get(s.project.OrchestratorID); orq.State() != protocol.SessionStateIdle {
		t.Errorf("with a child waiting the orchestrator is %s, want idle", orq.State())
	}

	s.approve(notes.ID, pending[0].ApprovalID)
	s.waitQuestion(notes.ID)
	if got := s.show(s.kid("notes").Head, "notes/n.md"); got != "read them\n" {
		t.Errorf("after the person's yes the child's commit holds %q", got)
	}
	s.inOrder(notes.ID, "dispatch", "child.state", "child.state", "child.completed", "review.verdict", "gate.measured", "question.asked")
	s.quiet()
}
