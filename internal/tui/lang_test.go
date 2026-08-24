package tui

import (
	"github.com/aguinelo/dcode/internal/protocol"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestResolutionOrder(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want Lang
	}{
		{"nothing set", nil, Fallback},
		{"DCODE_LANG wins over LC_ALL and LANG",
			map[string]string{"DCODE_LANG": "en", "LC_ALL": "pt_BR.UTF-8", "LANG": "pt_BR"}, En},
		{"LC_ALL wins over LANG",
			map[string]string{"LC_ALL": "en_US.UTF-8", "LANG": "pt_BR.UTF-8"}, En},
		{"LANG alone",
			map[string]string{"LANG": "pt_BR.UTF-8"}, PtBR},
		{"an encoding and a modifier are cut",
			map[string]string{"LANG": "en_GB.UTF-8@euro"}, En},
		{"case and separator do not matter",
			map[string]string{"DCODE_LANG": "PT-br"}, PtBR},
		// Not an error, and not a silent English fallback either: it lands
		// where everything with no information lands.
		{"a language dcode does not have",
			map[string]string{"DCODE_LANG": "de_DE.UTF-8"}, Fallback},
		{"an empty value is not a choice",
			map[string]string{"DCODE_LANG": "", "LANG": "en"}, En},
		{"C and POSIX are not languages",
			map[string]string{"LANG": "C"}, Fallback},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Resolve(func(k string) string { return c.env[k] })
			if got != c.want {
				t.Errorf("Resolve = %q, want %q", got, c.want)
			}
		})
	}
}

// A missing string must not fall back to English and must not show a raw key.
// Both hide a missing translation — the first forever, the second badly and
// late. This is the check that makes absence impossible to ship.
func TestEveryDeclaredLanguageCoversEveryString(t *testing.T) {
	typ := reflect.TypeOf(Strings{})
	for _, lang := range Languages() {
		v := reflect.ValueOf(Text(lang))
		for i := 0; i < typ.NumField(); i++ {
			if v.Field(i).String() == "" {
				t.Errorf("%s is missing %s", lang, typ.Field(i).Name)
			}
		}
	}
}

func TestAnUndeclaredLanguageGetsTheFallbackNotAnEmptyScreen(t *testing.T) {
	s := Text(Lang("de"))
	if s.HelpKeys == "" {
		t.Fatal("an undeclared language produced a blank interface, which is worse than the wrong language")
	}
	if s != Text(Fallback) {
		t.Error("an undeclared language did not land on the fallback")
	}
}

// Translated text is longer — German runs 30% over English — and some lines
// have fixed columns. Checking here is what stops the overflow being discovered
// in a screenshot.
func TestFixedWidthLinesFitInEveryLanguage(t *testing.T) {
	// The status line is the tightest: it carries the seal beside the mode and
	// neither may be dropped.
	for _, lang := range Languages() {
		s := Text(lang)
		for _, label := range []string{s.VerifiedLabel, s.NotVerifiedLabel, s.UnverifiedLabel} {
			if w := visibleWidth(label); w > 16 {
				t.Errorf("%s: the seal %q is %d cells wide, over the 16 the status line reserves", lang, label, w)
			}
		}
		for _, label := range []string{s.CompletionMet, s.CompletionUnmet, s.CompletionUnchecked, s.CompletionMeasure} {
			if w := visibleWidth(label); w > 22 {
				t.Errorf("%s: the report label %q is %d cells wide, over the 22 the column reserves", lang, label, w)
			}
		}
	}
}

// The structural half of RN-19: the packages that write for the MODEL must not
// be able to reach the catalogue at all.
//
// A convention would break at the first refactor. An import that does not exist
// cannot be used by accident.
func TestModelFacingPackagesCannotReachTheCatalogue(t *testing.T) {
	for _, pkg := range []string{"tools", "policy", "behavior"} {
		dir := "../" + pkg
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, dir, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parsing %s: %v", dir, err)
		}
		for _, p := range pkgs {
			for name, f := range p.Files {
				for _, imp := range f.Imports {
					if strings.Contains(imp.Path.Value, "internal/tui") {
						t.Errorf("%s imports the client package; tool descriptions, tool errors and the doctrine stay English because the model reads them", name)
					}
				}
			}
		}
		_ = ast.Print
	}
}

