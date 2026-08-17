package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
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
		return s.notFound(r.Name(), in.Path, err).Result(), nil
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
		return s.notFound(w.Name(), in.Path, err).Result(), nil
	}

	// How it stood before, so the turn can be undone. Taken here rather than
	// at the top because everything above can still refuse, and a snapshot of
	// a write that never happened is a file undo would rewrite for nothing.
	s.Snapshot(abs)
	if err := writeFile(abs, in.Content, s.Limits.AtomicWrite); err != nil {
		return errf(w.Name(), CodeNotFound, "", "could not write %s: %v", in.Path, err).Result(), nil
	}
	s.MarkRead(abs, in.Content, 0)
	s.MarkWritten(abs)

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
	// Edits is the batch form. When present it wins, and the four fields above
	// are ignored: two ways of saying the same call in one call is ambiguity
	// nobody can resolve from the outside.
	Edits []editOp `json:"edits,omitempty"`
}

func (Edit) Name() string { return "edit" }

func (Edit) Description() string {
	return "Replace exact strings in files. You must read a file before editing it. " +
		"old_string must match exactly and, unless replace_all is set, must be unique — " +
		"include surrounding lines to disambiguate. " +
		"Give edits to make several changes at once, across as many files as you need: " +
		"they are ALL checked before ANY is written, so a batch either lands whole or " +
		"changes nothing. Prefer one batch over several calls when the changes belong " +
		"together — half a rename is worse than no rename. " +
		"Edits to the same file apply in order, each seeing what the one before it left."
}

func (Edit) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{` +
		`"path":{"type":"string","description":"The file to edit. Omit when using edits."},` +
		`"old_string":{"type":"string","description":"Exact text to replace."},` +
		`"new_string":{"type":"string","description":"Replacement text."},` +
		`"replace_all":{"type":"boolean","description":"Replace every occurrence."},` +
		`"edits":{"type":"array","description":` +
		`"Several changes applied as one. All are checked before any is written.",` +
		`"items":{"type":"object","properties":{` +
		`"path":{"type":"string"},` +
		`"old_string":{"type":"string"},` +
		`"new_string":{"type":"string"},` +
		`"replace_all":{"type":"boolean"}},` +
		`"required":["path","old_string","new_string"]}}}}`)
}

func (e Edit) Declare(input json.RawMessage) (policy.Request, error) {
	var in EditInput
	if err := decode(e.Name(), input, &in); err != nil {
		return policy.Request{}, err
	}
	// Every path, once. The policy decides on paths, and a path it never saw is
	// a path nobody was asked about; the same file twice is one question.
	seen := map[string]struct{}{}
	var paths []policy.Access
	for _, op := range in.ops() {
		if _, dup := seen[op.Path]; dup {
			continue
		}
		seen[op.Path] = struct{}{}
		paths = append(paths, policy.Access{Path: op.Path, Write: true})
	}
	return policy.Request{Tool: e.Name(), Paths: paths}, nil
}

func (e Edit) Execute(_ context.Context, input json.RawMessage, s *State) (Result, error) {
	var in EditInput
	if err := decode(e.Name(), input, &in); err != nil {
		return err.(*ToolError).Result(), nil
	}
	ops := in.ops()
	if len(ops) == 0 {
		return errf(e.Name(), CodeBadInput,
			"Give path, old_string and new_string, or a list of edits.",
			"nothing to edit").Result(), nil
	}

	planned, terr := plan(e.Name(), ops, s)
	if terr != nil {
		return terr.Result(), nil
	}
	if terr := commit(e.Name(), planned, s); terr != nil {
		return terr.Result(), nil
	}

	// One file keeps the old rule about echoing the diff. For a batch the diff
	// is the only account of what happened across files, so it always goes.
	echo := len(planned) > 1
	if len(planned) == 1 {
		echo = echoDiff(s.Limits.EditEchoDiff, ops[0].ReplaceAll, planned[0].changes)
	}
	return report(planned, echo), nil
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

func (s *State) notFound(tool, path string, err error) *ToolError {
	// The suggestion is offered only when the session can act on it.
	find, list := "Check the path.", "Name a file inside it."
	if s.has("glob") {
		find = "Check the path, or use glob to find it."
		list = "Use glob to list what is in it, or name a file inside it."
	}
	if os.IsNotExist(err) {
		return errf(tool, CodeNotFound, find, "%s does not exist", path)
	}
	// Reading a directory is a thing a model does constantly, and the answer
	// used to be the wrapped Go error — which named the absolute path and said
	// nothing about what to do. The model would then read that path, and spend
	// the round discovering it was the same directory.
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) && errors.Is(pathErr.Err, syscall.EISDIR) {
		return errf(tool, CodeNotFound, list, "%s is a directory", path)
	}
	// The workspace path is never in an error either. It varies by machine,
	// which breaks golden output, and a model that reads it starts treating an
	// absolute path as a thing it may use (RN-7).
	return errf(tool, CodeNotFound, "", "could not read %s: %v", path, scrub(err, path))
}

// scrub keeps a filesystem error's meaning and drops the absolute path it
// carries, which is everything before the last colon.
func scrub(err error, path string) error {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		return fmt.Errorf("%s: %w", path, pathErr.Err)
	}
	return err
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
