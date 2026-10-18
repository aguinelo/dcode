package tools

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// Spec: orchestration .i step 6 — `git push` and `git rebase` in a child's
// bash are denied without an approval being asked, with the why. The denial
// is in the declaration: there is no request to put to anyone.
func TestGitThatWritesIsDeniedToAChildWithoutAsking(t *testing.T) {
	b := Bash{Deny: DenyGitWrite}
	for _, cmd := range []string{
		"git push origin main", "git rebase main", "cd x && git commit -m y", "GIT_DIR=. git -C . reset --hard",
		"echo $(git stash)", "sudo -E git checkout -b x", "/usr/bin/git branch -D x", "git -c a=b config x y",
		"true; git worktree add ../x", "git tag v1", "git switch main", "git merge x",
	} {
		_, err := b.Declare(mustJSON(t, BashInput{Command: cmd}))
		te, ok := err.(*ToolError)
		if !ok || te.Code != CodeDenied || !strings.Contains(te.Reason, "harness") {
			t.Errorf("%q declared %v, want a denial saying why", cmd, err)
		}
	}
	for _, cmd := range []string{"git status", "git log -1", "git diff", "go test ./...", "echo commit", "git show HEAD"} {
		if _, err := b.Declare(mustJSON(t, BashInput{Command: cmd})); err != nil {
			t.Errorf("%q was denied: %v", cmd, err)
		}
	}
	if _, err := (Bash{}).Declare(mustJSON(t, BashInput{Command: "git push"})); err != nil {
		t.Errorf("a session that is no child denied git push: %v", err)
	}
}

type fakeConductor struct {
	items     []DelegateItem
	asked     []AskInput
	summaries []string
	proposed  []ProposeInput
	err       error
}

func (f *fakeConductor) Delegate(items []DelegateItem) ([]protocol.Child, error) {
	f.items = items
	if f.err != nil {
		return nil, f.err
	}
	return []protocol.Child{{ID: "c1", Title: "index", State: "working", Branch: "dcode/index-abcdef", Owns: []string{"a"}}}, nil
}
func (f *fakeConductor) Children() ([]protocol.Child, error)     { return nil, nil }
func (f *fakeConductor) SteerChild(string, string, string) error { return f.err }
func (f *fakeConductor) CloseChild(string, string) error         { return f.err }
func (f *fakeConductor) Ask(in AskInput) (string, error) {
	f.asked = append(f.asked, in)
	return "q1", f.err
}
func (f *fakeConductor) Summarize(qid, summary string) error {
	f.summaries = append(f.summaries, qid+": "+summary)
	return f.err
}
func (f *fakeConductor) ProposeAnswers(in ProposeInput) (string, error) {
	f.proposed = append(f.proposed, in)
	return "c2", f.err
}

// Spec: orchestration .p §10 — summarize hands the orchestrator's lines to
// the question they are about; a refusal is said with its code.
func TestSummarizeAttachesTheLinesToTheQuestion(t *testing.T) {
	f := &fakeConductor{}
	in := json.RawMessage(`{"question_id":"q-1","summary":"Conta palavras.\nCom teste."}`)
	res, _ := SummarizeTool{Conductor: f}.Execute(context.Background(), in, nil)
	if res.IsError || len(f.summaries) != 1 || f.summaries[0] != "q-1: Conta palavras.\nCom teste." {
		t.Fatalf("summarize = %+v; %v", res, f.summaries)
	}
	f.err = protocol.Errorf(protocol.CodeQuestionAlreadyAnswered, "answered")
	if res, _ := (SummarizeTool{Conductor: f}).Execute(context.Background(), in, nil); !res.IsError || res.Code != protocol.CodeQuestionAlreadyAnswered {
		t.Errorf("a refused summary = %+v", res)
	}
	if res, _ := (SummarizeTool{}).Execute(context.Background(), in, nil); !res.IsError {
		t.Errorf("summarize without a project = %+v", res)
	}
}

