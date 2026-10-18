package orchestrate

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/protocol"
)

func (f *fakeSpawner) SpawnChild(req ChildRequest) (protocol.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.children = append(f.children, req)
	if f.failChild != nil {
		return protocol.Session{}, f.failChild
	}
	return protocol.Session{ID: req.ID, Workspace: req.Worktree, Model: "m-" + req.Profile,
		ParentID: req.ParentID, Role: protocol.RoleChild, Owns: req.Owns}, nil
}

func (f *fakeSpawner) Submit(id, text, origin string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.busy[id] {
		return protocol.Errorf(protocol.CodeTurnAlreadyActive, "a turn is already running")
	}
	f.submitted = append(f.submitted, id+": "+text)
	f.origins = append(f.origins, origin)
	return f.failSubmit
}

func (f *fakeSpawner) SpawnReviewer(req ReviewRequest) (protocol.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reviewers = append(f.reviewers, req)
	return protocol.Session{ID: req.ID, Workspace: req.Worktree, ParentID: req.ParentID, Role: protocol.RoleReviewer}, nil
}

func (f *fakeSpawner) Steer(id, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.steered = append(f.steered, id+": "+text)
	return nil
}

// wire is what the registry sent on the orchestrator's log.
type wire struct {
	mu     sync.Mutex
	events []protocol.EventType
	last   map[protocol.EventType]any
}

func (w *wire) emit(_ string, t protocol.EventType, payload any) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.events = append(w.events, t)
	if w.last == nil {
		w.last = map[protocol.EventType]any{}
	}
	w.last[t] = payload
}

func (w *wire) count(t protocol.EventType) int {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := 0
	for _, e := range w.events {
		if e == t {
			n++
		}
	}
	return n
}

// isolateGit keeps the git the harness runs from this machine's own
// configuration — its hooks, its signing — and gives it an identity.
func isolateGit(t *testing.T) {
	t.Helper()
	for k, v := range map[string]string{"GIT_CONFIG_GLOBAL": "/dev/null", "GIT_CONFIG_NOSYSTEM": "1",
		"GIT_AUTHOR_NAME": "t", "GIT_AUTHOR_EMAIL": "t@t", "GIT_COMMITTER_NAME": "t", "GIT_COMMITTER_EMAIL": "t@t"} {
		t.Setenv(k, v)
	}
}

// clock is a time a test moves.
type clock struct {
	mu sync.Mutex
	at time.Time
}

func (c *clock) now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.at }
func (c *clock) add(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.at = c.at.Add(d)
}

// project is one opened project with its fakes.
type project struct {
	root, dir string
	sp        *fakeSpawner
	w         *wire
	clk       *clock
	reg       *Registry
	o         *Orchestrator
	// merge is the merge step the registry hands a green child to.
	merge MergeQueue
}

func openWith(t *testing.T, keys map[string]string) *project {
	return openMerging(t, keys, nil)
}

func openMerging(t *testing.T, keys map[string]string, merge MergeQueue) *project {
	t.Helper()
	isolateGit(t)
	p := &project{root: gitRepo(t), dir: t.TempDir(), sp: &fakeSpawner{}, w: &wire{},
		clk: &clock{at: time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)}, merge: merge}
	p.reg = p.registry(t, keys)
	o, _, err := p.reg.Open(p.root)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	p.o = o
	return p
}

func (p *project) registry(t *testing.T, keys map[string]string) *Registry {
	t.Helper()
	r, err := NewRegistry(RegistryOptions{Dir: p.dir, Spawner: p.sp, Log: func(string) {}, Now: p.clk.now,
		Emit: p.w.emit, Retention: time.Hour, Merge: p.merge,
		Resolve: func(string) (config.Resolved, error) { return resolved(keys), nil }})
	if err != nil {
		t.Fatal(err)
	}
	r.async = func(f func()) { f() } // the pipeline's long steps run inline, in order
	t.Cleanup(r.Close)
	return r
}

func item(title string, owns ...string) Item {
	return Item{Title: title, Task: "do " + title, Owns: owns, Why: "it is its own theme"}
}

func wantCode(t *testing.T, err error, code string) *protocol.Error {
	t.Helper()
	var e *protocol.Error
	if !errors.As(err, &e) || e.Code != code {
		t.Fatalf("err = %v, want %s", err, code)
	}
	return e
}

// noWorktree fails when the batch made anything: a worktree, a session.
func (p *project) noWorktree(t *testing.T) {
	t.Helper()
	entries, _ := os.ReadDir(filepath.Join(p.dir, p.o.Project().ID, childrenDir))
	if len(entries) != 0 || len(p.sp.children) != 0 {
		t.Errorf("a refused batch made %d worktree(s) and %d session(s)", len(entries), len(p.sp.children))
	}
}

