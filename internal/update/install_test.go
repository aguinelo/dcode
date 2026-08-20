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
	tmp     string // TMPDIR, so the work directory can be checked afterwards
	target  string // DCODE_INSTALL_DIR
	cosign  string // "0" or "1" — the exit code the stub cosign returns
	absent  bool   // cosign is nowhere on PATH — the ordinary user's machine
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
		dir: filepath.Join(root, "release"), bin: filepath.Join(root, "bin"),
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

func (f *installFixture) run(t *testing.T) (string, error) {
	t.Helper()
	f.stubs(t)
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", filepath.Join(root, "install.sh"))
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
