package contextengine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// "Tools is frozen at session creation" is a comment on an exported slice. The
// spec goes further and says the session exposes no path to change it, which is
// not true of a public field — anyone can append to it.
//
// What IS true, and worth keeping true, is that nothing does. The tool
// definitions sit in the cached prefix; append one mid-session and every
// subsequent turn pays full price for the whole prompt, silently, with the only
// symptom being a bill.
//
// So the guarantee is a guard rather than a type: assignment to a session's
// Tools outside a composite literal is what this refuses. A composite literal IS
// creation; an assignment afterwards is the thing being forbidden.
func TestNothingAssignsToASessionsToolsAfterItIsBuilt(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	// `x.Tools = ...`, including `append`. A composite literal writes
	// `Tools:` with a colon and is not matched.
	assign := regexp.MustCompile(`\.Tools\s*(=[^=]|\+=)`)

	checked := 0
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if strings.HasPrefix(rel, "bin/") || strings.Contains(rel, "/testdata/") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		checked++
		for i, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			// A comment describing the rule is not a violation of it.
			if strings.HasPrefix(trimmed, "//") || !assign.MatchString(line) {
				continue
			}
			// The provider's own request body has a Tools field of its own, and
			// so does the loop's config. Only a session is frozen.
			if strings.Contains(line, "body.Tools") || strings.Contains(line, "cfg.Tools") {
				continue
			}
			t.Errorf("%s:%d assigns to Tools after construction:\n  %s\n"+
				"tool definitions live in the cached prefix; changing them mid-session "+
				"makes every later turn pay for the whole prompt again", rel, i+1, strings.TrimSpace(line))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked == 0 {
		t.Fatal("no file was scanned; the guard would pass vacuously")
	}
}

// RN-2: the occupancy fraction and its band are told to the model as an
// APPENDED reminder, never as part of what Assemble builds.
//
// A number in the assembled output is not a cosmetic problem. Two runs of the
// same session would differ by however full the context happened to be, so the
// history stops being reproducible — and the prefix, which is the whole reason
// this package is pure, would change on every turn.
func TestNoFractionOrBandReachesTheAssembledOutput(t *testing.T) {
	s := Session{
		Instructions: "You are dcode.",
		Tools:        []ToolDef{{Name: "read", Description: "reads"}},
		History: []Message{
			{Role: RoleUser, Text: "go"},
			{Role: RoleAssistant, Text: "done"},
		},
	}
	out, err := Assemble(s)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)

	// The bands themselves, in every shape they could be written.
	for _, band := range DefaultBands {
		for _, form := range []string{
			strconvFloat(band), strconvFloat(band * 100), percent(band),
		} {
			if strings.Contains(text, form) {
				t.Errorf("the assembled output carries %q, which is a band threshold", form)
			}
		}
	}
	// And the band names, which would leak the same fact in words.
	for _, name := range []string{"Band60", "Band80", "Band92", "budget", "% of the context"} {
		if strings.Contains(strings.ToLower(text), strings.ToLower(name)) {
			t.Errorf("the assembled output mentions %q; occupancy is an appended reminder", name)
		}
	}
}

func strconvFloat(f float64) string {
	b, _ := json.Marshal(f)
	return string(b)
}

func percent(f float64) string {
	b, _ := json.Marshal(int(f * 100))
	return string(b) + "%"
}
