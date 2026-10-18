package tools

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// A cut result says how much it left out. `Truncated` alone tells the model
// there is more and not whether it is a line or a megabyte — the difference
// between reading on and narrowing the question.
//
// The table is keyed by every type in this package that implements Execute,
// found by parsing the source rather than listed by hand: a tool added later
// that is in neither table fails here, instead of joining the ones that cut in
// silence. Each tool that can cut proves it with a scenario; each that cannot
// says why.
func TestTruncatedOutputAlwaysSaysWhatRemains(t *testing.T) {
	cuts := map[string]func(t *testing.T) (res Result, want int){
		"Read":    cutRead,
		"Glob":    cutGlob,
		"Grep":    cutGrep,
		"Symbol":  cutSymbol,
		"Explore": cutExplore,
		"Edit":    cutEdit,
		"Bash":    cutBash,
		"Process": cutProcess,
		"Fetch":   cutFetch,
	}
	neverCuts := map[string]string{
		"Write":              "its diff rides on Meta for the client and never enters Output",
		"Plan":               "renders the plan it was given, whole",
		"AskTool":            "returns the person's answer, whole",
		"DonePropose":        "returns a fixed acknowledgement",
		"Remember":           "returns a fixed acknowledgement",
		"RedTest":            "returns the verdict line, whole",
		"Verdict":            "returns the verdict line, whole",
		"DelegateTool":       "returns the child's identifier",
		"ChildrenTool":       "lists the children, whole",
		"SteerChildTool":     "returns an acknowledgement",
		"CloseChildTool":     "returns an acknowledgement",
		"SummarizeTool":      "returns an acknowledgement naming the question",
		"ProposeAnswersTool": "returns an acknowledgement naming the card",
	}

	for _, name := range executors(t) {
		_, cut := cuts[name]
		_, why := neverCuts[name]
		if !cut && !why {
			t.Errorf("%s implements Execute and is in neither table: say whether it can cut its output", name)
		}
	}

	for name, scenario := range cuts {
		t.Run(name, func(t *testing.T) {
			res, want := scenario(t)
			if !res.Truncated {
				t.Fatalf("output was cut without saying so:\n%s", head(res.Output))
			}
			if res.Remaining <= 0 {
				t.Fatalf("Truncated with Remaining=%d", res.Remaining)
			}
			if want > 0 && res.Remaining != want {
				t.Errorf("Remaining=%d, want %d", res.Remaining, want)
			}
			if !strings.Contains(res.Output, strconv.Itoa(res.Remaining)) {
				t.Errorf("the text does not say how much was left out (%d):\n%s", res.Remaining, head(res.Output))
			}
		})
	}
}

// executors returns the receiver type of every Execute method in the package.
func executors(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var names []string
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, e.Name(), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Name.Name != "Execute" {
				continue
			}
			typ := fn.Recv.List[0].Type
			if star, ok := typ.(*ast.StarExpr); ok {
				typ = star.X
			}
			if id, ok := typ.(*ast.Ident); ok {
				names = append(names, id.Name)
			}
		}
	}
	sort.Strings(names)
	if len(names) < 10 {
		t.Fatalf("found only %v: the scan is broken, not the tools", names)
	}
	return names
}

func head(s string) string {
	if len(s) > 300 {
		return s[:300] + "…"
	}
	return s
}

func cutRead(t *testing.T) (Result, int) {
	s, ws := setup(t)
	writeFileT(t, ws, "ten.txt", "1\n2\n3\n4\n5\n6\n7\n8\n9\n10")
	return run(t, Read{}, s, ReadInput{Path: "ten.txt", Limit: 3}), 7
}

func cutGlob(t *testing.T) (Result, int) {
	s, ws := setup(t)
	for i := 0; i < 10; i++ {
		writeFileT(t, ws, fmt.Sprintf("f%d.txt", i), "x")
	}
	s.Limits.GlobMaxResults = 3
	return run(t, Glob{}, s, GlobInput{Pattern: "*.txt"}), 7
}

// Matches, not lines: context lines around a match are not a unit anyone
// asked for, and two files are used so the count crosses a file boundary.
func cutGrep(t *testing.T) (Result, int) {
	s, ws := setup(t)
	writeFileT(t, ws, "a.txt", strings.Repeat("needle\n", 100))
	writeFileT(t, ws, "b.txt", strings.Repeat("needle\nhay\n", 20))
	s.Limits.GrepMaxMatches = 5
	return run(t, Grep{}, s, GrepInput{Pattern: "needle"}), 115
}

func cutSymbol(t *testing.T) (Result, int) {
	s, ws := setup(t)
	s.Limits.SymbolMaxMatches = 3
	writeFileT(t, ws, "many.go", "package p\n"+strings.Repeat("var _ = Parse\n", 50))
	return run(t, Symbol{}, s, SymbolInput{Name: "Parse"}), 47
}

func cutExplore(t *testing.T) (Result, int) {
	s, _ := setup(t)
	return run(t, Explore{Delegator: &fakeDelegator{conclusion: "…", dropped: 900}}, s,
		ExploreInput{Task: "explain"}), 900
}

