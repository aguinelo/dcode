package evals

import (
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/app"
	"github.com/aguinelo/dcode/internal/provider"
)

// A named family in this repository reads as a measured one.
//
// `Measurement.Model` exists precisely because a threshold belongs to a model
// and says nothing about another, so a family that carries a name and no
// measurement is a name standing where verification is not. That is the defect
// this project keeps finding in itself, and adding a second unmeasured family —
// gemini, after generic — is when a chain of equality checks in app.go would
// have started going quietly wrong.
//
// So the list of families that warn is checked against the measurements that
// actually exist, in both directions. Nothing here is typed: a family that gets
// measured and stays on the list fails, and one that leaves the list without a
// measurement fails too.
func TestEveryUnmeasuredFamilySaysSo(t *testing.T) {
	families := app.Families()
	if len(families) == 0 {
		t.Fatal("no families registered; this guard would pass reading nothing")
	}

	for _, f := range families {
		measured := false
		for _, m := range Measured {
			if familyClaims(f, m.Model) {
				measured = true
				break
			}
		}
		warning := provider.Unmeasured(f.Name())

		switch {
		case measured && warning != "":
			t.Errorf("%s has measurements and still warns that it has none; "+
				"the warning outlived what it described", f.Name())
		case !measured && warning == "":
			t.Errorf("%s carries a name and not one measurement, and says nothing about it. "+
				"A family name reads as a measured family; add it to provider.Unmeasured "+
				"or measure a contract against it", f.Name())
		}
	}
}

// The warning has to name the family it is about. A session that says something
// is unmeasured without saying what leaves the reader to guess, and a reader who
// guesses wrong reads every difference as a defect in dcode.
func TestAnUnmeasuredWarningNamesItsFamily(t *testing.T) {
	for _, f := range app.Families() {
		w := provider.Unmeasured(f.Name())
		if w == "" {
			continue
		}
		if !strings.Contains(w, f.Name()) {
			t.Errorf("the warning for %s does not name it: %q", f.Name(), w)
		}
		if !strings.Contains(w, "MiniMax-M3") && f.Name() != provider.GenericName {
			t.Errorf("the warning for %s does not say what the thresholds WERE measured "+
				"against, which is the fact that makes the caveat readable: %q", f.Name(), w)
		}
	}
}

// familyClaims reports whether a measured model name belongs to this family.
//
// Derived from the family's own Models() prefixes rather than from a second
// table mapping one to the other: `MiniMax-M3` is the model and `minimax-m3` is
// the family, and a table saying so is a copy of a truth that already exists.
func familyClaims(f provider.Family, model string) bool {
	for _, prefix := range f.Models() {
		if prefix != "" && strings.HasPrefix(strings.ToLower(model), strings.ToLower(prefix)) {
			return true
		}
	}
	return false
}
