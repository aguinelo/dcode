package evals

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

// The state table at the top of the changelog is the first thing anyone reads
// about this product, and every number in it used to be typed by hand.
//
// Three of them were wrong at once. "4 contracts measured against a model" was
// carried from the release before while a fifth had been measured since. The
// sentence beside it said "thirty-nine of the forty-two", from a tree that had
// forty-two contracts in it rather than forty-eight. And the two language
// editions could disagree with each other without anything noticing.
//
// It is the defect this repository keeps finding in itself — a value copied
// from a truth that moves — sitting in the document that exists to prevent it,
// which is why the numbers are counted here rather than trusted.
//
// What is NOT checked: coverage and the published version. Coverage is
// measured by a run this test does not do, and the version is a claim about a
// tag that a test with no network cannot see. Naming them here is better than
// a check that pretends.

// What changed with fragments (2026-10-17): the table is counted by the
// release step, not by every pull request. Before, each pull request that
// added a decision changelog edited the same row of both editions, and fifteen
// of them in flight conflicted on it at once. Now the row holds the count taken
// at the last release, and this test holds it to the tree in two ways:
//
//   - with no fragment pending — the state the release step leaves, before its
//     commit can pass CI and be tagged — the table must equal the tree, row by
//     row, exactly as before;
//   - always, the table, rewritten from the tree by this file's own writer,
//     must read back as the tree: the release step cannot be the one that
//     fails to count. And the prose must agree with its own table, the two
//     editions with each other.
//
// Between releases a table behind the tree is said in the test's log, never
// passed over: the drift is what the next release recounts.
//
// `go test ./internal/evals -run TestTheStateTableIsCountedAndNotCarried
// -args -write-state` rewrites both editions from the tree; it is what
// `scripts/changelog-assemble.sh release` runs.
var writeState = flag.Bool("write-state", false,
	"rewrite the state table of both CHANGELOG editions from the tree (the release step)")

// counted is what the tree says, keyed as the table rows are.
type counted map[string]int

func countTree(t *testing.T, root string) counted {
	t.Helper()
	declared := len(Contracts)
	needModel := Measurable(Contracts)
	// Sound ones. A run that lost an execution to a transport error measured
	// nineteen things and reported a rate over twenty, and counting it here
	// would put a number in the table that the record itself calls unreadable.
	measured := 0
	for _, m := range Measured {
		if m.Sound {
			measured++
		}
	}
	return counted{
		"families":     countDirs(t, filepath.Join(root, "docs", "specs", "architecture")),
		"changelogs":   countChangelogs(t, filepath.Join(root, "docs", "specs", "architecture")),
		"declared":     declared,
		"needModel":    needModel,
		"byAssertion":  declared - needModel,
		"everMeasured": measured,
		// A measurement that cannot say which prompt it saw. Counted here for
		// the same reason "ever actually measured" is: the distance between a
		// number and what it describes only stays visible while something
		// counts it.
		"unverifiable": unverifiable(Measured),
	}
}

