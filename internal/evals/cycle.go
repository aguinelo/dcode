package evals

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/aguinelo/dcode/internal/behavior"
	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/loop"
)

// Criteria is a scenario's definition of done, declared in criteria.json.
//
// It exists because the harness could not measure the verification cycle at
// all. Every scenario about what a turn does when a check fails INJECTED the
// reminder the cycle would have produced, so `checkDone`, `Moved`, the
// rollback and the reminder assembly never ran — and a family built entirely
// out of those was delivered with nothing measured about it.
//
// A criterion here is a predicate over the workspace, never a shell command.
// The harness does not execute what a model wrote and does not execute what a
// fixture wrote either: a scenario that shelled out would depend on what
// happened to be installed that afternoon, and a measurement nobody can
// reproduce is not a measurement.
type Criteria struct {
	Criteria []Criterion `json:"criteria"`
}

// Criterion is one check, expressed as something the harness can decide by
// reading the workspace.
type Criterion struct {
	Name string `json:"name"`
	// File is the path it is about, relative to the workspace.
	File string `json:"file"`
	// Contains passes when the file holds this text; Absent passes when it
	// does not. Exactly one of the two, so a criterion cannot be written that
	// says nothing.
	Contains string `json:"contains,omitempty"`
	Absent   string `json:"absent,omitempty"`
}

// LoadCriteria reads criteria.json, if the scenario has one.
func LoadCriteria(dir string) (Criteria, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "criteria.json"))
	if os.IsNotExist(err) {
		return Criteria{}, nil
	}
	if err != nil {
		return Criteria{}, err
	}
	var c Criteria
	if err := json.Unmarshal(raw, &c); err != nil {
		return Criteria{}, err
	}
	for _, one := range c.Criteria {
		switch {
		case strings.TrimSpace(one.Name) == "" || strings.TrimSpace(one.File) == "":
			return Criteria{}, errCriterion(one.Name, "needs a name and a file")
		case (one.Contains == "") == (one.Absent == ""):
			return Criteria{}, errCriterion(one.Name, "needs exactly one of contains or absent; a criterion that says both or neither decides nothing")
		}
	}
	return c, nil
}

func errCriterion(name, why string) error {
	return &criterionError{name: name, why: why}
}

type criterionError struct{ name, why string }

func (e *criterionError) Error() string {
	return "criteria.json: criterion " + e.name + " " + e.why
}

// DoneSet is the criteria as the product's own loop understands them.
//
// The command carries the criterion's name because loop.Check dispatches on
// it: the runner below is a lookup, not a shell.
func (c Criteria) DoneSet() loop.DoneSet {
	var set loop.DoneSet
	for _, one := range c.Criteria {
		set.Criteria = append(set.Criteria, loop.Criterion{Name: one.Name, Command: one.Name})
	}
	return set
}

// Runner decides each criterion by reading the workspace.
//
// Exit 0 for met and 1 for unmet, which is what loop.Check compares against.
// A file it cannot read is unmet rather than unavailable: for these scenarios
// an absent file is the ordinary way a criterion fails, and calling it
// uncheckable would hide the failure the scenario is about.
func (c Criteria) Runner(dir string) loop.CriterionRunner {
	by := map[string]Criterion{}
	for _, one := range c.Criteria {
		by[one.Name] = one
	}
	return func(_ context.Context, name string) (int, string, error) {
		one, ok := by[name]
		if !ok {
			return 1, "no such criterion: " + name, nil
		}
		body, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(one.File)))
		if err != nil {
			return 1, one.File + ": not there", nil
		}
		if one.Contains != "" {
			if strings.Contains(string(body), one.Contains) {
				return 0, "", nil
			}
			return 1, one.File + " does not contain " + one.Contains, nil
		}
		if !strings.Contains(string(body), one.Absent) {
			return 0, "", nil
		}
		return 1, one.File + " still contains " + one.Absent, nil
	}
}

// Cycle is one verification round: run the criteria, classify what the last
// round did to them, roll back what regressed, and render what the product
// would say next.
//
// It calls the product's own loop.Check, loop.Moved, State.UndoCycle and
// behavior.Emit. Nothing here decides anything the product decides — the
// harness's job is to put the pieces in the order the loop puts them, and a
// second implementation of any of them would measure the second one.
type Cycle struct {
	set    loop.DoneSet
	run    loop.CriterionRunner
	w      *Workspace
	unmet  []string
	Undone int
}

// NewCycle prepares a verification cycle over a scenario's criteria, or nil
// when the scenario declares none.
func NewCycle(c Criteria, w *Workspace) *Cycle {
	if len(c.Criteria) == 0 {
		return nil
	}
	return &Cycle{set: c.DoneSet(), run: c.Runner(w.Dir), w: w}
}

// Begin marks the point a regression would be rolled back to.
func (c *Cycle) Begin() {
	if c == nil {
		return
	}
	c.w.BeginCycle()
}

// After runs the criteria and returns what the product would append.
//
// Nil when everything passes: the turn is done and the loop would stop asking.
func (c *Cycle) After() []ce.Message {
	if c == nil {
		return nil
	}
	rep := loop.Check(context.Background(), c.set, c.run, 0)
	now := rep.Unmet()
	if len(now) == 0 {
		return nil
	}

	st := behavior.SessionState{
		UnmetCriteria:    now,
		CriterionOutputs: rep.OutputTexts(c.set),
	}
	if c.unmet != nil && loop.Moved(c.unmet, now) == loop.MovedBackward {
		st.Regressed = regressed(c.unmet, now)
		restored, kept, err := c.w.UndoCycle()
		if err == nil {
			st.CycleUndone, st.CycleKept = restored, kept
			c.Undone++
		}
	}
	c.unmet = now

	var out []ce.Message
	for _, r := range behavior.Emit(st) {
		out = append(out, ce.Message{
			Role: ce.RoleUser, Text: behavior.Render(r), Reminder: true,
		})
	}
	return out
}

func regressed(before, after []string) []string {
	had := map[string]struct{}{}
	for _, n := range before {
		had[n] = struct{}{}
	}
	var out []string
	for _, n := range after {
		if _, ok := had[n]; !ok {
			out = append(out, n)
		}
	}
	return out
}

// EveryCriterionMet passes when the scenario's own criteria all pass at the
// end of the run.
//
// The verdict is the workspace, not the transcript. Every other judge in this
// suite reads what the model called and said; this one re-runs the ruler. It is
// the only judge here that could be wrong about the model and right about the
// work, and that is the point — a correction is good when the check goes green,
// not when the sentence about it reads well.
func EveryCriterionMet() Judge {
	return func(t Transcript) bool { return t.CriteriaMet }
}

// Met reports whether every criterion passes now.
//
// Asked at the end rather than remembered from the last cycle: a run that hit
// the round ceiling mid-edit has a workspace the last cycle never saw.
//
// False for a scenario with no criteria, so a judge reading it cannot pass by
// having nothing to check — the mistake that once made does-not-delegate-trivial
// pass by having nothing to delegate to.
func (c *Cycle) Met() bool {
	if c == nil {
		return false
	}
	return len(loop.Check(context.Background(), c.set, c.run, 0).Unmet()) == 0
}
