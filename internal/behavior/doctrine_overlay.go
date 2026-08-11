package behavior

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DoctrineDirName is the directory searched under the user's config root.
const DoctrineDirName = "doctrine"

// DoctrineOverlay is what the user's configuration may change in the base
// layer (RN-11). An empty field leaves the shipped text intact.
//
// Safety is NOT here, and that absence is the guarantee (RN-12): there is no
// path to close because there is no path. A lock by convention breaks at the
// first refactor; a lock by type does not compile.
//
// ToolsMore is spelled differently from ToolPolicy for the same reason. No
// accidental assignment can swap one for the other.
type DoctrineOverlay struct {
	Identity  string // replaces
	Style     string // replaces
	ToolsMore string // APPENDS to ToolPolicy; never replaces
}

// Origin says where a section of the assembled prompt came from.
//
// It exists for the audit: an invisible replacement would be worse than the
// immutability it replaces, because the only way a user has today of knowing
// what reached the model is to read it.
type Origin string

const (
	OriginBuiltin  Origin = "builtin"
	OriginReplaced Origin = "replaced"
	OriginAppended Origin = "appended"
)

// SectionOrigins is the provenance of all four sections.
type SectionOrigins struct {
	Identity   Origin
	ToolPolicy Origin
	Safety     Origin
	Style      Origin
}

// Notice is something the loader refused to do silently.
type Notice struct {
	Path   string
	Reason string
}

func (n Notice) String() string { return n.Path + ": " + n.Reason }

// overlayFiles maps a filename to what it does. The map is the whole rule:
// which file exists decides which section changes, and HOW it changes is
// fixed per section and not configurable.
//
// There is no entry that replaces ToolPolicy, and no filename that reaches
// Safety.
var overlayFiles = map[string]string{
	"identity.md": "Identity",
	"style.md":    "Style",
	"tools.md":    "ToolPolicy",
}

// Apply returns the doctrine with the overlay applied. Pure.
//
// Safety is not read here and could not be written if it were: the overlay has
// no such field.
func (d Doctrine) Apply(o DoctrineOverlay) Doctrine {
	if o.Identity != "" {
		d.Identity = o.Identity
	}
	if o.Style != "" {
		d.Style = o.Style
	}
	if o.ToolsMore != "" {
		// Appended, never substituted. The shipped text stays a prefix, so the
		// real tool list remains non-negotiable while the habits around it do
		// not — which is the whole distinction RN-12 draws.
		d.ToolPolicy = d.ToolPolicy + "\n\n" + o.ToolsMore
	}
	return d
}

// Origins reports where each section will have come from once applied.
func (o DoctrineOverlay) Origins() SectionOrigins {
	s := SectionOrigins{
		Identity:   OriginBuiltin,
		ToolPolicy: OriginBuiltin,
		// Always builtin, and not because of a branch below: nothing in this
		// type can reach Safety.
		Safety: OriginBuiltin,
		Style:  OriginBuiltin,
	}
	if o.Identity != "" {
		s.Identity = OriginReplaced
	}
	if o.Style != "" {
		s.Style = OriginReplaced
	}
	if o.ToolsMore != "" {
		s.ToolPolicy = OriginAppended
	}
	return s
}

// LoadDoctrineOverlay reads the overlay from one directory.
//
// The parameter is ONE directory, not a list. The contrast with
// LoadSkills(dirs []string, ...) is deliberate: a skill comes from two roots,
// a doctrine overlay comes from one (RN-11). The singular type says that
// better than a comment could, and the workspace root never becomes an
// argument in the first place.
func LoadDoctrineOverlay(dir string, maxBytes int) (DoctrineOverlay, []Notice, error) {
	var o DoctrineOverlay
	var notices []Notice

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		// No overlay directory is the normal case, not a failure. Most
		// installations never write one.
		return o, nil, nil
	}
	if err != nil {
		return DoctrineOverlay{}, nil, err
	}

	// Sorted so that the notices come out in the same order on every run and
	// on every machine. A warning list that reshuffles between runs is one
	// nobody can diff.
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		path := filepath.Join(dir, name)

		section, known := overlayFiles[name]
		if !known {
			notices = append(notices, Notice{Path: path, Reason: unknownReason(name)})
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return DoctrineOverlay{}, nil, err
		}
		text := string(data)
		if maxBytes > 0 && len(text) > maxBytes {
			text = text[:maxBytes]
			notices = append(notices, Notice{
				Path:   path,
				Reason: fmt.Sprintf("cut to the %d byte limit; the rest is not in force", maxBytes),
			})
		}
		text = strings.TrimSpace(text)

		switch section {
		case "Identity":
			o.Identity = text
		case "Style":
			o.Style = text
		case "ToolPolicy":
			o.ToolsMore = text
		}
	}

	return o, notices, nil
}

// unknownReason explains why a file did nothing.
//
// safety.md gets its own sentence. It is not a typo — it is someone trying to
// move the one line that does not move, and an attempt nobody can see is an
// attempt nobody investigates. Same treatment RN-10 already requires.
func unknownReason(name string) string {
	if name == "safety.md" {
		return "ignored: Safety is not overridable, by any file, from any root. " +
			"There is no configuration path to it — the overlay type has no such field."
	}
	known := make([]string, 0, len(overlayFiles))
	for f := range overlayFiles {
		known = append(known, f)
	}
	sort.Strings(known)
	return "ignored: not a doctrine overlay file. Recognised names are " + strings.Join(known, ", ")
}
