package update

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The release pipeline writes the digests into install.sh and carries the
// result back to main, so that the URL the README publishes serves an installer
// that can check what it downloads.
//
// Everything here is read as data or driven against a throwaway git repository.
// Nothing in `make check` runs the release workflow, so a step that stops doing
// its job stops silently — which is how the build stamp came to need
// TestTheReleasePipelineStampsEveryFieldTheBinaryReportsOn.

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// gitAt runs git in dir and fails the test on error.
func gitAt(t *testing.T, dir string, args ...string) string {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	c.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func needTools(t *testing.T, names ...string) {
	t.Helper()
	for _, n := range names {
		if _, err := exec.LookPath(n); err != nil {
			t.Skipf("no %s", n)
		}
	}
}

// -- scripts/version.sh -------------------------------------------------------

// versionIn runs scripts/version.sh inside a repository and returns stdout.
func versionIn(t *testing.T, dir string) (string, error) {
	t.Helper()
	c := exec.Command("bash", filepath.Join(repoRoot(t), "scripts", "version.sh"))
	c.Dir = dir
	out, err := c.Output()
	return strings.TrimSpace(string(out)), err
}

// seededRepo returns a repository with one tagged commit.
func seededRepo(t *testing.T, tag string) string {
	t.Helper()
	needTools(t, "git", "bash")
	dir := t.TempDir()
	gitAt(t, dir, "init", "--quiet", "--initial-branch=main")
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitAt(t, dir, "add", "f")
	gitAt(t, dir, "commit", "--quiet", "-m", "feat: the beginning")
	gitAt(t, dir, "tag", "-a", tag, "-m", tag)
	return dir
}

func commitAt(t *testing.T, dir, subject, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte(subject+body), 0o644); err != nil {
		t.Fatal(err)
	}
	gitAt(t, dir, "add", "f")
	gitAt(t, dir, "commit", "--quiet", "-m", subject)
}

// The pin commit lands on main AFTER the tag, because the digests are not known
// until the artifacts are built. Counting it would make every post-release
// query answer "there are commits since the tag" when nothing human changed,
// and the derivation would start raising PATCH on its own.
//
// That is the shape this repository keeps finding: automation leaving a trace
// that another mechanism reads as a signal.
func TestTheVersionIgnoresThePipelinesOwnPinCommit(t *testing.T) {
	dir := seededRepo(t, "v0.1.0")
	commitAt(t, dir, "chore(release): pin the installer to v0.1.0", "")

	got, err := versionIn(t, dir)
	if err != nil {
		t.Fatalf("version.sh failed: %v", err)
	}
	if got != "v0.1.0" {
		t.Errorf("the pipeline's own pin commit was counted as a change: got %q, want v0.1.0", got)
	}
}

// And it must not become a way to hide work: a real commit alongside the pin
// still moves the version.
func TestTheVersionStillCountsRealWorkBesideAPinCommit(t *testing.T) {
	dir := seededRepo(t, "v0.1.0")
	commitAt(t, dir, "chore(release): pin the installer to v0.1.0", "")
	commitAt(t, dir, "fix: something that actually broke", "")

	got, err := versionIn(t, dir)
	if err != nil {
		t.Fatalf("version.sh failed: %v", err)
	}
	if got != "v0.1.1" {
		t.Errorf("real work next to a pin commit did not move the version: got %q, want v0.1.1", got)
	}
}

// The exemption is for the exact subject the pipeline writes, not for the
// prefix. `chore(release): anything else` is a human commit and counts.
func TestTheVersionOnlyExemptsTheExactSubjectThePipelineWrites(t *testing.T) {
	dir := seededRepo(t, "v0.1.0")
	commitAt(t, dir, "chore(release): tidy up the workflow", "")

	got, err := versionIn(t, dir)
	if err != nil {
		t.Fatalf("version.sh failed: %v", err)
	}
	if got != "v0.1.1" {
		t.Errorf("a human chore(release) commit was silently exempted: got %q, want v0.1.1", got)
	}
}

// -- scripts/publish-installer.sh ---------------------------------------------

