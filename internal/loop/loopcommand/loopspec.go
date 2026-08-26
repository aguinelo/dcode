// Package loopcommand is the /loop façade over the agent loop's done
// definition. It parses a `tasks.md`-shaped directory into a LoopSpec and
// returns it as a loop.DoneSet, without changing the turn cycle.
//
// Spec: docs/specs/architecture/loop-command/202608252000-loop-command.*.spec.md
package loopcommand

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/aguinelo/dcode/internal/loop"
)

// LoopSpec is a parsed specification external to the workspace.
//
// The parser is total: it never panics on malformed input and never invents
// data. A malformed spec returns an error; an empty one returns a LoopSpec
// with no criteria (which produces a DoneSet with no criteria, which the
// agent-loop reports as "no definition of done").
type LoopSpec struct {
	// Path is the directory containing tasks.md.
	Path string

	// Criteria are the verifiable conditions of done for this spec.
	// Sourced exclusively from tasks.md. Prose is not a source.
	Criteria []loop.Criterion

	// Protected are paths the agent may not modify silently during execution.
	// Sourced from the user (frontmatter or argument), not inferred from
	// where the spec lives.
	Protected []string
}

// tasksFile is the file name the parser reads. spec.md is for humans; the
// structure that becomes DoneSet lives here.
const tasksFile = "tasks.md"

// The shape of one criterion-bearing task line.
//
// `- [ ] 12. `path` — desc. verify: `cmd`, exit: K`
// `- [x] 12. `path` — desc. verify: `cmd`
//
// The number after the checkbox is the criterion Name. The path is a
// documentation anchor, not parsed into the type (it would be wrong to put
// it in Name — the report reads Names, and "12" reads; "path/to/very-long.ts"
// does not).
var taskLine = regexp.MustCompile(
	`^[\t ]*- \[([ x])\] (\d+)\. ` + "`([^`]+)`" + ` — .*verify: ` + "`([^`]+)`" + `(?:, exit: (-?\d+))?`,
)

// protectedLine in the frontmatter: `protected = ["**/*_test.go"]`
var protectedLine = regexp.MustCompile(`^protected\s*=\s*\[(.*)\]\s*$`)

// LoadSpec reads a LoopSpec from a directory containing tasks.md. Returns an
// error on malformed input or a missing file.
func LoadSpec(path string) (LoopSpec, error) {
	return LoadSpecWithProtect(path, nil)
}

// LoadSpecWithProtect reads a LoopSpec and layers the argument's globs on
// top of any declared in the file. File-declared protected is preserved
// even when it overlaps the argument; both end up in the final list.
func LoadSpecWithProtect(path string, protect []string) (LoopSpec, error) {
	tasksPath := path + "/" + tasksFile
	raw, err := os.ReadFile(tasksPath)
	if err != nil {
		return LoopSpec{}, fmt.Errorf("loopcommand: read %s: %w", tasksPath, err)
	}

	spec, err := parseTasks(string(raw))
	if err != nil {
		return LoopSpec{}, err
	}
	spec.Path = path

	// File-declared comes first; argument layers on top. Both go in.
	spec.Protected = append(spec.Protected, protect...)

	return spec, nil
}

// parseTasks is the actual parsing. Pulled out so it can be tested without
// touching the filesystem (the file-level wrapper handles I/O and errors).
func parseTasks(content string) (LoopSpec, error) {
	out := LoopSpec{}

	scanner := bufio.NewScanner(strings.NewReader(content))
	inFrontmatter := false
	frontmatterOpen := false
	var frontmatterLines []string

	for scanner.Scan() {
		line := scanner.Text()

		// Frontmatter handling. The frontmatter is the YAML-ish block
		// between two `---` lines at the very top of the file. Anything
		// other than `protected = [...]` inside it is ignored — the spec
		// declares one key per purpose, and inventing more would mean
		// re-reading the user's intent.
		if frontmatterOpen {
			if strings.TrimSpace(line) == "---" {
				frontmatterOpen = false
				out.Protected = protectedFromFrontmatter(frontmatterLines)
				continue
			}
			frontmatterLines = append(frontmatterLines, line)
			continue
		}
		if !inFrontmatter && strings.TrimSpace(line) == "---" {
			inFrontmatter = true
			frontmatterOpen = true
			continue
		}

		m := taskLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		// m[1] is the checkbox state (" " or "x") — read but not branched on.
		// Both are treated as criteria, because the harness re-runs the check
		// regardless of who marked the box.
		name := m[2]
		cmd := m[4]
		exit := 0
		if m[5] != "" {
			n, err := strconv.Atoi(m[5])
			if err != nil {
				return LoopSpec{}, fmt.Errorf("loopcommand: bad exit code on line %q: %w", line, err)
			}
			exit = n
		}
		out.Criteria = append(out.Criteria, loop.Criterion{
			Name:     name,
			Command:  cmd,
			ExitCode: exit,
		})
	}
	if err := scanner.Err(); err != nil {
		return LoopSpec{}, fmt.Errorf("loopcommand: scan: %w", err)
	}
	// If we saw an opening `---` but never a closing one, the file is malformed.
	if frontmatterOpen {
		return LoopSpec{}, fmt.Errorf("loopcommand: frontmatter opened with `---` but never closed")
	}
	return out, nil
}

// protectedFromFrontmatter extracts `protected = [...]` from the frontmatter
// block. Any other key is ignored — the spec does not declare them, and
// inventing behaviour for keys nobody promised is the silent-extension
// defect.
//
// Returns a sorted slice so the report reads the same regardless of order.
func protectedFromFrontmatter(lines []string) []string {
	for _, l := range lines {
		m := protectedLine.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		body := strings.TrimSpace(m[1])
		if body == "" {
			return nil
		}
		parts := strings.Split(body, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			p = strings.TrimPrefix(p, `"`)
			p = strings.TrimSuffix(p, `"`)
			if p == "" {
				continue
			}
			out = append(out, p)
		}
		return out
	}
	return nil
}
