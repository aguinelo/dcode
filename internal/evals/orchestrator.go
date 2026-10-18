package evals

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/aguinelo/dcode/internal/behavior"
	"github.com/aguinelo/dcode/internal/orchestrate"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/tools"
)

// The orchestrator's half of the harness: a project that accepts what it is
// given, the role a scenario declares, and the judge of a dispatched batch.
//
// Spec: docs/specs/architecture/behavior-definition/202608080016-behavior-definition.p.spec.md §2.1, §7.

// standInProject is the project an orchestrator's tools reach in a scenario.
//
// It accepts every batch and runs no child. The contract is what the model
// dispatches, which has already happened by the time a child would start; and
// a conductor that refused — the product's own answer with no project behind
// the session — would talk the model out of dispatching, the lesson the
// harness learned from `explore`.
type standInProject struct {
	mu   sync.Mutex
	kids []protocol.Child
}

func (p *standInProject) Delegate(items []tools.DelegateItem) ([]protocol.Child, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	made := make([]protocol.Child, 0, len(items))
	for _, it := range items {
		id := fmt.Sprintf("c%d", len(p.kids)+1)
		c := protocol.Child{ID: id, Title: it.Title, State: "working", Branch: "dcode/" + id, Owns: it.Owns}
		p.kids = append(p.kids, c)
		made = append(made, c)
	}
	return made, nil
}

func (p *standInProject) Children() ([]protocol.Child, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]protocol.Child(nil), p.kids...), nil
}

func (p *standInProject) SteerChild(string, string, string) error { return nil }
func (p *standInProject) CloseChild(string, string) error         { return nil }
func (p *standInProject) Ask(tools.AskInput) (string, error)      { return "q1", nil }
func (p *standInProject) Summarize(string, string) error          { return nil }
func (p *standInProject) ProposeAnswers(tools.ProposeInput) (string, error) {
	return "c1", nil
}

// validRole refuses a role the prompt cannot be built for, the way an unknown
// mode is refused: a typo must not degrade into a scenario measured without
// the role it names.
func validRole(role string) error {
	switch role {
	case "", behavior.RoleOrchestrator, behavior.RoleChild, behavior.RoleReviewer:
		return nil
	}
	return fmt.Errorf("world.json: %q is not a role; use %q, %q or %q",
		role, behavior.RoleOrchestrator, behavior.RoleChild, behavior.RoleReviewer)
}

// DisjointBatch reports a `delegate` call of at least min items whose `owns`
// are all non-empty and cross no other item's, by the product's own reading of
// a crossing (orchestrate.OwnsOverlap).
//
// One call, not the union of several: a batch is what the product validates
// whole, and three calls of one item each would split the themes and still
// say nothing about whether the model saw them as one piece of work.
func DisjointBatch(min int) Judge {
	return func(t Transcript) bool {
		for _, c := range t.Calls {
			if c.Name != "delegate" {
				continue
			}
			var in struct {
				Items []tools.DelegateItem `json:"items"`
			}
			if err := json.Unmarshal(c.Input, &in); err != nil || len(in.Items) < min {
				continue
			}
			if disjoint(in.Items) {
				return true
			}
		}
		return false
	}
}

func disjoint(items []tools.DelegateItem) bool {
	var seen []string
	for _, it := range items {
		if len(it.Owns) == 0 {
			return false
		}
		for _, o := range it.Owns {
			o = strings.TrimSpace(o)
			for _, s := range seen {
				if orchestrate.OwnsOverlap(o, s) {
					return false
				}
			}
		}
		seen = append(seen, it.Owns...)
	}
	return true
}
