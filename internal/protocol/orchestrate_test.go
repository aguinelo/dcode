package protocol

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// A session nobody orchestrates is the session of today, byte for byte: the
// four fields are additive only when an old client reading a new daemon sees
// none of them. TestGoldenSession pins the whole shape; this names the keys.
func TestASessionWithoutAParentEncodesAsToday(t *testing.T) {
	for name, v := range map[string]any{
		"session":      Session{ID: "s-1", Workspace: "/w", Model: "MiniMax-M3", CreatedAt: at},
		"conversation": Conversation{ID: "s-1", Workspace: "/w", Started: at, LastActivity: at},
	} {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"parent_id", "project_id", "role", "owns"} {
			if _, present := m[key]; present {
				t.Errorf("%s: %q is on the wire of a session without a parent: %s", name, key, b)
			}
		}
	}
}

func TestGoldenSessionWithAParent(t *testing.T) {
	golden(t, "session_child", Session{
		ID: "s-2", Workspace: "/p/children/s-2", Model: "MiniMax-M3",
		State: SessionStateRunning, SandboxMode: "workspace-write", Mode: ModeAssist,
		Branch: "dcode/index-s2a1b3", CreatedAt: at,
		ParentID: "s-1", ProjectID: "a1b2c3d4e5f6a1b2c3d4e5f6", Role: RoleChild,
		Owns: []string{"internal/index/", "changelog.d/202610171300-index.md"},
	})
	golden(t, "conversation_child", Conversation{
		ID: "s-2", Title: "index", Workspace: "/p/children/s-2", State: ConversationRecorded,
		Started: at, LastActivity: at,
		ParentID: "s-1", ProjectID: "a1b2c3d4e5f6a1b2c3d4e5f6", Role: RoleChild,
		Owns: []string{"internal/index/"},
	})
}

func TestGoldenProject(t *testing.T) {
	golden(t, "project", Project{
		ID: "a1b2c3d4e5f6a1b2c3d4e5f6", Root: "/src/dcode", Main: "main", Forge: "github",
		Autonomy: "step", OrchestratorID: "s-1", Created: at,
	})
}

func TestGoldenChild(t *testing.T) {
	ended := at.Add(time.Hour)
	golden(t, "child", Child{
		ID: "s-2", ParentID: "s-1", Title: "index", Profile: "fast", Model: "MiniMax-M3",
		State: "pr", Owns: []string{"internal/index/"},
		Branch: "dcode/index-s2a1b3", Worktree: "/o/p/children/s-2", Base: "6ec45f1",
		Head: "9a8b7c6", Returned: 1,
		Return: &Return{Why: "review", Phase: "fix", Paths: []string{"internal/index/a.go"}},
		Review: &Verdict{SessionID: "s-3", Commit: "9a8b7c6", Verdict: "approve"},
		Gate: &GateResult{
			Commit: "1f2e3d4", Tree: "5a6b7c8", Base: "6ec45f1", Command: "make check",
			DurationMS: 81000, Result: "green", At: at,
		},
		PR:      &ChildPR{Number: 518, URL: "https://github.com/o/r/pull/518", CI: "passing"},
		Usage:   Usage{InputTokens: 41000, OutputTokens: 2000},
		Started: at, Ended: &ended,
	})
	golden(t, "child_minimal", Child{
		ID: "s-2", ParentID: "s-1", Title: "index", State: "dispatched",
		Owns: []string{"internal/index/"}, Branch: "dcode/index-s2a1b3",
		Worktree: "/o/p/children/s-2", Base: "6ec45f1", Started: at,
	})
}

func TestGoldenQuestion(t *testing.T) {
	golden(t, "question", Question{
		ID: "q-1", Kind: "merge", Text: "Merge #518?", Recommended: "merge",
		Why: "gate green on 1f2e3d4", ChildID: "s-2",
		Options: []QuestionOption{{ID: "merge", Label: "Merge"}, {ID: "return", Label: "Return"}},
		Asked:   at,
	})
}

// Each new code has a status. HTTPStatus answers 500 for a code it does not
// know, so a code added without its line would ship as an internal failure.
func TestEveryOrchestrationCodeHasItsStatus(t *testing.T) {
	for code, want := range map[string]int{
		CodeProjectNotFound:         http.StatusNotFound,
		CodeOrchestratorUnavailable: http.StatusServiceUnavailable,
		CodeOwnsOverlap:             http.StatusConflict,
		CodeOutsideOwns:             http.StatusConflict,
		CodeMaxChildrenReached:      http.StatusServiceUnavailable,
		CodeModelNotAllowed:         http.StatusBadRequest,
		CodeMainRed:                 http.StatusConflict,
		CodeGateStale:               http.StatusConflict,
		CodeGateUndeclared:          http.StatusConflict,
		CodeCeilingReached:          http.StatusConflict,
		CodeChildInPipeline:         http.StatusConflict,
		CodeQuestionNotFound:        http.StatusNotFound,
		CodeQuestionAlreadyAnswered: http.StatusConflict,
	} {
		if _, mapped := httpStatus[code]; !mapped {
			t.Errorf("%s has no line in httpStatus", code)
		}
		if got := HTTPStatus(code); got != want {
			t.Errorf("%s: got %d want %d", code, got, want)
		}
	}
}

