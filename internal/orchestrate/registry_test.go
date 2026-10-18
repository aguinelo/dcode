package orchestrate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// gitRepo makes a repository with one commit on main and returns its root,
// links resolved.
func gitRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	git(t, root, "init", "-q", "-b", "main")
	git(t, root, "commit", "-q", "--allow-empty", "-m", "first")
	return root
}

// Spec: orchestration .config §4 — Project.ID is the first 24 hex of the
// sha256 of the resolved root, so it is stable across restarts.
func TestTheProjectIDIsTheSha256OfTheResolvedRoot(t *testing.T) {
	root := gitRepo(t)
	sum := sha256.Sum256([]byte(root))
	gotRoot, id, err := ResolveProject(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if gotRoot != root || id != hex.EncodeToString(sum[:])[:24] {
		t.Errorf("ResolveProject = %s, %s", gotRoot, id)
	}
}

// fakeSpawner stands in for the daemon: it hands out session ids and says
// what it was asked.
type fakeSpawner struct {
	mu       sync.Mutex
	spawned  []protocol.Project
	reopened []string
	reasons  []string
	closed   []string
	failOpen error
	// What step 6 asks of it: the children built, the turns submitted and
	// the steering handed over (children_test.go).
	children  []ChildRequest
	submitted []string
	steered   []string
	// failChild, failSubmit and failReopen make those calls fail; the last
	// for the ids it names alone.
	failChild, failSubmit error
	failReopen            map[string]error
	closeErr              error
}

func (f *fakeSpawner) SpawnOrchestrator(p protocol.Project) (protocol.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.spawned = append(f.spawned, p)
	return protocol.Session{ID: fmt.Sprintf("orq-%d", len(f.spawned)), Workspace: p.Root,
		ProjectID: p.ID, Role: protocol.RoleOrchestrator, Mode: protocol.ModePlan}, nil
}

func (f *fakeSpawner) Reopen(id, reason string) (protocol.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failOpen != nil {
		return protocol.Session{}, f.failOpen
	}
	if err := f.failReopen[id]; err != nil {
		return protocol.Session{}, err
	}
	f.reopened = append(f.reopened, id)
	f.reasons = append(f.reasons, reason)
	return protocol.Session{ID: id, Role: protocol.RoleOrchestrator}, nil
}

func (f *fakeSpawner) Close(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = append(f.closed, id)
	return f.closeErr
}

func newRegistry(t *testing.T, dir string, sp Spawner) *Registry {
	t.Helper()
	r, err := NewRegistry(RegistryOptions{Dir: dir, Spawner: sp, Log: func(string) {}})
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	t.Cleanup(r.Close)
	return r
}

func kindsIn(t *testing.T, dir, id string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, id, decisionsFile))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var d Decision
		if err := json.Unmarshal([]byte(line), &d); err != nil {
			t.Fatalf("line %q: %v", line, err)
		}
		out = append(out, d.Kind)
	}
	return out
}

// Spec: orchestration .i step 5 — a second open of the same tree, by another
// worktree, is the same project and the same orchestrator; nothing new is
// spawned.
func TestASecondOpenOfTheSameTreeIsTheSameProject(t *testing.T) {
	root := gitRepo(t)
	other := filepath.Join(t.TempDir(), "wt")
	git(t, root, "worktree", "add", "-q", "-b", "side", other)
	sp := &fakeSpawner{}
	r := newRegistry(t, t.TempDir(), sp)

	first, created, err := r.Open(root)
	if err != nil || !created {
		t.Fatalf("first Open = %v, created %v", err, created)
	}
	second, created, err := r.Open(other)
	if err != nil || created {
		t.Fatalf("second Open = %v, created %v; want the existing project", err, created)
	}
	if first.Project() != second.Project() {
		t.Errorf("two opens of one tree gave\n%+v\n%+v", first.Project(), second.Project())
	}
	if len(sp.spawned) != 1 {
		t.Errorf("%d orchestrators were spawned for one project", len(sp.spawned))
	}
	p := first.Project()
	if p.Root != root || p.Main != "main" || p.Autonomy != "step" || p.OrchestratorID != "orq-1" {
		t.Errorf("project = %+v", p)
	}
	if got := r.List(); len(got) != 1 || got[0].ID != p.ID {
		t.Errorf("List = %+v", got)
	}
}

// Spec: orchestration .i step 5 — a daemon that starts again brings the
// orchestrator back under the id the project holds, and says so in the log.
func TestReloadReopensTheOrchestratorUnderItsOwnID(t *testing.T) {
	root := gitRepo(t)
	dir := t.TempDir()
	first := newRegistry(t, dir, &fakeSpawner{})
	o, _, err := first.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	p := o.Project()
	first.Close()

	sp := &fakeSpawner{}
	again := newRegistry(t, dir, sp)
	if errs := again.Reload(context.Background()); len(errs) != 0 {
		t.Fatalf("Reload: %v", errs)
	}
	if len(sp.spawned) != 0 || len(sp.reopened) != 1 || sp.reopened[0] != p.OrchestratorID {
		t.Fatalf("spawned %v, reopened %v; want %s reopened", sp.spawned, sp.reopened, p.OrchestratorID)
	}
	if sp.reasons[0] != protocol.StopDaemonRestart {
		t.Errorf("reopened for %q, want %q", sp.reasons[0], protocol.StopDaemonRestart)
	}
	got, ok := again.Get(p.ID)
	if !ok || got.Project() != p {
		t.Fatalf("Get after Reload = %+v, %v; want %+v", got, ok, p)
	}
	if kinds := kindsIn(t, dir, p.ID); strings.Join(kinds, ",") != "project.opened,daemon.restart" {
		t.Errorf("log kinds = %v", kinds)
	}
	// And a later open finds it rather than making another.
	if _, created, err := again.Open(root); err != nil || created {
		t.Errorf("Open after Reload = %v, created %v", err, created)
	}
}

