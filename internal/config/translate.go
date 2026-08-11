// Translation of instruction files written for another tool.
//
// AGENTS.md is a convention shared between agent tools, and reading it is RN-4:
// use the instruction the user already wrote rather than making them write it
// again for every tool. The intent is right. The side effect is that a shared
// name means whoever wrote first wins — including having written for something
// else.
//
// Filtering by subject at run time would need semantic judgement, on every
// turn, automatically and invisibly. It would be wrong sometimes, and wrong in
// silence, discarding a legitimate rule of the user's while believing it
// belonged to another tool. A filter that errs quietly is worse than no filter.
//
// So the moment moves: translate ONCE, at setup, into a file a person reads,
// reviews and edits before it counts. The error becomes visible, reviewable and
// singular, instead of invisible, automatic and every message.
//
// NOTHING HERE EXECUTES ANYTHING. That is the point of the import guard in the
// test: AGENTS.md is content of a repository that may have been cloned from
// anywhere, and `npm install` fires a postinstall script. Verifying by running
// would turn setup into "execute the stranger's instructions", and the sandbox
// does not help, because the command runs inside the workspace, which is
// exactly where the damage would be done.
package config

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"regexp"
	"sort"
	"strings"
)

// Finding is something in a shared instruction file that does not apply here.
type Finding struct {
	Subject string // the tool or command named
	Reason  string // why it does not apply
}

func (f Finding) String() string { return f.Subject + " — " + f.Reason }

var backtickName = regexp.MustCompile("`([A-Za-z_][A-Za-z0-9_]*)`")

// namedTool finds the `<Name> tool` form, which carries its own context.
var namedTool = regexp.MustCompile(`(?i)\b([A-Za-z_][A-Za-z0-9_]*)\s+tools?\b`)

// bareList finds the names in `Key tools: a, b, c`.
var bareList = regexp.MustCompile(`(?i)\btools?\s*[:=]\s*(.+)$`)

var identifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// VerifyTools reports tools the text tells the agent to use that do not exist.
//
// This is the deterministic half, and where most of the damage is: dcode knows
// its own tools by heart. "This file says to create sub-agents; I have no
// sub-agents" is not an opinion.
//
// The detector is scoped to lines that are ABOUT tools, and that scoping is the
// difference between a number worth reading and a number full of noise.
// Measured against this repository, taking every backticked identifier turned a
// list of commit types — `chore`, `docs` — into findings. A count that inflates
// is a count nobody trusts, and the whole point of the notice is to be believed.
func VerifyTools(text string, have []string) []Finding {
	present := map[string]struct{}{}
	for _, h := range have {
		present[strings.ToLower(h)] = struct{}{}
	}

	seen := map[string]struct{}{}
	var out []Finding
	add := func(name string) {
		if name == "" || !identifier.MatchString(name) {
			return
		}
		lower := strings.ToLower(name)
		if _, ok := present[lower]; ok {
			return
		}
		if _, ok := commonWords[lower]; ok {
			return
		}
		if _, ok := seen[lower]; ok {
			return
		}
		seen[lower] = struct{}{}
		out = append(out, Finding{
			Subject: name,
			Reason:  "dcode has no such tool; its tools are " + strings.Join(have, ", "),
		})
	}

	for _, line := range strings.Split(text, "\n") {
		for _, m := range namedTool.FindAllStringSubmatch(line, -1) {
			add(m[1])
		}
		if !strings.Contains(strings.ToLower(line), "tool") {
			continue
		}
		for _, m := range backtickName.FindAllStringSubmatch(line, -1) {
			add(m[1])
		}
		if m := bareList.FindStringSubmatch(line); m != nil {
			for _, part := range strings.FieldsFunc(m[1], func(r rune) bool {
				return r == ',' || r == '·' || r == '|' || r == ';'
			}) {
				add(strings.Trim(strings.TrimSpace(part), "`.*_"))
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Subject < out[j].Subject })
	return out
}

// commonWords are words that appear next to "tool" in ordinary prose and are
// not tool names. Without this every "the tool" becomes a finding, and a report
// full of noise is a report nobody reads.
var commonWords = map[string]struct{}{
	"a": {}, "an": {}, "the": {}, "this": {}, "that": {}, "each": {}, "every": {},
	"any": {}, "no": {}, "one": {}, "other": {}, "another": {}, "same": {},
	"dedicated": {}, "right": {}, "correct": {}, "appropriate": {}, "which": {},
	"what": {}, "when": {}, "use": {}, "using": {}, "with": {}, "and": {}, "or": {},
}

// commandProbes maps a command prefix to the file whose presence would make it
// possible. Presence, never execution.
var commandProbes = []struct {
	prefix string
	needs  []string
}{
	{"npm ", []string{"package.json"}},
	{"npx ", []string{"package.json"}},
	{"yarn ", []string{"package.json"}},
	{"pnpm ", []string{"package.json"}},
	{"go ", []string{"go.mod"}},
	{"make ", []string{"Makefile", "makefile", "GNUmakefile"}},
	{"cargo ", []string{"Cargo.toml"}},
	{"poetry ", []string{"pyproject.toml"}},
	{"pytest", []string{"pyproject.toml", "setup.py", "setup.cfg", "tox.ini"}},
	{"bundle ", []string{"Gemfile"}},
	{"mvn ", []string{"pom.xml"}},
	{"gradle ", []string{"build.gradle", "build.gradle.kts"}},
}

// commandLine finds commands in fenced blocks and inline code.
var commandLine = regexp.MustCompile("`([^`\n]+)`")

// ProbeCommands reports commands the text tells the agent to run that this
// repository cannot possibly satisfy.
//
// `ls package.json` answers the question and runs nothing of the stranger's.
func ProbeCommands(text string, fsys fs.FS) []Finding {
	seen := map[string]struct{}{}
	var out []Finding

	for _, m := range commandLine.FindAllStringSubmatch(text, -1) {
		cmd := strings.TrimSpace(m[1])
		for _, p := range commandProbes {
			if !strings.HasPrefix(cmd, p.prefix) {
				continue
			}
			if anyExists(fsys, p.needs) {
				break
			}
			if _, ok := seen[cmd]; ok {
				break
			}
			seen[cmd] = struct{}{}
			out = append(out, Finding{
				Subject: cmd,
				Reason:  "no " + strings.Join(p.needs, " or ") + " in this repository",
			})
			break
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Subject < out[j].Subject })
	return out
}

func anyExists(fsys fs.FS, names []string) bool {
	for _, n := range names {
		if _, err := fs.Stat(fsys, n); err == nil {
			return true
		}
	}
	return false
}

// DigestMarker is the line a generated DCODE.md carries so a later session can
// tell whether its sources have moved.
const DigestMarker = "<!-- dcode:sources "

// fileDigest hashes one source file.
//
// Content, not modification time: a file rewritten with the same bytes has not
// changed, and a clock would make the digest differ between machines.
func fileDigest(fsys fs.FS, name string) string {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return "absent"
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:12]
}

// RenderDigest is the marker line to embed in a generated DCODE.md.
//
// Per file, not one digest for the set. A single value would answer "something
// moved", and the warning has to say WHAT moved: "something changed" sends a
// person looking, "AGENTS.md changed" is actionable (RN-6.2).
func RenderDigest(fsys fs.FS, files []string) string {
	names := append([]string(nil), files...)
	sort.Strings(names)

	parts := make([]string, 0, len(names))
	for _, n := range names {
		parts = append(parts, n+"="+fileDigest(fsys, n))
	}
	return DigestMarker + strings.Join(parts, ",") + " -->"
}

var digestLine = regexp.MustCompile(`(?m)^<!-- dcode:sources ([^ ]*) -->\s*$`)

// Diverged reports which source files have changed since the DCODE.md was
// generated, by name.
//
// It never rewrites anything. A generated DCODE.md is generated once and then
// belongs to the human — it is the file they will edit by hand, and that is what
// it is for. Regenerating over it loses work with nobody seeing.
func Diverged(dcodeMD string, fsys fs.FS) ([]string, bool) {
	m := digestLine.FindStringSubmatch(dcodeMD)
	if m == nil || m[1] == "" {
		return nil, false // never generated from sources; nothing to compare
	}

	var changed []string
	for _, entry := range strings.Split(m[1], ",") {
		name, want, ok := strings.Cut(entry, "=")
		if !ok || name == "" {
			continue
		}
		got := fileDigest(fsys, name)
		if got == want {
			continue
		}
		if got == "absent" {
			changed = append(changed, name+" (gone)")
			continue
		}
		changed = append(changed, name)
	}
	sort.Strings(changed)
	return changed, len(changed) > 0
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}
