package evals

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aguinelo/dcode/internal/behavior"
	ce "github.com/aguinelo/dcode/internal/contextengine"
)

// FixtureRoot is where a scenario's material lives, relative to the package
// that measures it. The path is the one written in the `.p.spec.md` fixture
// column, and it is spelled the same in both places on purpose: a fixture the
// spec names and the code cannot find is a contract nobody is measuring.
const FixtureRoot = "testdata/evals"

// Fixture is the material of one scenario.
//
// Task and Tools are data rather than Go literals so that changing what a
// scenario asks does not require rebuilding, and so the diff of a threshold
// moving shows what changed about the question.
type Fixture struct {
	ID    string
	Task  string
	Tools []ce.ToolDef
	// Instructions is the project material this scenario needs in the prompt,
	// and Skills is the index it needs to see.
	//
	// Both were missing, and their absence is what made six contracts score a
	// flat zero over twenty runs each. `follows-project-instruction` asks
	// whether the model follows a project convention without being reminded —
	// and no convention was ever sent. It could not have passed.
	Instructions []behavior.Instruction
	Skills       []behavior.SkillIndexEntry
	// Files is the workspace this scenario runs against: the shared miniature
	// repository, with the fixture's own files/ laid over it.
	//
	// It replaced a canned answer per tool, and that answer was the string
	// "ok". A model that globbed and grepped and got "ok" every time concluded
	// the workspace was empty — correctly, from what it had been told — and
	// refused to invent code. That is good behaviour scored as zero, and it
	// was poisoning every multi-round scenario at once.
	Files map[string]string
}

// LoadFixture reads testdata/evals/<id>/ under root.
//
// Every failure here is loud. A scenario whose material is missing must not
// degrade into a scenario that measures nothing — that is how a contract goes
// green for a year without ever having run.
func LoadFixture(root, id string) (Fixture, error) {
	dir := filepath.Join(root, id)

	task, err := os.ReadFile(filepath.Join(dir, "task.md"))
	if err != nil {
		return Fixture{}, fmt.Errorf("fixture %s: %w", id, err)
	}
	if strings.TrimSpace(string(task)) == "" {
		return Fixture{}, fmt.Errorf("fixture %s: task.md is empty", id)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "tools.json"))
	if err != nil {
		return Fixture{}, fmt.Errorf("fixture %s: %w", id, err)
	}
	tools, err := decodeTools(raw)
	if err != nil {
		return Fixture{}, fmt.Errorf("fixture %s: tools.json: %w", id, err)
	}
	if len(tools) == 0 {
		return Fixture{}, fmt.Errorf("fixture %s: tools.json declares no tools", id)
	}

	ins, err := loadInstructions(dir)
	if err != nil {
		return Fixture{}, fmt.Errorf("fixture %s: %w", id, err)
	}
	skills, err := loadSkills(dir)
	if err != nil {
		return Fixture{}, fmt.Errorf("fixture %s: %w", id, err)
	}

	shared, err := loadFiles(filepath.Join(filepath.Dir(root), filepath.Base(WorkspaceRoot)))
	if err != nil {
		return Fixture{}, fmt.Errorf("fixture %s: shared workspace: %w", id, err)
	}
	own, err := loadFiles(filepath.Join(dir, "files"))
	if err != nil {
		return Fixture{}, fmt.Errorf("fixture %s: %w", id, err)
	}

	return Fixture{
		ID:           id,
		Task:         strings.TrimSpace(string(task)),
		Tools:        tools,
		Instructions: ins,
		Skills:       skills,
		Files:        overlay(shared, own),
	}, nil
}

// Spelled out rather than accepted freely: a file named `porject.md` would
// otherwise load as an instruction from a source with no authority ranking,
// sort first, and quietly measure the opposite of what the scenario says.
var instructionSources = map[string]behavior.InstructionSource{
	"locked":    behavior.SourceLocked,
	"directory": behavior.SourceDirectory,
	"project":   behavior.SourceProject,
	"user":      behavior.SourceUser,
}

