package evals

import (
	"strings"
	"testing"
)

func TestTheSkillBodyReachesTheModel(t *testing.T) {
	f, err := LoadFixture(FixtureRoot, "skill-loaded-on-trigger")
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Skills) != 2 {
		t.Fatalf("got %d skills, want the release one and the decoy", len(f.Skills))
	}
	opening := f.Opening()
	var body string
	for _, m := range opening {
		if m.Reminder {
			body += m.Text
		}
	}
	if !strings.Contains(body, "RELEASING.md") {
		t.Errorf("the triggered skill's body did not reach the model:\n%v", opening)
	}
	if strings.Contains(body, "migração") {
		t.Errorf("a skill that does not match the task was loaded anyway:\n%s", body)
	}
	prompt, err := f.Prompt("")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(prompt, "RELEASING.md") {
		t.Errorf("the skill body is in the prefix; only the index belongs there (RN-7):\n%s", prompt)
	}
}
