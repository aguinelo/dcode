// Package workspace reads what a project declares about itself.
//
// Reading only, and running nothing. A gate found here is a command the
// project named as a way of checking itself; whether it passes is a different
// question, asked by 202608261730-done-qualifier. Confusing the two would put
// a list of guarantees in the prefix on the strength of a list of names.
//
// It is not internal/vcs. That package reads git and says so at the top; a
// declared gate is package.json and Makefile, which have nothing to do with
// version control.
package workspace

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MaxGates is how many reach the prefix.
//
// A Makefile with seventy targets would otherwise spend the context window on
// a list nobody reads. The cut is declared where it is rendered, because
// nothing in this codebase truncates in silence.
const MaxGates = 20

// maxCommand is how much of one command's text is carried. A script that is a
// hundred-character pipeline is named, not reproduced.
const maxCommand = 120

// Gate is a command the project itself declares as a way of checking it.
//
// Declared, never inferred. dcode does not decide that `npm test` is the gate
// of a project that never said so — the same rule that keeps Protected a
// declaration rather than a guess in the loop families.
type Gate struct {
	// Name is how the project names it: a script key, a Makefile target.
	Name string
	// Command is what would run.
	Command string
	// Source is the file that declared it.
	Source string
}

// Unreadable is a manifest that is there and could not be read or parsed.
//
// It is not "declares nothing". A project with no package.json is ordinary and
// earns no line anywhere; a project whose package.json does not parse has a
// declaration nobody could read, and reading that as an absence is the claim
// that the project declares nothing, made on the strength of a failure.
type Unreadable struct {
	// Source is the file, as Gate.Source names it.
	Source string
	// Err is why it could not be read: the read error or the parse error.
	Err error
}

func (u Unreadable) String() string { return u.Source + ": " + u.Err.Error() }

// Probe reads what the workspace declares.
//
// Gates is nil when nothing was declared — no manifest, or manifests with
// nothing in them — and that is ordinary and inconsequential, so it earns no
// line in the prefix. A manifest that exists and cannot be read is a different
// fact, and it comes back in unreadable for the caller to say: "did not manage
// to read it" and "there is nothing there" must never read the same.
//
// A cancelled probe reads nothing and reports nothing, which is the third
// fact: it did not look.
func Probe(ctx context.Context, dir string) (gates []Gate, unreadable []Unreadable) {
	if err := ctx.Err(); err != nil {
		return nil, nil
	}
	pkg, err := fromPackageJSON(dir)
	if err != nil {
		unreadable = append(unreadable, Unreadable{Source: "package.json", Err: err})
	}
	mk, err := fromMakefile(dir)
	if err != nil {
		unreadable = append(unreadable, Unreadable{Source: "Makefile", Err: err})
	}
	gates = append(pkg, mk...)
	if len(gates) == 0 {
		gates = nil
	}
	return gates, unreadable
}

// absent is a manifest that is simply not there, which is not an error.
func absent(err error) bool { return errors.Is(err, fs.ErrNotExist) }

// fromPackageJSON reads the `scripts` object, in sorted key order.
//
// Sorted because the prefix has to be byte-identical for the same tree, and Go
// randomises map iteration. A prefix that reshuffles between runs invalidates
// the provider cache on every session for no reason at all.
//
// A malformed package.json is the project's problem and not a reason to fail a
// session, so it is returned as an error for the caller to say rather than one
// that stops anything. What it must not become is "declared nothing".
func fromPackageJSON(dir string) ([]Gate, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		if absent(err) {
			return nil, nil
		}
		return nil, err
	}
	var doc struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(doc.Scripts))
	for n := range doc.Scripts {
		names = append(names, n)
	}
	sort.Strings(names)

	out := make([]Gate, 0, len(names))
	for _, n := range names {
		out = append(out, Gate{Name: n, Command: cut(doc.Scripts[n]), Source: "package.json"})
	}
	return out, nil
}

// fromMakefile reads target names, in the order the file declares them.
//
// Order of appearance rather than sorted: a Makefile is written to be read
// top to bottom and the first target is the default one.
//
// A Makefile the scanner cannot finish — a line longer than it holds, a read
// that fails halfway — is unreadable, not empty, and none of what was read
// before the failure is offered as the whole list.
func fromMakefile(dir string) ([]Gate, error) {
	f, err := os.Open(filepath.Join(dir, "Makefile"))
	if err != nil {
		if absent(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var out []Gate
	seen := map[string]bool{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// A rule line starts at column zero and carries a colon. Anything
		// indented is a recipe, and anything without a colon is not a rule.
		if line == "" || line[0] == ' ' || line[0] == '\t' || line[0] == '#' {
			continue
		}
		colon := strings.Index(line, ":")
		if colon <= 0 {
			continue
		}
		// `x := y` and `x ::= y` are variable assignments wearing a colon.
		if strings.HasPrefix(strings.TrimSpace(line[colon:]), ":=") || strings.HasPrefix(line[colon:], "=") {
			continue
		}
		if eq := strings.Index(line, "="); eq >= 0 && eq < colon {
			continue
		}
		name := strings.TrimSpace(line[:colon])
		// A target with a dot is .PHONY, .DEFAULT and friends — directives to
		// make, not things a person runs. A target with a space is a rule with
		// several, and naming one of them would be naming the wrong one.
		if name == "" || strings.ContainsAny(name, " \t$%") || strings.HasPrefix(name, ".") {
			continue
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, Gate{Name: name, Command: "make " + name, Source: "Makefile"})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func cut(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxCommand {
		return s
	}
	return s[:maxCommand] + "…"
}
