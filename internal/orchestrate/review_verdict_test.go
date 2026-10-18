package orchestrate

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/tools"
)

const reviewHead = "1111111111111111111111111111111111111111"

func testReview() *Review {
	return NewReview(ReviewRequest{ChildID: "c1", Commit: reviewHead}, "rev-1")
}

func measuredVerdict(t *testing.T, d Decision) protocol.Verdict {
	t.Helper()
	var v protocol.Verdict
	if err := json.Unmarshal(d.Measured, &v); err != nil {
		t.Fatalf("measured %s: %v", d.Measured, err)
	}
	return v
}

// Spec: orchestration .i step 8 — the reviewer's registry has no tool that
// writes: read, glob, grep, symbol and verdict, and nothing else.
func TestTheReviewerRegistryHasNoToolThatWrites(t *testing.T) {
	reg := ReviewerRegistry(tools.Verdict{})
	if got, want := reg.Names(), []string{"glob", "grep", "read", "symbol", "verdict"}; !slices.Equal(got, want) {
		t.Fatalf("the reviewer has %v, want %v", got, want)
	}
	inputs := map[string]string{
		"read": `{"path":"a.go"}`, "glob": `{"pattern":"**/*.go"}`, "grep": `{"pattern":"x"}`,
		"symbol": `{"name":"X"}`, "verdict": `{"verdict":"approve","why":"fine"}`,
	}
	for _, name := range reg.Names() {
		tool, _ := reg.Get(name)
		req, err := tool.Declare(json.RawMessage(inputs[name]))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if req.Command != "" || req.Network {
			t.Errorf("%s declares a command or the network: %+v", name, req)
		}
		for _, a := range req.Paths {
			if a.Write {
				t.Errorf("%s declares a write to %s", name, a.Path)
			}
		}
	}
}

// Spec: orchestration .p §7.2 — a review that ends without verdict is missing,
// recorded, and an error: never an approval.
func TestAReviewThatEndsWithoutAVerdictIsMissingNeverAnApproval(t *testing.T) {
	dir := t.TempDir()
	l := openLog(t, dir, testProject(t))
	v, err := testReview().Finish(l)
	if !errors.Is(err, ErrNoVerdict) {
		t.Fatalf("Finish without a verdict = %+v, %v; want ErrNoVerdict", v, err)
	}
	if v.Verdict != VerdictMissing || v.Commit != reviewHead || v.SessionID != "rev-1" {
		t.Errorf("verdict = %+v, want missing on %s by rev-1", v, reviewHead)
	}
	lines := readLines(t, dir)
	if len(lines) != 1 || lines[0].Kind != "review.verdict" || lines[0].Child != "c1" ||
		measuredVerdict(t, lines[0]).Verdict != VerdictMissing || !strings.Contains(lines[0].Why, "without calling verdict") {
		t.Errorf("log = %+v, want one review.verdict saying missing and why", lines)
	}
}

