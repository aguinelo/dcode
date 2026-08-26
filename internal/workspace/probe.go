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

// Probe reads what the workspace declares.
//
// Nil when nothing was found, and nil is also what a caller gets for a
// directory it could not read. The two are the same answer here and that is
// deliberate: unlike a missing repository, a project declaring no gate is
// ordinary and inconsequential, so neither case earns a line in the prefix.
// What must never happen is either one becoming the claim that the project
// declares nothing.
func Probe(ctx context.Context, dir string) []Gate {
	if err := ctx.Err(); err != nil {
		return nil
	}
	gates := append(fromPackageJSON(dir), fromMakefile(dir)...)
	if len(gates) == 0 {
		return nil
	}
	return gates
}

// fromPackageJSON reads the `scripts` object, in sorted key order.
//
// Sorted because the prefix has to be byte-identical for the same tree, and Go
// randomises map iteration. A prefix that reshuffles between runs invalidates
// the provider cache on every session for no reason at all.
func fromPackageJSON(dir string) []Gate {
	raw, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil
	}
	var doc struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		// Malformed package.json is the project's problem and not a reason to
		// fail a session. It reads as "declared nothing", which is what an
		// unreadable declaration honestly is.
		return nil
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
	return out
}

// fromMakefile reads target names, in the order the file declares them.
//
// Order of appearance rather than sorted: a Makefile is written to be read
// top to bottom and the first target is the default one.
func fromMakefile(dir string) []Gate {
	f, err := os.Open(filepath.Join(dir, "Makefile"))
	if err != nil {
		return nil
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
	if scanner.Err() != nil {
		return nil
	}
	return out
}

func cut(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxCommand {
		return s
	}
	return s[:maxCommand] + "…"
}
