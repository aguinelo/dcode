package policy

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// alwaysIn / neverIn stand in for a resolver so the decision table can be
// exercised without touching a filesystem.
func alwaysIn(Access) bool { return true }
func neverIn(Access) bool  { return false }

func read(p string) Request  { return Request{Tool: "read", Paths: []Access{{Path: p}}} }
func write(p string) Request { return Request{Tool: "edit", Paths: []Access{{Path: p, Write: true}}} }
func net() Request           { return Request{Tool: "bash", Network: true} }

// Every cell of the mode table gets an assertion. Sampling would be cheaper and
// exactly wrong: the untested cell is the one that will be wrong, and each cell
// here is a security decision.
func TestModeTableIsComplete(t *testing.T) {
	for _, tc := range []struct {
		name string
		req  Request
		in   func(Access) bool
		mode SandboxMode
		want Decision
	}{
		// read-only
		{"read inside / read-only", read("/w/a"), alwaysIn, ModeReadOnly, DecisionAllow},
		{"read outside / read-only", read("/etc/x"), neverIn, ModeReadOnly, DecisionEscalate},
		{"write inside / read-only", write("/w/a"), alwaysIn, ModeReadOnly, DecisionDeny},
		{"write outside / read-only", write("/etc/x"), neverIn, ModeReadOnly, DecisionDeny},
		{"network / read-only", net(), alwaysIn, ModeReadOnly, DecisionDeny},

		// workspace-write
		{"read inside / workspace-write", read("/w/a"), alwaysIn, ModeWorkspaceWrite, DecisionAllow},
		{"read outside / workspace-write", read("/etc/x"), neverIn, ModeWorkspaceWrite, DecisionEscalate},
		{"write inside / workspace-write", write("/w/a"), alwaysIn, ModeWorkspaceWrite, DecisionAllow},
		{"write outside / workspace-write", write("/etc/x"), neverIn, ModeWorkspaceWrite, DecisionEscalate},
		{"network / workspace-write", net(), alwaysIn, ModeWorkspaceWrite, DecisionEscalate},

		// full-access
		{"read inside / full-access", read("/w/a"), alwaysIn, ModeFullAccess, DecisionAllow},
		{"read outside / full-access", read("/etc/x"), neverIn, ModeFullAccess, DecisionAllow},
		{"write inside / full-access", write("/w/a"), alwaysIn, ModeFullAccess, DecisionAllow},
		{"write outside / full-access", write("/etc/x"), neverIn, ModeFullAccess, DecisionAllow},
		{"network / full-access", net(), alwaysIn, ModeFullAccess, DecisionAllow},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Evaluate(tc.req, tc.mode, PolicyOnRequest, Rules{}, tc.in)
			if got.Decision != tc.want {
				t.Errorf("got %s want %s (%s)", got.Decision, tc.want, got.Reason)
			}
		})
	}
}

func TestPolicyFilterIsComplete(t *testing.T) {
	for _, tc := range []struct {
		name string
		req  Request
		in   func(Access) bool
		pol  ApprovalPolicy
		want Decision
	}{
		// untrusted asks before writing even inside the workspace; that is the
		// only practical difference from on-request.
		{"workspace write / untrusted", write("/w/a"), alwaysIn, PolicyUntrusted, DecisionEscalate},
		{"workspace write / on-request", write("/w/a"), alwaysIn, PolicyOnRequest, DecisionAllow},
		{"workspace write / never", write("/w/a"), alwaysIn, PolicyNever, DecisionAllow},

		{"workspace read / untrusted", read("/w/a"), alwaysIn, PolicyUntrusted, DecisionAllow},
		{"workspace read / on-request", read("/w/a"), alwaysIn, PolicyOnRequest, DecisionAllow},
		{"workspace read / never", read("/w/a"), alwaysIn, PolicyNever, DecisionAllow},

		{"crossing / untrusted", net(), alwaysIn, PolicyUntrusted, DecisionEscalate},
		{"crossing / on-request", net(), alwaysIn, PolicyOnRequest, DecisionEscalate},
		{"crossing / never", net(), alwaysIn, PolicyNever, DecisionDeny},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Evaluate(tc.req, ModeWorkspaceWrite, tc.pol, Rules{}, tc.in)
			if got.Decision != tc.want {
				t.Errorf("got %s want %s (%s)", got.Decision, tc.want, got.Reason)
			}
		})
	}
}

