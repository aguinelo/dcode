package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// `dcode update` required cosign, and that requirement had the same shape the
// install script has already shed twice: it names a package nobody installs, on
// a machine that already has a working dcode.
//
// The installer stopped needing it because a release commits the digests of its
// artifacts into install.sh on main, where replacing a release asset leaves no
// public trace and changing a tracked line is a commit. The binary can read the
// same file, so it has the same second route — it just never used it.
//
// The rule is the installer's rule: a substituted release is covered by the
// carried digest OR by the signature, and either is enough.

// noVerifier stands for a machine with no cosign.
type noVerifier struct{}

func (noVerifier) Verify(context.Context, []byte, []byte, []byte) error {
	return ErrNoVerifier
}

// installerCarrying serves an install.sh pinned to the given digests, the way
// raw.githubusercontent.com serves main/install.sh.
func installerCarrying(t *testing.T, digests map[string]string) string {
	t.Helper()
	var b strings.Builder
	b.WriteString("#!/usr/bin/env sh\n# BEGIN PINNED\nPINNED_VERSION=\"1.2.3\"\npinned_sum() {\n  case \"$1\" in\n")
	for name, sum := range digests {
		fmt.Fprintf(&b, "    %s) echo %s ;;\n", name, sum)
	}
	b.WriteString("  esac\n}\n# END PINNED\n")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, b.String())
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// The parse, on its own. It reads a real generated block, so a change to
// scripts/installer.sh's output that this cannot read is a change that breaks
// the updater — and that is worth failing on here rather than in the field.
func TestPinnedDigestReadsTheBlockAnInstallerCarries(t *testing.T) {
	script := `#!/usr/bin/env sh
# BEGIN PINNED — gerado por scripts/installer.sh
PINNED_VERSION="0.0.1"
pinned_sum() {
  case "$1" in
    dcode_0.0.1_darwin_amd64.tar.gz) echo ` + strings.Repeat("a", 64) + ` ;;
    dcode_0.0.1_darwin_arm64.tar.gz) echo ` + strings.Repeat("b", 64) + ` ;;
  esac
}
# END PINNED
`
	if got := PinnedDigest([]byte(script), "dcode_0.0.1_darwin_arm64.tar.gz"); got != strings.Repeat("b", 64) {
		t.Errorf("the digest for darwin_arm64 came back %q", got)
	}
	if got := PinnedDigest([]byte(script), "dcode_0.0.1_linux_amd64.tar.gz"); got != "" {
		t.Errorf("a digest was invented for an artifact the block does not carry: %q", got)
	}
}

// A block outside the markers is not a pinned block. Reading digests from
// anywhere in the file would let an unrelated line decide what gets installed.
func TestPinnedDigestIgnoresAnythingOutsideTheMarkers(t *testing.T) {
	script := "dcode_0.0.1_linux_amd64.tar.gz) echo " + strings.Repeat("c", 64) + " ;;\n" +
		"# BEGIN PINNED\npinned_sum() {\n  case \"$1\" in\n  esac\n}\n# END PINNED\n"
	if got := PinnedDigest([]byte(script), "dcode_0.0.1_linux_amd64.tar.gz"); got != "" {
		t.Errorf("a digest outside the pinned block was used: %q", got)
	}
}

