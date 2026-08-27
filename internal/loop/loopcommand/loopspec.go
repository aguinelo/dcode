// Package loopcommand is the /loop façade over the agent loop's done
// definition. It parses a `tasks.md`-shaped directory into a LoopSpec and
// returns it as a loop.DoneSet, without changing the turn cycle.
//
// Nothing in the product calls this yet: the client-side recognition of
// `/loop` is Step 3 of the family's `.i` and has not been built. What ships
// here is the parser and the dispatch between sources, and the package is
// honest about that rather than reading as a delivered command.
//
// Spec: docs/specs/architecture/loop-command/202608252000-loop-command.*.spec.md
package loopcommand

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/aguinelo/dcode/internal/loop"
)

// LoopSpec is a parsed specification external to the workspace.
//
// The parser is total: it never panics on malformed input and never invents
// data. What it CANNOT read, it refuses to read — a file with no task list is
// an error, not an empty DoneSet. The difference matters more than it looks:
// an empty DoneSet is "nothing to verify", which the agent loop reports as
// done. Silently turning an unreadable file into "done" is the worst outcome
// this package can produce, and RN-6 exists to forbid it.
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

// taskHead identifies a task line and splits off everything after the number.
//
// Only the checkbox and the number are syntax. What follows — a path in
// backticks, a dash, a description — is documentation, and the parser reads
// past it without caring how it is punctuated.
//
// It used to care. The pattern required a literal em dash (`—`) between the
// path and the description, so a `tasks.md` written with a plain hyphen
// produced zero criteria and no error: an entire file of real work read as
// "no definition of done". A separator is not a contract, and making it one
// put the whole feature one keystroke away from silently reporting done.
var taskHead = regexp.MustCompile(`^[\t ]*- \[([ xX])\][\t ]+(\d+)\.[\t ]*(.*)$`)

// verifyPart is the only part of a task line that IS syntax: the command that
// decides whether the criterion is met.
var verifyPart = regexp.MustCompile("verify:[\t ]*`([^`]*)`[\t ]*(.*)$")

// exitPart is the optional expected exit code trailing the command.
var exitPart = regexp.MustCompile(`^,[\t ]*exit:[\t ]*(-?\d+)[\t ]*$`)

// protectedLine in the frontmatter: `protected = ["**/*_test.go"]`
var protectedLine = regexp.MustCompile(`^protected\s*=\s*\[(.*)\]\s*$`)

// LoadSpec reads a LoopSpec from a directory containing tasks.md. Returns an
// error on malformed input or a missing file.
func LoadSpec(path string) (LoopSpec, error) {
	return LoadSpecWithProtect(path, nil)
}

// LoadSpecWithProtect reads a LoopSpec and adds the argument's globs to any
// declared in the file.
//
// The two are a union, not a precedence. Protected is a list of paths whose
// modification gets surfaced, so "the file wins over the argument" would mean
// an argument could REMOVE a protection the file asked for — the one direction
// that must never be reachable by accident.
func LoadSpecWithProtect(path string, protect []string) (LoopSpec, error) {
	// A done.toml beside the spec wins over its tasks.md.
	//
	// Because the criteria a project can actually run are often nowhere in its
	// tasks. The specs this family was built for declare acceptance criteria as
	// SENTENCES — "Lighthouse >= 95", "the home page loads in under a second on
	// 4G" — which is what a person writes and what no parser may turn into a
	// command without inventing one.
	//
	// So the folder gets a place to say it in commands. Written by hand, or by
	// an agent in an ordinary turn, or one day by the qualifier: the file is
	// the same file either way, it is diffable, and it survives the session
	// that produced it.
	if set, found, err := doneBesideSpec(path, protect); found || err != nil {
		if err != nil {
			return LoopSpec{}, err
		}
		return LoopSpec{Path: path, Criteria: set.Criteria, Protected: set.Protected}, nil
	}

	tasksPath := filepath.Join(path, tasksFile)
	raw, err := os.ReadFile(tasksPath)
	if err != nil {
		return LoopSpec{}, fmt.Errorf("loopcommand: read %s: %w", tasksPath, err)
	}

	spec, err := parseTasks(string(raw))
	if err != nil {
		return LoopSpec{}, fmt.Errorf("loopcommand: %s: %w", tasksPath, err)
	}
	spec.Path = path
	spec.Protected = union(spec.Protected, protect)

	return spec, nil
}

