package specguard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeRepo lays out the two things Check reads: a `.p` spec with an invariants
// section, and a directory of test files.
func fakeRepo(t *testing.T, spec, testSrc string) (root, testDir string) {
	t.Helper()
	root = t.TempDir()
	dir := filepath.Join(root, "docs", "specs", "architecture", "fam")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "202601010000-fam.p.spec.md"), []byte(spec), 0o644); err != nil {
		t.Fatal(err)
	}
	testDir = filepath.Join(root, "src")
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "a_test.go"), []byte(testSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, testDir
}

const spec = `# Fam

## 7. Invariantes verificáveis

- A escrita é atômica.
- O resultado é ordenado.

## 8. Outra seção

- Esta linha não é invariante.
`

func TestAClaimedInvariantWithARealTestIsSilent(t *testing.T) {
	root, dir := fakeRepo(t, spec, "package a\n\nfunc TestAtomic(t *testing.T) {}\nfunc TestOrdered(t *testing.T) {}\n")

	got, err := Check(root, "fam", []string{dir}, map[string]string{
		"escrita é atômica": "TestAtomic",
		"é ordenado":        "TestOrdered",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("clean mapping reported %v", got)
	}
}

// The first of the two failures the guard exists for: a line nobody claimed.
func TestAnUnclaimedInvariantIsReported(t *testing.T) {
	root, dir := fakeRepo(t, spec, "package a\n\nfunc TestAtomic(t *testing.T) {}\n")

	got, err := Check(root, "fam", []string{dir}, map[string]string{"escrita é atômica": "TestAtomic"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.Contains(got[0], "ordenado") {
		t.Fatalf("findings = %v, want the unclaimed ordering line", got)
	}
}

// The second, and the one a mapping rots into: the test was renamed, so the
// claim now points at nothing.
func TestAClaimNamingAMissingTestIsReported(t *testing.T) {
	root, dir := fakeRepo(t, spec, "package a\n\nfunc TestAtomic(t *testing.T) {}\nfunc TestOrdered(t *testing.T) {}\n")

	got, err := Check(root, "fam", []string{dir}, map[string]string{
		"escrita é atômica": "TestAtomic",
		"é ordenado":        "TestSortedStably", // renamed away
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.Contains(got[0], "TestSortedStably") {
		t.Fatalf("findings = %v, want the dangling claim", got)
	}
}

// A prefix match would let TestOrderedAndStable satisfy a claim on TestOrdered,
// which is how a mapping starts pointing at a test that no longer means what it
// says.
func TestAClaimMatchesTheWholeTestName(t *testing.T) {
	root, dir := fakeRepo(t, spec, "package a\n\nfunc TestAtomic(t *testing.T) {}\nfunc TestOrderedAndStable(t *testing.T) {}\n")

	got, err := Check(root, "fam", []string{dir}, map[string]string{
		"escrita é atômica": "TestAtomic",
		"é ordenado":        "TestOrdered",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("findings = %v; TestOrderedAndStable must not satisfy a claim on TestOrdered", got)
	}
}

// Only the invariants section is read. A guard that swallowed the next section
// would report findings for prose nobody promised.
func TestOnlyTheInvariantsSectionIsRead(t *testing.T) {
	lines, err := Invariants(mustRoot(t, spec), "fam")
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("parsed %d lines, want 2: %v", len(lines), lines)
	}
	for _, l := range lines {
		if strings.Contains(l, "não é invariante") {
			t.Errorf("the guard read past the section: %q", l)
		}
	}
}

// A guard that parses nothing passes everything, which is worse than no guard:
// it reports coverage it never looked for.
func TestASpecWithNoInvariantsIsAnErrorRatherThanASilentPass(t *testing.T) {
	root := mustRoot(t, "# Fam\n\n## 7. Invariantes verificáveis\n\nNenhuma linha.\n")
	if _, err := Invariants(root, "fam"); err == nil {
		t.Fatal("an empty section passed silently")
	}

	root = mustRoot(t, "# Fam\n\n## 7. Outra coisa\n\n- a\n")
	if _, err := Invariants(root, "fam"); err == nil {
		t.Fatal("a spec with no invariants section passed silently")
	}
}

func TestAMissingSpecIsAnError(t *testing.T) {
	if _, err := Invariants(t.TempDir(), "nope"); err == nil {
		t.Fatal("a family with no spec passed silently")
	}
}

func TestAnUnreadableTestDirectoryIsAnError(t *testing.T) {
	root := mustRoot(t, spec)
	if _, err := Check(root, "fam", []string{filepath.Join(root, "absent")}, nil); err == nil {
		t.Fatal("a missing test directory passed silently")
	}
}

func TestALongInvariantIsShortenedForReading(t *testing.T) {
	long := "- " + strings.Repeat("á", 200)
	root := mustRoot(t, "# F\n\n## 7. Invariantes verificáveis\n\n"+long+"\n")
	dir := filepath.Join(root, "src")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Check(root, "fam", []string{dir}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.Contains(got[0], "…") {
		t.Fatalf("findings = %v, want a shortened line", got)
	}
	// Counted in runes: cutting 96 bytes of multi-byte text splits a character
	// and prints a replacement glyph where the invariant should be.
	if strings.Contains(got[0], "�") {
		t.Error("the shortened line was cut mid-character")
	}
}

func mustRoot(t *testing.T, spec string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "docs", "specs", "architecture", "fam")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "f.p.spec.md"), []byte(spec), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// A spec family is not a Go package. The configuration family asserts its
// credential invariants in internal/credential; the protocol family asserts its
// approval-event invariants where the events are emitted. Without this, those
// lines read as unclaimed because their test sits one package over — and the
// obvious fix would be to duplicate the test, which is what the mapping exists
// to avoid.
func TestAClaimMayNameATestInASiblingPackage(t *testing.T) {
	root, dir := fakeRepo(t, spec, "package a\n\nfunc TestAtomic(t *testing.T) {}\n")
	sibling := filepath.Join(root, "other")
	if err := os.MkdirAll(sibling, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sibling, "b_test.go"),
		[]byte("package b\n\nfunc TestOrdered(t *testing.T) {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	mapping := map[string]string{"escrita é atômica": "TestAtomic", "é ordenado": "TestOrdered"}

	got, err := Check(root, "fam", []string{dir}, mapping)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("with one directory the sibling's test must not be found; got %v", got)
	}

	got, err = Check(root, "fam", []string{dir, sibling}, mapping)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("with both directories the mapping is complete; got %v", got)
	}
}