// Two invariants worth stating on their own, because they are the ones an
// optimisation would quietly break.
func TestReadOnlyNeverAllowsAWrite(t *testing.T) {
	for _, pol := range []ApprovalPolicy{PolicyUntrusted, PolicyOnRequest, PolicyNever} {
		for _, in := range []func(Access) bool{alwaysIn, neverIn} {
			got := Evaluate(write("/w/a"), ModeReadOnly, pol, Rules{}, in)
			if got.Decision == DecisionAllow {
				t.Errorf("policy %s: read-only allowed a write", pol)
			}
		}
	}
}

func TestNeverPolicyNeverEscalates(t *testing.T) {
	for _, mode := range []SandboxMode{ModeReadOnly, ModeWorkspaceWrite, ModeFullAccess} {
		for _, req := range []Request{read("/x"), write("/x"), net()} {
			got := Evaluate(req, mode, PolicyNever, Rules{}, neverIn)
			if got.Decision == DecisionEscalate {
				t.Errorf("mode %s: 'never' produced an escalation it cannot resolve", mode)
			}
		}
	}
}

// A call that both reads and writes is a write for boundary purposes. Checking
// reads first would let a mixed request through on the weaker verdict.
func TestMixedReadWriteIsTreatedAsWrite(t *testing.T) {
	req := Request{Tool: "edit", Paths: []Access{
		{Path: "/w/in", Write: false},
		{Path: "/outside", Write: true},
	}}
	got := Evaluate(req, ModeWorkspaceWrite, PolicyOnRequest, Rules{}, func(a Access) bool {
		return strings.HasPrefix(a.Path, "/w/")
	})
	if got.Decision != DecisionEscalate || got.Boundary != BoundaryFilesystemWrit {
		t.Errorf("got %s/%s want escalate/filesystem_write", got.Decision, got.Boundary)
	}
}

func TestEvaluateIsPure(t *testing.T) {
	req := Request{Tool: "bash", Paths: []Access{{Path: "/w/a", Write: true}}, Network: true}
	first := Evaluate(req, ModeWorkspaceWrite, PolicyOnRequest, Rules{}, alwaysIn)
	for i := 0; i < 100; i++ {
		if got := Evaluate(req, ModeWorkspaceWrite, PolicyOnRequest, Rules{}, alwaysIn); got != first {
			t.Fatalf("run %d differs: %+v vs %+v", i, got, first)
		}
	}
}

func TestParseRejectsUnknownValues(t *testing.T) {
	for _, s := range []string{"", "readonly", "READ-ONLY", "workspace_write", "yolo"} {
		if _, err := ParseMode(s); err == nil {
			t.Errorf("mode %q should be rejected", s)
		}
	}
	for _, s := range []string{"", "always", "ON-REQUEST", "on_request"} {
		if _, err := ParsePolicy(s); err == nil {
			t.Errorf("policy %q should be rejected", s)
		}
	}
	if m, err := ParseMode("workspace-write"); err != nil || m != ModeWorkspaceWrite {
		t.Errorf("valid mode rejected: %v", err)
	}
	if p, err := ParsePolicy("never"); err != nil || p != PolicyNever {
		t.Errorf("valid policy rejected: %v", err)
	}
}