func TestTheStateTableIsCountedAndNotCarried(t *testing.T) {
	root := repoRootFromEvals(t)
	want := countTree(t, root)
	specs := editionSpecs()

	if *writeState {
		for _, spec := range specs {
			path := filepath.Join(root, spec.path)
			body, err := spec.write(readFile(t, path), want)
			if err != nil {
				t.Fatalf("%s: %v", spec.path, err)
			}
			if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			t.Logf("%s: state table rewritten from the tree", spec.path)
		}
	}

	pending := len(readFragments(t, filepath.Join(root, "changelog.d")))

	for _, spec := range specs {
		t.Run(filepath.Base(spec.path), func(t *testing.T) {
			body := readFile(t, filepath.Join(root, spec.path))
			e := spec.read(t, body)

			for _, name := range rowNames {
				got, ok := e.rows[name]
				if !ok {
					t.Errorf("the state table carries no %s row; the pattern has drifted and this guard is reading nothing", name)
					continue
				}
				if got == want[name] {
					continue
				}
				if pending == 0 {
					t.Errorf("the state table says %s = %d and the tree says %d", name, got, want[name])
				} else {
					t.Logf("counted now: %s = %d, the table counted %d at the last release; the release recounts it", name, want[name], got)
				}
			}

			// The sentence beside the table has to agree with the table.
			// Prose is the weakest layer this repository recognises, and the
			// half that went stale first was this one.
			if e.proseNeedModel != e.rows["needModel"] {
				t.Errorf("the prose says %d contracts need a model and the table says %d",
					e.proseNeedModel, e.rows["needModel"])
			}
			if w := e.rows["needModel"] - e.rows["everMeasured"]; e.proseNeverRun != w {
				t.Errorf("the prose says %d have never run and the table gives %d",
					e.proseNeverRun, w)
			}

			// The writer the release runs, run here on every change: the table
			// it would write has to read back as the tree.
			rewritten, err := spec.write(body, want)
			if err != nil {
				t.Fatalf("the release step could not rewrite this table: %v", err)
			}
			back := spec.read(t, rewritten)
			for _, name := range rowNames {
				if back.rows[name] != want[name] {
					t.Errorf("rewritten from the tree, the table reads %s = %d where the tree says %d",
						name, back.rows[name], want[name])
				}
			}
			if back.proseNeedModel != want["needModel"] || back.proseNeverRun != want["needModel"]-want["everMeasured"] {
				t.Errorf("rewritten from the tree, the prose reads (%d, %d) where the tree gives (%d, %d)",
					back.proseNeedModel, back.proseNeverRun, want["needModel"], want["needModel"]-want["everMeasured"])
			}
		})
	}
}

// The two editions are one document in two languages, and a number that
// differs between them is a reader being told two things.
func TestBothEditionsCarryTheSameNumbers(t *testing.T) {
	root := repoRootFromEvals(t)
	specs := editionSpecs()
	en := specs[0].read(t, readFile(t, filepath.Join(root, specs[0].path)))
	pt := specs[1].read(t, readFile(t, filepath.Join(root, specs[1].path)))

	for name, a := range en.rows {
		b, ok := pt.rows[name]
		if !ok {
			t.Errorf("the English table has %s and the Portuguese one does not", name)
			continue
		}
		if a != b {
			t.Errorf("%s: English says %d, Portuguese says %d", name, a, b)
		}
	}
	if en.proseNeedModel != pt.proseNeedModel || en.proseNeverRun != pt.proseNeverRun {
		t.Errorf("the sentence beside the table disagrees between editions: en=(%d,%d) pt=(%d,%d)",
			en.proseNeedModel, en.proseNeverRun, pt.proseNeedModel, pt.proseNeverRun)
	}
}

