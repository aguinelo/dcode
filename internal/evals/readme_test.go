package evals

import (
	"math"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The README is the first thing anyone reads about this product and it is the
// document that went stale hardest.
//
// On 31 August 2026 it still said "there is no TUI yet and no released binary;
// build it yourself with `make build`", four months and seventeen minor
// versions after both stopped being true. Its badge said ten specs against
// eighteen families, and its testing section claimed a 95% coverage gate
// against a script that has always read 90.
//
// The changelog beside it was correct the whole time, because a guard counts
// it. This one exists so the same is true of the front page: the badges, the
// sentence under them, and the receipts table are read from the tree and from
// `Measured`, in both language editions.
//
// What is NOT checked, deliberately, and for the same reasons as the changelog
// guard: the release badge (it is a shields.io endpoint that reads the tag, so
// it cannot go stale) and the coverage number against an actual run. Coverage
// is checked only for agreement with the changelog — two documents disagreeing
// about it is a real failure and a cheap one to catch, and pretending to
// verify the number itself would be worse than saying plainly that this test
// does not run the gate.

type readme struct {
	path               string
	families           int
	declared, measured int
	proseDeclared      int
	proseMeasured      int
	proseNeverMeasured int
	specRows           int
	coverage           string
	receipts           []receipt
}

type receipt struct {
	id   string
	rate int
	runs int
}

func TestTheReadmeBadgesAreCountedAndNotCarried(t *testing.T) {
	root := repoRootFromEvals(t)

	wantFamilies := countDirs(t, filepath.Join(root, "docs", "specs", "architecture"))
	wantDeclared := len(Contracts)
	wantNeedModel := Measurable(Contracts)
	wantMeasured := 0
	for _, m := range Measured {
		if m.Sound {
			wantMeasured++
		}
	}

	for _, r := range []readme{readReadmeEnglish(t, root), readReadmePortuguese(t, root)} {
		t.Run(filepath.Base(r.path), func(t *testing.T) {
			if r.families != wantFamilies {
				t.Errorf("the specs badge says %d families and the tree has %d", r.families, wantFamilies)
			}
			if r.specRows != wantFamilies {
				t.Errorf("the specs table lists %d families and the tree has %d", r.specRows, wantFamilies)
			}
			if r.declared != wantDeclared {
				t.Errorf("the contracts badge says %d declared and there are %d", r.declared, wantDeclared)
			}
			if r.measured != wantMeasured {
				t.Errorf("the contracts badge says %d measured and %d have been", r.measured, wantMeasured)
			}

			// The sentence under the badges repeats both numbers, and prose is
			// the weakest layer this repository recognises.
			if r.proseDeclared != wantDeclared || r.proseMeasured != wantMeasured {
				t.Errorf("the sentence under the badges says %d declared and %d measured; the badge says %d and %d",
					r.proseDeclared, r.proseMeasured, r.declared, r.measured)
			}

			// "The 35 contracts never run" is the badge's own subtraction, and
			// it is the number a reader carries away.
			if want := wantNeedModel - wantMeasured; r.proseNeverMeasured != want {
				t.Errorf("the README says %d contracts have never run and the numbers give %d",
					r.proseNeverMeasured, want)
			}
		})
	}
}

// Every rate in the receipts table is a measurement that was actually taken.
// A number in that table with no measurement behind it is the whole thesis of
// the front page failing on the front page.
func TestEveryReceiptNamesAMeasurement(t *testing.T) {
	root := repoRootFromEvals(t)

	for _, r := range []readme{readReadmeEnglish(t, root), readReadmePortuguese(t, root)} {
		t.Run(filepath.Base(r.path), func(t *testing.T) {
			if len(r.receipts) == 0 {
				t.Fatal("the receipts table is empty; this guard would pass reading nothing")
			}
			for _, got := range r.receipts {
				var found *Measurement
				for i := range Measured {
					if Measured[i].ID == got.id {
						found = &Measured[i]
						break
					}
				}
				if found == nil {
					t.Errorf("the receipts table reports %s and nothing was ever measured for it", got.id)
					continue
				}
				if want := int(math.Round(found.Rate * 100)); got.rate != want {
					t.Errorf("%s: the README says %d%% and the measurement says %d%%", got.id, got.rate, want)
				}
				if got.runs != found.Runs {
					t.Errorf("%s: the README says %d runs and the measurement says %d", got.id, got.runs, found.Runs)
				}
			}
		})
	}
}

// Coverage is measured by a run this test does not do, so the most it can say
// is that the two documents claiming it agree. They disagreed for months: the
// README said the gate was 95% while the script and the changelog said 90.
func TestTheReadmeAndTheChangelogAgreeOnCoverage(t *testing.T) {
	root := repoRootFromEvals(t)

	for _, pair := range []struct {
		readme, changelog string
		pattern           string
	}{
		{"README.md", "CHANGELOG.md", `\| coverage \| ([\d.,]+)%`},
		{"README.pt-BR.md", "CHANGELOG.pt-BR.md", `\| cobertura \| ([\d.,]+)%`},
	} {
		t.Run(pair.readme, func(t *testing.T) {
			body := readFile(t, filepath.Join(root, pair.changelog))
			m := regexp.MustCompile(pair.pattern).FindStringSubmatch(body)
			if m == nil {
				t.Fatalf("%s: no coverage row matched; this guard is reading nothing", pair.changelog)
			}
			var badge string
			if pair.readme == "README.md" {
				badge = readReadmeEnglish(t, root).coverage
			} else {
				badge = readReadmePortuguese(t, root).coverage
			}
			// One edition writes 93.4 and the other 93,4. They are the same
			// number and the separator is not the thing under test.
			if decimal(badge) != decimal(m[1]) {
				t.Errorf("%s says coverage is %s and %s says %s", pair.readme, badge, pair.changelog, m[1])
			}
		})
	}
}

// ---- reading the two editions ----

func readReadmeEnglish(t *testing.T, root string) readme {
	t.Helper()
	body := readFile(t, filepath.Join(root, "README.md"))
	r := readme{path: "README.md"}
	r.families = row(t, body, `badge/specs-(\d+)%20families`, 1)
	r.declared = row(t, body, `badge/contracts-(\d+)%20measured%20%2F%20(\d+)%20declared`, 2)
	r.measured = row(t, body, `badge/contracts-(\d+)%20measured%20%2F%20(\d+)%20declared`, 1)
	r.proseDeclared = row(t, body, `\*\*(\d+) contracts declared, (\d+) actually measured\.\*\*`, 1)
	r.proseMeasured = row(t, body, `\*\*(\d+) contracts declared, (\d+) actually measured\.\*\*`, 2)
	r.proseNeverMeasured = row(t, body, `The (\d+) contracts never run`, 1)
	r.coverage = text(t, body, `badge/coverage-([\d.,%C]+?)%25`, 1)
	r.specRows = specTableRows(body)
	r.receipts = receipts(t, body)
	return r
}

func readReadmePortuguese(t *testing.T, root string) readme {
	t.Helper()
	body := readFile(t, filepath.Join(root, "README.pt-BR.md"))
	r := readme{path: "README.pt-BR.md"}
	r.families = row(t, body, `badge/specs-(\d+)%20fam%C3%ADlias`, 1)
	r.declared = row(t, body, `badge/contratos-(\d+)%20medidos%20%2F%20(\d+)%20declarados`, 2)
	r.measured = row(t, body, `badge/contratos-(\d+)%20medidos%20%2F%20(\d+)%20declarados`, 1)
	r.proseDeclared = row(t, body, `\*\*(\d+) contratos declarados, (\d+) de fato medidos\.\*\*`, 1)
	r.proseMeasured = row(t, body, `\*\*(\d+) contratos declarados, (\d+) de fato medidos\.\*\*`, 2)
	r.proseNeverMeasured = row(t, body, `Os (\d+) contratos que nunca rodaram`, 1)
	r.coverage = text(t, body, `badge/cobertura-([\d.,%C]+?)%25`, 1)
	r.specRows = specTableRows(body)
	r.receipts = receipts(t, body)
	return r
}

// A row of the specs table is a link into a family's directory, which is the
// same thing countDirs counts.
var specRow = regexp.MustCompile(`(?m)^\| \[[a-z-]+\]\(docs/specs/architecture/[a-z-]+/\) \|`)

func specTableRows(body string) int { return len(specRow.FindAllString(body, -1)) }

// | `contract-id` | 98% | 50 | MiniMax-M3 |, bold or not.
var receiptRow = regexp.MustCompile("(?m)^\\| \\*{0,2}`([a-z-]+)`\\*{0,2} \\| \\*{0,2}(\\d+)%\\*{0,2} \\| (\\d+) \\|")

func receipts(t *testing.T, body string) []receipt {
	t.Helper()
	var out []receipt
	for _, m := range receiptRow.FindAllStringSubmatch(body, -1) {
		rate, err := strconv.Atoi(m[2])
		if err != nil {
			t.Fatal(err)
		}
		runs, err := strconv.Atoi(m[3])
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, receipt{id: m[1], rate: rate, runs: runs})
	}
	return out
}

func text(t *testing.T, body, pattern string, group int) string {
	t.Helper()
	m := regexp.MustCompile(pattern).FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("nothing matched %s; the README has drifted from what this guard reads", pattern)
	}
	return m[group]
}

// decimal reads "93.4" and "93,4" and "93%2C4" as the same number.
func decimal(s string) string {
	s = strings.ReplaceAll(s, "%2C", ".")
	return strings.ReplaceAll(s, ",", ".")
}
