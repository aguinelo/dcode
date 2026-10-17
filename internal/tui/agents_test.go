package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The orchestrator's live list, as text (orchestration RN-9, .i step 13). Fed
// events, never a daemon: the list is a fold over the wire, so the wire is all
// a test needs.

func orchestratorModel(t *testing.T, lang Lang) Model {
	t.Helper()
	m := NewModel("", "/w", "m", "workspace-write", lang)
	return apply(t, m, ev(t, 1, protocol.EventSessionCreated, protocol.Session{
		ID: "orq", Workspace: "/w", Role: protocol.RoleOrchestrator,
	}))
}

func child(id, title, state string, in, out, cache int) protocol.Child {
	return protocol.Child{
		ID: id, ParentID: "orq", Title: title, State: state,
		Usage: protocol.Usage{InputTokens: in, OutputTokens: out, CacheReadTokens: cache},
	}
}

func started(t *testing.T, seq uint64, c protocol.Child) protocol.Event {
	return ev(t, seq, protocol.EventChildStarted, protocol.ChildChanged{Child: c})
}

func agentsGeometry() Geometry {
	g := DefaultGeometry(100, 40)
	g.Palette = Palette{}
	g.Unicode = true
	return g
}

func eightStates(t *testing.T) Model {
	m := orchestratorModel(t, PtBR)
	tela := child("c3", "tela", "awaiting_person", 9000, 800, 0)
	tela.Waiting = "approval"
	busca := child("c5", "busca", "pr", 5000, 1000, 0)
	busca.PR = &protocol.ChildPR{Number: 518, CI: protocol.GitChecksPass}
	cache := child("c6", "cache", "merged", 3000, 200, 0)
	cache.PR = &protocol.ChildPR{Number: 517, CI: protocol.GitChecksPass}
	lint := child("c8", "lint", "failed", 7000, 2000, 0)
	lint.Reason = "incomplete"
	lint.PR = &protocol.ChildPR{Number: 519, CI: protocol.GitChecksFail}
	return apply(t, m,
		started(t, 2, child("c1", "índice", "dispatched", 0, 0, 0)),
		started(t, 3, child("c2", "api", "working", 41200, 3100, 12000)),
		started(t, 4, tela),
		started(t, 5, child("c4", "docs", "in_review", 2000, 300, 1000)),
		started(t, 6, busca),
		started(t, 7, cache),
		started(t, 8, child("c7", "flags", "closed", 100, 10, 0)),
		started(t, 9, lint),
		ev(t, 10, protocol.EventQuestionAsked, protocol.Question{ID: "q1", Kind: "merge", Text: "merge busca?"}),
		ev(t, 11, protocol.EventQuestionAsked, protocol.Question{ID: "q2", Kind: "design", Text: "qual índice?"}),
		ev(t, 12, protocol.EventMainHealth, protocol.MainHealth{Commit: "abc", Health: "green"}),
	)
}