// The usage block is the earliest text the product shows, and the only one that
// runs before the configuration chain exists.
func TestTheUsageBlockIsTranslatedAndKeepsItsVerbSlot(t *testing.T) {
	for _, lang := range Languages() {
		u := Text(lang).Usage
		if !strings.Contains(u, "%s") {
			t.Errorf("%s: the usage block lost the version slot", lang)
		}
		if strings.Count(u, "%") != strings.Count(u, "%s") {
			t.Errorf("%s: the usage block has a stray %% that Fprintf will mangle", lang)
		}
		// Every subcommand must still be listed, whatever the prose around it.
		for _, cmd := range []string{"dcode serve", "dcode tui", "dcode login", "dcode config", "dcode update"} {
			if !strings.Contains(u, cmd) {
				t.Errorf("%s: usage no longer lists %q", lang, cmd)
			}
		}
		if !strings.Contains(u, "DCODE_LANG") {
			t.Errorf("%s: the language key is not discoverable from the help that documents the others", lang)
		}
	}
}

// Every string in the catalogue is shown somewhere.
//
// A translated string nobody renders is the interface's version of a config key
// nobody reads: it looks like the product speaks the language, and a translator
// spends time on a sentence that will never reach a screen. Five of these had
// accumulated — one of them while the renderer wrote the English by hand three
// lines away.
//
// The other direction is covered by the existing completeness test: every
// language must define every field.
func TestEveryStringInTheCatalogueIsRenderedSomewhere(t *testing.T) {
	fset := token.NewFileSet()
	var fields []string
	f, err := parser.ParseFile(fset, "lang.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	ast.Inspect(f, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || ts.Name.Name != "Strings" {
			return true
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			return false
		}
		for _, fl := range st.Fields.List {
			for _, name := range fl.Names {
				fields = append(fields, name.Name)
			}
		}
		return false
	})
	if len(fields) == 0 {
		t.Fatal("no fields were read from Strings; the guard would pass vacuously")
	}

	// Everything in the package except the catalogue itself. A field mentioned
	// only where it is declared and translated is not rendered anywhere.
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var src string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") ||
			e.Name() == "lang.go" || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		data, err := os.ReadFile(e.Name())
		if err != nil {
			t.Fatal(err)
		}
		src += string(data)
	}

	for _, name := range fields {
		if !regexp.MustCompile(`\.` + name + `\b`).MatchString(src) {
			t.Errorf("Strings.%s is translated in every language and rendered nowhere; "+
				"a sentence that cannot reach a screen is a sentence someone translated for nothing", name)
		}
	}
}

// The queued-input prompt is the one place the renderer wrote English by hand
// while the catalogue three files over already had the word. A pt-BR user
// watched their own input pile up behind a label in someone else's language.
func TestTheQueuedPromptSpeaksTheResolvedLanguage(t *testing.T) {
	m := NewModel("s1", "/w", "m", "workspace-write", PtBR)
	m.State = protocol.SessionStateRunning
	m.Queue = []string{"primeiro", "segundo"}

	out := Render(m, DefaultGeometry(100, 30))
	if !strings.Contains(out, Text(PtBR).Queued) {
		t.Errorf("the prompt does not use the resolved language:\n%s", out)
	}
	if strings.Contains(out, "queued") && Text(PtBR).Queued != "queued" {
		t.Errorf("the English word survived in a pt-BR interface:\n%s", out)
	}

	m.Lang = En
	if out := Render(m, DefaultGeometry(100, 30)); !strings.Contains(out, Text(En).Queued) {
		t.Errorf("English lost its own word:\n%s", out)
	}
}

