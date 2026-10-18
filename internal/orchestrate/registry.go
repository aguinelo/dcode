package orchestrate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/vcs"
)

// One orchestrator per project, alive in the daemon and on disk.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §4, §8.

// gitDeadline bounds the git a project open asks: the root, the main branch.
const gitDeadline = 10 * time.Second

// autonomyStep is the only level of this slice (.config §4).
const autonomyStep = "step"

// Spawner is how the orchestrator reaches the daemon, which alone knows how a
// session is built (.p §3).
type Spawner interface {
	// SpawnOrchestrator opens the orchestrator of p: a recorded session in
	// plan mode at p.Root, with p's id and the orchestrator role.
	SpawnOrchestrator(p protocol.Project) (protocol.Session, error)
	// SpawnChild opens a child: a recorded session under req.ID in its
	// worktree, with the child role, its parent and its owns, whose writes
	// stay inside them, whose git that writes is refused, and whose
	// approvals wait for the person without a deadline.
	SpawnChild(req ChildRequest) (protocol.Session, error)
	// SpawnReviewer opens a reviewer under req.ID: a recorded session with
	// the reviewer role, in plan mode in the child's worktree, whose tools
	// are ReviewerRegistry's with req.Submit as its verdict.
	SpawnReviewer(req ReviewRequest) (protocol.Session, error)
	// Submit starts a turn in an idle session; origin is who started it,
	// empty for the person.
	Submit(sessionID, text, origin string) error
	// Steer hands the turn a session is in more of what to do.
	Steer(sessionID, text string) error
	// Reopen brings a recorded session back under its own id; a turn its
	// record shows open ends with reason.
	Reopen(sessionID, reason string) (protocol.Session, error)
	// Close closes a live session.
	Close(sessionID string) error
}

// RegistryOptions configure a Registry.
type RegistryOptions struct {
	// Dir is orchestrator.dir, absolute.
	Dir     string
	Spawner Spawner
	// Resolve is the configuration chain at a project's root, where
	// orchestrator.main_branch is read. Nil reads every key as unset.
	Resolve func(root string) (config.Resolved, error)
	// Log receives what a reload cut or could not reopen.
	Log func(string)
	Now func() time.Time
	// Emit sends an event on a session's log: how child.* and question.*
	// reach the orchestrator's wire. Nil sends nothing.
	Emit func(sessionID string, t protocol.EventType, payload any)
	// Retention is orchestrator.closed_retention: how long a closed child
	// stays in /children. Zero takes it out at once.
	Retention time.Duration
	// MaxSessions is the daemon's session ceiling, which max_children must
	// leave room under; zero is unknown.
	MaxSessions int
	// ConclusionMax is where a child's conclusion is cut in child.completed;
	// zero is DelegateMaxResultBytes' default.
	ConclusionMax int
	// KnownProfile says whether models.toml at root has a profile name:
	// orchestrator.reviewer_profiles is checked against it. Nil knows none.
	KnownProfile func(root, name string) bool
	// Merge is the merge step a child measured green is handed to; nil
	// leaves it in_review, reported.
	Merge MergeQueue
}

// Registry is every project this daemon conducts.
type Registry struct {
	dir     string
	spawner Spawner
	resolve func(string) (config.Resolved, error)
	log     func(string)
	now     func() time.Time
	lock    *os.File
	emit    func(string, protocol.EventType, any)

	retention     time.Duration
	maxSessions   int
	conclusionMax int
	known         func(root, name string) bool
	merge         MergeQueue
	// async runs a pipeline step that may take long — a gate, a red test —
	// off the goroutine that reads a session's events. A test runs it inline.
	async func(func())

	mu       sync.Mutex
	projects map[string]*Orchestrator
}

