package tools

import (
	"encoding/json"
	"strings"
	"testing"
)

const goSource = `package p

// Parse turns text into a thing. ParseTOML is a different function.
func Parse(s string) int { return 0 }

func ParseTOML(s string) int { return Parse(s) }

func parseMode(s string) int { return 0 }

func caller() int {
	n := Parse("x")
	return n + ParseTOML("y") + parseMode("z")
}
`

func symbolSetup(t *testing.T) (*State, string) {
	t.Helper()
	s, ws := setup(t)
	writeFileT(t, ws, "p.go", goSource)
	return s, ws
}

// The invariant that separates symbol from grep on a bare name.
func TestSymbolMatchesOnBoundaryNotOnLetters(t *testing.T) {
	s, _ := symbolSetup(t)
	res := run(t, Symbol{}, s, SymbolInput{Name: "Parse"})
	if res.IsError {
		t.Fatal(res.Output)
	}
	for _, unwanted := range []string{"ParseTOML", "parseMode"} {
		// The lines that carry ONLY the unwanted name must not appear. Lines
		// where Parse also appears legitimately are a different matter.
		for _, line := range strings.Split(res.Output, "\n") {
			if strings.Contains(line, unwanted) && !boundaryRegexp("Parse").MatchString(line) {
				t.Errorf("matched %q on letters alone:\n%s", unwanted, line)
			}
		}
	}
	if !strings.Contains(res.Output, "func Parse(") {
		t.Error("the declaration itself is missing")
	}
}

// Name is data, never pattern. This is what stops symbol becoming grep with a
// different name.
func TestNameIsEscapedAndNeverTreatedAsARegexp(t *testing.T) {
	s, ws := symbolSetup(t)
	writeFileT(t, ws, "dots.go", "package p\n\nvar axb = 1\nvar aQb = 2\n")

	res := run(t, Symbol{}, s, SymbolInput{Name: "a.b"})
	if strings.Contains(res.Output, "axb") || strings.Contains(res.Output, "aQb") {
		t.Errorf("`a.b` was compiled as a regexp and matched `axb`:\n%s", res.Output)
	}
}

func TestKindDefFindsTheDeclarationAndNotTheCall(t *testing.T) {
	s, _ := symbolSetup(t)
	res := run(t, Symbol{}, s, SymbolInput{Name: "Parse", Kind: KindDef})
	if !strings.Contains(res.Output, "func Parse(s string) int") {
		t.Fatalf("the declaration is missing:\n%s", res.Output)
	}
	if strings.Contains(res.Output, `n := Parse("x")`) {
		t.Errorf("a call was reported as a declaration:\n%s", res.Output)
	}
}

func TestKindRefFindsTheCallAndNotTheDeclaration(t *testing.T) {
	s, _ := symbolSetup(t)
	res := run(t, Symbol{}, s, SymbolInput{Name: "Parse", Kind: KindRef})
	if strings.Contains(res.Output, "func Parse(s string) int") {
		t.Errorf("the declaration was reported as a use:\n%s", res.Output)
	}
	if !strings.Contains(res.Output, `Parse("x")`) {
		t.Errorf("the call is missing:\n%s", res.Output)
	}
}

// A false negative that looks like a complete answer is the worst outcome this
// tool can produce, so it never produces one silently.
func TestEveryResultDeclaresItsOwnLimit(t *testing.T) {
	s, _ := symbolSetup(t)
	for _, in := range []SymbolInput{
		{Name: "Parse"},
		{Name: "Parse", Kind: KindDef},
		{Name: "Parse", Kind: KindRef},
		{Name: "NoSuchSymbolAnywhere"},
	} {
		res := run(t, Symbol{}, s, in)
		if !strings.Contains(res.Output, SymbolLimit) {
			t.Errorf("result for %+v does not declare its limit:\n%s", in, res.Output)
		}
	}
}

// Refusing the language would be worse than answering with a stated limit.
func TestUnknownExtensionAnswersAndSaysTheKindIsUnknown(t *testing.T) {
	s, ws := setup(t)
	writeFileT(t, ws, "thing.zig", "pub fn Parse(s: []u8) void {}\nconst x = Parse;\n")

	res := run(t, Symbol{}, s, SymbolInput{Name: "Parse"})
	if res.IsError {
		t.Fatalf("an unknown extension became an error: %s", res.Output)
	}
	if !strings.Contains(res.Output, "kind unknown") {
		t.Errorf("the result does not say the kind could not be determined:\n%s", res.Output)
	}
	if !strings.Contains(res.Output, ".zig") {
		t.Errorf("the result does not name the extension it could not classify:\n%s", res.Output)
	}
	if !strings.Contains(res.Output, "Parse") {
		t.Errorf("nothing was found in a file that plainly contains the symbol:\n%s", res.Output)
	}
}

