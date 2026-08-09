package update

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type stubUpdater struct {
	rel   Release
	err   error
	calls int
}

func (s *stubUpdater) Latest(context.Context) (Release, error) {
	s.calls++
	return s.rel, s.err
}

func (s *stubUpdater) Apply(context.Context, Release) error { return nil }

func TestParseInterval(t *testing.T) {
	for in, want := range map[string]time.Duration{
		"":         DefaultInterval,
		"6h":       6 * time.Hour,
		"1h":       time.Hour,
		"nonsense": DefaultInterval,
		// Below an hour is network noise for no gain: the release cadence does
		// not justify it.
		"30s": DefaultInterval,
		"59m": DefaultInterval,
	} {
		if got := ParseInterval(in); got != want {
			t.Errorf("%q: got %v want %v", in, got, want)
		}
	}
}

func TestParseBoolDefaultsRatherThanFailing(t *testing.T) {
	for _, tc := range []struct {
		in   string
		def  bool
		want bool
	}{
		{"", true, true},
		{"", false, false},
		{"false", true, false},
		{"1", false, true},
		// A malformed value must not stop the program from starting.
		{"maybe", true, true},
	} {
		if got := ParseBool(tc.in, tc.def); got != tc.want {
			t.Errorf("%q with default %v: got %v", tc.in, tc.def, got)
		}
	}
}

func TestNoticeRoundTripsThroughDisk(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", NoticeFileName)
	now := time.Now().Truncate(time.Second)
	want := VersionNotice{Current: "0.1.0", Latest: "v0.2.0", CheckedAt: now}
	if err := SaveNotice(path, want); err != nil {
		t.Fatal(err)
	}
	got, ok := LoadNotice(path)
	if !ok || got.Latest != want.Latest || !got.CheckedAt.Equal(now) {
		t.Fatalf("got %+v", got)
	}
}

// The worst a missing or corrupt cache can cost is one extra request, so it is
// not an error.
func TestLoadNoticeToleratesAMissingOrCorruptFile(t *testing.T) {
	dir := t.TempDir()
	if _, ok := LoadNotice(filepath.Join(dir, "absent.json")); ok {
		t.Error("a missing cache reports absent")
	}
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte("{{{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := LoadNotice(bad); ok {
		t.Error("a corrupt cache reports absent")
	}
}

func TestOutdatedAndMessage(t *testing.T) {
	for _, tc := range []struct {
		current, latest string
		outdated        bool
	}{
		{"0.1.0", "v0.2.0", true},
		{"0.1.0", "v0.1.0", false},
		{"v0.1.0", "0.1.0", false},
		{"0.1.0", "", false},
	} {
		n := VersionNotice{Current: tc.current, Latest: tc.latest}
		if n.Outdated() != tc.outdated {
			t.Errorf("%s vs %s: got %v", tc.current, tc.latest, n.Outdated())
		}
		if got := n.Message(); (got != "") != tc.outdated {
			t.Errorf("%s vs %s: got %q", tc.current, tc.latest, got)
		}
	}
	// The notice tells the user what to do; it never does it (RN-3).
	msg := VersionNotice{Current: "0.1.0", Latest: "v0.2.0"}.Message()
	if !strings.Contains(msg, "dcode update") {
		t.Errorf("got %q", msg)
	}
}

func TestCheckHonoursTheInterval(t *testing.T) {
	path := filepath.Join(t.TempDir(), NoticeFileName)
	now := time.Now()
	u := &stubUpdater{rel: Release{Version: "v0.2.0"}}

	first := Check(context.Background(), u, path, "0.1.0", now, time.Hour)
	if first.Latest != "v0.2.0" || u.calls != 1 {
		t.Fatalf("got %+v after %d calls", first, u.calls)
	}

	// Within the interval the cached answer stands, with no second request.
	again := Check(context.Background(), u, path, "0.1.0", now.Add(30*time.Minute), time.Hour)
	if u.calls != 1 {
		t.Errorf("the check must not repeat inside the interval, got %d calls", u.calls)
	}
	if again.Latest != "v0.2.0" {
		t.Errorf("got %+v", again)
	}

	// Past it, it refreshes.
	u.rel = Release{Version: "v0.3.0"}
	fresh := Check(context.Background(), u, path, "0.1.0", now.Add(2*time.Hour), time.Hour)
	if u.calls != 2 || fresh.Latest != "v0.3.0" {
		t.Errorf("got %+v after %d calls", fresh, u.calls)
	}
}

// Checking for a version can never degrade the use of the tool (RN-4).
func TestCheckIsSilentWhenTheNetworkFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), NoticeFileName)
	now := time.Now()

	// With nothing cached, a failure yields nothing to say — not an error.
	got := Check(context.Background(), &stubUpdater{err: errors.New("offline")}, path, "0.1.0", now, time.Hour)
	if got.Message() != "" {
		t.Errorf("got %q", got.Message())
	}

	// With something cached, the stale answer stands rather than being lost.
	if err := SaveNotice(path, VersionNotice{
		Current: "0.1.0", Latest: "v0.2.0", CheckedAt: now.Add(-48 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	got = Check(context.Background(), &stubUpdater{err: errors.New("offline")}, path, "0.1.0", now, time.Hour)
	if got.Latest != "v0.2.0" {
		t.Errorf("a network failure must not discard what was already known: %+v", got)
	}
}

func TestStale(t *testing.T) {
	now := time.Now()
	n := VersionNotice{CheckedAt: now.Add(-2 * time.Hour)}
	if !n.Stale(now, time.Hour) {
		t.Error("two hours old with a one-hour interval is stale")
	}
	if n.Stale(now, 3*time.Hour) {
		t.Error("two hours old with a three-hour interval is fresh")
	}
}

func TestSaveNoticeReportsAnUnwritablePath(t *testing.T) {
	file := filepath.Join(t.TempDir(), "occupied")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	// A regular file where a directory has to go.
	if err := SaveNotice(filepath.Join(file, "sub", NoticeFileName), VersionNotice{}); err == nil {
		t.Error("an unwritable cache path must be reported to the caller")
	}
}