// Spec: orchestration .p §5.5, §10 — propose_answers hands the person's words
// and the answers understood over as a proposal, and says that nothing is
// answered until the person confirms it.
func TestProposeAnswersIsAProposalAndSaysSo(t *testing.T) {
	f := &fakeConductor{}
	in := json.RawMessage(`{"from":"merge em todas","why":"the person said so","answers":[{"question_id":"q-1","answer":"merge"},{"question_id":"q-2","answer":"return","note":"more tests"}]}`)
	res, _ := ProposeAnswersTool{Conductor: f}.Execute(context.Background(), in, nil)
	if res.IsError || len(f.proposed) != 1 || f.proposed[0].From != "merge em todas" || len(f.proposed[0].Answers) != 2 ||
		f.proposed[0].Answers[1].Note != "more tests" || !strings.Contains(res.Output, "c2") || !strings.Contains(res.Output, "nothing is answered") {
		t.Fatalf("propose_answers = %+v; %+v", res, f.proposed)
	}
	if d := (ProposeAnswersTool{}).Description(); !strings.Contains(d, "never answers") {
		t.Errorf("propose_answers' description does not say it never answers:\n%s", d)
	}
}

// Spec: orchestration .p §5.5, §13 — the orchestrator cannot answer: no
// method of what its tools reach answers a question, so no tool can.
func TestNoToolOfTheOrchestratorReachesAnAnswer(t *testing.T) {
	ct := reflect.TypeOf((*Conductor)(nil)).Elem()
	for i := range ct.NumMethod() {
		if name := ct.Method(i).Name; strings.HasPrefix(name, "Answer") {
			t.Errorf("Conductor.%s: a tool of the orchestrator's would reach an answer", name)
		}
	}
	if _, ok := ct.MethodByName("ProposeAnswers"); !ok {
		t.Error("the orchestrator has no way to propose what the person said")
	}
}

// Spec: orchestration .p §10 — ask puts a question to the person and returns
// at once with its id; a refusal is said with its code.
func TestAskQueuesTheQuestionAndReturnsAtOnce(t *testing.T) {
	f := &fakeConductor{}
	in := json.RawMessage(`{"text":"which?","options":[{"id":"a","label":"A"},{"id":"b","label":"B"}],"recommended":"a","why":"w"}`)
	res, _ := AskTool{Conductor: f}.Execute(context.Background(), in, nil)
	if res.IsError || !strings.Contains(res.Output, "q1") || len(f.asked) != 1 || f.asked[0].Recommended != "a" {
		t.Fatalf("ask = %+v; asked %+v", res, f.asked)
	}
	f.err = protocol.Errorf(protocol.CodeInvalidInput, "no")
	if res, _ := (AskTool{Conductor: f}).Execute(context.Background(), in, nil); !res.IsError {
		t.Errorf("a refused question = %+v", res)
	}
	if res, _ := (AskTool{}).Execute(context.Background(), in, nil); !res.IsError {
		t.Errorf("ask without a project = %+v", res)
	}
}

func TestDelegateHandsTheBatchOverAndSaysARefusalWithItsCode(t *testing.T) {
	f := &fakeConductor{}
	in := json.RawMessage(`{"items":[{"title":"index","task":"t","owns":["a"],"why":"w","pr_title":"p"}]}`)
	res, err := DelegateTool{Conductor: f}.Execute(context.Background(), in, nil)
	if err != nil || res.IsError || !strings.Contains(res.Output, "c1  working") || f.items[0].PRTitle != "p" {
		t.Fatalf("delegate = %+v, %v; items %+v", res, err, f.items)
	}
	f.err = protocol.Errorf(protocol.CodeOwnsOverlap, "a crosses a")
	res, _ = DelegateTool{Conductor: f}.Execute(context.Background(), in, nil)
	if !res.IsError || res.Code != protocol.CodeOwnsOverlap || !strings.Contains(res.Output, protocol.CodeOwnsOverlap) {
		t.Errorf("a refusal = %+v", res)
	}
	res, _ = SteerChildTool{Conductor: f}.Execute(context.Background(), json.RawMessage(`{"child_id":"c1","text":"x","why":"y"}`), nil)
	if !res.IsError {
		t.Errorf("steer_child refusal = %+v", res)
	}
}

// The description is the only place an orchestrator reads about delegate at
// the point of use, so it has to stand on its own: who writes, what owns is,
// and that the harness, not the orchestrator, carries the change to main.
func TestDelegateSaysChildrenWriteAndWhatOwnsIs(t *testing.T) {
	d := DelegateTool{}.Description()
	for _, want := range []string{"you do not write, they do", "own git worktree", "write access to the paths in its `owns`",
		"relative to the project's root", "which no other child may own", "one item per theme",
		"the harness commits it, has it reviewed, runs the gate and asks the person to merge"} {
		if !strings.Contains(d, want) {
			t.Errorf("delegate's description does not say %q:\n%s", want, d)
		}
	}
}
