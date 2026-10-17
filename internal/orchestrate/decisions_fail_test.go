package orchestrate

import (
	"os"
	"strings"
	"testing"
)

// A failed write that could not be taken back may leave half a line at the
// end of the file. Appending after it would join two decisions on one line,
// so every later decision is refused, and says why.
func TestALogThatCouldNotTakeBackAWriteRefusesTheNext(t *testing.T) {
	l := openLog(t, t.TempDir(), testProject(t))
	_ = l.f.Close()

	if _, err := l.Record(dispatch("first")); err == nil {
		t.Fatal("a write to a closed descriptor was reported as recorded")
	}
	_, err := l.Record(dispatch("second"))
	if err == nil || !strings.Contains(err.Error(), "could not take back") {
		t.Fatalf("err = %v, want the refusal of a log that could not take a write back", err)
	}
}

func TestClosingTwiceIsNotAnError(t *testing.T) {
	l, err := OpenDecisionLog(t.TempDir(), testProject(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}

// Only the free-text fields are cut. A decision over the ceiling with all of
// them cut is an error, not a line over it.
func TestADecisionThatCannotBeCutUnderTheCeilingIsRefused(t *testing.T) {
	dir := t.TempDir()
	l := openLog(t, dir, testProject(t))

	d := dispatch("because")
	d.Child = strings.Repeat("c", maxLine)
	if _, err := l.Record(d); err == nil {
		t.Fatal("a line over the ceiling was written")
	}
	assertEmpty(t, dir)
}

func TestALogThatCannotBeReadRefusesToOpen(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(logPath(dir), 0o700); err != nil { // a directory where the file goes
		t.Fatal(err)
	}
	if l, err := OpenDecisionLog(dir, testProject(t)); err == nil {
		_ = l.Close()
		t.Fatal("the log opened")
	}
}
