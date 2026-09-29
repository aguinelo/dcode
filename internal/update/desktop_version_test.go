package update

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// -- desktop/ and the core's version --------------------------------------------
//
// The desktop app lives in desktop/, with its own changelog and its own version,
// tagged desktop-v*. A commit that only touches it is not a change to the core,
// and counting it would raise the core's version for work the core never saw —
// the shape the pin commit already taught: one mechanism's trace read as another
// mechanism's signal.

func TestTheVersionIgnoresCommitsThatOnlyTouchTheDesktop(t *testing.T) {
	dir := seededRepo(t, "v0.1.0")
	desktopCommitAt(t, dir, "feat(desktop): the main window")

	got, err := versionIn(t, dir)
	if err != nil {
		t.Fatalf("version.sh failed: %v", err)
	}
	if got != "v0.1.0" {
		t.Errorf("a commit that only touched desktop/ moved the core's version: got %q, want v0.1.0", got)
	}
}

// A commit that touches the desktop AND the core is a change to the core. A
// protocol change the desktop consumes is the usual case, and it must count.
func TestTheVersionCountsACommitThatAlsoTouchesTheCore(t *testing.T) {
	dir := seededRepo(t, "v0.1.0")
	if err := os.MkdirAll(filepath.Join(dir, "desktop"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"desktop/app": "window", "f": "core"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	gitAt(t, dir, "add", "desktop/app", "f")
	gitAt(t, dir, "commit", "--quiet", "-m", "feat(protocol,desktop): sessions say where they came from")

	got, err := versionIn(t, dir)
	if err != nil {
		t.Fatalf("version.sh failed: %v", err)
	}
	if got != "v0.2.0" {
		t.Errorf("a commit touching the core and the desktop was not counted: got %q, want v0.2.0", got)
	}
}

// The desktop's tags are not the core's reference point. Deriving from
// desktop-v0.3.0 would either refuse its shape or count from the wrong commit.
func TestTheVersionIsNotReadFromADesktopTag(t *testing.T) {
	dir := seededRepo(t, "v0.1.0")
	commitAt(t, dir, "fix: a core bug", "")
	gitAt(t, dir, "tag", "-a", "desktop-v0.3.0", "-m", "desktop-v0.3.0")

	got, err := versionIn(t, dir)
	if err != nil {
		t.Fatalf("version.sh failed: %v", err)
	}
	if got != "v0.1.1" {
		t.Errorf("the core's version was derived from a desktop tag: got %q, want v0.1.1", got)
	}
}

// The skeleton of the core's changelog follows the same two rules: what only the
// desktop did belongs to the desktop's changelog, and the desktop's tag must not
// cut the range short — a desktop tag at HEAD used to leave the skeleton empty.
func TestTheChangelogSkeletonLeavesTheDesktopToItsOwnChangelog(t *testing.T) {
	dir := seededRepo(t, "v0.1.0")
	desktopCommitAt(t, dir, "feat(desktop): the main window")
	commitAt(t, dir, "fix: a core bug", "")
	gitAt(t, dir, "tag", "-a", "desktop-v0.1.0", "-m", "desktop-v0.1.0")

	c := exec.Command("bash", filepath.Join(repoRoot(t), "scripts", "changelog.sh"), "v0.1.1", "en")
	c.Dir = dir
	out, err := c.Output()
	if err != nil {
		t.Fatalf("changelog.sh failed: %v", err)
	}
	skeleton := string(out)
	if strings.Contains(skeleton, "the main window") {
		t.Errorf("the core's changelog skeleton lists a desktop-only commit:\n%s", skeleton)
	}
	if !strings.Contains(skeleton, "a core bug") {
		t.Errorf("the core's changelog skeleton lost a core commit behind a desktop tag:\n%s", skeleton)
	}
}

// desktopCommitAt commits a change under desktop/ and nowhere else.
func desktopCommitAt(t *testing.T, dir, subject string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "desktop"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "desktop", "app"), []byte(subject), 0o644); err != nil {
		t.Fatal(err)
	}
	gitAt(t, dir, "add", "desktop/app")
	gitAt(t, dir, "commit", "--quiet", "-m", subject)
}
