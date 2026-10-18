package orchestrate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// Spec: orchestration .p §10 — a child is named by a short topic title: the
// item's own when it fits, else one derived from it, or from the task's first
// line when it has none, cut at a word, always the same for the same input.
func TestAChildTitleIsShortAndDerivedTheSameWayEachTime(t *testing.T) {
	long := "Crie no pacote `listas` (Go, módulo github.com/x/listas) a função Unicos que remove duplicados"
	for _, c := range []struct {
		name, title, task, want string
		derived                 bool
	}{
		{"fits", "listas: Unicos", "whatever", "listas: Unicos", false},
		{"trimmed", "  listas: Unicos \n", "whatever", "listas: Unicos", false},
		{"long title cut at a word", long, "whatever", "Crie no pacote `listas` (Go, módulo…", true},
		{"missing title from the task", "", long + "\nsecond line", "Crie no pacote `listas` (Go, módulo…", true},
		{"only the first line", "listas: Unicos\nand more", "whatever", "listas: Unicos", true},
		{"no space to cut at", strings.Repeat("x", 60), "t", strings.Repeat("x", TitleLimit-1) + "…", true},
	} {
		got, derived := ShortTitle(c.title, c.task)
		if got != c.want || derived != c.derived {
			t.Errorf("%s: ShortTitle = %q, %v; want %q, %v", c.name, got, derived, c.want, c.derived)
		}
		if n := utf8.RuneCountInString(got); n > TitleLimit {
			t.Errorf("%s: %q has %d characters, over %d", c.name, got, n, TitleLimit)
		}
		if again, _ := ShortTitle(c.title, c.task); again != got {
			t.Errorf("%s: not deterministic: %q then %q", c.name, got, again)
		}
	}
}

// Spec: orchestration .p §10 — a long or missing title does not refuse the
// batch: the child, its session and its branch take the short title, and the
// log keeps the one the orchestrator asked for.
func TestALongOrMissingTitleGivesTheChildAShortOneAndTheLogSaysSo(t *testing.T) {
	p := openWith(t, nil)
	long := Item{Title: "Crie no pacote `listas` (Go, módulo github.com/x/listas) a função Unicos",
		Task: "do it", Owns: []string{"listas"}, Why: "its own theme"}
	none := Item{Task: "Write the docs of the listas package\nin full", Owns: []string{"docs"}, Why: "its own theme"}
	kids, err := p.o.Delegate([]Item{long, none})
	if err != nil {
		t.Fatal(err)
	}
	if kids[0].Title != "Crie no pacote `listas` (Go, módulo…" || kids[1].Title != "Write the docs of the listas package" {
		t.Fatalf("titles = %q, %q", kids[0].Title, kids[1].Title)
	}
	if p.sp.children[0].Title != kids[0].Title || p.sp.children[1].Title != kids[1].Title {
		t.Errorf("the sessions were asked for %q and %q", p.sp.children[0].Title, p.sp.children[1].Title)
	}
	data, err := os.ReadFile(filepath.Join(p.dir, p.o.Project().ID, decisionsFile))
	if err != nil {
		t.Fatal(err)
	}
	asked := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var d Decision
		if err := json.Unmarshal([]byte(line), &d); err != nil {
			t.Fatal(err)
		}
		if d.Kind != "dispatch" {
			continue
		}
		var got struct {
			Title string `json:"title"`
			Asked string `json:"asked_title"`
		}
		if err := json.Unmarshal(d.Data, &got); err != nil {
			t.Fatal(err)
		}
		asked[got.Title] = got.Asked
	}
	if asked[kids[0].Title] != long.Title {
		t.Errorf("the log does not keep the asked title: %q", asked)
	}
	if v, ok := asked[kids[1].Title]; !ok || v != "" {
		t.Errorf("a missing title is logged as asked %q", v)
	}
}
