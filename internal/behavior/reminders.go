package behavior

import (
	"sort"
	"strings"
)

// ReminderKind identifies a reminder. The text is constant per kind, which is
// what keeps the history reproducible.
type ReminderKind string

const (
	ReminderFileChanged             ReminderKind = "file_changed"
	ReminderApprovalDenied          ReminderKind = "approval_denied"
	ReminderCompacted               ReminderKind = "compacted"
	ReminderToolsParallel           ReminderKind = "tools_parallel"
	ReminderInstructionOutOfChain   ReminderKind = "instruction_out_of_chain"
	ReminderContextBudget           ReminderKind = "context_budget"
	ReminderUnmetCriteria           ReminderKind = "unmet_criteria"
	ReminderVerificationUnavailable ReminderKind = "verification_unavailable"
	ReminderUnplannedChange         ReminderKind = "unplanned_change"
	ReminderProtectedTouched        ReminderKind = "protected_touched"
	ReminderInterrupted             ReminderKind = "interrupted"
)

// BudgetBand is how full the context is, as announced to the model.
//
// Mirrors contextengine.Band as a plain integer rather than importing it: the
// reminder channel is about text, and the arithmetic belongs where the window
// is measured. Importing would also point this package at one that assembles
// prompts, which is backwards.
type BudgetBand int

const (
	BudgetNone BudgetBand = iota
	Budget60
	Budget80
	Budget92
)

// budgetTexts is what each band asks for. One Kind with three texts, not three
// Kinds: the rule is one — spend what is left deliberately — and what changes
// is only how much is left.
//
// Each text is a CONSTANT. No number is interpolated, deliberately: a value
// that moves every turn makes the history irreproducible (RN-7 of the context
// engine), and the band already carries everything the model can act on.
var budgetTexts = map[BudgetBand]string{
	Budget60: "You have used about 60% of the context you get before earlier " +
		"history is summarised away. Prefer reading the part of a file you need " +
		"over reading all of it.",
	Budget80: "You have used about 80% of the context you get before earlier " +
		"history is summarised away. Write anything you have learned that must " +
		"survive that summary to a file — saying it in your answer does not " +
		"survive, because your answer is part of what gets summarised. Finish " +
		"what is open before starting something new.",
	Budget92: "You are close to the point where earlier history is summarised " +
		"away. If the remaining work does not fit in what is left, say so now " +
		"rather than starting it and losing the thread partway through.",
}

// Reminder is one appended note.
//
// Appended, never prefixed. That is the whole point of the channel: it steers
// the model mid-turn without touching the prefix, so the cache survives. A
// reminder in the prefix would cost the entire cached prompt every time
// anything changed on disk.
type Reminder struct {
	Kind ReminderKind
	Text string
}

// OutOfChainInstruction is an instruction file found in a directory the session
// touched, outside the chain frozen at session creation.
type OutOfChainInstruction struct {
	Path string
	Text string
}

// SessionState is everything Emit is a function of. Nothing else is read: no
// clock, no filesystem, no globals.
type SessionState struct {
	// ChangedFiles are paths read this session whose content on disk no longer
	// matches what the model was shown.
	ChangedFiles []string
	// DeniedTools are tool names whose boundary crossing the user refused.
	DeniedTools []string
	// Compacted reports that history was summarised since the last reminder.
	Compacted bool
	// ParallelBatch is the size of the batch that just ran. Only the fact that
	// it exceeds one reaches the text — the number itself would vary between
	// otherwise identical runs.
	ParallelBatch int
	// OutOfChain are instruction files discovered after the chain was frozen.
	OutOfChain []OutOfChainInstruction
	// UnmetCriteria are the names of the done criteria still not met. Set only
	// when the turn is being asked to continue because of them.
	UnmetCriteria []string
	// ProtectedTouched are paths that ARE the measurement and were written this
	// turn. Surfaced, never counted as progress in silence.
	ProtectedTouched []string
	// UnplannedChange reports that work has spread across several files with
	// no plan recorded.
	//
	// A boolean and not a count: the count would vary between otherwise
	// identical runs and reach the model through the text, which is what RN-7
	// of the context engine forbids and why the budget texts are constants too.
	UnplannedChange bool
	// VerificationUnavailable reports that files changed and no criterion was
	// able to run at all.
	//
	// Different from UnmetCriteria, which means a check ran and said no. Here
	// nothing ran, so there is no failure to fix and nothing to try again —
	// only something to admit.
	VerificationUnavailable bool
	// BudgetCrossed is the occupancy band to announce, set only on the turn the
	// band is crossed upward. BudgetNone announces nothing.
	//
	// The caller decides whether a crossing happened, because that needs the
	// previous band, and Emit is a function of this state alone.
	BudgetCrossed BudgetBand
}

