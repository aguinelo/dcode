package specguard

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Every exported name has a user, or a written reason for not having one.
//
// This is the class the whole sweep kept finding: a constant nothing produces,
// a field nothing reads, a method nothing calls, an error code mapped to a
// status and returned by nobody. Each one reads as a capability. Each one cost
// somebody an afternoon discovering it was scenery — and one of them, the
// verification seal, was printing "verified" over code no command had looked at.
//
// The exemptions are a MAP, not a list, and the value is the reason. A bare
// allowlist would grow silently every time someone found this test
// inconvenient; a reason has to be written and can be argued with.
var exportedWithoutUser = map[string]string{
	// pkg/client exists for consumers of the daemon, not for this repository.
	// Its surface is measured by what a client needs, not by what dcode uses.
	"DeleteSession": "pkg/client is the public API; a consumer deletes sessions",

	// The zero value of an enum, named so a reader knows what zero means. The
	// name is unused precisely because the value is the default — deleting it
	// would leave the zero case anonymous, which is worse than unused.
	"BoundaryNone": "names the zero Boundary",
	"BudgetNone":   "names the zero BudgetBand",
	"SourceLocal":  "names the zero version.Source",

	// Exported so tests can observe something they otherwise could not, and
	// exported rather than unexported because the tests are in other packages.
	"ClearSecrets":   "resets the secret registry between tests in other packages",
	"EnvToKey":       "built so the key/variable bijection can be asserted",
	"Keys":           "enumerates resolved keys for tests that sweep them",
	"Languages":      "enumerates declared languages for the catalogue tests",
	"Subscribers":    "lets a test assert a stream is cleaned up when a client leaves",
	"TransportNames": "lets a test assert the registry reports what it holds",
	"WasRead":        "lets a test observe the read-before-edit record directly",

	// The replay harness. It is production code by package and test
	// infrastructure by purpose: it is how the provider suite runs against
	// recorded frames instead of a network.
	"LoadTranscript":     "replay harness; the suite runs on recorded frames",
	"NewReplayTransport": "replay harness; the suite runs on recorded frames",
	"ParseSSE":           "replay harness; the suite runs on recorded frames",

	// The eval harness, behind its own build tag and run by `make eval`.
	"ContractByID": "eval harness, behind the eval build tag",
}

func TestEveryExportedNameHasAUserOrAWrittenReason(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	decls, uses := scanRepository(t, root)
	if len(decls) < 100 {
		t.Fatalf("only %d exported names were found; the scan is not reading the repository", len(decls))
	}

	var unused []string
	for name, where := range decls {
		if uses[name] > 1 { // one occurrence is the declaration itself
			continue
		}
		if _, excused := exportedWithoutUser[name]; excused {
			continue
		}
		unused = append(unused, name+" ("+where+")")
	}
	sort.Strings(unused)
	for _, u := range unused {
		t.Errorf("%s is exported and nothing outside a test reads it.\n"+
			"Delete it, or add it to exportedWithoutUser with the reason it stays.", u)
	}

	// An exemption for a name that IS used, or that no longer exists, is a note
	// about a world that moved on. Left alone they accumulate until the map is
	// the reason nothing fails rather than a record of decisions.
	for name := range exportedWithoutUser {
		if _, declared := decls[name]; !declared {
			t.Errorf("exportedWithoutUser excuses %q, which is not an exported name here", name)
			continue
		}
		if uses[name] > 1 {
			t.Errorf("exportedWithoutUser excuses %q, which now has a user; remove the excuse", name)
		}
	}
}

// scanRepository collects exported declarations under internal/ and pkg/, and
// counts identifier occurrences across everything that is not an ordinary test.
//
// cmd/ counts as a user: it is the layer the rest exists for. Eval-tagged tests
// count too, because a harness behind a build tag is still someone reading it.
func scanRepository(t *testing.T, root string) (map[string]string, map[string]int) {
	t.Helper()
	decls := map[string]string{}
	uses := map[string]int{}
	fset := token.NewFileSet()

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if !strings.HasPrefix(rel, "internal/") && !strings.HasPrefix(rel, "pkg/") &&
			!strings.HasPrefix(rel, "cmd/") {
			return nil
		}
		// testdata is material, not product code. The Go toolchain ignores it
		// for the same reason, and a scanner that does not will report a type
		// in an eval fixture as dead code — which it is, deliberately: the
		// scenario exists precisely so a model can be asked to change it.
		if strings.Contains(rel, "/testdata/") {
			return nil
		}

		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil // a file that does not parse is the compiler's problem
		}
		test := strings.HasSuffix(rel, "_test.go")
		evalTagged := strings.HasSuffix(rel, "_eval_test.go")
		owned := strings.HasPrefix(rel, "internal/") || strings.HasPrefix(rel, "pkg/")

		if !test && owned {
			collectExported(fset, f, rel, decls)
		}
		if !test || evalTagged {
			ast.Inspect(f, func(n ast.Node) bool {
				if id, ok := n.(*ast.Ident); ok {
					uses[id.Name]++
				}
				return true
			})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return decls, uses
}

func collectExported(fset *token.FileSet, f *ast.File, rel string, into map[string]string) {
	at := func(p token.Pos) string {
		return rel + ":" + itoa(fset.Position(p).Line)
	}
	for _, d := range f.Decls {
		switch n := d.(type) {
		case *ast.FuncDecl:
			if n.Name.IsExported() {
				into[n.Name.Name] = at(n.Pos())
			}
		case *ast.GenDecl:
			for _, sp := range n.Specs {
				switch s := sp.(type) {
				case *ast.TypeSpec:
					if s.Name.IsExported() {
						into[s.Name.Name] = at(s.Pos())
					}
					st, ok := s.Type.(*ast.StructType)
					if !ok {
						continue
					}
					for _, fl := range st.Fields.List {
						for _, nm := range fl.Names {
							if nm.IsExported() {
								into[nm.Name] = at(nm.Pos())
							}
						}
					}
				case *ast.ValueSpec:
					for _, nm := range s.Names {
						if nm.IsExported() {
							into[nm.Name] = at(nm.Pos())
						}
					}
				}
			}
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
