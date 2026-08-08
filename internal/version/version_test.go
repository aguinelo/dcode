package version

import (
	"strings"
	"testing"
)

// A build with no injected values must say "dev" rather than an empty string or
// something that looks like a release. A binary that cannot say what it is
// should say so.
func TestUninjectedBuildReportsDev(t *testing.T) {
	restore(t)
	Version, Commit, Date = "", "", ""

	if got := Short(); got != "dev" {
		t.Errorf("got %q want dev", got)
	}
	full := String()
	if !strings.Contains(full, "dev") {
		t.Errorf("got %q", full)
	}
	// Platform and toolchain are always useful in a bug report.
	if !strings.Contains(full, "go1.") {
		t.Errorf("the Go version should be reported: %q", full)
	}
}

func TestInjectedBuildReportsEverything(t *testing.T) {
	restore(t)
	Version, Commit, Date = "v1.2.3", "abcdef0123456789abcdef", "2026-08-08"

	if got := Short(); got != "v1.2.3" {
		t.Errorf("got %q", got)
	}
	full := String()
	for _, want := range []string{"v1.2.3", "2026-08-08"} {
		if !strings.Contains(full, want) {
			t.Errorf("%q missing from %q", want, full)
		}
	}
	// A full hash is noise in a version line; the short form still identifies
	// the build.
	if strings.Contains(full, "abcdef0123456789abcdef") {
		t.Errorf("the commit should be abbreviated: %q", full)
	}
	if !strings.Contains(full, "abcdef012345") {
		t.Errorf("the short commit should be present: %q", full)
	}
}

func TestShortCommitIsNotPaddedOrTruncatedWrongly(t *testing.T) {
	restore(t)
	Version, Commit, Date = "v1", "abc123", ""
	if !strings.Contains(String(), "abc123") {
		t.Errorf("a commit shorter than the cut must survive intact: %q", String())
	}
}

func restore(t *testing.T) {
	t.Helper()
	v, c, d := Version, Commit, Date
	t.Cleanup(func() { Version, Commit, Date = v, c, d })
}
