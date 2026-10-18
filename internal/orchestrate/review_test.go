package orchestrate

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// reviewRepo is a child's worktree: a base commit with a spec and a file the
// child owns, and a head commit that changes what changes maps to. It returns
// the repository and both full SHAs.
func reviewRepo(t *testing.T, changes map[string]string) (repo, base, head string) {
	t.Helper()
	repo = filepath.Join(t.TempDir(), "child")
	put := func(files map[string]string) {
		for name, body := range files {
			p := filepath.Join(repo, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	put(map[string]string{
		"docs/specs/index/202610010000-index.p.spec.md": "# the index spec\n",
		"internal/index/index.go":                       "package index\n",
		"internal/other/other.go":                       "package other\n",
	})
	git(t, repo, "init", "-q")
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-q", "-m", "base")
	base = revParse(t, repo, "HEAD")
	put(changes)
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-q", "-m", "head")
	return repo, base, revParse(t, repo, "HEAD")
}

func reviewInput(repo, base, head string) ReviewInput {
	return ReviewInput{
		ChildID: "c1", Worktree: repo, Base: base, Head: head,
		Task: "Make the index count words. Follow docs/specs/index/202610010000-index.p.spec.md " +
			"and 202610010000-index.r.spec.md.",
		Owns:         []string{"internal/index/"},
		MaxDiffBytes: reviewMaxDiffDefault,
	}
}

// Spec: orchestration .i step 8 — a diff with one file outside owns is
// Return{outside_owns}, without a reviewer: nothing is asked of a model.
func TestADiffOutsideOwnsReturnsWithoutAReview(t *testing.T) {
	repo, base, head := reviewRepo(t, map[string]string{
		"internal/index/index.go": "package index\n\nfunc Count() int { return 0 }\n",
		"internal/other/other.go": "package other\n\nvar Touched = true\n",
	})
	req, err := AssembleReview(context.Background(), reviewInput(repo, base, head))
	var v *OwnsViolation
	if !errors.As(err, &v) {
		t.Fatalf("AssembleReview = %+v, %v; want an OwnsViolation", req, err)
	}
	ret := v.Return()
	if ret.Why != protocol.CodeOutsideOwns || len(ret.Paths) != 1 || ret.Paths[0] != "internal/other/other.go" {
		t.Errorf("Return = %+v, want outside_owns with internal/other/other.go alone", ret)
	}
	if req.Prompt() != "" || req.Diff != "" {
		t.Errorf("a review was assembled for a diff outside owns: %+v", req)
	}
}

// Spec: orchestration .p §7.1 — a path outside owns in git status is returned
// before any commit; what is inside is listed for the commit by name.
func TestAPathOutsideOwnsInTheStatusIsReturnedBeforeAnyCommit(t *testing.T) {
	repo, _, _ := reviewRepo(t, map[string]string{"internal/index/index.go": "package index\n// one\n"})
	for name, body := range map[string]string{
		"internal/index/new.go":   "package index\n",
		"internal/index/index.go": "package index\n// two\n",
		"scripts/sneaky.sh":       "echo written by bash\n",
	} {
		p := filepath.Join(repo, filepath.FromSlash(name))
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	changed, err := Uncommitted(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"internal/index/index.go", "internal/index/new.go", "scripts/sneaky.sh"}
	if strings.Join(changed, ",") != strings.Join(want, ",") {
		t.Errorf("Uncommitted = %v, want %v", changed, want)
	}
	if out := OutsideOwns([]string{"internal/index/"}, changed); len(out) != 1 || out[0] != "scripts/sneaky.sh" {
		t.Errorf("OutsideOwns = %v, want scripts/sneaky.sh alone", out)
	}
}

func TestOwnsAreAGlobADirectoryOrAFile(t *testing.T) {
	owns := []string{"internal/index/", "cmd/**/*.go", "README.md"}
	for path, want := range map[string]bool{
		"internal/index/a.go": true, "internal/index/deep/b.go": true, "internal/indexer/a.go": false,
		"cmd/dcode/main.go": true, "cmd/dcode/main.md": false, "README.md": true, "docs/README.md": false,
	} {
		if got := len(OutsideOwns(owns, []string{path})) == 0; got != want {
			t.Errorf("%s owned = %v, want %v", path, got, want)
		}
	}
	if out := OutsideOwns(nil, []string{"a.go"}); len(out) != 1 {
		t.Errorf("no owns owns nothing; got %v outside", out)
	}
}

// Spec: orchestration .p §7.2 — the reviewer receives the task, the owns, the
// diff Base..Head and the specs the task names.
func TestTheReviewCarriesTheTaskTheOwnsTheDiffAndTheSpecsTheTaskNames(t *testing.T) {
	repo, base, head := reviewRepo(t, map[string]string{
		"internal/index/index.go": "package index\n\nfunc CountWords() int { return 42 }\n",
	})
	req, err := AssembleReview(context.Background(), reviewInput(repo, base, head))
	if err != nil {
		t.Fatal(err)
	}
	if req.Commit != head || req.Base != base || req.DiffCut || !strings.Contains(req.Diff, "+func CountWords() int") {
		t.Errorf("request = %+v; want the whole diff of %s..%s", req, base, head)
	}
	if len(req.Files) != 1 || req.Files[0] != "internal/index/index.go" {
		t.Errorf("Files = %v", req.Files)
	}
	if len(req.Specs) != 1 || req.Specs[0] != "docs/specs/index/202610010000-index.p.spec.md" {
		t.Errorf("Specs = %v, want the spec the task names and the commit has", req.Specs)
	}
	if len(req.MissingSpecs) != 1 || req.MissingSpecs[0] != "202610010000-index.r.spec.md" {
		t.Errorf("MissingSpecs = %v, want the named spec the commit does not have, said", req.MissingSpecs)
	}
	p := req.Prompt()
	for _, want := range []string{"Make the index count words.", "internal/index/", head,
		"docs/specs/index/202610010000-index.p.spec.md", "202610010000-index.r.spec.md",
		"+func CountWords() int", "verdict"} {
		if !strings.Contains(p, want) {
			t.Errorf("the prompt does not carry %q:\n%s", want, p)
		}
	}
}

// Spec: orchestration .p §7.2 and .config §3 — a diff over review_max_diff_bytes
// is cut whole and said, with the files to read; never a silent truncation.
func TestADiffOverTheLimitIsCutAndSaidNeverTruncated(t *testing.T) {
	big := "package index\n\n" + strings.Repeat("var x = \"a line that makes the diff long\"\n", 300)
	repo, base, head := reviewRepo(t, map[string]string{"internal/index/index.go": big})
	in := reviewInput(repo, base, head)
	in.MaxDiffBytes = reviewMaxDiffMin
	req, err := AssembleReview(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if !req.DiffCut || req.Diff != "" || req.DiffBytes <= reviewMaxDiffMin {
		t.Fatalf("DiffCut = %v, %d bytes kept of %d; want the diff cut whole", req.DiffCut, len(req.Diff), req.DiffBytes)
	}
	p := req.Prompt()
	if strings.Contains(p, "a line that makes the diff long") {
		t.Error("part of a cut diff reached the prompt, which would read as the whole of it")
	}
	if !strings.Contains(p, "diff cut") || !strings.Contains(p, "internal/index/index.go") {
		t.Errorf("the prompt does not say the diff was cut and which files to read:\n%s", p)
	}
	in.MaxDiffBytes = 0
	if _, err := AssembleReview(context.Background(), in); err == nil {
		t.Error("a review with no diff limit was assembled")
	}
}

func TestAReviewNeedsExactCommits(t *testing.T) {
	repo, base, _ := reviewRepo(t, map[string]string{"internal/index/index.go": "package index\n// x\n"})
	in := reviewInput(repo, base, "HEAD")
	if _, err := AssembleReview(context.Background(), in); err == nil {
		t.Error("a review of a ref, not a commit, was assembled")
	}
}
