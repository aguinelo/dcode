package behavior

import (
	"strings"
	"testing"
)

// Same state, same reminders, byte for byte. Without this a replayed history
// diverges from the live one and the prompt cache is lost on every turn.
func TestEmitIsPureAndOrdered(t *testing.T) {
	st := SessionState{
		ChangedFiles:  []string{"b.go", "a.go"},
		DeniedTools:   []string{"bash", "bash", "write"},
		Compacted:     true,
		ParallelBatch: 3,
		OutOfChain: []OutOfChainInstruction{
			{Path: "/z/AGENTS.md", Text: "z"},
			{Path: "/a/AGENTS.md", Text: "a"},
		},
	}
	first := Emit(st)
	second := Emit(st)
	if len(first) != len(second) {
		t.Fatalf("got %d and %d reminders", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("reminder %d differs between calls:\n%+v\n%+v", i, first[i], second[i])
		}
	}

	kinds := make([]ReminderKind, len(first))
	for i, r := range first {
		kinds[i] = r.Kind
	}
	want := []ReminderKind{
		ReminderFileChanged, ReminderApprovalDenied, ReminderCompacted,
		ReminderToolsParallel, ReminderInstructionOutOfChain, ReminderInstructionOutOfChain,
	}
	if len(kinds) != len(want) {
		t.Fatalf("got %v", kinds)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Errorf("position %d: got %s want %s", i, kinds[i], want[i])
		}
	}
}

// Input ordering must not reach the text: a set that differs only in the order
// it was collected has to render identically.
func TestEmitNormalisesOrderAndDuplicates(t *testing.T) {
	a := Emit(SessionState{ChangedFiles: []string{"b.go", "a.go"}, DeniedTools: []string{"bash", "bash"}})
	b := Emit(SessionState{ChangedFiles: []string{"a.go", "b.go"}, DeniedTools: []string{"bash"}})
	if len(a) != len(b) {
		t.Fatalf("got %d and %d", len(a), len(b))
	}
	for i := range a {
		if a[i].Text != b[i].Text {
			t.Errorf("collection order leaked into the text:\n%q\n%q", a[i].Text, b[i].Text)
		}
	}
}

func TestEmitSaysNothingWhenNothingHappened(t *testing.T) {
	if got := Emit(SessionState{}); len(got) != 0 {
		t.Errorf("got %+v", got)
	}
	// One tool is not a parallel batch.
	if got := Emit(SessionState{ParallelBatch: 1}); len(got) != 0 {
		t.Errorf("a single call is not concurrent, got %+v", got)
	}
}

// The text is constant per kind. A counter or a timestamp in here would put
// volatile data in the history and cost the whole cached prefix.
func TestReminderTextCarriesNoVolatileData(t *testing.T) {
	for _, r := range Emit(SessionState{ParallelBatch: 7, Compacted: true}) {
		if strings.ContainsAny(r.Text, "0123456789") {
			t.Errorf("%s carries a number: %q", r.Kind, r.Text)
		}
	}
}

func TestEmitNamesTheFilesAndTools(t *testing.T) {
	got := Emit(SessionState{ChangedFiles: []string{"main.go"}, DeniedTools: []string{"bash"}})
	if !strings.Contains(got[0].Text, "main.go") {
		t.Errorf("a path is data already in the history and belongs in the text: %q", got[0].Text)
	}
	if !strings.Contains(got[1].Text, "bash") {
		t.Errorf("got %q", got[1].Text)
	}
}

// A denial must not read as an invitation to find another way round.
func TestDenialReminderForbidsARetryAndAWorkaround(t *testing.T) {
	text := Emit(SessionState{DeniedTools: []string{"bash"}})[0].Text
	for _, phrase := range []string{"not retry", "another route"} {
		if !strings.Contains(text, phrase) {
			t.Errorf("the denial reminder must rule out %q: %q", phrase, text)
		}
	}
}

func TestOutOfChainReminderCarriesThePathAndTheRules(t *testing.T) {
	got := Emit(SessionState{OutOfChain: []OutOfChainInstruction{
		{Path: "/w/vendor/AGENTS.md", Text: "  never edit here  "},
	}})
	if len(got) != 1 {
		t.Fatalf("got %+v", got)
	}
	if !strings.Contains(got[0].Text, "/w/vendor/AGENTS.md") {
		t.Errorf("got %q", got[0].Text)
	}
	if !strings.Contains(got[0].Text, "never edit here") {
		t.Errorf("the rules themselves must be carried, not just the path: %q", got[0].Text)
	}
}

// Without the marker the model reads a reminder as the user speaking, and
// answers it.
func TestRenderMarksTheChannel(t *testing.T) {
	got := Render(Reminder{Kind: ReminderCompacted, Text: "hello"})
	if !strings.HasPrefix(got, "<system-reminder>") || !strings.HasSuffix(got, "</system-reminder>") {
		t.Errorf("got %q", got)
	}
	if !strings.Contains(got, "hello") {
		t.Errorf("got %q", got)
	}
}
