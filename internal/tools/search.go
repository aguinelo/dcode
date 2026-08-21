package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/aguinelo/dcode/internal/policy"

	"github.com/aguinelo/dcode/internal/protocol"
)

// ---------- glob ----------

// Glob finds files by pattern.
type Glob struct{}

// GlobInput is the argument shape.
type GlobInput struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
}

func (Glob) Name() string { return "glob" }

func (Glob) Description() string {
	return "Find files by glob pattern, for example **/*.go or internal/**/test*.go. " +
		"Results are sorted and ignore anything .gitignore excludes."
}

func (Glob) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{` +
		`"pattern":{"type":"string","description":"Glob pattern; ** matches across directories."},` +
		`"path":{"type":"string","description":"Root to search from; defaults to the workspace."}},` +
		`"required":["pattern"]}`)
}

func (g Glob) Declare(input json.RawMessage) (policy.Request, error) {
	var in GlobInput
	if err := decode(g.Name(), input, &in); err != nil {
		return policy.Request{}, err
	}
	return policy.Request{Tool: g.Name(), Paths: []policy.Access{{Path: g.root(in)}}}, nil
}

func (Glob) root(in GlobInput) string {
	if in.Path != "" {
		return in.Path
	}
	return "."
}

func (g Glob) Execute(ctx context.Context, input json.RawMessage, s *State) (Result, error) {
	var in GlobInput
	if err := decode(g.Name(), input, &in); err != nil {
		return err.(*ToolError).Result(), nil
	}
	re, terr := globToRegexp(g.Name(), in.Pattern)
	if terr != nil {
		return terr.Result(), nil
	}
	root, terr := resolvePath(g.Name(), s, g.root(in), false)
	if terr != nil {
		return terr.Result(), nil
	}

	root, only := searchRoot(root)
	// No total: the walk is what discovers how many there are, so the count on
	// its own is the only honest thing to send.
	step := Reporter(ctx, protocol.ProgressFiles, 0)
	matches, err := walkFiles(root, s.Limits.RespectGitignore, func(rel string) bool {
		if only != "" {
			return rel == only
		}
		return re.MatchString(rel)
	}, step)
	if err != nil {
		return errf(g.Name(), CodeNotFound, "", "could not search %s: %v", in.Path, err).Result(), nil
	}

	// Alphabetical, never filesystem order: the latter varies by machine and
	// would pass locally while breaking in CI.
	sort.Strings(matches)

	res := Result{}
	if n := s.Limits.GlobMaxResults; n > 0 && len(matches) > n {
		res.Truncated = true
		res.Remaining = len(matches) - n
		matches = matches[:n]
	}
	if len(matches) == 0 {
		return Result{Output: fmt.Sprintf("no files match %q", in.Pattern)}, nil
	}

	res.Meta = Meta{Files: len(matches)}
	res.Output = strings.Join(matches, "\n")
	if res.Truncated {
		res.Output += fmt.Sprintf("\n\n… %d more. Narrow the pattern.", res.Remaining)
	}
	return res, nil
}

// ---------- grep ----------

// GrepMaxContextLines caps how much surrounding text one match may carry.
//
// A ceiling rather than a configuration key, for the same reason the match
// limit is one: the number that matters is "enough to understand the match",
// and past a handful more lines stop informing and start costing. Asking for
// five thousand is asking for the file, which is what grep exists not to be.
const GrepMaxContextLines = 10

// Grep searches file contents.
type Grep struct{}

// GrepInput is the argument shape.
type GrepInput struct {
	Pattern      string `json:"pattern"`
	Path         string `json:"path,omitempty"`
	Glob         string `json:"glob,omitempty"`
	ContextLines int    `json:"context_lines,omitempty"`
}

func (Grep) Name() string { return "grep" }

func (Grep) Description() string {
	return "Search file contents with a regular expression. " +
		"Output is path:line:text, sorted. Narrow with glob to search fewer files. " +
		"When you are looking for an identifier rather than free text, use symbol instead: " +
		"it matches on symbol boundaries and tells a declaration from a use."
}

func (Grep) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{` +
		`"pattern":{"type":"string","description":"Go regular expression."},` +
		`"path":{"type":"string","description":"Root to search from."},` +
		`"glob":{"type":"string","description":"Only search files matching this glob."},` +
		`"context_lines":{"type":"integer","description":"Lines of context around each match."}},` +
		`"required":["pattern"]}`)
}

func (g Grep) Declare(input json.RawMessage) (policy.Request, error) {
	var in GrepInput
	if err := decode(g.Name(), input, &in); err != nil {
		return policy.Request{}, err
	}
	p := in.Path
	if p == "" {
		p = "."
	}
	return policy.Request{Tool: g.Name(), Paths: []policy.Access{{Path: p}}}, nil
}

