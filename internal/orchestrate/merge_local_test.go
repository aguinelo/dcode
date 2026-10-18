package orchestrate

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// localProject is a project with no forge: no origin, the person's tree on a
// branch of its own so main is checked out nowhere, a gate that passes, and
// one child measured green over main.
func localProject(t *testing.T) (mergeProject, Forge, protocol.Child) {
	t.Helper()
	p := newProject(t, false)
	main := revParse(t, p.root, "main")
	git(t, p.root, "switch", "-q", "-c", "work")
	_, head := p.child(t, "dcode/a", "main", "a.txt", "a\n")
	in, err := OpenIntegration(context.Background(), p.root, p.dir)
	if err != nil {
		t.Fatal(err)
	}
	f, err := NewForge(context.Background(), ForgeSetup{Declared: ForgeLocal, Root: p.root, Main: "main",
		Integration: in, Gate: Gate{Command: "true", Timeout: time.Minute}})
	if err != nil {
		t.Fatal(err)
	}
	return p, f, measured("c1", "dcode/a", head, main)
}

// Spec: orchestration .p §7.6 — with no forge the merge is local: a squash
// in the integration place, and main moved by a conditional update-ref; the
// first time, the log says what is lost, and main's health is the gate run
// on the merged commit.
func TestALocalMergeIsAConditionalUpdateOfMainAndSaysWhatIsLost(t *testing.T) {
	p, f, c := localProject(t)
	m, _ := newMerges(t, p, f, &events{})
	lost := decisionOf(t, p, "forge.local")
	for _, w := range []string{"pull request", "CI", "server"} {
		if !strings.Contains(lost.Why, w) {
			t.Errorf("forge.local does not say the %s is lost: %s", w, lost.Why)
		}
	}
	c = proposeAndAccept(t, m, c)
	if c.PR != nil {
		t.Errorf("a local proposal opened a PR: %+v", c.PR)
	}
	out, err := m.Merge(context.Background(), c, false)
	if err != nil || out.Completed == nil {
		t.Fatalf("Merge = %+v, %v", out, err)
	}
	s := revParse(t, p.root, "main")
	if out.Completed.Merged != s || out.Completed.Method != "squash" || out.Completed.Measured != c.Head {
		t.Errorf("merge.completed = %+v, main at %s", out.Completed, s)
	}
	if revParse(t, p.root, s+"^") != c.Gate.Base || revParse(t, p.root, s+"^{tree}") != revParse(t, p.root, c.Head+"^{tree}") {
		t.Error("the squash is not one commit over the measured main carrying the measured tree")
	}
	if out.Main.Health != "green" || out.Main.Commit != s {
		t.Errorf("main = %+v, want the gate green on the merged commit", out.Main)
	}
	if revParse(t, p.root, "HEAD") != revParse(t, p.root, "work") {
		t.Error("the person's tree moved")
	}
}

// Spec: orchestration .i step 10 — local with main checked out in a worktree
// is refused, saying where.
func TestALocalMergeIsRefusedWithMainCheckedOutSayingWhere(t *testing.T) {
	p, f, c := localProject(t)
	git(t, p.root, "switch", "-q", "main")
	m, _ := newMerges(t, p, f, &events{})
	c = proposeAndAccept(t, m, c)
	before := revParse(t, p.root, "main")
	out, err := m.Merge(context.Background(), c, false)
	if err != nil {
		t.Fatal(err)
	}
	if out.Refused == nil || out.Refused.Code != "forge" || !strings.Contains(out.Refused.Reason, p.root) {
		t.Fatalf("outcome = %+v, want the forge's refusal naming %s", out, p.root)
	}
	if revParse(t, p.root, "main") != before {
		t.Error("main moved under the person's checkout")
	}
}

// Spec: orchestration .i step 10 — update-ref with the wrong old value does
// not move main: the conditional swap is the lock on the measured commit.
func TestAConditionalUpdateWithTheWrongOldValueLeavesMainAlone(t *testing.T) {
	p, f, c := localProject(t)
	before := revParse(t, p.root, "main")
	_, elsewhere := p.child(t, "dcode/x", "main", "x.txt", "x\n")
	c.Gate.Base = elsewhere // not where main is
	if _, err := f.Merge(context.Background(), c, c.Head); err == nil {
		t.Fatal("the merge went through over a main that was not the measured one")
	}
	if revParse(t, p.root, "main") != before {
		t.Error("main moved")
	}
}

// Spec: orchestration .p §7.6 — the local merge's main health is the gate:
// red on the merged commit stops the queue.
func TestARedGateOnTheMergedMainStopsTheLocalQueue(t *testing.T) {
	p := newProject(t, false)
	main := revParse(t, p.root, "main")
	git(t, p.root, "switch", "-q", "-c", "work")
	_, h1 := p.child(t, "dcode/a", "main", "a.txt", "a\n")
	_, h2 := p.child(t, "dcode/b", "main", "b.txt", "b\n")
	in, err := OpenIntegration(context.Background(), p.root, p.dir)
	if err != nil {
		t.Fatal(err)
	}
	// Green while a.txt is absent: main before the merge passes, after it fails.
	f, err := NewForge(context.Background(), ForgeSetup{Declared: ForgeLocal, Root: p.root, Main: "main",
		Integration: in, Gate: Gate{Command: "test ! -f a.txt", Timeout: time.Minute}})
	if err != nil {
		t.Fatal(err)
	}
	m, _ := newMerges(t, p, f, &events{})
	c1 := proposeAndAccept(t, m, measured("c1", "dcode/a", h1, main))
	c2 := proposeAndAccept(t, m, measured("c2", "dcode/b", h2, main))
	if out, err := m.Merge(context.Background(), c1, false); err != nil || out.Main.Health != "red" {
		t.Fatalf("first merge = %+v, %v; want main red", out, err)
	}
	if out, err := m.Merge(context.Background(), c2, false); err != nil || out.Refused == nil || out.Refused.Code != protocol.CodeMainRed {
		t.Fatalf("second merge = %+v, %v; want main_red", out, err)
	}
}

// An answer is said with the merges' lock released: what hears it may ask
// the merges back, as the orchestrator saving their queue does. Said under
// the lock, the reply waited for the answer and the answer for the reply —
// the deadlock TestScenarioARedMainStopsTheQueue hit when the queue was
// saved while the person answered. Both answers are asked: merge, and
// return, which also says merge.refused.
func TestAnAnswerIsSaidWithTheMergesLockReleased(t *testing.T) {
	for _, answer := range []string{"merge", "return"} {
		t.Run(answer, func(t *testing.T) {
			p, f, c := localProject(t)
			m, _ := newMerges(t, p, f, &events{})
			_, q, err := m.Propose(context.Background(), Proposal{Child: c, Title: c.Title, Body: "What it does."})
			if err != nil {
				t.Fatal(err)
			}
			var stuck []protocol.EventType
			m.o.Emit = func(typ protocol.EventType, _ any) {
				back := make(chan struct{})
				go func() { m.Queued(); m.Questions(); close(back) }()
				select {
				case <-back:
				case <-time.After(10 * time.Second): // a ceiling: the reply needs no time, only the lock
					stuck = append(stuck, typ)
				}
			}
			if _, err := m.Answer(q.ID, answer, ""); err != nil && answer == "merge" {
				t.Fatal(err)
			}
			if len(stuck) > 0 {
				t.Errorf("asking the merges back from %v waited for the answer that was saying it", stuck)
			}
		})
	}
}
