package evals

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// contractRow matches a behavioural contract row in a `.p.spec.md` table.
//
// A contract row is the one that ends in a fixture path — that column is what
// distinguishes it from every other table in the file, and it is also the exact
// promise being checked.
var contractRow = regexp.MustCompile("(?m)^\\| `([a-z][a-z0-9-]*)` \\|.*`testdata/evals/([a-z][a-z0-9-]*)/`")

// The guard that stops this debt coming back.
//
// Fifteen contracts were declared with a fixture path and no fixture, some of
// them for months: behavior-definition named ten and agent-loop five before
// there was a harness to run any of them. Nothing failed, because nothing
// compared the two.
//
// A threshold with no material is not a weak measurement — it is no measurement
// at all, wearing the appearance of one. Anyone reading the spec sees "≥ 95%,
// testdata/evals/tool-over-shell/" and reasonably concludes it is being
// measured.
func TestEveryDeclaredContractHasItsFixture(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	specs, err := filepath.Glob(filepath.Join(root, "docs", "specs", "architecture", "*", "*.p.spec.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) == 0 {
		t.Fatal("no .p specs found; the guard would pass vacuously")
	}

	found := 0
	for _, spec := range specs {
		data, err := os.ReadFile(spec)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range contractRow.FindAllStringSubmatch(string(data), -1) {
			id, path := m[1], m[2]
			found++
			if id != path {
				t.Errorf("%s: contract %q points at testdata/evals/%s/ — the id and the fixture must be the same name, or neither can be found from the other",
					filepath.Base(spec), id, path)
				continue
			}
			if _, err := LoadFixture(FixtureRoot, id); err != nil {
				t.Errorf("%s declares %q with a fixture path and the fixture does not load: %v",
					filepath.Base(spec), id, err)
			}
		}
	}
	if found == 0 {
		t.Fatal("no contract rows matched; the pattern has drifted from the table format and the guard is measuring nothing")
	}
	t.Logf("%d declared contracts, all with material", found)
}

// Every fixture must explain itself. A scenario directory with a task and a
// tool set but no note is material nobody can review — and reviewing the
// material is the only defence against a contract that measures the wrong
// thing.
func TestEveryFixtureExplainsWhatItMeasures(t *testing.T) {
	entries, err := os.ReadDir(FixtureRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		note, err := os.ReadFile(filepath.Join(FixtureRoot, e.Name(), "scenario.md"))
		if err != nil {
			t.Errorf("%s has no scenario.md: material nobody can review is material nobody checks", e.Name())
			continue
		}
		body := string(note)
		if !strings.Contains(body, e.Name()) {
			t.Errorf("%s: the note does not name the scenario it belongs to", e.Name())
		}
		if !strings.Contains(body, "limiar") {
			t.Errorf("%s: the note does not state the threshold, so a reader cannot tell what passing means", e.Name())
		}
	}
}
