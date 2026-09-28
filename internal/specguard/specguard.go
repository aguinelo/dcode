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
// add an invariant and this goes red until someone names its test.
//
// It also holds WalkCheckout, the walk every guard that reads the source goes
// through. Those guards are tests in several packages, and what counts as this
// repository is one rule rather than one per guard: written separately, none of
// them left out a checkout nested inside this one.
package specguard

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Findings are the problems in one family, in spec order.
type Findings []string

// Check reports every invariant of family that no test claims, and every claim
// naming a test that does not exist.
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
// churn without the invariant changing.
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
	for _, line := range lines {
		name := claim(line, mapping)
		if name == "" {
			out = append(out, fmt.Sprintf("no test claims this invariant:\n  %s", short(line)))
			continue
		}
		if !regexp.MustCompile(`func ` + regexp.QuoteMeta(name) + `\b`).MatchString(src) {
			out = append(out, fmt.Sprintf("the invariant\n  %s\nnames %s, which does not exist", short(line), name))
		}
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

// WalkCheckout walks the checkout at root as filepath.WalkDir does, and leaves
// out what is not the checkout's: the .git at root, and every directory below
// root that holds a .git of its own.
//
// Such a directory is another checkout — a worktree, a clone, a submodule — and
// git does not descend into it either. The Claude desktop app keeps its session
// worktrees under .claude/worktrees/, inside the repository and ignored by git,
// so a guard walking everything under root read each one as more of this
// checkout: on 2026-09-28 the update guard found the one caller of Apply twice,
// and failed in a working copy while CI, with no worktree, passed.
//
// A directory it cannot look inside for a .git stops the walk with an error,
// rather than being read as this checkout's.
func WalkCheckout(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || path == root {
			return fn(path, d, err)
		}
		if d.Name() == ".git" {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			_, err := os.Lstat(filepath.Join(path, ".git"))
			if err == nil {
				return filepath.SkipDir
			}
			if !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("%s: cannot tell whether it is another checkout: %w", path, err)
			}
		}
		return fn(path, d, nil)
	})
}

func claim(line string, mapping map[string]string) string {
	for frag, name := range mapping {
		if strings.Contains(line, frag) {
			return name
		}
	}
	return ""
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
