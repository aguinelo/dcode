package evals

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"
)

// A change is recorded as a fragment: one file per change, carrying both
// editions, under changelog.d/ (the core) or desktop/changelog.d/ (the desktop
// app). The "Unreleased" sections are assembled from them, and folded into a
// version's section at release.
//
// On 16 October 2026 fifteen pull requests in flight all conflicted on the
// same lines — the top of "Unreleased" in both editions and the counted
// decision-changelog row — and integration became serial, one hand-resolved
// conflict per merge. A file nobody else writes cannot conflict.
//
// What these tests check: every fragment is well formed and carries both
// editions; no edition carries a hand-written entry in its Unreleased section;
// and the script that assembles the section renders exactly the fragments.
// What they cannot check is that the fragment says why, which stays a matter
// for whoever reads the pull request.

// changelogSet is one changelog in its two editions and the directory its
// fragments live in.
type changelogSet struct {
	name      string // what scripts/changelog-assemble.sh calls it
	fragments string
	en, pt    string
}

func changelogSets(root string) []changelogSet {
	return []changelogSet{
		{"root", filepath.Join(root, "changelog.d"),
			filepath.Join(root, "CHANGELOG.md"), filepath.Join(root, "CHANGELOG.pt-BR.md")},
		{"desktop", filepath.Join(root, "desktop", "changelog.d"),
			filepath.Join(root, "desktop", "CHANGELOG.md"), filepath.Join(root, "desktop", "CHANGELOG.pt-BR.md")},
	}
}

var fragmentName = regexp.MustCompile(`^(\d{12})-[a-z0-9]+(?:-[a-z0-9]+)*\.md$`)

// fragment is one change in its two editions.
type fragment struct {
	name   string
	en, pt string
}

// readFragments returns the fragments of a directory, newest first — the
// order the assembled section shows them in. A missing directory is no
// fragment at all, which is the state right after a release folds them.
func readFragments(t *testing.T, dir string) []fragment {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var out []fragment
	for _, e := range entries {
		if e.IsDir() {
			t.Errorf("%s: %s is a directory; fragments are files", dir, e.Name())
			continue
		}
		m := fragmentName.FindStringSubmatch(e.Name())
		if m == nil {
			t.Errorf("%s: %q is not named <yyyymmddhhmm>-<slug>.md", dir, e.Name())
			continue
		}
		if _, err := time.Parse("200601021504", m[1]); err != nil {
			t.Errorf("%s: %q does not start with a real date and time: %v", dir, e.Name(), err)
		}
		en, pt, err := parseFragment(readFile(t, filepath.Join(dir, e.Name())))
		if err != nil {
			t.Errorf("%s/%s: %v", filepath.Base(dir), e.Name(), err)
			continue
		}
		out = append(out, fragment{name: e.Name(), en: en, pt: pt})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name > out[j].name })
	return out
}

// parseFragment reads the two editions out of a fragment: a `## en` heading,
// then a `## pt-BR` heading, each followed by exactly one entry. An entry is a
// single bullet that opens with its bold title, continued by indented lines.
func parseFragment(body string) (en, pt string, err error) {
	var heads []string
	sections := map[string][]string{}
	current := ""
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "#") {
			heads = append(heads, line)
			current = line
			continue
		}
		if current == "" {
			if strings.TrimSpace(line) != "" {
				return "", "", fmt.Errorf("text before the `## en` heading: %q", line)
			}
			continue
		}
		sections[current] = append(sections[current], line)
	}
	if strings.Join(heads, "|") != "## en|## pt-BR" {
		return "", "", fmt.Errorf("headings are %q; a fragment has `## en` then `## pt-BR`, and nothing else", heads)
	}
	en, err = entry("en", sections["## en"])
	if err != nil {
		return "", "", err
	}
	pt, err = entry("pt-BR", sections["## pt-BR"])
	return en, pt, err
}

