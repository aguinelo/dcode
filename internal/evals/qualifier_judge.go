package evals

import (
	"encoding/json"
	"strings"
	"unicode"

	"github.com/aguinelo/dcode/internal/tools"
)

// proposals reads every definition of done the model handed over.
//
// Through the product's own argument type rather than a shape written here:
// the schema is the contract's surface, and a judge parsing a copy of it would
// keep scoring after the product changed what it asks for.
func proposals(t Transcript) []tools.DoneProposeInput {
	var out []tools.DoneProposeInput
	for _, c := range t.Calls {
		if c.Name != "done_propose" {
			continue
		}
		var in tools.DoneProposeInput
		if err := json.Unmarshal(c.Input, &in); err != nil {
			continue
		}
		out = append(out, in)
	}
	return out
}

// lastProposal is the one that stands. A model that proposes twice has changed
// its mind, and the product keeps only the second.
func lastProposal(t Transcript) (tools.DoneProposeInput, bool) {
	all := proposals(t)
	if len(all) == 0 {
		return tools.DoneProposeInput{}, false
	}
	return all[len(all)-1], true
}

// EveryCriterionIsACommand is the first thing the qualifying turn is for: what
// comes back has to be something that runs and exits.
//
// "Lighthouse >= 95" is what a person writes on a whiteboard and `pnpm lhci
// --assert` is what decides, and the difference is the whole reason this phase
// produces commands instead of prose.
//
// It is a check on SHAPE, and the limit is worth stating plainly: nothing here
// can tell a command that will run from one that will not. The harness does not
// execute what a model wrote — that refusal is older than this contract and
// deliberate — so a criterion naming a script that does not exist passes this
// judge and would come back 127 from the product's own measurement. What this
// catches is the failure the contract is named for: a sentence in the field
// where a command belongs.
func EveryCriterionIsACommand() Judge {
	return func(t Transcript) bool {
		in, ok := lastProposal(t)
		if !ok || len(in.Criteria) == 0 {
			return false
		}
		for _, c := range in.Criteria {
			if !isCommand(c.Command) {
				return false
			}
		}
		return true
	}
}

// proseInCommand are words a shell command does not contain outside quotes and
// a sentence about a threshold almost always does.
//
// Padded with spaces so `theme.css` is not prose because it starts with "the".
var proseInCommand = []string{
	" is ", " are ", " should ", " must ", " and ", " or ", " the ",
	" at least ", " greater than ", " less than ", " when ", " than ",
}

func isCommand(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	// A program name is not capitalised in any ecosystem this runs against,
	// and a sentence starts with a capital. It is a convention rather than a
	// rule, and it is the one that separates `pnpm lhci` from `Lighthouse >=`.
	if r := []rune(s)[0]; unicode.IsUpper(r) {
		return false
	}
	first := strings.Fields(s)[0]
	for _, r := range first {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) ||
			strings.ContainsRune("_./-+:[", r)) {
			return false
		}
	}
	padded := " " + strings.ToLower(s) + " "
	for _, w := range proseInCommand {
		if strings.Contains(padded, w) {
			return false
		}
	}
	return true
}

// ProposesAGuard asks for at least one criterion the model expects to pass
// already.
//
// A set made only of things that are red today says what the work must add and
// nothing about what it must not break. On a codebase with a suite that runs,
// leaving that out is the proposal declining to notice the strongest evidence
// in the room.
func ProposesAGuard() Judge {
	return func(t Transcript) bool {
		in, ok := lastProposal(t)
		if !ok {
			return false
		}
		for _, c := range in.Criteria {
			if strings.TrimSpace(c.Expects) == "pass" {
				return true
			}
		}
		return false
	}
}

// KeepsMeasuring is the answer to a criterion that came back broken: what it
// measured survives, and the command that measured nothing does not.
//
// Named by SUBJECT and not by handle, and that correction cost a measurement.
// The first version demanded a criterion with the same NAME, and it scored 30%
// over twenty runs against a threshold of 85% — while every failing transcript
// showed the model doing exactly the right thing:
//
//	# [slug]  gotestsum --run TestSlugify ./...   (broken, 127)
//	→ slug-test          go test -run TestSlugify ./...
//	→ slug-impl-exists   grep -q 'func Slugify' stats.go
//	→ basic              go test -run TestSlugify ./...
//
// It kept the measurement and gave it a better handle. The product's own type
// says what a name is — "what the report prints" — so a judge that insists on
// one measures vocabulary, which is the exact reason the contract beside this
// one was retired rather than built.
//
// Both halves stay, because the two ways to fail are still opposite. Dropping
// the subject leaves the spec measuring less than it did. Re-proposing the
// broken command redeclares one already shown to run nothing.
func KeepsMeasuring(subject, broken string) Judge {
	subject = strings.ToLower(subject)
	broken = strings.TrimSpace(broken)
	return func(t Transcript) bool {
		in, ok := lastProposal(t)
		if !ok {
			return false
		}
		kept := false
		for _, c := range in.Criteria {
			if strings.TrimSpace(c.Command) == broken {
				return false
			}
			if strings.Contains(strings.ToLower(c.Name), subject) ||
				strings.Contains(strings.ToLower(c.Command), subject) {
				kept = true
			}
		}
		return kept
	}
}
