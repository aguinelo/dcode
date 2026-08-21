package tui

import (
	"sort"
	"strings"
)

// The sidebar's file list is derived from the entries, and the entries are the
// reduction of the event log. So it is derived from the log, as the design asks
// — by one hop rather than by a second reducer beside the first.
//
// The design puts a `tree` field on the model. A field would be a SECOND thing
// reduced from the same events, and two reductions of one log is two things
// that can disagree; deriving here makes "the same session reopened reproduces
// the same tree" true by construction instead of by care.

// FileState is what the last event to touch a path said about it.
type FileState int

const (
	// FileReading and FileWriting are calls still in flight.
	FileReading FileState = iota
	FileWriting
	// FileDone is a call that reported back, and FileFailed one that did not
	// or came back an error.
	FileDone
	FileFailed
)

// FileRow is one path the turn touched, ready to draw.
type FileRow struct {
	// Path is the whole path; Label is what the row shows, already indented
	// and already compacted where a folder had a single child.
	Path  string
	Label string
	Depth int
	State FileState
	// Added is the line count the tool reported, never parsed back out of its
	// summary.
	Added int
	// Folder marks a row that stands for a directory rather than a file.
	Folder bool
}

// touchedFiles reports the paths this turn touched, in the order a reader
// expects: sorted by path, so the list does not reshuffle under them while work
// is going on.
//
// Only calls that name a real path. A grep over a pattern touched no file until
// it says which, and a file the turn has not created is not drawn — the design
// is explicit about that, and it is the invariant that keeps a sidebar from
// promising something that is not on disk.
func touchedFiles(entries []Entry) []FileRow {
	type acc struct {
		state FileState
		added int
	}
	seen := map[string]*acc{}
	for _, e := range entries {
		if e.Kind != KindTool || e.Target == "" || !namesAFile(e.Tool) || !looksLikePath(e.Target) {
			continue
		}
		a, ok := seen[e.Target]
		if !ok {
			a = &acc{}
			seen[e.Target] = a
		}
		switch {
		case e.IsError:
			a.state = FileFailed
		case e.Running:
			if writes(e.Tool) {
				a.state = FileWriting
			} else {
				a.state = FileReading
			}
		default:
			a.state = FileDone
		}
		if e.Added > a.added {
			a.added = e.Added
		}
	}

	paths := make([]string, 0, len(seen))
	for p := range seen {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	out := make([]FileRow, 0, len(paths))
	for _, p := range paths {
		out = append(out, FileRow{Path: p, State: seen[p].state, Added: seen[p].added})
	}
	return out
}

// namesAFile says whether this tool's target is a path at all.
//
// A delegated child's target is its NAME, and a name with no punctuation in it
// looks exactly like a filename — `alpha` and `bravo` walked into the file list
// and sat there as if they were on disk. Deciding by tool rather than by the
// shape of the string is the only reading that cannot be fooled.
func namesAFile(tool string) bool {
	switch strings.ToLower(tool) {
	case "explore", "bash", "process", "plan", "remember":
		return false
	}
	return true
}

func writes(tool string) bool {
	switch strings.ToLower(tool) {
	case "write", "edit":
		return true
	}
	return false
}

// looksLikePath keeps patterns and commands out of a list of files.
//
// A `bash` target is a command line and a `grep` target is a regex; drawing
// either as a file would put something in the sidebar that cannot be opened.
func looksLikePath(s string) bool {
	if s == "" || strings.ContainsAny(s, " \t*?[]()|\\$^") {
		return false
	}
	return true
}

// FileTree lays the touched paths out as a folder row followed by its files.
//
// Two levels, not a full tree, and the column's width is the reason: it is
// twenty to thirty characters, and every level of indentation is two of them
// taken from the only part that identifies a file — its name. A folder row
// carries its whole path, which is the design's single-child compaction taken
// to its conclusion: `internal/tui/` on one line rather than two rows nobody
// needs to see on their own.
func FileTree(entries []Entry) []FileRow {
	files := touchedFiles(entries)
	if len(files) == 0 {
		return nil
	}

	var out []FileRow
	dir := "\x00" // no real directory, so the first file always opens one
	for _, f := range files {
		d := ""
		if at := strings.LastIndex(f.Path, "/"); at >= 0 {
			d = f.Path[:at+1]
		}
		if d != dir {
			dir = d
			if d != "" {
				out = append(out, FileRow{Path: strings.TrimSuffix(d, "/"), Label: d, Folder: true})
			}
		}
		f.Label = f.Path[strings.LastIndex(f.Path, "/")+1:]
		if d != "" {
			f.Depth = 1
		}
		out = append(out, f)
	}
	return out
}
