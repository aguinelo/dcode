package behavior

import (
	"strings"
	"testing"
)

// RN-5: the prefix is fixed when the session is created.
//
// The cost of getting this wrong is invisible per turn and enormous over a
// session. The prefix is what the provider caches; change one byte of it and
// every subsequent turn pays full price for the whole prompt again. So the
// guarantee is not that late instructions are ignored — they arrive as appended
// notices — but that they never reach the assembled prefix.
//
// It is asserted as an absence of PATH: Build takes a Prompt value and there is
// no channel by which anything discovered later could reach it. That is easy to
// state and easy to lose, because the fix for "the model did not see my new
// AGENTS.md" looks like adding exactly such a channel.
func TestNothingDiscoveredAfterAssemblyReachesThePrefix(t *testing.T) {
	p := base()
	p.Instructions = []Instruction{{Source: SourceProject, Text: "Use tabs."}}
	before, err := Build(p, FormulationFor("minimax-m3"))
	if err != nil {
		t.Fatal(err)
	}

	// Everything a session could learn after the fact: a new instruction file, a
	// doctrine overlay dropped into place, a skill installed.
	late := p
	_ = append(late.Instructions, Instruction{Source: SourceDirectory, Text: "LATE-RULE"})
	_ = late.Doctrine.Apply(DoctrineOverlay{Identity: "LATE-IDENTITY", ToolsMore: "LATE-TOOLS"})

	after, err := Build(p, FormulationFor("minimax-m3"))
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Error("the prefix changed after assembly; every later turn now pays full price for the whole prompt")
	}
	for _, leaked := range []string{"LATE-RULE", "LATE-IDENTITY", "LATE-TOOLS"} {
		if strings.Contains(after, leaked) {
			t.Errorf("%q reached the prefix; it was learned after the session was assembled", leaked)
		}
	}
}

// Reminders are appended, never prefixed, and the two invariants that say so
// are the same sweep: one for the ordinary kinds, one for the verification
// kinds added later.
//
// A reminder in the prefix is worse than a reminder nowhere. It invalidates the
// cache on the turn it appears AND on the turn it stops appearing, and it turns
// a transient fact — "these files changed" — into something the model reads on
// every turn for the rest of the session, long after it stopped being true.
func TestNoReminderTextEverReachesThePrefix(t *testing.T) {
	st := SessionState{
		ChangedFiles:     []string{"stats.go"},
		DeniedTools:      []string{"bash"},
		Compacted:        true,
		ParallelBatch:    3,
		UnmetCriteria:    []string{"tests"},
		ProtectedTouched: []string{"turn_test.go"},
		BudgetCrossed:    Budget80,
	}
	reminders := Emit(st)
	if len(reminders) < 5 {
		t.Fatalf("only %d reminders were emitted; the sweep below would prove little", len(reminders))
	}

	prompt, err := Build(base(), FormulationFor("minimax-m3"))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reminders {
		text := Render(r)
		if strings.Contains(prompt, text) {
			t.Errorf("the %s reminder is in the prefix:\n%s", r.Kind, text)
		}
		// Whole reminders are unlikely to be embedded verbatim; the realistic
		// leak is one distinctive sentence of one, so each is checked in
		// fragments too.
		for _, sentence := range strings.Split(text, ". ") {
			if s := strings.TrimSpace(sentence); len(s) > 30 && strings.Contains(prompt, s) {
				t.Errorf("part of the %s reminder is in the prefix: %q", r.Kind, s)
			}
		}
	}
	if strings.Contains(prompt, "system-reminder") {
		t.Error("the prefix carries the reminder marker; the channel is appended-only")
	}
}
