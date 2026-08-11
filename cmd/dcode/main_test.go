package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/config"
)

// capture runs f with stdout and stderr redirected, and returns what they got.
//
// The command layer writes to the process streams rather than to an injected
// writer, and that is a reasonable shape for a main package — but it is not a
// reason for 1,116 lines to have no tests, which is what the coverage
// exclusion's justification ("wiring, no logic") had been taken to mean.
func capture(t *testing.T, f func()) (stdout, stderr string) {
	t.Helper()
	ro, wo, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	re, we, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = wo, we
	done := make(chan [2]string, 1)
	go func() {
		o, _ := io.ReadAll(ro)
		e, _ := io.ReadAll(re)
		done <- [2]string{string(o), string(e)}
	}()

	f()

	wo.Close()
	we.Close()
	os.Stdout, os.Stderr = oldOut, oldErr
	got := <-done
	return got[0], got[1]
}

func TestResolveWorkspaceIsAlwaysAbsolute(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	got, err := resolveWorkspace("")
	if err != nil {
		t.Fatal(err)
	}
	if got != cwd {
		t.Errorf("empty flag = %q, want the working directory %q", got, cwd)
	}

	rel, err := resolveWorkspace("..")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(rel) {
		t.Errorf("a relative flag stayed relative: %q", rel)
	}
	if rel != filepath.Dir(cwd) {
		t.Errorf("got %q, want %q", rel, filepath.Dir(cwd))
	}
}

// The key someone is looking for here is exactly the one with no value set, so
// an unset key lists the whole surface rather than saying nothing.
func TestAnUnsetKeyListsWhatCanBeSet(t *testing.T) {
	out, _ := capture(t, func() {
		if err := printConfig(config.Resolved{}, "model.name"); err != nil {
			t.Error(err)
		}
	})
	if !strings.Contains(out, "is not set") {
		t.Errorf("output does not say the key is unset:\n%s", out)
	}
	for _, want := range []string{"model.name", "sandbox.mode", "verify.command"} {
		if !strings.Contains(out, want) {
			t.Errorf("the list of known keys is missing %q", want)
		}
	}
}

func TestAResolvedKeyReportsItsValueAndOrigin(t *testing.T) {
	r := config.Resolve([]config.Layer{{
		Source: config.SourceUser, Origin: "/home/ada/config.toml",
		Values: map[string]string{"model.name": "MiniMax-M3"},
	}})
	out, _ := capture(t, func() {
		if err := printConfig(r, "model.name"); err != nil {
			t.Error(err)
		}
	})
	for _, want := range []string{"model.name = MiniMax-M3", "user", "/home/ada/config.toml"} {
		if !strings.Contains(out, want) {
			t.Errorf("output is missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "locked") {
		t.Error("an ordinary value was reported as locked")
	}
}

// A locked value that a user tried to change must say so where they look for
// it. Silence there is what sends someone to spend an afternoon on it.
func TestALockedKeySaysWhoWon(t *testing.T) {
	r := config.Resolve([]config.Layer{
		{Source: config.SourceUser, Origin: "config.toml",
			Values: map[string]string{"sandbox.mode": "full-access"}},
		{Source: config.SourceLocked, Origin: "requirements.toml",
			Values: map[string]string{"sandbox.mode": "read-only"}},
	})
	out, _ := capture(t, func() {
		if err := printConfig(r, "sandbox.mode"); err != nil {
			t.Error(err)
		}
	})
	if !strings.Contains(out, "read-only") {
		t.Errorf("the locked value did not win:\n%s", out)
	}
	if !strings.Contains(out, "locked by the administrator") {
		t.Errorf("the output does not say why the user's value lost:\n%s", out)
	}
}

func TestVersionAndHelpNeedNoConfiguration(t *testing.T) {
	// Both run before anything is resolved, which is the point: a broken
	// configuration must not stop someone finding out what they are running.
	out, _ := capture(t, func() {
		if err := dispatch([]string{"version"}); err != nil {
			t.Error(err)
		}
	})
	if !strings.Contains(out, "dcode") {
		t.Errorf("version said %q", out)
	}

	_, errOut := capture(t, func() {
		if err := dispatch([]string{"help"}); err != nil {
			t.Error(err)
		}
	})
	if !strings.Contains(errOut, "Usage:") && !strings.Contains(errOut, "Uso:") {
		t.Errorf("help said %q", errOut)
	}
}

func TestUsageListsEverySubcommandDispatchAccepts(t *testing.T) {
	_, errOut := capture(t, usage)
	// A subcommand that dispatch handles and usage never mentions is a feature
	// nobody can find.
	for _, sub := range []string{"serve", "tui", "update", "login", "config"} {
		if !strings.Contains(errOut, "dcode "+sub) {
			t.Errorf("usage does not list %q:\n%s", sub, errOut)
		}
	}
}

func TestShortTakesTheLastSegment(t *testing.T) {
	for in, want := range map[string]string{
		"sandbox.approval_policy": "approval_policy",
		"model.name":              "name",
		"bare":                    "bare",
		"":                        "",
	} {
		if got := short(in); got != want {
			t.Errorf("short(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestOriginNamesTheSourceAndFlagsALock(t *testing.T) {
	plain := origin(config.Value{Source: config.SourceEnv, Origin: "environment"})
	if plain != "env (environment)" {
		t.Errorf("got %q", plain)
	}
	locked := origin(config.Value{Source: config.SourceLocked, Origin: "requirements.toml", Locked: true})
	if !strings.Contains(locked, "locked") {
		t.Errorf("a locked value did not say so: %q", locked)
	}
}

func TestMax(t *testing.T) {
	if max(3, 7) != 7 || max(7, 3) != 7 || max(4, 4) != 4 {
		t.Fatal("max is wrong")
	}
}
