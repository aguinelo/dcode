package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/aguinelo/dcode/internal/policy"
)

// ---------- read ----------

// Read returns file contents with line numbers, so the model and edit share one
// frame of reference.
type Read struct{}

// ReadInput is the argument shape.
type ReadInput struct {
	Path   string `json:"path"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

func (Read) Name() string { return "read" }

func (Read) Description() string {
	return "Read a file from the workspace. Output is line-numbered. " +
		"You must read a file before editing it."
}

func (Read) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{` +
		`"path":{"type":"string","description":"File path, absolute or relative to the workspace."},` +
		`"offset":{"type":"integer","description":"First line to read, 1-based."},` +
		`"limit":{"type":"integer","description":"Maximum lines to read."}},` +
		`"required":["path"]}`)
}

func (r Read) Declare(input json.RawMessage) (policy.Request, error) {
	var in ReadInput
	if err := decode(r.Name(), input, &in); err != nil {
		return policy.Request{}, err
	}
	return policy.Request{Tool: r.Name(), Paths: []policy.Access{{Path: in.Path}}}, nil
}

func (r Read) Execute(_ context.Context, input json.RawMessage, s *State) (Result, error) {
	var in ReadInput
	if err := decode(r.Name(), input, &in); err != nil {
		return err.(*ToolError).Result(), nil
	}
	abs, terr := resolvePath(r.Name(), s, in.Path, false)
	if terr != nil {
		return terr.Result(), nil
	}

	raw, err := os.ReadFile(abs)
	if err != nil {
		return notFound(r.Name(), in.Path, err).Result(), nil
	}
	// A binary dump would flood the context with bytes the model cannot use
	// and cannot summarise.
	if !utf8.Valid(raw) || strings.IndexByte(string(raw), 0) >= 0 {
		return Result{Output: fmt.Sprintf("%s is a binary file (%d bytes); not shown.",
			in.Path, len(raw))}, nil
	}

	content := string(raw)
	lines := strings.Split(content, "\n")
	limit := in.Limit
	if limit <= 0 {
		limit = s.Limits.ReadMaxLines
	}
	start := in.Offset - 1
	if start < 0 {
		start = 0
	}
	if start > len(lines) {
		start = len(lines)
	}
	end := start + limit
	if end > len(lines) {
		end = len(lines)
	}

	var b strings.Builder
	for i := start; i < end; i++ {
		line := lines[i]
		if n := s.Limits.ReadMaxLineLength; n > 0 && len(line) > n {
			// Truncate the line, not the numbering: a minified file must not
			// cost the whole window, and the line numbers still have to match
			// what edit will see.
			line = line[:n] + fmt.Sprintf(" … (%d more bytes on this line)", len(line)-n)
		}
		fmt.Fprintf(&b, "%6d\t%s\n", i+1, line)
	}

	remaining := len(lines) - end
	res := Result{Output: b.String(), Meta: Meta{Lines: end - start, Files: 1}}
	if remaining > 0 {
		res.Truncated = true
		res.Remaining = remaining
		res.Output += fmt.Sprintf("\n… %d more lines. Use offset to continue.\n", remaining)
	}

	// Only the exact content read is recorded. Marking the whole file after a
	// partial read would let a later edit act on a region never seen.
	if start == 0 && remaining == 0 {
		s.MarkRead(abs, content, 0)
	}
	return res, nil
}

// ---------- write ----------

// Write creates a file or replaces one wholesale.
type Write struct{}

// WriteInput is the argument shape.
type WriteInput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (Write) Name() string { return "write" }

func (Write) Description() string {
	return "Create a file, or replace an existing one entirely. " +
		"Replacing an existing file requires reading it first. " +
		"Prefer edit for changing part of a file."
}

func (Write) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{` +
		`"path":{"type":"string"},"content":{"type":"string"}},` +
		`"required":["path","content"]}`)
}

func (w Write) Declare(input json.RawMessage) (policy.Request, error) {
	var in WriteInput
	if err := decode(w.Name(), input, &in); err != nil {
		return policy.Request{}, err
	}
	return policy.Request{Tool: w.Name(), Paths: []policy.Access{{Path: in.Path, Write: true}}}, nil
}

func (w Write) Execute(_ context.Context, input json.RawMessage, s *State) (Result, error) {
	var in WriteInput
	if err := decode(w.Name(), input, &in); err != nil {
		return err.(*ToolError).Result(), nil
	}
	abs, terr := resolvePath(w.Name(), s, in.Path, true)
	if terr != nil {
		return terr.Result(), nil
	}

	existing, err := os.ReadFile(abs)
	switch {
	case err == nil:
		// Overwriting carries the same risk as editing, so it carries the same
		// invariant.
		if e := s.CheckEditable(abs, string(existing)); e != nil {
			e.Tool = w.Name()
			return e.Result(), nil
		}
	case !os.IsNotExist(err):
		return notFound(w.Name(), in.Path, err).Result(), nil
	}

	if err := writeFile(abs, in.Content, s.Limits.AtomicWrite); err != nil {
		return errf(w.Name(), CodeNotFound, "", "could not write %s: %v", in.Path, err).Result(), nil
	}
	s.MarkRead(abs, in.Content, 0)

	verb := "created"
	if err == nil {
		verb = "replaced"
	}
	written := countLines(in.Content)
	meta := Meta{Files: 1, Lines: written, Added: written}
	previous := ""
	if verb == "replaced" {
		previous = string(existing)
		meta.Added, meta.Removed = lineDelta(previous, in.Content)
	}
	meta.Diff = UnifiedDiff(previous, in.Content, in.Path)
	return Result{
		Output: fmt.Sprintf("%s %s (%d bytes)", verb, in.Path, len(in.Content)),
		Meta:   meta,
	}, nil
}

// ---------- edit ----------

// Edit replaces an exact string. The most consequential tool in the set.
type Edit struct{}

// EditInput is the argument shape.
type EditInput struct {
	Path       string `json:"path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

