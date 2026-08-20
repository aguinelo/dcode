package sandbox

import (
	"os"
	"testing"

	"github.com/aguinelo/dcode/internal/policy"
)

// indexOf reports where an argument sits, or -1. Order is the whole point here:
// bubblewrap applies its mounts in the order it is given them, so two correct
// arguments in the wrong sequence produce a sandbox that is not what either of
// them asked for.
func indexOf(args []string, want ...string) int {
	for i := 0; i+len(want) <= len(args); i++ {
		ok := true
		for j, w := range want {
			if args[i+j] != w {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}

// A workspace under /tmp has to survive the tmpfs that gets mounted there.
//
// It did not. The writable /tmp was appended AFTER the workspace bind, so for a
// workspace anywhere under /tmp the fresh tmpfs was mounted straight over it and
// the directory stopped existing inside the sandbox. Every command then failed
// at `bwrap: Can't chdir to <workspace>` — before running.
//
// Invisible until now for two reasons that compounded: on macOS a temporary
// directory lives under /var/folders rather than /tmp, so seatbelt never met the
// case; and on Linux CI could not create a namespace at all, so every test that
// would have caught it skipped.
func TestAWorkspaceUnderTmpSurvivesTheWritableTmpfs(t *testing.T) {
	for _, mode := range []policy.SandboxMode{policy.ModeReadOnly, policy.ModeWorkspaceWrite} {
		t.Run(string(mode), func(t *testing.T) {
			b := &bubblewrap{bin: "bwrap"}
			// Asserted on the canonical spelling, which is what args() mounts.
			// Hard-coding the unresolved one made the assertion depend on
			// whether the directory existed on the machine running it.
			ws := canonical("/tmp/session-1234/ws")
			args, err := b.args("/tmp/session-1234/ws", mode, nil)
			if err != nil {
				t.Fatal(err)
			}

			tmpfs := indexOf(args, "--tmpfs", "/tmp")
			if tmpfs < 0 {
				t.Fatalf("no writable /tmp: %v", args)
			}

			bind := indexOf(args, "--bind", ws)
			if bind < 0 {
				bind = indexOf(args, "--ro-bind", ws)
			}
			if bind < 0 {
				t.Fatalf("the workspace is not mounted at all, so the tmpfs hides it: %v", args)
			}
			if bind < tmpfs {
				t.Errorf("the workspace is mounted before the tmpfs that covers /tmp, "+
					"so it disappears inside the sandbox: %v", args)
			}
		})
	}
}

// A workspace outside /tmp is unaffected, and read-only stays read-only: the
// mount that keeps the workspace visible must not be the one that makes it
// writable.
func TestKeepingTheWorkspaceVisibleDoesNotMakeItWritable(t *testing.T) {
	b := &bubblewrap{bin: "bwrap"}

	ws := canonical("/tmp/ws")
	args, err := b.args("/tmp/ws", policy.ModeReadOnly, nil)
	if err != nil {
		t.Fatal(err)
	}
	if indexOf(args, "--bind", ws) >= 0 {
		t.Errorf("read-only mounted the workspace writable: %v", args)
	}
	if indexOf(args, "--ro-bind", ws) < 0 {
		t.Errorf("read-only lost the workspace under the tmpfs: %v", args)
	}

	// Outside /tmp nothing changes: the whole filesystem is already mounted, so
	// a second mount of the workspace would be noise in the argument list.
	outside, err := b.args("/home/u/proj", policy.ModeReadOnly, nil)
	if err != nil {
		t.Fatal(err)
	}
	if indexOf(outside, "--ro-bind", canonical("/home/u/proj")) >= 0 {
		t.Errorf("a workspace outside /tmp was mounted again for no reason: %v", outside)
	}
}

// Whether the workspace is mounted back must not depend on whether it happens
// to exist on the machine running the test.
//
// canonical() resolves symlinks with filepath.EvalSymlinks, which only resolves
// a path that EXISTS. The three /tmp decisions in args() then compared a
// canonicalised workdir against the literal string "/tmp" — like against
// unlike. Where /tmp is a symlink, an existing workspace resolved to
// /private/tmp/ws, stopped matching "/tmp", and lost its mount; a workspace
// that did not exist yet kept the spelling it was given and got one.
//
// Same directory, two answers, decided by the filesystem rather than by the
// mode. It hid for a day behind Go's test cache, and every `make check` said
// green — a test of a security boundary that reports differently on different
// machines is worse than one that fails, because it is believed.
//
// On Linux, where bubblewrap actually runs, /tmp is not a symlink and both
// spellings already agree: this is red exactly where the defect lives.
func TestTheWorkspaceMountDoesNotDependOnThePathExisting(t *testing.T) {
	b := &bubblewrap{bin: "bwrap"}
	const ws = "/tmp/dcode-mount-invariance"

	decide := func() bool {
		t.Helper()
		args, err := b.args(ws, policy.ModeReadOnly, nil)
		if err != nil {
			t.Fatal(err)
		}
		return indexOf(args, "--ro-bind", ws) >= 0 ||
			indexOf(args, "--ro-bind", canonical(ws)) >= 0
	}

	absent := decide()

	if err := os.MkdirAll(ws, 0o755); err != nil {
		t.Skipf("cannot create %s here: %v", ws, err)
	}
	t.Cleanup(func() { os.RemoveAll(ws) })
	present := decide()

	if absent != present {
		t.Errorf("the workspace is mounted back when the directory is absent=%v "+
			"and present=%v — the decision follows the filesystem instead of the mode",
			absent, present)
	}
	if !present {
		t.Error("a workspace under /tmp that exists lost its mount under the tmpfs")
	}
}
