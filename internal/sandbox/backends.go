package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aguinelo/dcode/internal/policy"
)

// canonical resolves symlinks in a path before it reaches a sandbox profile.
//
// Without this the boundary the kernel enforces differs from the one policy
// evaluated: on macOS /var and /tmp are symlinks into /private, so a profile
// naming the unresolved path grants nothing and every write fails with no
// explanation. Falls back to the input when resolution is impossible, so a
// workspace that does not exist yet still produces a usable profile.
func canonical(p string) string {
	if real, err := filepath.EvalSymlinks(p); err == nil {
		return real
	}
	return p
}

// ---------- macOS: Apple Seatbelt ----------

// seatbelt confines through sandbox-exec, which ships with macOS. Driving the
// binary rather than calling sandbox_init through cgo is what keeps the static
// binary and cross-compilation working.
type seatbelt struct {
	bin          string
	allowNetwork func() bool
	scratch      []string
}

func (s *seatbelt) Name() string { return BackendSeatbelt }

func (s *seatbelt) Available() error {
	return lookPath(s.bin, "It ships with macOS; on other systems use a different backend.")
}

func (s *seatbelt) Wrap(ctx context.Context, workdir, command string, mode policy.SandboxMode) (*exec.Cmd, error) {
	profile, err := s.profile(workdir, mode, s.scratch)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, s.bin, "-p", profile, "sh", "-c", command)
	cmd.Dir = workdir
	return cmd, nil
}

// profile builds the Seatbelt policy.
//
// Deny by default, then grant the minimum: anything not named here is refused
// by the kernel rather than by us.
func (s *seatbelt) profile(workdir string, mode policy.SandboxMode, scratch []string) (string, error) {
	if workdir == "" {
		return "", fmt.Errorf("sandbox: workdir is required")
	}
	workdir = canonical(workdir)

	var b strings.Builder
	b.WriteString("(version 1)\n(deny default)\n")
	// Reading is always allowed: refusing it would block the interpreter from
	// loading before the command ever runs.
	b.WriteString("(allow file-read*)\n")
	b.WriteString("(allow process-exec)\n(allow process-fork)\n")
	b.WriteString("(allow sysctl-read)\n(allow mach-lookup)\n")
	b.WriteString("(allow signal (target self))\n")

	switch mode {
	case policy.ModeReadOnly:
		// No file-write rule at all. Writes fail in the kernel.
	case policy.ModeWorkspaceWrite:
		fmt.Fprintf(&b, "(allow file-write* (subpath %q))\n", workdir)
		// Temporary directories are where compilers and test runners stage
		// work; refusing them makes ordinary builds fail in ways that look
		// like the sandbox is broken rather than doing its job.
		for _, p := range []string{"/tmp", "/private/tmp", "/private/var/tmp", "/dev"} {
			fmt.Fprintf(&b, "(allow file-write* (subpath %q))\n", p)
		}
		// And the toolchain's own caches, which live outside the workspace
		// because they are shared across projects. Named one by one rather
		// than by granting the home that contains them.
		for _, p := range scratch {
			fmt.Fprintf(&b, "(allow file-write* (subpath %q))\n", canonical(p))
		}
	case policy.ModeFullAccess:
		b.WriteString("(allow file-write*)\n")
	default:
		return "", fmt.Errorf("sandbox: unknown mode %q", mode)
	}

	if permits(s.allowNetwork) || mode == policy.ModeFullAccess {
		b.WriteString("(allow network*)\n")
	}
	return b.String(), nil
}

// permits asks a boundary decision, treating an absent one as no.
//
// A nil decision must neither panic nor open. Both would be bad here and in
// opposite ways: a crash takes down a session over a wiring mistake, and a
// silent yes grants what nobody was asked about.
func permits(decide func() bool) bool { return decide != nil && decide() }

// ---------- Linux: bubblewrap ----------

// bubblewrap confines through unprivileged user namespaces.
type bubblewrap struct {
	bin          string
	allowNetwork func() bool
	scratch      []string
}

func (b *bubblewrap) Name() string { return BackendBubblewrap }

