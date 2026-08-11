package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/config"
)

// `/init` reads AGENTS.md and writes a DCODE.md from it. Months later AGENTS.md
// changes, and the DCODE.md still says what the old one said — silently, because
// a generated file that is now hand-edited looks exactly like one that is still
// current.
//
// The whole answer to that was written: RenderDigest builds the marker, Diverged
// reads it, and the warning names which source moved. Nobody ever wrote the
// marker into the file, so Diverged could only ever answer "nothing changed" and
// the warning was unreachable.
func TestStampingRecordsWhatTheFileWasGeneratedFrom(t *testing.T) {
	ws := t.TempDir()
	write(t, ws, "AGENTS.md", "use tabs\n")
	write(t, ws, "DCODE.md", "# dcode\n\nUse tabs.\n")

	StampGenerated(ws, []string{"AGENTS.md"}, []string{"DCODE.md"})

	got := read(t, ws, "DCODE.md")
	if !strings.Contains(got, config.DigestMarker) {
		t.Fatalf("the generated file carries no record of its sources:\n%s", got)
	}
	// Unchanged sources are not a divergence.
	if names, diverged := config.Diverged(got, os.DirFS(ws)); diverged {
		t.Errorf("a freshly stamped file already reports %v as changed", names)
	}

	// The source moves, and now the warning has something true to say.
	write(t, ws, "AGENTS.md", "use spaces\n")
	names, diverged := config.Diverged(read(t, ws, "DCODE.md"), os.DirFS(ws))
	if !diverged || len(names) != 1 || names[0] != "AGENTS.md" {
		t.Errorf("after the source changed, Diverged says %v/%v", names, diverged)
	}
}

// The file belongs to the human the moment it exists. Stamping it twice would
// rewrite what they have since edited, and stamping a file that already carries
// a marker would move the baseline forward — quietly turning "these sources
// changed" into "nothing changed", which is the exact warning being built.
func TestAFileThatAlreadyCarriesAMarkerIsLeftAlone(t *testing.T) {
	ws := t.TempDir()
	write(t, ws, "AGENTS.md", "v1\n")
	write(t, ws, "DCODE.md", "# dcode\n")
	StampGenerated(ws, []string{"AGENTS.md"}, []string{"DCODE.md"})
	first := read(t, ws, "DCODE.md")

	// The source moves and the user edits their file.
	write(t, ws, "AGENTS.md", "v2\n")
	write(t, ws, "DCODE.md", first+"\nA rule I added by hand.\n")
	edited := read(t, ws, "DCODE.md")

	StampGenerated(ws, []string{"AGENTS.md"}, []string{"DCODE.md"})

	if got := read(t, ws, "DCODE.md"); got != edited {
		t.Errorf("a file that was already stamped was rewritten:\n--- was\n%s\n--- now\n%s", edited, got)
	}
	if _, diverged := config.Diverged(read(t, ws, "DCODE.md"), os.DirFS(ws)); !diverged {
		t.Error("re-stamping moved the baseline forward and swallowed the change it exists to report")
	}
}

// Only the file the product generates. A turn that wrote twenty source files
// must not have any of them annotated, and a turn that did not write DCODE.md
// must not touch the one already there.
func TestNothingButTheGeneratedFileIsTouched(t *testing.T) {
	ws := t.TempDir()
	write(t, ws, "AGENTS.md", "v1\n")
	write(t, ws, "DCODE.md", "# dcode\n")
	write(t, ws, "main.go", "package main\n")
	before := read(t, ws, "DCODE.md")

	StampGenerated(ws, []string{"AGENTS.md"}, []string{"main.go", "internal/x.go"})

	if got := read(t, ws, "main.go"); strings.Contains(got, config.DigestMarker) {
		t.Error("a source file was annotated")
	}
	if got := read(t, ws, "DCODE.md"); got != before {
		t.Error("DCODE.md was stamped by a turn that did not write it")
	}
}

// Nothing to record is not an error, and neither is a file that vanished
// between the write and the stamp. This runs after every turn; failing here
// would turn a bookkeeping detail into a broken session.
func TestStampingIsSilentWhenThereIsNothingToRecord(t *testing.T) {
	ws := t.TempDir()
	StampGenerated(ws, []string{"AGENTS.md"}, []string{"DCODE.md"}) // no file at all
	StampGenerated(ws, nil, []string{"DCODE.md"})                   // no sources
	StampGenerated(ws, []string{"AGENTS.md"}, nil)                  // nothing written
}

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, dir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// End to end, and the point of the whole change: before this, InstructionNotice
// could never report a divergence, because nothing put the marker in the file
// for Diverged to read. The warning existed, was worded, was wired to a channel
// — and no input could produce it.
func TestTheDivergenceWarningIsReachableAtAll(t *testing.T) {
	ws := t.TempDir()
	write(t, ws, "AGENTS.md", "use tabs\n")
	write(t, ws, "DCODE.md", "# dcode\n\nUse tabs.\n")

	// The turn that generated it.
	StampGenerated(ws, []string{"AGENTS.md"}, []string{"DCODE.md"})

	if got := InstructionNotice(ws, []string{"AGENTS.md"}, []string{"read"}); got != "" {
		t.Errorf("nothing has changed yet and the session already warns:\n%s", got)
	}

	write(t, ws, "AGENTS.md", "use spaces\n")

	got := InstructionNotice(ws, []string{"AGENTS.md"}, []string{"read"})
	if got == "" {
		t.Fatal("the source changed and the session says nothing; the warning is unreachable")
	}
	// It has to name WHAT moved. "Something changed" sends a person looking.
	if !strings.Contains(got, "AGENTS.md") {
		t.Errorf("the warning does not name the file that changed:\n%s", got)
	}
	if !strings.Contains(got, "/init") {
		t.Errorf("the warning does not say what to do about it:\n%s", got)
	}
}

// A source that was deleted is a different fact from one that was edited, and
// the person needs to know which: re-running /init would carry over an
// instruction file that is no longer there.
func TestASourceThatDisappearedIsReportedAsGone(t *testing.T) {
	ws := t.TempDir()
	write(t, ws, "AGENTS.md", "use tabs\n")
	write(t, ws, "DCODE.md", "# dcode\n")
	StampGenerated(ws, []string{"AGENTS.md"}, []string{"DCODE.md"})

	if err := os.Remove(filepath.Join(ws, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	got := InstructionNotice(ws, []string{"AGENTS.md"}, []string{"read"})
	if !strings.Contains(got, "gone") {
		t.Errorf("a deleted source reads the same as an edited one:\n%s", got)
	}
}
