package evals

import (
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

type edition struct {
	path string
	// families, changelogs, declared, needModel, byAssertion, everMeasured
	rows map[string]int
	// prose holds the two numbers the sentence beside the table carries.
	proseNeedModel, proseNeverRun int
}

func TestTheStateTableIsCountedAndNotCarried(t *testing.T) {
	root := repoRootFromEvals(t)

	wantFamilies := countDirs(t, filepath.Join(root, "docs", "specs", "architecture"))
	wantChangelogs := countChangelogs(t, filepath.Join(root, "docs", "specs", "architecture"))
	wantDeclared := len(Contracts)
	wantNeedModel := Measurable(Contracts)
	wantByAssertion := wantDeclared - wantNeedModel
	// Sound ones. A run that lost an execution to a transport error measured
	// nineteen things and reported a rate over twenty, and counting it here
	// would put a number in the table that the record itself calls unreadable.
	wantMeasured := 0
	for _, m := range Measured {
		if m.Sound {
			wantMeasured++
		}
	}
	// A measurement that cannot say which prompt it saw. Counted here for the
	// same reason "ever actually measured" is: the distance between a number
	// and what it describes only stays visible while something counts it.
	wantUnverifiable := unverifiable(Measured)

	for _, e := range []edition{readEnglish(t, root), readPortuguese(t, root)} {
		t.Run(filepath.Base(e.path), func(t *testing.T) {
			for name, want := range map[string]int{
				"families":     wantFamilies,
				"changelogs":   wantChangelogs,
				"declared":     wantDeclared,
				"needModel":    wantNeedModel,
				"byAssertion":  wantByAssertion,
				"everMeasured": wantMeasured,
				"unverifiable": wantUnverifiable,
			} {
				got, ok := e.rows[name]
				if !ok {
					t.Errorf("the state table carries no %s row; the pattern has drifted and this guard is reading nothing", name)
					continue
				}
				if got != want {
					t.Errorf("the state table says %s = %d and the tree says %d", name, got, want)
				}
			}

			// The sentence beside the table has to agree with the table.
			// Prose is the weakest layer this repository recognises, and the
			// half that went stale first was this one.
			if e.proseNeedModel != wantNeedModel {
				t.Errorf("the prose says %d contracts need a model and the table says %d",
					e.proseNeedModel, wantNeedModel)
			}
			if want := wantNeedModel - wantMeasured; e.proseNeverRun != want {
				t.Errorf("the prose says %d have never run and the numbers give %d",
					e.proseNeverRun, want)
			}
		})
	}
}

// The two editions are one document in two languages, and a number that
// differs between them is a reader being told two things.
func TestBothEditionsCarryTheSameNumbers(t *testing.T) {
	root := repoRootFromEvals(t)
	en, pt := readEnglish(t, root), readPortuguese(t, root)

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

// ---- reading the two editions ----

func readEnglish(t *testing.T, root string) edition {
	t.Helper()
	body := readFile(t, filepath.Join(root, "CHANGELOG.md"))
	e := edition{path: "CHANGELOG.md", rows: map[string]int{}}
	e.rows["families"] = row(t, body, `\| spec families \| (\d+), with (\d+) decision changelogs \|`, 1)
	e.rows["changelogs"] = row(t, body, `\| spec families \| (\d+), with (\d+) decision changelogs \|`, 2)
	e.rows["declared"] = row(t, body, `\| behavioural contracts \| (\d+) declared \|`, 1)
	e.rows["needModel"] = row(t, body, `\| contracts needing a model \| (\d+) of the (\d+); (\d+) are settled by assertion \|`, 1)
	e.rows["byAssertion"] = row(t, body, `\| contracts needing a model \| (\d+) of the (\d+); (\d+) are settled by assertion \|`, 3)
	e.rows["everMeasured"] = row(t, body, `\| \*\*contracts ever actually measured\*\* \| \*\*(\d+)\*\* \|`, 1)
	e.rows["unverifiable"] = row(t, body,
		`\| of those, \*\*against a prompt they cannot name\*\* \| \*\*(\d+)\*\* \|`, 1)

	// "Of the forty-three contracts that need a model, thirty-eight have never
	// run against one"
	m := regexp.MustCompile(`Of the\s+([a-z-]+(?:\s+[a-z-]+)?)\s+contracts that need a model, \*\*([a-z-]+(?:\s+[a-z-]+)?) have never run`).FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("CHANGELOG.md: the sentence beside the table did not match; this guard is reading nothing")
	}
	e.proseNeedModel = words(t, m[1])
	e.proseNeverRun = words(t, m[2])
	return e
}

func readPortuguese(t *testing.T, root string) edition {
	t.Helper()
	body := readFile(t, filepath.Join(root, "CHANGELOG.pt-BR.md"))
	e := edition{path: "CHANGELOG.pt-BR.md", rows: map[string]int{}}
	e.rows["families"] = row(t, body, `\| famílias de spec \| (\d+), com (\d+) changelogs de decisão \|`, 1)
	e.rows["changelogs"] = row(t, body, `\| famílias de spec \| (\d+), com (\d+) changelogs de decisão \|`, 2)
	e.rows["declared"] = row(t, body, `\| contratos comportamentais \| (\d+) declarados \|`, 1)
	e.rows["needModel"] = row(t, body, `\| contratos que precisam de modelo \| (\d+) dos (\d+); (\d+) se resolvem por asserção \|`, 1)
	e.rows["byAssertion"] = row(t, body, `\| contratos que precisam de modelo \| (\d+) dos (\d+); (\d+) se resolvem por asserção \|`, 3)
	e.rows["everMeasured"] = row(t, body, `\| \*\*contratos de fato já medidos\*\* \| \*\*(\d+)\*\* \|`, 1)
	e.rows["unverifiable"] = row(t, body,
		`\| destes, \*\*contra um prompt que não sabem nomear\*\* \| \*\*(\d+)\*\* \|`, 1)

	m := regexp.MustCompile(`Dos\s+([a-zç-]+(?:\s+e\s+[a-zê-]+)?)\s+contratos que precisam de modelo, \*\*([a-zê-]+(?:\s+e\s+[a-zô-]+)?) nunca rodaram`).FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("CHANGELOG.pt-BR.md: the sentence beside the table did not match; this guard is reading nothing")
	}
	e.proseNeedModel = palavras(t, m[1])
	e.proseNeverRun = palavras(t, m[2])
	return e
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
