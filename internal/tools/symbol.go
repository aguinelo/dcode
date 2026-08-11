package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/aguinelo/dcode/internal/policy"
)

// Symbol finds where an identifier is declared and where it is used.
//
// The eighth tool, and what justifies it is NOT the word boundary — `\bParse\b`
// is a regular expression the model could already write, since grep takes Go
// regexps. What it could not write without knowing every language is the
// distinction between a definition and a use: `func Parse(` in Go, `def parse`
// in Python, `fn parse` in Rust, `function` or `const … =` in TypeScript.
//
// That knowledge is data this package carries, rather than something the model
// reconstructs per language on every call and gets wrong in half of them.
type Symbol struct{}

// SymbolInput is the argument shape.
type SymbolInput struct {
	// Name is the symbol. NOT a regular expression: it is escaped before it
	// becomes a pattern. Accepting a regexp here would reintroduce the problem
	// through the back door and make symbol into grep with another name.
	Name string `json:"name"`
	Kind string `json:"kind,omitempty"` // "def" | "ref" | "any"; default "any"
	Path string `json:"path,omitempty"`
	Glob string `json:"glob,omitempty"`
}

// The three kinds.
const (
	KindDef = "def"
	KindRef = "ref"
	KindAny = "any"
)

// SymbolLimit is appended to every result, always.
//
// A textual search cannot see a call made through an interface, a function
// pointer, reflection, or a name assembled at run time. That is a false
// negative, and a false negative is the worst failure mode there is: an empty
// result looks exactly like a complete answer. An agent that concludes "there
// are no other callers" from this and renames half of them does so with
// confidence.
//
// Same principle as RN-5 on truncation, and for the same reason: output must
// not look complete when it is not.
const SymbolLimit = "textual match on symbol boundary; does not resolve interface or dynamic dispatch"

func (Symbol) Name() string { return "symbol" }

func (Symbol) Description() string {
	return "Find where a symbol is declared and where it is used. " +
		"name is a plain identifier, never a regular expression — it is matched on symbol " +
		"boundaries, so Parse does not match ParseTOML or parseMode. " +
		"kind selects def (the declaration), ref (uses), or any. " +
		"Prefer this over grep whenever you are looking for an identifier."
}

