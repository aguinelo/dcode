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
