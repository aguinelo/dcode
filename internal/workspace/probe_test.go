package workspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func at(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func names(gs []Gate) []string {
	out := make([]string, 0, len(gs))
	for _, g := range gs {
		out = append(out, g.Name)
	}
	return out
}

// The scripts a project declares are what it measures itself by, and the
// prefix carries them so the agent does not have to go and read package.json
// to find out the project has a coverage gate.
func TestPackageScriptsBecomeGates(t *testing.T) {
	dir := at(t, map[string]string{
		"package.json": `{"scripts":{"test":"vitest run","lint":"next lint","test:coverage":"vitest run --coverage"}}`,
	})
	got := Probe(context.Background(), dir)
	if len(got) != 3 {
		t.Fatalf("got %d gates: %+v", len(got), got)
	}
	// Sorted, because Go randomises map iteration and a prefix that reshuffles
	// between runs invalidates the provider cache for nothing.
	want := []string{"lint", "test", "test:coverage"}
	for i, n := range want {
		if got[i].Name != n {
			t.Fatalf("gate %d is %q, want %q — the order is not sorted", i, got[i].Name, n)
		}
	}
	if got[0].Command != "next lint" || got[0].Source != "package.json" {
		t.Errorf("command or source wrong: %+v", got[0])
	}
}

// The same tree twice must produce the same list, or the prefix changes
// between sessions for no reason at all.
func TestProbeIsStable(t *testing.T) {
	dir := at(t, map[string]string{
		"package.json": `{"scripts":{"a":"1","b":"2","c":"3","d":"4","e":"5","f":"6"}}`,
	})
	first := names(Probe(context.Background(), dir))
	for i := 0; i < 10; i++ {
		got := names(Probe(context.Background(), dir))
		for j := range first {
			if got[j] != first[j] {
				t.Fatalf("run %d disagreed: %v vs %v", i, got, first)
			}
		}
	}
}

// Makefile targets are gates too, in the order the file declares them: a
// Makefile is written to be read top to bottom and the first target is the
// default one.
func TestMakefileTargetsBecomeGates(t *testing.T) {
	dir := at(t, map[string]string{
		"Makefile": "" +
			"BIN = bin/dcode\n" +
			".PHONY: check test\n" +
			"check: test lint\n\t@echo checking\n" +
			"test:\n\tgo test ./...\n" +
			"# a comment\n" +
			"lint:\n\tgolangci-lint run\n",
	})
	got := names(Probe(context.Background(), dir))
	want := []string{"check", "test", "lint"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v — order of appearance is not preserved", got, want)
		}
	}
}

// Everything a Makefile carries that is not a thing a person runs.
func TestMakefileNoiseIsNotAGate(t *testing.T) {
	dir := at(t, map[string]string{
		"Makefile": "" +
			".PHONY: all\n" +
			".DEFAULT_GOAL := real\n" +
			"VERSION := 1.0\n" +
			"FLAGS = -race\n" +
			"%.o: %.c\n\tcc $<\n" +
			"$(BIN): main.go\n\tgo build\n" +
			"real:\n\techo yes\n",
	})
	got := names(Probe(context.Background(), dir))
	if len(got) != 1 || got[0] != "real" {
		t.Fatalf("got %v, want only [real]", got)
	}
}

// A directory that declares nothing gets nothing. Unlike a missing repository,
// this is ordinary and inconsequential, and the prefix must simply not mention
// gates rather than claim there are none.
func TestNothingDeclaredIsNoGates(t *testing.T) {
	if got := Probe(context.Background(), t.TempDir()); got != nil {
		t.Errorf("got %+v, want nothing", got)
	}
}

// Malformed package.json is the project's problem, not a reason to fail a
// session. It reads as "declared nothing", which is what an unreadable
// declaration honestly is.
func TestMalformedPackageJSONIsNotAnError(t *testing.T) {
	dir := at(t, map[string]string{"package.json": "{not json at all"})
	if got := Probe(context.Background(), dir); got != nil {
		t.Errorf("got %+v, want nothing", got)
	}
}

// A cancelled probe reads nothing rather than reporting that nothing was
// declared. Same distinction the repository snapshot had to make: "did not
// look" and "looked and there is none" are different facts.
func TestACancelledProbeReadsNothing(t *testing.T) {
	dir := at(t, map[string]string{"package.json": `{"scripts":{"test":"x"}}`})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := Probe(ctx, dir); got != nil {
		t.Errorf("a cancelled probe returned %+v", got)
	}
}

// A very long script is named, not reproduced. Nothing here truncates without
// the caller being able to see it did.
func TestALongCommandIsCut(t *testing.T) {
	long := ""
	for i := 0; i < 400; i++ {
		long += "x"
	}
	dir := at(t, map[string]string{"package.json": `{"scripts":{"big":"` + long + `"}}`})
	got := Probe(context.Background(), dir)
	if len(got) != 1 {
		t.Fatal("expected one gate")
	}
	if len(got[0].Command) > maxCommand+4 {
		t.Errorf("the command is %d bytes, uncut", len(got[0].Command))
	}
}

// Both sources at once, and neither erases the other.
func TestBothSourcesAreRead(t *testing.T) {
	dir := at(t, map[string]string{
		"package.json": `{"scripts":{"test":"vitest"}}`,
		"Makefile":     "check:\n\techo x\n",
	})
	got := Probe(context.Background(), dir)
	if len(got) != 2 {
		t.Fatalf("got %+v, want one from each source", got)
	}
	if got[0].Source != "package.json" || got[1].Source != "Makefile" {
		t.Errorf("sources wrong: %+v", got)
	}
}
