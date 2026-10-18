// Package orchestrate conducts one project: its orchestrator, its children and
// the pipeline that decides what reaches main.
//
// This file is the decision log, the first piece of it: every decision the
// orchestrator or the harness takes is one line, with its reason, before the
// transition it decides happens.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §8, §9.1.
package orchestrate

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/aguinelo/dcode/internal/protocol"
)

const (
	decisionsFile = "decisions.jsonl"
	// logVersion is the format of every line, its "v".
	logVersion = 1
	// maxLine is the ceiling of one line, newline included (.config §4).
	maxLine = 64 << 10
)

var (
	errNoReason    = errors.New("orchestrate: a decision without a reason is not recorded")
	errUnknownKind = errors.New("orchestrate: unknown decision kind")
	errClosed      = errors.New("orchestrate: the decision log is closed")
)

// kinds is the list of .p §9.1. A kind outside it is refused rather than
// written: a reader filters by kind, and a misspelt one is a decision no
// filter finds.
var kinds = map[string]bool{
	"project.opened": true, "dispatch": true, "dispatch.refused": true, "model.chosen": true,
	"worktree.created": true, "hooks.changed": true, "child.state": true, "child.completed": true,
	"child.returned": true, "owns.checked": true, "review.verdict": true, "red_test.confirmed": true,
	"branch.updated": true, "gate.measured": true, "pr.opened": true, "pr.updated": true, "question.asked": true,
	"question.answered": true, "merge.queued": true, "merge.completed": true, "merge.refused": true,
	"main.health": true, "ceiling.reached": true, "forge.local": true, "worktree.removed": true,
	"child.closed": true, "daemon.restart": true,
}

// projectID is Project.ID: 24 hex of a sha256. It names a directory, so
// anything else is refused before it reaches a path.
var projectID = regexp.MustCompile(`^[0-9a-f]{24}$`)