// The single most common boundary bug in existence: /proj2 passing for /proj.
func TestContainmentIsByComponentNotByPrefix(t *testing.T) {
	root := t.TempDir()
	proj := filepath.Join(root, "proj")
	proj2 := filepath.Join(root, "proj2")
	for _, d := range []string{proj, proj2} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	r, err := NewResolver(proj)
	if err != nil {
		t.Fatal(err)
	}
	sibling, err := r.Resolve(filepath.Join(proj2, "file.go"), false)
	if err != nil {
		t.Fatal(err)
	}
	if r.InWorkspace(sibling) {
		t.Errorf("%s must not be contained in %s", sibling.Path, proj)
	}

	inside, err := r.Resolve("file.go", false)
	if err != nil {
		t.Fatal(err)
	}
	if !r.InWorkspace(inside) {
		t.Errorf("%s should be inside %s", inside.Path, proj)
	}
}

// Follow the symlink, then check. The other order loses the boundary to one
// `ln -s`.
func TestSymlinkPointingOutIsACrossing(t *testing.T) {
	root := t.TempDir()
	ws := filepath.Join(root, "ws")
	outside := filepath.Join(root, "secrets")
	for _, d := range []string{ws, outside} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	target := filepath.Join(outside, "key.txt")
	if err := os.WriteFile(target, []byte("shh"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(ws, "innocent.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	r, err := NewResolver(ws)
	if err != nil {
		t.Fatal(err)
	}
	a, err := r.Resolve(link, false)
	if err != nil {
		t.Fatal(err)
	}
	if r.InWorkspace(a) {
		t.Errorf("a symlink to %s must not read as inside the workspace (resolved to %s)",
			outside, a.Path)
	}
}

func TestDotDotEscapeIsACrossingNotAnError(t *testing.T) {
	root := t.TempDir()
	ws := filepath.Join(root, "ws")
	if err := os.MkdirAll(ws, 0o755); err != nil {
		t.Fatal(err)
	}
	r, err := NewResolver(ws)
	if err != nil {
		t.Fatal(err)
	}
	// It must resolve — and then be judged outside. Erroring here would hide
	// the attempt instead of surfacing it as a boundary decision.
	a, err := r.Resolve("../../etc/passwd", false)
	if err != nil {
		t.Fatalf("escaping paths must resolve, not error: %v", err)
	}
	if r.InWorkspace(a) {
		t.Errorf("%s must be outside", a.Path)
	}
}

// Creating a new file is legitimate work, so a path that does not exist yet has
// to resolve against its nearest real ancestor.
func TestNonexistentPathInsideWorkspaceResolves(t *testing.T) {
	ws := t.TempDir()
	r, err := NewResolver(ws)
	if err != nil {
		t.Fatal(err)
	}
	a, err := r.Resolve("new/dir/file.go", true)
	if err != nil {
		t.Fatal(err)
	}
	if !r.InWorkspace(a) {
		t.Errorf("a new file under the workspace must be inside: %s", a.Path)
	}

	out, err := r.Resolve(filepath.Join(filepath.Dir(ws), "elsewhere", "new.go"), true)
	if err != nil {
		t.Fatal(err)
	}
	if r.InWorkspace(out) {
		t.Errorf("a new file outside must still be outside: %s", out.Path)
	}
}

func TestResolverRejectsBadWorkspaces(t *testing.T) {
	for _, ws := range []string{"", "relative/path"} {
		if _, err := NewResolver(ws); err == nil {
			t.Errorf("workspace %q should be rejected", ws)
		}
	}
	if _, err := NewResolver(filepath.Join(t.TempDir(), "does-not-exist")); err != nil {
		// A workspace that does not exist yet is a configuration problem, but
		// resolving it is still well-defined; it must not panic either way.
		t.Logf("nonexistent workspace: %v", err)
	}
}

func TestResolveRejectsEmptyPath(t *testing.T) {
	r, err := NewResolver(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Resolve("", false); err == nil {
		t.Error("an empty path must be rejected")
	}
}

// Evaluate must not reach for the filesystem or the clock; the decision has to
// be reproducible from its arguments alone.
func TestPolicyFileIsPure(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "policy.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if path != "fmt" {
			t.Errorf("policy.go imports %q; the decision layer must stay pure", path)
		}
	}
}
