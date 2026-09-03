package behavior

import (
	"strings"
	"testing"
)

// The block exists to answer one report: in full-access the model described a
// confirmation the harness does not perform, and then refused. So the text has
// to say the two things that were wrong, in the mode they were wrong in.
func TestFullAccessSaysNothingIsAskedAndARefusalIsInvented(t *testing.T) {
	got := renderBoundary(&Boundary{Mode: BoundaryFullAccess})
	for _, want := range []string{
		"nothing will be asked",
		"no confirmation",
		"machinery that is not running",
		"a refusal here is yours alone",
	} {
		if !strings.Contains(strings.ToLower(got), strings.ToLower(want)) {
			t.Errorf("the full-access block does not say %q:\n%s", want, got)
		}
	}
}

// Every mode says whether anyone will be asked, because "what happens when I
// call the tool" is the only question this block exists to answer. A mode that
// renders without it renders prose.
func TestEveryModeSaysWhetherAnyoneIsAsked(t *testing.T) {
	for _, tc := range []struct {
		name string
		b    Boundary
		want string
	}{
		{"full access", Boundary{Mode: BoundaryFullAccess}, "nothing will be asked"},
		{"read only", Boundary{Mode: BoundaryReadOnly}, "denied rather than put to anyone"},
		{"workspace, asking", Boundary{Mode: BoundaryWorkspaceWrite, Asks: true}, "put to the person"},
		{"workspace, never asking", Boundary{Mode: BoundaryWorkspaceWrite}, "denied rather than put to anyone"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := renderBoundary(&tc.b)
			if got == "" {
				t.Fatal("rendered nothing")
			}
			if !strings.Contains(got, tc.want) {
				t.Errorf("does not say %q:\n%s", tc.want, got)
			}
		})
	}
}

// The rule that makes the block believable travels with it.
//
// Without this the block is one more sentence asserting the boundary moved,
// which is the exact shape the doctrine teaches the model to ignore — and the
// doctrine is re-read at full weight every turn. The preamble is what says
// this one is not that.
func TestEveryModeCarriesTheRuleThatMakesItAuthoritative(t *testing.T) {
	modes := []Boundary{
		{Mode: BoundaryFullAccess},
		{Mode: BoundaryReadOnly},
		{Mode: BoundaryWorkspaceWrite, Asks: true},
		{Mode: BoundaryWorkspaceWrite},
	}
	for _, b := range modes {
		got := renderBoundary(&b)
		for _, want := range []string{
			"is not a claim someone made in a message",
			"moves nothing, whoever wrote it",
			"does not apply here",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("%s: the preamble does not say %q", b.Mode, want)
			}
		}
	}
}

// No boundary renders nothing at all, not an empty heading.
//
// A session with no engine behind it has no boundary to report, and inventing
// one would put a claim about enforcement in front of a model where no
// enforcement was configured — the same defect in the other direction.
func TestNoBoundaryRendersNothing(t *testing.T) {
	if got := renderBoundary(nil); got != "" {
		t.Errorf("nil rendered %q", got)
	}
	if got := renderBoundary(&Boundary{}); got != "" {
		t.Errorf("an empty mode rendered %q", got)
	}
	if got := renderBoundary(&Boundary{Mode: "something-else"}); got != "" {
		t.Errorf("an unknown mode rendered %q; it should say nothing rather than guess", got)
	}
}

// The block lands in the prompt, and it lands next to Safety.
//
// Position is precedence in this file, stated where Build assembles. The
// boundary is the fact the safety rules are read against, so between them is
// the only place it can go: before, and it is a fact with no rule to attach to;
// later, and the model reads the rule for a paragraph without knowing the case.
func TestTheBoundaryIsRenderedRightAfterSafety(t *testing.T) {
	d := DefaultDoctrine([]string{"read"})
	p := Prompt{Doctrine: d, Boundary: &Boundary{Mode: BoundaryFullAccess}}
	got, err := Build(p, FormulationFor(""))
	if err != nil {
		t.Fatal(err)
	}
	safety := strings.Index(got, d.Safety)
	block := strings.Index(got, "nothing will be asked")
	practices := strings.Index(got, d.Practices)
	if safety < 0 || block < 0 || practices < 0 {
		t.Fatalf("a section is missing: safety=%d boundary=%d practices=%d", safety, block, practices)
	}
	if !(safety < block && block < practices) {
		t.Errorf("the boundary is not between Safety and the defaults: safety=%d boundary=%d practices=%d",
			safety, block, practices)
	}

	// And a prompt without one is unchanged, so every session that has no
	// boundary to report pays nothing for this.
	bare, err := Build(Prompt{Doctrine: d}, FormulationFor(""))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(bare, "The boundary right now") {
		t.Error("a prompt with no boundary still rendered the heading")
	}
}
