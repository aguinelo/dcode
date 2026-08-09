package behavior

import (
	"sort"
	"strings"
)

// ReminderKind identifies a reminder. The text is constant per kind, which is
// what keeps the history reproducible.
type ReminderKind string

const (
	ReminderFileChanged           ReminderKind = "file_changed"
	ReminderApprovalDenied        ReminderKind = "approval_denied"
	ReminderCompacted             ReminderKind = "compacted"
	ReminderToolsParallel         ReminderKind = "tools_parallel"
	ReminderInstructionOutOfChain ReminderKind = "instruction_out_of_chain"
)

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
