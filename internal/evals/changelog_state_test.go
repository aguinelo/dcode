package evals

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"
)

// The state table's two editions, described once and used both ways: to read
// the numbers the table carries, and to write the tree's numbers into it. A
// reader and a writer kept apart would be two patterns free to drift, and the
// drift would show up the day of a release.

// rowNames are the counted rows, in the order the table shows them.
var rowNames = []string{"families", "changelogs", "declared", "needModel", "byAssertion", "everMeasured", "unverifiable"}

type edition struct {
	rows map[string]int
	// prose holds the two numbers the sentence beside the table carries.
	proseNeedModel, proseNeverRun int
}

// tableRow is one row of the table: a pattern, and which counted number each
// of its groups carries.
type tableRow struct {
	pattern *regexp.Regexp
	groups  map[int]string
}

type editionSpec struct {
	path string
	rows []tableRow
	// prose matches the sentence beside the table: group 1 is the contracts
	// that need a model, group 2 the ones that have never run.
	prose   *regexp.Regexp
	toInt   func(*testing.T, string) int
	toWords func(int) (string, error)
}

func editionSpecs() []editionSpec {
	return []editionSpec{
		{
			path: "CHANGELOG.md",
			rows: []tableRow{
				{regexp.MustCompile(`\| spec families \| (\d+), with (\d+) decision changelogs \|`),
					map[int]string{1: "families", 2: "changelogs"}},
				{regexp.MustCompile(`\| behavioural contracts \| (\d+) declared \|`),
					map[int]string{1: "declared"}},
				{regexp.MustCompile(`\| contracts needing a model \| (\d+) of the (\d+); (\d+) are settled by assertion \|`),
					map[int]string{1: "needModel", 2: "declared", 3: "byAssertion"}},
				{regexp.MustCompile(`\| \*\*contracts ever actually measured\*\* \| \*\*(\d+)\*\* \|`),
					map[int]string{1: "everMeasured"}},
				{regexp.MustCompile(`\| of those, \*\*against a prompt they cannot name\*\* \| \*\*(\d+)\*\* \|`),
					map[int]string{1: "unverifiable"}},
			},
			// "Of the forty-three contracts that need a model, thirty-eight
			// have never run against one"
			prose:   regexp.MustCompile(`Of the\s+([a-z-]+(?:\s+[a-z-]+)?)\s+contracts that need a model, \*\*([a-z-]+(?:\s+[a-z-]+)?) have never run`),
			toInt:   words,
			toWords: englishWords,
		},
		{
			path: "CHANGELOG.pt-BR.md",
			rows: []tableRow{
				{regexp.MustCompile(`\| famílias de spec \| (\d+), com (\d+) changelogs de decisão \|`),
					map[int]string{1: "families", 2: "changelogs"}},
				{regexp.MustCompile(`\| contratos comportamentais \| (\d+) declarados \|`),
					map[int]string{1: "declared"}},
				{regexp.MustCompile(`\| contratos que precisam de modelo \| (\d+) dos (\d+); (\d+) se resolvem por asserção \|`),
					map[int]string{1: "needModel", 2: "declared", 3: "byAssertion"}},
				{regexp.MustCompile(`\| \*\*contratos de fato já medidos\*\* \| \*\*(\d+)\*\* \|`),
					map[int]string{1: "everMeasured"}},
				{regexp.MustCompile(`\| destes, \*\*contra um prompt que não sabem nomear\*\* \| \*\*(\d+)\*\* \|`),
					map[int]string{1: "unverifiable"}},
			},
			// Every letter a number up to ninety-nine is spelled with in
			// Portuguese, in either half: "vinte e três" has to read back.
			prose:   regexp.MustCompile(`Dos\s+([a-zçêô-]+(?:\s+e\s+[a-zçêô-]+)?)\s+contratos que precisam de modelo, \*\*([a-zçêô-]+(?:\s+e\s+[a-zçêô-]+)?) nunca rodaram`),
			toInt:   palavras,
			toWords: portugueseWords,
		},
	}
}

func (s editionSpec) read(t *testing.T, body string) edition {
	t.Helper()
	e := edition{rows: map[string]int{}}
	for _, r := range s.rows {
		m := r.pattern.FindStringSubmatch(body)
		if m == nil {
			t.Fatalf("%s: no state-table row matched %s; the table has drifted from what this guard reads", s.path, r.pattern)
		}
		for group, name := range r.groups {
			n, err := strconv.Atoi(m[group])
			if err != nil {
				t.Fatal(err)
			}
			// "declared" is carried twice, and two rows of one table
			// disagreeing is a reader being told two things.
			if prev, seen := e.rows[name]; seen && prev != n {
				t.Errorf("%s: the table says %s = %d in one row and %d in another", s.path, name, prev, n)
			}
			e.rows[name] = n
		}
	}
	m := s.prose.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("%s: the sentence beside the table did not match; this guard is reading nothing", s.path)
	}
	e.proseNeedModel = s.toInt(t, m[1])
	e.proseNeverRun = s.toInt(t, m[2])
	return e
}