// Emit is PURE: the same state always produces the same reminders, in the same
// order. That is what makes a replayed history byte-identical to a live one.
func Emit(s SessionState) []Reminder {
	var out []Reminder

	if len(s.ChangedFiles) > 0 {
		paths := append([]string(nil), s.ChangedFiles...)
		sort.Strings(paths)
		out = append(out, Reminder{
			Kind: ReminderFileChanged,
			Text: "These files changed on disk since you read them: " +
				strings.Join(paths, ", ") +
				". Read a file again before editing it — editing from content you " +
				"no longer have is how work gets overwritten.",
		})
	}

	if len(s.DeniedTools) > 0 {
		names := uniqueSorted(s.DeniedTools)
		out = append(out, Reminder{
			Kind: ReminderApprovalDenied,
			Text: "The user refused this: " + strings.Join(names, ", ") +
				". Do not retry it and do not look for another route to the same " +
				"effect. Say what you cannot do, and carry on with what you can.",
		})
	}

	if s.Compacted {
		out = append(out, Reminder{
			Kind: ReminderCompacted,
			Text: "Earlier history was summarised to fit the window. The summary " +
				"is above. If you need a detail it does not carry, read the file " +
				"again rather than recalling it.",
		})
	}

	if s.ParallelBatch > 1 {
		out = append(out, Reminder{
			Kind: ReminderToolsParallel,
			Text: "Those tools ran at the same time, so their results do not " +
				"describe a sequence. Do not read one as having happened before " +
				"another, and do not infer that one caused the next.",
		})
	}

	if len(s.UnmetCriteria) > 0 {
		names := uniqueSorted(s.UnmetCriteria)
		out = append(out, Reminder{
			Kind: ReminderUnmetCriteria,
			Text: "You changed files and this is not done yet: " +
				strings.Join(names, ", ") + " did not pass. Fix the cause. " +
				"Do not weaken the check to make it pass, and do not report " +
				"success — if you cannot get there, say what is left.",
		})
	}

	if s.UnplannedChange {
		out = append(out, Reminder{
			Kind: ReminderUnplannedChange,
			Text: "You have changed several files and there is no plan. " +
				"Call `plan` with the steps you are working through, and keep it current. " +
				"The person watching sees the plan, not this reasoning — without one " +
				"they cannot follow the work or stop it early, which is the whole " +
				"reason the plan is a tool and not a paragraph.",
		})
	}

	if s.VerificationUnavailable {
		out = append(out, Reminder{
			Kind: ReminderVerificationUnavailable,
			Text: "You changed files and there is no check here that could confirm them. " +
				"Say plainly what you changed and what you could not verify. " +
				"Do not claim it works, and do not go looking for something to run — " +
				"the honest answer is that it was not checked.",
		})
	}

	if len(s.ProtectedTouched) > 0 {
		paths := uniqueSorted(s.ProtectedTouched)
		out = append(out, Reminder{
			Kind: ReminderProtectedTouched,
			Text: "You changed files that are part of how this work is checked: " +
				strings.Join(paths, ", ") + ". That is sometimes the right thing " +
				"to do, and it is being recorded either way. Say why you changed " +
				"them.",
		})
	}

	if text, ok := budgetTexts[s.BudgetCrossed]; ok {
		out = append(out, Reminder{Kind: ReminderContextBudget, Text: text})
	}

	for _, oc := range sortedOutOfChain(s.OutOfChain) {
		out = append(out, Reminder{
			Kind: ReminderInstructionOutOfChain,
			Text: "You worked in a directory carrying its own instructions, at " +
				oc.Path + ", which were not loaded when this session started. " +
				"They apply to that directory and they take precedence over the " +
				"project's:\n\n" + strings.TrimSpace(oc.Text),
		})
	}

	return out
}

// Render wraps a reminder in the marker the base doctrine teaches the model to
// recognise.
//
// Without the marker the model reads a reminder as the user speaking, and
// answers it — which is both wrong and unnerving to watch.
func Render(r Reminder) string {
	return "<system-reminder>\n" + r.Text + "\n</system-reminder>"
}

func uniqueSorted(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func sortedOutOfChain(in []OutOfChainInstruction) []OutOfChainInstruction {
	out := append([]OutOfChainInstruction(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
