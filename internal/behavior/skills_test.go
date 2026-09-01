package behavior

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestParseSkillReadsFrontmatterAndBody(t *testing.T) {
	s, err := ParseSkill(`---
name: migrations
when_to_use: writing or reviewing a database migration
triggers: [migration, schema change]
---
Always write the down migration first.
`, "migrations.md")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "migrations" {
		t.Errorf("got %q", s.Name)
	}
	if s.WhenToUse == "" || len(s.Triggers) != 2 {
		t.Errorf("got %+v", s)
	}
	if s.Triggers[1] != "schema change" {
		t.Errorf("a multi-word trigger must survive as one phrase, got %q", s.Triggers[1])
	}
	if !strings.HasPrefix(s.Body, "Always write") {
		t.Errorf("got %q", s.Body)
	}
}

// The index line is paid for on every turn of every session, so it has to be
// one line — and a skill with no index line is one the model never learns
// exists.
func TestParseSkillRequiresAShortWhenToUse(t *testing.T) {
	if _, err := ParseSkill("---\nname: x\n---\nbody\n", "x.md"); err == nil {
		t.Error("a skill without when_to_use must be rejected")
	} else if !strings.Contains(err.Error(), "when_to_use") {
		t.Errorf("the error must name the missing field: %v", err)
	}

	long := strings.Repeat("a", MaxWhenToUse+1)
	if _, err := ParseSkill("---\nwhen_to_use: "+long+"\n---\nbody\n", "x.md"); err == nil {
		t.Error("an oversized index line must be rejected")
	}
}

func TestParseSkillRejectsBrokenInput(t *testing.T) {
	for name, body := range map[string]string{
		"unterminated frontmatter": "---\nname: x\n",
		"no body":                  "---\nwhen_to_use: x\n---\n\n",
	} {
		if _, err := ParseSkill(body, "x.md"); err == nil {
			t.Errorf("%s: must be rejected", name)
		}
	}
}

func TestParseSkillAcceptsTheAlternateFieldNames(t *testing.T) {
	for _, field := range []string{"description", "whenToUse"} {
		s, err := ParseSkill("---\n"+field+": doing the thing\n---\nbody\n", "x.md")
		if err != nil {
			t.Fatalf("%s: %v", field, err)
		}
		if s.WhenToUse != "doing the thing" {
			t.Errorf("%s: got %q", field, s.WhenToUse)
		}
	}
}

func TestLoadSkillsReadsBothLayoutsAndLetsTheProjectWin(t *testing.T) {
	home, ws := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(home, "review.md"),
		"---\nwhen_to_use: reviewing code\n---\nuser body\n")
	writeFile(t, filepath.Join(home, "packaged", "SKILL.md"),
		"---\nwhen_to_use: packaging a release\n---\npackaged body\n")
	writeFile(t, filepath.Join(home, "notes.txt"), "ignored")
	writeFile(t, filepath.Join(ws, "review.md"),
		"---\nwhen_to_use: reviewing code\n---\nproject body\n")

	got, err := LoadSkills([]string{home, ws, filepath.Join(t.TempDir(), "absent")}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d skills: %+v", len(got), got)
	}
	// Sorted by name, so the index is stable between runs.
	if got[0].Name != "packaged" || got[1].Name != "review" {
		t.Errorf("got %q, %q", got[0].Name, got[1].Name)
	}
	if got[1].Body != "project body" {
		t.Errorf("the project skill must win, got %q", got[1].Body)
	}
	// The name falls back to the file or directory it came from.
	if got[0].Body != "packaged body" {
		t.Errorf("got %q", got[0].Body)
	}
}

func TestLoadSkillsEnforcesTheSizeCapAndSurfacesErrors(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "big.md"),
		"---\nwhen_to_use: x\n---\n"+strings.Repeat("y", 500))
	if _, err := LoadSkills([]string{dir}, 100); err == nil {
		t.Error("an oversized skill must be rejected")
	}

	broken := t.TempDir()
	writeFile(t, filepath.Join(broken, "b.md"), "---\nname: b\n---\nno index line\n")
	if _, err := LoadSkills([]string{broken}, 0); err == nil {
		t.Error("a skill that cannot be indexed must fail loudly")
	}
}

func TestIndexCarriesOnlyTheLine(t *testing.T) {
	got := Index([]Skill{{Name: "x", WhenToUse: "when x", Body: "a very long body"}})
	if len(got) != 1 || got[0].Name != "x" || got[0].WhenToUse != "when x" {
		t.Fatalf("got %+v", got)
	}
}

// The body is loaded on trigger and nowhere else. Loading every body into the
// prefix is the fastest route to a prompt of tens of thousands of tokens paid
// on every turn.
func TestMatchFiresOnAnExplicitTrigger(t *testing.T) {
	skills := []Skill{{
		Name: "migrations", WhenToUse: "writing a migration",
		Triggers: []string{"migration"}, Body: "b",
	}}
	if got := Match("add a migration for users", skills); len(got) != 1 {
		t.Errorf("got %+v", got)
	}
	if got := Match("rename a variable", skills); len(got) != 0 {
		t.Errorf("an unrelated task must not load the body, got %+v", got)
	}
}

