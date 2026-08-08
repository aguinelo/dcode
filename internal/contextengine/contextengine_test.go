package contextengine

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func msg(r Role, text string) Message { return Message{Role: r, Text: text} }

func baseSession() Session {
	return Session{
		Instructions: "You are dcode.",
		Tools: []ToolDef{
			{Name: "read", Description: "Read a file.", Schema: json.RawMessage(`{"type":"object"}`)},
		},
		History: []Message{msg(RoleUser, "hi"), msg(RoleAssistant, "hello")},
	}
}

func TestAssembleIsPure(t *testing.T) {
	s := baseSession()
	first, err := Assemble(s)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 50; i++ {
		got, err := Assemble(s)
		if err != nil {
			t.Fatal(err)
		}
		if encode(t, got) != encode(t, first) {
			t.Fatalf("Assemble is not deterministic on run %d", i)
		}
	}
}

// The single most important test in the package: appending to history must not
// change a single byte of what came before. If this breaks, every turn re-bills
// the full prompt and the product loses its main cost advantage.
func TestPrefixIsStableUnderAppend(t *testing.T) {
	s := baseSession()
	before, err := Assemble(s)
	if err != nil {
		t.Fatal(err)
	}

	s.History = append(s.History, msg(RoleUser, "and now something else"))
	after, err := Assemble(s)
	if err != nil {
		t.Fatal(err)
	}

	if len(after) <= len(before) {
		t.Fatalf("append should grow the message list: %d -> %d", len(before), len(after))
	}
	for i := range before {
		if encode(t, before[i]) != encode(t, after[i]) {
			t.Errorf("message %d changed after append:\nbefore=%s\n after=%s",
				i, encode(t, before[i]), encode(t, after[i]))
		}
	}
}

