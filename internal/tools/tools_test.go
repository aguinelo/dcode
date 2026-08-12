package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
)

func setup(t *testing.T) (*State, string) {
	t.Helper()
	ws := t.TempDir()
	r, err := policy.NewResolver(ws)
	if err != nil {
		t.Fatal(err)
	}
	return NewState(r, DefaultLimits()), r.Workspace
}

func writeFileT(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func run(t *testing.T, tool Tool, s *State, args any) Result {
	t.Helper()
	b, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	res, err := tool.Execute(context.Background(), b, s)
	if err != nil {
		t.Fatalf("%s returned a hard error: %v", tool.Name(), err)
	}
	return res
}

// The invariant that stops the most expensive failure this product can produce.
// It must hold even when the path exists and the text matches — those are
// exactly the conditions under which a blind edit looks like it worked.
func TestEditRefusesAFileNotReadInThisSession(t *testing.T) {
	s, ws := setup(t)
	writeFileT(t, ws, "main.go", "package main\n\nfunc main() {}\n")

	res := run(t, Edit{}, s, EditInput{
		Path: "main.go", OldString: "func main() {}", NewString: "func main() { println() }",
	})
	if !res.IsError || res.Code != CodeFileNotRead {
		t.Fatalf("want file_not_read, got %+v", res)
	}

	// And the file is untouched.
	got, _ := os.ReadFile(filepath.Join(ws, "main.go"))
	if !strings.Contains(string(got), "func main() {}") {
		t.Error("the file was modified despite the refusal")
	}
	if !strings.Contains(res.Output, "Read it first") {
		t.Errorf("the error must teach the recovery: %q", res.Output)
	}
}

func TestEditRefusesAFileChangedSinceItWasRead(t *testing.T) {
	s, ws := setup(t)
	path := writeFileT(t, ws, "a.go", "one\ntwo\n")
	run(t, Read{}, s, ReadInput{Path: "a.go"})

	// Someone else edits it — the exact case the invariant exists for.
	if err := os.WriteFile(path, []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := run(t, Edit{}, s, EditInput{Path: "a.go", OldString: "two", NewString: "TWO"})
	if !res.IsError || res.Code != CodeFileChanged {
		t.Fatalf("want file_changed, got %+v", res)
	}
	if !strings.Contains(string(mustRead(t, path)), "three") {
		t.Error("the concurrent edit was overwritten")
	}
}

// Picking the first occurrence is right most of the time and, when it is wrong,
// edits the wrong place silently. Failing explicitly is always better when the
// cost of being wrong is corrupted code.
func TestEditRefusesAnAmbiguousMatchAndChangesNothing(t *testing.T) {
	s, ws := setup(t)
	path := writeFileT(t, ws, "d.go", "x := 1\ny := 1\nz := 1\n")
	run(t, Read{}, s, ReadInput{Path: "d.go"})
	before := string(mustRead(t, path))

	res := run(t, Edit{}, s, EditInput{Path: "d.go", OldString: "1", NewString: "2"})
	if !res.IsError || res.Code != CodeAmbiguousMatch {
		t.Fatalf("want ambiguous_match, got %+v", res)
	}
	if !strings.Contains(res.Output, "3 times") {
		t.Errorf("the count is what lets the model disambiguate: %q", res.Output)
	}
	if got := string(mustRead(t, path)); got != before {
		t.Errorf("the file must be byte-identical after a refusal:\n%q", got)
	}
}

func TestEditReplaceAllIsExplicit(t *testing.T) {
	s, ws := setup(t)
	path := writeFileT(t, ws, "d.go", "a\na\na\n")
	run(t, Read{}, s, ReadInput{Path: "d.go"})

	res := run(t, Edit{}, s, EditInput{Path: "d.go", OldString: "a", NewString: "b", ReplaceAll: true})
	if res.IsError {
		t.Fatalf("unexpected error: %+v", res)
	}
	if got := string(mustRead(t, path)); got != "b\nb\nb\n" {
		t.Errorf("got %q", got)
	}
}

// A second edit must work without a re-read. Forgetting to re-mark after a
// write makes the next edit fail as file_changed for no reason the user can see.
func TestTwoEditsInARowNeedNoReread(t *testing.T) {
	s, ws := setup(t)
	path := writeFileT(t, ws, "a.go", "one\ntwo\n")
	run(t, Read{}, s, ReadInput{Path: "a.go"})

	if res := run(t, Edit{}, s, EditInput{Path: "a.go", OldString: "one", NewString: "ONE"}); res.IsError {
		t.Fatalf("first edit: %+v", res)
	}
	if res := run(t, Edit{}, s, EditInput{Path: "a.go", OldString: "two", NewString: "TWO"}); res.IsError {
		t.Fatalf("second edit should not need a re-read: %+v", res)
	}
	if got := string(mustRead(t, path)); got != "ONE\nTWO\n" {
		t.Errorf("got %q", got)
	}
}

func TestEditRejectsANoOp(t *testing.T) {
	s, ws := setup(t)
	writeFileT(t, ws, "a.go", "x\n")
	run(t, Read{}, s, ReadInput{Path: "a.go"})

	res := run(t, Edit{}, s, EditInput{Path: "a.go", OldString: "x", NewString: "x"})
	if !res.IsError || res.Code != CodeNoOpEdit {
		t.Fatalf("want no_op_edit, got %+v", res)
	}
}

func TestEditReportsNoMatch(t *testing.T) {
	s, ws := setup(t)
	writeFileT(t, ws, "a.go", "hello\n")
	run(t, Read{}, s, ReadInput{Path: "a.go"})

	res := run(t, Edit{}, s, EditInput{Path: "a.go", OldString: "goodbye", NewString: "hi"})
	if !res.IsError || res.Code != CodeNoMatch {
		t.Fatalf("want no_match, got %+v", res)
	}
}

// Overwriting carries the same risk as editing, so it carries the same rule.
func TestWriteOverExistingFileRequiresARead(t *testing.T) {
	s, ws := setup(t)
	writeFileT(t, ws, "a.go", "original\n")

	res := run(t, Write{}, s, WriteInput{Path: "a.go", Content: "replaced\n"})
	if !res.IsError || res.Code != CodeFileNotRead {
		t.Fatalf("want file_not_read, got %+v", res)
	}
	if got := string(mustRead(t, filepath.Join(ws, "a.go"))); got != "original\n" {
		t.Errorf("the file was replaced anyway: %q", got)
	}
}

func TestWriteCreatesNewFilesWithParents(t *testing.T) {
	s, ws := setup(t)
	res := run(t, Write{}, s, WriteInput{Path: "deep/nested/new.go", Content: "package x\n"})
	if res.IsError {
		t.Fatalf("%+v", res)
	}
	if got := string(mustRead(t, filepath.Join(ws, "deep/nested/new.go"))); got != "package x\n" {
		t.Errorf("got %q", got)
	}
	// A file just written is known content, so editing it next needs no read.
	if !s.WasRead(filepath.Join(ws, "deep/nested/new.go")) {
		t.Error("a written file should count as read")
	}
}

func TestReadNumbersLinesAndTruncates(t *testing.T) {
	s, ws := setup(t)
	var b strings.Builder
	for i := 1; i <= 50; i++ {
		b.WriteString("line\n")
	}
	writeFileT(t, ws, "big.txt", b.String())

	s.Limits.ReadMaxLines = 10
	res := run(t, Read{}, s, ReadInput{Path: "big.txt"})
	if !res.Truncated || res.Remaining == 0 {
		t.Errorf("truncation must be declared: %+v", res)
	}
	if !strings.Contains(res.Output, "     1\t") {
		t.Errorf("output must be line-numbered so edit and the model agree: %q", res.Output)
	}
	if strings.Contains(res.Output, "    11\t") {
		t.Error("more lines were returned than the limit allows")
	}
	// A partial read must not count as having read the file: a later edit
	// would then act on a region never seen.
	if s.WasRead(filepath.Join(ws, "big.txt")) {
		t.Error("a truncated read must not satisfy the read-before-edit invariant")
	}
}

func TestReadClampsVeryLongLines(t *testing.T) {
	s, ws := setup(t)
	writeFileT(t, ws, "min.js", strings.Repeat("x", 10_000)+"\n")
	s.Limits.ReadMaxLineLength = 100

	res := run(t, Read{}, s, ReadInput{Path: "min.js"})
	if len(res.Output) > 1000 {
		t.Errorf("a minified file must not cost the whole window: %d bytes", len(res.Output))
	}
	if !strings.Contains(res.Output, "more bytes on this line") {
		t.Errorf("the clamp should be visible: %q", res.Output)
	}
}

func TestReadRefusesToDumpBinary(t *testing.T) {
	s, ws := setup(t)
	if err := os.WriteFile(filepath.Join(ws, "bin"), []byte{0x00, 0x01, 0xff, 0xfe}, 0o644); err != nil {
		t.Fatal(err)
	}
	res := run(t, Read{}, s, ReadInput{Path: "bin"})
	if !strings.Contains(res.Output, "binary") {
		t.Errorf("binary content must be described, not dumped: %q", res.Output)
	}
}

func TestReadOffset(t *testing.T) {
	s, ws := setup(t)
	writeFileT(t, ws, "a.txt", "one\ntwo\nthree\nfour\n")
	res := run(t, Read{}, s, ReadInput{Path: "a.txt", Offset: 3, Limit: 1})
	if !strings.Contains(res.Output, "three") || strings.Contains(res.Output, "one") {
		t.Errorf("offset ignored: %q", res.Output)
	}
}

func TestReadMissingFileIsRecoverable(t *testing.T) {
	s, _ := setup(t)
	res := run(t, Read{}, s, ReadInput{Path: "nope.go"})
	if !res.IsError || res.Code != CodeNotFound {
		t.Fatalf("want not_found, got %+v", res)
	}
	if !strings.Contains(res.Output, "glob") {
		t.Errorf("the hint should point at a way forward: %q", res.Output)
	}
}

// Filesystem order varies by machine: sorting is what stops a test passing
// locally and failing in CI.
func TestGlobIsSortedAndStable(t *testing.T) {
	s, ws := setup(t)
	for _, n := range []string{"c.go", "a.go", "b.go", "sub/d.go"} {
		writeFileT(t, ws, n, "package x\n")
	}
	var first string
	for i := 0; i < 10; i++ {
		res := run(t, Glob{}, s, GlobInput{Pattern: "**/*.go"})
		if i == 0 {
			first = res.Output
			continue
		}
		if res.Output != first {
			t.Fatalf("glob order changed between runs:\n%q\n%q", first, res.Output)
		}
	}
	lines := strings.Split(strings.TrimSpace(first), "\n")
	if len(lines) != 4 {
		t.Fatalf("want 4 files, got %d: %q", len(lines), first)
	}
	if lines[0] != "a.go" {
		t.Errorf("results must be alphabetical, got %q", lines)
	}
}

func TestGlobRespectsGitignore(t *testing.T) {
	s, ws := setup(t)
	writeFileT(t, ws, ".gitignore", "node_modules/\n*.log\n!keep.log\n")
	for _, n := range []string{"a.go", "node_modules/dep.go", "debug.log", "keep.log"} {
		writeFileT(t, ws, n, "x")
	}

	res := run(t, Glob{}, s, GlobInput{Pattern: "**/*"})
	if strings.Contains(res.Output, "node_modules") {
		t.Errorf("ignored directory leaked in: %q", res.Output)
	}
	if strings.Contains(res.Output, "debug.log") {
		t.Errorf("ignored file leaked in: %q", res.Output)
	}
	// A later negation must win, which is how gitignore resolves them.
	if !strings.Contains(res.Output, "keep.log") {
		t.Errorf("negation should re-include the file: %q", res.Output)
	}
}

func TestGlobTruncatesAndSaysSo(t *testing.T) {
	s, ws := setup(t)
	for i := 0; i < 20; i++ {
		writeFileT(t, ws, filepath.Join("d", "f"+string(rune('a'+i))+".go"), "x")
	}
	s.Limits.GlobMaxResults = 5
	res := run(t, Glob{}, s, GlobInput{Pattern: "**/*.go"})
	if !res.Truncated || res.Remaining == 0 {
		t.Errorf("truncation must be declared: %+v", res)
	}
	if !strings.Contains(res.Output, "Narrow the pattern") {
		t.Errorf("the model needs to know what to do next: %q", res.Output)
	}
}

func TestGlobInvalidPatternIsRecoverable(t *testing.T) {
	s, _ := setup(t)
	res := run(t, Glob{}, s, GlobInput{Pattern: "["})
	if !res.IsError || res.Code != CodeInvalidPattern {
		t.Fatalf("want invalid_pattern, got %+v", res)
	}
}

func TestGrepFindsAndFormats(t *testing.T) {
	s, ws := setup(t)
	writeFileT(t, ws, "a.go", "package main\nfunc target() {}\n")
	writeFileT(t, ws, "b.go", "package main\n")

	res := run(t, Grep{}, s, GrepInput{Pattern: "func target"})
	if !strings.Contains(res.Output, "a.go:2:") {
		t.Errorf("output should be path:line:text, got %q", res.Output)
	}
	if strings.Contains(res.Output, "b.go") {
		t.Errorf("non-matching file listed: %q", res.Output)
	}
}

func TestGrepGlobNarrowsTheSearch(t *testing.T) {
	s, ws := setup(t)
	writeFileT(t, ws, "a.go", "needle\n")
	writeFileT(t, ws, "a.md", "needle\n")

	res := run(t, Grep{}, s, GrepInput{Pattern: "needle", Glob: "*.go"})
	if !strings.Contains(res.Output, "a.go") || strings.Contains(res.Output, "a.md") {
		t.Errorf("glob ignored: %q", res.Output)
	}
}

func TestGrepInvalidRegexIsRecoverable(t *testing.T) {
	s, _ := setup(t)
	res := run(t, Grep{}, s, GrepInput{Pattern: "([unclosed"})
	if !res.IsError || res.Code != CodeInvalidPattern {
		t.Fatalf("want invalid_pattern, got %+v", res)
	}
	// A panic here would take down a session over a typo in a search.
}

func TestGrepReportsNoMatchesPlainly(t *testing.T) {
	s, ws := setup(t)
	writeFileT(t, ws, "a.go", "nothing here\n")
	res := run(t, Grep{}, s, GrepInput{Pattern: "zzz"})
	if res.IsError {
		t.Errorf("no matches is not an error: %+v", res)
	}
	if !strings.Contains(res.Output, "no matches") {
		t.Errorf("got %q", res.Output)
	}
}

func mustRead(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// Declare must be free of side effects: policy decides on a declaration, and a
// declaration that already did something leaves nothing to decide.
func TestDeclareTouchesNothing(t *testing.T) {
	s, ws := setup(t)
	target := filepath.Join(ws, "should-not-exist.go")

	for _, tc := range []struct {
		tool Tool
		args any
	}{
		{Read{}, ReadInput{Path: "should-not-exist.go"}},
		{Write{}, WriteInput{Path: "should-not-exist.go", Content: "x"}},
		{Edit{}, EditInput{Path: "should-not-exist.go", OldString: "a", NewString: "b"}},
		{Glob{}, GlobInput{Pattern: "*"}},
		{Grep{}, GrepInput{Pattern: "x"}},
		{Bash{}, BashInput{Command: "touch should-not-exist.go"}},
		{Plan{}, PlanInput{Items: []protocol.PlanItem{{ID: 1, Text: "x", Status: protocol.PlanPending}}}},
	} {
		b, _ := json.Marshal(tc.args)
		if _, err := tc.tool.Declare(b); err != nil {
			t.Errorf("%s: %v", tc.tool.Name(), err)
		}
	}

	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Error("Declare created a file")
	}
	if len(s.Plan()) != 0 {
		t.Error("Declare mutated the plan")
	}
}

// bash is opaque, so it must declare the worst case: anything, plus network.
// Anything less would let the loop run it in parallel with a conflicting call.
// A shell command is opaque, so the worst case is what gets declared — bounded
// by the sandbox that will run it, because a crossing the mechanism already
// prevents is a false alarm rather than honesty.
func TestBashDeclaresTheWorstCaseTheSandboxAllows(t *testing.T) {
	req, err := (Bash{}).Declare(json.RawMessage(`{"command":"rm -rf /"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !req.Network {
		t.Error("with the network open a shell command could reach it and must say so")
	}
	if len(req.Paths) == 0 || !req.Paths[0].Write {
		t.Error("a shell command could write and must say so, in every configuration")
	}
	if req.Command == "" {
		t.Error("the rendered command must reach the approval prompt")
	}
}

// plan changes session state only, so it must never cross a boundary — in any
// mode, including read-only.
func TestPlanNeverCrossesABoundary(t *testing.T) {
	req, err := (Plan{}).Declare(json.RawMessage(`{"items":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []policy.SandboxMode{
		policy.ModeReadOnly, policy.ModeWorkspaceWrite, policy.ModeFullAccess,
	} {
		for _, pol := range []policy.ApprovalPolicy{
			policy.PolicyUntrusted, policy.PolicyOnRequest, policy.PolicyNever,
		} {
			v := policy.Evaluate(req, mode, pol, policy.Rules{}, func(policy.Access) bool { return true })
			if v.Decision != policy.DecisionAllow {
				t.Errorf("mode=%s policy=%s: plan got %s", mode, pol, v.Decision)
			}
		}
	}
}

// The schema tells the model "Lines of context around each match" and Execute
// never read the field. A tool description is a behaviour surface (RN-3), so
// this was not a missing feature — it was the product lying to the only reader
// it has. The model asks for context, plans on having it, gets bare match lines
// and re-reads the file, spending the window the parameter existed to save.
func TestGrepReturnsTheContextItAdvertises(t *testing.T) {
	s, ws := setup(t)
	writeFileT(t, ws, "pay.go", strings.Join([]string{
		"package pay",       // 1
		"",                  // 2
		"func before() {",   // 3
		"\treturn",          // 4
		"}",                 // 5
		"",                  // 6
		"func Validate() {", // 7
		"\tcheck()",         // 8
		"}",                 // 9
	}, "\n"))

	res := run(t, Grep{}, s, GrepInput{Pattern: "Validate", ContextLines: 2})

	// The match itself, with its own line number.
	if !strings.Contains(res.Output, "pay.go:7:func Validate() {") {
		t.Fatalf("the match line is missing:\n%s", res.Output)
	}
	// Two lines either side, each numbered, so the model can cite what it saw.
	for _, want := range []string{"pay.go:5", "pay.go:6", "pay.go:8", "pay.go:9"} {
		if !strings.Contains(res.Output, want) {
			t.Errorf("context line %s is missing:\n%s", want, res.Output)
		}
	}
	// And not more than asked: context that quietly grows is a window nobody
	// budgeted for.
	if strings.Contains(res.Output, "pay.go:4") || strings.Contains(res.Output, "pay.go:3") {
		t.Errorf("more context than requested:\n%s", res.Output)
	}
}

// Zero is the default and must mean what it always meant, or every existing
// caller silently starts paying for context it never asked for.
func TestGrepWithoutContextIsUnchanged(t *testing.T) {
	s, ws := setup(t)
	writeFileT(t, ws, "a.go", "package a\n\nfunc Validate() {}\n")

	res := run(t, Grep{}, s, GrepInput{Pattern: "Validate"})
	if strings.Contains(res.Output, "a.go:1") || strings.Contains(res.Output, "a.go:2") {
		t.Errorf("context appeared without being asked for:\n%s", res.Output)
	}
	if !strings.Contains(res.Output, "a.go:3:func Validate() {}") {
		t.Errorf("the match itself is missing:\n%s", res.Output)
	}
}

// Context is bounded like everything else here. A generous number against a
// common pattern would otherwise return the file, which is the one outcome the
// match-limit exists to prevent.
func TestGrepContextIsCapped(t *testing.T) {
	s, ws := setup(t)
	var lines []string
	for i := 0; i < 200; i++ {
		lines = append(lines, "filler")
	}
	lines[100] = "needle"
	writeFileT(t, ws, "big.txt", strings.Join(lines, "\n"))

	res := run(t, Grep{}, s, GrepInput{Pattern: "needle", ContextLines: 5000})
	if n := strings.Count(res.Output, "big.txt:"); n > GrepMaxContextLines*2+1 {
		t.Errorf("%d lines returned for one match; context is not capped", n)
	}
}

// Narrowing a search to one file is the most natural thing a model does after
// a broad search finds too much — and it answered "no matches for X" while X
// was on line two. Not an error: a confident report of absence, which is the
// one kind of wrong answer nobody re-checks.
//
// walkFiles only ever walked directories, and a file root landed on the
// `rel == "."` skip, so the search ran over nothing and found nothing.
func TestGrepSearchesTheFileItWasPointedAt(t *testing.T) {
	s, ws := setup(t)
	writeFileT(t, ws, "internal/config/toml.go", "package config\n\nvar KnownKeys = map[string]string{}\n")
	writeFileT(t, ws, "other.go", "package other\n\nvar KnownKeys = 1\n")

	res := run(t, Grep{}, s, GrepInput{Pattern: "KnownKeys", Path: "internal/config/toml.go"})
	if res.IsError {
		t.Fatalf("grep on a file errored: %+v", res)
	}
	// The format, not the word: "no matches for \"KnownKeys\"" contains
	// KnownKeys too, and asserting the word alone passes on the failure.
	if !strings.Contains(res.Output, "toml.go:3:") {
		t.Errorf("grep on a file reported no matches for something on line three: %q", res.Output)
	}
	if strings.Contains(res.Output, "other.go") {
		t.Errorf("grep on a file searched outside it: %q", res.Output)
	}
}

// Same root cause, same silent absence: a file is a legitimate place to look
// for a definition.
func TestSymbolSearchesTheFileItWasPointedAt(t *testing.T) {
	s, ws := setup(t)
	writeFileT(t, ws, "internal/config/toml.go", "package config\n\nfunc ParseTOML() {}\n")

	res := run(t, Symbol{}, s, SymbolInput{Name: "ParseTOML", Path: "internal/config/toml.go"})
	if res.IsError {
		t.Fatalf("symbol on a file errored: %+v", res)
	}
	if !strings.Contains(res.Output, "toml.go:3:") {
		t.Errorf("symbol on a file found nothing: %q", res.Output)
	}
}

// And glob: pointing it at a file is a narrower question, not a broken one.
func TestGlobPointedAtAFileFindsIt(t *testing.T) {
	s, ws := setup(t)
	writeFileT(t, ws, "internal/config/toml.go", "package config\n")

	res := run(t, Glob{}, s, GlobInput{Pattern: "*.go", Path: "internal/config/toml.go"})
	if res.IsError {
		t.Fatalf("glob on a file errored: %+v", res)
	}
	if !strings.Contains(res.Output, "toml.go") {
		t.Errorf("glob on a file found nothing: %q", res.Output)
	}
}
