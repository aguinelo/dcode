// Package memory reads what the agent noted for itself in a previous session.
//
// Reading only, and reading a file a person can edit by hand — that is the whole
// design. The memory lives in the user's repository and is reviewed in a diff,
// because a wrong memory does not sit still: the agent reads its own mistake
// back as fact and acts on it with more confidence than the first time.
package memory

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileName is where a workspace keeps what was learned in it.
//
// Not configurable, and the absence of a key is the decision: a configurable
// path is a memory one machine reads and another does not, for the same check on
// the same repository.
const FileName = ".dcode/memory.md"

// Kind is what a memory is. The list is closed: memory without a kind becomes a
// diary, and a diary becomes noise in two weeks.
type Kind string

const (
	// KindGotcha cost time and will cost it again.
	KindGotcha Kind = "gotcha"
	// KindDecision is a choice and its reason, so it is not relitigated.
	KindDecision Kind = "decision"
	// KindConvention is how this repository does something, discovered rather
	// than documented.
	KindConvention Kind = "convention"
)

// Kinds is the closed list, in the order the spec names them.
func Kinds() []Kind { return []Kind{KindGotcha, KindDecision, KindConvention} }

// Valid reports whether k is one of the three.
func (k Kind) Valid() bool {
	for _, v := range Kinds() {
		if k == v {
			return true
		}
	}
	return false
}

// Entry is one thing that was learned.
type Entry struct {
	Kind    Kind
	Subject string
	// Learned and Commit are what make staleness checkable rather than guessed
	// at. Either may be empty in a file somebody wrote by hand, and that is
	// allowed: a memory with no provenance is still a memory.
	Learned string
	Commit  string
	Body    string
}

// File is what a workspace remembers, and what could not be read.
type File struct {
	Entries []Entry
	// Malformed names the lines that looked like a memory and were not. Reported
	// rather than swallowed: a session must not fail over a crooked block, and a
	// block silently dropped is knowledge lost with nobody told.
	Malformed []string
}

// Path is where the memory of this workspace lives.
func Path(workspace string) string { return filepath.Join(workspace, FileName) }

// Read parses the memory of a workspace.
//
// A missing file is the ordinary case and is silent: a workspace that never
// learned anything is every workspace on its first day, and nothing new may fail
// because there is nothing to read.
func Read(workspace string) (File, error) {
	f, err := os.Open(Path(workspace))
	if err != nil {
		if os.IsNotExist(err) {
			return File{}, nil
		}
		return File{}, err
	}
	defer f.Close()
	return parse(f)
}

// header matches `## kind: subject`, which is the whole grammar.
//
// Deliberately not YAML front matter: a header only a tool reads is a header
// that rots the moment somebody edits the file by hand, and editing by hand is
// the point.
func splitHeader(line string) (Kind, string, bool) {
	rest, ok := strings.CutPrefix(line, "## ")
	if !ok {
		return "", "", false
	}
	kind, subject, ok := strings.Cut(rest, ":")
	if !ok {
		return "", "", false
	}
	k := Kind(strings.ToLower(strings.TrimSpace(kind)))
	subject = strings.TrimSpace(subject)
	if !k.Valid() || subject == "" {
		return "", "", false
	}
	return k, subject, true
}

// provenance reads `<!-- learned 2026-08-18 · commit abc1234 -->`.
func provenance(line string) (learned, commit string, ok bool) {
	inner, ok := strings.CutPrefix(strings.TrimSpace(line), "<!--")
	if !ok {
		return "", "", false
	}
	inner, ok = strings.CutSuffix(strings.TrimSpace(inner), "-->")
	if !ok {
		return "", "", false
	}
	for _, part := range strings.Split(inner, "·") {
		part = strings.TrimSpace(part)
		if v, found := strings.CutPrefix(part, "learned "); found {
			learned = strings.TrimSpace(v)
		}
		if v, found := strings.CutPrefix(part, "commit "); found {
			commit = strings.TrimSpace(v)
		}
	}
	return learned, commit, learned != "" || commit != ""
}

func parse(r interface{ Read([]byte) (int, error) }) (File, error) {
	var out File
	var cur *Entry
	var body []string

	flush := func() {
		if cur == nil {
			return
		}
		cur.Body = strings.TrimSpace(strings.Join(body, "\n"))
		out.Entries = append(out.Entries, *cur)
		cur, body = nil, nil
	}

	sc := bufio.NewScanner(r)
	// A memory holds a stack trace or a command someone pasted; the default
	// limit would cut it and report the tail as a separate line.
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)

	for sc.Scan() {
		line := sc.Text()
		if k, subject, ok := splitHeader(line); ok {
			flush()
			cur = &Entry{Kind: k, Subject: subject}
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "## ") {
			// It looked like a memory and was not: an unknown kind, or no
			// subject. Saying so is the difference between a file somebody can
			// fix and a file that quietly holds less than it looks like.
			out.Malformed = append(out.Malformed, strings.TrimSpace(line))
			continue
		}
		if cur == nil {
			continue
		}
		if learned, commit, ok := provenance(line); ok && cur.Learned == "" && cur.Commit == "" {
			cur.Learned, cur.Commit = learned, commit
			continue
		}
		body = append(body, line)
	}
	flush()
	if err := sc.Err(); err != nil {
		return out, fmt.Errorf("memory: %w", err)
	}
	return out, nil
}
