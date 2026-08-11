package provider

import "time"

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
