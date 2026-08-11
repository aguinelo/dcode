package config

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Every `.p.spec.md` opens with the same rule: "Use EXATAMENTE os nomes, campos
// e tipos definidos aqui." Several of them named types the code does not have —
// SessionFiles for State, FileState for an unexported fileState — and
// signatures the code does not use, because a later changelog changed the code
// and never came back to the .p.
//
// A spec that cannot be followed literally is worse than one that is vague:
// someone who tries writes code that does not compile, and concludes the spec
// is decorative.
//
// This checks the Go type names a .p declares against the package it names. It
// deliberately does not check signatures — parsing Go out of markdown to
// compare parameter lists would be a second compiler, and a guard nobody trusts
// gets deleted. Names are the half that can be checked cheaply and the half
// that was actually wrong.
func TestTheTypesTheSpecsNameExist(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	// Which packages each spec family describes. Several span more than one,
	// and legitimately: sandbox-policy defines both the decision and the
	// mechanism, and every spec that mentions a tool result reaches the shared
	// protocol types.
	pkgs := map[string][]string{
		"tool-suite":             {"internal/tools", "internal/protocol"},
		"sandbox-policy":         {"internal/policy", "internal/sandbox"},
		"context-engine":         {"internal/contextengine"},
		"behavior-definition":    {"internal/behavior"},
		"agent-loop":             {"internal/loop", "internal/tools", "internal/protocol"},
		"client-server-protocol": {"internal/protocol"},
		"provider-adapter":       {"internal/provider"},
		"configuration":          {"internal/config", "internal/credential"},
	}

	// `type Name struct` / `type Name interface` inside a Go block.
	decl := regexp.MustCompile(`(?m)^type ([A-Z][A-Za-z0-9]*) (struct|interface)`)

	for family, dirs := range pkgs {
		specs, err := filepath.Glob(filepath.Join(root, "docs", "specs", "architecture", family, "*.p.spec.md"))
		if err != nil || len(specs) == 0 {
			t.Errorf("no .p spec found for %s", family)
			continue
		}
		var src string
		for _, d := range dirs {
			src += packageSource(t, filepath.Join(root, d))
		}

		data, err := os.ReadFile(specs[0])
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range decl.FindAllStringSubmatch(string(data), -1) {
			name := m[1]
			if !strings.Contains(src, "type "+name+" ") {
				t.Errorf("%s declares type %s and %s does not have it: "+
					"the spec's own rule is to use exactly these names, so this one cannot be followed",
					family, name, strings.Join(dirs, " or "))
			}
		}
	}
}

func packageSource(t *testing.T, dir string) string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	var b strings.Builder
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			t.Fatal(err)
		}
		b.Write(data)
	}
	if b.Len() == 0 {
		t.Fatalf("no sources in %s; the guard would pass vacuously", dir)
	}
	return b.String()
}