// Volatile data in the prefix invalidates the cache every turn and breaks no
// other test, so it gets its own sweep.
func TestNoVolatileDataInOutput(t *testing.T) {
	s := baseSession()
	s.Summary = &Summary{Text: "earlier we did things", UpToIdx: 1}
	out, err := Assemble(s)
	if err != nil {
		t.Fatal(err)
	}
	blob := encode(t, out)

	for _, pat := range []struct {
		name string
		re   *regexp.Regexp
	}{
		{"RFC3339 timestamp", regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}`)},
		{"clock time", regexp.MustCompile(`\b\d{2}:\d{2}:\d{2}\b`)},
		{"token counter", regexp.MustCompile(`(?i)\b(tokens?\s+(remaining|used|left))\b`)},
		{"iteration counter", regexp.MustCompile(`(?i)\biteration\s+\d+`)},
	} {
		if pat.re.MatchString(blob) {
			t.Errorf("prefix contains %s: %s", pat.name, blob)
		}
	}
}

func TestSummaryAbsentEmitsNoMarker(t *testing.T) {
	s := baseSession()
	without, err := Assemble(s)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(without[0].Text, "Earlier in this session") {
		t.Fatal("a nil summary must contribute nothing, not an empty section")
	}

	s.Summary = &Summary{Text: "stuff", UpToIdx: 0}
	with, err := Assemble(s)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(with[0].Text, "Earlier in this session") {
		t.Fatal("a present summary must be rendered")
	}
}

func TestAssembleSectionOrder(t *testing.T) {
	s := baseSession()
	s.Summary = &Summary{Text: "SUMMARY-MARK", UpToIdx: 1}
	out, err := Assemble(s)
	if err != nil {
		t.Fatal(err)
	}
	sys := out[0].Text
	iInstr := strings.Index(sys, "You are dcode.")
	iTools := strings.Index(sys, "## Tools")
	iSum := strings.Index(sys, "SUMMARY-MARK")
	if !(iInstr >= 0 && iInstr < iTools && iTools < iSum) {
		t.Errorf("order must be instructions < tools < summary, got %d %d %d", iInstr, iTools, iSum)
	}
	if out[0].Role != RoleSystem {
		t.Errorf("first message must be the system block, got %q", out[0].Role)
	}
}

func TestAssembleHonoursSummaryOffset(t *testing.T) {
	s := baseSession()
	s.History = []Message{msg(RoleUser, "a"), msg(RoleAssistant, "b"), msg(RoleUser, "c")}
	s.Summary = &Summary{Text: "covered a and b", UpToIdx: 2}
	out, err := Assemble(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[1].Text != "c" {
		t.Fatalf("only history past the summary should be live, got %d messages", len(out))
	}
}

// An out-of-range offset must clamp rather than panic: it can only arrive from
// a corrupted session file, and crashing loses the whole session.
func TestAssembleClampsBadSummaryOffset(t *testing.T) {
	for _, idx := range []int{-5, 99} {
		s := baseSession()
		s.Summary = &Summary{Text: "x", UpToIdx: idx}
		if _, err := Assemble(s); err != nil {
			t.Errorf("UpToIdx=%d: %v", idx, err)
		}
	}
}

func TestAssembleRejectsEmptyInstructions(t *testing.T) {
	for _, in := range []string{"", "   ", "\n\t"} {
		s := baseSession()
		s.Instructions = in
		if _, err := Assemble(s); err == nil {
			t.Errorf("instructions %q should be rejected", in)
		}
	}
}

func TestEstimateIsDeterministic(t *testing.T) {
	msgs, err := Assemble(baseSession())
	if err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	first := Estimate(msgs, cfg)
	for i := 0; i < 100; i++ {
		if got := Estimate(msgs, cfg); got != first {
			t.Fatalf("Estimate drifted on run %d: %d != %d", i, got, first)
		}
	}
	if first <= 0 {
		t.Errorf("estimate should be positive, got %d", first)
	}
}

func TestEstimateGrowsWithContent(t *testing.T) {
	cfg := DefaultConfig()
	small := Estimate([]Message{msg(RoleUser, "hi")}, cfg)
	big := Estimate([]Message{msg(RoleUser, strings.Repeat("hi ", 500))}, cfg)
	if big <= small {
		t.Errorf("more text must estimate higher: %d vs %d", small, big)
	}
}

func TestEstimateCountsToolTraffic(t *testing.T) {
	cfg := DefaultConfig()
	plain := Estimate([]Message{{Role: RoleAssistant}}, cfg)
	withCall := Estimate([]Message{{Role: RoleAssistant, ToolCalls: []ToolCall{
		{ID: "c1", Name: "bash", Input: json.RawMessage(`{"command":"go test ./..."}`)},
	}}}, cfg)
	withResult := Estimate([]Message{{Role: RoleTool, ToolResult: &ToolResult{
		ToolCallID: "c1", Output: strings.Repeat("x", 400),
	}}}, cfg)
	if withCall <= plain || withResult <= plain {
		t.Errorf("tool calls and results must count: plain=%d call=%d result=%d",
			plain, withCall, withResult)
	}
}

func TestPlanBelowThresholdDoesNothing(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Window = 1_000_000
	if _, ok := Plan(baseSession(), cfg); ok {
		t.Error("a tiny session must not trigger compaction")
	}
}

func TestPlanWithoutWindowDoesNothing(t *testing.T) {
	// Window 0 means the provider has not reported one yet. Guessing here would
	// compact a session that had plenty of room.
	if _, ok := Plan(longSession(40), DefaultConfig()); ok {
		t.Error("no window means no compaction")
	}
}

// The cut may never separate an assistant tool call from its results.
func TestPlanNeverSplitsAToolCallFromItsResults(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Window = 200
	cfg.KeepTurns = 0

	for turns := 2; turns <= 14; turns++ {
		s := toolHeavySession(turns)
		plan, ok := Plan(s, cfg)
		if !ok {
			continue
		}
		if !isCleanBoundary(s.History, plan.ToIdx) {
			t.Fatalf("turns=%d: cut at %d splits a turn", turns, plan.ToIdx)
		}
	}
}

// The current task survives by construction. A summary that loses what the user
// asked makes the agent useless exactly when the task got long enough to matter.
func TestPlanNeverCompactsTheCurrentTask(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Window = 200
	cfg.KeepTurns = 0

	s := longSession(30)
	plan, ok := Plan(s, cfg)
	if !ok {
		t.Skip("no plan produced")
	}
	lastUser := -1
	for i := len(s.History) - 1; i >= 0; i-- {
		if s.History[i].Role == RoleUser {
			lastUser = i
			break
		}
	}
	if plan.ToIdx > lastUser {
		t.Errorf("cut at %d swallows the current task at %d", plan.ToIdx, lastUser)
	}
}

func TestPlanRespectsKeepTurns(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Window = 200

	cfg.KeepTurns = 0
	aggressive, okA := Plan(longSession(30), cfg)
	cfg.KeepTurns = 6
	gentle, okB := Plan(longSession(30), cfg)

	if !okA || !okB {
		t.Skip("no plan produced")
	}
	if gentle.ToIdx > aggressive.ToIdx {
		t.Errorf("keeping more turns must cut no later: keep0=%d keep6=%d",
			aggressive.ToIdx, gentle.ToIdx)
	}
}

func TestPlanIsDeterministic(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Window = 200
	s := longSession(30)
	first, ok := Plan(s, cfg)
	for i := 0; i < 30; i++ {
		got, ok2 := Plan(s, cfg)
		if ok != ok2 || got != first {
			t.Fatalf("Plan drifted on run %d", i)
		}
	}
}

func TestApplyDoesNotMutateHistory(t *testing.T) {
	s := longSession(10)
	before := encode(t, s.History)
	out := Apply(s, CompactionPlan{FromIdx: 0, ToIdx: 4}, "summary text")
	if encode(t, s.History) != before {
		t.Error("Apply mutated the caller's history")
	}
	if out.Summary == nil || out.Summary.UpToIdx != 4 {
		t.Errorf("Apply should record the cut point, got %+v", out.Summary)
	}
}

func TestApplyMergesSuccessiveSummaries(t *testing.T) {
	s := longSession(10)
	s = Apply(s, CompactionPlan{ToIdx: 2}, "first")
	s = Apply(s, CompactionPlan{ToIdx: 6}, "second")
	if !strings.Contains(s.Summary.Text, "first") || !strings.Contains(s.Summary.Text, "second") {
		t.Errorf("a second compaction must not drop the first: %q", s.Summary.Text)
	}
	// Losing the earlier summary would silently erase the oldest history: the
	// span it covered is already gone from the live window.
	if s.Summary.UpToIdx != 6 {
		t.Errorf("UpToIdx should advance, got %d", s.Summary.UpToIdx)
	}
}

func TestApplyWithEmptyTextKeepsPrevious(t *testing.T) {
	s := Apply(longSession(10), CompactionPlan{ToIdx: 2}, "first")
	s = Apply(s, CompactionPlan{ToIdx: 4}, "   ")
	if !strings.Contains(s.Summary.Text, "first") {
		t.Errorf("empty new text must not erase the previous summary: %q", s.Summary.Text)
	}
}

// Purity guard. Parses the package's own sources and fails on any import that
// could smuggle in a clock, the environment or randomness.
//
// This looks like overkill until someone adds a time.Now() "just for a log
// line" and the cache quietly stops working in production. The test is far
// cheaper than that investigation.
func TestPackageImportsNothingImpure(t *testing.T) {
	forbidden := map[string]string{
		"os":          "environment and filesystem",
		"net":         "network",
		"net/http":    "network",
		"time":        "clock",
		"math/rand":   "randomness",
		"crypto/rand": "randomness",
		"syscall":     "system calls",
		"os/exec":     "process execution",
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(".", name), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if why, bad := forbidden[path]; bad {
				t.Errorf("%s imports %q (%s) — this package must stay pure", name, path, why)
			}
		}
	}
}

func encode(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// longSession builds n user/assistant exchanges with enough text to cross a
// small window.
func longSession(n int) Session {
	s := baseSession()
	s.History = nil
	for i := 0; i < n; i++ {
		s.History = append(s.History,
			msg(RoleUser, strings.Repeat("question ", 20)),
			msg(RoleAssistant, strings.Repeat("answer ", 20)),
		)
	}
	return s
}

// toolHeavySession builds turns where each assistant message calls two tools,
// so every possible cut point has a chance to land mid-turn.
func toolHeavySession(turns int) Session {
	s := baseSession()
	s.History = nil
	for i := 0; i < turns; i++ {
		a := "a" + string(rune('A'+i%26))
		b := "b" + string(rune('A'+i%26))
		s.History = append(s.History,
			msg(RoleUser, strings.Repeat("do it ", 20)),
			Message{Role: RoleAssistant, ToolCalls: []ToolCall{
				{ID: a, Name: "read", Input: json.RawMessage(`{}`)},
				{ID: b, Name: "grep", Input: json.RawMessage(`{}`)},
			}},
			Message{Role: RoleTool, ToolResult: &ToolResult{ToolCallID: a, Output: strings.Repeat("x", 40)}},
			Message{Role: RoleTool, ToolResult: &ToolResult{ToolCallID: b, Output: strings.Repeat("y", 40)}},
			msg(RoleAssistant, strings.Repeat("done ", 20)),
		)
	}
	return s
}
