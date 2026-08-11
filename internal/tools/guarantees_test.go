package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The two invariants in this file are the ones TestEveryInvariantHasATest found
// nothing claiming. Both are cross-cutting — they are properties of the suite
// rather than of any one tool — which is exactly why each individual tool's
// tests kept not covering them.

// A machine-varying value in the output is invisible damage: the history is
// part of the cached prefix, so an absolute path or a clock reading makes two
// runs of the same session diverge, and a transcript stops being reproducible
// (RN-7 of context-engine). The exception the spec allows is when the varying
// value IS the answer — a tool asked for a timestamp may return one — and no
// tool here is.
func TestNoOutputCarriesAClockOrAnAbsolutePath(t *testing.T) {
	s, ws := setup(t)
	writeFileT(t, ws, "pkg/stats.go", "package pkg\n\nfunc Count() int { return 1 }\n")
	writeFileT(t, ws, "README.md", "# title\n")

	cases := []struct {
		tool Tool
		args any
	}{
		{Read{}, ReadInput{Path: "pkg/stats.go"}},
		{Glob{}, GlobInput{Pattern: "**/*.go"}},
		{Grep{}, GrepInput{Pattern: "Count"}},
		{Symbol{}, SymbolInput{Name: "Count"}},
		{Write{}, WriteInput{Path: "new.txt", Content: "x\n"}},
	}

	// A clock in any of the shapes a Go program produces one by accident:
	// time.Now().String(), RFC3339, or a bare wall clock.
	clock := regexp.MustCompile(`\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}`)

	for _, c := range cases {
		res := run(t, c.tool, s, c.args)
		if strings.Contains(res.Output, ws) {
			t.Errorf("%s: output carries the absolute workspace path:\n%s", c.tool.Name(), res.Output)
		}
		if m := clock.FindString(res.Output); m != "" {
			t.Errorf("%s: output carries a clock reading %q:\n%s", c.tool.Name(), m, res.Output)
		}
	}
}

// RN-6: nothing runs on a verdict that was never asked for.
//
// Two halves, because neither alone is the guarantee. Every tool must DECLARE
// something the evaluator can rule on — a tool that declared an empty request
// would pass any policy while touching whatever it likes. And the loop must
// have exactly one place where a tool runs, with the evaluation above it: a
// second call site is how an exemption gets added for one tool and inherited by
// none of the reasoning that justified the first.
func TestEveryToolPassesThroughPolicy(t *testing.T) {
	all := []Tool{Read{}, Write{}, Edit{}, Glob{}, Grep{}, Bash{}, Plan{}, Symbol{}, Explore{}}

	for _, tool := range all {
		args := map[string]any{
			"path": "a.go", "pattern": "x", "name": "X", "task": "t",
			"content": "x", "old_string": "a", "new_string": "b",
			"command": "true", "items": []any{},
		}
		b, err := json.Marshal(args)
		if err != nil {
			t.Fatal(err)
		}
		req, err := tool.Declare(b)
		if err != nil {
			t.Errorf("%s: Declare refused a well-formed input: %v", tool.Name(), err)
			continue
		}
		// plan declares no path and no network on purpose; what it may not do is
		// decline to name itself, because the verdict is recorded against a tool.
		if req.Tool != tool.Name() {
			t.Errorf("%s: declares itself as %q; a verdict recorded against the wrong "+
				"subject is a verdict nobody can audit", tool.Name(), req.Tool)
		}
	}

	src, err := os.ReadFile(filepath.Join("..", "loop", "turn.go"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	if n := strings.Count(body, "tool.Execute("); n != 1 {
		t.Fatalf("the loop runs a tool from %d places; one of them is an ungated path "+
			"waiting to happen", n)
	}
	gate := strings.Index(body, "e.evaluate(")
	exec := strings.Index(body, "tool.Execute(")
	if gate < 0 || gate > exec {
		t.Error("the only execution site is not preceded by the evaluator")
	}
}
