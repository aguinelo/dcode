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
