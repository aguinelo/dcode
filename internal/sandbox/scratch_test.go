package sandbox

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/policy"
)

// A compiled language cannot build without writing to its cache, and the cache
// lives outside the workspace by design — shared across projects, and far too
// large to copy in.
//
// Measured, not guessed: an unattended run on this repository could not execute
// `go test` at all. The first failure was
// `open ~/Library/Caches/go-build/…: operation not permitted`, and the run ended
// having changed files it could never verify. A sandbox that permits editing and
// forbids checking produces exactly the outcome nobody wants.
func TestAToolchainCanWriteToItsOwnCache(t *testing.T) {
	home := t.TempDir()
	caches := []string{filepath.Join(home, "Library", "Caches", "go-build")}

	s := &seatbelt{}
	profile, err := s.profile(t.TempDir(), policy.ModeWorkspaceWrite, caches)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(profile, caches[0]) {
		t.Errorf("the build cache is not writable:\n%s", profile)
	}
}

// The home directory itself is never granted. The whole point of naming caches
// is that they are named — a rule that reached $HOME would hand over the ssh
// keys along with the compiler's scratch space.
func TestTheHomeDirectoryIsNeverGranted(t *testing.T) {
	home := t.TempDir()
	s := &seatbelt{}
	profile, err := s.profile(t.TempDir(), policy.ModeWorkspaceWrite,
		[]string{filepath.Join(home, ".cache")})
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(profile, "\n") {
		if !strings.Contains(line, "file-write*") {
			continue
		}
		if strings.Contains(line, home+`"`) {
			t.Errorf("the home directory itself is writable:\n%s", line)
		}
	}
}

// Read-only stays read-only. A cache is somewhere a build writes, and a mode
// that forbids writing forbids this too — otherwise the mode would mean
// something different depending on which directory was named.
func TestReadOnlyGrantsNoCache(t *testing.T) {
	home := t.TempDir()
	s := &seatbelt{}
	profile, err := s.profile(t.TempDir(), policy.ModeReadOnly,
		[]string{filepath.Join(home, ".cache")})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(profile, "file-write*") {
		t.Errorf("read-only granted a write:\n%s", profile)
	}
}

// The caches a toolchain uses are read from the environment when it says so,
// and fall back to where each toolchain puts them. A machine with GOCACHE set
// somewhere unusual is the machine most likely to be denied.
func TestCachesFollowTheEnvironmentAndThenTheDefaults(t *testing.T) {
	home := t.TempDir()
	env := map[string]string{
		"HOME":    home,
		"GOCACHE": filepath.Join(home, "elsewhere", "go"),
	}
	got := Scratch(func(k string) string { return env[k] })

	if !hasPath(got, filepath.Join(home, "elsewhere", "go")) {
		t.Errorf("GOCACHE was ignored: %v", got)
	}
	// And the ones nothing named still arrive.
	if !hasPath(got, filepath.Join(home, ".cargo")) {
		t.Errorf("the cargo home is missing: %v", got)
	}
	for _, p := range got {
		if p == home {
			t.Fatalf("the home directory itself is in the list: %v", got)
		}
	}
}

// Without a home there is nothing to derive, and nothing is invented. A path
// built from an empty string is "/.cache", which is not a cache and is
// somewhere nobody meant to grant.
func TestNoHomeMeansNoCaches(t *testing.T) {
	if got := Scratch(func(string) string { return "" }); len(got) != 0 {
		t.Errorf("caches were invented without a home: %v", got)
	}
}

// A caller that wired no environment gets nothing, and does not crash. It is
// the same rule the nil network decision already follows: a session must not
// die over a wiring mistake, and nothing said is never a yes.
func TestANilEnvironmentGrantsNothingAndDoesNotPanic(t *testing.T) {
	if got := Scratch(nil); len(got) != 0 {
		t.Errorf("a nil environment granted %v", got)
	}
}

