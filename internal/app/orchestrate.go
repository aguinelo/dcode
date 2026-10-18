package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/orchestrate"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/session"
)

// The daemon's half of the orchestrator: it is the orchestrate.Spawner, since
// only it knows how a session is built, and it answers the project routes.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §3, §5.1, §8.

// OrchestratorOptions say whether a daemon conducts projects, and where their
// state lives: orchestrator.enabled and orchestrator.dir, read where the
// daemon starts. One daemon conducts a directory; the registry locks it.
type OrchestratorOptions struct {
	Enabled bool
	Dir     string
	// ClosedRetention is orchestrator.closed_retention: how long a closed
	// child stays in /children. Zero takes it out at once.
	ClosedRetention time.Duration
	// Elsewhere, when set, is why this daemon never conducts projects
	// whatever the configuration says: a daemon embedded in one terminal
	// client dies with it.
	Elsewhere string
}

// daemonProjects answers the project routes. off is why it cannot conduct, said
// on every route that would need it.
type daemonProjects struct {
	reg *orchestrate.Registry
	off string
}

// newProjects opens the registry when the orchestrator can run here, and says
// why when it cannot.
func (d *Daemon) newProjects() *daemonProjects {
	o := d.opts.Orchestrator
	switch {
	case o.Elsewhere != "":
		return &daemonProjects{off: o.Elsewhere}
	case !o.Enabled:
		return &daemonProjects{off: "orchestrator.enabled is false: the orchestrator is opt-in in this slice"}
	case d.opts.RecordDir == "":
		return &daemonProjects{off: "record.enabled is false: an orchestrator comes back from its record after a restart, and there would be none"}
	}
	reg, err := orchestrate.NewRegistry(orchestrate.RegistryOptions{
		Dir: o.Dir, Spawner: d, Resolve: d.resolveAt, Log: d.opts.Log,
		Emit: d.emitTo, Retention: o.ClosedRetention, MaxSessions: d.opts.MaxSessions,
		ConclusionMax: d.opts.Base.DelegateMaxResultBytes, KnownProfile: d.knownProfile,
	})
	if err != nil {
		off := fmt.Sprintf("the orchestrator cannot run in this daemon: %v", err)
		if d.opts.Log != nil {
			d.opts.Log(off)
		}
		return &daemonProjects{off: off}
	}
	return &daemonProjects{reg: reg}
}

func (p *daemonProjects) unavailable() error {
	return protocol.Errorf(protocol.CodeOrchestratorUnavailable, "%s", p.off)
}

func (p *daemonProjects) Open(workspace string) (protocol.Project, bool, error) {
	if p.reg == nil {
		return protocol.Project{}, false, p.unavailable()
	}
	o, created, err := p.reg.Open(workspace)
	if err != nil {
		return protocol.Project{}, false, err
	}
	return o.Project(), created, nil
}

// List is empty when the orchestrator is off: this daemon conducts nothing,
// and that is the answer, not a failure.
func (p *daemonProjects) List() []protocol.Project {
	if p.reg == nil {
		return nil
	}
	return p.reg.List()
}

func (p *daemonProjects) Get(id string) (protocol.Project, error) {
	if p.reg == nil {
		return protocol.Project{}, p.unavailable()
	}
	o, ok := p.reg.Get(id)
	if !ok {
		return protocol.Project{}, protocol.Errorf(protocol.CodeProjectNotFound, "this daemon conducts no project %s", id)
	}
	return o.Project(), nil
}

// reloadProjects reopens every project on disk, at the start. What did not
// reopen is said by the registry and stays closed.
func (d *Daemon) reloadProjects(ctx context.Context) []error {
	if d.projects.reg == nil {
		return nil
	}
	return d.projects.reg.Reload(ctx)
}

func (d *Daemon) closeProjects() {
	if d.projects.reg != nil {
		d.projects.reg.Close()
	}
}

// resolveAt is the configuration chain at a project's root. Options built by
// hand have no environment to resolve from, and read every key as unset.
func (d *Daemon) resolveAt(root string) (config.Resolved, error) {
	if d.opts.Base.Env == nil {
		return config.Resolved{}, nil
	}
	return Resolve(d.opts.Base.Env, root)
}

// planMode is the orchestrator's boundary: it reads, talks and conducts, and
// never writes to the tree (.p §10). Set last, so no request can widen it.
func planMode(o *Options) {
	o.SandboxMode, o.Policy = policy.ModeReadOnly, policy.PolicyNever
}

