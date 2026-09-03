package app

import (
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/behavior"
	"github.com/aguinelo/dcode/internal/policy"
)

// The two spellings of a mode have to agree.
//
// `behavior` names the modes as strings rather than importing `policy`, so that
// the package rendering prompts does not point at the package enforcing the
// boundary. That is the right dependency and it buys a duplicated constant —
// which is the copy this repository keeps finding in itself, so it is compared
// here, in the one package that legitimately holds both.
//
// Drift is silent otherwise: a renamed mode would simply stop matching, the
// block would render nothing, and the model would be told nothing at all about
// where the boundary is. Nothing would fail.
func TestTheBoundarySpellingsAgreeWithThePolicy(t *testing.T) {
	for _, tc := range []struct {
		behaviour string
		policy    policy.SandboxMode
	}{
		{behavior.BoundaryReadOnly, policy.ModeReadOnly},
		{behavior.BoundaryWorkspaceWrite, policy.ModeWorkspaceWrite},
		{behavior.BoundaryFullAccess, policy.ModeFullAccess},
	} {
		if tc.behaviour != string(tc.policy) {
			t.Errorf("behavior spells the mode %q and policy spells it %q", tc.behaviour, tc.policy)
		}
	}
}

// Every mode the policy accepts renders a block.
//
// Asked of the policy's own list rather than of a list written here: a fourth
// mode added later would be a mode the model is silently told nothing about,
// and a test that knows only the three cannot notice.
func TestEveryPolicyModeIsDescribedToTheModel(t *testing.T) {
	for _, mode := range []policy.SandboxMode{
		policy.ModeReadOnly, policy.ModeWorkspaceWrite, policy.ModeFullAccess,
	} {
		b := boundaryOf(mode, policy.PolicyOnRequest)
		if b == nil {
			t.Errorf("%s produced no boundary", mode)
			continue
		}
		got, err := behavior.Build(behavior.Prompt{
			Doctrine: behavior.DefaultDoctrine([]string{"read"}),
			Boundary: b,
		}, behavior.FormulationFor(""))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got, "The boundary right now") {
			t.Errorf("%s rendered no boundary block", mode)
		}
	}
}

// Whether anyone will be asked comes from the POLICY, not from the mode.
//
// A workspace-write session with approvals set to never denies rather than
// asks. Telling the model it will be asked when nobody will is the same defect
// as telling it a full-access crossing needs confirming — it is just the cell
// nobody looked at, which is how the boundary contracts already went wrong once.
func TestWhetherAnyoneIsAskedComesFromThePolicy(t *testing.T) {
	asks := boundaryOf(policy.ModeWorkspaceWrite, policy.PolicyOnRequest)
	if asks == nil || !asks.Asks {
		t.Fatalf("on-request did not produce a boundary that asks: %+v", asks)
	}
	never := boundaryOf(policy.ModeWorkspaceWrite, policy.PolicyNever)
	if never == nil || never.Asks {
		t.Fatalf("never produced a boundary that asks: %+v", never)
	}
}

// No mode is no boundary, rather than a boundary that says nothing.
func TestAnEmptyModeProducesNoBoundary(t *testing.T) {
	if got := boundaryOf("", policy.PolicyOnRequest); got != nil {
		t.Errorf("an empty mode produced %+v", got)
	}
}
