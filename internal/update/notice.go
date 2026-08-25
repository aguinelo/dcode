package update

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// VersionNotice is the cached result of a version check.
//
// It is persisted to disk and NEVER placed in the model's context. A timestamp
// that changes between turns would break the append-only prefix, and the
// resulting cache miss would cost far more than the notice is worth (ADR-03).
type VersionNotice struct {
	Current   string    `json:"current"`
	Latest    string    `json:"latest"`
	CheckedAt time.Time `json:"checked_at"`
}

// NoticeFileName is where the check result is cached.
const NoticeFileName = "update-check.json"

// DefaultInterval is the minimum gap between checks. Anything below an hour is
// network noise for no gain — the release cadence does not justify it.
const DefaultInterval = 24 * time.Hour

// MinInterval is the floor a configured interval is clamped to.
const MinInterval = time.Hour

// ParseInterval reads DCODE_UPDATE_CHECK_INTERVAL, clamped to MinInterval.
func ParseInterval(s string) time.Duration {
	if s == "" {
		return DefaultInterval
	}
	d, err := time.ParseDuration(s)
	if err != nil || d < MinInterval {
		return DefaultInterval
	}
	return d
}

// ParseBool reads a boolean environment value, defaulting when unset or
// unparsable — a malformed value must not stop the program from starting.
func ParseBool(s string, def bool) bool {
	if s == "" {
		return def
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return def
	}
	return v
}

// LoadNotice reads the cached check. A missing or corrupt cache is not an
// error: the worst it costs is one extra request.
func LoadNotice(path string) (VersionNotice, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return VersionNotice{}, false
	}
	var n VersionNotice
	if err := json.Unmarshal(data, &n); err != nil {
		return VersionNotice{}, false
	}
	return n, true
}

// SaveNotice writes the cache, creating the directory if needed.
func SaveNotice(path string, n VersionNotice) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(n)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// Stale reports whether the cached check is old enough to redo.
func (n VersionNotice) Stale(now time.Time, interval time.Duration) bool {
	return now.Sub(n.CheckedAt) >= interval
}

// Outdated reports whether a NEWER version exists.
//
// It used to ask whether the two differed, which is not the same question and
// answers it wrong in the one direction that matters. A binary ahead of the
// last release — a local build, or a release the cached check has not caught up
// with yet — was told `dcode v0.4.0 is available (you have 0.5.0). Run `dcode
// update`.`: an offer to go backwards, wearing the word update.
//
// `update` itself already refuses to go backwards. The tool was contradicting
// itself in two places on the same screen.
func (n VersionNotice) Outdated() bool {
	return n.Latest != "" && Newer(n.Latest, n.Current)
}

// Newer reports whether version a is later than version b.
//
// Numeric field by field, missing fields read as zero, and a pre-release suffix
// (`-dev+abc`, `-rc1`) is ignored for ordering: it exists to say "not a
// release", which the caller has already decided by other means. Anything it
// cannot read compares as not newer — the safe answer for a function whose
// output is an offer to replace a working binary.
func Newer(a, b string) bool {
	x, okA := fields(a)
	y, okB := fields(b)
	if !okA || !okB {
		return false
	}
	for i := range x {
		if x[i] != y[i] {
			return x[i] > y[i]
		}
	}
	return false
}

// fields reads MAJOR.MINOR.PATCH, tolerating a missing tail and a suffix.
func fields(v string) ([3]int, bool) {
	var out [3]int
	s := normalise(v)
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		s = s[:i]
	}
	if s == "" {
		return out, false
	}
	for i, part := range strings.SplitN(s, ".", 3) {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// Message is the one line a client shows. Empty when there is nothing to say.
func (n VersionNotice) Message() string {
	if !n.Outdated() {
		return ""
	}
	return "dcode " + n.Latest + " is available (you have " + n.Current + "). Run `dcode update`."
}

func normalise(v string) string { return strings.TrimPrefix(strings.TrimSpace(v), "v") }

// Check refreshes the notice at most once per interval.
//
// A network failure is silent by contract: it is logged at debug level and the
// cached answer is returned unchanged. Checking for a version can never degrade
// the use of the tool, and it never changes an exit code (RN-4).
func Check(ctx context.Context, u Updater, path, current string, now time.Time, interval time.Duration) VersionNotice {
	cached, ok := LoadNotice(path)
	if ok {
		cached.Current = current
		if !cached.Stale(now, interval) {
			return cached
		}
	}

	rel, err := u.Latest(ctx)
	if err != nil {
		// Silent on purpose. Returning the stale answer beats reporting a
		// network problem the user did not ask about.
		return cached
	}

	fresh := VersionNotice{Current: current, Latest: rel.Version, CheckedAt: now}
	_ = SaveNotice(path, fresh)
	return fresh
}