// write puts the tree's numbers into the table and the sentence beside it,
// and changes nothing else.
func (s editionSpec) write(body string, c counted) (string, error) {
	for _, r := range s.rows {
		idx := r.pattern.FindStringSubmatchIndex(body)
		if idx == nil {
			return "", fmt.Errorf("no row matched %s", r.pattern)
		}
		repl := map[int]string{}
		for group, name := range r.groups {
			repl[group] = strconv.Itoa(c[name])
		}
		body = replaceGroups(body, idx, repl)
	}
	idx := s.prose.FindStringSubmatchIndex(body)
	if idx == nil {
		return "", fmt.Errorf("the sentence beside the table did not match %s", s.prose)
	}
	need, err := s.toWords(c["needModel"])
	if err != nil {
		return "", err
	}
	never, err := s.toWords(c["needModel"] - c["everMeasured"])
	if err != nil {
		return "", err
	}
	return replaceGroups(body, idx, map[int]string{1: need, 2: never}), nil
}

// replaceGroups replaces submatches by index, from the last to the first so
// that an earlier offset never moves under a later one.
func replaceGroups(body string, idx []int, repl map[int]string) string {
	for group := len(idx)/2 - 1; group >= 1; group-- {
		text, ok := repl[group]
		if !ok || idx[2*group] < 0 {
			continue
		}
		body = body[:idx[2*group]] + text + body[idx[2*group+1]:]
	}
	return body
}

var (
	enUnits = []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine",
		"ten", "eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen", "seventeen", "eighteen", "nineteen"}
	enTens  = []string{"", "", "twenty", "thirty", "forty", "fifty", "sixty", "seventy", "eighty", "ninety"}
	ptUnits = []string{"zero", "um", "dois", "três", "quatro", "cinco", "seis", "sete", "oito", "nove",
		"dez", "onze", "doze", "treze", "catorze", "quinze", "dezesseis", "dezessete", "dezoito", "dezenove"}
	ptTens = []string{"", "", "vinte", "trinta", "quarenta", "cinquenta", "sessenta", "setenta", "oitenta", "noventa"}
)

// englishWords spells 0 to 99 the way the prose does. Past that it refuses:
// a number the writer cannot spell is a sentence somebody has to rewrite, and
// writing digits into it would read back as nothing.
func englishWords(n int) (string, error) {
	switch {
	case n < 0 || n > 99:
		return "", fmt.Errorf("cannot spell %d; the sentence beside the table needs rewriting by hand", n)
	case n < 20:
		return enUnits[n], nil
	case n%10 == 0:
		return enTens[n/10], nil
	}
	return enTens[n/10] + "-" + enUnits[n%10], nil
}

func portugueseWords(n int) (string, error) {
	switch {
	case n < 0 || n > 99:
		return "", fmt.Errorf("não sei escrever %d; a frase ao lado da tabela tem de ser reescrita à mão", n)
	case n < 20:
		return ptUnits[n], nil
	case n%10 == 0:
		return ptTens[n/10], nil
	}
	return ptTens[n/10] + " e " + ptUnits[n%10], nil
}

// Every number the prose can carry reads back as itself, in both languages.
func TestTheWriterSpellsWhatTheReaderReads(t *testing.T) {
	for n := 0; n <= 99; n++ {
		en, err := englishWords(n)
		if err != nil {
			t.Fatal(err)
		}
		if got := words(t, en); got != n {
			t.Errorf("%d spelled %q reads back as %d", n, en, got)
		}
		pt, err := portugueseWords(n)
		if err != nil {
			t.Fatal(err)
		}
		if got := palavras(t, pt); got != n {
			t.Errorf("%d escrito %q volta como %d", n, pt, got)
		}
	}
	// And the sentence the writer leaves is one the reader matches.
	sentence := map[string]string{
		"CHANGELOG.md":       "Of the %s contracts that need a model, **%s have never run",
		"CHANGELOG.pt-BR.md": "Dos %s contratos que precisam de modelo, **%s nunca rodaram",
	}
	for _, spec := range editionSpecs() {
		for n := 0; n <= 99; n++ {
			w, _ := spec.toWords(n)
			s := fmt.Sprintf(sentence[spec.path], w, w)
			m := spec.prose.FindStringSubmatch(s)
			if m == nil || spec.toInt(t, m[1]) != n || spec.toInt(t, m[2]) != n {
				t.Errorf("%s: the writer's %q does not read back as %d", spec.path, s, n)
			}
		}
	}
	if _, err := englishWords(100); err == nil {
		t.Error("100 was spelled; the reader cannot read it back")
	}
}
