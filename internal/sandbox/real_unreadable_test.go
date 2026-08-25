//go:build unix

package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/policy"
)

// The boundary itself, asked of the kernel.
//
// Reading is granted everywhere by design, so this is the one rule that takes
// something back — and a rule that takes something back is exactly the kind
// that has to be proven against the kernel rather than against the profile
// text. Measured before this existed: a command under workspace-write read a
// private SSH key without a murmur.
func TestARealNamedStoreCannotBeReadFromInside(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("a place outside the workspace is needed: %v", err)
	}
	store, err := os.MkdirTemp(home, "dcs")
	if err != nil {
		t.Skipf("the store could not be created: %v", err)
	}
	defer os.RemoveAll(store)

	secret := filepath.Join(store, "key")
	if err := os.WriteFile(secret, []byte("PRIVATE"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Readable when nothing is named — the behaviour that existed before, and
	// the control that makes the second half mean something.
	open, err := New(Config{}, policy.ModeWorkspaceWrite)
	if err != nil {
		t.Skipf("no sandbox available: %v", err)
	}
	ws := t.TempDir()
	r := Runner{Sandbox: open, Mode: Fixed(policy.ModeWorkspaceWrite)}
	if _, code, err := r.Run(context.Background(), ws, "cat "+shellQuote(secret)); err != nil || code != 0 {
		t.Fatalf("the control failed: a store nobody named must still be readable (code=%d err=%v)", code, err)
	}

	shut, err := New(Config{Unreadable: []string{store}}, policy.ModeWorkspaceWrite)
	if err != nil {
		t.Skipf("no sandbox available: %v", err)
	}
	r = Runner{Sandbox: shut, Mode: Fixed(policy.ModeWorkspaceWrite)}
	out, code, err := r.Run(context.Background(), ws, "cat "+shellQuote(secret))
	if err != nil {
		t.Fatalf("running under the sandbox failed outright: %v", err)
	}
	if code == 0 {
		t.Errorf("a named store was read from inside the sandbox (output %q)", out)
	}
}