// A role nobody defined is not a role: a session read as "orchestrator" by a
// typo would be handed what only the orchestrator may do.
func TestOnlyTheThreeRolesAreRoles(t *testing.T) {
	for _, r := range []string{RoleOrchestrator, RoleChild, RoleReviewer} {
		if !ValidRole(r) {
			t.Errorf("%q should be a role", r)
		}
	}
	for _, r := range []string{"", "Child", "parent", "orchestrator "} {
		if ValidRole(r) {
			t.Errorf("%q should not be a role", r)
		}
	}
}

func TestOrchestrationEventNames(t *testing.T) {
	for got, want := range map[EventType]string{
		EventChildStarted:     "child.started",
		EventChildState:       "child.state",
		EventChildCompleted:   "child.completed",
		EventReviewVerdict:    "review.verdict",
		EventGateMeasured:     "gate.measured",
		EventQuestionAsked:    "question.asked",
		EventQuestionAnswered: "question.answered",
		EventQuestionCard:     "question.card",
		EventMergeCompleted:   "merge.completed",
		EventMergeRefused:     "merge.refused",
		EventMainHealth:       "main.health",
	} {
		if string(got) != want {
			t.Errorf("got %q want %q", got, want)
		}
	}
}

// The verdict and the gate travel flat with the child they are about, so a
// client reads verdict.findings, not verdict.verdict.findings.
func TestAVerdictAndAGateCarryTheirChildFlat(t *testing.T) {
	golden(t, "review_verdict", ReviewVerdict{
		ChildID: "s-2",
		Verdict: Verdict{SessionID: "s-3", Commit: "9a8b7c6", Verdict: "changes_requested",
			Findings: []Finding{{Path: "internal/index/a.go", Line: 12, What: "no test", Why: "TESTING.md"}}},
	})
	golden(t, "gate_measured", GateMeasured{
		ChildID: "s-2",
		GateResult: GateResult{Commit: "1f2e3d4", Tree: "5a6b7c8", Base: "6ec45f1",
			Command: "make check", ExitCode: 2, DurationMS: 4000, Result: "red",
			OutputTail: "FAIL", OutputPath: "/o/p/gates/s-2-1f2e3d4.log", At: at},
	})
}

func TestOrchestrationRoundTrip(t *testing.T) {
	answered := at.Add(time.Minute)
	roundTrip(t, Question{ID: "q", Kind: "ceiling", Options: []QuestionOption{{ID: "stop", Label: "Stop"}},
		Asked: at, Answer: "stop", Answered: &answered})
	roundTrip(t, ChildCompleted{ChildID: "s-2", TurnID: "t", Reason: StopDone, Conclusion: "ok",
		Wrote: []string{"a.go"}, Usage: Usage{InputTokens: 1}})
	roundTrip(t, MergeCompleted{ChildID: "s-2", Method: "squash", PR: 518, Merged: "abc", Measured: "def"})
	roundTrip(t, MergeRefused{ChildID: "s-2", Code: CodeMainRed, Reason: "main is red"})
	roundTrip(t, ChildChanged{Child: Child{ID: "s-2", State: "working", Started: at}})
	roundTrip(t, MainHealth{Commit: "abc", Health: "pending", Checked: at})
	roundTrip(t, OpenProjectRequest{Workspace: "/w"})
	roundTrip(t, AnswerQuestionRequest{Answer: "merge", Note: "go"})
	roundTrip(t, ListProjectsResponse{Projects: []Project{{ID: "p", Created: at}}})
	roundTrip(t, ListChildrenResponse{Children: []Child{}})
	roundTrip(t, ListQuestionsResponse{Questions: []Question{}, Cards: []QuestionCard{}})
	roundTrip(t, AnswerQuestionsRequest{CardID: "c1", Answers: []AnswerItem{{QuestionID: "q1", Answer: "merge"}}, Skipped: []string{"q2"}})
	roundTrip(t, AnswerQuestionsResponse{Results: []AnswerResult{{QuestionID: "q1", Status: AnswerRefused,
		Code: CodeQuestionAlreadyAnswered, Message: "first wins"}}, Cards: []QuestionCard{}})
}

// A card carries everything its rendering needs — the order of its items,
// each question flat with its header, options and their descriptions, the
// recommendation, the selection, the person's own words allowed, the
// orchestrator's summary and the log's evidence — and how many still wait
// (.p §5.5).
func TestACardCarriesItsItemsFlatWithWhatTheyNeed(t *testing.T) {
	golden(t, "question_card", QuestionCard{ID: "c1", Kind: "merge", Title: "Merges ready", Pending: 1,
		Items: []CardItem{{
			Question: Question{ID: "q-1", Header: "index", Kind: "merge", Text: "Merge \"index\"?", Recommended: "merge",
				Why: "approved and green", ChildID: "s-2", AllowOther: true, Summary: "Conta palavras.\nCom teste.",
				Options: []QuestionOption{{ID: "merge", Label: "Merge", Description: "Into main."}, {ID: "return", Label: "Return"}},
				Evidence: &QuestionEvidence{Verdict: "approve", Reviewer: "s-3", ReviewedCommit: "1f2e3d4", ReviewSeq: 7,
					Gate: "green", Command: "make check", Commit: "1f2e3d4", Base: "6ec45f1", DurationMS: 4000, GateSeq: 9},
				Asked: at},
			State: CardItemPending, Selected: "merge",
		}},
		Created: at, Updated: at})
}
