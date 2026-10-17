package orchestrate

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func readAll(t *testing.T, r *LogReader) []LogLine {
	t.Helper()
	var got []LogLine
	for {
		l, err := r.Next()
		if errors.Is(err, io.EOF) {
			return got
		}
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, l)
	}
}

const (
	line1 = `{"v":1,"seq":1,"at":"2026-10-17T21:04:05Z","project":"0123456789abcdef01234567","kind":"dispatch","why":"three themes"}`
	line2 = `{"v":1,"seq":2,"at":"2026-10-17T21:04:06Z","project":"0123456789abcdef01234567","kind":"gate.measured","child":"c1","why":"the gate ran"}`
	line4 = `{"v":1,"seq":4,"at":"2026-10-17T21:04:08Z","project":"0123456789abcdef01234567","kind":"pr.opened","child":"c1","why":"green"}`
)

// Spec: orchestration .p §9.2 — a line that does not read is said with its
// number, and the rest is still read.
func TestReadingTheLogSaysEveryLineThatDoesNotReadWithItsNumber(t *testing.T) {
	in := line1 + "\n" + "{not json\n" + `{"v":2,"seq":2,"kind":"dispatch","why":"x"}` + "\n" + line2 + "\n"
	got := readAll(t, NewLogReader(strings.NewReader(in)))
	if len(got) != 4 {
		t.Fatalf("read %d lines, want 4", len(got))
	}
	if got[0].Err != nil || got[3].Err != nil {
		t.Fatalf("readable lines came back with errors: %v, %v", got[0].Err, got[3].Err)
	}
	for _, i := range []int{1, 2} {
		if got[i].Err == nil {
			t.Errorf("line %d read as a decision", got[i].N)
		}
		if got[i].N != i+1 {
			t.Errorf("line number = %d, want %d", got[i].N, i+1)
		}
	}
	if !strings.Contains(got[2].Err.Error(), "v2") {
		t.Errorf("an unknown format does not say which: %v", got[2].Err)
	}
	if string(got[3].Raw) != line2 {
		t.Errorf("raw = %q, want the line as written", got[3].Raw)
	}
	if got[3].Decision.Child != "c1" || got[3].Decision.Kind != "gate.measured" {
		t.Errorf("decision = %+v", got[3].Decision)
	}
}

// Spec: orchestration .p §9.2 — a seq with a gap is said.
func TestReadingTheLogSaysAGapInTheSeq(t *testing.T) {
	got := readAll(t, NewLogReader(strings.NewReader(line1+"\n"+line2+"\n"+line4+"\n"+line2+"\n")))
	if got[0].Gap || got[1].Gap {
		t.Errorf("a continuous seq was said to have a gap")
	}
	if !got[2].Gap || got[2].Prev != 2 {
		t.Errorf("seq 4 after 2: gap=%v prev=%d, want a gap after 2", got[2].Gap, got[2].Prev)
	}
	if !got[3].Gap || got[3].Prev != 4 {
		t.Errorf("seq 2 after 4: gap=%v prev=%d, want a repetition said", got[3].Gap, got[3].Prev)
	}
}

// A line still being written is not a line yet: it waits for its newline, so
// a reader that follows the log never reads half a decision.
func TestReadingTheLogKeepsAPartialLastLineUntilItEnds(t *testing.T) {
	path := t.TempDir() + "/decisions.jsonl"
	if err := os.WriteFile(path, []byte(line1+"\n"+line2[:20]), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r := NewLogReader(f)
	if got := readAll(t, r); len(got) != 1 {
		t.Fatalf("read %d lines, want only the complete one", len(got))
	}
	if string(r.Pending()) != line2[:20] {
		t.Errorf("pending = %q, want the partial line", r.Pending())
	}

	w, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteString(line2[20:] + "\n"); err != nil {
		t.Fatal(err)
	}
	w.Close()
	got := readAll(t, r)
	if len(got) != 1 || got[0].Err != nil || string(got[0].Raw) != line2 || got[0].N != 2 {
		t.Fatalf("after the line ended: %+v", got)
	}
	if r.Pending() != nil {
		t.Errorf("pending = %q after the line ended", r.Pending())
	}
}
