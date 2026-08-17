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

// Not every directory is a repository, and that is ordinary rather than a
// problem. No section at all beats a section saying nothing.
func TestNoRepositoryMeansNoSection(t *testing.T) {
	out := promptWith(t, nil)
	if strings.Contains(strings.ToLower(out), "branch") {
		t.Errorf("a directory that is not a repository got a git section:\n%s", out)
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
