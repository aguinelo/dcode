package evals

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

// contractRow matches a behavioural contract row in a `.p.spec.md` table.
//
// A contract row is the one that ends in a fixture path — that column is what
// distinguishes it from every other table in the file, and it is also the exact
// promise being checked.
var contractRow = regexp.MustCompile("(?m)^\\| `([a-z][a-z0-9-]*)` \\|.*`testdata/evals/([a-z][a-z0-9-]*)/`")

// The guard that stops this debt coming back.
//
// Fifteen contracts were declared with a fixture path and no fixture, some of
// them for months: behavior-definition named ten and agent-loop five before
// there was a harness to run any of them. Nothing failed, because nothing
// compared the two.
//
// A threshold with no material is not a weak measurement — it is no measurement
// at all, wearing the appearance of one. Anyone reading the spec sees "≥ 95%,
// testdata/evals/tool-over-shell/" and reasonably concludes it is being
// measured.
func TestEveryDeclaredContractHasItsFixture(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	specs, err := filepath.Glob(filepath.Join(root, "docs", "specs", "architecture", "*", "*.p.spec.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) == 0 {
		t.Fatal("no .p specs found; the guard would pass vacuously")
	}

	found := 0
	for _, spec := range specs {
		data, err := os.ReadFile(spec)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range contractRow.FindAllStringSubmatch(string(data), -1) {
			id, path := m[1], m[2]
			found++
			if id != path {
				t.Errorf("%s: contract %q points at testdata/evals/%s/ — the id and the fixture must be the same name, or neither can be found from the other",
					filepath.Base(spec), id, path)
				continue
			}
			if _, err := LoadFixture(FixtureRoot, id); err != nil {
				t.Errorf("%s declares %q with a fixture path and the fixture does not load: %v",
					filepath.Base(spec), id, err)
			}
		}
	}
	if found == 0 {
		t.Fatal("no contract rows matched; the pattern has drifted from the table format and the guard is measuring nothing")
	}
	t.Logf("%d declared contracts, all with material", found)
}

// Every fixture must explain itself. A scenario directory with a task and a
// tool set but no note is material nobody can review — and reviewing the
// material is the only defence against a contract that measures the wrong
// thing.
func TestEveryFixtureExplainsWhatItMeasures(t *testing.T) {
	entries, err := os.ReadDir(FixtureRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		note, err := os.ReadFile(filepath.Join(FixtureRoot, e.Name(), "scenario.md"))
		if err != nil {
			t.Errorf("%s has no scenario.md: material nobody can review is material nobody checks", e.Name())
			continue
		}
		body := string(note)
		if !strings.Contains(body, e.Name()) {
			t.Errorf("%s: the note does not name the scenario it belongs to", e.Name())
		}
		if !strings.Contains(body, "limiar") {
			t.Errorf("%s: the note does not state the threshold, so a reader cannot tell what passing means", e.Name())
		}
	}
}

// Thirty of thirty-five fixtures were inert: declared with a threshold and a
// path, loaded by a test that checked only that they load, and measured by
// nothing. A threshold nobody runs is not a weak measurement, it is a claim.
func TestEveryFixtureHasAJudge(t *testing.T) {
	entries, err := os.ReadDir(FixtureRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		c, ok := ContractByID(e.Name())
		if !ok {
			t.Errorf("%s has material and no judge: it would load, pass the fixture test, and measure nothing", e.Name())
			continue
		}
		if c.Judge == nil && c.Measured() {
			t.Errorf("%s is in the table with a nil judge", e.Name())
		}
		if !c.Measured() {
			continue // settled by assertion; the rounds below describe a run it never makes
		}
		if c.Rounds < 1 {
			t.Errorf("%s asks for %d rounds", e.Name(), c.Rounds)
		}
		// The old rule here was that a multi-round scenario had to inject
		// something, which was true only because the loop refused to continue
		// without one — and that refusal is what capped every scenario at a
		// single round. Extra rounds now see the tool results, which is what
		// the product shows the model.
		//
		// The invariant that survives is the other direction: an injection
		// needs a round after it, or it never reaches the model at all.
		if c.Inject != "" && c.Rounds < 2 {
			t.Errorf("%s injects something and runs %d round(s), so the injection never reaches the model",
				e.Name(), c.Rounds)
		}
	}
}

// The other direction: a judge for material that does not exist measures
// nothing and hides that it does.
func TestEveryJudgeHasItsFixture(t *testing.T) {
	for _, c := range Contracts {
		if _, err := LoadFixture(FixtureRoot, c.ID); err != nil {
			t.Errorf("%s is judged and its material does not load: %v", c.ID, err)
		}
	}
}

// The threshold is written in two places — the spec table and the contract
// table — so the spec cannot be edited into disagreement with what runs.
func TestTheThresholdsAgreeWithTheSpecs(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	specs, err := filepath.Glob(filepath.Join(root, "docs", "specs", "architecture", "*", "*.p.spec.md"))
	if err != nil {
		t.Fatal(err)
	}

	declared := map[string]float64{}
	row := regexp.MustCompile("(?m)^\\| `([a-z][a-z0-9-]*)` \\|[^|]*\\|[^|]*\\|\\s*\\**≥?\\s*([0-9]+)%\\**\\s*\\|")
	for _, spec := range specs {
		data, err := os.ReadFile(spec)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range row.FindAllStringSubmatch(string(data), -1) {
			pct, err := strconv.Atoi(m[2])
			if err != nil {
				continue
			}
			declared[m[1]] = float64(pct) / 100
		}
	}
	if len(declared) == 0 {
		t.Fatal("no thresholds parsed from the specs; the pattern has drifted and the guard measures nothing")
	}

	for _, c := range Contracts {
		want, ok := declared[c.ID]
		if !ok {
			// The escape hatch this used to have — "declared only in a
			// changelog, the fixture guard covers it" — is what let a contract
			// live in the runner with no row in any `.p`. The fixture guard
			// walks the other direction and could never catch it.
			t.Errorf("%s runs at %.0f%% and no `.p` spec table declares it, so nothing keeps the two in agreement",
				c.ID, c.Threshold*100)
			continue
		}
		if c.Threshold != want {
			t.Errorf("%s: the spec says %.0f%% and the runner uses %.0f%%",
				c.ID, want*100, c.Threshold*100)
		}
	}
}

// A contract is answered by a model or by an assertion, and it must say which.
//
// Neither leaves an ID that nothing establishes. Both is the shape that started
// this: a contract carrying a judge that returned true unconditionally, sitting
// in the measured set, ready to spend twenty model calls printing MET at 100%
// without looking at the transcript.
func TestEveryContractIsEitherMeasuredOrAsserted(t *testing.T) {
	for _, c := range Contracts {
		switch {
		case c.Judge == nil && len(c.Asserted) == 0:
			t.Errorf("%s is declared and nothing establishes it: no judge, no assertion named", c.ID)
		case c.Judge != nil && len(c.Asserted) > 0:
			t.Errorf("%s carries a judge and names assertions; one of the two is not doing what it looks like", c.ID)
		}
	}
}

// A named assertion has to exist. A contract that points at a test that was
// renamed away is back to being established by nothing, and it says the
// opposite in the table.
func TestEveryNamedAssertionExists(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	have := map[string]bool{}
	decl := regexp.MustCompile(`(?m)^func (Test[A-Za-z0-9_]*)\(`)
	err = filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range decl.FindAllStringSubmatch(string(data), -1) {
			have[m[1]] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(have) == 0 {
		t.Fatal("no test names collected; the guard would pass vacuously")
	}

	for _, c := range Contracts {
		for _, name := range c.Asserted {
			if !have[name] {
				t.Errorf("%s names %s and no such test exists", c.ID, name)
			}
		}
	}
	t.Logf("%d contracts asserted deterministically, %d measured against a model",
		len(Contracts)-Measurable(Contracts), Measurable(Contracts))
}

// needsMaterial is the scenario whose question cannot be asked without the
// extra material, and which of the two kinds it needs.
//
// The map is explicit because the alternative is inference, and inference is
// what let this go unnoticed: nothing could tell that
// `follows-project-instruction` was being asked to follow an instruction that
// was never sent. It scored zero for twenty runs and read as a model problem.
var needsMaterial = map[string]string{
	"follows-project-instruction": "instructions",
	"directory-over-project":      "instructions",
	"skill-loaded-on-trigger":     "skills",
}

func TestAScenarioAboutMaterialShipsThatMaterial(t *testing.T) {
	for id, kind := range needsMaterial {
		f, err := LoadFixture(FixtureRoot, id)
		if err != nil {
			t.Errorf("%s: %v", id, err)
			continue
		}
		switch kind {
		case "instructions":
			if len(f.Instructions) == 0 {
				t.Errorf("%s is about following an instruction and ships none, so the model is asked to follow nothing", id)
			}
		case "skills":
			if len(f.Skills) == 0 {
				t.Errorf("%s is about a skill index and ships none", id)
			}
		}
	}
	// A scenario that stops needing material should leave this map, or the
	// guard becomes a list of names nobody maintains.
	for id := range needsMaterial {
		if _, ok := ContractByID(id); !ok {
			t.Errorf("%s is claimed here and is not a contract", id)
		}
	}
}

// The defect: everything arrived as a failed tool result whenever the model
// had called a tool, so a `<system-reminder>` reached the model as the read
// having failed. A model that then re-read looked like it had acted on the
// reminder when it had only retried.
func TestAReminderArrivesAsAReminderNextToASuccessfulResult(t *testing.T) {
	w := workspaceWith(t, map[string]string{"stats.go": "package stats\n"})
	c := Contract{Inject: "<system-reminder>stats.go changed</system-reminder>", InjectAs: InjectReminder}
	out := answers(context.Background(), w, c,
		[]ce.ToolCall{{ID: "c1", Name: "read", Input: []byte(`{"path":"stats.go"}`)}}, true)

	if len(out) != 2 {
		t.Fatalf("got %d messages, want a tool result and a reminder: %+v", len(out), out)
	}
	if out[0].ToolResult == nil || out[0].ToolResult.IsError {
		t.Errorf("the read was reported as having failed: %+v", out[0].ToolResult)
	}
	if !strings.Contains(out[0].ToolResult.Output, "package stats") {
		t.Errorf("the read returned %q, not the workspace's content", out[0].ToolResult.Output)
	}
	if !out[1].Reminder || out[1].Role != ce.RoleUser {
		t.Errorf("the reminder did not arrive as a reminder: %+v", out[1])
	}
}

// A tool error is still a tool error, and it lands on the call that failed.
func TestAToolErrorLandsOnTheCallAndNotBesideIt(t *testing.T) {
	w := workspaceWith(t, map[string]string{"stats.go": "package stats\n"})
	c := Contract{Inject: "old_string was not found", InjectAs: InjectToolError}
	out := answers(context.Background(), w, c, []ce.ToolCall{{ID: "c1", Name: "edit"}}, true)

	if len(out) != 1 {
		t.Fatalf("got %d messages, want just the failed result: %+v", len(out), out)
	}
	if out[0].ToolResult == nil || !out[0].ToolResult.IsError {
		t.Errorf("the error did not arrive as a failed result: %+v", out[0].ToolResult)
	}
}

// Every call is answered. An unanswered call is a malformed exchange, and the
// provider that tolerates it today is not a guarantee.
func TestEveryCallIsAnsweredAndOnlyTheFirstCarriesTheError(t *testing.T) {
	w := workspaceWith(t, map[string]string{"stats.go": "package stats\n"})
	c := Contract{Inject: "boom", InjectAs: InjectToolError}
	calls := []ce.ToolCall{
		{ID: "c1", Name: "read", Input: []byte(`{"path":"stats.go"}`)},
		{ID: "c2", Name: "grep", Input: []byte(`{"pattern":"package"}`)},
	}
	out := answers(context.Background(), w, c, calls, true)

	if len(out) != 2 {
		t.Fatalf("got %d results for %d calls", len(out), len(calls))
	}
	if !out[0].ToolResult.IsError {
		t.Error("the first call did not carry the error")
	}
	if out[1].ToolResult.IsError {
		t.Error("the second call carried the error too")
	}
	for i, m := range out {
		if m.ToolResult.ToolCallID != calls[i].ID {
			t.Errorf("result %d answers %q, want %q", i, m.ToolResult.ToolCallID, calls[i].ID)
		}
	}
}

// An error with no call to attach it to has nowhere to go. The reminder still
// arrives, because the product sends those whether or not a tool ran.
func TestWithNoCallTheInjectionStillReachesTheModel(t *testing.T) {
	w := workspaceWith(t, map[string]string{"stats.go": "package stats\n"})
	for _, as := range []Injection{InjectToolError, InjectReminder} {
		out := answers(context.Background(), w, Contract{Inject: "something", InjectAs: as}, nil, true)
		if len(out) != 1 || !out[0].Reminder {
			t.Errorf("injection %v produced %+v, want one reminder", as, out)
		}
	}
}

// A tool with no declared result still answers, or the exchange breaks on any
// scenario whose fixture did not think to name every tool.
func TestAToolWithNoDeclaredResultStillAnswers(t *testing.T) {
	w := workspaceWith(t, map[string]string{"stats.go": "package stats\n"})
	out := answers(context.Background(), w, Contract{Inject: "x", InjectAs: InjectReminder},
		[]ce.ToolCall{{ID: "c1", Name: "bash", Input: []byte(`{"command":"ls"}`)}}, true)
	if out[0].ToolResult.Output == "" {
		t.Error("the shell answered with nothing at all, which reads as a command that ran and printed nothing")
	}
}

// The guard that stops the delivery defect returning. A contract whose
// injected text is a reminder must say so, or the harness hands it to the
// model as a tool failure and measures a different behaviour with the same
// shape.
func TestEveryInjectedReminderIsDeclaredAsOne(t *testing.T) {
	for _, c := range Contracts {
		if c.Inject == "" {
			continue
		}
		isReminder := strings.Contains(c.Inject, "<system-reminder>")
		switch {
		case isReminder && c.InjectAs != InjectReminder:
			t.Errorf("%s injects a system-reminder and delivers it as a tool error", c.ID)
		case !isReminder && c.InjectAs == InjectReminder:
			t.Errorf("%s delivers a tool error as a reminder", c.ID)
		}
	}
}

// A contract that injects nothing must not say how to deliver it, or the field
// reads as a decision that was made when it never applied.
func TestAContractWithNothingToInjectDeclaresNoDelivery(t *testing.T) {
	for _, c := range Contracts {
		if c.Inject == "" && c.InjectAs != InjectToolError {
			t.Errorf("%s injects nothing and declares a delivery", c.ID)
		}
	}
}

// The injection happens once, on the round it belongs to. Repeating a reminder
// every round would measure a model being nagged, which is a different
// scenario from a model being told something once.
func TestAnInjectionHappensOnceAndOnlyOnce(t *testing.T) {
	w := workspaceWith(t, map[string]string{"stats.go": "package stats\n"})
	c := Contract{Inject: "<system-reminder>x</system-reminder>", InjectAs: InjectReminder}
	calls := []ce.ToolCall{{ID: "c1", Name: "read", Input: []byte(`{"path":"stats.go"}`)}}

	later := answers(context.Background(), w, c, calls, false)
	if len(later) != 1 {
		t.Fatalf("a later round produced %d messages, want just the tool result", len(later))
	}
	if later[0].Reminder {
		t.Error("the reminder was repeated on a later round")
	}
	if later[0].ToolResult == nil || later[0].ToolResult.IsError {
		t.Error("a later round reported the call as having failed")
	}
}

// workspaceWith builds a scenario workspace over a temp directory.
func workspaceWith(t *testing.T, files map[string]string) *Workspace {
	t.Helper()
	w, err := NewWorkspace(t.TempDir(), files)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// A judge must not name a tool its scenario does not offer.
//
// `does-not-delegate-trivial` judged NotCalled("explore") and its fixture did
// not offer `explore`. It scored 100% by having nothing to delegate to, which
// is not restraint — it is an empty room reported as good manners. Its sibling
// `delegates-wide-reads` had the opposite half of the same defect: it judged
// Called("explore", ...) and could only ever pass through the alternatives.
//
// Read from the source rather than from the judge, because a judge is a
// closure and its arguments are gone by the time anything can ask.
// judgesOnAbsence is the scenario whose whole question is a tool NOT being
// there, with the reason it is exempt.
//
// One entry, and it should stay hard to add: every other case of a judge naming
// an absent tool was a contract deciding its own verdict from its tool list.
var judgesOnAbsence = map[string]string{
	"no-phantom-tool": "the contract is that a name outside the offered set never reaches execution, " +
		"so the names it watches for are absent on purpose — that absence is the scenario",
}

func TestNoJudgeNamesAToolItsScenarioDoesNotOffer(t *testing.T) {
	src, err := os.ReadFile("contracts.go")
	if err != nil {
		t.Fatal(err)
	}
	known := map[string]bool{}
	for _, d := range ProductRegistry().Defs() {
		known[d.Name] = true
	}

	// Each contract's literal block, from its ID to the next one.
	blocks := splitContractBlocks(string(src))
	if len(blocks) < len(Contracts)/2 {
		t.Fatalf("only %d contract blocks parsed from %d contracts; the pattern has drifted and the guard is measuring nothing",
			len(blocks), len(Contracts))
	}
	quoted := regexp.MustCompile(`"([a-z_][a-z0-9_]*)"`)

	for id, body := range blocks {
		c, ok := ContractByID(id)
		if !ok || !c.Measured() {
			continue
		}
		if _, exempt := judgesOnAbsence[id]; exempt {
			continue
		}
		f, err := LoadFixture(FixtureRoot, id)
		if err != nil {
			t.Errorf("%s: %v", id, err)
			continue
		}
		offered := map[string]bool{}
		for _, n := range f.ToolNames() {
			offered[n] = true
		}
		for _, m := range quoted.FindAllStringSubmatch(body, -1) {
			name := m[1]
			if !known[name] || offered[name] {
				continue
			}
			t.Errorf("%s judges on %q and its scenario does not offer that tool, so the verdict is decided by the tool set rather than by the model. Offered: %v",
				id, name, f.ToolNames())
		}
	}
}

// splitContractBlocks cuts the contract table into one source block per ID.
//
// By ID boundary rather than by brace matching: the literals nest, and a regex
// that tried to balance them would silently match too little — which is how a
// guard ends up passing because it read nothing.
func splitContractBlocks(src string) map[string]string {
	head := regexp.MustCompile(`\{ID: "([a-z0-9-]+)"`)
	locs := head.FindAllStringSubmatchIndex(src, -1)
	out := map[string]string{}
	for i, loc := range locs {
		end := len(src)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		out[src[loc[2]:loc[3]]] = src[loc[1]:end]
	}
	return out
}

// An exemption for a scenario that no longer needs one is a note about a world
// that moved on, and left alone they become the reason nothing fails.
func TestNoStaleAbsenceExemption(t *testing.T) {
	for id, reason := range judgesOnAbsence {
		if _, ok := ContractByID(id); !ok {
			t.Errorf("judgesOnAbsence excuses %q, which is not a contract", id)
		}
		if len(reason) < 40 {
			t.Errorf("%q is exempt with a reason too short to be one: %q", id, reason)
		}
	}
}