// Decision is one line of the log (.p §9.1).
//
// V, Seq, At and Project belong to the log: Record fills them, and a Project
// naming another project is refused. Measured and Data are JSON objects the
// caller encodes — a GateResult, a Verdict, a MainHealth — kept as raw bytes
// so a reader gets back exactly what was written.
type Decision struct {
	V       int       `json:"v"`
	Seq     uint64    `json:"seq"`
	At      time.Time `json:"at"`
	Project string    `json:"project"`
	Kind    string    `json:"kind"`
	Child   string    `json:"child,omitempty"`
	Session string    `json:"session,omitempty"`
	// Why is the model's reason when it chose, the harness's when a rule did.
	// Never empty.
	Why string `json:"why"`
	// Claim is what a child said, kept beside what was measured and never
	// read as proof.
	Claim    string          `json:"claim,omitempty"`
	Measured json.RawMessage `json:"measured,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
}

// DecisionLog is one project's decisions.jsonl, append-only.
//
// Record returns only once the line is on disk, and a line that could not be
// written is an error: the caller's transition does not happen (.p §8, the log
// before state.json before the wire). Safe for concurrent use.
type DecisionLog struct {
	mu      sync.Mutex
	f       *os.File
	path    string
	project string
	seq     uint64
	size    int64
	torn    []byte
	// broken is set when a failed write could not be taken back: the file
	// may end in a partial line, and appending after it would join two.
	broken error
	now    func() time.Time
}

// OpenDecisionLog opens the log of project p under dir, orchestrator.dir,
// creating <dir>/<p.ID>/decisions.jsonl owner-only when it is not there.
//
// The seq continues from the last line. A last line with no newline is a write
// a crash cut short — Record had not returned, so nothing acted on it — and is
// cut off; Torn says what it was. A complete line that does not read, or a seq
// that does not follow the one before, is a log this writer did not leave, and
// the log refuses to open rather than carry the gap forward.
func OpenDecisionLog(dir string, p protocol.Project) (*DecisionLog, error) {
	if !filepath.IsAbs(dir) {
		return nil, fmt.Errorf("orchestrate: the log's directory must be absolute, got %q", dir)
	}
	if !projectID.MatchString(p.ID) {
		return nil, fmt.Errorf("orchestrate: %q is not a project id (24 hex)", p.ID)
	}
	dir = filepath.Clean(dir)
	if p.Root != "" && within(dir, filepath.Clean(p.Root)) {
		return nil, fmt.Errorf("orchestrate: %s is inside the project %s it would conduct; orchestrator.dir must be outside it",
			dir, p.Root)
	}

	pdir := filepath.Join(dir, p.ID)
	if err := os.MkdirAll(pdir, 0o700); err != nil {
		return nil, fmt.Errorf("orchestrate: create %s: %w", pdir, err)
	}
	path := filepath.Join(pdir, decisionsFile)

	data, err := os.ReadFile(path)
	created := errors.Is(err, fs.ErrNotExist)
	if err != nil && !created {
		return nil, fmt.Errorf("orchestrate: read %s: %w", path, err)
	}
	seq, whole, err := scan(path, data)
	if err != nil {
		return nil, err
	}
	var torn []byte
	if whole < len(data) {
		torn = bytes.Clone(data[whole:])
		if err := os.Truncate(path, int64(whole)); err != nil {
			return nil, fmt.Errorf("orchestrate: cut the torn last line of %s: %w", path, err)
		}
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("orchestrate: open %s: %w", path, err)
	}
	if err := settle(f, pdir, created, torn != nil); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("orchestrate: %s: %w", path, err)
	}
	return &DecisionLog{
		f: f, path: path, project: p.ID, seq: seq, size: int64(whole), torn: torn,
		now: time.Now,
	}, nil
}

// settle makes what opening changed durable: a new file's directory entry,
// and a cut torn line.
func settle(f *os.File, dir string, created, cut bool) error {
	if cut {
		if err := f.Sync(); err != nil {
			return err
		}
	}
	if !created {
		return nil
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

// scan reads every complete line and returns the last seq and how many bytes
// the complete lines take. What follows the last newline is the torn tail.
func scan(path string, data []byte) (uint64, int, error) {
	var seq uint64
	whole := 0
	for n := 1; ; n++ {
		i := bytes.IndexByte(data[whole:], '\n')
		if i < 0 {
			return seq, whole, nil
		}
		var d Decision
		if err := json.Unmarshal(data[whole:whole+i], &d); err != nil {
			return 0, 0, fmt.Errorf("orchestrate: %s:%d: unreadable decision: %w", path, n, err)
		}
		if d.V != logVersion {
			return 0, 0, fmt.Errorf("orchestrate: %s:%d: format v%d, this writer knows v%d", path, n, d.V, logVersion)
		}
		if d.Seq != seq+1 {
			return 0, 0, fmt.Errorf("orchestrate: %s:%d: seq %d follows %d; the log has a gap or a repetition",
				path, n, d.Seq, seq)
		}
		seq = d.Seq
		whole += i + 1
	}
}

// Record writes d as one line, synced, and returns its seq.
//
// A decision without a reason, of a kind outside .p §9.1, or for another
// project is refused and nothing is written. A write that fails is returned
// and spends no seq; the partial line is taken back, and when even that fails
// the log refuses every later decision rather than append after it.
func (l *DecisionLog) Record(d Decision) (uint64, error) {
	if strings.TrimSpace(d.Why) == "" {
		return 0, fmt.Errorf("%w (kind %q)", errNoReason, d.Kind)
	}
	if !kinds[d.Kind] {
		return 0, fmt.Errorf("%w %q", errUnknownKind, d.Kind)
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if d.Project != "" && d.Project != l.project {
		return 0, fmt.Errorf("orchestrate: a decision for project %s does not go in the log of %s", d.Project, l.project)
	}
	if l.f == nil {
		return 0, errClosed
	}
	if l.broken != nil {
		return 0, fmt.Errorf("orchestrate: %s refuses decisions after a write it could not take back: %w", l.path, l.broken)
	}

	d.V, d.Seq, d.At, d.Project = logVersion, l.seq+1, l.now().UTC(), l.project
	line, err := encodeLine(d)
	if err != nil {
		return 0, err
	}
	if err := l.append(line); err != nil {
		return 0, fmt.Errorf("orchestrate: decision %d (%s) not written to %s: %w", d.Seq, d.Kind, l.path, err)
	}
	l.seq = d.Seq
	l.size += int64(len(line))
	return d.Seq, nil
}

// append writes and syncs one line, and takes it back when either fails.
func (l *DecisionLog) append(line []byte) error {
	_, err := l.f.Write(line)
	if err == nil {
		err = l.f.Sync()
	}
	if err == nil {
		return nil
	}
	if terr := l.f.Truncate(l.size); terr != nil {
		l.broken = errors.Join(err, terr)
	} else if serr := l.f.Sync(); serr != nil {
		l.broken = errors.Join(err, serr)
	}
	return err
}

// Torn is the partial last line opening cut off, nil when there was none. The
// caller says it — in daemon.restart, and to the person — because a cut that
// nobody reports is a line lost in silence.
func (l *DecisionLog) Torn() []byte {
	l.mu.Lock()
	defer l.mu.Unlock()
	return bytes.Clone(l.torn)
}

// Close closes the file. A Record after it is refused.
func (l *DecisionLog) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f == nil {
		return nil
	}
	err := l.f.Close()
	l.f = nil
	return err
}

// within reports whether path is root or below it.
func within(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// encode is one line, newline included, with <, > and & as written: a reader
// of the log reads it as text, not inside HTML.
func encode(d Decision) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(d); err != nil {
		return nil, fmt.Errorf("orchestrate: encode %s decision: %w", d.Kind, err)
	}
	return b.Bytes(), nil
}

// encodeLine is encode under the 64 KiB ceiling: while the line is over it,
// the largest of the free-text fields is cut and says how many bytes it lost
// (.p §9.1). A cut Measured or Data becomes the text of what was kept, a JSON
// string, since half an object is not an object.
func encodeLine(d Decision) ([]byte, error) {
	line, err := encode(d)
	if err != nil || len(line) <= maxLine {
		return line, err
	}
	fields := fieldsOf(&d)
	done := map[int]bool{}
	for len(line) > maxLine {
		pick := -1
		for i, f := range fields {
			if !done[i] && f.get() != "" && (pick < 0 || len(f.get()) > len(fields[pick].get())) {
				pick = i
			}
		}
		if pick < 0 {
			break
		}
		done[pick] = true
		f, orig := fields[pick], fields[pick].get()
		for excess := len(line) - maxLine; ; {
			f.set(cut(orig, excess))
			if line, err = encode(d); err != nil {
				return nil, err
			}
			if len(line) <= maxLine || excess >= len(orig) {
				break
			}
			excess += len(line) - maxLine
		}
	}
	if len(line) > maxLine {
		return nil, fmt.Errorf("orchestrate: a %s decision is %d bytes with every field cut, over the %d of a line",
			d.Kind, len(line), maxLine)
	}
	return line, nil
}

type field struct {
	get func() string
	set func(string)
}

func fieldsOf(d *Decision) []field {
	raw := func(m *json.RawMessage) field {
		return field{
			get: func() string { return string(*m) },
			set: func(s string) { *m, _ = json.Marshal(s) }, // a string always encodes
		}
	}
	return []field{
		{func() string { return d.Why }, func(s string) { d.Why = s }},
		{func() string { return d.Claim }, func(s string) { d.Claim = s }},
		raw(&d.Measured),
		raw(&d.Data),
	}
}

// cut drops at least excess bytes from the end of s, at a rune boundary, and
// says how many it dropped.
func cut(s string, excess int) string {
	keep := max(len(s)-excess, 0)
	for keep > 0 && !utf8.RuneStart(s[keep]) {
		keep--
	}
	return fmt.Sprintf("%s…[%d bytes cortados]", s[:keep], len(s)-keep)
}
