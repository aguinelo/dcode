package config

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// RN-5/RN-6: the instruction chain is frozen when the session is created.
//
// The rule is not that a late file is ignored — it becomes an appended notice,
// which is what OutOfChain is for. The rule is that it never reaches the chain
// the prefix was built from, because the prefix is what the provider caches and
// one changed byte makes every later turn pay for the whole prompt again.
//
// Freezing is a property of the CALLER — it walks once and keeps the answer —
// so what can be asserted here is the half that makes it possible: discovery is
// a pure walk with no memory, so calling it again is the only way the chain can
// change, and calling it again is a decision someone has to make on purpose.
func TestDiscoveryIsAWalkWithNoMemorySoTheChainCanOnlyChangeOnPurpose(t *testing.T) {
	ws := t.TempDir()
	sub := filepath.Join(ws, "internal", "api")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, "AGENTS.md"), []byte("ROOT-RULE"), 0o644); err != nil {
		t.Fatal(err)
	}

	frozen, err := DiscoverInstructions(ws, sub, []string{"AGENTS.md", "DCODE.md"}, 4096, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(frozen) != 1 {
		t.Fatalf("expected the one root file, got %d", len(frozen))
	}

	// A file appears mid-session, in a directory already in scope.
	if err := os.WriteFile(filepath.Join(sub, "AGENTS.md"), []byte("LATE-RULE"), 0o644); err != nil {
		t.Fatal(err)
	}

	// The value the session is holding is untouched: nothing watches the disk,
	// nothing re-walks behind the caller's back.
	if len(frozen) != 1 || frozen[0].Text != "ROOT-RULE" {
		t.Errorf("the chain captured at creation changed underneath: %+v", frozen)
	}

	// And the late file IS discoverable — by asking again, which is the
	// deliberate act. Nothing here is hiding it; it is being kept out of the
	// prefix, which is a different thing.
	again, err := DiscoverInstructions(ws, sub, []string{"AGENTS.md", "DCODE.md"}, 4096, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 2 {
		t.Fatalf("a fresh walk should see both files, got %d — then it is not discovery that froze, it is a bug", len(again))
	}
}

// Resolve answers from the layers it is handed and nothing else. If it read the
// environment or the disk, two calls with the same layers could disagree — and
// `--config`, whose whole job is to show a value WITH its origin, would be
// reporting an origin that did not produce the value.
func TestResolveIsPureOverTheLayersItIsGiven(t *testing.T) {
	layers := []Layer{
		{Source: SourceDefault, Origin: "built-in", Values: map[string]string{"model.name": "a", "limits.parallel": "4"}},
		{Source: SourceEnv, Origin: "environment", Values: map[string]string{"model.name": "b"}},
	}
	first := Resolve(layers)

	// The environment says something else entirely, and must not be consulted.
	t.Setenv("DCODE_MODEL_NAME", "from-the-environment")
	t.Setenv("DCODE_PARALLEL", "99")

	snapshot := func(r Resolved) map[string]Value {
		out := map[string]Value{}
		for _, k := range r.Keys() {
			v, _ := r.Get(k)
			out[k] = v
		}
		return out
	}
	want := snapshot(first)
	for i := 0; i < 20; i++ {
		if got := snapshot(Resolve(layers)); !reflect.DeepEqual(got, want) {
			t.Fatalf("run %d differs from the first; Resolve is reading something outside its argument", i)
		}
	}
	if v, _ := first.Get("model.name"); v.Value != "b" {
		t.Fatalf("model.name = %q, want the env LAYER's value rather than the process environment", v.Value)
	}

	// The layers themselves come back unmodified, or a caller that resolves
	// twice gets a different answer the second time.
	if layers[0].Values["model.name"] != "a" {
		t.Error("Resolve mutated the layer it was given")
	}
}

// RN-10: expansion is text substitution. A command file is user-authored and
// arrives from a repository that may have been cloned from anywhere, so
// "expand" must never become "run".
//
// The determinism test beside this one asserts that two calls agree, which a
// command shelling out to `date` would still satisfy on a fast enough machine.
// This is the half that cannot be satisfied by luck.
func TestExpansionCannotReachForExecutionOrTheDisk(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "commands.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	forbidden := map[string]bool{`"os/exec"`: true, `"syscall"`: true, `"net"`: true, `"net/http"`: true}
	for _, imp := range f.Imports {
		if forbidden[imp.Path.Value] {
			t.Errorf("commands.go imports %s; a command file comes from a repository that "+
				"may have been cloned from anywhere, and expanding it must never run it", imp.Path.Value)
		}
	}
}