// Spec: orchestration .i step 5 — an unreadable state.json keeps the project
// closed and Reload returns the error, never an empty project in its place.
func TestAnUnreadableStateKeepsTheProjectClosed(t *testing.T) {
	root := gitRepo(t)
	dir := t.TempDir()
	first := newRegistry(t, dir, &fakeSpawner{})
	o, _, err := first.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	p := o.Project()
	first.Close()
	if err := os.WriteFile(filepath.Join(dir, p.ID, stateFile), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	sp := &fakeSpawner{}
	again := newRegistry(t, dir, sp)
	errs := again.Reload(context.Background())
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), p.ID) {
		t.Fatalf("Reload = %v, want one error naming %s", errs, p.ID)
	}
	if _, ok := again.Get(p.ID); ok {
		t.Error("a project whose state does not read was opened anyway")
	}
	if len(sp.reopened) != 0 || len(sp.spawned) != 0 {
		t.Errorf("reopened %v, spawned %v for a project that did not read", sp.reopened, sp.spawned)
	}
	if _, _, err := again.Open(root); err == nil {
		t.Error("Open replaced an unreadable project with a new one")
	}
	if len(sp.spawned) != 0 {
		t.Error("Open spawned a fresh orchestrator over an unreadable state")
	}
}

// Spec: orchestration .p §8 — daemon.restart says what could not reopen, and
// the project stays closed.
func TestAnOrchestratorThatDoesNotReopenIsSaidAndTheProjectStaysClosed(t *testing.T) {
	root := gitRepo(t)
	dir := t.TempDir()
	first := newRegistry(t, dir, &fakeSpawner{})
	o, _, err := first.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	p := o.Project()
	first.Close()

	again := newRegistry(t, dir, &fakeSpawner{failOpen: errors.New("its record is gone")})
	errs := again.Reload(context.Background())
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "its record is gone") {
		t.Fatalf("Reload = %v", errs)
	}
	if _, ok := again.Get(p.ID); ok {
		t.Error("a project whose orchestrator did not reopen is held")
	}
	data, err := os.ReadFile(filepath.Join(dir, p.ID, decisionsFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"daemon.restart"`) || !strings.Contains(string(data), "its record is gone") {
		t.Errorf("the log does not say what failed to reopen:\n%s", data)
	}
}

// Spec: orchestration .p §9.1 — a torn last line cut on reopen is said in
// daemon.restart, not lost in silence.
func TestATornLineIsSaidInTheRestart(t *testing.T) {
	root := gitRepo(t)
	dir := t.TempDir()
	first := newRegistry(t, dir, &fakeSpawner{})
	o, _, err := first.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	p := o.Project()
	first.Close()
	f, err := os.OpenFile(filepath.Join(dir, p.ID, decisionsFile), os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(`{"v":1,"seq":2,"kind":"disp`); err != nil {
		t.Fatal(err)
	}
	f.Close()

	again := newRegistry(t, dir, &fakeSpawner{})
	if errs := again.Reload(context.Background()); len(errs) != 0 {
		t.Fatalf("Reload: %v", errs)
	}
	data, err := os.ReadFile(filepath.Join(dir, p.ID, decisionsFile))
	if err != nil {
		t.Fatal(err)
	}
	last := strings.Split(strings.TrimSpace(string(data)), "\n")
	var d Decision
	if err := json.Unmarshal([]byte(last[len(last)-1]), &d); err != nil {
		t.Fatal(err)
	}
	if d.Kind != "daemon.restart" || d.Seq != 2 || !strings.Contains(string(d.Data), `disp`) {
		t.Errorf("last decision = %+v data %s; want daemon.restart at seq 2 quoting the torn line", d, d.Data)
	}
}

// Two daemons conducting one directory would reopen one orchestrator twice and
// write one log with two sequences: the second is refused, and says why.
func TestASecondDaemonOnTheSameDirectoryIsRefused(t *testing.T) {
	dir := t.TempDir()
	newRegistry(t, dir, &fakeSpawner{})
	if r, err := NewRegistry(RegistryOptions{Dir: dir, Spawner: &fakeSpawner{}}); err == nil {
		r.Close()
		t.Fatal("a second registry opened a directory the first holds")
	} else if !strings.Contains(err.Error(), "another daemon") {
		t.Errorf("refused with %v, want the reason", err)
	}
}

func TestAnOpenOutsideAnyRepositoryIsRefused(t *testing.T) {
	r := newRegistry(t, t.TempDir(), &fakeSpawner{})
	_, _, err := r.Open(t.TempDir())
	if pe, ok := protocol.AsError(err); !ok || pe.Code != protocol.CodeWorkspaceInvalid {
		t.Errorf("Open outside a repository = %v, want workspace_invalid", err)
	}
	if _, _, err := r.Open("rel/path"); err == nil {
		t.Error("a relative workspace opened a project")
	}
}