// The golden: one child in each of the eight states, with its tokens, its PR
// and its CI, and what it waits on.
func TestTheColumnDrawsAChildInEachOfTheEightStates(t *testing.T) {
	m := eightStates(t)
	got := renderAgentsPane(m, agentsGeometry(), 40, 40)
	want := []string{
		"[a] AGENTES                       main ✓",
		"◆ 2 perguntas esperam você",
		"tokens entrada/saída/cache",
		"○ índice                      despachado",
		"  0/0/0",
		"● api                        trabalhando",
		"  41.2k/3.1k/12.0k",
		"◆ tela                    esperando você",
		"  9.0k/800/0",
		"  espera: aprovação",
		"◐ docs                        em revisão",
		"  2.0k/300/1.0k",
		"◎ busca                               PR",
		"  5.0k/1.0k/0                     #518 ✓",
		"✓ cache                           merged",
		"  3.0k/200/0                      #517 ✓",
		"– flags                          fechado",
		"  100/10/0",
		"⊘ lint                            falhou",
		"  7.0k/2.0k/0                     #519 ✗",
		"  terminou: incomplete",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("the agents pane drifted:\n got:\n%s\nwant:\n%s",
			strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// child.state changes the row it names and nothing else, with no reread of
// /children: the event carries the whole child.
func TestAChildStateChangesItsRowWithoutRereadingTheList(t *testing.T) {
	m := apply(t, orchestratorModel(t, En),
		started(t, 2, child("a", "index", "working", 100, 10, 0)),
		started(t, 3, child("b", "api", "working", 200, 20, 0)),
	)
	moved := child("a", "index", "pr", 5000, 500, 0)
	moved.PR = &protocol.ChildPR{Number: 7, CI: protocol.GitChecksPending}
	m = apply(t, m, ev(t, 4, protocol.EventChildState, protocol.ChildChanged{Child: moved}))

	if len(m.Agents.Children) != 2 {
		t.Fatalf("a state change added or lost a row: %d", len(m.Agents.Children))
	}
	if a := m.Agents.Children[0]; a.ID != "a" || a.State != "pr" || a.Usage.InputTokens != 5000 {
		t.Errorf("the row did not follow the event, or moved: %+v", a.Child)
	}
	if b := m.Agents.Children[1]; b.State != "working" {
		t.Errorf("a change to one child reached another: %+v", b.Child)
	}
	out := strings.Join(renderAgentsPane(m, agentsGeometry(), 40, 40), "\n")
	if !strings.Contains(out, "#7 …") {
		t.Errorf("the PR with its pending CI is not on the row:\n%s", out)
	}
}

// Closed leaves when the daemon's answer no longer brings it, not by a clock
// of the client's; failed stays, because failed waits for someone.
func TestAClosedChildLeavesWhenTheDaemonNoLongerListsItAndAFailedOneStays(t *testing.T) {
	m := apply(t, orchestratorModel(t, En),
		started(t, 2, child("gone", "old", "closed", 1, 1, 0)),
		started(t, 3, child("merged", "done", "merged", 1, 1, 0)),
		started(t, 4, child("broken", "lint", "failed", 1, 1, 0)),
		started(t, 5, child("live", "api", "working", 1, 1, 0)),
	)
	m = m.LoadAgents([]protocol.Child{child("live", "api", "working", 9, 9, 0)}, nil)

	var ids []string
	for _, a := range m.Agents.Children {
		ids = append(ids, a.ID)
	}
	if strings.Join(ids, ",") != "live,broken" {
		t.Errorf("after the daemon's answer the list is %v, want [live broken]", ids)
	}
	if m.Agents.Children[0].Usage.InputTokens != 9 {
		t.Error("the daemon's answer did not replace what the client had")
	}
}

// A question waits until it is answered, and the count says how many.
func TestAQuestionWaitsUntilItIsAnswered(t *testing.T) {
	m := apply(t, orchestratorModel(t, En),
		ev(t, 2, protocol.EventQuestionAsked, protocol.Question{ID: "q1", Kind: "merge"}),
	)
	out := strings.Join(renderAgentsPane(m, agentsGeometry(), 40, 40), "\n")
	if !strings.Contains(out, "1 question waits for you") {
		t.Errorf("one waiting question is not said:\n%s", out)
	}
	now := time.Now()
	m = apply(t, m, ev(t, 3, protocol.EventQuestionAnswered,
		protocol.Question{ID: "q1", Kind: "merge", Answer: "merge", Answered: &now}))
	if len(m.Agents.Questions) != 0 {
		t.Errorf("an answered question still waits: %+v", m.Agents.Questions)
	}
	m = m.LoadAgents(nil, []protocol.Question{{ID: "q2"}, {ID: "q3", Answered: &now}})
	if len(m.Agents.Questions) != 1 || m.Agents.Questions[0].ID != "q2" {
		t.Errorf("the daemon's list did not leave only the unanswered: %+v", m.Agents.Questions)
	}
}

// Review, gate and merge land on the child they name.
func TestPipelineEventsLandOnTheirChild(t *testing.T) {
	m := apply(t, orchestratorModel(t, PtBR),
		started(t, 2, child("a", "api", "in_review", 1, 1, 0)),
		ev(t, 3, protocol.EventReviewVerdict, protocol.ReviewVerdict{ChildID: "a",
			Verdict: protocol.Verdict{Commit: "c1", Verdict: "approve"}}),
		ev(t, 4, protocol.EventGateMeasured, protocol.GateMeasured{ChildID: "a",
			GateResult: protocol.GateResult{Commit: "c1", Result: "green"}}),
		ev(t, 5, protocol.EventChildCompleted, protocol.ChildCompleted{ChildID: "a", Reason: "done"}),
		ev(t, 6, protocol.EventMergeRefused, protocol.MergeRefused{ChildID: "a", Code: "main_red"}),
	)
	a := m.Agents.Children[0]
	if a.Review == nil || a.Review.Verdict != "approve" || a.Gate == nil || a.Gate.Result != "green" || a.Reason != "done" {
		t.Errorf("the pipeline did not land on the child: %+v", a.Child)
	}
	out := strings.Join(renderAgentsPane(m, agentsGeometry(), 40, 40), "\n")
	if !strings.Contains(out, "merge recusado: main_red") {
		t.Errorf("a refused merge is not said on its row:\n%s", out)
	}
	m = apply(t, m, ev(t, 7, protocol.EventMergeCompleted, protocol.MergeCompleted{ChildID: "a", Merged: "m1"}))
	if m.Agents.Children[0].Refused != nil {
		t.Error("a merge that went through still shows the refusal before it")
	}
}

// An event about a child the list has not heard of is not dropped: the row
// appears, its state said as unknown, until the child's own event fills it.
func TestAnEventForAnUnknownChildStillShowsTheChild(t *testing.T) {
	m := apply(t, orchestratorModel(t, En),
		ev(t, 2, protocol.EventGateMeasured, protocol.GateMeasured{ChildID: "x",
			GateResult: protocol.GateResult{Result: "red"}}),
	)
	if len(m.Agents.Children) != 1 || m.Agents.Children[0].ID != "x" {
		t.Fatalf("the gate result for an unseen child was dropped: %+v", m.Agents.Children)
	}
}

// A payload the client cannot read is said, never folded as nothing.
func TestAnUnreadableOrchestratorEventIsSaid(t *testing.T) {
	m := orchestratorModel(t, En)
	m = m.Apply(protocol.Event{Seq: 2, Type: protocol.EventChildState, Payload: []byte(`{"child":`)})
	if len(m.Entries) == 0 || m.Entries[len(m.Entries)-1].Kind != KindError {
		t.Fatalf("an unreadable child.state left no trace: %+v", m.Entries)
	}
	if !strings.Contains(m.Entries[len(m.Entries)-1].Summary, "child.state") {
		t.Errorf("the error does not name the event: %q", m.Entries[len(m.Entries)-1].Summary)
	}
}

// A state the client does not know is drawn as it came, never as one it knows.
func TestAnUnknownStateIsDrawnAsItCame(t *testing.T) {
	m := apply(t, orchestratorModel(t, En), started(t, 2, child("a", "api", "paused", 1, 1, 0)))
	out := strings.Join(renderAgentsPane(m, agentsGeometry(), 40, 40), "\n")
	if !strings.Contains(out, "paused") {
		t.Errorf("an unknown state was not drawn as it came:\n%s", out)
	}
}

// The orchestrator's column trades the diff for the list; an ordinary session
// keeps its diff. And the orchestrator, which never writes, still has a column.
func TestTheOrchestratorTradesTheDiffForTheList(t *testing.T) {
	g := agentsGeometry()
	g.RailMode = RailShown
	m := eightStates(t)
	out := Render(m, g)
	if !strings.Contains(out, "AGENTES") || strings.Contains(out, "[d] DIFF") {
		t.Errorf("the orchestrator's column is not the list:\n%s", out)
	}
	plain := NewModel("s", "/w", "m", "workspace-write", PtBR)
	plain.Window = 1000
	if out := Render(plain, g); strings.Contains(out, "AGENTES") {
		t.Errorf("an ordinary session drew the agents pane:\n%s", out)
	}
}

// More children than rows: the rest is counted, never cut silently.
func TestChildrenThatDoNotFitAreCounted(t *testing.T) {
	m := eightStates(t)
	got := renderAgentsPane(m, agentsGeometry(), 40, 8)
	if len(got) > 8 {
		t.Fatalf("the pane drew %d rows in 8", len(got))
	}
	if last := got[len(got)-1]; !strings.Contains(last, "+") {
		t.Errorf("the children that did not fit are not counted: %q", last)
	}
}

// In ASCII the eight states stay distinct by character, and no row passes the
// column's width.
func TestTheEightStatesStayDistinctInASCII(t *testing.T) {
	m := eightStates(t)
	g := agentsGeometry()
	g.Unicode = false
	got := renderAgentsPane(m, g, 40, 40)
	seen := map[string]bool{}
	for _, row := range got {
		if visibleWidth(row) > 40 {
			t.Errorf("row wider than the column: %q", row)
		}
		if strings.HasPrefix(row, " ") || strings.HasPrefix(row, "[") || strings.HasPrefix(row, "tokens") {
			continue
		}
		seen[strings.SplitN(row, " ", 2)[0]] = true
	}
	// Eight state marks plus the question line's.
	if len(seen) < 8 {
		t.Errorf("states share a mark in ASCII: %v", seen)
	}
	for _, row := range got {
		for _, r := range row {
			if r > 127 && !strings.ContainsRune("íçãõêéáó", r) {
				t.Errorf("a non-ASCII mark in ASCII mode: %q", row)
				break
			}
		}
	}
}
