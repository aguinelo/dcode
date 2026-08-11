// Package sandbox enforces the boundary the policy package decides on.
//
// The boundary is applied by the operating system, never by path checks inside
// this process. An agent runs arbitrary commands; any check written in Go is
// bypassable by the very command it just allowed. In-process validation still
// exists upstream — it gives a better, faster error — but it is never the
// guarantee.
//
// Every backend wraps an external binary rather than calling a C API. That
// keeps CGO_ENABLED=0 and cross-compilation intact, which ADR-01 depends on.
//
// Spec: docs/specs/architecture/sandbox-policy/202608072336-*.
package sandbox

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/aguinelo/dcode/internal/policy"
)

// ErrUnavailable means the mechanism could not be established. It is never
// downgraded into running without a boundary: a harness that silently drops the
// sandbox is worse than one that never had it, because it promises what it does
// not deliver.
var ErrUnavailable = errors.New("sandbox: mechanism unavailable")

// Sandbox wraps a command so the operating system confines it.
type Sandbox interface {
	Name() string
	// Available reports whether this machine can establish the boundary. The
	// error names the missing binary and how to install it.
	Available() error
	// Wrap returns the command to execute, already confined.
	Wrap(ctx context.Context, workdir, command string, mode policy.SandboxMode) (*exec.Cmd, error)
}

// Backend names.
const (
	BackendAuto       = "auto"
	BackendSeatbelt   = "seatbelt"
	BackendBubblewrap = "bubblewrap"
	BackendNone       = "none"
)

// Config selects and tunes a backend.
type Config struct {
	Backend string
	Binary  string
	// AllowNetwork is asked once per command rather than read once per session.
	//
	// The permission can arrive mid-session: the user is asked at the first
	// crossing and answers "this project". A boundary decided at construction
	// would leave that answer with no effect until a restart — the user grants
	// it, the command fails anyway, and the only reading available to them is
	// that the permission did not work.
	//
	// Nil means denied, which is the reading that holds when nobody said
	// otherwise.
	AllowNetwork func() bool
	ProfileDir   string
}

// New returns the sandbox for cfg.
//
// BackendNone is accepted only together with full-access. In any other mode it
// is an initialisation error, because it would claim a boundary that does not
// exist.
func New(cfg Config, mode policy.SandboxMode) (Sandbox, error) {
	backend := cfg.Backend
	if backend == "" || backend == BackendAuto {
		backend = defaultBackend()
	}

	if backend == BackendNone {
		if mode != policy.ModeFullAccess {
			return nil, fmt.Errorf(
				"%w: backend %q only makes sense with sandbox mode %q, not %q — "+
					"running unconfined while claiming a boundary is worse than having none",
				ErrUnavailable, BackendNone, policy.ModeFullAccess, mode)
		}
		return &noneSandbox{}, nil
	}

	// Nil is denied. A boundary that opens because nobody wired a decision is
	// the wrong direction to be wrong in.
	allow := cfg.AllowNetwork
	if allow == nil {
		allow = func() bool { return false }
	}

	var s Sandbox
	switch backend {
	case BackendSeatbelt:
		s = &seatbelt{bin: orDefault(cfg.Binary, "sandbox-exec"), allowNetwork: allow}
	case BackendBubblewrap:
		s = &bubblewrap{bin: orDefault(cfg.Binary, "bwrap"), allowNetwork: allow}
	default:
		return nil, fmt.Errorf("sandbox: unknown backend %q; valid: %s, %s, %s, %s",
			backend, BackendAuto, BackendSeatbelt, BackendBubblewrap, BackendNone)
	}

	if err := s.Available(); err != nil {
		return nil, err
	}
	return s, nil
}

func defaultBackend() string {
	switch runtime.GOOS {
	case "darwin":
		return BackendSeatbelt
	case "linux":
		return BackendBubblewrap
	}
	// An unsupported platform must say so rather than silently pick something
	// that will not confine anything.
	return "unsupported"
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// Runner adapts a Sandbox to the interface the bash tool consumes, so the tool
// never reaches for os/exec itself.
type Runner struct {
	Sandbox Sandbox
	Mode    policy.SandboxMode
}

// Run executes command inside the boundary.
func (r Runner) Run(ctx context.Context, workdir, command string) (string, int, error) {
	if r.Sandbox == nil {
		return "", -1, ErrUnavailable
	}
	cmd, err := r.Sandbox.Wrap(ctx, workdir, command, r.Mode)
	if err != nil {
		return "", -1, err
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			// A non-zero exit is the command's answer, not a failure to run it.
			return string(out), ee.ExitCode(), nil
		}
		return string(out), -1, err
	}
	return string(out), 0, nil
}

// noneSandbox runs unconfined. Reachable only with full-access, checked in New.
type noneSandbox struct{}

func (noneSandbox) Name() string     { return BackendNone }
func (noneSandbox) Available() error { return nil }

func (noneSandbox) Wrap(ctx context.Context, workdir, command string, _ policy.SandboxMode) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = workdir
	return cmd, nil
}

// lookPath reports a missing binary with the name and how to get it, because
// "sandbox unavailable" alone sends the user reading source code.
func lookPath(bin, install string) error {
	if _, err := exec.LookPath(bin); err != nil {
		return fmt.Errorf("%w: %q not found in PATH. %s", ErrUnavailable, bin, install)
	}
	return nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
