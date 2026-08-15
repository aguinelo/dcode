package config

import "testing"

// Granting returns a new set rather than editing the one in hand. The standing
// grants are read from a file and handed around; a setter that mutated in place
// would let a grant given for one workspace appear in a copy somebody else was
// still holding, which is a permission nobody gave.
func TestGrantingLeavesTheSetItWasGivenAlone(t *testing.T) {
	base := Grants{networkProjects: map[string]struct{}{}}

	one := base.GrantNetwork("/a")
	if base.Network("/a") {
		t.Error("granting for one workspace changed the set it was given")
	}
	if !one.Network("/a") {
		t.Error("the grant did not land in the new set")
	}

	two := one.GrantNetwork("/b")
	if one.Network("/b") {
		t.Error("a second grant reached back into the first set")
	}
	if !two.Network("/a") || !two.Network("/b") {
		t.Error("the second set lost the first grant")
	}

	// Always is the same story, and it must carry the per-workspace grants
	// forward rather than replacing them: answering "always" does not withdraw
	// what was already answered.
	always := two.GrantNetworkAlways()
	if one.Network("/never-asked") {
		t.Error("granting always reached back into an earlier set")
	}
	if !always.Network("/never-asked") {
		t.Error("always does not allow a workspace nobody asked about")
	}
	if !always.Network("/a") {
		t.Error("granting always dropped a grant already given")
	}
}

// A malformed array is an error rather than a value read as far as it parsed.
// Half a list silently becoming a shorter list is a setting that looks applied
// and is not.
func TestAnUnterminatedArrayIsAnErrorNotAShorterList(t *testing.T) {
	if _, err := parseStringArray(`["a", "b"`); err == nil {
		t.Error("an unterminated array parsed successfully")
	}
	got, err := parseStringArray(`["a", "b"]`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "a,b" {
		t.Errorf("got %q, want a,b", got)
	}
	if got, err := parseStringArray(`[]`); err != nil || got != "" {
		t.Errorf("an empty array gave %q, %v", got, err)
	}
}
