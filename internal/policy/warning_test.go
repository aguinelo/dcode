package policy

import (
	"strings"
	"testing"
)

// The combination the protocol spec required a boot warning for, and that
// nothing ever raised.
func TestFullAccessWithNeverAskIsWarnedAbout(t *testing.T) {
	got := BoundaryWarning(ModeFullAccess, PolicyNever)
	if got == "" {
		t.Fatal("the one combination that removes every boundary says nothing")
	}
	for _, want := range []string{"full-access", "never", "no boundary"} {
		if !strings.Contains(got, want) {
			t.Errorf("the warning does not carry %q: %s", want, got)
		}
	}
}

// Either alone is a deliberate, defensible choice, and warning about them would
// train someone to ignore the warning that matters.
func TestNeitherHalfAloneWarns(t *testing.T) {
	cases := []struct {
		mode SandboxMode
		pol  ApprovalPolicy
	}{
		{ModeFullAccess, PolicyOnRequest},
		{ModeFullAccess, PolicyUntrusted},
		{ModeWorkspaceWrite, PolicyNever},
		{ModeReadOnly, PolicyNever},
		{ModeWorkspaceWrite, PolicyOnRequest},
	}
	for _, c := range cases {
		if got := BoundaryWarning(c.mode, c.pol); got != "" {
			t.Errorf("%s + %s warned: %s", c.mode, c.pol, got)
		}
	}
}

// It warns rather than refuses. Someone running a throwaway container wants
// exactly this combination, and a product that argues with them there is one
// they route around — what it must not do is let it happen quietly.
func TestItWarnsAndDoesNotRefuse(t *testing.T) {
	w := BoundaryWarning(ModeFullAccess, PolicyNever)
	if strings.Contains(strings.ToLower(w), "refus") || strings.Contains(strings.ToLower(w), "cannot") {
		t.Errorf("the warning reads as a refusal: %s", w)
	}
}
