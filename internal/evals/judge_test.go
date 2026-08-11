package evals

import (
	"encoding/json"
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