// A measurement names a contract, a model and a date, or it is not a
// measurement — it is a number with no way back to what produced it.
func TestEveryMeasurementNamesWhatProducedIt(t *testing.T) {
	if len(Measured) == 0 {
		t.Fatal("no measurements recorded; the guard would pass vacuously")
	}
	date := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	seen := map[string]bool{}
	for _, m := range Measured {
		c, ok := ContractByID(m.ID)
		if !ok {
			t.Errorf("%s was measured and no contract declares it", m.ID)
			continue
		}
		// A contract settled by assertion is not measured against a model, so
		// a measurement of one is a category error rather than a fact.
		if !c.Measured() {
			t.Errorf("%s is settled by assertion and carries a measurement; one of the two is wrong", m.ID)
		}
		if m.Model == "" {
			t.Errorf("%s was measured against nothing named; a threshold measured against one model says nothing about another", m.ID)
		}
		if !date.MatchString(m.Date) {
			t.Errorf("%s has date %q, want YYYY-MM-DD", m.ID, m.Date)
		}
		if m.Runs <= 0 {
			t.Errorf("%s reports a rate over %d runs; a rate with no denominator is not a number", m.ID, m.Runs)
		}
		if m.Rate < 0 || m.Rate > 1 {
			t.Errorf("%s reports rate %v, which is not a fraction", m.ID, m.Rate)
		}
		if !m.Sound && m.Note == "" {
			t.Errorf("%s is recorded as unsound and does not say why", m.ID)
		}
		if seen[m.ID] {
			t.Errorf("%s is recorded twice; the later measurement replaces the earlier one rather than joining it", m.ID)
		}
		seen[m.ID] = true
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func row(t *testing.T, body, pattern string, group int) int {
	t.Helper()
	m := regexp.MustCompile(pattern).FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no state-table row matched %s; the table has drifted from what this guard reads", pattern)
	}
	n, err := strconv.Atoi(m[group])
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// words turns "forty-three" into 43. Only the range the table can reach, and
// an unknown word fails rather than defaulting: a number silently read as zero
// is how a guard passes while measuring nothing.
func words(t *testing.T, s string) int {
	t.Helper()
	units := map[string]int{
		"zero": 0, "one": 1, "two": 2, "three": 3, "four": 4, "five": 5,
		"six": 6, "seven": 7, "eight": 8, "nine": 9, "ten": 10,
		"eleven": 11, "twelve": 12, "thirteen": 13, "fourteen": 14, "fifteen": 15,
		"sixteen": 16, "seventeen": 17, "eighteen": 18, "nineteen": 19,
	}
	tens := map[string]int{
		"twenty": 20, "thirty": 30, "forty": 40, "fifty": 50,
		"sixty": 60, "seventy": 70, "eighty": 80, "ninety": 90,
	}
	total, ok := 0, false
	// Split on the hyphen and on any whitespace, newline included: the sentence
	// is wrapped prose, and "forty-\nthree" is the same number as "forty-three".
	for _, part := range strings.FieldsFunc(strings.ToLower(strings.TrimSpace(s)), func(r rune) bool {
		return r == '-' || unicode.IsSpace(r)
	}) {
		if v, is := tens[part]; is {
			total += v
			ok = true
			continue
		}
		if v, is := units[part]; is {
			total += v
			ok = true
			continue
		}
		t.Fatalf("cannot read %q as a number in %q", part, s)
	}
	if !ok {
		t.Fatalf("cannot read %q as a number", s)
	}
	return total
}

func palavras(t *testing.T, s string) int {
	t.Helper()
	unidades := map[string]int{
		"zero": 0, "um": 1, "dois": 2, "três": 3, "quatro": 4, "cinco": 5,
		"seis": 6, "sete": 7, "oito": 8, "nove": 9, "dez": 10,
		"onze": 11, "doze": 12, "treze": 13, "catorze": 14, "quatorze": 14,
		"quinze": 15, "dezesseis": 16, "dezessete": 17, "dezoito": 18, "dezenove": 19,
	}
	dezenas := map[string]int{
		"vinte": 20, "trinta": 30, "quarenta": 40, "cinquenta": 50,
		"sessenta": 60, "setenta": 70, "oitenta": 80, "noventa": 90,
	}
	total, ok := 0, false
	for _, part := range strings.FieldsFunc(strings.ToLower(strings.TrimSpace(s)), func(r rune) bool {
		return r == '-' || unicode.IsSpace(r)
	}) {
		if part == "e" {
			continue
		}
		if v, is := dezenas[part]; is {
			total += v
			ok = true
			continue
		}
		if v, is := unidades[part]; is {
			total += v
			ok = true
			continue
		}
		t.Fatalf("não sei ler %q como número em %q", part, s)
	}
	if !ok {
		t.Fatalf("não sei ler %q como número", s)
	}
	return total
}

// ---- counting the tree ----

func repoRootFromEvals(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func countDirs(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() {
			n++
		}
	}
	if n == 0 {
		t.Fatalf("no spec families under %s; the guard would compare zero to zero", dir)
	}
	return n
}

func countChangelogs(t *testing.T, dir string) int {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "*", "changelog", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatal(fmt.Sprintf("no decision changelogs under %s; the guard would compare zero to zero", dir))
	}
	return len(matches)
}