// /help had a fourth section prepared and never printed. The approval keys were
// discoverable only by triggering a boundary crossing, which is a poor way to
// learn what pressing A does.
func TestHelpDocumentsHowToAnswerAnApproval(t *testing.T) {
	for _, lang := range Languages() {
		txt := Text(lang)
		out := HelpText(userCommands(), lang)
		if !strings.Contains(out, txt.HelpApprovals) {
			t.Errorf("%s: /help has no approvals section:\n%s", lang, out)
			continue
		}
		for _, want := range []string{txt.ApprovalAllowOnce, txt.ApprovalAllowSession, txt.ApprovalDeny} {
			if !strings.Contains(out, want) {
				t.Errorf("%s: /help does not say %q", lang, want)
			}
		}
	}
}

// No word that belongs only to the English catalogue may appear on a
// Portuguese screen.
//
// The coverage guard above asks whether every declared string has a
// translation. It cannot ask whether the RENDERER uses them, so nine literals
// sat in the drawing code untouched by it — including the whole approval modal,
// the one screen that asks whether a boundary may be crossed. Consent given to
// a sentence somebody could not read is not consent.
//
// The forbidden set is DERIVED: every word in the English catalogue that the
// Portuguese one does not also use. It grows on its own with the catalogue, so
// a string added tomorrow is checked tomorrow — the same reasoning as the ASCII
// guard, and for the same reason, which is that a list written by hand is a
// list that stops being edited.
func TestNoEnglishSurvivesAPortugueseScreen(t *testing.T) {
	forbidden := englishOnlyWords()
	if len(forbidden) < 20 {
		t.Fatalf("the derived word set is %d words; something is wrong with it", len(forbidden))
	}

	// The model's own content is Portuguese and carries none of those words, so
	// anything found came from the layout.
	m := NewModel("s", "/w", "modelo", "workspace-write", PtBR)
	m.Entries = []Entry{
		{Kind: KindUser, Summary: "escreve o arquivo"},
		{Kind: KindTool, Tool: "write", Target: "arquivo.go", Summary: "criado"},
	}
	m.Plan = []protocol.PlanItem{
		{ID: 1, Text: "primeiro passo", Status: protocol.PlanDone},
		{ID: 2, Text: "segundo passo", Status: protocol.PlanBlocked, Blocked: "sem rede"},
	}
	m.Rounds, m.MaxRounds, m.InFlight, m.MaxInFlight = 60, 100, 4, 4
	m.Sessions = []SessionChoice{{ID: "a", Title: "uma conversa", Turns: 2, When: m.Now}}
	m.Pending = &protocol.ApprovalRequest{
		ApprovalID: "a1", Tool: "bash", Command: "curl exemplo",
		BoundaryCrossed: "network", Reason: "sai do projeto",
	}

	g := DefaultGeometry(120, 30)
	g.Palette = Palette{}
	g.RailMode = RailShown

	screen := strings.ToLower(Render(m, g))
	for _, w := range forbidden {
		if strings.Contains(screen, w) {
			t.Errorf("%q is English and reached a Portuguese screen:\n%s", w, Render(m, g))
		}
	}
}

// englishOnlyWords is every word the English catalogue uses and the Portuguese
// one does not, long enough not to collide by accident.
func englishOnlyWords() []string {
	pt := map[string]bool{}
	for _, w := range catalogueWords(Text(PtBR)) {
		pt[w] = true
	}
	var out []string
	for _, w := range catalogueWords(Text(En)) {
		if !pt[w] && len(w) >= 5 {
			out = append(out, " "+w)
		}
	}
	return out
}

func catalogueWords(s Strings) []string {
	rv := reflect.ValueOf(s)
	var out []string
	for i := 0; i < rv.NumField(); i++ {
		if rv.Field(i).Kind() != reflect.String {
			continue
		}
		for _, w := range strings.FieldsFunc(strings.ToLower(rv.Field(i).String()),
			func(r rune) bool { return r < 'a' || r > 'z' }) {
			out = append(out, w)
		}
	}
	return out
}