// Spec: orchestration .i step 6 — OwnsOverlap in a table: a prefix that holds
// a prefix, the separator, a metacharacter, the conservative case.
func TestOwnsOverlapIsConservative(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"internal/a", "internal/a/b.go", true},      // prefix holds prefix
		{"internal/a", "internal/ab", false},         // by component, not by string
		{"internal/a/x.go", "internal/a/x.go", true}, // the same path
		{"docs/*.md", "docs/guide.md", true},         // a metacharacter cuts at its folder
		{"docs/*.md", "docs/*.go", true},             // the conservative case
		{"docs/*.md", "internal/x.go", false},        // other folders
		{"*.go", "internal/x.go", true},              // a glob at the root is the whole tree
		{"a/b/", "a/b/c", true},
	} {
		if got := OwnsOverlap(c.a, c.b); got != c.want {
			t.Errorf("OwnsOverlap(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
		if got := OwnsOverlap(c.b, c.a); got != c.want {
			t.Errorf("OwnsOverlap(%q, %q) = %v, want %v (symmetric)", c.b, c.a, got, c.want)
		}
	}
}

// Spec: orchestration .i step 6 — a batch with two items whose owns cross:
// owns_overlap with both paths, and no worktree.
func TestABatchWhoseOwnsCrossIsRefusedWholeAndNoWorktreeIsMade(t *testing.T) {
	p := openWith(t, nil)
	_, err := p.o.Delegate([]Item{item("index", "internal/index"), item("search", "internal/index/search.go")})
	e := wantCode(t, err, protocol.CodeOwnsOverlap)
	if !strings.Contains(e.Message, "internal/index") || !strings.Contains(e.Message, "internal/index/search.go") {
		t.Errorf("the refusal does not name both paths: %s", e.Message)
	}
	p.noWorktree(t)
	if kinds := kindsIn(t, p.dir, p.o.Project().ID); kinds[len(kinds)-1] != "dispatch.refused" {
		t.Errorf("log = %v, want dispatch.refused last", kinds)
	}
	if p.w.count(protocol.EventSessionError) != 1 {
		t.Errorf("the refusal did not reach the person as a session error")
	}
}

// Spec: orchestration .i step 6 — a batch against a live child with crossing
// owns is refused whole: its disjoint item is not made either.
func TestABatchCrossingALiveChildIsRefusedWhole(t *testing.T) {
	p := openWith(t, nil)
	if _, err := p.o.Delegate([]Item{item("index", "internal/index")}); err != nil {
		t.Fatal(err)
	}
	_, err := p.o.Delegate([]Item{item("docs", "docs"), item("index tests", "internal/index/index_test.go")})
	e := wantCode(t, err, protocol.CodeOwnsOverlap)
	if !strings.Contains(e.Message, p.o.Children()[0].ID) {
		t.Errorf("the refusal does not name the live child: %s", e.Message)
	}
	if n := len(p.o.Children()); n != 1 || len(p.sp.children) != 1 {
		t.Errorf("%d children after a refused batch, want the one live", n)
	}
}

// Spec: orchestration .i step 6 — a batch above max_children:
// max_children_reached, nothing made.
func TestABatchAboveMaxChildrenIsRefusedWhole(t *testing.T) {
	p := openWith(t, map[string]string{"orchestrator.max_children": "1"})
	_, err := p.o.Delegate([]Item{item("a", "a"), item("b", "b")})
	wantCode(t, err, protocol.CodeMaxChildrenReached)
	p.noWorktree(t)
}

// Spec: orchestration .i step 6 — a profile outside child_profiles is
// model_not_allowed; an empty model is the first of the list, said in the log.
func TestAProfileOutsideTheListIsModelNotAllowed(t *testing.T) {
	p := openWith(t, map[string]string{"orchestrator.child_profiles": "fast, strong"})
	bad := item("a", "a")
	bad.Model = "other"
	_, err := p.o.Delegate([]Item{item("b", "b"), bad})
	e := wantCode(t, err, protocol.CodeModelNotAllowed)
	if !strings.Contains(e.Message, "fast, strong") {
		t.Errorf("the refusal does not list the allowed profiles: %s", e.Message)
	}
	p.noWorktree(t)

	kids, err := p.o.Delegate([]Item{item("b", "b")})
	if err != nil {
		t.Fatal(err)
	}
	if kids[0].Profile != "fast" || kids[0].Model != "m-fast" || p.sp.children[0].Profile != "fast" {
		t.Errorf("an empty model ran on %q (%q), want fast", kids[0].Profile, kids[0].Model)
	}
	data, _ := os.ReadFile(filepath.Join(p.dir, p.o.Project().ID, decisionsFile))
	if !strings.Contains(string(data), `"kind":"model.chosen"`) || !strings.Contains(string(data), "first of the child role's list") {
		t.Errorf("the model choice and its why are not in the log:\n%s", data)
	}
}

// Spec: orchestration .i step 6 — creating children comes back with their ids
// while they run: each in its own worktree from main's tip, on its branch,
// with its first turn submitted and nothing waited for.
func TestCreatingChildrenReturnsTheirIdsWhileTheyRun(t *testing.T) {
	p := openWith(t, nil)
	kids, err := p.o.Delegate([]Item{item("The Index!", "internal/index"), item("docs", "docs")})
	if err != nil {
		t.Fatal(err)
	}
	tip := strings.TrimSpace(gitOut(t, p.root, "rev-parse", "main"))
	for i, c := range kids {
		if c.ID == "" || c.State != stateWorking || c.Base != tip || c.ParentID != p.o.Project().OrchestratorID {
			t.Errorf("child %d = %+v", i, c)
		}
		if c.Worktree != filepath.Join(p.dir, p.o.Project().ID, "children", c.ID) {
			t.Errorf("worktree %s is not under orchestrator.dir", c.Worktree)
		}
		if got := strings.TrimSpace(gitOut(t, c.Worktree, "rev-parse", "--abbrev-ref", "HEAD")); got != c.Branch {
			t.Errorf("worktree is on %s, want %s", got, c.Branch)
		}
	}
	if want := "dcode/the-index-" + kids[0].ID[len(kids[0].ID)-6:]; kids[0].Branch != want {
		t.Errorf("branch %s, want %s", kids[0].Branch, want)
	}
	if len(p.sp.submitted) != 2 || !strings.Contains(p.sp.submitted[0], "internal/index") ||
		!strings.Contains(p.sp.submitted[0], "Do not run git commands that write") {
		t.Errorf("first turns = %q", p.sp.submitted)
	}
	if p.w.count(protocol.EventChildStarted) != 2 || p.w.count(protocol.EventChildState) != 2 {
		t.Errorf("wire = %v", p.w.events)
	}
	kinds := strings.Join(kindsIn(t, p.dir, p.o.Project().ID), " ")
	if !strings.Contains(kinds, "dispatch model.chosen worktree.created child.state") {
		t.Errorf("log = %s", kinds)
	}
}

// Spec: orchestration .i step 6 — Spent at the ceiling: delegate refused with
// ceiling_reached, no worktree, ceiling.reached in the log and a ceiling
// question; continue zeroes the window and the next batch goes through.
func TestAtTheCeilingDispatchStopsAndContinueOpensANewWindow(t *testing.T) {
	p := openWith(t, map[string]string{"orchestrator.token_ceiling": "100000"})
	usage, _ := json.Marshal(protocol.TurnCompleted{TurnID: "t1", Reason: protocol.StopDone,
		Usage: &protocol.Usage{InputTokens: 60000, OutputTokens: 30000, CacheReadTokens: 10000, CacheWriteTokens: 99999}})
	p.reg.Observe(p.o.Project().OrchestratorID, protocol.Event{Type: protocol.EventTurnCompleted, Payload: usage})
	if got := p.o.snapshot().Spent; got != 100000 {
		t.Fatalf("Spent = %d, want input+output+cache read", got)
	}

	for range 2 {
		_, err := p.o.Delegate([]Item{item("a", "a")})
		wantCode(t, err, protocol.CodeCeilingReached)
	}
	p.noWorktree(t)
	qs := p.o.Questions()
	if len(qs) != 1 || qs[0].Kind != "ceiling" || len(qs[0].Options) != 2 {
		t.Fatalf("questions = %+v, want one ceiling question for two refusals", qs)
	}
	if kinds := strings.Join(kindsIn(t, p.dir, p.o.Project().ID), " "); !strings.Contains(kinds, "ceiling.reached question.asked dispatch.refused") {
		t.Errorf("log = %s", kinds)
	}

	if _, err := p.o.Answer(qs[0].ID, "continue", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := p.o.Answer(qs[0].ID, "stop", ""); err == nil {
		t.Error("a second answer was taken")
	}
	if p.o.snapshot().Spent != 0 {
		t.Error("continue did not zero the window")
	}
	if _, err := p.o.Delegate([]Item{item("a", "a")}); err != nil {
		t.Errorf("after continue the batch is %v", err)
	}
}

func TestInvalidLimitsAreRefusedNamingTheKey(t *testing.T) {
	for key, v := range map[string]string{"orchestrator.max_children": "17", "orchestrator.token_ceiling": "5",
		"orchestrator.branch_prefix": "a..b"} {
		_, err := ReadLimits(t.Context(), resolved(map[string]string{key: v}), 0)
		if err == nil || !strings.Contains(err.Error(), key) || !strings.Contains(err.Error(), v) {
			t.Errorf("%s = %q: %v", key, v, err)
		}
	}
	if _, err := ReadLimits(t.Context(), resolved(nil), 6); err == nil {
		t.Error("four children under six sessions leave no room for the orchestrator and a reviewer")
	}
	if _, err := ClosedRetention(resolved(map[string]string{"orchestrator.closed_retention": "25h"})); err == nil {
		t.Error("a retention past 24h was taken")
	}
}
