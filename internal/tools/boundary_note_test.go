package tools

import (
	"context"
	"strings"
	"testing"
)

// modeRunner is a runner that also reports the boundary it ran under, which is
// what the bash tool asks for through an anonymous interface.
type modeRunner struct {
	out  string
	code int
	mode string
}

func (m modeRunner) Run(context.Context, string, string) (string, int, error) {
	return m.out, m.code, nil
}
func (m modeRunner) SandboxMode() string { return m.mode }

// TestAWallSaysHowItOpens is the promise this closes.
//
// The doctrine tells the model to attempt rather than refuse, and to let the
// boundary ask. For a path crossing inside a shell command nothing asks: the
// command is opaque, so bash declares the workspace as what it writes and the
// policy sees no crossing to escalate. Observed in a real session, the model
// did exactly as told, was refused, and then told the user — in good faith —
// that the harness would ask. The user waited for a question never coming.
//
// A wall that cannot become a question must at least say what opens it.
func TestAWallSaysHowItOpens(t *testing.T) {
	s, _ := setup(t)
	r := modeRunner{out: "mkdir: /x: Operation not permitted\n", code: 1, mode: "workspace-write"}
	res := run(t, Bash{Runner: r}, s, BashInput{Command: "mkdir -p /x"})

	if !strings.Contains(res.Output, "/mode auto") {
		t.Errorf("the way through is not named:\n%s", res.Output)
	}
	if !strings.Contains(res.Output, "sandbox.writable") {
		t.Errorf("the other way through is not named:\n%s", res.Output)
	}
	if !strings.Contains(res.Output, "Do not tell them a prompt is coming") {
		t.Errorf("nothing stops the model repeating the promise:\n%s", res.Output)
	}
	if !strings.Contains(res.Output, "Operation not permitted") {
		t.Errorf("the note replaced the output instead of following it:\n%s", res.Output)
	}
}

// TestAnOrdinaryFailureGetsNoNote keeps the note narrow.
//
// A command can fail for a hundred reasons of the workspace's own, and telling
// someone the sandbox did it would send them somewhere wrong.
func TestAnOrdinaryFailureGetsNoNote(t *testing.T) {
	s, _ := setup(t)
	for _, c := range []struct {
		name string
		r    modeRunner
	}{
		{"exit zero", modeRunner{out: "Operation not permitted\n", code: 0, mode: "workspace-write"}},
		{"eacces, not eperm", modeRunner{out: "cp: /x: Permission denied\n", code: 1, mode: "workspace-write"}},
		{"no boundary to blame", modeRunner{out: "mkdir: /x: Operation not permitted\n", code: 1, mode: "full-access"}},
		{"ordinary failure", modeRunner{out: "FAIL\tpkg 0.1s\n", code: 1, mode: "workspace-write"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			res := run(t, Bash{Runner: c.r}, s, BashInput{Command: "x"})
			if strings.Contains(res.Output, "/mode auto") {
				t.Errorf("a note was attached where the sandbox is not to blame:\n%s", res.Output)
			}
		})
	}
}

// TestARunnerThatCannotSayItsModeStillGetsTheNote: the mode is a refinement,
// not a precondition. A runner that does not report one is the ordinary case
// in tests and in any embedder, and losing the note there would make the fix
// depend on an optional method.
func TestARunnerThatCannotSayItsModeStillGetsTheNote(t *testing.T) {
	s, _ := setup(t)
	r := &fakeRunner{out: "mkdir: /x: Operation not permitted\n", code: 1}
	res := run(t, Bash{Runner: r}, s, BashInput{Command: "mkdir -p /x"})

	if !strings.Contains(res.Output, "/mode auto") {
		t.Errorf("no note without a mode source:\n%s", res.Output)
	}
}