func (Edit) Name() string { return "edit" }

func (Edit) Description() string {
	return "Replace an exact string in a file. You must read the file first. " +
		"old_string must match exactly and, unless replace_all is set, must be unique — " +
		"include surrounding lines to disambiguate."
}

func (Edit) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{` +
		`"path":{"type":"string"},` +
		`"old_string":{"type":"string","description":"Exact text to replace."},` +
		`"new_string":{"type":"string","description":"Replacement text."},` +
		`"replace_all":{"type":"boolean","description":"Replace every occurrence."}},` +
		`"required":["path","old_string","new_string"]}`)
}

func (e Edit) Declare(input json.RawMessage) (policy.Request, error) {
	var in EditInput
	if err := decode(e.Name(), input, &in); err != nil {
		return policy.Request{}, err
	}
	return policy.Request{Tool: e.Name(), Paths: []policy.Access{{Path: in.Path, Write: true}}}, nil
}

func (e Edit) Execute(_ context.Context, input json.RawMessage, s *State) (Result, error) {
	var in EditInput
	if err := decode(e.Name(), input, &in); err != nil {
		return err.(*ToolError).Result(), nil
	}
	if in.OldString == in.NewString {
		return errf(e.Name(), CodeNoOpEdit,
			"Check whether the edit was already applied.",
			"old_string and new_string are identical; nothing would change").Result(), nil
	}
	abs, terr := resolvePath(e.Name(), s, in.Path, true)
	if terr != nil {
		return terr.Result(), nil
	}

	raw, err := os.ReadFile(abs)
	if err != nil {
		return notFound(e.Name(), in.Path, err).Result(), nil
	}
	content := string(raw)

	if terr := s.CheckEditable(abs, content); terr != nil {
		terr.Tool = e.Name()
		return terr.Result(), nil
	}

	count := strings.Count(content, in.OldString)
	switch {
	case count == 0:
		return errf(e.Name(), CodeNoMatch,
			"Read the file again — the text may have changed, or the whitespace may differ.",
			"old_string was not found in %s", in.Path).Result(), nil
	case count > 1 && !in.ReplaceAll:
		// Picking the first occurrence is the tempting implementation and the
		// wrong one: it is right most of the time and, when it is wrong, it
		// edits the wrong place silently.
		return (&ToolError{
			Tool: e.Name(), Code: CodeAmbiguousMatch,
			Reason: fmt.Sprintf("old_string appears %d times in %s; nothing was changed", count, in.Path),
			Hint:   "Include surrounding lines to make it unique, or set replace_all if every occurrence should change.",
		}).Result(), nil
	}

	updated := strings.ReplaceAll(content, in.OldString, in.NewString)
	if err := writeFile(abs, updated, s.Limits.AtomicWrite); err != nil {
		return errf(e.Name(), CodeNotFound, "", "could not write %s: %v", in.Path, err).Result(), nil
	}
	// Re-marking with the new content is what lets a second edit follow without
	// a re-read; forgetting it makes the next edit fail as file_changed.
	s.MarkRead(abs, updated, 0)

	added, removed := lineDelta(content, updated)
	return Result{
		Output: fmt.Sprintf("edited %s (%d replacement(s), +%d −%d)",
			in.Path, count, added, removed),
		Meta: Meta{
			Files: 1, Added: added, Removed: removed,
			Diff: UnifiedDiff(content, updated, in.Path),
		},
	}, nil
}

// ---------- shared ----------

func resolvePath(tool string, s *State, path string, write bool) (string, *ToolError) {
	if strings.TrimSpace(path) == "" {
		return "", errf(tool, CodeBadInput, "", "path is required")
	}
	a, err := s.Resolver.Resolve(path, write)
	if err != nil {
		return "", errf(tool, CodeNotFound, "", "could not resolve %s: %v", path, err)
	}
	return a.Path, nil
}

func notFound(tool, path string, err error) *ToolError {
	if os.IsNotExist(err) {
		return errf(tool, CodeNotFound,
			"Check the path, or use glob to find it.", "%s does not exist", path)
	}
	return errf(tool, CodeNotFound, "", "could not read %s: %v", path, err)
}

// writeFile writes atomically by default: a crash between truncate and write
// would otherwise leave a half-written source file, which is worse than the
// original state.
func writeFile(path, content string, atomic bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if !atomic {
		return os.WriteFile(path, []byte(content), 0o644)
	}
	// The temp file lives in the destination directory so rename stays on one
	// filesystem, which is the only way it is atomic.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".dcode-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func lineDelta(before, after string) (added, removed int) {
	b := strings.Count(before, "\n")
	a := strings.Count(after, "\n")
	if a > b {
		return a - b, 0
	}
	return 0, b - a
}
