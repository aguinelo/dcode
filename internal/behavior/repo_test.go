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
	// Named claims, not a word scan. This used to sweep the whole prefix for
	// "repository", which was fine until the doctrine's floor said "a summary
	// written before the edits describes the repository as it was" — a
	// sentence about writing summaries, failing a test about git. A guard that
	// matches a loose string is a guard that goes wrong when the world moves,
	// and this one did.
	for _, claim := range []string{"This workspace", "not a git repository", "Current branch:", "Main branch:"} {
		if strings.Contains(out, claim) {
			t.Errorf("a snapshot that was never taken carried %q:\n%s", claim, out)
		}
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

// The declared gates reach the prefix as facts, so the agent does not have to
// read package.json to find out the project has a coverage gate.
//
// The audited project declared four and two had been red since the first day.
// Nobody ran them, and the one that was green measured that one plus one is
// two. Naming them is the floor of that; measuring them is done-qualifier.
func TestTheDeclaredGatesReachThePrefix(t *testing.T) {
	out, err := Build(Prompt{
		Doctrine: DefaultDoctrine([]string{"read"}),
		Workspace: &Workspace{Gates: []Gate{
			{Name: "test", Command: "vitest run", Source: "package.json"},
			{Name: "test:coverage", Command: "vitest run --coverage", Source: "package.json"},
		}},
	}, FormulationFor(""))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"test:coverage", "vitest run --coverage", "This workspace"} {
		if !strings.Contains(out, want) {
			t.Errorf("the prefix does not carry %q:\n%s", want, out)
		}
	}
}

// The sentence that keeps a list of gates from reading as a list of
// guarantees. Without it this section would have produced the very defect that
// asked for it.
func TestTheGateListSaysNothingHasRunThem(t *testing.T) {
	out, err := Build(Prompt{
		Doctrine:  DefaultDoctrine([]string{"read"}),
		Workspace: &Workspace{Gates: []Gate{{Name: "test", Command: "go test ./...", Source: "Makefile"}}},
	}, FormulationFor(""))
	if err != nil {
		t.Fatal(err)
	}
	low := strings.ToLower(out)
	if !strings.Contains(low, "nothing here says they pass") {
		t.Errorf("the gate list does not disclaim passing:\n%s", out)
	}
	if !strings.Contains(low, "nothing has run them") {
		t.Errorf("the gate list does not say nothing ran them:\n%s", out)
	}
}

// A project that declares no gate gets no section, and nothing in the prefix
// claims it declares none.
//
// The distinction from an absent repository is consequence: having no
// repository changes what finishing the work means, while declaring no gate is
// ordinary. Not every absence earns a line — what must not happen is an
// absence nobody checked becoming a claim.
func TestNoDeclaredGatesMeansNoClaim(t *testing.T) {
	for name, ws := range map[string]*Workspace{
		"nil":   nil,
		"empty": {},
	} {
		out, err := Build(Prompt{Doctrine: DefaultDoctrine([]string{"read"}), Workspace: ws}, FormulationFor(""))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.ToLower(out), "declares its own checks") {
			t.Errorf("%s workspace rendered a gate section:\n%s", name, out)
		}
		if strings.Contains(strings.ToLower(out), "no checks") || strings.Contains(strings.ToLower(out), "declares none") {
			t.Errorf("%s workspace produced a claim about having no gates:\n%s", name, out)
		}
	}
}

// A cut list says it was cut. Nothing in this codebase truncates in silence.
func TestATruncatedGateListSaysSo(t *testing.T) {
	out, err := Build(Prompt{
		Doctrine:  DefaultDoctrine([]string{"read"}),
		Workspace: &Workspace{Gates: []Gate{{Name: "a", Command: "x"}}, Truncated: true},
	}, FormulationFor(""))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "not shown") {
		t.Errorf("a truncated gate list did not say so:\n%s", out)
	}
}

// The repository and the gates share one block, and both survive it.
func TestTheWorkspaceBlockCarriesBothFacts(t *testing.T) {
	out, err := Build(Prompt{
		Doctrine:  DefaultDoctrine([]string{"read"}),
		Repo:      &Repo{Branch: "fix/thing", MainBranch: "main", Clean: true},
		Workspace: &Workspace{Gates: []Gate{{Name: "check", Command: "make check"}}},
	}, FormulationFor(""))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"fix/thing", "make check"} {
		if !strings.Contains(out, want) {
			t.Errorf("the workspace block lost %q:\n%s", want, out)
		}
	}
}

// Build stays pure with the gates: the same probe must produce a
// byte-identical prefix.
func TestTheGateSectionIsPure(t *testing.T) {
	ws := &Workspace{Gates: []Gate{
		{Name: "a", Command: "1"}, {Name: "b", Command: "2"}, {Name: "c", Command: "3"},
	}}
	build := func() string {
		out, err := Build(Prompt{Doctrine: DefaultDoctrine([]string{"read"}), Workspace: ws}, FormulationFor(""))
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	first := build()
	for i := 0; i < 5; i++ {
		if build() != first {
			t.Fatal("the same workspace produced a different prefix")
		}
	}
}

// Two sources declaring the same name are told apart on screen.
//
// Found by running the release: a project with a `test` script in package.json
// and a `test` target in the Makefile printed two rows called `test` with
// different commands, and nothing said which was which. Source was being
// carried and never shown.
//
// It is the same defect the task parser refuses in a tasks.md — two rows a
// reader cannot tell apart — arriving at the other end of the same session.
func TestGatesThatShareANameAreToldApart(t *testing.T) {
	out, err := Build(Prompt{
		Doctrine: DefaultDoctrine([]string{"read"}),
		Workspace: &Workspace{Gates: []Gate{
			{Name: "test", Command: "vitest run", Source: "package.json"},
			{Name: "lint", Command: "eslint .", Source: "package.json"},
			{Name: "test", Command: "make test", Source: "Makefile"},
		}},
	}, FormulationFor(""))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"test (package.json)", "test (Makefile)"} {
		if !strings.Contains(out, want) {
			t.Errorf("the prefix does not carry %q:\n%s", want, out)
		}
	}
	// The one that is unambiguous stays unqualified: spending a column on a
	// distinction that does not exist is noise on every other project.
	if strings.Contains(out, "lint (package.json)") {
		t.Errorf("an unambiguous gate was qualified anyway:\n%s", out)
	}
}
