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
	"github.com/aguinelo/dcode/internal/memory"
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
	// Skills are whole skills, index and body.
	//
	// The index alone was not enough: the product puts one line per skill in
	// the prefix and appends the *body* when the task matches (RN-7), and a
	// fixture that shipped only the index measured whether the model could
	// guess a procedure it had never been shown.
	Skills []behavior.Skill
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

// loadSkills reads skills/*.md, if the directory is there.
//
// Whole skills, parsed by the product's own ParseSkill, because the body is
// what the contract is about. The index is derived from them rather than
// written beside them: two lists of the same skills is one list that drifts.
func loadSkills(dir string) ([]behavior.Skill, error) {
	entries, err := os.ReadDir(filepath.Join(dir, "skills"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []behavior.Skill
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, "skills", e.Name())
		text, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		skill, err := behavior.ParseSkill(string(text), e.Name())
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(skill.WhenToUse) == "" {
			return nil, fmt.Errorf("skills/%s has no when_to_use, so nothing tells the model when to load it", e.Name())
		}
		out = append(out, skill)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
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
		SkillIndex:   behavior.Index(f.Skills),
	}, behavior.FormulationFor(family))
}

// PromptIn is the prompt for a scenario whose workspace exists on disk.
//
// A scenario carrying `.dcode/memory.md` in its files gets it read and rendered
// by the PRODUCT's reader, never by a block copied into the fixture. A fixture
// that copies product text is a fixture that drifts from it, and this suite has
// found that four times — the reminder whose truncated copy dropped the clause
// the judge measured is the one that cost the most.
func (f Fixture) PromptIn(family, dir string) (string, error) {
	if learned, err := memory.Read(dir); err == nil {
		if block := memory.Render(learned, memory.DefaultMax, nil); block != "" {
			f.Instructions = append(append([]behavior.Instruction(nil), f.Instructions...),
				behavior.Instruction{
					Source: behavior.SourceLearned,
					Scope:  memory.FileName,
					Text:   block,
				})
		}
	}
	return f.Prompt(family)
}

// Opening is the history a scenario starts from: the task, and the body of any
// skill it triggers.
//
// The loop appends a matched skill body as a reminder after the user message
// (turn.go, RN-7), and the harness sent only the task. So the scenario about
// using a skill body measured whether the model could invent a procedure it had
// never been shown — and the procedure exists in the fixture precisely because
// nobody would guess it.
func (f Fixture) Opening() []ce.Message {
	out := []ce.Message{{Role: ce.RoleUser, Text: f.Task}}
	for _, s := range behavior.Match(f.Task, f.Skills) {
		out = append(out, ce.Message{
			Role: ce.RoleUser, Text: behavior.RenderSkill(s), Reminder: true,
		})
	}
	return out
}

// Messages assembles one call the way the product assembles it.
//
// Through ce.Assemble rather than by hand, which is the whole correction: the
// hand-built list was `[]Message{{Role: RoleUser, Text: task}}`, and Assemble
// refuses to build anything at all without a system prompt. Bypassing it
// bypassed the refusal that would have caught this on the first run.
// dir is the scenario's workspace, because the prompt now depends on it: a
// scenario carrying a memory has that memory in its prefix, read by the
// product's own reader.
func (f Fixture) Messages(family, dir string, history []ce.Message) ([]ce.Message, error) {
	prompt, err := f.PromptIn(family, dir)
	if err != nil {
		return nil, err
	}
	return ce.Assemble(ce.Session{
		Instructions: prompt,
		Tools:        f.Tools,
		History:      history,
	})
}
