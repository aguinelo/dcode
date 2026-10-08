package memory

import (
	"strings"
	"testing"
)

// A block that is not a memory is reported with what is wrong in it, in the
// words a person fixing the file needs: which kind is not one, or which half of
// the header is missing. The grammar is this package's, so the reason is too.
func TestACrookedBlockSaysWhatIsWrongWithIt(t *testing.T) {
	for line, want := range map[string]string{
		"## diary: what I did today": `"diary" is not a kind of memory; the kinds are gotcha, decision and convention`,
		"## gotcha:":                 "no subject after the kind",
		"## just a heading":          "no kind before the subject",
		"## gotcha: indented":        "indented",
	} {
		if got := Diagnose(line); !strings.Contains(got, want) {
			t.Errorf("Diagnose(%q) = %q, want it to say %q", line, got, want)
		}
	}
}

// Past the cap the oldest go, and Hidden says how many: the same count Render
// declares, so a list of memories and the prefix agree on which ones a session
// reads.
func TestHiddenCountsTheOldestPastTheCap(t *testing.T) {
	f := File{Entries: entries(5)}
	for _, tc := range []struct{ max, want int }{{3, 2}, {5, 0}, {9, 0}, {0, 0}} {
		if got := Hidden(f, tc.max); got != tc.want {
			t.Errorf("Hidden(5 entries, max %d) = %d, want %d", tc.max, got, tc.want)
		}
	}
	got := Render(f, 3, nil)
	if !strings.Contains(got, "2 older memories not shown") {
		t.Errorf("the prefix disagrees with Hidden:\n%s", got)
	}
}

// Stale is the mark Render prints: a commit the repository no longer has, and
// only when something could be checked and the memory claimed a commit.
func TestStaleIsTheMarkTheBlockPrints(t *testing.T) {
	known := map[string]bool{"aaa": true}
	for _, tc := range []struct {
		e     Entry
		known map[string]bool
		want  bool
	}{
		{Entry{Commit: "bbb"}, known, true},
		{Entry{Commit: "aaa"}, known, false},
		{Entry{}, known, false},
		{Entry{Commit: "bbb"}, nil, false},
	} {
		if got := Stale(tc.e, tc.known); got != tc.want {
			t.Errorf("Stale(%+v, %v) = %v, want %v", tc.e, tc.known, got, tc.want)
		}
	}
}