// SpawnOrchestrator opens p's orchestrator: a recorded session in plan mode at
// the root, on orchestrator.profile. A session that is not being recorded is
// closed and refused — it could not come back after a restart.
func (d *Daemon) SpawnOrchestrator(p protocol.Project) (protocol.Session, error) {
	cfg, err := d.resolveAt(p.Root)
	if err != nil {
		return protocol.Session{}, protocol.Errorf(protocol.CodeWorkspaceInvalid,
			"the configuration of %s cannot be read: %v", p.Root, err)
	}
	profile := orchestrate.Profile(cfg)
	if profile != "" {
		opts, err := d.optionsFor(p.Root)
		if err != nil {
			return protocol.Session{}, err
		}
		if _, ok := opts.Profiles[profile]; !ok {
			return protocol.Session{}, protocol.Errorf(protocol.CodeWorkspaceInvalid,
				"orchestrator.profile is %q, which is no profile of models.toml", profile)
		}
	}
	sess, err := d.buildAs("", protocol.CreateSessionRequest{Workspace: p.Root, Model: profile}, d.orchestratorMode(p.ID))
	if err != nil {
		return protocol.Session{}, err
	}
	sess.Role, sess.ProjectID = protocol.RoleOrchestrator, p.ID
	if err := d.manager.Add(sess); err != nil {
		sess.Close()
		return protocol.Session{}, err
	}
	sess.Emit(protocol.EventSessionCreated, sess.Describe())
	if _, err := os.Stat(filepath.Join(d.opts.RecordDir, sess.ID+".jsonl")); err != nil {
		_ = d.manager.Remove(sess.ID)
		return protocol.Session{}, protocol.Errorf(protocol.CodeOrchestratorUnavailable,
			"the orchestrator of %s is not being recorded, so it could not survive a restart: %v", p.Root, err)
	}
	d.watch(sess)
	return sess.Describe(), nil
}

// Reopen brings a recorded session back under its own id, its open turn
// ended with reason. Only daemon_restart is a reason in this slice.
func (d *Daemon) Reopen(sessionID, reason string) (protocol.Session, error) {
	if reason != protocol.StopDaemonRestart {
		return protocol.Session{}, protocol.Errorf(protocol.CodeInvalidInput,
			"a session is reopened only after a daemon restart, not for %q", reason)
	}
	s, err := d.manager.Reopen(d.opts.RecordDir, sessionID, d.opts.EventRetention, time.Now, d.rebuild)
	if err != nil {
		return protocol.Session{}, err
	}
	d.watch(s)
	return s.Describe(), nil
}

// Close closes a live session.
func (d *Daemon) Close(sessionID string) error { return d.manager.Remove(sessionID) }

// rebuild is the session.ReopenBuilder: the engine a record's session had —
// its workspace, its model bundle, its history — around the record itself.
// An orchestrator and its children are reopened; nothing else is.
func (d *Daemon) rebuild(r session.Reopening, log *session.EventLog) (*session.Session, error) {
	o := r.Origin
	var mode func(*Options)
	switch o.Role {
	case protocol.RoleOrchestrator:
		mode = d.orchestratorMode(o.ProjectID)
	case protocol.RoleChild:
		mode = d.childMode(r.ID, o.Owns)
	default:
		return nil, protocol.Errorf(protocol.CodeInvalidInput,
			"session %s is neither an orchestrator nor a child; only those are reopened", r.ID)
	}
	ws, err := validWorkspace(o.Workspace)
	if err != nil {
		return nil, err
	}
	opts, err := d.optionsFor(ws)
	if err != nil {
		return nil, err
	}
	// The bundle it was built with, as a continued session reconnects to its
	// own (build, Resume): not whatever model.name says today.
	if o.Model != "" && o.Family != "" {
		opts.Model, opts.Family, opts.Transport, opts.BaseURL = o.Model, o.Family, o.Transport, o.BaseURL
		if o.ContextWindow > 0 {
			opts.Window = o.ContextWindow
		}
	}
	opts.History = r.History
	mode(&opts)
	log.OnRecorded(d.conversations.Observe)
	return d.assemble(opts, r.ID, func(e *loop.Engine) *session.Session {
		return session.NewReopened(r, e, log, time.Now)
	})
}