// bareRepo stands in for the repository on GitHub: clonable, pushable, with
// main already carrying an unpinned install.sh.
func bareRepo(t *testing.T) (bare, seedDir string) {
	t.Helper()
	needTools(t, "git", "bash")
	bare = filepath.Join(t.TempDir(), "origin.git")
	gitAt(t, t.TempDir(), "init", "--bare", "--initial-branch=main", bare)

	seedDir = t.TempDir()
	gitAt(t, seedDir, "clone", "--quiet", bare, ".")
	if err := os.WriteFile(filepath.Join(seedDir, "install.sh"),
		[]byte("#!/bin/sh\nPINNED_VERSION=\"\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	gitAt(t, seedDir, "add", "install.sh")
	gitAt(t, seedDir, "commit", "--quiet", "-m", "feat: the installer")
	gitAt(t, seedDir, "push", "--quiet", "origin", "main")
	return bare, seedDir
}

func publishInstaller(t *testing.T, remote, version, path string) (string, error) {
	t.Helper()
	c := exec.Command("bash",
		filepath.Join(repoRoot(t), "scripts", "publish-installer.sh"), version, path)
	c.Env = append(os.Environ(), "GH_TOKEN=", "INSTALLER_REMOTE="+remote,
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	out, err := c.CombinedOutput()
	return string(out), err
}

// pinnedFile writes an installer carrying a version, standing in for the output
// of scripts/installer.sh.
func pinnedFile(t *testing.T, version string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "install.sh")
	body := "#!/bin/sh\nPINNED_VERSION=\"" + version + "\"\n"
	if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// The point of the whole change: the URL the README publishes has to serve an
// installer that carries digests. Publishing it only as a release asset leaves
// `main/install.sh` — the one people actually run — unable to check anything by
// a second route.
func TestThePinnedInstallerReachesTheBranchTheReadmePublishes(t *testing.T) {
	bare, _ := bareRepo(t)

	out, err := publishInstaller(t, bare, "0.2.0", pinnedFile(t, "0.2.0"))
	if err != nil {
		t.Fatalf("publishing the pinned installer failed: %v\n%s", err, out)
	}

	check := t.TempDir()
	gitAt(t, check, "clone", "--quiet", bare, ".")
	got, err := os.ReadFile(filepath.Join(check, "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `PINNED_VERSION="0.2.0"`) {
		t.Errorf("main does not carry the pinned installer:\n%s", got)
	}
	subject := gitAt(t, check, "log", "-1", "--format=%s")
	if strings.TrimSpace(subject) != "chore(release): pin the installer to v0.2.0" {
		t.Errorf("the pin commit subject is %q, which scripts/version.sh will then count", strings.TrimSpace(subject))
	}
}

// This runs after the release is public and the pinned installer is already
// attached to it. Failing here does not un-publish anything — it paints a
// successful release red, the one report nobody can act on. Same reasoning as
// publish-tap.sh, and the same behaviour.
func TestAnUnreachableRepositoryDoesNotRedenAPublishedRelease(t *testing.T) {
	needTools(t, "git", "bash")
	out, err := publishInstaller(t, filepath.Join(t.TempDir(), "nowhere.git"),
		"0.2.0", pinnedFile(t, "0.2.0"))
	if err != nil {
		t.Fatalf("an unreachable repository reddened a published release: %v\n%s", err, out)
	}
	if !strings.Contains(out, "warning") {
		t.Errorf("the failure was swallowed rather than reported:\n%s", out)
	}
}

// The one condition that is NOT recoverable. Pushing on without the pinned file
// would leave main carrying the PREVIOUS release's digests while the release
// reports success — an installer that silently falls back for everyone, with
// nothing anywhere saying so.
func TestAMissingPinnedInstallerStopsRatherThanLeavingMainStale(t *testing.T) {
	needTools(t, "git", "bash")
	bare, _ := bareRepo(t)

	out, err := publishInstaller(t, bare, "0.2.0", filepath.Join(t.TempDir(), "absent.sh"))
	if err == nil {
		t.Fatalf("a missing pinned installer was treated as success:\n%s", out)
	}
}

// Re-running a release workflow must not redden it. `git commit` exits non-zero
// with nothing staged, and the second-order harm is worse than the first:
// someone re-runs the workflow for another reason and reads this red as theirs.
func TestPublishingTheSameInstallerTwiceIsNotAFailure(t *testing.T) {
	bare, _ := bareRepo(t)
	pinned := pinnedFile(t, "0.2.0")

	if out, err := publishInstaller(t, bare, "0.2.0", pinned); err != nil {
		t.Fatalf("first publish failed: %v\n%s", err, out)
	}
	out, err := publishInstaller(t, bare, "0.2.0", pinned)
	if err != nil {
		t.Fatalf("re-running the release reddened it: %v\n%s", err, out)
	}
	if !strings.Contains(out, "unchanged") {
		t.Errorf("the no-op was not reported as one:\n%s", out)
	}
}

// -- .github/workflows/release.yml --------------------------------------------

// The workflow is read as data, for the reason the build-stamp test already
// records: nothing in `make check` runs it, so a step that stops doing its job
// stops in silence and the discovery is made by users.
//
// Order is the substance here. The digests must be taken from a checksums.txt
// that has already been signed and verified — pinning before that would carry
// digests nobody vouched for, which is the failure the whole feature exists to
// prevent, reproduced inside the pipeline that implements it.
func TestTheReleasePinsTheInstallerFromChecksumsItAlreadyVerified(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	yaml := string(data)

	for _, want := range []string{
		"scripts/installer.sh",
		"scripts/publish-installer.sh",
	} {
		if !strings.Contains(yaml, want) {
			t.Errorf("release.yml never runs %s, so the block it fills stays empty", want)
		}
	}

	verify := strings.Index(yaml, "cosign verify-blob")
	pin := strings.Index(yaml, "scripts/installer.sh")
	publish := strings.Index(yaml, "gh release create")
	carry := strings.Index(yaml, "scripts/publish-installer.sh")

	if verify < 0 || pin < 0 || publish < 0 || carry < 0 {
		t.Fatal("release.yml is missing one of verify, pin, publish or carry")
	}
	if pin < verify {
		t.Error("the installer is pinned before the checksums are verified, so it would " +
			"carry digests nobody vouched for")
	}
	if publish < pin {
		t.Error("the release is published before the installer is pinned, so the asset " +
			"would be the unpinned one")
	}
	if carry < publish {
		t.Error("main is updated before the release exists; a failure there would then " +
			"redden a release that never published")
	}
}
