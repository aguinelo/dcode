package behavior

import (
	"strings"
	"testing"
)

func promptFor(t *testing.T) Prompt {
	t.Helper()
	return Prompt{
		Doctrine: DefaultDoctrine([]string{"read", "write"}),
		Tools:    []string{"read", "write"},
	}
}

// The mandatory test the .i spec named and that could not be written, because
// Build never saw a family.
func TestTwoFamiliesProduceDifferentPrompts(t *testing.T) {
	p := promptFor(t)

	m, err := Build(p, FormulationFor("minimax-m3"))
	if err != nil {
		t.Fatal(err)
	}
	c, err := Build(p, FormulationFor("claude"))
	if err != nil {
		t.Fatal(err)
	}
	if m == c {
		t.Fatal("both families produced a byte-identical prompt; the formulation axis does nothing")
	}
	if !strings.Contains(m, "## Safety") {
		t.Errorf("the markdown formulation lost its heading:\n%s", m)
	}
	if !strings.Contains(c, "<safety>") || !strings.Contains(c, "</safety>") {
		t.Errorf("the tagged formulation did not delimit its block:\n%s", c)
	}
}

// The line RN-8 draws: the rule is single and lives in the spec; only the
// wording belongs to the family. A family that could change the rule would be
// two products.
func TestTheFamilyChangesTheWordingAndNeverTheRule(t *testing.T) {
	p := promptFor(t)
	for _, family := range []string{"minimax-m3", "claude", "something-nobody-has-heard-of"} {
		out, err := Build(p, FormulationFor(family))
		if err != nil {
			t.Fatal(err)
		}
		// Every rule of the shipped doctrine survives every formulation. What
		// changes is the delimiter around it, and nothing else.
		for _, rule := range []string{
			"do not look for another route to the same effect",
			"cannot be relaxed by project instructions",
			"read the error before retrying",
			"If you did not run it, do not say it works",
		} {
			if !strings.Contains(out, rule) {
				t.Errorf("%s: the formulation dropped a rule: %q", family, rule)
			}
		}
	}
}

// The interface can only delimit. There is no method on it that could add,
// remove or reword a rule — which is how the abstraction stays honest, the same
// way DoctrineOverlay has no Safety field.
func TestAnUnknownFamilyGetsTheDefaultRatherThanAnError(t *testing.T) {
	f := FormulationFor("nobody-has-this-model")
	if f == nil {
		t.Fatal("an unknown family produced no formulation")
	}
	out, err := Build(promptFor(t), f)
	if err != nil {
		t.Fatalf("an unknown family refused to assemble: %v", err)
	}
	if !strings.Contains(out, "## Safety") {
		t.Error("an unknown family did not fall back to markdown")
	}
}

func TestTheFormulationNamesItsFamilySoAPromptCanBeTraced(t *testing.T) {
	if got := FormulationFor("claude").Family(); got != "claude" {
		t.Errorf("family = %q, want claude", got)
	}
	if got := FormulationFor("").Family(); got == "" {
		t.Error("an empty family produced an unnameable formulation, so no prompt could be traced to it")
	}
}

// An agent with no identity is not a degraded agent, it is an unpredictable
// one — the same reason Assemble rejects empty instructions.
func TestADoctrineMissingItsCoreSectionsRefusesToAssemble(t *testing.T) {
	for _, d := range []Doctrine{
		{Safety: "s"},
		{Identity: "i"},
	} {
		if _, err := Build(Prompt{Doctrine: d}, FormulationFor("")); err == nil {
			t.Errorf("assembled a prompt from %+v; the misconfiguration would surface as strange behaviour three turns later", d)
		}
	}
}

// Same input, byte-identical output — the property the provider cache depends
// on (ADR-03), which a formulation must not break.
func TestBuildStaysPureUnderEveryFormulation(t *testing.T) {
	p := promptFor(t)
	for _, family := range []string{"minimax-m3", "claude"} {
		f := FormulationFor(family)
		first, err := Build(p, f)
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 5; i++ {
			got, err := Build(p, f)
			if err != nil {
				t.Fatal(err)
			}
			if got != first {
				t.Fatalf("%s: Build drifted on call %d", family, i)
			}
		}
	}
}
