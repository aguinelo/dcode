package orchestrate

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/config"
)

func resolved(values map[string]string) config.Resolved {
	return config.Resolve([]config.Layer{{Source: config.SourceUser, Origin: "test", Values: values}})
}

// Spec: orchestration .config §1 — orchestrator.dir, absolute, by default
// <state>/orchestrate.
func TestTheDirectoryIsTheStateRootsUnlessDeclared(t *testing.T) {
	state := t.TempDir()
	env := func(k string) string {
		if k == "DCODE_STATE_DIR" {
			return state
		}
		if k == "HOME" {
			return t.TempDir()
		}
		return ""
	}

	got, err := Dir(resolved(nil), env)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(state, "orchestrate"); got != want {
		t.Errorf("default = %q, want %q", got, want)
	}

	declared := t.TempDir()
	got, err = Dir(resolved(map[string]string{"orchestrator.dir": declared + "/"}), env)
	if err != nil {
		t.Fatal(err)
	}
	if got != declared {
		t.Errorf("declared = %q, want %q", got, declared)
	}
}

func TestARelativeDirectoryIsAnErrorThatNamesTheKey(t *testing.T) {
	_, err := Dir(resolved(map[string]string{"orchestrator.dir": "orchestrate"}), func(string) string { return "" })
	if err == nil || !strings.Contains(err.Error(), "orchestrator.dir") || !strings.Contains(err.Error(), `"orchestrate"`) {
		t.Fatalf("err = %v, want one naming the key and the value", err)
	}
}

func TestNoStateRootIsAnErrorRatherThanADirectory(t *testing.T) {
	if _, err := Dir(resolved(nil), func(string) string { return "" }); err == nil {
		t.Fatal("a directory was made up without HOME or a state root")
	}
}
