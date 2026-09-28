// Package specguard checks that every invariant a spec declares is claimed by a
// test that exists.
//
// Section 8 of each `.p.spec.md` lists the package's invariants and the `.i`
// demands "um teste por linha". Nothing checked that, so an invariant could be
// written, reviewed and merged while being asserted by nothing — the same shape
// as a config key nobody reads or a threshold nobody measures.
//
// What this does NOT do is judge whether the named test really covers the line;
// that is a reading a person does once, at review. What it does is make the
// claim explicit and keep it from rotting: rename the test and this goes red,
// add an invariant and this goes red until someone names its test, remove one
// and this goes red until its claim is removed too.
package specguard

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Findings are the problems in one family: the invariants in spec order, then
// the fragments that claim nothing, sorted.
type Findings []string

// Check reports every invariant of family that no fragment claims or that more
// than one does, every claim naming a test that does not exist, and every
// fragment that claims no invariant at all.
//
// specRoot is the repository root; testDirs are the directories whose
// `_test.go` files are searched. Several, because a spec family is not a Go
// package: the configuration family's invariants about credentials are asserted
// in internal/credential, and the protocol family's invariants about approval
// events are asserted where the events are emitted. Listing the directories
// keeps that visible instead of letting an invariant read as unclaimed because
// its test sits one package over.
//
// mapping keys are fragments matched against the invariant line — a fragment,
// not the whole line, because the lines carry markup and rule references that
// churn without the invariant changing. A line takes exactly one fragment. Two
// on one line used to be settled by map order: the line went to whichever the
// range reached first, and the other's test was never looked for on that run.
func Check(specRoot, family string, testDirs []string, mapping map[string]string) (Findings, error) {
	lines, err := Invariants(specRoot, family)
	if err != nil {
		return nil, err
	}
	var src string
	for _, dir := range testDirs {
		part, err := testSource(dir)
		if err != nil {
			return nil, err
		}
		src += part
	}

	var out Findings
	used := map[string]bool{}
	for _, line := range lines {
		frags := claims(line, mapping)
		switch len(frags) {
		case 0:
			out = append(out, fmt.Sprintf("no test claims this invariant:\n  %s", short(line)))
		case 1:
		default:
			var who strings.Builder
			for _, frag := range frags {
				fmt.Fprintf(&who, "\n  %q → %s", frag, mapping[frag])
			}
			out = append(out, fmt.Sprintf("the invariant\n  %s\nis claimed by %d fragments, and takes exactly one:%s",
				short(line), len(frags), who.String()))
		}
		for _, frag := range frags {
			used[frag] = true
			name := mapping[frag]
			if !regexp.MustCompile(`func ` + regexp.QuoteMeta(name) + `\b`).MatchString(src) {
				out = append(out, fmt.Sprintf("the invariant\n  %s\nnames %s, which does not exist", short(line), name))
			}
		}
	}
	var unused []string
	for frag := range mapping {
		if !used[frag] {
			unused = append(unused, frag)
		}
	}
	slices.Sort(unused)
	for _, frag := range unused {
		out = append(out, fmt.Sprintf("the fragment %q is in no invariant, so its claim for %s is on nothing",
			frag, mapping[frag]))
	}
	return out, nil
}

// Invariants reads the invariant lines of a family's `.p` spec.
func Invariants(specRoot, family string) ([]string, error) {
	specs, err := filepath.Glob(filepath.Join(specRoot, "docs", "specs", "architecture", family, "*.p.spec.md"))
	if err != nil {
		return nil, err
	}
	if len(specs) == 0 {
		return nil, fmt.Errorf("no .p spec for %s", family)
	}
	data, err := os.ReadFile(specs[0])
	if err != nil {
		return nil, err
	}

	body := string(data)
	head := regexp.MustCompile(`(?m)^## \d+\. Invariantes verificáveis\s*$`)
	loc := head.FindStringIndex(body)
	if loc == nil {
		return nil, fmt.Errorf("%s has no invariants section", family)
	}
	rest := body[loc[1]:]
	if j := regexp.MustCompile(`(?m)^## `).FindStringIndex(rest); j != nil {
		rest = rest[:j[0]]
	}

	var lines []string
	for _, l := range strings.Split(rest, "\n") {
		if strings.HasPrefix(l, "- ") {
			lines = append(lines, strings.TrimPrefix(l, "- "))
		}
	}
	if len(lines) == 0 {
		// A guard that parses nothing passes everything, which is worse than no
		// guard: it reports coverage it never looked for.
		return nil, fmt.Errorf("%s: no invariant lines parsed", family)
	}
	return lines, nil
}

// claims returns every fragment the line contains, sorted, so that nothing
// Check reports depends on the order a map was ranged in.
func claims(line string, mapping map[string]string) []string {
	var out []string
	for frag := range mapping {
		if strings.Contains(line, frag) {
			out = append(out, frag)
		}
	}
	slices.Sort(out)
	return out
}

func testSource(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return "", err
		}
		b.Write(data)
	}
	return b.String(), nil
}

func short(s string) string {
	r := []rune(s)
	if len(r) > 96 {
		return string(r[:96]) + "…"
	}
	return s
}
