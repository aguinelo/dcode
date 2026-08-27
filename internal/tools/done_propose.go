package tools

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/aguinelo/dcode/internal/policy"
)

// DoneProposeFile is what a proposal is written to, inside the spec folder.
const DoneProposeFile = "done.toml"

// DonePropose is how the model hands over a definition of done it derived.
//
// A tool, and not prose, for the reason everything else in this harness is a
// tool: the call IS how the thing gets done, and there is no other way to do
// it. A model that describes criteria in a sentence has described them.
//
// It is available only in a qualifying turn. A tool that can redefine done,
// within reach of a working turn, is the shortest way out of a loop — the
// agent rewrites the ruler instead of meeting it.
type DonePropose struct {
	// Spec is the folder the proposal is for, absolute. Set by whoever built
	// the qualifying turn: the model does not get to choose which folder its
	// proposal lands in.
	Spec string
	// Submit records the proposal. It does not measure it and does not write
	// anything: the loop does both, after the turn.
	//
	// Injected because it keeps this package from importing the loop, which
	// imports this one — and because what happens to a proposal is not a
	// tool's business. A tool is the boundary the model reaches through.
	Submit func(ctx context.Context, in DoneProposeInput) (string, error)
}

// DoneProposeInput is the argument shape.
type DoneProposeInput struct {
	Criteria  []DoneProposeCriterion `json:"criteria"`
	Protected []string               `json:"protected,omitempty"`
}

// DoneProposeCriterion is one candidate.
type DoneProposeCriterion struct {
	Name     string `json:"name"`
	Command  string `json:"command"`
	ExitCode int    `json:"exit_code,omitempty"`
	Expects  string `json:"expects"`
	Why      string `json:"why,omitempty"`
}

func (DonePropose) Name() string { return "done_propose" }

func (DonePropose) Description() string {
	return "Propose how this specification will be known to be finished, as COMMANDS. " +
		"Read the spec and the code first: a criterion is something that runs and exits, " +
		"never a sentence — \"Lighthouse >= 95\" is what a person writes and `pnpm lhci --assert` " +
		"is what decides.\n\n" +
		"For each one say what you EXPECT it to do right now, before any work: `fail` for " +
		"something the work has to make true, `pass` for something that already works and " +
		"must keep working. Every one is run before anybody sees it, and what you said is " +
		"compared with what happened — a criterion you expected to fail and which passes is " +
		"measuring nothing.\n\n" +
		"Propose the smallest set that would convince a reviewer, and leave out what you " +
		"cannot check from a command line."
}

func (DonePropose) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{` +
		`"criteria":{"type":"array","description":"The conditions of done, as commands.",` +
		`"items":{"type":"object","properties":{` +
		`"name":{"type":"string","description":"A short handle the report prints."},` +
		`"command":{"type":"string","description":"What runs. It decides; nothing else does."},` +
		`"exit_code":{"type":"integer","description":"What counts as met. Zero unless you say otherwise."},` +
		`"expects":{"type":"string","enum":["fail","pass"],` +
		`"description":"What this does NOW, before the work: fail for something to be made true, pass for a regression guard."},` +
		`"why":{"type":"string","description":"One line for the person reviewing this."}},` +
		`"required":["name","command","expects"]}},` +
		`"protected":{"type":"array","items":{"type":"string"},` +
		`"description":"Globs that ARE the measurement — the test files. Changing one gets surfaced."}},` +
		`"required":["criteria"]}`)
}

// Declare touches nothing, and that is the design rather than a shortcut.
//
// The first version declared a write to the spec folder, which was honest
// about the consequence and wrong about the actor. A qualifying turn runs in
// plan mode — read-only, because working out what you will be measured by is
// reading — and read-only denies every write with no exception. So a tool that
// declared one was denied, and the model correctly reported that it could not
// propose.
//
// Nothing here writes. The proposal is RECORDED, the turn ends, and the LOOP
// measures it and writes it down afterwards, under the boundary the work will
// actually run under. Keeping the write out of the turn is what lets plan mode
// stay a guarantee with no hole in it — and it measures the criteria somewhere
// they can actually run, which read-only is not.
func (d DonePropose) Declare(json.RawMessage) (policy.Request, error) {
	return policy.Request{Tool: d.Name()}, nil
}

func (d DonePropose) Execute(ctx context.Context, input json.RawMessage, _ *State) (Result, error) {
	var in DoneProposeInput
	if err := decode(d.Name(), input, &in); err != nil {
		return err.(*ToolError).Result(), nil
	}
	if d.Submit == nil {
		return errf(d.Name(), CodeDenied, "",
			"done_propose is only available in a qualifying turn").Result(), nil
	}
	if len(in.Criteria) == 0 {
		return errf(d.Name(), CodeBadInput,
			"Propose at least one criterion that runs.",
			"a definition of done with nothing in it reports done").Result(), nil
	}
	if terr := d.check(in); terr != nil {
		return terr.Result(), nil
	}

	out, err := d.Submit(ctx, in)
	if err != nil {
		return errf(d.Name(), CodeNotFound, "", "%v", err).Result(), nil
	}
	return Result{Output: out}, nil
}

// check refuses what the harness above should never be handed.
//
// Here rather than there because the model is the one that has to correct it,
// and an error it reads is what makes that possible.
func (d DonePropose) check(in DoneProposeInput) *ToolError {
	seen := map[string]bool{}
	for _, c := range in.Criteria {
		name := strings.TrimSpace(c.Name)
		if name == "" || strings.TrimSpace(c.Command) == "" {
			return errf(d.Name(), CodeBadInput,
				"Give every criterion a name and a command.",
				"a criterion with no %s is not one", either(name == "", "name", "command"))
		}
		if seen[name] {
			return errf(d.Name(), CodeBadInput,
				"Use a different handle for each.",
				"two criteria are both called %q, and the report cannot tell them apart", name)
		}
		seen[name] = true
		switch strings.TrimSpace(c.Expects) {
		case "fail", "pass":
		default:
			return errf(d.Name(), CodeBadInput,
				"Say `fail` for something the work has to make true, `pass` for a guard.",
				"%q is not something a criterion can be expected to do", c.Expects)
		}
	}
	return nil
}

func either(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
