package tools

import (
	"strings"
	"testing"
)

func TestUnifiedDiffOnAnOrdinaryEdit(t *testing.T) {
	before := "package a\n\nfunc main() {\n\tprintln(\"old\")\n}\n"
	after := "package a\n\nfunc main() {\n\tprintln(\"new\")\n}\n"

	got := UnifiedDiff(before, after, "main.go")

	if !strings.Contains(got, "-\tprintln(\"old\")") {
		t.Errorf("the removed line must be marked:\n%s", got)
	}
	if !strings.Contains(got, "+\tprintln(\"new\")") {
		t.Errorf("the added line must be marked:\n%s", got)
	}
	// Context is what lets a reviewer see where the change landed.
	if !strings.Contains(got, " func main() {") {
		t.Errorf("surrounding context must be present:\n%s", got)
	}
	if !strings.HasPrefix(got, "@@ ") {
		t.Errorf("a hunk header locates the change:\n%s", got)
	}
	if !strings.Contains(got, "main.go") {
		t.Errorf("the path belongs in the header:\n%s", got)
	}
	// Removal before addition: it is the order every diff tool prints, and it
	// reads as a replacement rather than as two unrelated events.
	if strings.Index(got, "-\tprintln") > strings.Index(got, "+\tprintln") {
		t.Errorf("removals come first:\n%s", got)
	}
}

// A diff with no hunks is noise, and the empty string is how the caller says
// there is nothing to show.
func TestUnifiedDiffOfIdenticalTextIsEmpty(t *testing.T) {
	if got := UnifiedDiff("same\n", "same\n", "x.go"); got != "" {
		t.Errorf("got %q", got)
	}
	if got := UnifiedDiff("", "", "x.go"); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestUnifiedDiffOfACreationAndOfADeletion(t *testing.T) {
	created := UnifiedDiff("", "um\ndois\n", "novo.go")
	if !strings.Contains(created, "+um") || !strings.Contains(created, "+dois") {
		t.Errorf("a new file is all additions:\n%s", created)
	}
	// Body lines only: the header legitimately carries a `-` for the old range.
	for _, l := range strings.Split(created, "\n") {
		if strings.HasPrefix(l, "-") {
			t.Errorf("nothing was removed, got %q", l)
		}
	}

	emptied := UnifiedDiff("um\ndois\n", "", "x.go")
	if !strings.Contains(emptied, "-um") || !strings.Contains(emptied, "-dois") {
		t.Errorf("an emptied file is all removals:\n%s", emptied)
	}
}

// Untouched runs must collapse, or a one-line change in a large file prints the
// whole file and the diff stops being reviewable.
func TestUnifiedDiffCollapsesUntouchedRuns(t *testing.T) {
	var before, after strings.Builder
	for i := 0; i < 200; i++ {
		before.WriteString("linha\n")
		after.WriteString("linha\n")
	}
	before.WriteString("antes\n")
	after.WriteString("depois\n")

	got := UnifiedDiff(before.String(), after.String(), "big.go")
	lines := strings.Count(got, "\n") + 1

	// One hunk: three lines of context either side, the change, and the header.
	if lines > 12 {
		t.Errorf("the diff should be a single small hunk, got %d lines:\n%s", lines, got)
	}
	if !strings.Contains(got, "-antes") || !strings.Contains(got, "+depois") {
		t.Errorf("got:\n%s", got)
	}
}

// Two changes far apart are two hunks, not one hunk spanning everything
// between them.
func TestUnifiedDiffProducesSeparateHunks(t *testing.T) {
	var before, after strings.Builder
	before.WriteString("topo\n")
	after.WriteString("TOPO\n")
	for i := 0; i < 50; i++ {
		before.WriteString("meio\n")
		after.WriteString("meio\n")
	}
	before.WriteString("base\n")
	after.WriteString("BASE\n")

	got := UnifiedDiff(before.String(), after.String(), "x.go")
	// `@@ ` appears twice per header — opening and closing — so the opening
	// marker is what counts hunks.
	if n := strings.Count(got, "@@ -"); n != 2 {
		t.Errorf("want two hunks, got %d:\n%s", n, got)
	}
}

// The hunk header has to point at the right lines, or the reviewer looks in the
// wrong place in the file.
func TestHunkHeaderLocatesTheChange(t *testing.T) {
	before := "a\nb\nc\nd\ne\nf\ng\nh\n"
	after := "a\nb\nc\nd\nE\nf\ng\nh\n"

	got := UnifiedDiff(before, after, "")
	// The change is on line 5, so with three lines of context the hunk starts
	// at line 2.
	if !strings.HasPrefix(got, "@@ -2,") {
		t.Errorf("got:\n%s", got)
	}
}

// A payload larger than the file itself helps nobody, and the client can only
// show a screenful anyway.
func TestUnifiedDiffIsCapped(t *testing.T) {
	var before, after strings.Builder
	for i := 0; i < 2000; i++ {
		before.WriteString("antes\n")
		after.WriteString("depois\n")
	}
	got := UnifiedDiff(before.String(), after.String(), "x.go")
	if n := strings.Count(got, "\n") + 1; n > DiffMaxLines+1 {
		t.Errorf("the diff must be capped, got %d lines", n)
	}
	if !strings.Contains(got, "truncated") {
		t.Errorf("truncating in silence reads as a complete diff:\n%s", got[len(got)-200:])
	}
}

// Two files with nothing in common degrade to a plain replacement rather than
// stalling the turn in a quadratic table.
func TestUnifiedDiffDegradesOnHugeUnrelatedInput(t *testing.T) {
	var before, after strings.Builder
	for i := 0; i < 3000; i++ {
		before.WriteString("antes ")
		before.WriteString(strings.Repeat("x", i%7))
		before.WriteString("\n")
		after.WriteString("depois ")
		after.WriteString(strings.Repeat("y", i%5))
		after.WriteString("\n")
	}
	// The assertion is that it returns at all, and within the cap.
	got := UnifiedDiff(before.String(), after.String(), "x.go")
	if got == "" {
		t.Fatal("a diff of two unrelated files is still a diff")
	}
	if n := strings.Count(got, "\n") + 1; n > DiffMaxLines+1 {
		t.Errorf("got %d lines", n)
	}
}

// A file without a trailing newline must not grow a phantom empty last line.
func TestUnifiedDiffHandlesAMissingTrailingNewline(t *testing.T) {
	got := UnifiedDiff("um\ndois", "um\ntres", "x.go")
	if strings.Contains(got, "+\n") || strings.HasSuffix(got, "+") {
		t.Errorf("a phantom line appeared:\n%q", got)
	}
	if !strings.Contains(got, "-dois") || !strings.Contains(got, "+tres") {
		t.Errorf("got:\n%s", got)
	}
}
