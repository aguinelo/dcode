package app

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/server"
	"github.com/aguinelo/dcode/internal/session"
)

// DaemonOptions configure the server process.
type DaemonOptions struct {
	SocketPath     string
	MaxSessions    int
	EventRetention int
	// RecordDir is where each session is written, one JSONL file per session.
	// Empty turns recording off, and then retention is a hard horizon: a
	// client away longer than it gets events_expired, and nobody can read the
	// session afterwards either.
	RecordDir string
	// RecordBudget is how much history survives. Applied when a session opens
	// rather than on a timer: nothing should be deleting a person's history
	// while the program is not running.
	RecordBudget    session.PruneBudget
	ApprovalTimeout time.Duration
	Base            Options
	// Log receives operational notices. Nil silences them, which a test wants
	// and a daemon must not.
	Log func(string)
}

// DefaultSocketPath resolves where the daemon listens.
//
// Kept short deliberately: a Unix socket path is capped near 104 bytes on
// macOS, and the XDG state directory alone can exhaust that. Falling back to
// the temp directory with the uid keeps two users on one machine apart.
func DefaultSocketPath(env func(string) string) string {
	if v := env("DCODE_SOCKET"); v != "" {
		return v
	}
	if dir := env("XDG_RUNTIME_DIR"); dir != "" {
		return filepath.Join(dir, "dcode.sock")
	}
	tmp := env("TMPDIR")
	if tmp == "" {
		tmp = "/tmp"
	}
	return filepath.Join(tmp, fmt.Sprintf("dcode-%d.sock", osUID()))
}

// Daemon owns the server and the session manager.
type Daemon struct {
	opts    DaemonOptions
	manager *session.Manager
	server  *server.Server
}

// NewDaemon builds the daemon.
func NewDaemon(opts DaemonOptions) *Daemon {
	if opts.MaxSessions <= 0 {
		opts.MaxSessions = 64
	}
	if opts.ApprovalTimeout <= 0 {
		opts.ApprovalTimeout = 120 * time.Second
	}
	d := &Daemon{opts: opts, manager: session.NewManager(opts.MaxSessions)}
	d.server = server.New(server.Config{
		SocketPath:  opts.SocketPath,
		Manager:     d.manager,
		Build:       d.build,
		MaxSessions: opts.MaxSessions,
		// Raised once at boot rather than per session, so it is read before
		// anything runs rather than scrolling past during work.
		DefaultMode:   opts.Base.SandboxMode,
		DefaultPolicy: opts.Base.Policy,
		Log:           opts.Log,
	})
	return d
}

