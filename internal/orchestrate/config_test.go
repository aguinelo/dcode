package orchestrate

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/protocol"
)

// Spec: orchestration .config §1 — orchestrator.enabled is false unless
// declared, and a value that is not a boolean is an error naming it.
func TestEnabledIsOffUnlessDeclaredAndRefusesANonBoolean(t *testing.T) {
	for v, want := range map[string]bool{"": false, "true": true, "ON": true, "0": false, "no": false} {
		got, err := Enabled(resolved(map[string]string{"orchestrator.enabled": v}))
		if err != nil || got != want {
			t.Errorf("Enabled(%q) = %v, %v; want %v", v, got, err, want)
		}
	}
	if _, err := Enabled(resolved(map[string]string{"orchestrator.enabled": "sometimes"})); err == nil ||
		!strings.Contains(err.Error(), "orchestrator.enabled") || !strings.Contains(err.Error(), "sometimes") {
		t.Errorf("Enabled(sometimes) = %v, want an error naming the key and the value", err)
	}
}

func TestProfileAndMainBranchAreReadTrimmed(t *testing.T) {
	r := resolved(map[string]string{"orchestrator.profile": " strong ", "orchestrator.main_branch": " trunk "})
	if Profile(r) != "strong" || MainBranch(r) != "trunk" {
		t.Errorf("Profile = %q, MainBranch = %q", Profile(r), MainBranch(r))
	}
}

// Spec: orchestration .config §2 — orchestrator.main_branch names a branch
// that exists; one that does not is refused, saying so, and nothing spawns.
func TestADeclaredMainBranchMustExist(t *testing.T) {
	root := gitRepo(t)
	git(t, root, "branch", "trunk")
	for name, want := range map[string]string{"trunk": "trunk", "nowhere": ""} {
		sp := &fakeSpawner{}
		r, err := NewRegistry(RegistryOptions{Dir: t.TempDir(), Spawner: sp,
			Resolve: func(string) (config.Resolved, error) {
				return resolved(map[string]string{"orchestrator.main_branch": name}), nil
			}})
		if err != nil {
			t.Fatal(err)
		}
		o, _, err := r.Open(root)
		switch {
		case want != "" && (err != nil || o.Project().Main != want):
			t.Errorf("main_branch %s: Open = %v", name, err)
		case want == "" && (err == nil || !strings.Contains(err.Error(), "orchestrator.main_branch") || len(sp.spawned) != 0):
			t.Errorf("main_branch %s: Open = %v, spawned %d", name, err, len(sp.spawned))
		}
		r.Close()
	}
}

// A repository with no main git can name is refused rather than given one.
func TestARepositoryWithNoMainIsRefused(t *testing.T) {
	root := gitRepo(t)
	git(t, root, "branch", "-m", "main", "trunk")
	r := newRegistry(t, t.TempDir(), &fakeSpawner{})
	if _, _, err := r.Open(root); err == nil || !strings.Contains(err.Error(), "no main branch") {
		t.Errorf("Open = %v, want the missing main said", err)
	}
}

func TestAConfigurationThatDoesNotReadIsRefused(t *testing.T) {
	root := gitRepo(t)
	r, err := NewRegistry(RegistryOptions{Dir: t.TempDir(), Spawner: &fakeSpawner{},
		Resolve: func(string) (config.Resolved, error) { return config.Resolved{}, errors.New("bad toml") }})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if _, _, err := r.Open(root); err == nil || !strings.Contains(err.Error(), "bad toml") {
		t.Errorf("Open = %v", err)
	}
}

func TestARegistryNeedsAnAbsoluteDirectoryAndASpawner(t *testing.T) {
	if _, err := NewRegistry(RegistryOptions{Dir: "rel", Spawner: &fakeSpawner{}}); err == nil {
		t.Error("a relative orchestrator.dir was accepted")
	}
	if _, err := NewRegistry(RegistryOptions{Dir: t.TempDir()}); err == nil {
		t.Error("a registry without a spawner was built")
	}
}

// A state.json that cannot even be read keeps the project closed on both
// paths that would open it, and neither spawns a fresh orchestrator over it.
func TestAStateThatCannotBeReadKeepsBothPathsClosed(t *testing.T) {
	root := gitRepo(t)
	dir := t.TempDir()
	_, id, err := ResolveProject(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, id, stateFile), 0o700); err != nil {
		t.Fatal(err)
	}
	sp := &fakeSpawner{}
	r := newRegistry(t, dir, sp)
	if errs := r.Reload(t.Context()); len(errs) != 1 {
		t.Errorf("Reload = %v, want one error", errs)
	}
	if _, _, err := r.Open(root); err == nil || !strings.Contains(err.Error(), "did not open") {
		t.Errorf("Open = %v", err)
	}
	if len(sp.spawned) != 0 {
		t.Error("an orchestrator was spawned over a state that does not read")
	}
}

type failingSpawner struct{ fakeSpawner }

func (*failingSpawner) SpawnOrchestrator(protocol.Project) (protocol.Session, error) {
	return protocol.Session{}, protocol.Errorf(protocol.CodeOrchestratorUnavailable, "no record")
}

// An orchestrator that cannot be spawned leaves no project behind: no state,
// and the next open tries again.
func TestASpawnThatFailsLeavesNoProject(t *testing.T) {
	root := gitRepo(t)
	dir := t.TempDir()
	r := newRegistry(t, dir, &failingSpawner{})
	if _, _, err := r.Open(root); err == nil {
		t.Fatal("Open succeeded without an orchestrator")
	}
	if len(r.List()) != 0 {
		t.Error("a project without an orchestrator is listed")
	}
	_, id, _ := ResolveProject(t.Context(), root)
	if _, err := storeAt(dir, id).Load(); err == nil {
		t.Error("a state.json was written for a project that did not open")
	}
	if matches, _ := filepath.Glob(filepath.Join(dir, id, stateFile)); len(matches) != 0 {
		t.Errorf("found %v", matches)
	}
}
