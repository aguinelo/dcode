package loopcommand

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/loop"
)

// Source declares where a DoneSet comes from.
//
// SourceAuto preserves the historical behaviour: done.toml if it exists,
// else the legacy verifyCommand. SourceDoneFile forces done.toml. SourceLoopSpec
// forces a LoopSpec — failure to load is an error, not a fallback to legacy.
type Source int

const (
	SourceAuto Source = iota
	SourceDoneFile
	SourceLoopSpec
)

// Load resolves the source and returns the DoneSet.
//
// Same inputs → same DoneSet. Reading the filesystem is the only I/O.
//
// The verifyCommand is consulted by SourceAuto only when neither done.toml
// nor a LoopSpec is available. Legacy behaviour preserved.
func Load(workspace, specPath string, src Source, verifyCommand string) (loop.DoneSet, error) {
	switch src {
	case SourceDoneFile:
		return loadDoneTOML(workspace+"/.dcode/done.toml", verifyCommand)

	case SourceLoopSpec:
		if specPath == "" {
			return loop.DoneSet{}, fmt.Errorf("loopcommand: source=loop_spec requires a spec path")
		}
		spec, err := LoadSpec(specPath)
		if err != nil {
			return loop.DoneSet{}, err
		}
		return spec.DoneSet(), nil

	case SourceAuto:
		// done.toml if present.
		if _, err := os.Stat(workspace + "/.dcode/done.toml"); err == nil {
			return loadDoneTOML(workspace+"/.dcode/done.toml", verifyCommand)
		}
		// LoopSpec if a path was set.
		if specPath != "" {
			spec, err := LoadSpec(specPath)
			if err == nil {
				return spec.DoneSet(), nil
			}
			// A spec that is not THERE falls through to the legacy command;
			// a spec that is there and cannot be read is an error.
			//
			// This has to unwrap: LoadSpec wraps with %w, and os.IsNotExist
			// does not follow a wrapped chain. With it, every missing spec
			// path came back as a hard error and this fall-through was
			// unreachable code under a comment saying it was reachable.
			if !errors.Is(err, fs.ErrNotExist) {
				return loop.DoneSet{}, err
			}
		}
		return doneFromVerify(verifyCommand), nil
	}
	return loop.DoneSet{}, fmt.Errorf("loopcommand: unhandled source %d", src)
}

// DoneSet converts a LoopSpec into the loop.DoneSet the engine consumes.
// Criteria == nil becomes DoneSet with no criteria, which the agent-loop
// reports as "no definition of done" — not an error.
func (s LoopSpec) DoneSet() loop.DoneSet {
	return loop.DoneSet{
		Criteria:  s.Criteria,
		Protected: s.Protected,
	}
}

// loadDoneTOML reads the strict TOML subset the rest of the configuration
// uses and turns it into a DoneSet. Same parser as internal/app/done.go
// uses; duplication is small but real — the alternative is a dependency
// from loopcommand to app, which inverts the layering this package exists
// to avoid.
func loadDoneTOML(path, verifyCommand string) (loop.DoneSet, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return doneFromVerify(verifyCommand), nil
	}
	if err != nil {
		return loop.DoneSet{}, err
	}

	set, err := parseDoneTOML(string(raw), path)
	if err != nil {
		return loop.DoneSet{}, err
	}
	if len(set.Criteria) == 0 {
		return doneFromVerify(verifyCommand), nil
	}
	return set, nil
}

func doneFromVerify(command string) loop.DoneSet {
	if strings.TrimSpace(command) == "" {
		return loop.DoneSet{}
	}
	return loop.DoneSet{Criteria: []loop.Criterion{{Name: "verify", Command: command}}}
}

func splitList(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func atoi(v string) int {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0
	}
	return n
}

// doneFileName is what a spec folder calls its own definition of done. The
// same name the workspace uses, because it is the same file in the same
// format: two names for one thing is how a person learns one of them and
// misses the other.
const doneFileName = "done.toml"

// doneBesideSpec reads a done.toml inside the spec folder.
//
// found is false when there is no such file, which is the ordinary case and
// not an error: the folder simply declares its criteria in tasks.md, or not at
// all. A file that IS there and cannot be read is an error, because falling
// back to tasks.md would measure the turn against something other than what
// the folder's own file says.
func doneBesideSpec(path string, protect []string) (loop.DoneSet, bool, error) {
	full := filepath.Join(path, doneFileName)
	raw, err := os.ReadFile(full)
	if errors.Is(err, fs.ErrNotExist) {
		return loop.DoneSet{}, false, nil
	}
	if err != nil {
		return loop.DoneSet{}, true, fmt.Errorf("loopcommand: read %s: %w", full, err)
	}
	set, err := parseDoneTOML(string(raw), full)
	if err != nil {
		return loop.DoneSet{}, true, err
	}
	if len(set.Criteria) == 0 {
		// An empty definition of done is "nothing to verify", which the loop
		// reports as done. A file that exists and declares nothing is the same
		// unreadable-becomes-green defect the parser refuses in a tasks.md.
		return loop.DoneSet{}, true, fmt.Errorf(
			"loopcommand: %s declares no criterion; a definition of done with nothing in it reports done", full)
	}
	set.Protected = union(set.Protected, protect)
	return set, true, nil
}

// parseDoneTOML turns the strict TOML subset into a DoneSet. One parser for
// the workspace's file and the spec folder's: they are the same format, and a
// second implementation is a second set of edge cases.
func parseDoneTOML(body, path string) (loop.DoneSet, error) {
	sections, err := config.ParseSections(body, path)
	if err != nil {
		return loop.DoneSet{}, err
	}
	var set loop.DoneSet
	for _, name := range sections.Order {
		values := sections.Values[name]
		if name == "" {
			if p := values["protected"]; p != "" {
				set.Protected = splitList(p)
			}
			continue
		}
		c := loop.Criterion{Name: name, Command: values["command"]}
		if v := values["exit_code"]; v != "" {
			c.ExitCode = atoi(v)
		}
		set.Criteria = append(set.Criteria, c)
	}
	return set, nil
}
