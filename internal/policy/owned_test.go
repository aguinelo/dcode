package policy

import (
	"path/filepath"
	"testing"
)

// A delegated child writes only inside what it declared it owns.
//
// The narrowing is the containment predicate the policy already consults, not a
// second check beside it. Ownership promised and unverified is the shape of
// defect this repository keeps finding in itself, so ownership is a boundary:
// the same machinery that refuses a write outside the workspace refuses a write
// outside the owned set.
func TestANarrowedResolverRefusesAWriteOutsideWhatIsOwned(t *testing.T) {
	ws := t.TempDir()
	r, err := NewResolver(ws)
	if err != nil {
		t.Fatal(err)
	}
	owned := r.Owning([]string{filepath.Join(ws, "docs")})

	inside, err := owned.Resolve(filepath.Join(ws, "docs", "a.md"), true)
	if err != nil {
		t.Fatal(err)
	}
	if !owned.InWorkspace(inside) {
		t.Error("a write inside the owned set must be contained")
	}

	outside, err := owned.Resolve(filepath.Join(ws, "src", "a.go"), true)
	if err != nil {
		t.Fatal(err)
	}
	if owned.InWorkspace(outside) {
		t.Error("a write inside the workspace but outside the owned set must not be contained")
	}
}

// Reading stays whole. A child that catalogues a repository has to read all of
// it and writes one file; narrowing reads as well would make the capability
// useless for the case it exists for.
func TestANarrowedResolverStillReadsTheWholeWorkspace(t *testing.T) {
	ws := t.TempDir()
	r, _ := NewResolver(ws)
	owned := r.Owning([]string{filepath.Join(ws, "docs")})

	read, err := owned.Resolve(filepath.Join(ws, "src", "a.go"), false)
	if err != nil {
		t.Fatal(err)
	}
	if !owned.InWorkspace(read) {
		t.Error("reading outside the owned set is still reading inside the workspace")
	}
}

// Narrowing never widens. An owned path outside the workspace is not an
// escape hatch: the workspace is still the outer boundary, and a child may only
// ever drop capability.
func TestOwningNeverReachesOutsideTheWorkspace(t *testing.T) {
	ws := t.TempDir()
	outsideDir := t.TempDir()
	r, _ := NewResolver(ws)
	owned := r.Owning([]string{outsideDir})

	a, err := owned.Resolve(filepath.Join(outsideDir, "a.md"), true)
	if err != nil {
		t.Fatal(err)
	}
	if owned.InWorkspace(a) {
		t.Error("owning a path outside the workspace must not put it inside")
	}
}

// Containment by path component, never by string prefix — the same bug the
// workspace boundary already refuses. /w/docs2 is not inside /w/docs.
func TestOwnershipIsByComponentNotByPrefix(t *testing.T) {
	ws := t.TempDir()
	r, _ := NewResolver(ws)
	owned := r.Owning([]string{filepath.Join(ws, "docs")})

	a, err := owned.Resolve(filepath.Join(ws, "docs2", "a.md"), true)
	if err != nil {
		t.Fatal(err)
	}
	if owned.InWorkspace(a) {
		t.Error("docs2 is not inside docs")
	}
}

// An empty set is not "everything". Nothing said is never a yes — the same
// reading a nil environment already gets from the sandbox.
func TestOwningNothingOwnsNothing(t *testing.T) {
	ws := t.TempDir()
	r, _ := NewResolver(ws)
	owned := r.Owning(nil)

	a, err := owned.Resolve(filepath.Join(ws, "a.md"), true)
	if err != nil {
		t.Fatal(err)
	}
	if owned.InWorkspace(a) {
		t.Error("owning nothing must not contain a write")
	}
}

// The original is untouched. A narrowed child must never be able to widen its
// parent, and sharing a mutable resolver would be exactly that.
func TestOwningLeavesTheParentResolverAlone(t *testing.T) {
	ws := t.TempDir()
	r, _ := NewResolver(ws)
	_ = r.Owning([]string{filepath.Join(ws, "docs")})

	a, err := r.Resolve(filepath.Join(ws, "src", "a.go"), true)
	if err != nil {
		t.Fatal(err)
	}
	if !r.InWorkspace(a) {
		t.Error("narrowing a child must not narrow the parent")
	}
}