// The point: an ordinary machine updates.
func TestUpdateWithoutCosignUsesTheDigestCommittedToMain(t *testing.T) {
	asRelease(t)
	f := newFixture(t, "v1.2.3", false)
	target := filepath.Join(t.TempDir(), "dcode")
	if err := os.WriteFile(target, []byte("#!/bin/sh\necho old\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	name := ArtifactName("v1.2.3", runtime.GOOS, runtime.GOARCH)
	installer := installerCarrying(t, map[string]string{name: f.sum})

	u := newUpdater(t, f, target, func(c *Config) {
		c.Verifier = noVerifier{}
		c.InstallerURL = installer
	})
	rel, err := u.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := u.Apply(context.Background(), rel); err != nil {
		t.Fatalf("a machine without cosign could not update: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "old") {
		t.Error("the binary was not replaced")
	}
}

// The carried digest is a check, not a formality. When it disagrees with what
// was downloaded, the update stops — and the working binary stays.
func TestUpdateRefusesWhenTheCarriedDigestDisagrees(t *testing.T) {
	asRelease(t)
	f := newFixture(t, "v1.2.3", false)
	target := filepath.Join(t.TempDir(), "dcode")
	if err := os.WriteFile(target, []byte("#!/bin/sh\necho old\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	name := ArtifactName("v1.2.3", runtime.GOOS, runtime.GOARCH)
	other := sha256.Sum256([]byte("something else"))
	installer := installerCarrying(t, map[string]string{name: hex.EncodeToString(other[:])})

	u := newUpdater(t, f, target, func(c *Config) {
		c.Verifier = noVerifier{}
		c.InstallerURL = installer
	})
	rel, err := u.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := u.Apply(context.Background(), rel); err == nil {
		t.Fatal("an artifact the committed digest disagrees with was installed")
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "old") {
		t.Error("the working binary was replaced despite the refusal")
	}
}

// When neither route covered a substituted release, the update stops rather
// than proceeding on a checksum that came from the same host as the artifact.
// Unlike a first install there is something to lose here: a working binary.
//
// And the way out is never a package. It is the release whose digests main
// carries, which is the latest one.
func TestUpdateRefusesWhenNothingCoveredSubstitution(t *testing.T) {
	asRelease(t)
	f := newFixture(t, "v1.2.3", false)
	target := filepath.Join(t.TempDir(), "dcode")
	if err := os.WriteFile(target, []byte("#!/bin/sh\necho old\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// An installer that carries digests for some other release.
	installer := installerCarrying(t, map[string]string{"dcode_9.9.9_linux_amd64.tar.gz": strings.Repeat("d", 64)})

	u := newUpdater(t, f, target, func(c *Config) {
		c.Verifier = noVerifier{}
		c.InstallerURL = installer
	})
	rel, err := u.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	err = u.Apply(context.Background(), rel)
	if err == nil {
		t.Fatal("an unverifiable update was applied")
	}
	if !strings.Contains(err.Error(), "substituted") {
		t.Errorf("the refusal does not say what could not be ruled out: %v", err)
	}
	for _, forbidden := range []string{"Install cosign", "install cosign"} {
		if strings.Contains(err.Error(), forbidden) {
			t.Errorf("the refusal tells the user to install a package: %v", err)
		}
	}
	if !strings.Contains(string(mustRead(t, target)), "old") {
		t.Error("the working binary was replaced despite the refusal")
	}
}

// A signature that verifies still covers substitution on its own, so an
// unreachable installer is not a reason to refuse.
func TestUpdateStillWorksOnASignatureAlone(t *testing.T) {
	asRelease(t)
	f := newFixture(t, "v1.2.3", false)
	target := filepath.Join(t.TempDir(), "dcode")
	if err := os.WriteFile(target, []byte("#!/bin/sh\necho old\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	u := newUpdater(t, f, target, func(c *Config) {
		c.InstallerURL = "http://127.0.0.1:1/never" // unreachable on purpose
	})
	rel, err := u.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := u.Apply(context.Background(), rel); err != nil {
		t.Fatalf("a verified signature was not enough on its own: %v", err)
	}
}

// And a signature that fails still stops the update, whatever the carried
// digest said. Making a check optional must not make it decorative.
func TestASignatureThatFailsStopsTheUpdateEvenWithACarriedDigest(t *testing.T) {
	asRelease(t)
	f := newFixture(t, "v1.2.3", false)
	target := filepath.Join(t.TempDir(), "dcode")
	if err := os.WriteFile(target, []byte("#!/bin/sh\necho old\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	name := ArtifactName("v1.2.3", runtime.GOOS, runtime.GOARCH)
	u := newUpdater(t, f, target, func(c *Config) {
		c.Verifier = failVerifier{}
		c.InstallerURL = installerCarrying(t, map[string]string{name: f.sum})
	})
	rel, err := u.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := u.Apply(context.Background(), rel); err == nil {
		t.Fatal("a failing signature was overridden by the carried digest")
	}
	if errors.Is(err, ErrNoVerifier) {
		t.Error("a failing signature was reported as a missing one")
	}
}

func mustRead(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// A field the updater reads and no command sets is a field that does nothing,
// and this repository has already shipped one: the build stamp, which every
// test passed without and which made every published release call itself a
// local build.
//
// So the wiring is read as data, the same way release.yml is. Nothing in the
// suite runs `dcode update` against a mirror, so an InstallerURL that stopped
// being passed would leave the mirror reading its digests from upstream — the
// one place the override exists to avoid — and nothing would say so.
func TestTheUpdateCommandPassesTheInstallerOverrideItDocuments(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "cmd", "dcode", "update.go"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	if !strings.Contains(body, "DCODE_UPDATE_INSTALLER_URL") {
		t.Error("the update command never reads DCODE_UPDATE_INSTALLER_URL, which the " +
			"configuration spec documents")
	}
	if !strings.Contains(body, "InstallerURL:") {
		t.Error("the update command never sets InstallerURL, so the override is inert")
	}
}
