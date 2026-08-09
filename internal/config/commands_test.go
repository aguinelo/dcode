package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCommandReadsFrontmatterAndBody(t *testing.T) {
	c, err := ParseCommand(`---
name: revisar
description: reviews the current diff against the project conventions
---
Review the current diff against docs/conventions/.
`, "revisar.md")
	if err != nil {
		t.Fatal(err)
	}
	if c.Name != "revisar" {
		t.Errorf("got %q", c.Name)
	}
	if !strings.Contains(c.Description, "conventions") {
		t.Errorf("got %q", c.Description)
	}
	if !strings.HasPrefix(c.Body, "Review the current diff") {
		t.Errorf("got %q", c.Body)
	}
}

func TestParseCommandWorksWithoutFrontmatter(t *testing.T) {
	c, err := ParseCommand("Just do the thing.\n", "thing.md")
	if err != nil {
		t.Fatal(err)
	}
	if c.Name != "" {
		t.Errorf("the name comes from the filename when frontmatter is absent, got %q", c.Name)
	}
	if c.Body != "Just do the thing." {
		t.Errorf("got %q", c.Body)
	}
}

func TestParseCommandRejectsBrokenInput(t *testing.T) {
	for name, body := range map[string]string{
		"unterminated frontmatter": "---\nname: x\n",
		"empty body":               "---\nname: x\n---\n\n",
		"bad frontmatter line":     "---\njust words\n---\nbody\n",
	} {
		if _, err := ParseCommand(body, "c.md"); err == nil {
			t.Errorf("%s: must be rejected", name)
		}
	}
}

func TestParseCommandStripsALeadingSlashFromTheName(t *testing.T) {
	c, err := ParseCommand("---\nname: /review\n---\nbody\n", "c.md")
	if err != nil {
		t.Fatal(err)
	}
	if c.Name != "review" {
		t.Errorf("got %q; the slash is how it is invoked, not part of the name", c.Name)
	}
}

// A command is a prompt and nothing else. Expand doing I/O would be an
// execution path with no approval in front of it.
func TestExpandIsDeterministicAndSideEffectFree(t *testing.T) {
	c := Command{Name: "x", Body: "Look at $ARGUMENTS and report."}
	first, err := Expand(c, "main.go")
	if err != nil {
		t.Fatal(err)
	}
	second, _ := Expand(c, "main.go")
	if first != second {
		t.Errorf("Expand must be deterministic:\n%q\n%q", first, second)
	}
	if first != "Look at main.go and report." {
		t.Errorf("got %q", first)
	}
}

// The user typed those words meaning them to reach the model; a command author
// who never used the placeholder did not decide otherwise.
func TestExpandAppendsArgumentsWithoutAPlaceholder(t *testing.T) {
	got, err := Expand(Command{Name: "x", Body: "Review the diff."}, "focus on tests")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "Review the diff.") || !strings.Contains(got, "focus on tests") {
		t.Errorf("got %q", got)
	}
}

