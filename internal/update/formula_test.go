package update

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The mandatory test the .i spec names: the SHA-256 in the formula is identical
// to the one in checksums.txt.
//
// It is what makes RN-1 true — one artefact for every channel. A formula with a
// hand-typed digest passes every local check, installs everywhere, and one day
// points at a binary nobody signed.
func TestTheFormulaCarriesTheReleaseDigests(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("no bash")
	}
	dir := t.TempDir()
	sums := filepath.Join(dir, "checksums.txt")

	want := map[string]string{
		"dcode_0.4.0_darwin_arm64.tar.gz": "1111111111111111111111111111111111111111111111111111111111111111",
		"dcode_0.4.0_darwin_amd64.tar.gz": "2222222222222222222222222222222222222222222222222222222222222222",
		"dcode_0.4.0_linux_arm64.tar.gz":  "3333333333333333333333333333333333333333333333333333333333333333",
		"dcode_0.4.0_linux_amd64.tar.gz":  "4444444444444444444444444444444444444444444444444444444444444444",
	}
	var b strings.Builder
	for name, sum := range want {
		b.WriteString(sum + "  " + name + "\n")
	}
	if err := os.WriteFile(sums, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := exec.Command("bash", filepath.Join("..", "..", "scripts", "formula.sh"), "0.4.0", sums).Output()
	if err != nil {
		t.Fatalf("generating the formula failed: %v", err)
	}
	formula := string(out)

	for _, sum := range want {
		if !strings.Contains(formula, sum) {
			t.Errorf("the formula does not carry the digest %s from checksums.txt", sum[:8])
		}
	}
	// And nothing else: a digest in the formula that is not in checksums.txt is
	// the exact failure this guards.
	for _, m := range regexp.MustCompile(`sha256 "([0-9a-f]+)"`).FindAllStringSubmatch(formula, -1) {
		found := false
		for _, sum := range want {
			if m[1] == sum {
				found = true
			}
		}
		if !found {
			t.Errorf("the formula carries %s, which is not in checksums.txt", m[1])
		}
	}
}

// A platform missing from the release must stop the formula being written at
// all. Publishing one that points at an artefact nobody built produces a
// download error at install time, which reads as the project being broken.
func TestAMissingArtefactStopsTheFormula(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("no bash")
	}
	dir := t.TempDir()
	sums := filepath.Join(dir, "checksums.txt")
	// Only one of the four.
	if err := os.WriteFile(sums, []byte("1111  dcode_0.4.0_darwin_arm64.tar.gz\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("bash", filepath.Join("..", "..", "scripts", "formula.sh"), "0.4.0", sums).Run(); err == nil {
		t.Fatal("a formula was generated with three of its four platforms missing")
	}
}

// The platforms the formula covers are the platforms the release publishes, and
// both are the platforms that have a sandbox backend.
func TestTheFormulaCoversExactlyThePublishedMatrix(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "scripts", "formula.sh"))
	if err != nil {
		t.Fatal(err)
	}
	script := string(data)
	for _, p := range SupportedPlatforms {
		if !strings.Contains(script, p) {
			t.Errorf("the release publishes %s and the formula does not mention it", p)
		}
	}
	for _, absent := range []string{"windows_amd64", "windows_arm64"} {
		if strings.Contains(script, absent) {
			t.Errorf("the formula covers %s, which the release no longer publishes", absent)
		}
	}
}