// loadInstructions reads instructions/<source>.md, if the directory is there.
//
// The source is the file name because the precedence between sources is the
// thing several scenarios exist to measure — `directory-over-project` is
// nothing but that ordering — and a scenario cannot state it if the material
// cannot carry it.
func loadInstructions(dir string) ([]behavior.Instruction, error) {
	entries, err := os.ReadDir(filepath.Join(dir, "instructions"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil // most scenarios need none
	}
	if err != nil {
		return nil, err
	}

	var out []behavior.Instruction
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		source, ok := instructionSources[name]
		if !ok {
			return nil, fmt.Errorf("instructions/%s: %q is not an instruction source", e.Name(), name)
		}
		text, err := os.ReadFile(filepath.Join(dir, "instructions", e.Name()))
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(string(text)) == "" {
			return nil, fmt.Errorf("instructions/%s is empty", e.Name())
		}
		out = append(out, behavior.Instruction{
			Source: source,
			Locked: source == behavior.SourceLocked,
			Text:   strings.TrimSpace(string(text)),
		})
	}
	// Stable order regardless of what the filesystem hands back, so the prompt
	// a scenario builds is the same prompt on every machine.
	sort.Slice(out, func(i, j int) bool { return out[i].Source < out[j].Source })
	return out, nil
}

// loadSkills reads skills.json, if it is there.
//
// The index only — one line per skill, never a body. That is what the product
// puts in the prefix (RN-7), and a fixture that shipped bodies would measure a
// prompt the product never builds.
func loadSkills(dir string) ([]behavior.SkillIndexEntry, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "skills.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var skills []behavior.SkillIndexEntry
	if err := json.Unmarshal(raw, &skills); err != nil {
		return nil, fmt.Errorf("skills.json: %w", err)
	}
	for _, s := range skills {
		if strings.TrimSpace(s.Name) == "" || strings.TrimSpace(s.WhenToUse) == "" {
			return nil, fmt.Errorf("skills.json: an entry is missing its name or when-to-use, "+
				"so nothing in the index would tell the model when to load it: %+v", s)
		}
	}
	return skills, nil
}

// decodeTools reads a scenario's tool set.
//
// A bare string names a product tool and takes the product's own definition. An
// object describes a tool the product does not have, which a few scenarios need
// — `record_release` for the schema contract, `delete_file` for the phantom one.
//
// Hand-written copies of product schemas are refused, because they drifted: a
// fixture declared `plan` with a free-string `state` while the product declares
// `status` with an enum, and the model spent every round of those scenarios
// sending a shape the fixture had described wrongly.
func decodeTools(raw []byte) ([]ce.ToolDef, error) {
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}
	registry := ProductRegistry()
	defs := map[string]ce.ToolDef{}
	for _, d := range registry.Defs() {
		defs[d.Name] = d
	}

	out := make([]ce.ToolDef, 0, len(entries))
	for _, entry := range entries {
		var name string
		if err := json.Unmarshal(entry, &name); err == nil {
			def, ok := defs[name]
			if !ok {
				return nil, fmt.Errorf("%q is not a tool the product has; describe it inline if the scenario needs one that is not real", name)
			}
			out = append(out, def)
			continue
		}
		var def ce.ToolDef
		if err := json.Unmarshal(entry, &def); err != nil {
			return nil, err
		}
		if _, real := defs[def.Name]; real {
			return nil, fmt.Errorf("%q is a product tool and is described by hand here; name it as a string so it takes the product's own definition", def.Name)
		}
		out = append(out, def)
	}
	return out, nil
}

// ToolNames is the fixture's tool set, in the order it was declared.

// ToolNames is the fixture's tool set, in the order it was declared.
func (f Fixture) ToolNames() []string {
	names := make([]string, 0, len(f.Tools))
	for _, t := range f.Tools {
		names = append(names, t.Name)
	}
	return names
}

// Prompt builds the system prompt this scenario is measured against.
//
// The doctrine is the shipped one, not a stand-in. Almost every contract here
// is about behaviour the prompt produces — plan before a wide change, prefer
// the dedicated tool over the shell, re-read a file the reminder says moved —
// so a measurement that omits the prompt measures a bare model and reports the
// number as though it were about dcode.
func (f Fixture) Prompt(family string) (string, error) {
	names := f.ToolNames()
	return behavior.Build(behavior.Prompt{
		Doctrine:     behavior.DefaultDoctrine(names),
		Tools:        names,
		Instructions: f.Instructions,
		SkillIndex:   f.Skills,
	}, behavior.FormulationFor(family))
}

// Messages assembles one call the way the product assembles it.
//
// Through ce.Assemble rather than by hand, which is the whole correction: the
// hand-built list was `[]Message{{Role: RoleUser, Text: task}}`, and Assemble
// refuses to build anything at all without a system prompt. Bypassing it
// bypassed the refusal that would have caught this on the first run.
func (f Fixture) Messages(family string, history []ce.Message) ([]ce.Message, error) {
	prompt, err := f.Prompt(family)
	if err != nil {
		return nil, err
	}
	return ce.Assemble(ce.Session{
		Instructions: prompt,
		Tools:        f.Tools,
		History:      history,
	})
}
