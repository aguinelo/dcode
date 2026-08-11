package specguard

import (
	"os"
	"path/filepath"
	"regexp"
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

// Every path a `.i` spec names as a file must exist.
//
// Fifty-four did not. A package was renamed, files were merged, tests were
// consolidated — and each time, the step naming the old path stayed as written.
// Nothing checked, so the drift accumulated quietly.
//
// A step naming a file nobody can open is worse than a step with no file at
// all: it reads as a location, so the next person looks there, finds nothing,
// and reconstructs from the code the mapping the spec was supposed to carry.
var fileExtensions = map[string]bool{
	".go": true, ".md": true, ".sh": true, ".toml": true,
	".yml": true, ".yaml": true, ".json": true, ".txt": true,
}

func TestEveryFileNamedByAnImplementationSpecExists(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	specs, err := filepath.Glob(filepath.Join(root, "docs", "specs", "architecture", "*", "*.i.spec.md"))
	if err != nil || len(specs) == 0 {
		t.Fatalf("no .i specs found (%v); the guard would pass vacuously", err)
	}

	// Backticked paths under a directory the repository actually has. Anything
	// else in backticks is a type, a command or a config key, and this is not
	// the guard for those.
	path := regexp.MustCompile("`((?:internal|cmd|pkg|scripts|docs)/[A-Za-z0-9_./-]+)`")
	checked := 0
	for _, spec := range specs {
		data, err := os.ReadFile(spec)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range path.FindAllStringSubmatch(string(data), -1) {
			named := m[1]
			// Only files, and only by a known extension. A directory named in
			// prose is a location rather than a claim that something is there,
			// and `internal/policy.Glob` is a function.
			if !fileExtensions[filepath.Ext(named)] {
				continue
			}
			checked++
			if _, err := os.Stat(filepath.Join(root, named)); err != nil {
				t.Errorf("%s names %s, which is not in the repository",
					filepath.Base(spec), named)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no path was checked; the guard would pass vacuously")
	}
	t.Logf("%d paths checked across %d specs", checked, len(specs))
}

// Every family that declares invariants must have something claiming them.
//
// Without this, the guard covers whichever families someone remembered. Six of
// the ten were done and the four that were not looked exactly like the six from
// anywhere except a directory listing — which is how "every invariant is
// claimed" gets said about a repository where seventy-five are not.
//
// A guard that has to be remembered per family is not a guard, it is a habit.
func TestEveryFamilyThatDeclaresInvariantsHasAGuard(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	families, err := filepath.Glob(filepath.Join(root, "docs", "specs", "architecture", "*"))
	if err != nil {
		t.Fatal(err)
	}
	guards, err := filepath.Glob(filepath.Join(root, "internal", "*", "invariants_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if len(guards) == 0 {
		t.Fatal("no invariants_test.go anywhere; the check below would report every family")
	}
	var claimed string
	for _, g := range guards {
		data, err := os.ReadFile(g)
		if err != nil {
			t.Fatal(err)
		}
		claimed += string(data)
	}

	checked := 0
	for _, dir := range families {
		if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			continue
		}
		family := filepath.Base(dir)
		// A family with no invariants section declares nothing to claim.
		if _, err := Invariants(root, family); err != nil {
			continue
		}
		checked++
		if !strings.Contains(claimed, `"`+family+`"`) {
			t.Errorf("%s declares invariants and no invariants_test.go names it", family)
		}
	}
	if checked == 0 {
		t.Fatal("no family was checked; the guard would pass vacuously")
	}
	t.Logf("%d families with invariants", checked)
}
