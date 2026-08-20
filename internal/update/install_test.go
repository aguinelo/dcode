package update

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// install.sh had no test at all — not a Go test, not a shell test, and neither
// CI workflow runs or lints it. It is the channel the README tells people to
// use, and its whole purpose is to refuse a download that does not verify.
//
// The script is exercised AS SHIPPED. Rather than adding a base-URL hook so a
// fixture server could stand in, the tools it reaches for are stubbed on PATH:
// a script that only passes its tests because it was modified to be testable
// proves something about a different script.

type installFixture struct {
	dir     string // the fake release, served by the stub curl
	bin     string // stub executables, first on PATH
	root    string // the fixture root, for files that must not look like residue
	tmp     string // TMPDIR, so the work directory can be checked afterwards
	target  string // DCODE_INSTALL_DIR
	cosign  string // "0" or "1" — the exit code the stub cosign returns
	absent  bool   // cosign is nowhere on PATH — the ordinary user's machine
	pin     string // pin the installer to this release before running it
	unameS  string
	unameM  string
	version string
}

func newInstallFixture(t *testing.T) *installFixture {
	t.Helper()
	for _, needed := range []string{"sh", "tar"} {
		if _, err := exec.LookPath(needed); err != nil {
			t.Skipf("no %s", needed)
		}
	}
	root := t.TempDir()
	f := &installFixture{
		root: root,
		dir:  filepath.Join(root, "release"), bin: filepath.Join(root, "bin"),
		tmp: filepath.Join(root, "tmp"), target: filepath.Join(root, "install"),
		cosign: "0", unameS: "Linux", unameM: "x86_64", version: "v1.2.3",
	}
	for _, d := range []string{f.dir, f.bin, f.tmp} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

// artifact writes a tarball containing a runnable dcode and returns its digest.
func (f *installFixture) artifact(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(f.dir, name)
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(out)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{
		Name: "dcode", Mode: 0o755, Size: int64(len(body)),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	for _, c := range []interface{ Close() error }{tw, gz, out} {
		if err := c.Close(); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (f *installFixture) write(t *testing.T, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(f.dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// stubs installs the executables the script reaches for. curl serves the fake
// release by filename; cosign answers with whatever the test decided; uname
// reports the platform under test.
func (f *installFixture) stubs(t *testing.T) {
	t.Helper()
	put := func(name, body string) {
		t.Helper()
		p := filepath.Join(f.bin, name)
		if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// The script calls: curl -fsSL <url> -o <dest>. Only the basename matters.
	put("curl", fmt.Sprintf(`
dest=""
url=""
while [ $# -gt 0 ]; do
  case "$1" in
    -o) dest="$2"; shift 2 ;;
    -*) shift ;;
    *) url="$1"; shift ;;
  esac
done
name="${url##*/}"
[ -f %q/"$name" ] || exit 22
if [ -n "$dest" ]; then cp %q/"$name" "$dest"; else cat %q/"$name"; fi
`, f.dir, f.dir, f.dir))
	if !f.absent {
		put("cosign", fmt.Sprintf("exit %s\n", f.cosign))
	}
	put("uname", fmt.Sprintf(`
case "$1" in
  -s) printf '%%s\n' %q ;;
  -m) printf '%%s\n' %q ;;
esac
`, f.unameS, f.unameM))
}

// searchPath puts the stubs first. When the test wants cosign absent, every
// directory that carries one is dropped: "absent" has to mean absent from the
// machine, not merely missing from the stub directory, or the test would pass
// or fail according to who runs it.
func (f *installFixture) searchPath() string {
	dirs := []string{f.bin}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			continue
		}
		if f.absent {
			if info, err := os.Stat(filepath.Join(dir, "cosign")); err == nil && !info.IsDir() {
				continue
			}
		}
		dirs = append(dirs, dir)
	}
	return strings.Join(dirs, string(os.PathListSeparator))
}

// script returns the installer to run. With no pin that is the repository's own
// file; with one it is a copy that scripts/installer.sh has pinned to a
// release, which is exactly what a published installer is. Pinning a copy keeps
// the "exercised as shipped" property: the thing under test is the generator's
// output, not a hand-written approximation of it.
func (f *installFixture) script(t *testing.T, root string) string {
	t.Helper()
	if f.pin == "" {
		return filepath.Join(root, "install.sh")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("no bash")
	}
	src, err := os.ReadFile(filepath.Join(root, "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	// Outside TMPDIR on purpose: the residue check reads that directory, and a
	// test fixture sitting in it would read as a download the installer forgot.
	copied := filepath.Join(f.root, "pinned-install.sh")
	if err := os.WriteFile(copied, src, 0o755); err != nil {
		t.Fatal(err)
	}
	gen := exec.Command(filepath.Join(root, "scripts", "installer.sh"),
		f.pin, filepath.Join(f.dir, "checksums.txt"), copied)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("scripts/installer.sh could not pin the installer: %v\n%s", err, out)
	}
	return copied
}

func (f *installFixture) run(t *testing.T) (string, error) {
	t.Helper()
	f.stubs(t)
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", f.script(t, root))
	cmd.Env = append(os.Environ(),
		"PATH="+f.searchPath(),
		"TMPDIR="+f.tmp,
		"DCODE_VERSION="+f.version,
		"DCODE_INSTALL_DIR="+f.target,
		"DCODE_SKIP_VERIFY=false",
	)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (f *installFixture) installed() bool {
	_, err := os.Stat(filepath.Join(f.target, "dcode"))
	return err == nil
}

// leftovers reports anything the run left behind in TMPDIR.
func (f *installFixture) leftovers(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(f.tmp)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

// completeRelease lays out a release that verifies.
func completeRelease(t *testing.T, f *installFixture) {
	t.Helper()
	name := "dcode_1.2.3_linux_amd64.tar.gz"
	sum := f.artifact(t, name, "#!/bin/sh\necho dcode 1.2.3\n")
	f.write(t, "checksums.txt", sum+"  "+name+"\n")
	f.write(t, "checksums.txt.sig", "signature")
	f.write(t, "checksums.txt.pem", "certificate")
}

func TestTheInstallerInstallsAVerifiedRelease(t *testing.T) {
	f := newInstallFixture(t)
	completeRelease(t, f)

	out, err := f.run(t)
	if err != nil {
		t.Fatalf("a verified release failed to install: %v\n%s", err, out)
	}
	if !f.installed() {
		t.Errorf("nothing was installed:\n%s", out)
	}
	if got := f.leftovers(t); len(got) != 0 {
		t.Errorf("the work directory survived a successful run: %v", got)
	}
}

// The signature covers the whole release. Failing it means the checksums file
// itself cannot be trusted, so nothing below it can be either.
//
// "Installed, but unverified" is the worst of both worlds: the user ends up
// with a binary AND the impression that it went fine.
func TestASignatureThatDoesNotVerifyInstallsNothingAndLeavesNoResidue(t *testing.T) {
	f := newInstallFixture(t)
	completeRelease(t, f)
	f.cosign = "1"

	out, err := f.run(t)
	if err == nil {
		t.Fatalf("a bad signature installed anyway:\n%s", out)
	}
	if f.installed() {
		t.Error("a binary was installed despite the signature failing")
	}
	if !strings.Contains(out, "signature") {
		t.Errorf("the failure does not say what went wrong:\n%s", out)
	}
	if got := f.leftovers(t); len(got) != 0 {
		t.Errorf("the download survived the failure: %v", got)
	}
}

// The signature says the checksums file is authentic; this says the artifact is
// the one it describes. Both are needed: a valid signature over a list that
// does not match what was downloaded proves the list, not the download.
func TestAChecksumMismatchInstallsNothing(t *testing.T) {
	f := newInstallFixture(t)
	name := "dcode_1.2.3_linux_amd64.tar.gz"
	f.artifact(t, name, "#!/bin/sh\necho dcode 1.2.3\n")
	f.write(t, "checksums.txt", strings.Repeat("0", 64)+"  "+name+"\n")
	f.write(t, "checksums.txt.sig", "signature")
	f.write(t, "checksums.txt.pem", "certificate")

	out, err := f.run(t)
	if err == nil {
		t.Fatalf("a mismatched artifact installed anyway:\n%s", out)
	}
	if f.installed() {
		t.Error("a binary was installed despite the checksum not matching")
	}
	if !strings.Contains(out, "checksum") {
		t.Errorf("the failure does not name the checksum:\n%s", out)
	}
	if got := f.leftovers(t); len(got) != 0 {
		t.Errorf("the download survived the failure: %v", got)
	}
}

// An artifact absent from the signed list is not a missing file — it is a file
// nobody signed. Installing it would defeat both checks at once.
func TestAnArtifactMissingFromTheSignedListInstallsNothing(t *testing.T) {
	f := newInstallFixture(t)
	f.artifact(t, "dcode_1.2.3_linux_amd64.tar.gz", "#!/bin/sh\necho dcode 1.2.3\n")
	f.write(t, "checksums.txt", strings.Repeat("a", 64)+"  something_else.tar.gz\n")
	f.write(t, "checksums.txt.sig", "signature")
	f.write(t, "checksums.txt.pem", "certificate")

	out, err := f.run(t)
	if err == nil {
		t.Fatalf("an unlisted artifact installed anyway:\n%s", out)
	}
	if f.installed() {
		t.Error("a binary nobody signed was installed")
	}
}

// The user learns what IS available, rather than only what failed. A bare
// "unsupported" sends someone to the issue tracker to ask a question the
// message could have answered.
func TestAnUnsupportedPlatformAbortsAndListsWhatIsSupported(t *testing.T) {
	for _, c := range []struct{ name, s, m string }{
		{"operating system", "SunOS", "x86_64"},
		{"architecture", "Linux", "ppc64"},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newInstallFixture(t)
			completeRelease(t, f)
			f.unameS, f.unameM = c.s, c.m

			out, err := f.run(t)
			if err == nil {
				t.Fatalf("an unsupported platform proceeded:\n%s", out)
			}
			for _, pair := range []string{
				"darwin/amd64", "darwin/arm64", "linux/amd64", "linux/arm64",
			} {
				if !strings.Contains(out, pair) {
					t.Errorf("the message does not offer %s:\n%s", pair, out)
				}
			}
			if f.installed() {
				t.Error("something was installed for a platform that cannot run it")
			}
		})
	}
}

// The last check before claiming success: the binary runs HERE. A correct
// download for the wrong libc verifies perfectly and cannot execute, and
// reporting that as a successful install is how someone spends an afternoon on
// a message the installer already had.
func TestABinaryThatCannotRunIsNotReportedAsInstalled(t *testing.T) {
	f := newInstallFixture(t)
	name := "dcode_1.2.3_linux_amd64.tar.gz"
	sum := f.artifact(t, name, "#!/bin/sh\nexit 1\n")
	f.write(t, "checksums.txt", sum+"  "+name+"\n")
	f.write(t, "checksums.txt.sig", "signature")
	f.write(t, "checksums.txt.pem", "certificate")

	out, err := f.run(t)
	if err == nil {
		t.Fatalf("a binary that does not run was reported as installed:\n%s", out)
	}
	if !strings.Contains(out, "does not run") {
		t.Errorf("the failure does not say the binary could not run:\n%s", out)
	}
}

// The release pipeline stamps Source=release, and that stamp is what makes a
// released binary willing to update itself: Apply refuses a local build without
// --force. Delete the flag and every test in the repository still passes, while
// the shipped binary quietly reports "local build" and declines to upgrade —
// discovered by users, not by CI.
//
// So the workflow is read as data, the same way the formula test reads
// scripts/formula.sh. A build stamp nobody asserts is a build stamp that can be
// dropped in a refactor of the YAML.
func TestTheReleasePipelineStampsEveryFieldTheBinaryReportsOn(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	yaml := string(data)

	const pkg = "github.com/aguinelo/dcode/internal/version."
	for _, field := range []string{"Version", "Commit", "Date", "Source"} {
		if !strings.Contains(yaml, "-X "+pkg+field+"=") {
			t.Errorf("release.yml does not stamp %s; the binary reports it and nothing sets it", field)
		}
	}
	// Source is the one with teeth: its value decides whether the binary will
	// update itself at all.
	if !strings.Contains(yaml, "-X "+pkg+"Source=release") {
		t.Error("release.yml does not stamp Source=release; the shipped binary would " +
			"call itself a local build and refuse to upgrade without --force")
	}
	// And the build must actually use what it assembled.
	if !strings.Contains(yaml, `-ldflags "$LDFLAGS"`) {
		t.Error("the build step does not use the assembled LDFLAGS")
	}
}

// The first real user ran the documented command and got
// "dcode: cosign is required but not installed" — no binary, and no check
// performed either. Every test above stubs cosign into existence, so the one
// configuration every ordinary machine is in was the one never exercised.
//
// The two checks are independent and were wired in series: `need cosign` sat
// immediately before the signature check, and the SHA-256 comparison came
// after it, so the absence of the stronger check cancelled the weaker one. The
// outcome is the worst available — nothing installed AND nothing verified —
// from a script whose own header argues against exactly that.
//
// The release here is complete, so the run reaches the reported symptom rather
// than tripping earlier on a missing file. What it must not do is stop.
func TestAMissingCosignStillChecksTheChecksumAndInstalls(t *testing.T) {
	f := newInstallFixture(t)
	f.absent = true
	completeRelease(t, f)

	out, err := f.run(t)
	if err != nil {
		t.Fatalf("a machine without cosign could not install: %v\n%s", err, out)
	}
	if !f.installed() {
		t.Errorf("nothing was installed:\n%s", out)
	}
	if got := f.leftovers(t); len(got) != 0 {
		t.Errorf("the work directory survived a successful run: %v", got)
	}
}

// Installing without the signature is acceptable; installing without saying so
// is not. The distinction the original conflated is between unverified and
// unverified-in-silence, and only the second one is indefensible.
//
// The message has to name the tool, or the user cannot act on it.
func TestAMissingCosignSaysTheSignatureWasNotVerified(t *testing.T) {
	f := newInstallFixture(t)
	f.absent = true
	completeRelease(t, f)

	out, _ := f.run(t)
	for _, want := range []string{"signature", "cosign"} {
		if !strings.Contains(out, want) {
			t.Errorf("the installer never mentions %q, so the user cannot tell "+
				"what was skipped or how to check it:\n%s", want, out)
		}
	}
}

// Degrading the signature check must not degrade the checksum. If a missing
// cosign turned the SHA-256 comparison into a formality too, the fix would
// have replaced a loud failure with a quiet one — the trade this repository
// keeps finding itself on the wrong side of.
//
// The signature files are present and unused: their absence would abort the
// run earlier, and this has to fail on the checksum specifically.
func TestAMissingCosignStillRefusesAMismatchedChecksum(t *testing.T) {
	f := newInstallFixture(t)
	f.absent = true
	name := "dcode_1.2.3_linux_amd64.tar.gz"
	f.artifact(t, name, "#!/bin/sh\necho dcode 1.2.3\n")
	f.write(t, "checksums.txt", strings.Repeat("0", 64)+"  "+name+"\n")
	f.write(t, "checksums.txt.sig", "signature")
	f.write(t, "checksums.txt.pem", "certificate")

	out, err := f.run(t)
	if err == nil {
		t.Fatalf("a mismatched artifact installed on a machine without cosign:\n%s", out)
	}
	if f.installed() {
		t.Error("a binary was installed despite the checksum not matching")
	}
	if !strings.Contains(out, "mismatch") {
		t.Errorf("the failure does not report a checksum mismatch:\n%s", out)
	}
	if got := f.leftovers(t); len(got) != 0 {
		t.Errorf("the download survived the failure: %v", got)
	}
}

// Nothing is gained by fetching a signature no tool on this machine can check,
// and something is lost: a release published without them would fail to
// install for a reason that has nothing to do with the user.
//
// The stub curl refuses a file it was not given, so a run that still asks for
// the signature fails here — which is what makes the claim testable at all.
func TestAMissingCosignDoesNotDownloadTheSignatureItCannotCheck(t *testing.T) {
	f := newInstallFixture(t)
	f.absent = true
	name := "dcode_1.2.3_linux_amd64.tar.gz"
	sum := f.artifact(t, name, "#!/bin/sh\necho dcode 1.2.3\n")
	f.write(t, "checksums.txt", sum+"  "+name+"\n")

	out, err := f.run(t)
	if err != nil {
		t.Fatalf("the installer asked for a signature it cannot use: %v\n%s", err, out)
	}
	if !f.installed() {
		t.Errorf("nothing was installed:\n%s", out)
	}
}

// And where cosign IS present it still has teeth: a bad signature aborts.
// Asserted next to the degradation so the two cannot drift apart — the danger
// in making a check optional is that it quietly becomes decorative.
func TestCosignStillAbortsWhenItIsPresentAndTheSignatureIsBad(t *testing.T) {
	f := newInstallFixture(t)
	completeRelease(t, f)
	f.cosign = "1"

	out, err := f.run(t)
	if err == nil {
		t.Fatalf("a bad signature installed anyway:\n%s", out)
	}
	if f.installed() {
		t.Error("a binary was installed despite the signature failing")
	}
}

// fourPlatformChecksums writes a checksums.txt covering every published
// platform, with the real digest for the one under test. The generator refuses
// a release that is missing a platform — deliberately, since a release short
// one artifact is broken — so a fixture that lists only its own would fail for
// a reason that has nothing to do with the test.
func fourPlatformChecksums(t *testing.T, f *installFixture, name, sum string) {
	t.Helper()
	lines := ""
	for _, p := range []string{"darwin_amd64", "darwin_arm64", "linux_amd64", "linux_arm64"} {
		other := "dcode_1.2.3_" + p + ".tar.gz"
		if other == name {
			lines += sum + "  " + other + "\n"
			continue
		}
		lines += strings.Repeat("b", 64) + "  " + other + "\n"
	}
	f.write(t, "checksums.txt", lines)
}

// The whole point of carrying the digest, and the only test that can show it.
//
// checksums.txt travels from the same host as the tarball, so on its own it
// catches a corrupted download and not a substituted release: whoever can
// replace one can replace the other, and the pair stays self-consistent. Here
// they ARE consistent — the list vouches for the swapped artifact — and the
// installer must still refuse, because the digest it carries came by a
// different route and says otherwise.
//
// That route is the reason cosign can be optional (#222) without the checksum
// becoming decorative: a release asset can be replaced with no public trace,
// while this digest lives in git history, where changing it is a commit.
func TestACarriedDigestRefusesAnArtifactTheSignedListAccepts(t *testing.T) {
	f := newInstallFixture(t)
	f.absent, f.pin = true, "1.2.3"
	name := "dcode_1.2.3_linux_amd64.tar.gz"

	// Pin the installer to the honest release.
	honest := f.artifact(t, name, "#!/bin/sh\necho dcode 1.2.3\n")
	fourPlatformChecksums(t, f, name, honest)
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	pinned := f.script(t, root)

	// Now swap both the artifact and the list it is checked against.
	swapped := f.artifact(t, name, "#!/bin/sh\necho pwned\n")
	if swapped == honest {
		t.Fatal("the fixture did not actually change the artifact")
	}
	fourPlatformChecksums(t, f, name, swapped)
	f.pin = "" // the copy is already pinned; do not regenerate it against the swap

	f.stubs(t)
	cmd := exec.Command("sh", pinned)
	cmd.Env = append(os.Environ(),
		"PATH="+f.searchPath(), "TMPDIR="+f.tmp,
		"DCODE_VERSION="+f.version, "DCODE_INSTALL_DIR="+f.target,
		"DCODE_SKIP_VERIFY=false")
	raw, err := cmd.CombinedOutput()
	out := string(raw)

	if err == nil {
		t.Fatalf("a substituted release installed, and its own list vouched for it:\n%s", out)
	}
	if f.installed() {
		t.Error("a substituted binary was installed")
	}
	if !strings.Contains(out, "mismatch") {
		t.Errorf("the failure does not report a mismatch:\n%s", out)
	}
	if got := f.leftovers(t); len(got) != 0 {
		t.Errorf("the download survived the failure: %v", got)
	}
}

// The honest release still installs, pinned. Without this the test above is
// satisfied by an installer that refuses everything.
func TestACarriedDigestInstallsTheReleaseItWasPinnedTo(t *testing.T) {
	f := newInstallFixture(t)
	f.absent, f.pin = true, "1.2.3"
	name := "dcode_1.2.3_linux_amd64.tar.gz"
	sum := f.artifact(t, name, "#!/bin/sh\necho dcode 1.2.3\n")
	fourPlatformChecksums(t, f, name, sum)

	out, err := f.run(t)
	if err != nil {
		t.Fatalf("a pinned installer refused its own release: %v\n%s", err, out)
	}
	if !f.installed() {
		t.Errorf("nothing was installed:\n%s", out)
	}
}

// An installer pinned to one release, asked for another, cannot check what it
// does not carry. It falls back to the release's own list and says so, naming
// the installer that does carry the right digests — the same answer uv gives,
// and the only one that is both usable and honest.
func TestAnInstallerAskedForAnotherReleaseSaysItCannotCheckIt(t *testing.T) {
	f := newInstallFixture(t)
	f.absent, f.pin = true, "9.9.9"
	name := "dcode_1.2.3_linux_amd64.tar.gz"
	sum := f.artifact(t, name, "#!/bin/sh\necho dcode 1.2.3\n")
	// Pinned against a release whose names do not match the one being installed.
	f.write(t, "checksums.txt", strings.Join([]string{
		strings.Repeat("b", 64) + "  dcode_9.9.9_darwin_amd64.tar.gz",
		strings.Repeat("b", 64) + "  dcode_9.9.9_darwin_arm64.tar.gz",
		strings.Repeat("b", 64) + "  dcode_9.9.9_linux_amd64.tar.gz",
		strings.Repeat("b", 64) + "  dcode_9.9.9_linux_arm64.tar.gz",
	}, "\n")+"\n")
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	pinned := f.script(t, root)

	// The release actually being served is 1.2.3, and its list is honest.
	f.write(t, "checksums.txt", sum+"  "+name+"\n")

	f.stubs(t)
	cmd := exec.Command("sh", pinned)
	cmd.Env = append(os.Environ(),
		"PATH="+f.searchPath(), "TMPDIR="+f.tmp,
		"DCODE_VERSION="+f.version, "DCODE_INSTALL_DIR="+f.target,
		"DCODE_SKIP_VERIFY=false")
	raw, err := cmd.CombinedOutput()
	out := string(raw)

	if err != nil {
		t.Fatalf("an installer pinned elsewhere refused to fall back: %v\n%s", err, out)
	}
	if !f.installed() {
		t.Errorf("nothing was installed:\n%s", out)
	}
	if !strings.Contains(out, "9.9.9") {
		t.Errorf("the notice does not say which release this installer carries:\n%s", out)
	}
	if !strings.Contains(out, "releases/download/v1.2.3/install.sh") {
		t.Errorf("the notice does not name the installer that can check v1.2.3:\n%s", out)
	}
}

// The installer in the repository carries no digests until a release pins it,
// and that state has to be quiet: warning on every install about a pin that
// was never applied trains people to ignore the line that matters.
func TestAnUnpinnedInstallerFallsBackWithoutComplaining(t *testing.T) {
	f := newInstallFixture(t)
	f.absent = true
	completeRelease(t, f)

	out, err := f.run(t)
	if err != nil {
		t.Fatalf("the unpinned installer failed: %v\n%s", err, out)
	}
	if strings.Contains(out, "carries") {
		t.Errorf("an installer with no pins complained about not having them:\n%s", out)
	}
}

// The generator writes the digests of the artifacts that were signed, for every
// published platform. Hand-typing one passes every local test, installs
// everywhere, and one day points at a binary nobody signed — the reasoning
// scripts/formula.sh already carries, in the other channel.
func TestTheGeneratorCarriesEveryPublishedDigest(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("no bash")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	sums, lines := filepath.Join(dir, "checksums.txt"), ""
	want := map[string]string{}
	for i, p := range []string{"darwin_amd64", "darwin_arm64", "linux_amd64", "linux_arm64"} {
		digest := strings.Repeat(string(rune('a'+i)), 64)
		want["dcode_4.5.6_"+p+".tar.gz"] = digest
		lines += digest + "  dcode_4.5.6_" + p + ".tar.gz\n"
	}
	if err := os.WriteFile(sums, []byte(lines), 0o644); err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join(root, "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "install.sh")
	if err := os.WriteFile(target, src, 0o755); err != nil {
		t.Fatal(err)
	}
	gen := exec.Command(filepath.Join(root, "scripts", "installer.sh"), "4.5.6", sums, target)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("the generator failed: %v\n%s", err, out)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	for artifact, digest := range want {
		if !strings.Contains(string(got), artifact) {
			t.Errorf("the pinned installer does not mention %s", artifact)
		}
		if !strings.Contains(string(got), digest) {
			t.Errorf("the pinned installer does not carry the digest of %s", artifact)
		}
	}
	if !strings.Contains(string(got), `PINNED_VERSION="4.5.6"`) {
		t.Error("the pinned installer does not record which release it carries")
	}
}

// A release short one platform is broken, and the generator has to say so
// rather than write an empty digest — which would verify nothing while looking
// exactly like a digest that verified.
func TestTheGeneratorRefusesAReleaseMissingAPlatform(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("no bash")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	sums := filepath.Join(dir, "checksums.txt")
	if err := os.WriteFile(sums,
		[]byte(strings.Repeat("a", 64)+"  dcode_4.5.6_linux_amd64.tar.gz\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join(root, "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "install.sh")
	if err := os.WriteFile(target, src, 0o755); err != nil {
		t.Fatal(err)
	}
	gen := exec.Command(filepath.Join(root, "scripts", "installer.sh"), "4.5.6", sums, target)
	out, err := gen.CombinedOutput()
	if err == nil {
		t.Fatalf("the generator pinned an incomplete release:\n%s", out)
	}
	if !strings.Contains(string(out), "darwin_amd64") {
		t.Errorf("the failure does not name the platform that is missing:\n%s", out)
	}
}
