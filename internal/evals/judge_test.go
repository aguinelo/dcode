package evals

import (
	"encoding/json"
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

func tr(text string, calls ...ce.ToolCall) Transcript {
	return Transcript{Calls: calls, Text: text, Rounds: 1}
}

func c(name, input string) ce.ToolCall {
	return ce.ToolCall{Name: name, Input: json.RawMessage(input)}
}

func TestCalledAndNotCalled(t *testing.T) {
	x := tr("", c("read", `{"path":"a.go"}`), c("edit", `{}`))
	if !Called("read")(x) || !Called("bash", "edit")(x) {
		t.Error("Called missed a call that is there")
	}
	if Called("bash")(x) {
		t.Error("Called found one that is not")
	}
	if !NotCalled("bash")(x) || NotCalled("read")(x) {
		t.Error("NotCalled is wrong")
	}
}

func TestCalledWithNeedsEveryFragment(t *testing.T) {
	x := tr("", c("read", `{"path":"internal/config/toml.go","limit":40}`))
	if !CalledWith("read", "toml.go", "limit")(x) {
		t.Error("both fragments are present and it said no")
	}
	if CalledWith("read", "toml.go", "offset")(x) {
		t.Error("a missing fragment still matched")
	}
	if CalledWith("grep", "toml.go")(x) {
		t.Error("matched the wrong tool")
	}
}

// Order is the whole assertion in the contracts about re-reading: a run that
// edits and then reads has not recovered.
func TestCalledBeforeRequiresTheOrder(t *testing.T) {
	good := tr("", c("read", `{}`), c("edit", `{}`))
	bad := tr("", c("edit", `{}`), c("read", `{}`))
	if !CalledBefore("read", "edit")(good) {
		t.Error("read then edit was rejected")
	}
	if CalledBefore("read", "edit")(bad) {
		t.Error("edit then read was accepted")
	}
	if CalledBefore("read", "edit")(tr("", c("read", `{}`))) {
		t.Error("only the first call present was accepted")
	}
}

func TestSaysIsCaseInsensitiveAndSaysNoneInverts(t *testing.T) {
	x := tr("I could NOT verify this.")
	if !Says("could not verify")(x) {
		t.Error("Says missed a case difference")
	}
	if !SaysNone("everything passed")(x) || SaysNone("could not verify")(x) {
		t.Error("SaysNone is wrong")
	}
}

func TestCallCountRange(t *testing.T) {
	x := tr("", c("read", `{}`), c("read", `{}`), c("read", `{}`))
	if !CallCount("read", 1, 0)(x) {
		t.Error("no upper bound rejected three calls")
	}
	if CallCount("read", 1, 2)(x) {
		t.Error("three calls passed a ceiling of two")
	}
	if !CallCount("bash", 0, 0)(x) {
		t.Error("zero calls to an absent tool failed a minimum of zero")
	}
}

// The property the repeat detector cannot see: it needs input to be
// byte-identical, and a model trying to escape varies the attempt each round
// while repeating the same conceptual error.
func TestDistinctIgnoresWhitespaceOnlyDifferences(t *testing.T) {
	same := tr("",
		c("edit", `{"old_string":"a b"}`),
		c("edit", `{"old_string":"a b"}   `),
		c("edit", "{\"old_string\":\"a b\"}\n"),
	)
	if Distinct("edit", 2)(same) {
		t.Error("three attempts differing only in whitespace counted as distinct")
	}
	real := tr("",
		c("edit", `{"old_string":"a b"}`),
		c("edit", `{"old_string":"c d"}`),
	)
	if !Distinct("edit", 2)(real) {
		t.Error("two genuinely different attempts were not counted")
	}
}

func TestAllAndAny(t *testing.T) {
	x := tr("done", c("read", `{}`))
	yes := func(Transcript) bool { return true }
	no := func(Transcript) bool { return false }

	if !All(yes, yes)(x) || All(yes, no)(x) {
		t.Error("All is wrong")
	}
	if !Any(no, yes)(x) || Any(no, no)(x) {
		t.Error("Any is wrong")
	}
}

// CalledWithout is not the negation of CalledWith: a model that wrote nothing
// must not satisfy "did not write the wrong thing" by having written nothing.
func TestCalledWithoutNeedsTheCallToHaveHappened(t *testing.T) {
	empty := Transcript{}
	if CalledWithout("write", "Must")(empty) {
		t.Error("a transcript with no calls satisfied CalledWithout")
	}

	right := Transcript{Calls: []ce.ToolCall{
		{Name: "write", Input: []byte(`{"content":"func legacyTrim() {}"}`)},
	}}
	if !CalledWithout("write", "Must")(right) {
		t.Error("a call that avoided the fragment was rejected")
	}

	wrong := Transcript{Calls: []ce.ToolCall{
		{Name: "write", Input: []byte(`{"content":"func MustTrim() {}"}`)},
	}}
	if CalledWithout("write", "Must")(wrong) {
		t.Error("a call carrying the fragment was accepted")
	}
}

// Every call has to avoid it, not just one. A model that wrote the wrong
// convention and then wrote the right one has still written the wrong one.
func TestCalledWithoutChecksEveryCall(t *testing.T) {
	both := Transcript{Calls: []ce.ToolCall{
		{Name: "write", Input: []byte(`{"content":"func legacyTrim() {}"}`)},
		{Name: "write", Input: []byte(`{"content":"func MustTrim() {}"}`)},
	}}
	if CalledWithout("write", "Must")(both) {
		t.Error("one clean call was enough to pass while another carried the fragment")
	}
}

// The judge has to see only what happened after the product spoke. A model
// that read once at the start and then edited without re-reading has ignored
// the reminder, and judging the whole transcript scores it as a pass.
func TestSinceInjectionIgnoresWhatCameBefore(t *testing.T) {
	ignored := Transcript{
		Calls: []ce.ToolCall{
			{Name: "read"}, // first round, before the reminder
			{Name: "edit"}, // second round, straight to the edit
		},
		InjectedAt: 1,
	}
	if SinceInjection(CalledBefore("read", "edit"))(ignored) {
		t.Error("editing without re-reading passed the reminder contract")
	}

	acted := Transcript{
		Calls: []ce.ToolCall{
			{Name: "read"},
			{Name: "read"}, {Name: "edit"},
		},
		InjectedAt: 1,
	}
	if !SinceInjection(CalledBefore("read", "edit"))(acted) {
		t.Error("re-reading before editing failed the reminder contract")
	}
}

// Nothing injected means nothing to narrow to, and the judge must not silently
// see an empty transcript.
func TestSinceInjectionWithNoInjectionSeesEverything(t *testing.T) {
	whole := Transcript{Calls: []ce.ToolCall{{Name: "read"}, {Name: "edit"}}}
	if !SinceInjection(Called("read"))(whole) {
		t.Error("a transcript with no injection lost its calls")
	}
}

// An index past the end would panic on a slice, and a harness that panics
// mid-measurement loses every result collected so far.
func TestSinceInjectionSurvivesAnImpossibleIndex(t *testing.T) {
	for _, at := range []int{-3, 99} {
		tr := Transcript{Calls: []ce.ToolCall{{Name: "read"}}, InjectedAt: at}
		if !SinceInjection(Called("read"))(tr) {
			t.Errorf("InjectedAt %d lost the calls", at)
		}
	}
}

// A failing contract has to show what happened. `0.0% of 20 runs` reads as a
// model problem and is just as often a scenario that cannot reach the
// behaviour it judges — and the call sequence is what separates them.
func TestTheDigestShowsTheCallsAndWhereTheProductSpoke(t *testing.T) {
	tr := Transcript{
		Rounds:     2,
		Calls:      []ce.ToolCall{{Name: "read"}, {Name: "edit"}},
		InjectedAt: 1,
		Text:       "  I read   the file\nand edited it. ",
	}
	got := tr.Digest()
	for _, want := range []string{"2 round(s)", "read", "edit", "product spoke", "I read the file and edited it."} {
		if !strings.Contains(got, want) {
			t.Errorf("the digest lost %q:\n%s", want, got)
		}
	}
	// The marker sits between the two calls, not before both.
	if strings.Index(got, "product spoke") < strings.Index(got, "read") {
		t.Errorf("the marker is before the first call:\n%s", got)
	}
}

// A run that called nothing is the most common failure and the easiest to
// misread as missing output.
func TestTheDigestSaysWhenNothingWasCalled(t *testing.T) {
	got := Transcript{Rounds: 1, Text: "I cannot do that."}.Digest()
	if !strings.Contains(got, "no tool calls") {
		t.Errorf("a run with no calls does not say so:\n%s", got)
	}
	if strings.Contains(got, "product spoke") {
		t.Errorf("nothing was injected and the digest says otherwise:\n%s", got)
	}
}

// A long answer must not bury the next failure in the log.
func TestTheDigestTrimsALongAnswer(t *testing.T) {
	got := Transcript{Rounds: 1, Text: strings.Repeat("x", digestText*3)}.Digest()
	if len(got) > digestText+120 {
		t.Errorf("the digest is %d chars, long enough to bury the next one", len(got))
	}
	if !strings.Contains(got, "…") {
		t.Errorf("the answer was cut without saying so:\n%s", got)
	}
}

// The tool name alone leaves the question half answered. `bash` running `cat`
// is the contract being broken; `bash` running `ls` is an agent looking around.
// Those are opposite findings and the name is the same in both.
func TestTheDigestShowsWhatTheCallAskedFor(t *testing.T) {
	got := Transcript{Rounds: 1, Calls: []ce.ToolCall{
		{Name: "bash", Input: []byte(`{"command": "cat internal/config/toml.go"}`)},
	}}.Digest()

	if !strings.Contains(got, "cat internal/config/toml.go") {
		t.Errorf("the digest does not say what bash was asked to run:\n%s", got)
	}
}

// A call with no arguments must not print empty brackets, which read as an
// argument that was there and got lost.
func TestTheDigestOmitsEmptyArguments(t *testing.T) {
	for _, input := range [][]byte{nil, []byte("{}"), []byte("  ")} {
		got := Transcript{Rounds: 1, Calls: []ce.ToolCall{{Name: "plan", Input: input}}}.Digest()
		if strings.Contains(got, "()") {
			t.Errorf("input %q printed empty brackets:\n%s", input, got)
		}
	}
}

// A write carries a whole file. Printing it would bury every other call in the
// run, and the digest exists to be read at a glance.
func TestTheDigestTrimsALongArgument(t *testing.T) {
	body := strings.Repeat("x", previewArg*4)
	got := Transcript{Rounds: 1, Calls: []ce.ToolCall{
		{Name: "write", Input: []byte(`{"content":"` + body + `"}`)},
	}}.Digest()

	if len(got) > previewArg+80 {
		t.Errorf("the digest is %d chars, long enough to bury the rest of the run", len(got))
	}
	if !strings.Contains(got, "…") {
		t.Errorf("the argument was cut without saying so:\n%s", got)
	}
}

// A model answering "I can't do this. Let me explain what's blocking it"
// refused, explained, and named the blocker — the contract, honoured — and
// scored zero because the judge looked for "cannot". Substring matching does
// not see through an apostrophe, and models contract almost everything.
func TestSaysSeesThroughAContraction(t *testing.T) {
	said := Transcript{Text: "I can't do this. The database isn't running and I didn't run the suite."}

	for _, fragment := range []string{"cannot", "is not", "did not"} {
		if !Says(fragment)(said) {
			t.Errorf("Says(%q) missed its contracted form in: %q", fragment, said.Text)
		}
	}
}

// The written-out form still matches, or the expansion has replaced one blind
// spot with another.
func TestSaysStillSeesTheWrittenOutForm(t *testing.T) {
	said := Transcript{Text: "I cannot reach the database and did not run the suite."}
	for _, fragment := range []string{"cannot", "did not"} {
		if !Says(fragment)(said) {
			t.Errorf("Says(%q) missed the written-out form", fragment)
		}
	}
}

// Expansion must not invent a match. A judge that fires on text that says the
// opposite is worse than one that misses.
func TestSaysDoesNotInventAMatch(t *testing.T) {
	said := Transcript{Text: "Everything passed and the suite is green."}
	for _, fragment := range []string{"cannot", "could not", "did not pass"} {
		if Says(fragment)(said) {
			t.Errorf("Says(%q) matched text that says the opposite: %q", fragment, said.Text)
		}
	}
}

// SaysNone inherits the expansion, which matters more there: a contract that
// forbids a false claim of success must not be passed by a contraction.
func TestSaysNoneAlsoSeesThroughAContraction(t *testing.T) {
	said := Transcript{Text: "It doesn't matter, everything works now."}
	if SaysNone("works now")(said) {
		t.Error("SaysNone missed a phrase that is present verbatim")
	}
}

// Not opening a shell at all is the cleanest way to honour "did not verify",
// and a judge that needed a shell call in order to inspect it would fail the
// best possible run. That is the difference from CalledWithout.
func TestNeverCalledWithIsTrueWhenTheToolWasNeverCalled(t *testing.T) {
	if !NeverCalledWith("bash", "test")(Transcript{}) {
		t.Error("a run that never opened a shell failed a contract about not running tests")
	}
	if !NeverCalledWith("bash", "test")(tr("", ce.ToolCall{Name: "read"})) {
		t.Error("a run that only read failed it too")
	}
}

// Orientation is not verification. `ls` answers a different question than
// "did it run the suite", and conflating them made a real finding — reaching
// for the shell to look around — land on the wrong contract.
func TestNeverCalledWithSeparatesOrientationFromVerification(t *testing.T) {
	looking := tr("", ce.ToolCall{Name: "bash", Input: []byte(`{"command":"ls -la"}`)})
	if !NeverCalledWith("bash", "test", "make")(looking) {
		t.Error("listing the workspace counted as running the tests")
	}

	verifying := tr("", ce.ToolCall{Name: "bash", Input: []byte(`{"command":"go test ./..."}`)})
	if NeverCalledWith("bash", "test", "make")(verifying) {
		t.Error("running the tests was not noticed")
	}
}

// wrote builds a transcript containing one write of content to path.
func wrote(path, content string) Transcript {
	in, err := json.Marshal(map[string]string{"path": path, "content": content})
	if err != nil {
		panic(err)
	}
	return Transcript{Calls: []ce.ToolCall{{Name: "write", Input: in}}}
}

func TestWroteFileReadsTheContentAndNotTheRawArguments(t *testing.T) {
	judge := WroteFile("DCODE.md", SaysAll("hello"))

	if !judge(wrote("DCODE.md", "hello there")) {
		t.Error("did not see content it was given")
	}
	// The fragment is in the path, not in the file. CalledWith cannot tell the
	// difference, which is the whole reason this decodes.
	if judge(wrote("hello/DCODE.md", "nothing here")) {
		t.Error("matched the path as if it were the content")
	}
	if judge(wrote("NOTES.md", "hello there")) {
		t.Error("judged a file the contract is not about")
	}
	if judge(Transcript{}) {
		t.Error("a run that wrote nothing satisfied a judge about what was written")
	}
}

func TestWroteFileIgnoresACallItCannotDecode(t *testing.T) {
	judge := WroteFile("DCODE.md", SaysAll("hello"))
	bad := Transcript{Calls: []ce.ToolCall{{Name: "write", Input: []byte(`{not json`)}}}
	if judge(bad) {
		t.Error("undecodable arguments were read as a passing write")
	}
}

// The three init contracts scored identically for months because their judges
// were identical. This is the test that says they are not any more: each
// content passes exactly one of them.
func TestTheThreeInitJudgesDisagreeWithEachOther(t *testing.T) {
	have := ProductRegistry().Names()
	dropsTool := WroteFile("DCODE.md", NamesNoToolThatDoesNotExist(have))
	dropsCommand := WroteFile("DCODE.md", SaysNoneOf("npm run build", "npm install"))
	keepsConvention := WroteFile("DCODE.md", Both(
		SaysAll("50"),
		SaysAny("doc comment", "doc comments", "documentation comment", "godoc"),
	))

	// Carries the tool that does not exist, drops everything else.
	carriesTool := wrote("DCODE.md", "Use the Task tool to spawn subagents.")
	if dropsTool(carriesTool) {
		t.Error("a DCODE.md ordering a tool dcode does not have was accepted")
	}
	if !dropsCommand(carriesTool) {
		t.Error("the command judge reacted to a tool problem")
	}

	// Carries the command that cannot run here.
	carriesCommand := wrote("DCODE.md", "Always run `npm run build` before reporting done.")
	if dropsCommand(carriesCommand) {
		t.Error("a build command with no package.json was accepted")
	}
	if !dropsTool(carriesCommand) {
		t.Error("the tool judge reacted to a command problem")
	}

	// Drops the user's real rules along with the noise — the naive fix.
	overEager := wrote("DCODE.md", "Build with `go build ./...`.")
	if keepsConvention(overEager) {
		t.Error("a DCODE.md that deleted the user's conventions was accepted")
	}
	if !dropsTool(overEager) || !dropsCommand(overEager) {
		t.Error("the discard judges punished a clean discard")
	}

	// What a correct translation looks like.
	good := wrote("DCODE.md", "Build with `go build ./...`.\n\n"+
		"Keep functions under 50 lines.\nEvery exported symbol carries a doc comment starting with its name.")
	for name, j := range map[string]Judge{
		"drops-absent-tool": dropsTool, "drops-absent-command": dropsCommand,
		"keeps-real-convention": keepsConvention,
	} {
		if !j(good) {
			t.Errorf("%s rejected a correct translation", name)
		}
	}
}

func TestSaysAllAndSaysAnyIgnoreCase(t *testing.T) {
	if !SaysAll("Doc Comment")("every symbol carries a doc comment") {
		t.Error("SaysAll is case-sensitive")
	}
	if SaysAll("a", "b")("only a") {
		t.Error("SaysAll passed without every fragment")
	}
	if !SaysAny("godoc", "doc comment")("a DOC COMMENT on each") {
		t.Error("SaysAny is case-sensitive")
	}
	if SaysAny("x", "y")("neither") {
		t.Error("SaysAny passed with no fragment present")
	}
	if !SaysNoneOf("npm")("go build ./...") {
		t.Error("SaysNoneOf rejected clean text")
	}
	if SaysNoneOf("NPM")("run npm install") {
		t.Error("SaysNoneOf is case-sensitive")
	}
}
