package tui

import (
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/config"
)

// `/loop` is recognised before the line becomes turn input.
//
// RN-3 of the loop-command family: the command text must never reach the
// history. Syntax in the prefix invalidates the cached prefix on every turn
// and spends tokens saying nothing to the model.
func TestLoopIsACommandAndNotTurnInput(t *testing.T) {
	got := ResolveInput("/loop specs/2026-08-25-home-page", config.CommandSet{})
	if got.Kind != CmdBuiltin {
		t.Fatalf("/loop resolved as kind %v, want a built-in", got.Kind)
	}
	if got.Name != "loop" {
		t.Fatalf("resolved name %q", got.Name)
	}
	if got.Text != "" {
		t.Errorf("the command text leaked into the turn: %q", got.Text)
	}
}

// A user command cannot shadow it, for the same reason none of the others can
// be shadowed: advice about dcode would stop being true of any installation.
func TestLoopCannotBeShadowed(t *testing.T) {
	user := config.CommandSet{Commands: map[string]config.Command{
		"loop": {Path: "somewhere.md", Body: "something else entirely"},
	}}
	if got := ResolveInput("/loop x", user); got.Kind != CmdBuiltin {
		t.Errorf("a user command shadowed /loop: %+v", got)
	}
}

func TestParseLoopArgs(t *testing.T) {
	for _, c := range []struct {
		name    string
		in      string
		spec    string
		protect []string
		wantErr string // the message, or "" for no error
	}{
		{name: "just a path", in: "specs/home-page", spec: "specs/home-page"},
		{name: "protect", in: "specs/x --protect **/*_test.go",
			spec: "specs/x", protect: []string{"**/*_test.go"}},
		{name: "protect with equals", in: "specs/x --protect=**/*_test.go",
			spec: "specs/x", protect: []string{"**/*_test.go"}},
		{name: "protect twice", in: "specs/x --protect a --protect b",
			spec: "specs/x", protect: []string{"a", "b"}},
		{name: "flag before path", in: "--protect a specs/x",
			spec: "specs/x", protect: []string{"a"}},
		{name: "nothing", in: "", wantErr: ""},
		{name: "unknown flag", in: "specs/x --max-iterations 3", wantErr: "--max-iterations"},
		{name: "protect with nothing after it", in: "specs/x --protect", wantErr: "--protect needs a glob"},
		{name: "two paths", in: "specs/x specs/y", wantErr: "specs/y"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseLoopArgs(c.in)
			if c.name == "nothing" || c.wantErr != "" {
				if err == nil {
					t.Fatalf("expected an error, got %+v", got)
				}
				if c.wantErr != "" && !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("error %q does not name %q", err, c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Spec != c.spec {
				t.Errorf("spec = %q, want %q", got.Spec, c.spec)
			}
			if len(got.Protect) != len(c.protect) {
				t.Fatalf("protect = %v, want %v", got.Protect, c.protect)
			}
			for i := range c.protect {
				if got.Protect[i] != c.protect[i] {
					t.Errorf("protect = %v, want %v", got.Protect, c.protect)
				}
			}
		})
	}
}

// A mistyped flag stops the command rather than being ignored.
//
// Ignoring it opens a session measured against a definition of done the person
// did not ask for, and they find out at the end of the turn.
func TestAMistypedFlagStopsTheCommand(t *testing.T) {
	if _, err := ParseLoopArgs("specs/x --protct a"); err == nil {
		t.Fatal("a mistyped flag was accepted")
	}
}

// It appears in /help and in the completion menu, or nobody discovers it.
func TestLoopIsDiscoverable(t *testing.T) {
	help := HelpText(config.CommandSet{}, En)
	if !strings.Contains(help, "/loop") {
		t.Errorf("/help does not list /loop:\n%s", help)
	}
	found := false
	for _, c := range Complete("/lo", config.CommandSet{}, En) {
		if c.Name == "loop" {
			found = true
		}
	}
	if !found {
		t.Error("/loop does not complete from /lo")
	}
}