// Orchestrator conducts one project. No method waits for a child.
type Orchestrator struct {
	reg    *Registry
	limits Limits
	// ownProfile is orchestrator.profile, the child role's list when
	// child_profiles names none.
	ownProfile string
	// dispatch serialises what changes the set of children — a batch, a
	// close — so a batch is validated against the children it will join.
	dispatch sync.Mutex

	mu    sync.Mutex
	state State
	store *Store
	log   *DecisionLog
	// conclusions are what each child said in its turn since its last tool
	// call: the conclusion child.completed carries.
	conclusions map[string]*strings.Builder
	// gate and review are the pipeline's configuration, read when the
	// project opens.
	gate   Gate
	review ReviewConfig
	// reviews are the reviewer sessions at work, by session id.
	reviews map[string]*reviewing
	// reportTimer is the inbox's debounce while it runs; closed stops it.
	reportTimer *time.Timer
	closed      bool
}

// NewRegistry opens orchestrator.dir, owner-only, for this daemon alone.
func NewRegistry(o RegistryOptions) (*Registry, error) {
	if !filepath.IsAbs(o.Dir) {
		return nil, fmt.Errorf("orchestrator.dir must be an absolute path, got %q", o.Dir)
	}
	if o.Spawner == nil {
		return nil, errors.New("orchestrate: a registry needs a spawner")
	}
	if err := os.MkdirAll(o.Dir, 0o700); err != nil {
		return nil, fmt.Errorf("orchestrate: create %s: %w", o.Dir, err)
	}
	lock, err := lockDir(o.Dir)
	if err != nil {
		return nil, err
	}
	r := &Registry{dir: filepath.Clean(o.Dir), spawner: o.Spawner, resolve: o.Resolve, log: o.Log,
		now: o.Now, lock: lock, projects: map[string]*Orchestrator{}, emit: o.Emit,
		retention: o.Retention, maxSessions: o.MaxSessions, conclusionMax: o.ConclusionMax,
		known: o.KnownProfile, merge: o.Merge, async: func(f func()) { go f() }}
	if r.conclusionMax <= 0 {
		r.conclusionMax = defaultConclusionMax
	}
	if r.log == nil {
		r.log = func(string) {}
	}
	if r.now == nil {
		r.now = time.Now
	}
	return r, nil
}

// Close closes every project's log and lets the directory go.
func (r *Registry) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, o := range r.projects {
		o.stop()
		_ = o.log.Close()
	}
	r.projects = map[string]*Orchestrator{}
	if r.lock != nil {
		_ = r.lock.Close()
		r.lock = nil
	}
}