// build creates a session for a request.
//
// Each session gets its own sandbox, resolver and provider: a session is the
// unit of confinement, so sharing any of it across sessions would let one
// workspace's boundary apply to another.
func (d *Daemon) build(req protocol.CreateSessionRequest) (*session.Session, error) {
	ws, err := validWorkspace(req.Workspace)
	if err != nil {
		return nil, err
	}

	opts := d.opts.Base
	opts.Workspace = ws
	var (
		carried      []protocol.Event
		carriedFrom  string
		carriedTurns int
		turns        int
		cerr         error
	)
	if req.Model != "" {
		opts.Model = req.Model
	}
	if req.Resume != "" {
		// The record is the only copy of what happened, so continuing means
		// reading it. A failure here is fatal to the request rather than
		// quietly starting fresh: someone who asked to continue and got an
		// empty session would not find out until the model had forgotten
		// everything.
		path := filepath.Join(d.opts.RecordDir, req.Resume+".jsonl")
		history, herr := session.Rebuild(path)
		if herr != nil {
			return nil, protocol.Errorf(protocol.CodeSessionNotFound,
				"session %s cannot be continued: %v", req.Resume, herr)
		}
		opts.History = history
		// Read a second time, as events rather than messages. The model needs
		// what was said; the person needs to see that it survived, and the next
		// session to continue this one needs both — so the events go in this
		// session's log and the file behind it.
		carried, turns, cerr = session.Carry(path)
		if cerr != nil {
			return nil, protocol.Errorf(protocol.CodeSessionNotFound,
				"session %s cannot be continued: %v", req.Resume, cerr)
		}
		carriedFrom, carriedTurns = req.Resume, turns
	}
	if req.SandboxMode != "" {
		mode, perr := parseMode(req.SandboxMode)
		if perr != nil {
			return nil, perr
		}
		opts.SandboxMode = mode
	}

	id := session.NewID(time.Now, randomUint32)
	log := session.NewEventLog(id, d.opts.EventRetention, time.Now)
	// Opening a session is when history is tidied. The live set is what the
	// manager holds, because a session being written is not garbage however
	// old its first line is.
	if d.opts.RecordDir != "" {
		budget := d.opts.RecordBudget
		budget.Live = d.manager.LiveIDs()
		if n, err := session.Prune(d.opts.RecordDir, budget); err != nil && d.opts.Log != nil {
			d.opts.Log(fmt.Sprintf("session records could not be pruned: %v", err))
		} else if n > 0 && d.opts.Log != nil {
			d.opts.Log(fmt.Sprintf("removed %d old session record(s)", n))
		}
	}

	// The record serves two readers with one file: a client rejoining below the
	// retention horizon, and a person reading afterwards what the agent did.
	if rec, err := session.NewRecord(d.opts.RecordDir, id); err == nil {
		log.SetRecord(rec)
	} else if d.opts.Log != nil {
		// Not fatal. A session that cannot be recorded is still a session, and
		// refusing to start one because a directory is unwritable would be the
		// audit trail holding the product hostage.
		d.opts.Log(fmt.Sprintf("session %s is not being recorded: %v", id, err))
	}

	// The session is its own emitter and its own approver: the loop announces
	// a crossing through the log and blocks until a client answers, which is
	// what lets any attached client resolve it.
	var sess *session.Session
	// Bound late, like the emitter and the approver above it: the engine is
	// built before the session that holds the queue exists.
	opts.Steer = func() string {
		if sess == nil {
			return ""
		}
		return sess.TakeSteering()
	}

	appSession, err := New(opts,
		emitterFunc(func(t protocol.EventType, payload any) {
			if sess != nil {
				sess.Emit(t, payload)
			}
		}),
		approverFunc(func(ctx context.Context, r protocol.ApprovalRequest) (protocol.ApprovalDecision, error) {
			if sess == nil {
				return protocol.ApprovalDeny, nil
			}
			return sess.Approve(ctx, r, d.opts.ApprovalTimeout)
		}),
	)
	if err != nil {
		return nil, err
	}

	sess = session.New(id, opts.Workspace, opts.Model, string(opts.SandboxMode),
		appSession.Engine, log, time.Now)
	sess.ContextWindow = appSession.ContextWindow
	sess.Carried, sess.CarriedFrom, sess.CarriedTurns = carried, carriedFrom, carriedTurns

	// The same record the sandbox is asking, not a second copy: two would
	// answer differently the moment one of them is granted.
	sess.Standing = appSession.Standing

	return sess, nil
}

// validWorkspace checks the one field a client fully controls.
//
// The workspace is the anchor of every boundary the session will enforce, so a
// path that is not an existing directory is refused at creation rather than at
// the first tool call — by which point the user has already started waiting,
// and the failure looks like the tool's rather than the request's.
func validWorkspace(path string) (string, error) {
	if path == "" {
		return "", protocol.Errorf(protocol.CodeWorkspaceInvalid, "a workspace is required")
	}
	if !filepath.IsAbs(path) {
		return "", protocol.Errorf(protocol.CodeWorkspaceInvalid,
			"the workspace %q must be an absolute path", path)
	}
	fi, err := os.Stat(path)
	if err != nil {
		return "", protocol.Errorf(protocol.CodeWorkspaceInvalid,
			"the workspace %s cannot be opened: %v", path, err)
	}
	if !fi.IsDir() {
		return "", protocol.Errorf(protocol.CodeWorkspaceInvalid,
			"the workspace %s is not a directory", path)
	}
	return filepath.Clean(path), nil
}

// Serve runs until the context is cancelled.
func (d *Daemon) Serve(ctx context.Context) error { return d.server.Serve(ctx) }

// Listen binds the socket ahead of Serve, so a caller can report the address
// before the blocking call.
func (d *Daemon) Listen() error { return d.server.Listen() }

// Addr returns the socket path.
func (d *Daemon) Addr() string { return d.server.Addr() }

// Manager exposes the session manager.
func (d *Daemon) Manager() *session.Manager { return d.manager }

type emitterFunc func(protocol.EventType, any)

func (f emitterFunc) Emit(t protocol.EventType, payload any) { f(t, payload) }

type approverFunc func(context.Context, protocol.ApprovalRequest) (protocol.ApprovalDecision, error)

func (f approverFunc) Approve(ctx context.Context, r protocol.ApprovalRequest) (protocol.ApprovalDecision, error) {
	return f(ctx, r)
}

func randomUint32() uint32 {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0
	}
	return binary.BigEndian.Uint32(b[:])
}

func parseMode(s string) (policy.SandboxMode, error) { return policy.ParseMode(s) }

func osUID() int { return os.Getuid() }
