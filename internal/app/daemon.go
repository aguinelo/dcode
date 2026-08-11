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
	// EventSpillDir is where trimmed events are kept so a replay below the
	// retention horizon still answers. Empty makes retention a hard limit.
	EventSpillDir   string
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
	if req.Model != "" {
		opts.Model = req.Model
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
	// Retention without a spill is a hard horizon: a client away longer than it
	// gets events_expired, and the session it was watching becomes unreadable
	// for a reason that has nothing to do with the session.
	if sp, err := session.NewSpill(d.opts.EventSpillDir, id); err == nil {
		log.SetSpill(sp)
	}

	// The session is its own emitter and its own approver: the loop announces
	// a crossing through the log and blocks until a client answers, which is
	// what lets any attached client resolve it.
	var sess *session.Session
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