func TestExpandWithoutArguments(t *testing.T) {
	got, err := Expand(Command{Name: "x", Body: "Review the diff."}, "  ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Review the diff." {
		t.Errorf("got %q", got)
	}
	if _, err := Expand(Command{Name: "x"}, ""); err == nil {
		t.Error("an empty body has nothing to send")
	}
}

func TestSplitInvocation(t *testing.T) {
	for _, tc := range []struct {
		in         string
		name, args string
		ok         bool
	}{
		{"/review", "review", "", true},
		{"  /review the diff  ", "review", "the diff", true},
		{"/review\tmain.go", "review", "main.go", true},
		{"review", "", "", false},
		{"/", "", "", false},
		{"", "", "", false},
		{"not /a command", "", "", false},
	} {
		name, args, ok := SplitInvocation(tc.in)
		if ok != tc.ok || name != tc.name || args != tc.args {
			t.Errorf("%q: got (%q, %q, %v)", tc.in, name, args, ok)
		}
	}
}

func TestDiscoverCommandsLetsTheProjectWinAndRecordsIt(t *testing.T) {
	home, ws := t.TempDir(), t.TempDir()
	write(t, filepath.Join(home, CommandsDirName, "review.md"), "---\nname: review\n---\nuser body\n")
	write(t, filepath.Join(home, CommandsDirName, "onlyuser.md"), "user only\n")
	write(t, filepath.Join(ws, ".dcode", CommandsDirName, "review.md"), "---\nname: review\n---\nproject body\n")

	set, err := DiscoverCommands(Roots{Config: home}, ws, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := set.Commands["review"].Body; got != "project body" {
		t.Errorf("the project command must win, got %q", got)
	}
	if set.Commands["review"].Source != SourceProject {
		t.Errorf("got %v", set.Commands["review"].Source)
	}
	if len(set.Collisions) != 1 {
		t.Fatalf("the shadowing must be recorded, got %v", set.Collisions)
	}
	if !strings.Contains(set.Collisions[0], "review") {
		t.Errorf("got %q", set.Collisions[0])
	}
	if got := set.Names(); len(got) != 2 || got[0] != "onlyuser" || got[1] != "review" {
		t.Errorf("names must be stable and sorted, got %v", got)
	}
	// The filename is the name when frontmatter does not give one.
	if _, ok := set.Commands["onlyuser"]; !ok {
		t.Error("a command without frontmatter takes its name from the file")
	}
}

func TestDiscoverCommandsIgnoresNonMarkdownAndMissingDirs(t *testing.T) {
	home := t.TempDir()
	write(t, filepath.Join(home, CommandsDirName, "notes.txt"), "ignored")
	write(t, filepath.Join(home, CommandsDirName, "sub", "x.md"), "ignored too")

	set, err := DiscoverCommands(Roots{Config: home}, filepath.Join(t.TempDir(), "no-workspace"), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Commands) != 0 {
		t.Errorf("got %v", set.Names())
	}
}

func TestDiscoverCommandsEnforcesTheSizeCap(t *testing.T) {
	home := t.TempDir()
	write(t, filepath.Join(home, CommandsDirName, "big.md"), strings.Repeat("x", 200))
	if _, err := DiscoverCommands(Roots{Config: home}, t.TempDir(), 100); err == nil {
		t.Error("an oversized command must be rejected")
	}
}

func TestDiscoverCommandsSurfacesAParseError(t *testing.T) {
	home := t.TempDir()
	write(t, filepath.Join(home, CommandsDirName, "broken.md"), "---\nname: x\n")
	if _, err := DiscoverCommands(Roots{Config: home}, t.TempDir(), 0); err == nil {
		t.Error("a broken command file must fail rather than disappear")
	}
}

// ---------- out of chain ----------

func TestOutOfChainFindsAnInstructionTheSessionNeverLoaded(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "vendor")
	write(t, filepath.Join(nested, "AGENTS.md"), "vendor rules")

	path, found := OutOfChain(nested, []string{filepath.Join(dir, "AGENTS.md")})
	if !found {
		t.Fatal("an instruction file in a touched directory must be found")
	}
	if path != filepath.Join(nested, "AGENTS.md") {
		t.Errorf("got %q", path)
	}
}

// Reported once and only once: a file already in the frozen chain was ranked
// deliberately, and re-reporting it would undo that ranking.
func TestOutOfChainIgnoresWhatIsAlreadyInTheChain(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "DCODE.md")
	write(t, path, "rules")

	if _, found := OutOfChain(dir, []string{path}); found {
		t.Error("a file already in the chain must not be reported again")
	}
}

// DCODE.md is the higher-precedence name, so it is the one reported when both
// are present.
func TestOutOfChainPrefersTheMoreSpecificName(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "AGENTS.md"), "shared")
	write(t, filepath.Join(dir, "DCODE.md"), "specific")

	path, found := OutOfChain(dir, nil)
	if !found || filepath.Base(path) != "DCODE.md" {
		t.Errorf("got %q, %v", path, found)
	}
}

func TestOutOfChainAcceptsAFileAndLooksAtItsDirectory(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "AGENTS.md"), "rules")
	target := filepath.Join(dir, "main.go")
	if err := os.WriteFile(target, []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, found := OutOfChain(target, nil); !found {
		t.Error("a touched file must resolve to its directory")
	}
}

func TestOutOfChainFindsNothingInAnEmptyDirectory(t *testing.T) {
	if _, found := OutOfChain(t.TempDir(), nil); found {
		t.Error("nothing to find")
	}
}
