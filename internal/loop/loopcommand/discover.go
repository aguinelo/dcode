package loopcommand

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/aguinelo/dcode/internal/loop"
)

// SpecsDirName is where spec folders are looked for. One place, not a search:
// walking a repository for anything that resembles a spec finds node_modules.
const SpecsDirName = "specs"

// Found is one spec folder and where it stands.
//
// Where it STANDS, not what it declares. "Pending" cannot be answered by
// counting checkboxes in a tasks.md — a box is marked by whoever felt like
// marking it. It is answered by running the folder's own criteria, which is
// the only definition of done this product recognises.
type Found struct {
	// Path is relative to the workspace, as a person would type it.
	Path string
	// Criteria is how many the folder declares.
	Criteria int
	// Unmet is how many did not pass. Zero with Criteria above zero is a spec
	// that is done.
	Unmet int
	// Unavailable is how many could not be run at all.
	Unavailable int
	// Err is why the folder could not be read, when it could not.
	Err string
}

// Pending says whether there is work here.
//
// A folder that declares nothing is pending: there is no evidence it is
// finished, and treating "nothing to check" as "done" is the defect this whole
// family exists to prevent. A folder whose criteria all pass is not.
func (f Found) Pending() bool {
	if f.Err != "" {
		return false
	}
	if f.Criteria == 0 {
		return true
	}
	return f.Unmet > 0 || f.Unavailable > 0
}

// Discover lists the spec folders under a workspace and runs each one's
// criteria to find out where it stands.
//
// It runs them. There is no cheaper honest answer to "which specs are
// pending": the criteria are the definition of done, and anything else —
// checkbox counts, file timestamps, a marker in the spec — is a second
// statement of something that can move.
//
// Ordered by folder name, which for a dated spec is chronological. The
// sequencing is the operator's, and a list that reshuffles between runs is one
// nobody can act on.
func Discover(ctx context.Context, workspace string, run loop.CriterionRunner, timeout time.Duration) []Found {
	root := filepath.Join(workspace, SpecsDirName)
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	out := make([]Found, 0, len(names))
	for _, name := range names {
		if ctx.Err() != nil {
			return out
		}
		rel := filepath.Join(SpecsDirName, name)
		f := Found{Path: rel}
		dir := filepath.Join(root, name)
		switch {
		case isNotASpec(dir):
			// A folder under specs/ carrying none of a spec's files is a
			// folder. Reporting it would fill the list with assets/ and img/.
			continue
		case !declaresTasks(dir):
			// A spec.md and nothing else is a spec that has not been broken
			// into tasks yet. It is not unreadable — it is the most pending
			// thing in the list, and calling it an error kept eleven of them
			// OUT of the queue in the first workspace this ran against.
			out = append(out, f)
			continue
		}
		spec, err := LoadSpec(dir)
		if err != nil {
			f.Err = err.Error()
			out = append(out, f)
			continue
		}
		f.Criteria = len(spec.Criteria)
		if f.Criteria > 0 && run != nil {
			rep := loop.Check(ctx, spec.DoneSet(), run, timeout)
			for _, state := range rep.States {
				switch state {
				case loop.CriterionUnmet:
					f.Unmet++
				case loop.CriterionUnavailable:
					f.Unavailable++
				}
			}
		}
		out = append(out, f)
	}
	return out
}

// declaresTasks says the folder has something for the parser to read at all.
//
// Separate from isNotASpec because the two answers are different: a folder
// with nothing is not a spec, and a folder with only a spec.md is a spec that
// declares no work yet.
func declaresTasks(dir string) bool {
	for _, name := range []string{tasksFile, doneFileName} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

// isNotASpec says a folder carries none of the files a spec folder has, so it
// is something else that happens to live under specs/.
func isNotASpec(dir string) bool {
	for _, name := range []string{tasksFile, doneFileName, "spec.md"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return false
		}
	}
	return true
}
