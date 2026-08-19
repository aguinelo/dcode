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
// Both backends carry an `unreadable` field, and the reason is written once here.
//
// Reading is allowed everywhere on purpose: refusing it outright stops the
// interpreter loading before the command runs. The cost is that the sandbox
// protects no secret — measured here, a command under workspace-write read a
// private SSH key without a murmur. For editing code that is a fair trade; for
// a session that reaches servers it leaves the most valuable thing on the
// machine on the table.
//
// A named set is the answer, not a blanket rule. Every candidate for a default
// is needed by some ordinary tool — hiding ~/.ssh breaks `git push`, hiding
// ~/.aws breaks the aws CLI — and a default that breaks the ordinary case is a
// default people switch off entirely.
type seatbelt struct {
	bin string
	// unreadable are paths put out of reach for this session. See
	// unreadableDoc.
	unreadable   []string
	allowNetwork func() bool
	scratch      []string
}

func (s *seatbelt) Name() string { return BackendSeatbelt }

func (s *seatbelt) Available() error {
	if err := lookPath(s.bin, "It ships with macOS; on other systems use a different backend."); err != nil {
		return err
	}
	// Presence is not enough, which is the same lesson the Linux backend
	// already carries: the kernel refuses to apply a profile from inside a
	// process that is already confined, and reports
	// `sandbox_apply: Operation not permitted`.
	//
	// Without this, a session running inside dcode failed six boundary tests at
	// once, and nothing in that failure distinguishes "this environment cannot
	// nest" from "the work is wrong". An agent reading it spent a session
	// fixing the harness.
	//
	// The probe profile grants everything on purpose: what is being asked is
	// whether the kernel will accept any profile at all from here, not whether
	// a particular one is right.
	probe := exec.Command(s.bin, "-p", "(version 1)(allow default)", "true")
	if out, err := probe.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s cannot apply a profile here: %s\n%s",
			ErrUnavailable, s.bin, strings.TrimSpace(string(out)), nestedHint)
	}
	return nil
}

