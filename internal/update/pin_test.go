package update

import (
	"bytes"
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

// commitFileAt commits to a named file, so two branches can be merged without
// colliding on the one file seededRepo writes.
func commitFileAt(t *testing.T, dir, name, subject string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(subject), 0o644); err != nil {
		t.Fatal(err)
	}
	gitAt(t, dir, "add", name)
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

// A merge commit's subject is not a change. The changes are its parents', and
// they are already in the range. Counting the subject stopped the derivation
// dead on a sentence no tool will ever write conventionally — "Merge branch X
// into Y" — and the whole release was blocked by the merge that assembled it.
func TestTheVersionDoesNotReadAMergeCommitsSubject(t *testing.T) {
	dir := seededRepo(t, "v0.1.0")
	gitAt(t, dir, "checkout", "--quiet", "-b", "side")
	commitFileAt(t, dir, "side.txt", "feat: something on the side")
	gitAt(t, dir, "checkout", "--quiet", "main")
	commitFileAt(t, dir, "main.txt", "fix: something on main")
	gitAt(t, dir, "merge", "--quiet", "--no-ff", "-m", "Merge branch 'side'", "side")

	got, err := versionIn(t, dir)
	if err != nil {
		t.Fatalf("version.sh failed: %v", err)
	}
	// The feat came in through the merge, so it counts — through its own commit.
	if got != "v0.2.0" {
		t.Errorf("got %q, want v0.2.0", got)
	}
}

// A revert is classified by what it undid. Removing a feature is a behaviour
// change of the same class the addition was — VERSIONING.md says removal is at
// least MINOR — so `Revert "feat: …"` raises MINOR, and the release after a
// revert is derivable instead of refused.
func TestTheVersionClassifiesARevertByWhatItUndid(t *testing.T) {
	dir := seededRepo(t, "v0.1.0")
	commitAt(t, dir, "feat: the thing", "")
	gitAt(t, dir, "revert", "--no-edit", "HEAD")

	got, err := versionIn(t, dir)
	if err != nil {
		t.Fatalf("version.sh failed: %v", err)
	}
	if got != "v0.2.0" {
		t.Errorf("got %q, want v0.2.0", got)
	}
}

// One turn only. A revert of a revert leaves `Revert "…"` still wrapped, which
// matches nothing — and matching nothing is refusing, which is what this script
// does with everything it cannot read.
func TestTheVersionRefusesADoubleRevert(t *testing.T) {
	dir := seededRepo(t, "v0.1.0")
	commitAt(t, dir, "feat: the thing", "")
	gitAt(t, dir, "revert", "--no-edit", "HEAD")
	gitAt(t, dir, "revert", "--no-edit", "HEAD")

	if _, err := versionIn(t, dir); err == nil {
		t.Error("a double revert was given a version instead of a refusal")
	}
}

// The refusal exists to be read by whoever will fix it. Unquoted, it split on
// whitespace: a seven-word subject printed as seven lines, and the one message
// that had to be legible was the one that was not.
func TestTheRefusalNamesOneCommitPerLine(t *testing.T) {
	dir := seededRepo(t, "v0.1.0")
	commitAt(t, dir, "a subject with several words and no type", "")

	c := exec.Command("bash", filepath.Join(repoRoot(t), "scripts", "version.sh"))
	c.Dir = dir
	var errbuf bytes.Buffer
	c.Stderr = &errbuf
	if err := c.Run(); err == nil {
		t.Fatal("a subject with no type was accepted")
	}
	for _, line := range strings.Split(strings.TrimSpace(errbuf.String()), "\n") {
		if strings.HasPrefix(line, "  ") && len(strings.Fields(line)) < 3 {
			t.Errorf("the refusal broke the subject into words:\n%s", errbuf.String())
			break
		}
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
// the tap publisher had, which is where the shape came from — it was removed
// with the tap step, and this is now the only place that carries it.
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

// -- what an install is allowed to ask of the machine -------------------------

// Nothing in this class of tool requires an external verification package, and
// a first install is the worst moment to ask for one. rustup, bun, deno and nvm
// verify nothing at all; k3s checks a SHA-256 from the release host; uv carries
// the digest in the installer and skips even that when the hashing tool is
// missing. None of the six requires anything to be installed first.
//
// So cosign stops being something the installer talks about. What matters is
// whether a substituted release was covered, and two independent things cover
// it: the digest this installer carries, and the signature. Either one is
// enough, and when either held there is nothing to report — telling someone the
// signature went unchecked, while the check that matters passed by a route that
// does not depend on it, is noise dressed as diligence.
func TestAnInstallCoveredByItsCarriedDigestSaysNothingAboutCosign(t *testing.T) {
	f := newInstallFixture(t)
	f.absent, f.pin = true, "1.2.3"
	name := "dcode_1.2.3_linux_amd64.tar.gz"
	sum := f.artifact(t, name, "#!/bin/sh\necho dcode 1.2.3\n")
	fourPlatformChecksums(t, f, name, sum)

	out, err := f.run(t)
	if err != nil {
		t.Fatalf("the install failed: %v\n%s", err, out)
	}
	if strings.Contains(out, "cosign") {
		t.Errorf("a covered install still talks about a package the user will "+
			"never install:\n%s", out)
	}
}

// The other half of the same rule. No carried digest, but the signature did
// verify — substitution is covered by the other route, so again nothing to say.
func TestAnInstallCoveredByItsSignatureIsAlsoQuiet(t *testing.T) {
	f := newInstallFixture(t)
	completeRelease(t, f) // cosign stub present, exits 0

	out, err := f.run(t)
	if err != nil {
		t.Fatalf("the install failed: %v\n%s", err, out)
	}
	if strings.Contains(out, "substituted release") {
		t.Errorf("an install whose signature verified was told substitution went "+
			"unchecked:\n%s", out)
	}
}

// And when NEITHER covered it, that is worth saying — but the way out is the
// installer that carries this release's digests, never a package to install.
// An installer that answers a problem with "install something else first" has
// handed the user a second problem.
func TestAnUncoveredInstallPointsAtThePinnedInstallerAndNotAtAPackage(t *testing.T) {
	f := newInstallFixture(t)
	f.absent = true
	completeRelease(t, f)

	out, err := f.run(t)
	if err != nil {
		t.Fatalf("the install failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "substituted release") {
		t.Errorf("nothing covered substitution and the installer did not say so:\n%s", out)
	}
	if !strings.Contains(out, "releases/download/v1.2.3/install.sh") {
		t.Errorf("the notice does not name the installer that carries the digests:\n%s", out)
	}
	for _, forbidden := range []string{"install cosign", "Install cosign"} {
		if strings.Contains(out, forbidden) {
			t.Errorf("the installer tells the user to install a package:\n%s", out)
		}
	}
}

// A dev build is named for the version it is heading TO.
//
// It took the last tag, so every build between two releases reported the older
// one: a binary carrying two days of work called itself 0.1.0, and the only
// thing saying otherwise was a commit hash nobody reads. Somebody watched that
// number not move and reasonably concluded nothing had been installed.
//
// Read as data, like release.yml, because nothing in `make check` inspects the
// Makefile — a build that quietly went back to the tag would say so only in a
// version string somebody has already learned to distrust.
func TestADevBuildIsNamedForTheVersionItIsHeadingTo(t *testing.T) {
	body, err := os.ReadFile(filepath.Join(repoRoot(t), "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	make := string(body)
	if !strings.Contains(make, "scripts/version.sh") {
		t.Error("the build does not derive its version from scripts/version.sh, " +
			"so a dev build reports the release it left rather than the one it is heading to")
	}
	if !strings.Contains(make, "NEXT_VER") || !strings.Contains(make, "$(NEXT_VER)") {
		t.Error("the derived version is computed and not used")
	}
}
