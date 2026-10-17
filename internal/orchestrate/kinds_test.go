package orchestrate

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// The kinds the log accepts are the list in .p §9.1, read from the spec rather
// than copied: a kind added to one and not the other is refused by Record or
// promised by a spec nothing writes.
func TestTheKindsAreTheSpecsList(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "specs", "architecture", "orchestration",
		"202610132000-orchestration.p.spec.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	start := strings.Index(body, "`kind`: ")
	if start < 0 {
		t.Fatal("the spec has no `kind`: list; this guard reads nothing")
	}
	para := body[start+len("`kind`: "):]
	if end := strings.Index(para, "\n\n"); end >= 0 {
		para = para[:end]
	}
	var spec []string
	for _, m := range regexp.MustCompile("`([a-z_.]+)`").FindAllStringSubmatch(para, -1) {
		spec = append(spec, m[1])
	}
	if len(spec) < 20 {
		t.Fatalf("only %d kinds read from the spec; the pattern has drifted", len(spec))
	}

	var code []string
	for k := range kinds {
		code = append(code, k)
	}
	slices.Sort(spec)
	slices.Sort(code)
	if !slices.Equal(spec, code) {
		t.Errorf("the log accepts\n  %v\nand the spec lists\n  %v", code, spec)
	}
}
