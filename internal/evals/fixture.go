package evals

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	var tools []ce.ToolDef
	if err := json.Unmarshal(raw, &tools); err != nil {
		return Fixture{}, fmt.Errorf("fixture %s: tools.json: %w", id, err)
	}
	if len(tools) == 0 {
		return Fixture{}, fmt.Errorf("fixture %s: tools.json declares no tools", id)
	}

	return Fixture{ID: id, Task: strings.TrimSpace(string(task)), Tools: tools}, nil
}

// Declares reports whether name is in the fixture's tool set.
//
// This is what a phantom-tool judge asks, and it lives here rather than in the
// scenario so the question is asked the same way everywhere.
func (f Fixture) Declares(name string) bool {
	for _, t := range f.Tools {
		if t.Name == name {
			return true
		}
	}
	return false
}
