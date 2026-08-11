package provider

import (
	"strconv"
	"strings"
	"time"
)

// Backoff is how long to wait before trying again.
//
// Pure: it computes a duration and never sleeps, and it holds no clock. The
// caller waits, which is what keeps this decidable in a test without one — and
// what keeps the wait out of a package that must stay reproducible.
type Backoff struct {
	// Base is the first delay. Each attempt doubles it.
	Base time.Duration
	// Max caps a single wait. Without it, exponential growth turns a long
	// outage into a process that appears hung.
	Max time.Duration
	// Tries is how many attempts are made in total, the first included.
	Tries int
}

// DefaultBackoff is the shipped policy.
//
// Five attempts over roughly fifteen seconds. Enough to ride out a rate limit
// or a dropped connection, short enough that a real outage is reported rather
// than waited on — a user watching a cursor learns nothing from the sixth
// silent retry.
func DefaultBackoff() Backoff {
	return Backoff{Base: time.Second, Max: 8 * time.Second, Tries: 5}
}

// Wait returns how long to pause before attempt, and whether to try at all.
//
// attempt is 1-based and counts the attempt about to be made, so Wait(1, …) is
// the pause before the first retry.
//
// A rate limit that names its own delay wins over the computed one, always,
// including when it is longer than Max. The server knows when it will accept
// the next request and this does not; guessing shorter turns one refusal into
// several, which is how a rate limit becomes a ban.
func (b Backoff) Wait(attempt int, e *ProviderError) (time.Duration, bool) {
	if b.Tries <= 0 {
		b = DefaultBackoff()
	}
	if attempt < 1 || attempt >= b.Tries {
		return 0, false
	}
	if e != nil && !e.Retryable {
		return 0, false
	}
	if e != nil && e.RetryAfter > 0 {
		return e.RetryAfter, true
	}

	base := b.Base
	if base <= 0 {
		base = time.Second
	}
	d := base << (attempt - 1)
	// No jitter, deliberately. Jitter exists to spread a herd of clients, and
	// there is one client here; what it would cost is a wait that differs
	// between two runs of the same recorded session, which is exactly the
	// property the replayed-transcript tests depend on.
	if max := b.Max; max > 0 && d > max {
		d = max
	}
	return d, true
}

// RetryAfterOf reads an HTTP Retry-After value.
//
// Both forms the header allows: delta-seconds, and an HTTP date. The date form
// is why this takes a clock — a deadline is only a wait relative to now, and a
// date already past is not a wait at all.
//
// Anything absent, unparseable or negative answers zero, which the backoff
// reads as "no instruction" and falls back to its own schedule. An absent
// signal must not become a number: a wait invented from a header nobody sent is
// worse than the default it replaced.
func RetryAfterOf(header string, now time.Time) time.Duration {
	h := strings.TrimSpace(header)
	if h == "" {
		return 0
	}
	if secs, err := strconv.Atoi(h); err == nil {
		if secs <= 0 {
			return 0
		}
		return time.Duration(secs) * time.Second
	}
	// The three layouts the HTTP date form allows, written out rather than
	// reached for through net/http: this package declares that its standard
	// suite runs with the network off, and the cheapest way to keep that true
	// is not to import the thing that dials.
	for _, layout := range []string{time.RFC1123, time.RFC850, time.ANSIC} {
		when, err := time.Parse(layout, h)
		if err != nil {
			continue
		}
		if d := when.Sub(now); d > 0 {
			return d
		}
		return 0
	}
	return 0
}