// Open opens the project workspace belongs to, or finds the one it already is;
// created says which. A project on disk that this daemon does not hold — its
// reload failed, or never ran — is reopened, never replaced by a new one.
func (r *Registry) Open(workspace string) (*Orchestrator, bool, error) {
	if !filepath.IsAbs(workspace) {
		return nil, false, protocol.Errorf(protocol.CodeWorkspaceInvalid,
			"the workspace %q must be an absolute path", workspace)
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitDeadline)
	defer cancel()
	root, id, err := ResolveProject(ctx, workspace)
	if err != nil {
		return nil, false, protocol.Errorf(protocol.CodeWorkspaceInvalid, "%v", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if o, ok := r.projects[id]; ok {
		return o, false, nil
	}
	if _, err := os.Stat(filepath.Join(r.dir, id, stateFile)); err == nil {
		o, err := r.load(id, "the project was asked for and was not held: its reload had failed or had not run")
		if err != nil {
			return nil, false, protocol.Errorf(protocol.CodeInternal, "project %s (%s) is on disk and did not open: %v", id, root, err)
		}
		return o, false, nil
	}
	o, err := r.create(ctx, root, id, workspace)
	if err != nil {
		return nil, false, err
	}
	return o, true, nil
}

// create opens a project that has never been: its orchestrator, its first
// decision, its state — in that order, so the state never names an
// orchestrator the log does not.
func (r *Registry) create(ctx context.Context, root, id, workspace string) (*Orchestrator, error) {
	cfg, err := r.config(root)
	if err != nil {
		return nil, err
	}
	main, err := r.mainBranch(ctx, root, cfg)
	if err != nil {
		return nil, err
	}
	limits, err := ReadLimits(ctx, cfg, r.maxSessions)
	if err != nil {
		return nil, protocol.Errorf(protocol.CodeWorkspaceInvalid, "%v", err)
	}
	p := protocol.Project{ID: id, Root: root, Main: main, Autonomy: autonomyStep, Created: r.now().UTC()}
	log, err := OpenDecisionLog(r.dir, p)
	if err != nil {
		return nil, protocol.Errorf(protocol.CodeInternal, "%v", err)
	}
	s, err := r.spawner.SpawnOrchestrator(p)
	if err != nil {
		_ = log.Close()
		return nil, err
	}
	p.OrchestratorID = s.ID
	gate, review, err := r.pipelineConfig(root, id, cfg)
	if err != nil {
		_ = log.Close()
		return nil, protocol.Errorf(protocol.CodeWorkspaceInvalid, "%v", err)
	}
	o := &Orchestrator{reg: r, limits: limits, ownProfile: Profile(cfg), gate: gate, review: review,
		state: State{V: stateVersion, Project: p}, store: storeAt(r.dir, id), log: log}
	data, _ := json.Marshal(map[string]string{"workspace": workspace, "orchestrator": s.ID, "main": main})
	if err := o.record(Decision{Kind: "project.opened", Session: s.ID, Data: data,
		Why: "the person opened the project of " + workspace + "; a project has one orchestrator"}); err != nil {
		_ = log.Close()
		if cerr := r.spawner.Close(s.ID); cerr != nil {
			r.log(fmt.Sprintf("orchestrator %s of a project that did not open could not be closed: %v", s.ID, cerr))
		}
		return nil, protocol.Errorf(protocol.CodeInternal, "project %s did not open: %v", id, err)
	}
	r.projects[id] = o
	return o, nil
}

// mainBranch is orchestrator.main_branch when declared and present, else the
// branch the repository says is its main. Neither is an error saying so: a
// wrong main is worse than none, since everything is measured against it.
func (r *Registry) mainBranch(ctx context.Context, root string, cfg config.Resolved) (string, error) {
	if name := MainBranch(cfg); name != "" {
		cmd := exec.CommandContext(ctx, "git", "rev-parse", "--verify", "--quiet", "refs/heads/"+name)
		cmd.Dir = root
		if err := cmd.Run(); err != nil {
			return "", protocol.Errorf(protocol.CodeWorkspaceInvalid,
				"orchestrator.main_branch is %q, which is no branch of %s", name, root)
		}
		return name, nil
	}
	if repo := vcs.Read(ctx, root); repo != nil && repo.MainBranch != "" {
		return repo.MainBranch, nil
	}
	return "", protocol.Errorf(protocol.CodeWorkspaceInvalid,
		"%s has no main branch git can name (no origin/HEAD, no main, no master); set orchestrator.main_branch", root)
}

// config is the configuration chain at a project's root.
func (r *Registry) config(root string) (config.Resolved, error) {
	if r.resolve == nil {
		return config.Resolved{}, nil
	}
	cfg, err := r.resolve(root)
	if err != nil {
		return config.Resolved{}, protocol.Errorf(protocol.CodeWorkspaceInvalid, "the configuration of %s cannot be read: %v", root, err)
	}
	return cfg, nil
}

// Reload opens every project on disk, at the daemon's start: each reopens its
// orchestrator under the same id and says so in daemon.restart. A project that
// does not open is returned as an error and stays closed — never an empty
// project in its place (.p §8).
func (r *Registry) Reload(ctx context.Context) []error {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return []error{fmt.Errorf("orchestrate: read %s: %w", r.dir, err)}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var errs []error
	for _, e := range entries {
		if ctx.Err() != nil {
			return append(errs, ctx.Err())
		}
		id := e.Name()
		if !e.IsDir() || !projectID.MatchString(id) || r.projects[id] != nil {
			continue
		}
		if _, err := os.Stat(filepath.Join(r.dir, id, stateFile)); errors.Is(err, fs.ErrNotExist) {
			continue // a project whose first open did not finish: nothing was conducted
		}
		if _, err := r.load(id, "the daemon started again"); err != nil {
			errs = append(errs, fmt.Errorf("project %s did not reopen: %w", id, err))
			r.log(errs[len(errs)-1].Error())
		}
	}
	return errs
}

// load reopens one project from disk under r.mu: its state, its log, its
// orchestrator, and daemon.restart saying what reopened, what did not, and
// what a crash cut from the log.
func (r *Registry) load(id, why string) (*Orchestrator, error) {
	store := storeAt(r.dir, id)
	st, err := store.Load()
	if err != nil {
		return nil, err
	}
	log, err := OpenDecisionLog(r.dir, st.Project)
	if err != nil {
		return nil, err
	}
	cfg, err := r.config(st.Project.Root)
	if err != nil {
		_ = log.Close()
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitDeadline)
	defer cancel()
	limits, err := ReadLimits(ctx, cfg, r.maxSessions)
	if err != nil {
		_ = log.Close()
		return nil, err
	}
	gate, review, err := r.pipelineConfig(st.Project.Root, id, cfg)
	if err != nil {
		_ = log.Close()
		return nil, err
	}
	o := &Orchestrator{reg: r, limits: limits, ownProfile: Profile(cfg), gate: gate, review: review,
		state: st, store: store, log: log}
	orq := st.Project.OrchestratorID
	restart := map[string]any{}
	if torn := log.Torn(); len(torn) > 0 {
		restart["torn"] = string(torn)
		r.log(fmt.Sprintf("decision log of project %s: a crash cut its last line, %d bytes, which are kept in daemon.restart", id, len(torn)))
	}
	_, rerr := r.spawner.Reopen(orq, protocol.StopDaemonRestart)
	if rerr != nil {
		restart["failed"] = map[string]string{orq: rerr.Error()}
		why += "; the orchestrator could not be reopened, and the project stays closed"
	} else {
		why += "; the orchestrator was reopened under its own id"
	}
	var after []func()
	if rerr == nil {
		reopened, failed, moves := o.reopenChildren()
		restart["reopened"] = append([]string{orq}, reopened...)
		if len(failed) > 0 {
			restart["failed"] = failed
		}
		after = moves
	}
	data, _ := json.Marshal(restart)
	if err := o.record(Decision{Kind: "daemon.restart", Session: orq, Why: why, Data: data}); err != nil {
		_ = log.Close()
		if rerr == nil {
			_ = r.spawner.Close(orq)
		}
		return nil, errors.Join(rerr, err)
	}
	if rerr != nil {
		_ = log.Close()
		return nil, fmt.Errorf("orchestrator %s: %w", orq, rerr)
	}
	r.projects[id] = o
	for _, move := range after {
		move()
	}
	o.resume()
	o.scheduleReports() // a report pending at the restart is delivered now
	return o, nil
}

// List is every project held, by id.
func (r *Registry) List() []protocol.Project {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]protocol.Project, 0, len(r.projects))
	for _, o := range r.projects {
		out = append(out, o.Project())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Get is the project id, when this daemon holds it.
func (r *Registry) Get(id string) (*Orchestrator, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	o, ok := r.projects[id]
	return o, ok
}

// Project is the project this orchestrator conducts.
func (o *Orchestrator) Project() protocol.Project {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.state.Project
}

// pipelineConfig is the gate and the review a project's chain declares; the
// whole output of each measurement goes to its gates/ directory.
func (r *Registry) pipelineConfig(root, id string, cfg config.Resolved) (Gate, ReviewConfig, error) {
	gate, err := GateFrom(cfg)
	if err != nil {
		return Gate{}, ReviewConfig{}, err
	}
	gate.Logs = filepath.Join(r.dir, id, "gates")
	var known func(string) bool
	if r.known != nil {
		known = func(name string) bool { return r.known(root, name) }
	}
	review, err := ReviewFrom(cfg, known)
	return gate, review, err
}
