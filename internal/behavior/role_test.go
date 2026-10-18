package behavior

import (
	"strings"
	"testing"
)

// The report this file answers: a live orchestrator on a real model read the
// repository, said the terminal was read-only so neither it nor its children
// could write, and asked the person to switch the sandbox mode. It never called
// `delegate`. Nothing in its prompt said what an orchestrator is; the only
// guidance it had was the read-only block, which tells an ordinary session to
// say what it needs and let the person change the mode.

// switchAdvice is the read-only block's advice to an ordinary session, which is
// the sentence the orchestrator followed.
const switchAdvice = "let the person decide whether to change the mode"

func rolePrompt(t *testing.T, role string, b *Boundary) string {
	t.Helper()
	p := Prompt{Doctrine: DefaultDoctrine([]string{"read", "delegate"}), Role: role, Boundary: b}
	got, err := Build(p, FormulationFor("minimax-m3"))
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestTheOrchestratorIsToldItsRoleAndNotToAskForTheMode(t *testing.T) {
	got := rolePrompt(t, RoleOrchestrator, &Boundary{Mode: BoundaryReadOnly})
	for _, want := range []string{
		"## Your role",
		"You are this project's orchestrator",
		"never ask anyone to change the sandbox mode",
		"own git worktree",
		"`owns`",
		"call `delegate`",
		"disjoint",
		"You do not commit, run the gate, open pull requests or merge",
		"origin `children`",
		"`steer_child`", "`close_child`", "`ask`",
		"Answer the person at once",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the orchestrator's prompt does not say %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, switchAdvice) {
		t.Errorf("the orchestrator's prompt still tells it to let the person change the mode:\n%s", got)
	}
	if !strings.Contains(got, "denied rather than put to anyone") {
		t.Errorf("the orchestrator's boundary no longer says whether anyone is asked:\n%s", got)
	}
}

func TestTheChildIsToldItsRole(t *testing.T) {
	got := rolePrompt(t, RoleChild, &Boundary{Mode: BoundaryWorkspaceWrite, Asks: true})
	for _, want := range []string{
		"## Your role",
		"own git worktree",
		"only inside the paths you own",
		"Run no git that writes",
		"The harness commits your turn",
		"`red_test`",
		"a claim, not evidence",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the child's prompt does not say %q:\n%s", want, got)
		}
	}
	// The phrase the scripted scenarios tell a child's first turn by: the
	// prefix must not carry it, or every request of a child reads as one.
	if strings.Contains(got, "You are a child of an orchestrator") {
		t.Error("the child's prefix carries the first turn's marker phrase")
	}
}

func TestTheReviewerIsToldItsRoleAndNotToAskForTheMode(t *testing.T) {
	got := rolePrompt(t, RoleReviewer, &Boundary{Mode: BoundaryReadOnly})
	for _, want := range []string{"## Your role", "reviewer", "only read", "calling `verdict`"} {
		if !strings.Contains(got, want) {
			t.Errorf("the reviewer's prompt does not say %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, switchAdvice) {
		t.Errorf("the reviewer's prompt still tells it to let the person change the mode:\n%s", got)
	}
}

// A session with no role is every session that is not orchestrated, and its
// prefix is the one the goldens record: a role section must cost nothing there.
func TestASessionWithNoRoleHasNoRoleSection(t *testing.T) {
	got := rolePrompt(t, "", &Boundary{Mode: BoundaryReadOnly})
	if strings.Contains(got, "Your role") {
		t.Errorf("a session with no role was given a role section:\n%s", got)
	}
	if !strings.Contains(got, switchAdvice) {
		t.Errorf("an ordinary read-only session lost its advice:\n%s", got)
	}
}

// An unknown role is a misconfiguration made visible at assembly, the same as
// a doctrine with no identity — never a session quietly given no role.
func TestAnUnknownRoleRefusesToAssemble(t *testing.T) {
	p := Prompt{Doctrine: DefaultDoctrine([]string{"read"}), Role: "conductor"}
	if _, err := Build(p, FormulationFor("minimax-m3")); err == nil {
		t.Error("a prompt with an unknown role was assembled")
	}
}

// The role sits right after the boundary: it is how the boundary is read.
func TestTheRoleIsRenderedRightAfterTheBoundary(t *testing.T) {
	got := rolePrompt(t, RoleOrchestrator, &Boundary{Mode: BoundaryReadOnly})
	boundary := strings.Index(got, "## The boundary right now")
	role := strings.Index(got, "## Your role")
	floor := strings.Index(got, "## How this works by default")
	if boundary < 0 || role < 0 || floor < 0 || !(boundary < role && role < floor) {
		t.Errorf("want boundary < role < floor, got %d, %d, %d", boundary, role, floor)
	}
}

func TestGoldenPromptOrchestrator(t *testing.T) {
	f := goldenFixture()
	f.Role, f.Boundary = RoleOrchestrator, &Boundary{Mode: BoundaryReadOnly}
	got, err := Build(f, FormulationFor("minimax-m3"))
	if err != nil {
		t.Fatal(err)
	}
	goldenPrompt(t, "prompt_orchestrator", got)
}