func TestDeclarationPatternsPerLanguage(t *testing.T) {
	cases := []struct {
		ext     string
		def     string // a line that IS a declaration
		use     string // a line that is NOT
		symbol  string
		wantDef bool
	}{
		{".go", "func Parse(s string) int {", `x := Parse(s)`, "Parse", true},
		{".go", "type Parse struct {", `var p Parse`, "Parse", true},
		{".py", "def parse(self):", "    return parse(x)", "parse", true},
		{".rs", "fn parse(s: &str) -> u8 {", "    let n = parse(s);", "parse", true},
		{".ts", "function parse(s: string) {", "  const n = parse(s);", "parse", true},
		{".ts", "class Parser {", "  new Parser();", "Parser", true},
		{".rb", "def parse(s)", "  parse(s)", "parse", true},
		{".java", "class Parser {", "    new Parser();", "Parser", true},
	}
	for _, c := range cases {
		t.Run(c.ext+"/"+c.symbol, func(t *testing.T) {
			re, known := declRegexp(c.ext, c.symbol)
			if !known {
				t.Fatalf("%s is in the table but declRegexp reported it unknown", c.ext)
			}
			if !re.MatchString(c.def) {
				t.Errorf("declaration not recognised: %q", c.def)
			}
			if re.MatchString(c.use) {
				t.Errorf("a use was classified as a declaration: %q", c.use)
			}
		})
	}
}

func TestOrderingIsStableBetweenRuns(t *testing.T) {
	s, ws := symbolSetup(t)
	writeFileT(t, ws, "a/one.go", goSource)
	writeFileT(t, ws, "b/two.go", goSource)

	first := run(t, Symbol{}, s, SymbolInput{Name: "Parse"}).Output
	for i := 0; i < 3; i++ {
		if got := run(t, Symbol{}, s, SymbolInput{Name: "Parse"}).Output; got != first {
			t.Fatal("symbol output is not stable between runs")
		}
	}
}

func TestTruncationIsDeclared(t *testing.T) {
	s, ws := setup(t)
	s.Limits.SymbolMaxMatches = 3
	var b strings.Builder
	b.WriteString("package p\n")
	for i := 0; i < 50; i++ {
		b.WriteString("var _ = Parse\n")
	}
	writeFileT(t, ws, "many.go", b.String())

	res := run(t, Symbol{}, s, SymbolInput{Name: "Parse"})
	if !res.Truncated {
		t.Fatal("result was cut and does not say so")
	}
	if !strings.Contains(res.Output, "stopped at 3") {
		t.Errorf("the truncation notice does not say where it stopped:\n%s", res.Output)
	}
}

func TestAnEmptyNameIsRefusedRatherThanMatchingEverything(t *testing.T) {
	s, _ := symbolSetup(t)
	res := run(t, Symbol{}, s, SymbolInput{Name: "   "})
	if !res.IsError {
		t.Fatal("an empty name was accepted; it would match every line in the workspace")
	}
}

func TestAnUnknownKindIsRefused(t *testing.T) {
	s, _ := symbolSetup(t)
	res := run(t, Symbol{}, s, SymbolInput{Name: "Parse", Kind: "definition"})
	if !res.IsError {
		t.Fatal("an unknown kind was accepted silently")
	}
	if !strings.Contains(res.Output, "def") {
		t.Errorf("the error does not say what the valid kinds are:\n%s", res.Output)
	}
}

func TestSymbolDescribesItselfAndDeclaresARead(t *testing.T) {
	var sy Symbol
	if sy.Name() != "symbol" {
		t.Error("wrong name")
	}
	d := sy.Description()
	// The description is the high-precision layer, read at the moment of the
	// decision — so it has to say the thing that separates this from grep.
	for _, want := range []string{"never a regular expression", "ParseTOML", "def", "ref"} {
		if !strings.Contains(d, want) {
			t.Errorf("the description does not carry %q:\n%s", want, d)
		}
	}
	if len(sy.Schema()) == 0 {
		t.Error("no schema")
	}

	in := mustJSONSymbol(t, SymbolInput{Name: "Parse", Path: "sub"})
	req, err := sy.Declare(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(req.Paths) != 1 || req.Paths[0].Write {
		t.Fatalf("declared %+v, want a single read", req.Paths)
	}
	bare, _ := sy.Declare(mustJSONSymbol(t, SymbolInput{Name: "Parse"}))
	if bare.Paths[0].Path != "." {
		t.Errorf("path = %q, want the workspace root", bare.Paths[0].Path)
	}
}

func TestKindNounNamesEachKind(t *testing.T) {
	for kind, want := range map[string]string{
		KindDef: "declaration", KindRef: "use", KindAny: "occurrence",
	} {
		if got := kindNoun(kind); got != want {
			t.Errorf("kindNoun(%q) = %q, want %q", kind, got, want)
		}
	}
}

func mustJSONSymbol(t *testing.T, in SymbolInput) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
