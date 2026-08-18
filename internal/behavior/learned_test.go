package behavior

import (
	"strings"
	"testing"
)

// Nothing the agent taught itself outranks anything a person wrote.
//
// This is the guarantee the whole component rests on. Without it the memory is
// the path by which the agent slowly rewrites its own constraints — each session
// appending a note the next session reads as established fact.
//
// Asserted over every human source rather than over `user` alone: `user` is the
// weakest of them today, and a test written against the weakest stops being the
// test it was the moment somebody adds a weaker one.
func TestNothingLearnedOutranksAnythingAPersonWrote(t *testing.T) {
	human := []InstructionSource{SourceUser, SourceProject, SourceDirectory, SourceLocked}
	for _, h := range human {
		if authority[SourceLearned] >= authority[h] {
			t.Errorf("learned (%d) is not below %s (%d)",
				authority[SourceLearned], h, authority[h])
		}
	}
}

// Stacking puts the most specific last, which is the position of greatest
// weight. Learned goes first, whatever order it arrives in.
func TestLearnedIsRenderedBeforeEveryHumanInstruction(t *testing.T) {
	out, err := Build(Prompt{
		Doctrine: DefaultDoctrine([]string{"read"}),
		Instructions: []Instruction{
			{Source: SourceLocked, Text: "LOCKED-RULE"},
			{Source: SourceLearned, Scope: ".dcode/memory.md", Text: "LEARNED-NOTE"},
			{Source: SourceUser, Text: "USER-RULE"},
			{Source: SourceProject, Text: "PROJECT-RULE"},
		},
	}, FormulationFor(""))
	if err != nil {
		t.Fatal(err)
	}

	learned := strings.Index(out, "LEARNED-NOTE")
	if learned < 0 {
		t.Fatal("the learned note is not in the prompt")
	}
	for _, rule := range []string{"USER-RULE", "PROJECT-RULE", "LOCKED-RULE"} {
		if at := strings.Index(out, rule); at < learned {
			t.Errorf("%s appears before the learned note, giving the note more weight", rule)
		}
	}
}

// The model has to be able to tell what a person required from what it noted
// itself. An instruction with no provenance is one nobody can argue with.
func TestThePromptNamesLearnedProvenanceAsLearned(t *testing.T) {
	out, err := Build(Prompt{
		Doctrine:     DefaultDoctrine([]string{"read"}),
		Instructions: []Instruction{{Source: SourceLearned, Scope: ".dcode/memory.md", Text: "a note"}},
	}, FormulationFor(""))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, string(SourceLearned)) {
		t.Errorf("the prompt does not say the note was learned:\n%s", out)
	}
	if !strings.Contains(out, ".dcode/memory.md") {
		t.Errorf("the prompt does not say where the note came from:\n%s", out)
	}
}

// The ordering is code, not configuration. A guarantee a setting can switch off
// is not a guarantee — the reasoning that keeps Safety out of the overlay.
func TestNoConfigurationReachesTheAuthorityTable(t *testing.T) {
	before := authority[SourceLearned]
	// Everything a caller can hand Build, at once. None of it may move the
	// learned source.
	_, err := Build(Prompt{
		Doctrine:   DefaultDoctrine([]string{"read", "edit"}).Apply(DoctrineOverlay{Identity: "x", Style: "y"}),
		Tools:      []string{"read", "edit"},
		SkillIndex: []SkillIndexEntry{{Name: "s", WhenToUse: "w"}},
		Instructions: []Instruction{
			{Source: SourceLearned, Text: "note"},
			{Source: SourceLocked, Locked: true, Text: "rule"},
		},
	}, FormulationFor("minimax-m3"))
	if err != nil {
		t.Fatal(err)
	}
	if authority[SourceLearned] != before {
		t.Fatal("assembling a prompt changed where learned instructions rank")
	}
	if before != 0 {
		t.Errorf("learned ranks %d, want 0 — below every human source", before)
	}
}
