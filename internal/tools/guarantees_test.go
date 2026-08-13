package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/policy"
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

	// A failing call leaks more easily than a successful one, and a model
	// reads an error harder than it reads a result. `read` on a directory
	// wrapped the raw Go error and printed the whole temp path; the model then
	// tried to read that path, which is a round spent on something the tool
	// told it to do.
	cases = append(cases,
		struct {
			tool Tool
			args any
		}{Read{}, ReadInput{Path: "."}},
		struct {
			tool Tool
			args any
		}{Read{}, ReadInput{Path: "pkg"}},
		struct {
			tool Tool
			args any
		}{Read{}, ReadInput{Path: "nope.go"}},
		struct {
			tool Tool
			args any
		}{Edit{}, EditInput{Path: "pkg/stats.go", OldString: "x", NewString: "y"}},
	)

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

// Reading a directory is a thing a model does constantly, and the answer was
// the raw Go error: "read /private/var/folders/.../001: is a directory". It
// leaked the path, and it never said what to do instead — so the model tried
// reading that path next, and spent the round finding out it was the same
// directory.
//
// Every other error in this suite carries a way forward. This one is the
// commonest and it carried none.
func TestReadingADirectorySaysToUseGlob(t *testing.T) {
	s, ws := setup(t)
	writeFileT(t, ws, "pkg/stats.go", "package pkg\n")

	for _, path := range []string{".", "pkg"} {
		res := run(t, Read{}, s, ReadInput{Path: path})
		if !res.IsError {
			t.Fatalf("reading the directory %q succeeded", path)
		}
		if strings.Contains(res.Output, ws) {
			t.Errorf("read(%q) leaked the workspace path: %s", path, res.Output)
		}
		if !strings.Contains(res.Output, "directory") {
			t.Errorf("read(%q) does not say it is a directory: %s", path, res.Output)
		}
		if !strings.Contains(res.Output, "glob") {
			t.Errorf("read(%q) does not say what to use instead: %s", path, res.Output)
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

// A tool error may only point at a capability this session has.
//
// Error text is a behaviour surface (RN-3), so naming a tool that is not there
// is worse than naming nothing: the model follows the instruction, finds no
// such tool, and has spent a round learning that the error lied.
//
// The v13 digest of records-before-compaction is what that looks like. The
// scenario offered no glob, the model read directories anyway, and every error
// answered "use glob to list what is in it" — thirty reads at guessed paths,
// ending in the model writing "Glob isn't available. Let me try to list by
// guessing common file names in those directories."
func TestAToolErrorNamesGlobOnlyWhenTheSessionHasIt(t *testing.T) {
	r, err := policy.NewResolver(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeFileT(t, r.Workspace, "pkg/stats.go", "package pkg\n")

	for _, tc := range []struct {
		name  string
		tools []string
		want  bool
	}{
		{"offered", []string{"read", "glob", "grep"}, true},
		{"not offered", []string{"read", "grep"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := NewState(r, DefaultLimits(), tc.tools)

			dir := run(t, Read{}, s, ReadInput{Path: "pkg"})
			missing := run(t, Read{}, s, ReadInput{Path: "nope.go"})

			for _, res := range []Result{dir, missing} {
				if !res.IsError {
					t.Fatalf("expected an error, got: %s", res.Output)
				}
				if got := strings.Contains(strings.ToLower(res.Output), "glob"); got != tc.want {
					t.Errorf("mentions glob = %v, want %v: %s", got, tc.want, res.Output)
				}
			}
			// Dropping the suggestion must not drop the way forward. An error
			// that only says what went wrong leaves the model to guess, which
			// is the behaviour the message exists to prevent.
			if !strings.Contains(dir.Output, "file inside it") {
				t.Errorf("no way forward offered: %s", dir.Output)
			}
			if !strings.Contains(missing.Output, "Check the path") {
				t.Errorf("no way forward offered: %s", missing.Output)
			}
		})
	}
}
