package behavior

import (
	"strings"
	"testing"
)

func TestNoBudgetReminderWhenNoBandWasCrossed(t *testing.T) {
	for _, r := range Emit(SessionState{}) {
		if r.Kind == ReminderContextBudget {
			t.Fatal("a budget reminder was emitted with no crossing: this is the deterministic half of no-budget-noise-when-low")
		}
	}
}

func TestEachBandHasItsOwnTextUnderOneKind(t *testing.T) {
	seen := map[string]bool{}
	for _, band := range []BudgetBand{Budget60, Budget80, Budget92} {
		rs := Emit(SessionState{BudgetCrossed: band})
		var found *Reminder
		for i := range rs {
			if rs[i].Kind == ReminderContextBudget {
				found = &rs[i]
			}
		}
		if found == nil {
			t.Fatalf("band %v emitted no reminder", band)
		}
		if seen[found.Text] {
			t.Errorf("band %v repeats the text of another band", band)
		}
		seen[found.Text] = true
	}
	if len(seen) != 3 {
		t.Fatalf("got %d distinct texts, want 3", len(seen))
	}
}

// The text is constant per band. A number that moves every turn makes the
// history irreproducible, which is RN-7 of the context engine.
func TestABudgetBandCarriesNoVaryingNumber(t *testing.T) {
	for _, band := range []BudgetBand{Budget60, Budget80, Budget92} {
		first := Emit(SessionState{BudgetCrossed: band})
		second := Emit(SessionState{BudgetCrossed: band})
		if len(first) != len(second) {
			t.Fatal("Emit is not a function of its state")
		}
		for i := range first {
			if first[i] != second[i] {
				t.Fatalf("band %v produced different text on two calls", band)
			}
		}
	}
}

func TestTheThreeBandsAskForDifferentThings(t *testing.T) {
	// 60 is about how to read, 80 is about recording, 92 is about telling the
	// user. If two of them said the same thing there would be no reason for
	// three bands.
	want := map[BudgetBand]string{
		Budget60: "part of a file",
		Budget80: "must survive",
		Budget92: "say so now",
	}
	for band, phrase := range want {
		rs := Emit(SessionState{BudgetCrossed: band})
		var text string
		for _, r := range rs {
			if r.Kind == ReminderContextBudget {
				text = r.Text
			}
		}
		if !strings.Contains(text, phrase) {
			t.Errorf("band %v does not ask what it is for (%q):\n%s", band, phrase, text)
		}
	}
}
