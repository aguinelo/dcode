package behavior

import (
	"strings"
	"testing"
)

func promptWith(t *testing.T, r *Repo) string {
	t.Helper()
	out, err := Build(Prompt{Doctrine: DefaultDoctrine([]string{"read", "edit", "bash"}), Repo: r}, FormulationFor(""))
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// The agent works in a repository and had no idea it was in one.
//
// Nothing put the branch, the state of the tree or the recent history in the
// prefix, so every turn started blind: it could not tell a clean tree from a
// dirty one, could not know it was on main, and could not follow a working
// agreement about branches without being told the branch each time.
//
// It is not a tool — bash already runs git. It is a fact the prefix carries,
// like the instruction chain, and for the same reason: a rule the model has to
// go and look up is a rule it follows when it remembers to.
func TestThePromptSaysWhereInTheRepositoryWeAre(t *testing.T) {
	out := promptWith(t, &Repo{
		Branch:     "fix/the-thing",
		MainBranch: "main",
		Clean:      false,
		Status:     " M internal/app/app.go\n?? notes.txt",
		Commits:    []string{"abc1234 fix: something", "def5678 test: something else"},
	})

	for _, want := range []string{
		"fix/the-thing",
		"main",
		"internal/app/app.go",
		"abc1234",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the prompt does not carry %q:\n%s", want, out)
		}
	}
}

// The snapshot is taken once and the repository moves under it — the agent's
// own commits move it. Saying so is the difference between a fact and a lie
// that was true at breakfast.
func TestTheRepositorySnapshotSaysItIsASnapshot(t *testing.T) {
	out := promptWith(t, &Repo{Branch: "main", MainBranch: "main", Clean: true})
	if !strings.Contains(strings.ToLower(out), "snapshot") {
		t.Errorf("the prompt presents stale state as current:\n%s", out)
	}
}

// A clean tree is worth saying out loud rather than leaving as an empty status
// the model has to interpret. "Nothing here" and "I did not look" read the same
// when both are blank.
func TestACleanTreeIsStatedRatherThanLeftBlank(t *testing.T) {
	out := promptWith(t, &Repo{Branch: "main", MainBranch: "main", Clean: true})
	if !strings.Contains(strings.ToLower(out), "clean") {
		t.Errorf("a clean tree is not stated:\n%s", out)
	}
}

// A workspace with no repository says so, once.
//
// It used to say nothing: nil rendered no section, and the invariant claimed
// the prefix carries "nothing when it is not" a repository. An agent then
// worked a full day in exactly that state — writing a project file of its own
// that demanded a commit per task and a pull request per spec — and nothing
// ever told it that none of that was possible.
//
// The line is a fact, not a warning: no diff, no review, no undo. What the
// agent does with it is work anyway, having said it.
func TestAWorkspaceWithNoRepositorySaysSo(t *testing.T) {
	out := promptWith(t, &Repo{Absent: true})

	low := strings.ToLower(out)
	if !strings.Contains(low, "not a git repository") {
		t.Errorf("the absence of a repository is not stated:\n%s", out)
	}
	for _, want := range []string{"history", "git init"} {
		if !strings.Contains(low, want) {
			t.Errorf("the line does not mention %q:\n%s", want, out)
		}
	}
}

// The absent case must not claim a branch, a tree state or a history it does
// not have. Inventing "main" here is how an agent commits to a repository that
// is not there.
func TestAnAbsentRepositoryClaimsNothingElse(t *testing.T) {
	out := promptWith(t, &Repo{Absent: true})
	for _, unwanted := range []string{"Current branch:", "Main branch:", "Recent commits:", "Working tree:"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("an absent repository claimed %q:\n%s", unwanted, out)
		}
	}
}

// Nil is the snapshot that was never taken, and it stays silent.
//
// "We did not look" and "we looked and there is none" are different facts, and
// only the second one is worth a line. Collapsing them would put a claim about
// the workspace in the prefix on the strength of never having checked — which
// is the defect the Absent field exists to remove.
func TestASnapshotThatWasNeverTakenSaysNothing(t *testing.T) {
	out := promptWith(t, nil)
	if strings.Contains(strings.ToLower(out), "repository") {
		t.Errorf("a snapshot that was never taken made a claim:\n%s", out)
	}
	if strings.Contains(strings.ToLower(out), "branch") {
		t.Errorf("a snapshot that was never taken got a git section:\n%s", out)
	}
}

// Build is pure and has to stay pure: the same session must produce a
// byte-identical prefix, or every cached prefix and every reproducibility claim
// in the context engine goes with it.
func TestTheRepositorySectionIsPure(t *testing.T) {
	r := &Repo{
		Branch: "main", MainBranch: "main", Clean: false,
		Status:  " M a.go",
		Commits: []string{"abc1234 one", "def5678 two"},
	}
	first := promptWith(t, r)
	for i := 0; i < 5; i++ {
		if got := promptWith(t, r); got != first {
			t.Fatal("the same repository state produced a different prefix")
		}
	}
}

// A repository mid-rebase or on a detached head has no branch name, and the
// prefix must not claim one. Inventing "main" there is how an agent commits to
// the wrong place.
func TestADetachedHeadIsNotGivenABranchName(t *testing.T) {
	out := promptWith(t, &Repo{MainBranch: "main", Clean: true, Detached: true})
	if strings.Contains(out, "Current branch:") {
		t.Errorf("a detached head was reported as being on a branch:\n%s", out)
	}
	if !strings.Contains(strings.ToLower(out), "detached") {
		t.Errorf("a detached head is not stated:\n%s", out)
	}
}
