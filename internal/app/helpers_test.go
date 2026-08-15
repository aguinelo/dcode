package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/config"
)

// The number a definition of done was written with reaches the loop through
// this. A value a person typed is not a value a program wrote, so anything that
// is not a plain number reads as zero rather than as an error that stops a
// session over a typo in a setting.
func TestAnUnreadableNumberReadsAsZeroRatherThanStoppingTheSession(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want int
	}{
		{"12", 12},
		{"  7  ", 7},
		{"-3", -3},
		{"", 0},
		{"12s", 0},
		{"twelve", 0},
		{"1.5", 0},
		{"-", 0},
	} {
		if got := atoi(tc.in); got != tc.want {
			t.Errorf("atoi(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

// The definition of done lives in the workspace unless somebody named another
// file. Naming one has to win, or a person pointing at a shared definition gets
// the workspace's silently instead.
func TestTheDefinitionOfDoneComesFromTheWorkspaceUnlessNamed(t *testing.T) {
	if got := doneFilePath("", "/w"); got != filepath.Join("/w", ".dcode", DoneFileName) {
		t.Errorf("got %q, want the workspace default", got)
	}
	if got := doneFilePath("/elsewhere/done.toml", "/w"); got != "/elsewhere/done.toml" {
		t.Errorf("got %q, want the named file", got)
	}
}

// A locked layer is the normal case's absence: a single person on their own
// laptop has none. Naming one wins over looking for the default, and a default
// that is not there produces no path rather than one that cannot be read.
func TestTheLockedLayerIsFoundWhereItIsNamedOrNotAtAll(t *testing.T) {
	dir := t.TempDir()
	roots := config.Roots{Config: dir}

	if got := requirementsPath(func(string) string { return "" }, roots); got != "" {
		t.Errorf("got %q with no requirements file present, want none", got)
	}

	named := filepath.Join(dir, "elsewhere.toml")
	env := func(k string) string {
		if k == "DCODE_REQUIREMENTS_FILE" {
			return "  " + named + "  "
		}
		return ""
	}
	// Named but absent is still the answer: saying the file is not there is the
	// administrator's problem to see, not something to silently replace.
	if got := requirementsPath(env, roots); got != named {
		t.Errorf("got %q, want the named path", got)
	}

	def := filepath.Join(dir, config.RequirementsFileName)
	if err := os.WriteFile(def, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := requirementsPath(func(string) string { return "" }, roots); got != def {
		t.Errorf("got %q, want the default beside the configuration", got)
	}
}

// Output pasted into a notice is capped and says how much it dropped. A wall of
// text where a summary belongs is how the one line that mattered stops being
// read, and a silent cut reads as "that was all of it".
func TestQuotedOutputIsCappedAndSaysWhatItDropped(t *testing.T) {
	short := indent("one\ntwo")
	if !strings.Contains(short, "one") || !strings.Contains(short, "two") {
		t.Errorf("short output was altered: %q", short)
	}
	if strings.Contains(short, "more lines") {
		t.Errorf("short output claimed a cut: %q", short)
	}

	long := indent(strings.Repeat("line\n", 40))
	if !strings.Contains(long, "more lines") {
		t.Errorf("long output was cut without saying so: %q", long)
	}
	if n := strings.Count(long, "\n"); n > 13 {
		t.Errorf("the cut output still holds %d lines", n)
	}
}

// The notice reads as a sentence in both shapes. "1 file have changed" is the
// kind of seam that makes generated text read as generated.
func TestTheNoticeAgreesWithItsCount(t *testing.T) {
	if got := plural(1); got != "has" {
		t.Errorf("plural(1) = %q", got)
	}
	for _, n := range []int{0, 2, 17} {
		if got := plural(n); got != "have" {
			t.Errorf("plural(%d) = %q", n, got)
		}
	}
}

// Reading the environment goes through the same chain a resolved configuration
// does, so what a person sets and what a file sets cannot disagree about
// precedence. A workspace that does not exist is refused rather than resolved
// into something that half works.
func TestOptionsFromTheEnvironmentRefuseAWorkspaceThatIsNotThere(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "not-here")
	if _, _, err := FromEnv(func(string) string { return "" }, missing); err == nil {
		t.Fatal("a workspace that does not exist resolved successfully")
	}
}

// The identifier only has to be unlikely to collide, and a machine with no
// randomness available still has to start rather than fail: the number is not a
// secret and nothing depends on it being unguessable.
func TestTheSessionIdentifierIsANumberEvenWhenItIsNotRandom(t *testing.T) {
	seen := map[uint32]bool{}
	for i := 0; i < 32; i++ {
		seen[randomUint32()] = true
	}
	if len(seen) < 2 {
		t.Errorf("32 draws produced %d distinct values", len(seen))
	}
}