// parseTasks is the actual parsing. Pulled out so it can be tested without
// touching the filesystem (the file-level wrapper handles I/O and errors).
func parseTasks(content string) (LoopSpec, error) {
	out := LoopSpec{}

	scanner := bufio.NewScanner(strings.NewReader(content))
	frontmatterOpen := false
	var frontmatterLines []string
	// sawTask separates "this file declares no verifiable criteria" from "this
	// file is not a task list at all". The first is a legitimate empty
	// DoneSet; the second is RN-6's error.
	sawTask := false
	// seen catches the same task number declared twice. Name is what the
	// report prints and what Progressed compares, so two criteria called "3"
	// are two rows a human cannot tell apart.
	seen := map[string]int{}
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := scanner.Text()

		if frontmatterOpen {
			if strings.TrimSpace(line) == "---" {
				frontmatterOpen = false
				out.Protected = protectedFromFrontmatter(frontmatterLines)
				continue
			}
			frontmatterLines = append(frontmatterLines, line)
			continue
		}
		// Frontmatter opens on the FIRST line or not at all. `---` is also
		// markdown's horizontal rule, and treating one mid-file as an opening
		// delimiter swallowed every line until the next one — or failed the
		// whole file as "frontmatter never closed" because there was no next
		// one. A section break is not a declaration.
		if lineNo == 1 && strings.TrimSpace(line) == "---" {
			frontmatterOpen = true
			continue
		}

		m := taskHead.FindStringSubmatch(line)
		if m == nil {
			// Headings, prose, blank lines. Not a task, not an error.
			continue
		}
		sawTask = true

		// m[1] is the checkbox state (" " or "x") — read but not branched on.
		// Both are treated as criteria, because the harness re-runs the check
		// regardless of who marked the box.
		name := m[2]
		rest := m[3]

		if !strings.Contains(rest, "verify:") {
			// RN-5: a task with nothing to run is not a criterion. It is the
			// human's work item, and it reaches the report as unavailable
			// through the empty-command path the agent loop already has.
			continue
		}

		v := verifyPart.FindStringSubmatch(rest)
		if v == nil {
			return LoopSpec{}, fmt.Errorf(
				"line %d: task %s says `verify:` and no command in backticks follows it", lineNo, name)
		}
		cmd := strings.TrimSpace(v[1])
		if cmd == "" {
			return LoopSpec{}, fmt.Errorf("line %d: task %s has an empty verify command", lineNo, name)
		}

		exit := 0
		if tail := strings.TrimSpace(v[2]); tail != "" {
			// Trailing prose after the command is fine — the command was read
			// correctly and the rest is for the human. A trailing `exit` that
			// does not parse is NOT fine: the author asked for one exit code
			// and would silently be given zero.
			if strings.Contains(tail, "exit") {
				e := exitPart.FindStringSubmatch(tail)
				if e == nil {
					return LoopSpec{}, fmt.Errorf(
						"line %d: task %s trails %q after the verify command, which mentions an exit code and does not read as `, exit: N`",
						lineNo, name, tail)
				}
				n, err := strconv.Atoi(e[1])
				if err != nil {
					return LoopSpec{}, fmt.Errorf("line %d: task %s has an unreadable exit code: %w", lineNo, name, err)
				}
				exit = n
			}
		}

		if first, dup := seen[name]; dup {
			return LoopSpec{}, fmt.Errorf(
				"line %d: task %s was already declared on line %d; the number is the criterion name and the report cannot tell two of them apart",
				lineNo, name, first)
		}
		seen[name] = lineNo

		out.Criteria = append(out.Criteria, loop.Criterion{
			Name:     name,
			Command:  cmd,
			ExitCode: exit,
		})
	}
	if err := scanner.Err(); err != nil {
		return LoopSpec{}, fmt.Errorf("scan: %w", err)
	}
	if frontmatterOpen {
		return LoopSpec{}, fmt.Errorf("frontmatter opened with `---` on line 1 and never closed")
	}
	if !sawTask {
		// RN-6. Not "zero criteria" — zero TASKS, which means the file is not
		// the thing it was asked to be. Returning an empty DoneSet here is how
		// an unreadable file becomes a green report.
		return LoopSpec{}, fmt.Errorf("no task line found; expected at least one `- [ ] N. ...` entry")
	}
	return out, nil
}

// protectedFromFrontmatter extracts `protected = [...]` from the frontmatter
// block. Any other key is ignored — the spec does not declare them, and
// inventing behaviour for keys nobody promised is the silent-extension
// defect.
func protectedFromFrontmatter(lines []string) []string {
	for _, l := range lines {
		m := protectedLine.FindStringSubmatch(strings.TrimSpace(l))
		if m == nil {
			continue
		}
		body := strings.TrimSpace(m[1])
		if body == "" {
			return nil
		}
		out := make([]string, 0, strings.Count(body, ",")+1)
		for _, p := range strings.Split(body, ",") {
			p = strings.TrimSpace(p)
			p = strings.Trim(p, `"'`)
			if p == "" {
				continue
			}
			out = append(out, p)
		}
		if len(out) == 0 {
			return nil
		}
		return out
	}
	return nil
}

// union appends the second list to the first, dropping what is already there.
// Order is first-declared-first, so the report reads in the order the user
// wrote rather than in the order the layers happened to be applied.
func union(first, second []string) []string {
	have := make(map[string]struct{}, len(first))
	for _, p := range first {
		have[p] = struct{}{}
	}
	out := first
	for _, p := range second {
		if p == "" {
			continue
		}
		if _, dup := have[p]; dup {
			continue
		}
		have[p] = struct{}{}
		out = append(out, p)
	}
	return out
}