func (g Grep) Execute(ctx context.Context, input json.RawMessage, s *State) (Result, error) {
	var in GrepInput
	if err := decode(g.Name(), input, &in); err != nil {
		return err.(*ToolError).Result(), nil
	}
	re, err := regexp.Compile(in.Pattern)
	if err != nil {
		// A bad pattern is the model's to fix, so the message says what broke
		// rather than panicking or returning nothing.
		return errf(g.Name(), CodeInvalidPattern,
			"Escape any regex metacharacters you meant literally.",
			"invalid regular expression: %v", err).Result(), nil
	}

	var fileRe *regexp.Regexp
	if in.Glob != "" {
		fileRe, _ = globToRegexpOrNil(in.Glob)
	}

	rootIn := in.Path
	if rootIn == "" {
		rootIn = "."
	}
	root, terr := resolvePath(g.Name(), s, rootIn, false)
	if terr != nil {
		return terr.Result(), nil
	}

	root, only := searchRoot(root)
	files, err := walkFiles(root, s.Limits.RespectGitignore, func(rel string) bool {
		if only != "" {
			return rel == only
		}
		return fileRe == nil || fileRe.MatchString(rel)
	}, nil)
	if err != nil {
		return errf(g.Name(), CodeNotFound, "", "could not search: %v", err).Result(), nil
	}
	sort.Strings(files)

	// Context is bounded like everything else here. A generous number against a
	// common pattern would otherwise return the file, which is the outcome the
	// match limit exists to prevent.
	ctxLines := in.ContextLines
	if ctxLines < 0 {
		ctxLines = 0
	}
	if ctxLines > GrepMaxContextLines {
		ctxLines = GrepMaxContextLines
	}

	type hit struct {
		file string
		line int
		text string
	}
	var hits []hit
	limit := s.Limits.GrepMaxMatches
	truncated := false

	// The list is known before the scan starts, so this one can say `n of N`.
	// Its own walk stays quiet: two phases with different totals would show a
	// count that climbs, restarts and climbs again, which reads as a mistake.
	step := Reporter(ctx, protocol.ProgressFiles, len(files))
	for i, rel := range files {
		step(i + 1)
		raw, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil || !utf8.Valid(raw) {
			continue
		}
		lines := strings.Split(string(raw), "\n")
		// Which lines to emit for this file. A set rather than a per-match
		// slice, because two matches close together share context and printing
		// the overlap twice would read as two separate places in the file.
		emit := map[int]struct{}{}
		matched := 0
		for i, line := range lines {
			if !re.MatchString(line) {
				continue
			}
			if limit > 0 && matched+len(hits) >= limit {
				truncated = true
				break
			}
			matched++
			for j := i - ctxLines; j <= i+ctxLines; j++ {
				if j >= 0 && j < len(lines) {
					emit[j] = struct{}{}
				}
			}
		}
		idx := make([]int, 0, len(emit))
		for i := range emit {
			idx = append(idx, i)
		}
		sort.Ints(idx)
		for _, i := range idx {
			hits = append(hits, hit{rel, i + 1, lines[i]})
		}
		if truncated {
			break
		}
	}

	if len(hits) == 0 {
		return Result{Output: fmt.Sprintf("no matches for %q", in.Pattern)}, nil
	}

	var b strings.Builder
	for _, h := range hits {
		text := h.text
		if len(text) > 400 {
			text = text[:400] + " …"
		}
		fmt.Fprintf(&b, "%s:%d:%s\n", h.file, h.line, text)
	}
	// Distinct files, counted here because the hit type is local to this
	// function: "18 matches in 4 files" is what reads at a glance.
	distinct := map[string]struct{}{}
	for _, h := range hits {
		distinct[h.file] = struct{}{}
	}
	res := Result{Output: b.String(), Meta: Meta{Lines: len(hits), Files: len(distinct)}}
	if truncated {
		res.Truncated = true
		res.Output += fmt.Sprintf("\n… stopped at %d matches. Narrow the pattern or the glob.\n", limit)
	}
	return res, nil
}

// ---------- walking ----------

// walkFiles returns workspace-relative paths under root that keep matches true.
//
// Native Go rather than shelling out: it keeps the single static binary, and it
// removes the question of whether to bundle a search tool or depend on one the
// user may not have.
// searchRoot splits a search target into the directory to walk and, when the
// target names a single file, the one entry to keep.
//
// Pointing a search at one file is the most natural thing a model does after a
// broad search returns too much, and it did not work: WalkDir visits a file
// root exactly once with `rel == "."`, which walkFiles skips, so the search ran
// over nothing and answered "no matches". Not an error — a confident report of
// absence, which is the one kind of wrong answer nobody re-checks. A model that
// narrowed to the file it had just been shown was told the symbol was not in
// it.
//
// Splitting here rather than special-casing inside walkFiles keeps the returned
// paths relative to the directory, which is what every caller joins against and
// prints.
func searchRoot(root string) (dir, only string) {
	if fi, err := os.Stat(root); err == nil && !fi.IsDir() {
		return filepath.Dir(root), filepath.Base(root)
	}
	return root, ""
}

// walkFiles collects the paths a search will look at.
//
// seen is called with the running count as it goes, and may be nil. It exists
// for the caller that is DISCOVERING as it walks — glob has no total until the
// walk ends, so the count on its own is the only honest thing to report.
func walkFiles(root string, respectGitignore bool, keep func(rel string) bool, seen func(int)) ([]string, error) {
	ig := loadIgnores(root, respectGitignore)
	var out []string
	if seen == nil {
		seen = func(int) {}
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// One unreadable directory must not abort the whole search.
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil || rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			if d.Name() == ".git" || ig.match(rel, true) {
				return fs.SkipDir
			}
			return nil
		}
		if ig.match(rel, false) {
			return nil
		}
		if keep(rel) {
			out = append(out, rel)
			seen(len(out))
		}
		return nil
	})
	return out, err
}