func TestMatchIsDeterministic(t *testing.T) {
	skills := []Skill{
		{Name: "a", WhenToUse: "writing database migration scripts", Body: "x"},
		{Name: "b", Triggers: []string{"deploy"}, WhenToUse: "deploying", Body: "y"},
	}
	task := "writing migration scripts before deploy"
	first := Match(task, skills)
	for i := 0; i < 5; i++ {
		got := Match(task, skills)
		if len(got) != len(first) {
			t.Fatalf("Match must be deterministic: %d then %d", len(first), len(got))
		}
		for j := range got {
			if got[j].Name != first[j].Name {
				t.Fatalf("order changed: %v vs %v", got[j].Name, first[j].Name)
			}
		}
	}
}

// A single common word must not drag a skill into a task that merely mentioned
// it in passing.
func TestMatchWithoutTriggersNeedsTwoSignificantWords(t *testing.T) {
	skills := []Skill{{Name: "a", WhenToUse: "writing database migration scripts", Body: "x"}}
	if got := Match("something about database", skills); len(got) != 0 {
		t.Errorf("one word is not enough, got %+v", got)
	}
	if got := Match("database migration work", skills); len(got) != 1 {
		t.Errorf("two words should fire, got %+v", got)
	}
}

func TestMatchFiresWhenTheSkillIsNamedOutright(t *testing.T) {
	skills := []Skill{{Name: "packaging", WhenToUse: "unrelated words entirely", Body: "x"}}
	if got := Match("use /packaging here", skills); len(got) != 1 {
		t.Errorf("naming a skill must load it, got %+v", got)
	}
}

func TestRenderSkillMarksTheChannel(t *testing.T) {
	got := RenderSkill(Skill{Name: "x", Body: "body"})
	if !strings.Contains(got, `<skill name="x">`) || !strings.Contains(got, "body") {
		t.Errorf("got %q", got)
	}
}

// The stop list was English only, in a product whose users write prompts in
// Portuguese. `quando`, `projeto` and `estiver` counted as significant words,
// two of them were enough, and a task about nothing in particular loaded whole
// skill bodies into the turn.
func TestMatchDoesNotFireOnPortugueseFillerWords(t *testing.T) {
	skills := []Skill{
		{Name: "release", WhenToUse: "quando for cortar uma versão nova do projeto", Body: "x"},
		{Name: "deploy", WhenToUse: "para publicar o projeto quando a versão estiver pronta", Body: "y"},
	}
	for _, task := range []string{
		"quando o projeto estiver pronto me avisa",
		"olha esse projeto e me diz quando a versão sobe",
	} {
		if got := Match(task, skills); len(got) != 0 {
			t.Errorf("%q loaded %s on filler words alone", task, names(got))
		}
	}
}

// The words a skill shares with its neighbours cannot be what selects it: two
// skills that both say "projeto" and "versão" are indistinguishable by them, so
// a match needs at least one word that belongs to this skill and not the others.
func TestMatchNeedsAWordThatBelongsToThisSkillAlone(t *testing.T) {
	skills := []Skill{
		{Name: "release", WhenToUse: "quando for cortar uma versão nova do projeto", Body: "x"},
		{Name: "deploy", WhenToUse: "para publicar o projeto quando a versão estiver pronta", Body: "y"},
	}
	if got := Match("quero cortar uma versão nova", skills); len(got) != 1 || got[0].Name != "release" {
		t.Errorf("the discriminating words are cortar and nova, got %s", names(got))
	}
	if got := Match("quero publicar essa versão", skills); len(got) != 1 || got[0].Name != "deploy" {
		t.Errorf("the discriminating word is publicar, got %s", names(got))
	}
	// One discriminating word is still one word. The two-hit rule is older than
	// this fix and it is the reason a task that merely mentions a subject does
	// not drag a body in; `triggers` is the escape hatch for a skill that wants
	// to fire on a single term.
	if got := Match("como faço para publicar isso", skills); len(got) != 0 {
		t.Errorf("one word is not enough even when it discriminates, got %s", names(got))
	}
}

// Sharing a word must not make two neighbours unreachable: skills in the same
// domain say the same domain word, and each still has its own.
func TestSkillsInTheSameDomainStayReachable(t *testing.T) {
	skills := []Skill{
		{Name: "release-go", WhenToUse: "cortar uma versão nova do módulo golang", Body: "x"},
		{Name: "release-node", WhenToUse: "cortar uma versão nova do pacote typescript", Body: "y"},
	}
	if got := Match("quero cortar uma versão nova do golang", skills); len(got) != 1 || got[0].Name != "release-go" {
		t.Errorf("got %s, want release-go alone", names(got))
	}
}

func names(ss []Skill) string {
	out := ""
	for _, s := range ss {
		out += s.Name + " "
	}
	if out == "" {
		return "(nothing)"
	}
	return out
}
