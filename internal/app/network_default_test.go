package app

import (
	"testing"
)

// homeOnly gives the chain a configuration root and nothing else, so what is
// asserted below is the built-in default rather than whatever this machine has.
func homeOnly(dir string) func(string) string {
	return func(k string) string {
		if k == "DCODE_HOME" {
			return dir
		}
		return ""
	}
}

// Ordinary shell work does not ask, and that is a default rather than something
// each person has to discover and set.
//
// It was false, and the consequence was written nowhere: bash declares the
// network on every call, so the declaration alone was a question, and wherever
// nobody could answer it the whole shell was denied. An unattended run measured
// 120 tool calls and zero commands executed — it could edit and never verify.
func TestReachingOutIsGrantedByDefault(t *testing.T) {
	opts, _, err := FromEnv(homeOnly(t.TempDir()), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !opts.AllowNetwork {
		t.Error("a fresh install asks before every build, test and commit")
	}
}

// And the switch works in the direction that costs something. A default nobody
// can undo is not a default, it is a decision taken for them.
func TestConfigurationCanTakeTheGrantBack(t *testing.T) {
	home := t.TempDir()
	env := func(k string) string {
		if k == "DCODE_ALLOW_NETWORK" {
			return "false"
		}
		return homeOnly(home)(k)
	}
	opts, _, err := FromEnv(env, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if opts.AllowNetwork {
		t.Error("withholding the grant was ignored")
	}
}
