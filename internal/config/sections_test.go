package config

import (
	"strings"
	"testing"
)

func TestParseSectionsKeepsArbitrarySectionNames(t *testing.T) {
	in := `
protected = ["**/*_test.go"]

[tests]
command = "make test"

[lint]
command = "make lint"
exit_code = 0
`
	got, err := ParseSections(in, "done.toml")
	if err != nil {
		t.Fatal(err)
	}
	if got.Values["tests"]["command"] != "make test" {
		t.Errorf("tests.command = %q", got.Values["tests"]["command"])
	}
	if got.Values["lint"]["exit_code"] != "0" {
		t.Errorf("lint.exit_code = %q", got.Values["lint"]["exit_code"])
	}
	if got.Values[""]["protected"] == "" {
		t.Error("a key before any section was dropped")
	}
}

// Order is kept because criteria become an ordered list, and a list that
// reshuffles between runs is one nobody can diff.
func TestSectionOrderIsPreserved(t *testing.T) {
	in := "[zebra]\ncommand = \"z\"\n\n[aardvark]\ncommand = \"a\"\n"
	for i := 0; i < 5; i++ {
		got, err := ParseSections(in, "done.toml")
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Order) != 2 || got.Order[0] != "zebra" || got.Order[1] != "aardvark" {
			t.Fatalf("order = %v, want the order written", got.Order)
		}
	}
}

// The schema check is relaxed here; the credential rule is not.
func TestASecretIsStillRefusedInAnUnknownSection(t *testing.T) {
	for _, in := range []string{
		"[deploy]\napi_key = \"sk-live-123\"\n",
		"[whatever]\ntoken = \"abc\"\n",
		"password = \"hunter2\"\n",
	} {
		if _, err := ParseSections(in, "done.toml"); err == nil {
			t.Errorf("a credential was accepted:\n%s", in)
		} else if !strings.Contains(strings.ToLower(err.Error()), "credential") &&
			!strings.Contains(strings.ToLower(err.Error()), "secret") {
			t.Errorf("the error does not say what was wrong: %v", err)
		}
	}
}

func TestMalformedInputIsRefusedWithAPosition(t *testing.T) {
	for _, in := range []string{
		"[unterminated\n",
		"[]\n",
		"[tests]\nnot an assignment\n",
		"[tests]\n= 3\n",
	} {
		_, err := ParseSections(in, "done.toml")
		if err == nil {
			t.Errorf("accepted malformed input:\n%s", in)
			continue
		}
		if !strings.Contains(err.Error(), "done.toml:") {
			t.Errorf("error does not carry a position: %v", err)
		}
	}
}
