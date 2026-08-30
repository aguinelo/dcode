package evals

import (
	"context"
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
	"github.com/aguinelo/dcode/internal/tui"
	"github.com/aguinelo/dcode/internal/vcs"
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
	// World is what a scenario needs to be true of its workspace beyond the
	// files it carries. See World.
	World World
	// Criteria is the scenario's definition of done, when it has one. A
	// scenario that declares criteria gets a real verification cycle instead
	// of an injected reminder describing one.
	Criteria Criteria
}

// World is the part of a scenario's situation that no file in it can express.
//
// Two things reach the prefix from outside the file tree: whether the
// directory is a repository at all, and whether this turn is the one working
// out how the work will be measured. A fixture that could say neither could
// host no contract of the working-defaults or done-qualifier families.
//
// Every field is DECLARED and then checked against the product's own reader,
// never asserted into the prompt. A scenario whose premise turns out not to
// hold fails loudly rather than measuring the opposite of what it says — a
// TMPDIR that happened to sit inside a repository would otherwise invert every
// floor contract at once, silently, and the rates would still look like
// evidence.
type World struct {
	// Repo is what the scenario needs git to say about the directory.
	//
	// "absent" is the only value any contract needs today, and an unknown one
	// is an error rather than a shrug: a typo must not degrade into "do not
	// look", which is the value that renders nothing.
	Repo string `json:"repo"`
	// Qualify is the spec folder this turn is working out the measurement for.
	//
	// Non-empty makes the scenario a qualifying turn: the task comes from the
	// product rather than from task.md, done_propose answers, and the boundary
	// is the one the product forces. Empty is every other scenario.
	Qualify string `json:"qualify"`
}

// RepoAbsent is the only repository state a scenario declares today.
const RepoAbsent = "absent"

// Qualifying reports whether this scenario is a qualifying turn.
func (w World) Qualifying() bool { return strings.TrimSpace(w.Qualify) != "" }

// LoadFixture reads testdata/evals/<id>/ under root.
//
// Every failure here is loud. A scenario whose material is missing must not
// degrade into a scenario that measures nothing — that is how a contract goes
// green for a year without ever having run.
func LoadFixture(root, id string) (Fixture, error) {
	dir := filepath.Join(root, id)

	world, err := loadWorld(dir)
	if err != nil {
		return Fixture{}, fmt.Errorf("fixture %s: %w", id, err)
	}

	task, err := loadTask(dir, world)
	if err != nil {
		return Fixture{}, fmt.Errorf("fixture %s: %w", id, err)
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

	criteria, err := LoadCriteria(dir)
	if err != nil {
		return Fixture{}, fmt.Errorf("fixture %s: %w", id, err)
	}

	return Fixture{
		ID:           id,
		Criteria:     criteria,
		Task:         task,
		World:        world,
		Tools:        tools,
		Instructions: ins,
		Skills:       skills,
		Files:        overlay(shared, own),
	}, nil
}

// loadWorld reads world.json, if the scenario has one.
func loadWorld(dir string) (World, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "world.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return World{}, nil // most scenarios need none
	}
	if err != nil {
		return World{}, err
	}
	var w World
	if err := json.Unmarshal(raw, &w); err != nil {
		return World{}, fmt.Errorf("world.json: %w", err)
	}
	if w.Repo != "" && w.Repo != RepoAbsent {
		return World{}, fmt.Errorf("world.json: %q is not a repository state a scenario can declare; the only one is %q", w.Repo, RepoAbsent)
	}
	return w, nil
}

// loadTask reads what the scenario asks, from the one place that owns it.
//
// A qualifying turn does not get to write its own opening line. The product
// composes that instruction, it is most of what the qualifier contracts
// measure, and a copy of it here would drift from the product exactly the way
// four copies of reminder text already have — each drift reading as a
// plausible number describing something else.
func loadTask(dir string, w World) (string, error) {
	path := filepath.Join(dir, "task.md")
	if w.Qualifying() {
		if _, err := os.Stat(path); err == nil {
			return "", fmt.Errorf("task.md sits beside a world.json that declares a qualifying turn; the instruction comes from the product, and two of them is one that drifts")
		}
		return tui.LoopTask(tui.LoopArgs{Qualify: true, Spec: w.Qualify}), nil
	}
	task, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(string(task)) == "" {
		return "", fmt.Errorf("task.md is empty")
	}
	return strings.TrimSpace(string(task)), nil
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
	return f.prompt(family, nil, nil)
}

func (f Fixture) prompt(family string, repo *behavior.Repo, ws *behavior.Workspace) (string, error) {
	names := f.ToolNames()
	return behavior.Build(behavior.Prompt{
		Doctrine:     behavior.DefaultDoctrine(names),
		Tools:        names,
		Instructions: f.Instructions,
		SkillIndex:   behavior.Index(f.Skills),
		Repo:         repo,
		Workspace:    ws,
	}, behavior.FormulationFor(family))
}

// survey takes the snapshots the scenario declared, with the product's own
// readers, and refuses if what came back is not what the scenario says.
//
// The refusal is the point. Both of these are read from a directory the
// harness created moments earlier, and both have an answer that renders
// nothing at all — a repository that exists, a project that declares no gate.
// A scenario built on "there is no repository here" running in a directory
// that turned out to be inside one would measure a model being silent about
// something that was never in its prompt, and report the silence as the
// contract being honoured.
func (f Fixture) survey(ctx context.Context, dir string) (*behavior.Repo, *behavior.Workspace, error) {
	var repo *behavior.Repo
	if f.World.Repo == RepoAbsent {
		repo = vcs.Read(ctx, dir)
		switch {
		case repo == nil:
			return nil, nil, fmt.Errorf("fixture %s declares no repository and git could not be asked, so nothing in the prefix says either way", f.ID)
		case !repo.Absent:
			return nil, nil, fmt.Errorf("fixture %s declares no repository and %s is one; the scenario would measure the opposite of what it says", f.ID, dir)
		}
	}

	// No gate inventory here yet. The working-defaults `.p` declares no
	// contract that needs one — F-3 is still to do — and a field nothing reads
	// is a control that is not there.
	return repo, nil, nil
}

// PromptIn is the prompt for a scenario whose workspace exists on disk.
//
// A scenario carrying `.dcode/memory.md` in its files gets it read and rendered
// by the PRODUCT's reader, never by a block copied into the fixture. A fixture
// that copies product text is a fixture that drifts from it, and this suite has
// found that four times — the reminder whose truncated copy dropped the clause
// the judge measured is the one that cost the most.
func (f Fixture) PromptIn(ctx context.Context, family, dir string) (string, error) {
	repo, ws, err := f.survey(ctx, dir)
	if err != nil {
		return "", err
	}
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
	return f.prompt(family, repo, ws)
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
func (f Fixture) Messages(ctx context.Context, family, dir string, history []ce.Message) ([]ce.Message, error) {
	prompt, err := f.PromptIn(ctx, family, dir)
	if err != nil {
		return nil, err
	}
	return ce.Assemble(ce.Session{
		Instructions: prompt,
		Tools:        f.Tools,
		History:      history,
	})
}
