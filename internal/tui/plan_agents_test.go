package tui

import (
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The orchestrator's plan names, under each step, the children that fulfil
// it and the state each is in (desktop D68, in text): a step with no child is
// drawn as before, and a child with no step is not drawn under any.
func TestThePlanNamesEachStepsChildrenAndTheirState(t *testing.T) {
	m := modelWithPlan()
	m.Agents.Children = []Agent{
		{Child: protocol.Child{ID: "c1", Title: "index the words", State: childInReview, PlanStep: 2}},
		{Child: protocol.Child{ID: "c2", Title: "write the notes", State: childMerged, PlanStep: 2}},
		{Child: protocol.Child{ID: "c3", Title: "loose child", State: childWorking}},
	}
	g := DefaultGeometry(100, 40)
	g.Palette = Palette{}
	out := Render(m, g)
	var at, review, merged int
	for i, l := range lines(out) {
		switch {
		case strings.Contains(l, "add the test"):
			at = i
		case strings.Contains(l, "index the words"):
			review = i
			if !strings.Contains(l, "in review") {
				t.Errorf("the child in review does not say so: %q", l)
			}
		case strings.Contains(l, "write the notes"):
			merged = i
			if !strings.Contains(l, "merged") {
				t.Errorf("the merged child does not say so: %q", l)
			}
		}
	}
	if at == 0 || review != at+1 || merged != at+2 {
		t.Errorf("the children of step 2 are not under it (step %d, children %d and %d):\n%s", at, review, merged, out)
	}
	if strings.Contains(out, "loose child") {
		t.Errorf("a child with no plan step was drawn under the plan:\n%s", out)
	}
}
