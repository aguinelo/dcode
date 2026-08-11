package update

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The tap is published AFTER the release exists. That ordering decides how every
// failure here must behave: by the time this runs, the binaries are signed, the
// release is public and the formula is attached to it and installable by URL.
// Failing the job at that point does not un-publish anything — it paints a
// successful release red, which is the one report nobody can act on.
//
// So every recoverable condition exits zero and says so loudly. What must never
// happen is the opposite: exiting zero having silently done nothing.

func tap(t *testing.T, env map[string]string, args ...string) (string, error) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("no bash")
	}
	cmd := exec.Command("bash", append([]string{filepath.Join("..", "..", "scripts", "publish-tap.sh")}, args...)...)
	cmd.Env = append(os.Environ(), "TAP_TOKEN=")
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// formulaAt writes a formula file and returns its path.
func formulaAt(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, "dcode.rb")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// bareTap is a git repository that can be cloned and pushed to, standing in for
// the tap on GitHub.
func bareTap(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "--bare", "--initial-branch=main", ".")
	return dir
}

// Without the secret there is no tap to push to, and that is a supported
// configuration rather than a failure: the formula is on the release and
// installable by URL. One channel fewer, never a broken release.
func TestNoTokenIsASupportedConfigurationAndSaysSo(t *testing.T) {
	dir := t.TempDir()
	out, err := tap(t, nil, "0.4.0", formulaAt(t, dir, "class Dcode\nend\n"))
	if err != nil {
		t.Fatalf("a missing token failed the release: %v\n%s", err, out)
	}
	if !strings.Contains(out, "TAP_TOKEN") {
		t.Errorf("it exited quietly; a channel that did not publish must say why:\n%s", out)
	}
}

// The condition this replaces was `if: env.TAP_TOKEN != ”` on the step that
// defines TAP_TOKEN in its own env block. Whether a step's own env is in scope
// for its own `if` is a subtlety I could not settle from the documentation, and
// the failure mode of guessing wrong is silent: the tap simply never updates,
// and the release stays green. Deciding it in the script removes the question.
func TestTheTokenDecidesInsideTheScriptWhereItCanBeTested(t *testing.T) {
	dir := t.TempDir()
	remote := bareTap(t)
	out, err := tap(t, map[string]string{"TAP_TOKEN": "x", "TAP_REMOTE": remote},
		"0.4.0", formulaAt(t, dir, "class Dcode\n  version \"0.4.0\"\nend\n"))
	if err != nil {
		t.Fatalf("publishing to a reachable tap failed: %v\n%s", err, out)
	}

	// The formula really landed, on the branch a clone would read.
	show := exec.Command("git", "--git-dir", remote, "show", "main:Formula/dcode.rb")
	got, err := show.Output()
	if err != nil {
		t.Fatalf("nothing was committed to the tap: %v", err)
	}
	if !strings.Contains(string(got), `version "0.4.0"`) {
		t.Errorf("the tap carries something else:\n%s", got)
	}
}

// A tap that was never created is a misconfiguration, not a reason to redden a
// release that already succeeded. It warns where a warning is visible on the
// run, and leaves the job green.
func TestAnUnreachableTapWarnsAndLeavesTheReleaseGreen(t *testing.T) {
	dir := t.TempDir()
	out, err := tap(t, map[string]string{
		"TAP_TOKEN": "x", "TAP_REMOTE": filepath.Join(t.TempDir(), "does-not-exist"),
	}, "0.4.0", formulaAt(t, dir, "class Dcode\nend\n"))
	if err != nil {
		t.Fatalf("an absent tap failed the release after it was published: %v\n%s", err, out)
	}
	if !strings.Contains(out, "::warning") {
		t.Errorf("the failure was not surfaced as a warning on the run:\n%s", out)
	}
}

// Re-running a release with an identical formula must not fail. `git commit`
// exits non-zero with nothing staged, so the naive form turns a harmless re-run
// into a red job — and the second half of that is worse: someone re-runs to fix
// something else and reads this red as their fault.
func TestRepublishingAnIdenticalFormulaIsNotAFailure(t *testing.T) {
	dir := t.TempDir()
	remote := bareTap(t)
	body := "class Dcode\n  version \"0.4.0\"\nend\n"

	if out, err := tap(t, map[string]string{"TAP_TOKEN": "x", "TAP_REMOTE": remote},
		"0.4.0", formulaAt(t, dir, body)); err != nil {
		t.Fatalf("first publish: %v\n%s", err, out)
	}
	out, err := tap(t, map[string]string{"TAP_TOKEN": "x", "TAP_REMOTE": remote},
		"0.4.0", formulaAt(t, dir, body))
	if err != nil {
		t.Fatalf("republishing the same formula failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "unchanged") {
		t.Errorf("it should say nothing needed doing:\n%s", out)
	}
}

// The one condition that is NOT recoverable. Publishing a formula that is not
// there would leave the tap pointing at whatever it held before, while the
// release reports success — an installer serving the previous version to
// everyone who runs `brew upgrade`.
func TestAMissingFormulaIsTheOneFailureThatStops(t *testing.T) {
	out, err := tap(t, map[string]string{"TAP_TOKEN": "x", "TAP_REMOTE": bareTap(t)},
		"0.4.0", filepath.Join(t.TempDir(), "absent.rb"))
	if err == nil {
		t.Fatalf("a missing formula was published as success:\n%s", out)
	}
}