// The echoed diff is the edit's output when it is echoed, so its cut is the
// edit's cut. Counted in diff lines, the unit of the cap.
func cutEdit(t *testing.T) (Result, int) {
	body := strings.Repeat("target\n", DiffMaxLines+200)
	s, ws := setup(t)
	s.Limits.EditEchoDiff = EchoDiffMulti
	p := writeFileT(t, ws, "big.txt", body)
	s.MarkRead(p, body, 0)
	return run(t, Edit{}, s, EditInput{
		Path: "big.txt", OldString: "target", NewString: "changed", ReplaceAll: true,
	}), 2*(DiffMaxLines+200) + 1 - DiffMaxLines // one hunk: its header, every line out and in
}

func cutBash(t *testing.T) (Result, int) {
	s, _ := setup(t)
	s.Limits.MaxToolOutput = 100
	r := &fakeRunner{out: strings.Repeat("x", 10_000), code: 0}
	return run(t, Bash{Runner: r}, s, BashInput{Command: "cat big"}), 9_900
}

const tailOut = 200*len("noise\n") + len("THE LAST LINE\n")

func cutProcess(t *testing.T) (Result, int) {
	s, _ := setup(t)
	s.Limits.MaxToolOutput = 200
	bg := &fakeBackground{}
	runBash(t, backgroundBash(bg), s, `{"command":"noisy","background":true}`)
	bg.handle(0).write(strings.Repeat("noise\n", 200) + "THE LAST LINE\n")
	return runProcess(t, s, `{"id":"bg1"}`), 0
}

func cutFetch(t *testing.T) (Result, int) {
	const body, limit = 4096, 512
	f, srv := fetchWith(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Length", strconv.Itoa(body))
		_, _ = w.Write([]byte(strings.Repeat("x", body)))
	})
	t.Cleanup(srv.Close)
	f.Limit = limit
	s, _ := setup(t)
	return runFetch(t, f, s, srv.URL), body - limit
}

// The paths PR #541 fixed keep their own checks: a body with no declared
// length, and a background command whose tail is kept.
func TestTheCountsFetchAndTheTailKeepAreExact(t *testing.T) {
	const body, limit = 4096, 512
	t.Run("fetch with no declared length", func(t *testing.T) {
		f, srv := fetchWith(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			for i := 0; i < body/64; i++ {
				_, _ = w.Write([]byte(strings.Repeat("y", 64)))
				w.(http.Flusher).Flush() // chunked: the length is not known up front
			}
		})
		defer srv.Close()
		f.Limit = limit
		s, _ := setup(t)
		res := runFetch(t, f, s, srv.URL)
		if !res.Truncated || res.Remaining != body-limit {
			t.Errorf("Truncated=%v Remaining=%d, want true and %d", res.Truncated, res.Remaining, body-limit)
		}
	})

	checkTail := func(t *testing.T, res Result) {
		t.Helper()
		if !res.Truncated || res.Remaining < tailOut-200 || res.Remaining >= tailOut {
			t.Errorf("Truncated=%v Remaining=%d for %d bytes kept to a 200-byte window", res.Truncated, res.Remaining, tailOut)
		}
	}
	t.Run("process", func(t *testing.T) {
		res, _ := cutProcess(t)
		checkTail(t, res)
	})
	t.Run("bash in the background", func(t *testing.T) {
		s, _ := setup(t)
		s.Limits.MaxToolOutput = 200
		bg := &fakeBackground{seed: []*fakeHandle{{out: strings.Repeat("noise\n", 200) + "THE LAST LINE\n"}}}
		checkTail(t, runBash(t, backgroundBash(bg), s, `{"command":"noisy","background":true}`))
	})
}

// A deadline cuts time, not output. Marking it Truncated would claim bytes
// were held back when every byte the command produced is in the result; the
// cut is already said by Code = timeout. Output that also overflowed is a
// truncation on top, counted in bytes like any other.
func TestATimeoutIsNotATruncationOfOutput(t *testing.T) {
	t.Run("short output", func(t *testing.T) {
		s, _ := setup(t)
		r := &fakeRunner{out: "started…", wait: time.Second, err: context.DeadlineExceeded}
		res := run(t, Bash{Runner: r, Timeout: 20 * time.Millisecond}, s, BashInput{Command: "sleep 10"})
		if res.Code != CodeTimeout {
			t.Fatalf("want timeout, got %+v", res)
		}
		if res.Truncated || res.Remaining != 0 {
			t.Errorf("nothing was held back, yet Truncated=%v Remaining=%d", res.Truncated, res.Remaining)
		}
	})
	t.Run("output over the cap", func(t *testing.T) {
		s, _ := setup(t)
		s.Limits.MaxToolOutput = 100
		r := &fakeRunner{out: strings.Repeat("x", 1_000), wait: time.Second, err: context.DeadlineExceeded}
		res := run(t, Bash{Runner: r, Timeout: 20 * time.Millisecond}, s, BashInput{Command: "yes"})
		if res.Code != CodeTimeout || !res.Truncated || res.Remaining != 900 {
			t.Errorf("Code=%q Truncated=%v Remaining=%d, want timeout, true and 900", res.Code, res.Truncated, res.Remaining)
		}
	})
}
