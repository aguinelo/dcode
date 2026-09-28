package provider

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

func TestCancelClosesChannelWithCanceled(t *testing.T) {
	r, _ := registry(t, `{"choices":[{"delta":{"content":"a"}}]}`,
		`{"choices":[{"delta":{"content":"b"}}]}`, "[DONE]")
	p, _ := r.Resolve("MiniMax-M3", "")

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := p.Stream(ctx, request())
	if err != nil {
		t.Fatal(err)
	}
	cancel()

	// Whatever was already buffered may still arrive; what matters is that the
	// channel closes rather than leaking the goroutine.
	done := make(chan struct{})
	var got []StreamEvent
	go func() { got = drain(t, ch); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("channel did not close after cancellation")
	}

	// And that the reason travels. A stream that simply stops is
	// indistinguishable from one that finished, so the caller cannot tell a
	// cancelled turn from a complete one — which is the difference between
	// "the user interrupted" and "the model had nothing more to say".
	last := got[len(got)-1]
	if last.Type != EventError || last.Err == nil || last.Err.Class != ErrClassCanceled {
		t.Errorf("the stream ended with %v / %+v; want an EventError classed %v", last.Type, last.Err, ErrClassCanceled)
	}
}

// Cancelling and the transport closing become ready at the same moment, and a
// select picks between ready cases at random. So a cancelled stream reported
// itself as a truncated one about one run in twenty.
//
// It is not a cosmetic misfile. Decide sends ErrClassTransport to
// DecisionRetry and ErrClassCanceled to DecisionSilent, so the loop answered a
// user's interrupt by calling the provider again — spending money and time on
// work that had just been called off, in the one case where the user was
// watching.
//
// Deterministic, where it used to repeat sixty times and hope.
//
// The repetition was racing its own setup: the pump could drain the single
// frame, find no terminal event and emit a truncated-stream error *before*
// cancel() ran on the test goroutine. In that interleaving the product is
// right — nobody had cancelled anything yet — and the test failed roughly one
// run in fifty, on whichever machine scheduled it that way.
//
// Two invariants, each pinned without a race. A context already cancelled, and
// a stream held open until the cancel has certainly happened.
func TestCancellationIsNeverReportedAsATruncatedStream(t *testing.T) {
	t.Run("cancelled before the stream starts", func(t *testing.T) {
		r, _ := registry(t, `{"choices":[{"delta":{"content":"a"}}]}`)
		p, _ := r.Resolve("MiniMax-M3", "")

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		ch, err := p.Stream(ctx, request())
		if err != nil {
			t.Fatal(err)
		}
		assertCancelled(t, drain(t, ch))
	})

	t.Run("cancelled while the stream is in flight", func(t *testing.T) {
		held := &heldTransport{released: make(chan struct{})}
		r := NewRegistry()
		r.RegisterTransport(held)
		if err := r.RegisterFamily(MiniMaxM3{}); err != nil {
			t.Fatal(err)
		}
		p, _ := r.Resolve("MiniMax-M3", "")

		ctx, cancel := context.WithCancel(context.Background())
		ch, err := p.Stream(ctx, request())
		if err != nil {
			t.Fatal(err)
		}
		// The transport is still holding, so the pump cannot have finished.
		cancel()
		close(held.released)

		assertCancelled(t, drain(t, ch))
	})
}

func assertCancelled(t *testing.T, got []StreamEvent) {
	t.Helper()
	if len(got) == 0 {
		t.Fatal("the stream produced nothing at all, not even a terminal event")
	}
	last := got[len(got)-1]
	if last.Type != EventError || last.Err == nil {
		t.Fatalf("ended with %v; a cancelled stream must say so", last.Type)
	}
	if last.Err.Class != ErrClassCanceled {
		t.Fatalf("classed a cancellation as %v — the loop retries that class, "+
			"so an interrupt becomes another call to the provider", last.Err.Class)
	}
}

// heldTransport keeps a stream open until the test lets it go, which is what
// makes "cancelled in flight" a fact rather than a hope.
type heldTransport struct{ released chan struct{} }

func (heldTransport) Name() string { return TransportOpenAI }

func (h *heldTransport) Do(ctx context.Context, _ WireRequest) (<-chan WireEvent, error) {
	out := make(chan WireEvent)
	go func() {
		defer close(out)
		select {
		case <-ctx.Done():
		case <-h.released:
		}
	}()
	return out, nil
}

// The fix guarded the channel-closed path and left the two error paths beside
// it. Cancelling closes the transport, so the read fails with whatever the
// operating system says about a socket that went away — "use of closed network
// connection", or on Darwin something else again — and none of those satisfy
// errors.Is(err, context.Canceled).
//
// So an interrupt arrived as a transport error, which Decide sends to retry:
// the loop answered the user calling it off by calling the provider again.
// Deterministic here; the surviving flake in the repeated test was this.
func TestATransportErrorDuringCancellationIsACancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	for _, err := range []error{
		errors.New("use of closed network connection"),
		errors.New("read: connection reset by peer"),
		io.ErrUnexpectedEOF,
	} {
		got := classify(ctx, err)
		if got.Class != ErrClassCanceled {
			t.Errorf("with the context cancelled, %v classed as %v — the loop retries that class",
				err, got.Class)
		}
		if got.Retryable {
			t.Errorf("%v was marked retryable while the context was cancelled", err)
		}
	}
}

// And with a live context the same errors are still transport, or cancellation
// handling has swallowed every real failure.
func TestATransportErrorWithALiveContextStaysTransport(t *testing.T) {
	got := classify(context.Background(), errors.New("connection reset by peer"))
	if got.Class != ErrClassTransport {
		t.Errorf("a real transport failure classed as %v", got.Class)
	}
	if !got.Retryable {
		t.Error("a real transport failure is not retryable, so a blip ends the turn")
	}
}
