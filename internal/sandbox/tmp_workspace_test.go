package sandbox

import (
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
			args, err := b.args("/tmp/session-1234/ws", mode)
			if err != nil {
				t.Fatal(err)
			}

			tmpfs := indexOf(args, "--tmpfs", "/tmp")
			if tmpfs < 0 {
				t.Fatalf("no writable /tmp: %v", args)
			}

			bind := indexOf(args, "--bind", "/tmp/session-1234/ws")
			if bind < 0 {
				bind = indexOf(args, "--ro-bind", "/tmp/session-1234/ws")
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

	args, err := b.args("/tmp/ws", policy.ModeReadOnly)
	if err != nil {
		t.Fatal(err)
	}
	if indexOf(args, "--bind", "/tmp/ws") >= 0 {
		t.Errorf("read-only mounted the workspace writable: %v", args)
	}
	if indexOf(args, "--ro-bind", "/tmp/ws") < 0 {
		t.Errorf("read-only lost the workspace under the tmpfs: %v", args)
	}

	// Outside /tmp nothing changes: the whole filesystem is already mounted, so
	// a second mount of the workspace would be noise in the argument list.
	outside, err := b.args("/home/u/proj", policy.ModeReadOnly)
	if err != nil {
		t.Fatal(err)
	}
	if indexOf(outside, "--ro-bind", "/home/u/proj") >= 0 {
		t.Errorf("a workspace outside /tmp was mounted again for no reason: %v", outside)
	}
}