// Spec: orchestration .p §5.3 and §9.1 — review.verdict carries the verdict
// and the reviewer's why.
func TestAVerdictIsRecordedWithTheReviewersReason(t *testing.T) {
	dir := t.TempDir()
	l := openLog(t, dir, testProject(t))
	r := testReview()
	tool := tools.Verdict{Submit: r.Submit}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"verdict":"changes_requested",`+
		`"findings":[{"path":"internal/index/index.go","line":3,"what":"counts bytes","why":"the task says words"}],`+
		`"why":"the count is of bytes, not of words"}`), nil)
	if err != nil || res.IsError {
		t.Fatalf("verdict = %+v, %v", res, err)
	}
	v, err := r.Finish(l)
	if err != nil {
		t.Fatal(err)
	}
	if v.Verdict != tools.VerdictChanges || len(v.Findings) != 1 || v.Findings[0].Line != 3 {
		t.Errorf("verdict = %+v", v)
	}
	lines := readLines(t, dir)
	if len(lines) != 1 || lines[0].Kind != "review.verdict" || lines[0].Session != "rev-1" ||
		lines[0].Why != "the count is of bytes, not of words" || len(measuredVerdict(t, lines[0]).Findings) != 1 {
		t.Errorf("log = %+v", lines)
	}
	if _, err := r.Finish(l); err == nil {
		t.Error("a review finished twice")
	}
}

func TestAVerdictThatDoesNotHoldTogetherIsRefused(t *testing.T) {
	finding := []tools.VerdictFinding{{Path: "a.go", What: "w", Why: "y"}}
	for name, in := range map[string]tools.VerdictInput{
		"changes without findings":  {Verdict: tools.VerdictChanges, Why: "bad"},
		"approve with findings":     {Verdict: tools.VerdictApprove, Findings: finding, Why: "ok"},
		"no why":                    {Verdict: tools.VerdictApprove},
		"missing said by the model": {Verdict: VerdictMissing, Why: "x"},
		"a finding without why":     {Verdict: tools.VerdictChanges, Findings: []tools.VerdictFinding{{Path: "a.go", What: "w"}}, Why: "bad"},
	} {
		if _, err := testReview().Submit(context.Background(), in); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	r := testReview()
	if _, err := r.Submit(context.Background(), tools.VerdictInput{Verdict: tools.VerdictApprove, Why: "ok"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Submit(context.Background(), tools.VerdictInput{Verdict: tools.VerdictChanges, Findings: finding, Why: "no"}); err == nil {
		t.Error("a second verdict replaced the first")
	}
	res, _ := tools.Verdict{}.Execute(context.Background(), json.RawMessage(`{"verdict":"approve","why":"x"}`), nil)
	if !res.IsError {
		t.Error("a verdict with no review to give it to was accepted")
	}
}

// Spec: orchestration .config §2 and §3 — reviewer_profiles is an allow-list,
// and review_max_diff_bytes is bounded.
func TestTheReviewerRunsOnAProfileOfItsAllowList(t *testing.T) {
	known := func(p string) bool { return p == "strong" || p == "fast" }
	c, err := ReviewFrom(resolved(map[string]string{"orchestrator.reviewer_profiles": " strong, fast "}), known)
	if err != nil || c.MaxDiffBytes != reviewMaxDiffDefault {
		t.Fatalf("ReviewFrom = %+v, %v", c, err)
	}
	if p, err := c.Profile("", "child"); err != nil || p != "strong" {
		t.Errorf("Profile(\"\") = %q, %v; want the first of the list", p, err)
	}
	if p, err := c.Profile("fast", "child"); err != nil || p != "fast" {
		t.Errorf("Profile(fast) = %q, %v", p, err)
	}
	var pe *protocol.Error
	if _, err := c.Profile("child", "child"); !errors.As(err, &pe) || pe.Code != protocol.CodeModelNotAllowed {
		t.Errorf("Profile(child) = %v, want model_not_allowed", err)
	}
	empty, err := ReviewFrom(resolved(nil), known)
	if err != nil {
		t.Fatal(err)
	}
	if p, err := empty.Profile("", "child"); err != nil || p != "child" {
		t.Errorf("an empty list = %q, %v; want the fallback, the first child profile", p, err)
	}
	if _, err := empty.Profile("", ""); err == nil {
		t.Error("a reviewer with no profile at all was chosen")
	}

	for key, v := range map[string]string{
		"orchestrator.reviewer_profiles":     "strong,nope",
		"orchestrator.review_max_diff_bytes": "100",
	} {
		_, err := ReviewFrom(resolved(map[string]string{key: v}), known)
		if err == nil || !strings.Contains(err.Error(), key) || !strings.Contains(err.Error(), v) {
			t.Errorf("%s = %q: %v, want an error naming the key and the value", key, v, err)
		}
	}
	c, err = ReviewFrom(resolved(map[string]string{"orchestrator.review_max_diff_bytes": "1048576"}), known)
	if err != nil || c.MaxDiffBytes != 1048576 {
		t.Errorf("the ceiling itself = %+v, %v", c, err)
	}
}
