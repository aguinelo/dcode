package config

import (
	"os"
	"path/filepath"
	"testing"
)

// A grant is a decision the user made once and should not be asked again.
// Nothing remembered it, so the only two states the product could offer were
// "network never works" and "answer the same question every command".

func TestAGrantForOneProjectDoesNotCoverAnother(t *testing.T) {
	root := t.TempDir()
	g, err := LoadGrants(root)
	if err != nil {
		t.Fatal(err)
	}
	if g.Network("/home/ada/proj") {
		t.Fatal("a fresh install already granted the network")
	}

	g = g.GrantNetwork("/home/ada/proj")
	if err := g.Save(root); err != nil {
		t.Fatal(err)
	}

	back, err := LoadGrants(root)
	if err != nil {
		t.Fatal(err)
	}
	if !back.Network("/home/ada/proj") {
		t.Error("the answer did not survive; the user would be asked again")
	}
	if back.Network("/home/ada/other") {
		t.Error("a grant for one project covered another — the question was about this one")
	}
}

// "Always" is a different answer from "this project", and the difference is the
// whole point of asking two questions instead of one.
func TestGrantingAlwaysCoversEveryProjectIncludingNewOnes(t *testing.T) {
	root := t.TempDir()
	g, _ := LoadGrants(root)
	g = g.GrantNetworkAlways()
	if err := g.Save(root); err != nil {
		t.Fatal(err)
	}

	back, _ := LoadGrants(root)
	for _, ws := range []string{"/home/ada/proj", "/tmp/cloned-yesterday", "/anything"} {
		if !back.Network(ws) {
			t.Errorf("always did not cover %s", ws)
		}
	}
}

// The workspace is identified by its resolved path. Two spellings of one
// directory are one project, or the user answers again for the same place.
func TestTwoSpellingsOfOneProjectAreOneGrant(t *testing.T) {
	root := t.TempDir()
	ws := t.TempDir()
	g, _ := LoadGrants(root)
	g = g.GrantNetwork(ws)

	if !g.Network(filepath.Join(ws, "sub", "..")) {
		t.Error("the same directory spelled differently was treated as another project")
	}
	if !g.Network(ws + string(filepath.Separator)) {
		t.Error("a trailing separator made it another project")
	}
}

// The record of what the user permitted is written where credentials are:
// owner-only. It is a security decision, and a decision anyone on the machine
// can edit is a decision the user did not make.
func TestTheGrantFileIsOwnerOnly(t *testing.T) {
	root := t.TempDir()
	g, _ := LoadGrants(root)
	if err := g.GrantNetwork("/w").Save(root); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(root, GrantsFile))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %o, want 600", perm)
	}
}

// A grants file that cannot be parsed must not be read as "everything is
// permitted". Failing closed is the only safe reading of a record nobody can
// interpret.
func TestAnUnreadableGrantFileGrantsNothing(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, GrantsFile), []byte("this is not toml ]["), 0o600); err != nil {
		t.Fatal(err)
	}
	g, err := LoadGrants(root)
	if err == nil {
		t.Error("a corrupt grants file was accepted in silence")
	}
	if g.Network("/anything") {
		t.Error("an unreadable record was read as permission")
	}
}

// Absent is the ordinary case on a fresh machine, and it is not an error.
func TestNoGrantFileIsNotAFailure(t *testing.T) {
	g, err := LoadGrants(t.TempDir())
	if err != nil {
		t.Fatalf("a fresh install failed to start: %v", err)
	}
	if g.Network("/w") {
		t.Error("a fresh install granted the network")
	}
}