func hasPath(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// The Linux backend binds each cache writable, and skips the ones already
// covered. A cache under /tmp is inside the fresh tmpfs, and binding the host's
// copy back over it would undo the isolation the tmpfs exists for.
func TestBubblewrapBindsCachesAndSkipsWhatIsCovered(t *testing.T) {
	// A literal path, not one derived from t.TempDir(): on Linux the temporary
	// directory lives under /tmp, so the fixture would land inside the tmpfs
	// and be skipped by the very rule this test exists to check. The fixture
	// must not depend on where the platform puts its scratch space.
	cache := "/home/someone/.cache/go-build"
	present(t, cache)
	// args() mounts the canonical spelling, so that is what to look for. A
	// hard-coded literal made this depend on whether the platform resolves the
	// ancestors of the fixture path.
	bound := canonical(cache)
	b := &bubblewrap{}
	args, err := b.args("/nowhere", policy.ModeWorkspaceWrite, []string{cache, "/tmp/inside"})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--bind "+bound+" "+bound) {
		t.Errorf("the cache is not bound writable: %v", args)
	}
	if strings.Contains(joined, "/tmp/inside") {
		t.Errorf("a cache under the tmpfs was bound over it: %v", args)
	}
}

// Read-only binds no cache on Linux either — the two backends have to mean the
// same thing, or the mode means whichever platform you happen to be on.
func TestBubblewrapReadOnlyBindsNoCache(t *testing.T) {
	cache := "/home/someone/.cache"
	b := &bubblewrap{}
	args, err := b.args("/w", policy.ModeReadOnly, []string{cache})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(args, " "), "--bind "+cache) {
		t.Errorf("read-only bound a writable cache: %v", args)
	}
}

// The per-user temporary directory is granted, and on macOS it is not /tmp.
//
// The second blocker, found after the first was fixed: with the build cache
// writable, `go test` got further and then failed staging compilation in
// $TMPDIR under /var/folders. Granting /tmp and /var/tmp and stopping short of
// this one moved the failure rather than removing it.
func TestThePerUserTemporaryDirectoryIsGranted(t *testing.T) {
	home := t.TempDir()
	tmp := filepath.Join(t.TempDir(), "folders", "xx", "T")
	got := Scratch(func(k string) string {
		switch k {
		case "HOME":
			return home
		case "TMPDIR":
			return tmp
		}
		return ""
	})
	if !hasPath(got, tmp) {
		t.Errorf("the per-user temporary directory is missing: %v", got)
	}
}

// And an unset TMPDIR invents nothing. There is no sensible default for it —
// /tmp is already granted, and guessing a path under the home would name a
// directory that does not exist.
func TestAnUnsetTemporaryDirectoryInventsNothing(t *testing.T) {
	home := t.TempDir()
	for _, p := range Scratch(func(k string) string {
		if k == "HOME" {
			return home
		}
		return ""
	}) {
		if strings.HasSuffix(p, "/T") || filepath.Base(p) == "tmp" {
			t.Errorf("a temporary directory was invented: %q", p)
		}
	}
}

// A cache that is not there yet is skipped rather than bound. bubblewrap fails
// the whole command on a missing bind source, so a machine that has never
// compiled would be unable to run anything at all.
func TestAnAbsentCacheIsSkippedRatherThanBound(t *testing.T) {
	present(t) // nothing is there
	b := &bubblewrap{}
	args, err := b.args("/nowhere", policy.ModeWorkspaceWrite,
		[]string{"/definitely/not/here/go-build"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(args, " "), "/definitely/not/here") {
		t.Errorf("a missing directory was bound, which fails every command: %v", args)
	}
}

// Both platform defaults are reachable from either platform. A branch only one
// machine can execute is a branch only one machine can check, and this one
// decides where a compiler may write.
func TestBothPlatformDefaultsAreReachable(t *testing.T) {
	home := t.TempDir()
	env := func(k string) string {
		if k == "HOME" {
			return home
		}
		return ""
	}
	original := goos
	t.Cleanup(func() { goos = original })

	goos = "darwin"
	if !hasPath(Scratch(env), filepath.Join(home, "Library", "Caches", "go-build")) {
		t.Error("the macOS build cache is missing")
	}
	goos = "linux"
	if !hasPath(Scratch(env), filepath.Join(home, ".cache", "go-build")) {
		t.Error("the Linux build cache is missing")
	}
}

// present makes exactly these paths look like existing directories, so a mount
// rule can be asserted without creating one. The real check is os.Stat; what is
// under test is what the rule does with the answer.
func present(t *testing.T, paths ...string) {
	t.Helper()
	set := map[string]bool{}
	for _, p := range paths {
		set[canonical(p)] = true
	}
	original := exists
	t.Cleanup(func() { exists = original })
	exists = func(p string) bool { return set[p] }
}
