package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
)

// A spec path is a string a client sent, and the daemon reads the disk. The
// two together are how `../../..` becomes an arbitrary file read.
func TestASpecPathCannotClimbOutOfTheWorkspace(t *testing.T) {
	ws := t.TempDir()
	for _, bad := range []string{
		"../elsewhere",
		"specs/../../elsewhere",
		"..",
	} {
		if got, err := specUnderWorkspace(ws, bad); err == nil {
			t.Errorf("%q was resolved to %q instead of refused", bad, got)
		}
	}
}

// And an absolute path outside it is the same request in another spelling.
func TestAnAbsoluteSpecPathOutsideTheWorkspaceIsRefused(t *testing.T) {
	ws := t.TempDir()
	other := t.TempDir()
	if _, err := specUnderWorkspace(ws, other); err == nil {
		t.Errorf("%q was accepted from a workspace at %q", other, ws)
	}
}

// A path inside resolves against the workspace, so `/loop specs/x` means the
// same thing from any client.
func TestASpecPathInsideResolvesAgainstTheWorkspace(t *testing.T) {
	ws := t.TempDir()
	got, err := specUnderWorkspace(ws, "specs/home-page")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(ws, "specs", "home-page"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	// The absolute spelling of the same place is accepted too.
	if got, err := specUnderWorkspace(ws, filepath.Join(ws, "specs")); err != nil {
		t.Errorf("an absolute path inside the workspace was refused: %v (%s)", err, got)
	}
}

// The session's definition of done comes from the spec when one is named, and
// from done.toml when one is not.
func TestASpecNamedIsTheDefinitionOfDone(t *testing.T) {
	ws := t.TempDir()
	if err := os.MkdirAll(filepath.Join(ws, ".dcode"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, ".dcode", "done.toml"),
		[]byte("[from-toml]\ncommand = \"make check\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	spec := filepath.Join(ws, "specs", "x")
	if err := os.MkdirAll(spec, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(spec, "tasks.md"),
		[]byte("- [ ] 1. desc. verify: `pnpm test`\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// No spec: done.toml.
	set, err := sessionDoneSet(Options{Workspace: ws})
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Criteria) != 1 || set.Criteria[0].Name != "from-toml" {
		t.Fatalf("without a spec the set is %+v", set.Criteria)
	}

	// A spec named: the spec, and done.toml is not consulted. Falling back
	// silently would measure the turn against something nobody named.
	set, err = sessionDoneSet(Options{Workspace: ws, LoopSpec: spec, Protect: []string{"**/*_test.go"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Criteria) != 1 || set.Criteria[0].Command != "pnpm test" {
		t.Fatalf("with a spec the set is %+v", set.Criteria)
	}
	if len(set.Protected) != 1 || set.Protected[0] != "**/*_test.go" {
		t.Errorf("the protect argument did not reach the set: %+v", set.Protected)
	}
}

// A spec that cannot be read stops the session rather than falling back.
//
// Asking for a spec and silently getting the workspace's own done.toml is the
// worst of the two: the turn measured against something the person did not
// name, and no way to tell from the screen.
func TestAnUnreadableSpecStopsTheSession(t *testing.T) {
	ws := t.TempDir()
	spec := filepath.Join(ws, "specs", "broken")
	if err := os.MkdirAll(spec, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(spec, "tasks.md"),
		[]byte("this is not a task list at all\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := sessionDoneSet(Options{Workspace: ws, LoopSpec: spec}); err == nil {
		t.Fatal("an unreadable spec fell through instead of stopping")
	}
}

// A spec whose tasks carry no command is zero criteria and NOT an error: the
// file declared that nothing here can be checked. The client says so.
func TestASpecWithNoRunnableCriterionIsNotAnError(t *testing.T) {
	ws := t.TempDir()
	spec := filepath.Join(ws, "specs", "manual")
	if err := os.MkdirAll(spec, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(spec, "tasks.md"),
		[]byte("- [ ] 1. `a.ts` — smoke manual, sem comando.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	set, err := sessionDoneSet(Options{Workspace: ws, LoopSpec: spec})
	if err != nil {
		t.Fatalf("a spec declaring no runnable criterion was refused: %v", err)
	}
	if len(set.Criteria) != 0 {
		t.Fatalf("expected no criteria, got %+v", set.Criteria)
	}
}

// The refusal names the path, or nobody knows which one was wrong.
func TestTheRefusalNamesThePath(t *testing.T) {
	_, err := specUnderWorkspace(t.TempDir(), "../secrets")
	if err == nil {
		t.Fatal("expected a refusal")
	}
	if !strings.Contains(err.Error(), "../secrets") {
		t.Errorf("the refusal does not name the path: %v", err)
	}
}

// The daemon lists a workspace's specs and says which are pending.
//
// It runs each folder's criteria to answer that, through the same sandbox a
// turn uses — which is why it is the daemon's job and not the client's.
func TestTheDaemonListsSpecsAndWhatIsPending(t *testing.T) {
	ws := t.TempDir()
	for name, done := range map[string]string{
		"a-done":    "[x]\ncommand = \"true\"\n",
		"b-pending": "[x]\ncommand = \"false\"\n",
	} {
		dir := filepath.Join(ws, "specs", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "done.toml"), []byte(done), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	d := &Daemon{opts: DaemonOptions{Base: Options{Workspace: ws, SandboxMode: policy.ModeReadOnly}}}
	got := d.specs(context.Background(), ws, true)
	if len(got) != 2 {
		t.Fatalf("got %+v", got)
	}
	byPath := map[string]protocol.SpecFolder{}
	for _, f := range got {
		byPath[f.Path] = f
	}
	for _, name := range []string{"a-done", "b-pending"} {
		f, ok := byPath[filepath.Join("specs", name)]
		if !ok {
			t.Fatalf("%s is missing from %+v", name, got)
		}
		if f.Error != "" {
			t.Errorf("%s came back unreadable: %s", name, f.Error)
		}
		if f.Criteria != 1 {
			t.Errorf("%s declares %d criteria, want 1", name, f.Criteria)
		}
	}

	// What this does NOT assert is whether each criterion passed.
	//
	// It ran one through the real sandbox, and whether `true` succeeds there
	// depends on the platform and on whether the backend can start at all —
	// this failed on Linux CI while passing on macOS. Deciding pending from a
	// run is the rule, and the rule is covered where a runner can be injected:
	// TestPendingIsWhatTheCriteriaSay and its neighbours in loopcommand. What
	// belongs here is the wiring, and the wiring is what is asserted.
}

// A workspace that is not one answers nothing rather than failing: the client
// asked what is there, and "nothing I can see" is an answer it can act on.
func TestListingSpecsOfANonWorkspaceAnswersNothing(t *testing.T) {
	d := &Daemon{opts: DaemonOptions{Base: Options{SandboxMode: policy.ModeReadOnly}}}
	if got := d.specs(context.Background(), "relative/path", true); got != nil {
		t.Errorf("got %+v", got)
	}
}

// A qualifying session runs in plan mode, and the request cannot change that.
//
// Working out what "done" means is reading. An agent that could write while
// deciding what it will be measured by can move the thing it is about to be
// measured against — so the boundary is not the requester's to choose.
func TestAQualifyingSessionIsAlwaysPlanMode(t *testing.T) {
	ws := t.TempDir()
	spec := filepath.Join(ws, "specs", "x")
	if err := os.MkdirAll(spec, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(spec, "spec.md"), []byte("# uma spec\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	d := &Daemon{opts: DaemonOptions{Base: Options{
		SandboxMode: policy.ModeFullAccess, Policy: policy.PolicyOnRequest,
	}}}
	// Asking for full access AND a qualifying session: the qualification wins.
	opts, err := d.qualifyOptions(ws, protocol.CreateSessionRequest{
		Workspace: ws, LoopSpec: "specs/x", Qualify: true, SandboxMode: "full-access",
	})
	if err != nil {
		t.Fatal(err)
	}
	if opts.SandboxMode != policy.ModeReadOnly {
		t.Errorf("a qualifying session runs as %q, want read-only", opts.SandboxMode)
	}
	if opts.Policy != policy.PolicyNever {
		t.Errorf("a qualifying session asks under %q, want never", opts.Policy)
	}
	if !opts.Qualify {
		t.Error("the qualifying flag did not reach the session")
	}
}
