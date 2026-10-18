package orchestrate

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Spec: orchestration .config §3 — forge, ci_timeout, sensitive and
// sensitive_diff_lines, with their defaults; a value out of bounds is an error
// naming the key and the value, never the default in silence.
func TestMergeFromReadsTheKeysAndRefusesWhatDoesNotHold(t *testing.T) {
	c, err := MergeFrom(resolved(nil))
	if err != nil {
		t.Fatal(err)
	}
	if c.Forge != ForgeAuto || c.CITimeout != 45*time.Minute || c.SensitiveDiffLines != 800 ||
		strings.Join(c.Sensitive, ",") != strings.Join(defaultSensitive, ",") {
		t.Errorf("defaults = %+v", c)
	}
	c, err = MergeFrom(resolved(map[string]string{"orchestrator.forge": "local", "orchestrator.ci_timeout": "2h",
		"orchestrator.sensitive": " a/** , b.go ", "orchestrator.sensitive_diff_lines": "50"}))
	if err != nil || c.Forge != ForgeLocal || c.CITimeout != 2*time.Hour || strings.Join(c.Sensitive, ",") != "a/**,b.go" || c.SensitiveDiffLines != 50 {
		t.Errorf("declared = %+v, %v", c, err)
	}
	for k, v := range map[string]string{"orchestrator.forge": "gitlab", "orchestrator.ci_timeout": "30s",
		"orchestrator.sensitive_diff_lines": "49"} {
		if _, err := MergeFrom(resolved(map[string]string{k: v})); err == nil || !strings.Contains(err.Error(), k) || !strings.Contains(err.Error(), v) {
			t.Errorf("%s=%s: %v, want an error naming both", k, v, err)
		}
	}
}

// Spec: orchestration .config §3 — auto is github with an origin and a gh
// that answers, local otherwise; github declared without gh is an error,
// never a quiet fall to local.
func TestAutoIsGitHubWithAnOriginAndALoginAndLocalOtherwise(t *testing.T) {
	gh := fakeGh(t)
	withOrigin := newProject(t, true)
	setup := func(p mergeProject, declared string) ForgeSetup {
		in, err := OpenIntegration(context.Background(), p.root, p.dir)
		if err != nil {
			t.Fatal(err)
		}
		return ForgeSetup{Declared: declared, Root: p.root, Main: "main", Integration: in, Gate: Gate{Command: "true", Timeout: time.Minute}}
	}
	ctx := context.Background()
	if f, err := NewForge(ctx, setup(withOrigin, ForgeAuto)); err != nil || f.Name() != ForgeGitHub {
		t.Errorf("auto with origin and login = %v, %v", f, err)
	}
	without := newProject(t, false)
	if f, err := NewForge(ctx, setup(without, ForgeAuto)); err != nil || f.Name() != ForgeLocal {
		t.Errorf("auto without origin = %v, %v", f, err)
	}
	ghSays(t, gh, "logged-out", "")
	if f, err := NewForge(ctx, setup(withOrigin, ForgeAuto)); err != nil || f.Name() != ForgeLocal {
		t.Errorf("auto logged out = %v, %v", f, err)
	}
	if _, err := NewForge(ctx, setup(withOrigin, ForgeGitHub)); err == nil || !strings.Contains(err.Error(), "gh") {
		t.Errorf("github declared, logged out = %v, want an error about gh", err)
	}
	if err := os.Remove(filepath.Join(gh, "logged-out")); err != nil {
		t.Fatal(err)
	}
	s := setup(withOrigin, ForgeGitHub)
	t.Setenv("PATH", t.TempDir())
	if _, err := NewForge(ctx, s); err == nil {
		t.Error("github declared without gh on PATH was accepted")
	}
}

// Spec: orchestration .p §7.6 — main's health on the forge is what CI says of
// the commit: a failure is red, a run still going or none yet is pending,
// every run passed is green.
func TestMainHealthOnGitHubReadsTheRunsOfTheCommit(t *testing.T) {
	gh := fakeGh(t)
	p := newProject(t, true)
	f, err := NewForge(context.Background(), ForgeSetup{Declared: ForgeGitHub, Root: p.root, Main: "main"})
	if err != nil {
		t.Fatal(err)
	}
	sha := revParse(t, p.root, "main")
	for runs, want := range map[string]string{
		`[]`: "pending",
		`[{"status":"completed","conclusion":"success"},{"status":"completed","conclusion":"skipped"}]`:  "green",
		`[{"status":"completed","conclusion":"success"},{"status":"in_progress","conclusion":""}]`:       "pending",
		`[{"status":"in_progress","conclusion":""},{"status":"completed","conclusion":"failure"}]`:       "red",
		`[{"status":"completed","conclusion":"success"},{"status":"completed","conclusion":"whatever"}]`: "red",
	} {
		ghSays(t, gh, "runs.json", runs)
		h, err := f.MainHealth(context.Background(), sha)
		if err != nil || h.Health != want || h.Commit != sha {
			t.Errorf("runs %s = %+v, %v; want %s", runs, h, err, want)
		}
	}
	if !strings.Contains(ghCalls(t, gh), "run\nlist\n--commit\n"+sha+"\n") {
		t.Errorf("gh calls:\n%s", ghCalls(t, gh))
	}
}