func (b *bubblewrap) Available() error {
	if err := lookPath(b.bin, installHint); err != nil {
		return err
	}
	// Presence is not enough: several distributions restrict unprivileged user
	// namespaces, and the failure that produces is opaque. Probing here turns
	// it into one clear message at startup rather than a confusing error on the
	// first command.
	probe := exec.Command(b.bin, "--ro-bind", "/", "/", "--dev", "/dev", "true")
	if out, err := probe.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s cannot create a namespace here: %s\n%s",
			ErrUnavailable, b.bin, strings.TrimSpace(string(out)), namespaceHint)
	}
	return nil
}

// exists reports whether a bind source is there, injectable so the mount rules
// can be asserted without a fixture directory. A test that has to create a real
// directory outside /tmp cannot be written portably — on Linux the temporary
// directory IS under /tmp — and the version of this test that worked around it
// with a skip left the Linux mount path unchecked on the one platform that uses
// it.
var exists = func(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// under reports whether path sits inside dir.
//
// Path-prefix, not string-prefix: "/tmpfoo" is not under "/tmp", and treating it
// as such would mount a directory the mode did not ask to mount.
func under(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

const installHint = "Install it with your package manager, for example: apt install bubblewrap."

const namespaceHint = `Unprivileged user namespaces appear to be restricted.
On Ubuntu 24.04 and later this is usually AppArmor; see
/etc/apparmor.d/ and the kernel.apparmor_restrict_unprivileged_userns sysctl.`

func (b *bubblewrap) Wrap(ctx context.Context, workdir, command string, mode policy.SandboxMode) (*exec.Cmd, error) {
	args, err := b.args(workdir, mode, b.scratch)
	if err != nil {
		return nil, err
	}
	args = append(args, "sh", "-c", command)
	cmd := exec.CommandContext(ctx, b.bin, args...)
	cmd.Dir = workdir
	return cmd, nil
}

func (b *bubblewrap) args(workdir string, mode policy.SandboxMode, scratch []string) ([]string, error) {
	if workdir == "" {
		return nil, fmt.Errorf("sandbox: workdir is required")
	}
	workdir = canonical(workdir)

	args := []string{
		"--ro-bind", "/", "/",
		"--dev", "/dev",
		"--proc", "/proc",
		"--die-with-parent",
		"--chdir", workdir,
	}

	// The tmpfs goes on BEFORE the workspace, always. bubblewrap applies mounts
	// in the order it is given them, and a workspace under /tmp mounted first
	// vanishes under the fresh tmpfs that lands on top of it — every command
	// then failing at "Can't chdir to <workspace>", before running.
	switch mode {
	case policy.ModeReadOnly:
		// Everything stays read-only. A writable /tmp is still granted because
		// too much ordinary tooling cannot start without one.
		args = append(args, "--tmpfs", "/tmp")
		// Read-only, so putting the workspace back is a --ro-bind: keeping it
		// visible must not be the thing that makes it writable.
		if under(workdir, "/tmp") {
			args = append(args, "--ro-bind", workdir, workdir)
		}
	case policy.ModeWorkspaceWrite:
		args = append(args, "--tmpfs", "/tmp", "--bind", workdir, workdir)
		// The toolchain's caches, bound writable one by one. A cache under
		// /tmp is already covered by the tmpfs and binding it again would put
		// the host's copy back over the fresh one.
		for _, p := range scratch {
			p = canonical(p)
			if under(p, "/tmp") || under(p, workdir) {
				continue
			}
			// bubblewrap refuses to bind a source that is not there, and the
			// refusal takes down the whole command rather than the one mount.
			// A cache directory that does not exist yet is ordinary: nothing
			// has compiled on this machine.
			if !exists(p) {
				continue
			}
			args = append(args, "--bind", p, p)
		}
	case policy.ModeFullAccess:
		args = []string{"--bind", "/", "/", "--dev", "/dev", "--proc", "/proc",
			"--die-with-parent", "--chdir", workdir}
	default:
		return nil, fmt.Errorf("sandbox: unknown mode %q", mode)
	}

	if !permits(b.allowNetwork) && mode != policy.ModeFullAccess {
		args = append(args, "--unshare-net")
	}
	return args, nil
}
