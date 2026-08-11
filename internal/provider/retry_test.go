package provider

import (
	"testing"
	"time"
)

func TestBackoffDoublesAndStopsAtTheCeiling(t *testing.T) {
	b := Backoff{Base: time.Second, Max: 4 * time.Second, Tries: 6}
	want := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 4 * time.Second, 4 * time.Second}
	for i, w := range want {
		got, ok := b.Wait(i+1, &ProviderError{Retryable: true})
		if !ok {
			t.Fatalf("attempt %d refused", i+1)
		}
		if got != w {
			t.Errorf("attempt %d waited %v, want %v", i+1, got, w)
		}
	}
}

func TestAttemptsRunOut(t *testing.T) {
	b := Backoff{Base: time.Second, Max: time.Minute, Tries: 3}
	if _, ok := b.Wait(3, &ProviderError{Retryable: true}); ok {
		t.Fatal("the policy allowed more attempts than it declares; a real outage would be waited on rather than reported")
	}
}

// The server knows when it will accept the next request and this does not.
// Guessing shorter turns one refusal into several, which is how a rate limit
// becomes a ban.
func TestRetryAfterWinsOverTheComputedDelayEvenWhenLonger(t *testing.T) {
	b := Backoff{Base: time.Second, Max: 2 * time.Second, Tries: 5}
	got, ok := b.Wait(1, &ProviderError{Retryable: true, RetryAfter: 30 * time.Second})
	if !ok {
		t.Fatal("a rate limit naming its own delay was refused")
	}
	if got != 30*time.Second {
		t.Fatalf("waited %v, want the 30s the server asked for — the ceiling must not shorten it", got)
	}
}

func TestAnErrorThatIsNotRetryableIsNotRetried(t *testing.T) {
	b := DefaultBackoff()
	if _, ok := b.Wait(1, &ProviderError{Class: ErrClassAuth, Retryable: false}); ok {
		t.Fatal("a bad credential was retried; no amount of waiting fixes it")
	}
}

func TestAZeroPolicyFallsBackToTheShippedOne(t *testing.T) {
	got, ok := Backoff{}.Wait(1, &ProviderError{Retryable: true})
	if !ok || got != DefaultBackoff().Base {
		t.Fatalf("zero policy gave (%v, %v), want the shipped base", got, ok)
	}
}

// Deterministic on purpose: jitter spreads a herd of clients and there is one
// here, while what it would cost is a wait that differs between two runs of the
// same recorded session.
func TestTheWaitIsTheSameOnEveryRun(t *testing.T) {
	b := DefaultBackoff()
	first, _ := b.Wait(2, &ProviderError{Retryable: true})
	for i := 0; i < 5; i++ {
		if got, _ := b.Wait(2, &ProviderError{Retryable: true}); got != first {
			t.Fatalf("the wait drifted between calls: %v != %v", got, first)
		}
	}
}

// Decide already classified these; nothing acted on the classification.
func TestTheClassesDecideSendsToRetryAreRetryable(t *testing.T) {
	for _, class := range []ErrorClass{ErrClassRateLimit, ErrClassTransport, ErrClassProvider} {
		d := Decide(&ProviderError{Class: class, Retryable: true})
		if d != DecisionWait && d != DecisionRetry {
			t.Errorf("%s decides %v, which the loop does not wait on", class, d)
		}
	}
}

// The server says how long to wait and the product declares it obeys: Wait
// honours RetryAfter over its own backoff. Nothing ever set the field, so on a
// 429 the client picked its own delay and retried into the same wall — polite
// in the type system, impolite on the wire.
func TestRetryAfterIsReadFromTheHeader(t *testing.T) {
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name, header string
		want         time.Duration
	}{
		{"delta seconds", "30", 30 * time.Second},
		{"zero is no instruction", "0", 0},
		{"http date", "Tue, 11 Aug 2026 12:00:45 GMT", 45 * time.Second},
		// A date already past is not a wait, and a negative duration would run
		// the backoff backwards.
		{"a date in the past", "Tue, 11 Aug 2026 11:59:00 GMT", 0},
		{"absent", "", 0},
		{"nonsense", "soon", 0},
		{"negative", "-5", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := RetryAfterOf(c.header, now); got != c.want {
				t.Errorf("RetryAfterOf(%q) = %v, want %v", c.header, got, c.want)
			}
		})
	}
}

// The invariant, and it is structural rather than a convention: only the
// rate-limit branch can carry the field, so no later caller can attach a wait
// to an error that does not have one to give.
func TestOnlyARateLimitCarriesARetryAfter(t *testing.T) {
	for _, status := range []int{401, 402, 413, 400, 404, 500, 503} {
		pe := ClassifyStatus(status, "body", "30")
		if pe != nil && pe.RetryAfter != 0 {
			t.Errorf("status %d produced %v with RetryAfter %v", status, pe.Class, pe.RetryAfter)
		}
	}
	pe := ClassifyStatus(429, "body", "30")
	if pe.Class != ErrClassRateLimit || pe.RetryAfter != 30*time.Second {
		t.Errorf("429 = %v with RetryAfter %v, want rate_limit with 30s", pe.Class, pe.RetryAfter)
	}
	if pe := ClassifyStatus(429, "body", ""); pe.RetryAfter != 0 {
		t.Errorf("a 429 with no header carries %v; absent must not become a number", pe.RetryAfter)
	}
}
