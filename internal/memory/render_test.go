package memory

import (
	"fmt"
	"strings"
	"testing"
)

func entries(n int) []Entry {
	var out []Entry
	for i := 0; i < n; i++ {
		out = append(out, Entry{
			Kind: KindGotcha, Subject: fmt.Sprintf("memory-%d", i),
			Commit: fmt.Sprintf("c%d", i),
		})
	}
	return out
}

// Nothing learned renders nothing. A section saying there is no memory is worse
// than no section.
func TestNoMemoryRendersNothing(t *testing.T) {
	if got := Render(File{}, DefaultMax, nil); got != "" {
		t.Errorf("got %q, want nothing", got)
	}
}

// The block says these were the agent's own notes, because the model has to
// weigh them below what a person wrote and cannot do that without being told.
func TestTheBlockSaysWhoWroteThese(t *testing.T) {
	got := Render(File{Entries: entries(1)}, DefaultMax, nil)
	low := strings.ToLower(got)
	if !strings.Contains(low, "yourself") && !strings.Contains(low, "learned") {
		t.Errorf("the block does not say where these came from:\n%s", got)
	}
	if !strings.Contains(low, "person") {
		t.Errorf("the block does not say they rank below a person:\n%s", got)
	}
}

// Past the cap the oldest go, the newest stay, and the cut is declared. Nothing
// here cuts output in silence.
func TestPastTheCapTheOldestGoAndTheCutIsDeclared(t *testing.T) {
	got := Render(File{Entries: entries(50)}, 40, nil)

	if !strings.Contains(got, "memory-49") {
		t.Error("the newest memory was cut")
	}
	if strings.Contains(got, "memory-0") {
		t.Error("the oldest memory survived the cap")
	}
	if !strings.Contains(got, "10 older memories not shown") {
		t.Errorf("the cut was not declared:\n%s", tail(got))
	}
}

// One is one, and a block that says "1 older memories" reads as generated.
func TestTheCutReadsAsASentence(t *testing.T) {
	got := Render(File{Entries: entries(41)}, 40, nil)
	if !strings.Contains(got, "1 older memory not shown") {
		t.Errorf("got the plural for one:\n%s", tail(got))
	}
}

// Within the cap nothing is said about a cut that did not happen.
func TestNoCutIsAnnouncedWhenNothingWasCut(t *testing.T) {
	got := Render(File{Entries: entries(3)}, 40, nil)
	if strings.Contains(got, "not shown") {
		t.Errorf("a cut was announced with nothing cut:\n%s", got)
	}
}

// A memory whose commit the repository no longer has is MARKED and stays. The
// heuristic will be wrong sometimes, and one that deletes loses knowledge with
// nobody told.
func TestAMemoryFromAVanishedCommitIsMarkedAndKept(t *testing.T) {
	f := File{Entries: []Entry{
		{Kind: KindGotcha, Subject: "still true", Commit: "aaa"},
		{Kind: KindGotcha, Subject: "from a rebased commit", Commit: "bbb"},
	}}
	got := Render(f, DefaultMax, map[string]bool{"aaa": true})

	if !strings.Contains(got, "from a rebased commit") {
		t.Fatal("a memory from a vanished commit was dropped")
	}
	marked := strings.Index(got, "no longer in this repository")
	if marked < 0 {
		t.Fatalf("it was not marked:\n%s", got)
	}
	// And only that one.
	if strings.Count(got, "no longer in this repository") != 1 {
		t.Errorf("a memory whose commit is present was marked too:\n%s", got)
	}
}

// Nothing known means nothing could be checked, and "we did not look" must not
// read as "we looked and it is gone".
func TestWithNothingToCheckAgainstNothingIsMarked(t *testing.T) {
	f := File{Entries: []Entry{{Kind: KindGotcha, Subject: "x", Commit: "aaa"}}}
	for _, known := range []map[string]bool{nil, {}} {
		if got := Render(f, DefaultMax, known); strings.Contains(got, "no longer") {
			t.Errorf("marked a memory with nothing to check against:\n%s", got)
		}
	}
}

// A memory written by hand carries no commit, and cannot be judged stale.
func TestAMemoryWithNoCommitIsNeverMarked(t *testing.T) {
	f := File{Entries: []Entry{{Kind: KindGotcha, Subject: "by hand"}}}
	if got := Render(f, DefaultMax, map[string]bool{"aaa": true}); strings.Contains(got, "no longer") {
		t.Errorf("marked a memory that never claimed a commit:\n%s", got)
	}
}

// Blocks that could not be read are counted in the prefix, so the person is
// told their file holds less than it looks like.
func TestCrookedBlocksAreCountedInThePrefix(t *testing.T) {
	f := File{Entries: entries(1), Malformed: []string{"## nope:", "## also nope"}}
	got := Render(f, DefaultMax, nil)
	if !strings.Contains(got, "2 blocks in the memory file could not be read") {
		t.Errorf("the unreadable blocks were not reported:\n%s", tail(got))
	}
}

// A file with nothing but crooked blocks still says so. Rendering nothing there
// would be the silence the report exists to break.
func TestAFileOfNothingButCrookedBlocksStillSaysSo(t *testing.T) {
	got := Render(File{Malformed: []string{"## nope:"}}, DefaultMax, nil)
	if !strings.Contains(got, "could not be read") {
		t.Errorf("got %q", got)
	}
}

// A cap of zero or less is the default, not a prefix with no memory in it. A
// misconfiguration must not silently switch the component off.
func TestACapOfZeroFallsBackToTheDefault(t *testing.T) {
	for _, max := range []int{0, -5} {
		got := Render(File{Entries: entries(3)}, max, nil)
		if !strings.Contains(got, "memory-0") {
			t.Errorf("max=%d rendered nothing:\n%s", max, got)
		}
	}
}

func tail(s string) string {
	if len(s) < 400 {
		return s
	}
	return "…" + s[len(s)-400:]
}

// The block is a pure function of what was read. Build's purity depends on it:
// the same memory must produce the same prefix, or every cached prefix and every
// reproducibility claim in the context engine goes with it.
func TestTheLearnedBlockIsPure(t *testing.T) {
	f := File{
		Entries:   entries(3),
		Malformed: []string{"## nope:"},
	}
	known := map[string]bool{"c1": true}
	first := Render(f, DefaultMax, known)
	for i := 0; i < 5; i++ {
		if got := Render(f, DefaultMax, known); got != first {
			t.Fatal("the same memory produced a different block")
		}
	}
}