// nestedHint names the cause people actually hit, because the kernel's own
// message does not.
const nestedHint = "This usually means the process is already inside a sandbox; " +
	"a profile cannot be applied from within one."

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

	// After the blanket allow, never before: Seatbelt takes the LAST matching
	// rule, so a deny written above would be overruled by the line that grants
	// everything. Not in full-access, which promises no boundary and must not
	// quietly keep one.
	if mode != policy.ModeFullAccess {
		for _, p := range s.unreadable {
			fmt.Fprintf(&b, "(deny file-read* (subpath %q))\n", p)
		}
	}

	// The writable set, collected once and used twice: file writes below, and
	// unix sockets further down. Two rules reading the same list is what keeps
	// them from drifting into two different answers about the same boundary.
	var writable []string

	switch mode {
	case policy.ModeReadOnly:
		// Nothing writable, so no file-write rule at all. Writes fail in the
		// kernel.
	case policy.ModeWorkspaceWrite:
		writable = append(writable, workdir)
		// Temporary directories are where compilers and test runners stage
		// work; refusing them makes ordinary builds fail in ways that look
		// like the sandbox is broken rather than doing its job.
		writable = append(writable, "/tmp", "/private/tmp", "/private/var/tmp", "/dev")
		// And the toolchain's own caches, which live outside the workspace
		// because they are shared across projects. Named one by one rather
		// than by granting the home that contains them.
		for _, p := range scratch {
			writable = append(writable, canonical(p))
		}
	case policy.ModeFullAccess:
		b.WriteString("(allow file-write*)\n")
	default:
		return "", fmt.Errorf("sandbox: unknown mode %q", mode)
	}

	for _, p := range writable {
		fmt.Fprintf(&b, "(allow file-write* (subpath %q))\n", p)
	}

	switch {
	case mode == policy.ModeFullAccess:
		// This mode claims no boundary. Narrowing it would be pretending to
		// confine something that says it does not.
		b.WriteString("(allow network*)\n")
	case permits(s.allowNetwork):
		// The grant is for the network, not for this machine's own daemons.
		// `(allow network*)` also covers unix sockets, and a unix socket is
		// how an unconfined privileged process is reached: hand over
		// /var/run/docker.sock and `docker run -v /:/host` writes anywhere on
		// the machine, as root, from inside workspace-write.
		//
		// Found by the harness in itself. Asked to fix a guard, and reasoning
		// about the Linux side of CI, a model ran `docker version`, `docker
		// images` and then `docker run --rm ubuntu:26.10 ...` from inside the
		// sandbox. All three succeeded, and nothing was asked.
		b.WriteString("(allow network-outbound (remote ip \"*:*\"))\n")
		// Listening is not reaching out, and a suite that cannot listen cannot
		// run: httptest binds a port, and so does anything that tests a
		// server. Granting only outbound left every such test failing at
		// `bind: operation not permitted`, which is how the first version of
		// this rule broke the suite it was meant to protect.
		b.WriteString("(allow network-bind (local ip \"*:*\"))\n")
		b.WriteString("(allow network-inbound (local ip \"*:*\"))\n")
		// Denying every unix socket denies name resolution too: getaddrinfo
		// talks to mDNSResponder over one. It is named because it resolves
		// names — it does not act on the caller's behalf, which is the
		// property that separates it from a container runtime's socket.
		b.WriteString("(allow network-outbound (literal \"/private/var/run/mDNSResponder\"))\n")
		// And a unix socket is reachable exactly where writing already is.
		//
		// That is the whole rule, and it is the mode's own boundary rather
		// than a second one invented beside it: a socket the process may
		// create is a socket it may talk to, and /var/run — where a container
		// runtime listens — is not somewhere workspace-write can write.
		// Naming the writable set twice keeps the two answers from drifting
		// apart.
		for _, p := range writable {
			fmt.Fprintf(&b, "(allow network-bind (subpath %q))\n", p)
			fmt.Fprintf(&b, "(allow network-outbound (subpath %q))\n", p)
		}
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
	// sockets are the unix sockets that lead out of the sandbox — see
	// LocalSockets. Named here rather than consulted from the environment,
	// for the same reason scratch is: this package builds profiles and does
	// not read the world.
	sockets []string
	// unreadable are paths put out of reach for this session, for the reason
	// written above seatbelt.
	unreadable []string
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

	if mode != policy.ModeFullAccess {
		// A unix socket on a path is not in any network namespace, so
		// --unshare-net does not cover it and the read-only bind of / does not
		// either: connecting to a socket is not a write. Each one is covered
		// with /dev/null, which is not a socket, so connect fails in the
		// kernel.
		//
		// This is a list, and a list is weaker than a boundary — on macOS the
		// profile denies every unix socket instead. Here the whole filesystem
		// is bound by design, so the sockets have to be named. Naming them is
		// worse than denying them and better than handing them over.
		// A named store is covered with a fresh tmpfs: the mount is what the
		// kernel reads, so there is nothing left to allow or deny. Same move
		// as the sockets below, and for the same reason.
		for _, p := range b.unreadable {
			if !exists(p) {
				continue
			}
			args = append(args, "--tmpfs", p)
		}
		for _, p := range b.sockets {
			// bubblewrap refuses a bind whose source or target is absent, and
			// the refusal takes down the whole command rather than the one
			// mount.
			if !exists(p) {
				continue
			}
			// Not canonicalised: bubblewrap resolves the target itself, and
			// a path that resolves differently here than inside would cover
			// the wrong thing.
			args = append(args, "--ro-bind", "/dev/null", p)
		}
	}

	if !permits(b.allowNetwork) && mode != policy.ModeFullAccess {
		args = append(args, "--unshare-net")
	}
	return args, nil
}