func (Symbol) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{` +
		`"name":{"type":"string","description":"The symbol. A plain identifier, not a regular expression."},` +
		`"kind":{"type":"string","enum":["def","ref","any"],"description":"def finds the declaration, ref finds uses, any finds both. Default any."},` +
		`"path":{"type":"string","description":"Root to search from."},` +
		`"glob":{"type":"string","description":"Only search files matching this glob."}},` +
		`"required":["name"]}`)
}

func (sy Symbol) Declare(input json.RawMessage) (policy.Request, error) {
	var in SymbolInput
	if err := decode(sy.Name(), input, &in); err != nil {
		return policy.Request{}, err
	}
	p := in.Path
	if p == "" {
		p = "."
	}
	return policy.Request{Tool: sy.Name(), Paths: []policy.Access{{Path: p}}}, nil
}

// declPatterns is how each language spells a declaration, keyed by extension.
//
// %s is the ALREADY-ESCAPED symbol. Every entry is anchored on a symbol
// boundary at the end so that a pattern for Parse cannot match ParseTOML.
//
// The table is deliberately incomplete rather than clever. A language that is
// not here is not an error — it falls back to a boundary match and reports the
// kind as unknown, which is a worse answer than a real parser and a much better
// one than a refusal.
var declPatterns = map[string][]string{
	".go": {
		`func\s+(?:\([^)]*\)\s*)?%s\b`, // function and method
		`type\s+%s\b`,
		`(?:var|const)\s+%s\b`,
		`%s\s*:?=`, // assignment or short declaration
	},
	".py": {
		`def\s+%s\b`,
		`class\s+%s\b`,
		`%s\s*=`,
	},
	".rs": {
		`fn\s+%s\b`,
		`(?:struct|enum|trait|type|const|static)\s+%s\b`,
		`let\s+(?:mut\s+)?%s\b`,
	},
	".ts":  tsPatterns,
	".tsx": tsPatterns,
	".js":  tsPatterns,
	".jsx": tsPatterns,
	".mjs": tsPatterns,
	".java": {
		`(?:class|interface|enum|record)\s+%s\b`,
		// A method declaration is required to open a body. The obvious wider
		// form, allowing a semicolon so abstract and interface members count
		// too, also matches `new Parser();` — `new` is just another word, and
		// RE2 has no lookbehind to exclude it.
		//
		// So an abstract or interface method is reported as a use rather than
		// a declaration. That is the safe direction to be wrong in: a missed
		// classification is a line still shown under `any`, while the other
		// error turns every constructor call into a declaration.
		`\w[\w<>\[\].]*\s+%s\s*\([^)]*\)\s*(?:throws[\w\s,.]+)?\{`,
	},
	".rb": {
		`def\s+(?:self\.)?%s\b`,
		`(?:class|module)\s+%s\b`,
	},
	".c":   cPatterns,
	".h":   cPatterns,
	".cc":  cPatterns,
	".cpp": cPatterns,
	".hpp": cPatterns,
	".sh": {
		`(?:function\s+)?%s\s*\(\s*\)`,
		`%s=`,
	},
}

var tsPatterns = []string{
	`(?:function|class|interface|type|enum)\s+%s\b`,
	`(?:const|let|var)\s+%s\b`,
	// Object-literal forms. Both require what FOLLOWS a declaration and not
	// merely a call: `parse: function`, `parse: (` or a method shorthand whose
	// parentheses are followed by a body. A bare `%s\s*\(` would classify
	// every call site as a declaration, which a test caught.
	`%s\s*:\s*(?:function\b|\()`,
	`%s\s*\([^)]*\)\s*\{`,
}

var cPatterns = []string{
	`(?:struct|class|enum|union|typedef)\s+%s\b`,
	`\w[\w \*&:<>]*\b%s\s*\([^;]*\)\s*\{`, // definition, not a prototype
	`#define\s+%s\b`,
}

// declRegexp builds the definition matcher for one extension.
//
// The bool reports whether the language was known. An unknown one still gets a
// usable answer — the boundary match — and the caller says so in the output
// rather than pretending the kind was honoured.
func declRegexp(ext, name string) (*regexp.Regexp, bool) {
	quoted := regexp.QuoteMeta(name)
	pats, known := declPatterns[strings.ToLower(ext)]
	if !known {
		return boundaryRegexp(name), false
	}
	alts := make([]string, 0, len(pats))
	for _, p := range pats {
		alts = append(alts, "(?:"+fmt.Sprintf(p, quoted)+")")
	}
	return regexp.MustCompile(strings.Join(alts, "|")), true
}

// boundaryRegexp matches the symbol as a whole word and nothing else.
//
// QuoteMeta is what keeps `a.b` from matching `axb`: the name is data, never
// pattern.
func boundaryRegexp(name string) *regexp.Regexp {
	return regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`)
}

func (sy Symbol) Execute(_ context.Context, input json.RawMessage, s *State) (Result, error) {
	var in SymbolInput
	if err := decode(sy.Name(), input, &in); err != nil {
		return err.(*ToolError).Result(), nil
	}
	if strings.TrimSpace(in.Name) == "" {
		return errf(sy.Name(), CodeInvalidPattern,
			"Give the identifier you are looking for.",
			"name is empty").Result(), nil
	}
	kind := in.Kind
	if kind == "" {
		kind = KindAny
	}
	if kind != KindDef && kind != KindRef && kind != KindAny {
		return errf(sy.Name(), CodeInvalidPattern,
			"Use def, ref or any.",
			"unknown kind %q", in.Kind).Result(), nil
	}

	var fileRe *regexp.Regexp
	if in.Glob != "" {
		fileRe, _ = globToRegexpOrNil(in.Glob)
	}
	rootIn := in.Path
	if rootIn == "" {
		rootIn = "."
	}
	root, terr := resolvePath(sy.Name(), s, rootIn, false)
	if terr != nil {
		return terr.Result(), nil
	}

	files, err := walkFiles(root, s.Limits.RespectGitignore, func(rel string) bool {
		return fileRe == nil || fileRe.MatchString(rel)
	})
	if err != nil {
		return errf(sy.Name(), CodeNotFound, "", "could not search: %v", err).Result(), nil
	}
	sort.Strings(files)

	boundary := boundaryRegexp(in.Name)

	type hit struct {
		file  string
		line  int
		text  string
		isDef bool
	}
	var hits []hit
	unknownExt := map[string]struct{}{}
	limit := s.Limits.SymbolMaxMatches
	truncated := false

	for _, rel := range files {
		raw, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil || !utf8.Valid(raw) {
			continue
		}
		ext := filepath.Ext(rel)
		defRe, known := declRegexp(ext, in.Name)
		if !known {
			unknownExt[ext] = struct{}{}
		}

		for i, line := range strings.Split(string(raw), "\n") {
			// The boundary decides membership; the declaration pattern only
			// classifies. A definition pattern that matched without the
			// boundary matching would be a pattern bug, not a hit.
			if !boundary.MatchString(line) {
				continue
			}
			isDef := known && defRe.MatchString(line)
			switch kind {
			case KindDef:
				if !isDef {
					continue
				}
			case KindRef:
				if isDef {
					continue
				}
			}
			if limit > 0 && len(hits) >= limit {
				truncated = true
				break
			}
			hits = append(hits, hit{rel, i + 1, line, isDef})
		}
		if truncated {
			break
		}
	}

	var defs, refs int
	var b strings.Builder
	for _, h := range hits {
		text := strings.TrimRight(h.text, "\r")
		if len(text) > 400 {
			text = text[:400] + " …"
		}
		if h.isDef {
			defs++
		} else {
			refs++
		}
		fmt.Fprintf(&b, "%s:%d:%s\n", h.file, h.line, text)
	}

	var head strings.Builder
	fmt.Fprintf(&head, "symbol: %s (%s) — %d match, %d refs\n", in.Name, kind, defs, refs)
	head.WriteString(SymbolLimit + "\n")
	if len(unknownExt) > 0 {
		exts := make([]string, 0, len(unknownExt))
		for e := range unknownExt {
			if e == "" {
				e = "(no extension)"
			}
			exts = append(exts, e)
		}
		sort.Strings(exts)
		fmt.Fprintf(&head, "kind unknown for %s: no declaration patterns for these, matched on word boundary only\n",
			strings.Join(exts, ", "))
	}
	head.WriteString("\n")

	distinct := map[string]struct{}{}
	for _, h := range hits {
		distinct[h.file] = struct{}{}
	}
	res := Result{
		Output: head.String() + b.String(),
		Meta:   Meta{Lines: len(hits), Files: len(distinct)},
	}
	if len(hits) == 0 {
		res.Output = head.String() + fmt.Sprintf("no %s of %q found\n", kindNoun(kind), in.Name)
	}
	if truncated {
		res.Truncated = true
		res.Output += fmt.Sprintf("\n… stopped at %d matches. Narrow with path or glob.\n", limit)
	}
	return res, nil
}

func kindNoun(kind string) string {
	switch kind {
	case KindDef:
		return "declaration"
	case KindRef:
		return "use"
	default:
		return "occurrence"
	}
}
