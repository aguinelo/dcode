package provider

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/specguard"
)

var providerInvariants = map[string]string{
	"termina em exatamente um":       "TestStreamAlwaysEndsWithExactlyOneTerminal",
	"Cancelar `ctx` fecha o canal":   "TestCancelClosesChannelWithCanceled",
	"cruza a fronteira do pacote":    "TestNoProviderSpecificTypeCrossesTheBoundary",
	"Nenhuma credencial aparece":     "TestCredentialsNeverAppearInErrorMessages",
	"nunca chega ao consumidor":      "TestUndeclaredToolNeverReachesTheLoop",
	"uso e conteúdo juntos":          "TestAFrameCarryingUsageStillYieldsItsToolCall",
	"sem reemitir a chamada":         "TestUsageOnItsOwnFrameTerminatesWithoutRepeatingTheCall",
	"roda com a rede desligada":      "TestTheSuiteCannotReachTheNetwork",
	"herdar a codificação não herda": "TestGeminiEncodesOpenAIAndRefusesTheOther",
	"conferida contra as medições":   "TestEveryUnmeasuredFamilySaysSo",
	"apenas em `ErrClassRateLimit":   "TestOnlyARateLimitCarriesARetryAfter",
	"não se sobrepõem entre":         "TestOverlappingModelPrefixesAreRejected",
	"nomeando os compatíveis":        "TestTransportOverrideIsHonouredAndValidated",
	"corpos distintos e ambos":       "TestOneFamilyEncodesForBothTransports",
	"devolve o default da família":   "TestLimitsComeFromTheFamily",
	"partidos entre frames":          "TestToolCallArgumentsAreAssembledAcrossFrames",
	"Duas tool calls paralelas":      "TestParallelToolCallsAreKeptApart",
	"repetido não emite":             "TestUsageSurvivesARepeatedFinishReason",
	"vira erro `tool_schema`":        "TestATruncatedToolCallIsAnErrorRatherThanAnEmptyCall",
	"nem no histórico (RN-10)":       "TestReasoningNeverBecomesAnswerText",
	"marcador de raciocínio e esp":   "TestAFrameOfPureFramingProducesNothing",
}

func TestEveryInvariantHasATest(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	// The unmeasured-family rule is asserted in internal/evals, and it has to
	// be: it compares the warning list against evals.Measured, and this package
	// cannot import that one — evals imports provider, so the dependency only
	// runs the other way. Asserting it here would mean a second copy of the
	// measurements, which is the thing the rule exists to prevent.
	findings, err := specguard.Check(root, "provider-adapter",
		[]string{".", filepath.Join("..", "evals")}, providerInvariants)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Errorf("provider-adapter: %s", f)
	}
}

// RN-2: the loop must not learn which provider it is talking to.
//
// A leak here does not look like a leak. It looks like one convenient field on
// an exported struct, and then the loop branches on it, and then the abstraction
// that lets a second provider exist is gone — discovered when adding the second
// provider, which is the most expensive moment to discover it.
//
// The check is over the EXPORTED surface, because that is the boundary. Inside,
// a family is allowed to know exactly what it is.
func TestNoProviderSpecificTypeCrossesTheBoundary(t *testing.T) {
	// Brand names, not import paths: there are no vendor SDKs here, and the way
	// this would actually go wrong is a dialect-shaped type given a public name.
	brands := []string{"anthropic", "openai", "minimax", "claude", "gpt", "sse"}

	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !fn.Name.IsExported() || fn.Recv != nil && !receiverIsExported(fn) {
				continue
			}
			if !fn.Name.IsExported() {
				continue
			}
			checked++
			for _, ident := range signatureTypes(fn.Type) {
				low := strings.ToLower(ident)
				for _, b := range brands {
					if strings.Contains(low, b) {
						t.Errorf("%s: exported %s carries %q in its signature; "+
							"the loop must not learn which provider it is talking to",
							name, fn.Name.Name, ident)
					}
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no exported function was inspected; the guard would pass vacuously")
	}
}

func receiverIsExported(fn *ast.FuncDecl) bool {
	for _, f := range fn.Recv.List {
		for _, n := range typeNames(f.Type) {
			if ast.IsExported(n) {
				return true
			}
		}
	}
	return false
}

func signatureTypes(ft *ast.FuncType) []string {
	var out []string
	collect := func(fl *ast.FieldList) {
		if fl == nil {
			return
		}
		for _, f := range fl.List {
			out = append(out, typeNames(f.Type)...)
		}
	}
	collect(ft.Params)
	collect(ft.Results)
	return out
}

func typeNames(e ast.Expr) []string {
	switch t := e.(type) {
	case *ast.Ident:
		return []string{t.Name}
	case *ast.StarExpr:
		return typeNames(t.X)
	case *ast.ArrayType:
		return typeNames(t.Elt)
	case *ast.ChanType:
		return typeNames(t.Value)
	case *ast.Ellipsis:
		return typeNames(t.Elt)
	case *ast.MapType:
		return append(typeNames(t.Key), typeNames(t.Value)...)
	case *ast.SelectorExpr:
		return append(typeNames(t.X), t.Sel.Name)
	}
	return nil
}

// RN-4: the standard suite runs with the network off.
//
// It held de facto — every test drives recorded frames — and de facto is not a
// guarantee. The first test that reaches for a real endpoint makes the suite
// slow, flaky and dependent on someone else's uptime, and it does so quietly:
// it passes on the machine that wrote it.
//
// Structural rather than a runtime trap, because the honest version of a
// runtime trap is a dialer that panics, and installing one would mean the
// package under test can no longer be built without it.
func TestTheSuiteCannotReachTheNetwork(t *testing.T) {
	dialing := map[string]bool{
		"net": true, "net/http": true, "net/url": false,
		"crypto/tls": true, "net/http/httptest": true,
	}
	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		f, err := parser.ParseFile(fset, e.Name(), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		seen++
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if dialing[path] {
				t.Errorf("%s imports %q; the transport is injected and the suite runs "+
					"against recorded frames, so nothing here has cause to reach a socket",
					e.Name(), path)
			}
		}
	}
	if seen == 0 {
		t.Fatal("no file was inspected; the guard would pass vacuously")
	}
}
