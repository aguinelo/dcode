package tools

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// One edit inside a batch. The same four fields the single form takes, because
// a batch of one has to mean exactly what one meant.
type editOp struct {
	Path       string `json:"path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

// ops flattens whichever shape the call used into one list.
//
// The single form is the batch of one, not a separate path through the tool.
// Two implementations of "replace this text" is how the two drift, and the
// consequential one drifts silently.
func (in EditInput) ops() []editOp {
	if len(in.Edits) > 0 {
		return in.Edits
	}
	if in.Path == "" && in.OldString == "" && in.NewString == "" {
		return nil
	}
	return []editOp{{
		Path: in.Path, OldString: in.OldString,
		NewString: in.NewString, ReplaceAll: in.ReplaceAll,
	}}
}

// pending is a file the batch has finished computing but not yet written.
type pending struct {
	abs     string
	shown   string // the path as the model wrote it, for messages
	before  string
	after   string
	changes int
}

// plan works out what every edit would do, and writes nothing.
//
// This is the whole reason the batch exists. A rename across twelve files
// applied one call at a time fails on the seventh and leaves six done: the code
// no longer builds and the reason is spread across a conversation. Checking
// everything first turns that into one refusal naming the edit that could not
// be made, against files nobody has touched.
func plan(tool string, ops []editOp, s *State) ([]*pending, *ToolError) {
	// Keyed by resolved path so several edits to one file chain through the
	// same buffer, each seeing what the one before it left.
	byPath := map[string]*pending{}
	var order []*pending

	for i, op := range ops {
		where := fmt.Sprintf("edit %d of %d (%s)", i+1, len(ops), op.Path)

		if op.OldString == op.NewString {
			return nil, &ToolError{
				Tool: tool, Code: CodeNoOpEdit,
				Reason: where + ": old_string and new_string are identical; nothing would change",
				Hint:   "Check whether the edit was already applied.",
			}
		}
		abs, terr := resolvePath(tool, s, op.Path, true)
		if terr != nil {
			terr.Reason = where + ": " + terr.Reason
			return nil, terr
		}

		p, seen := byPath[abs]
		if !seen {
			raw, err := readFileFor(abs)
			if err != nil {
				terr := s.notFound(tool, op.Path, err)
				terr.Reason = where + ": " + terr.Reason
				return nil, terr
			}
			if terr := s.CheckEditable(abs, raw); terr != nil {
				terr.Tool = tool
				terr.Reason = where + ": " + terr.Reason
				return nil, terr
			}
			p = &pending{abs: abs, shown: op.Path, before: raw, after: raw}
			byPath[abs] = p
			order = append(order, p)
		}

		count := strings.Count(p.after, op.OldString)
		switch {
		case count == 0:
			return nil, &ToolError{
				Tool: tool, Code: CodeNoMatch,
				Reason: where + ": old_string was not found",
				Hint: "Read the file again — the text may have changed, the whitespace may differ, " +
					"or an earlier edit in this batch may have already changed it.",
			}
		case count > 1 && !op.ReplaceAll:
			// Picking the first occurrence is the tempting implementation and
			// the wrong one: right most of the time and, when it is wrong, it
			// edits the wrong place silently.
			return nil, &ToolError{
				Tool: tool, Code: CodeAmbiguousMatch,
				Reason: fmt.Sprintf("%s: old_string appears %d times; nothing was changed", where, count),
				Hint:   "Include surrounding lines to make it unique, or set replace_all if every occurrence should change.",
			}
		}
		p.after = strings.ReplaceAll(p.after, op.OldString, op.NewString)
		p.changes += count
	}
	return order, nil
}

// commit writes the planned files and records them.
//
// A write can still fail here — a full disk, a permission changed under us —
// and what has landed by then has landed. That is not the failure the batch is
// for: undo covers the turn, and refusing to write anything because the last
// write might fail would mean never writing at all.
func commit(tool string, planned []*pending, s *State) *ToolError {
	for _, p := range planned {
		s.Snapshot(p.abs)
		if err := writeFile(p.abs, p.after, s.Limits.AtomicWrite); err != nil {
			return &ToolError{
				Tool: tool, Code: CodeNotFound,
				Reason: fmt.Sprintf("could not write %s: %v", p.shown, err),
			}
		}
		// Re-marking with the new content is what lets a later edit follow
		// without a re-read; forgetting it makes the next edit fail as
		// file_changed.
		s.MarkRead(p.abs, p.after, 0)
		s.MarkWritten(p.abs)
	}
	return nil
}

// report accounts for every file, because which file moved how far is what a
// person scans for. A single total hides the one that did not change.
func report(planned []*pending, echo bool) Result {
	sort.Slice(planned, func(i, j int) bool { return planned[i].shown < planned[j].shown })

	var lines []string
	var diffs []string
	totalAdded, totalRemoved := 0, 0
	for _, p := range planned {
		added, removed := lineDelta(p.before, p.after)
		totalAdded += added
		totalRemoved += removed
		lines = append(lines, fmt.Sprintf("%s (%d replacement(s), +%d −%d)",
			p.shown, p.changes, added, removed))
		diffs = append(diffs, UnifiedDiff(p.before, p.after, p.shown))
	}

	head := "edited " + lines[0]
	if len(lines) > 1 {
		head = fmt.Sprintf("edited %d files:\n  %s", len(lines), strings.Join(lines, "\n  "))
	}
	diff := strings.Join(diffs, "\n")
	if echo {
		// The diff already declares its own truncation, which is what stops
		// the model concluding about a part it never saw with the confidence
		// of having seen all of it.
		head += "\n\n" + diff
	}
	return Result{
		Output: head,
		Meta: Meta{
			Files: len(planned), Added: totalAdded, Removed: totalRemoved,
			// Meta.Diff goes to the client in every mode. What the key
			// controls is only whether the model also pays for it.
			Diff: diff,
		},
	}
}

// readFileFor reads a file for editing. Separate so plan reads exactly once per
// path however many edits target it.
func readFileFor(abs string) (string, error) {
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