func entry(lang string, lines []string) (string, error) {
	text := strings.Trim(strings.Join(lines, "\n"), "\n")
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("the %s edition is empty", lang)
	}
	if !strings.HasPrefix(text, "- **") {
		return "", fmt.Errorf("the %s edition does not open with `- **`, the bold title every entry has", lang)
	}
	for i, l := range strings.Split(text, "\n") {
		if i > 0 && strings.HasPrefix(l, "- ") {
			return "", fmt.Errorf("the %s edition carries a second entry (%q); one change, one fragment", lang, l)
		}
		if i > 0 && l != "" && !strings.HasPrefix(l, "  ") {
			return "", fmt.Errorf("the %s edition has a line that is not indented under its bullet: %q", lang, l)
		}
	}
	return text, nil
}

func TestEveryFragmentCarriesBothEditions(t *testing.T) {
	root := repoRootFromEvals(t)
	total := 0
	for _, set := range changelogSets(root) {
		total += len(readFragments(t, set.fragments))
	}
	if total == 0 {
		t.Log("no fragment pending in either changelog: the state right after a release")
	}
}

// unreleased returns the body of an edition's Unreleased section: everything
// between its heading and the next `## ` heading, or the end of the file.
func unreleased(t *testing.T, path string) string {
	t.Helper()
	body := readFile(t, path)
	heading := "## Unreleased"
	if strings.HasSuffix(path, ".pt-BR.md") {
		heading = "## Não publicado"
	}
	start := strings.Index(body, "\n"+heading+"\n")
	if start < 0 {
		t.Fatalf("%s: no %q heading; the section this guard reads is gone", path, heading)
	}
	rest := body[start+len(heading)+2:]
	if end := strings.Index(rest, "\n## "); end >= 0 {
		rest = rest[:end]
	}
	return rest
}

// The committed Unreleased section is a pointer to the fragments, never a
// list: an entry written there by hand is the shared line every pull request
// used to conflict on.
func TestNoUnreleasedEntryIsWrittenByHand(t *testing.T) {
	root := repoRootFromEvals(t)
	for _, set := range changelogSets(root) {
		for _, path := range []string{set.en, set.pt} {
			body := unreleased(t, path)
			for _, line := range strings.Split(body, "\n") {
				if strings.HasPrefix(line, "- ") {
					t.Errorf("%s: the Unreleased section carries an entry written by hand (%.60q); it goes in a fragment under %s",
						rel(root, path), line, rel(root, set.fragments))
					break
				}
			}
			if want := rel(root, set.fragments) + "/"; !strings.Contains(body, want) {
				t.Errorf("%s: the Unreleased section does not point at %s, where its entries are", rel(root, path), want)
			}
		}
	}
}

// The assembler is a shell script, and a script nothing runs rots in silence.
// Its output is compared with the fragments read here, in both editions.
func TestTheAssemblerRendersExactlyTheFragments(t *testing.T) {
	root := repoRootFromEvals(t)
	for _, set := range changelogSets(root) {
		frags := readFragments(t, set.fragments)
		for _, lang := range []string{"en", "pt"} {
			t.Run(set.name+"/"+lang, func(t *testing.T) {
				cmd := exec.Command("bash", filepath.Join(root, "scripts", "changelog-assemble.sh"), "show", set.name, lang)
				cmd.Dir = root
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("changelog-assemble.sh show %s %s: %v\n%s", set.name, lang, err, out)
				}
				var want []string
				for _, f := range frags {
					if lang == "en" {
						want = append(want, f.en)
					} else {
						want = append(want, f.pt)
					}
				}
				if got, w := strings.TrimRight(string(out), "\n"), strings.Join(want, "\n"); got != w {
					t.Errorf("the assembled section differs from the fragments.\n--- got\n%s\n--- want\n%s", got, w)
				}
			})
		}
	}
}

func rel(root, path string) string {
	r, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(r)
}
