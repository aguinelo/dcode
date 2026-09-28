package provider

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

// Cancelled mid-answer, with the transport held open after its frames so the
// stream cannot end first.
//
// It replayed a whole answer, [DONE] included, and cancelled as soon as Stream
// returned. On a loaded machine the pump could take all three frames and end
// with done before cancel() ran on this goroutine — right, since nothing had
// been cancelled yet, and red about one run in a thousand: once on CI, on a
// pull request that touched no Go. The same race the test below had, taken out
// the same way.
func TestCancelClosesChannelWithCanceled(t *testing.T) {
	held := &heldTransport{frames: []string{
		`{"choices":[{"delta":{"content":"a"}}]}`,
		`{"choices":[{"delta":{"content":"b"}}]}`,
	}}
	r := NewRegistry()
	r.RegisterTransport(held)
	if err := r.RegisterFamily(MiniMaxM3{}); err != nil {
		t.Fatal(err)
	}
	p, _ := r.Resolve("MiniMax-M3", "")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := p.Stream(ctx, request())
	if err != nil {
		t.Fatal(err)
	}
	// In flight for certain: the answer has started, and the transport will
	// not end it on its own.
	if ev := <-ch; ev.Type != EventTextDelta {
		t.Fatalf("the stream opened with %v; want the first frame's text", ev.Type)
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
//
// Its frames go out first, one at a time and each racing the context, the way
// both real transports send them. Then it holds.
type heldTransport struct {
	frames   []string
	released chan struct{}
}

func (heldTransport) Name() string { return TransportOpenAI }

func (h *heldTransport) Do(ctx context.Context, _ WireRequest) (<-chan WireEvent, error) {
	out := make(chan WireEvent)
	go func() {
		defer close(out)
		for _, f := range h.frames {
			select {
			case <-ctx.Done():
				return
			case out <- WireEvent{Data: []byte(f)}:
			}
		}
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

// A cancelled stream ends canceled, whichever case the pump's select takes.
//
// With the context done, ctx.Done() is ready — and so is the transport's
// channel, whenever the transport still has something in it. select picks
// between ready cases at random, so a cancelled stream ended however that coin
// landed:
//
//   - on the close. Both real transports close once cancelled, and after a
//     frame that finished the answer but not its usage — MiniMax and OpenAI
//     send usage on a frame of its own — the decoder answers the close with
//     done;
//   - on a frame still waiting, decoded after the cancel. The transports here
//     cannot hand one over, since they send unbuffered and select on
//     ctx.Done() themselves, but Transport does not promise that.
//
// Done is not a cosmetic misfile: Decide sends canceled to silence, while a
// done stream is an answer the loop records and acts on, tool calls and all.
//
// Nothing is left to scheduling. The first frame is taken while the context is
// live, the cancel lands while it is decoded, and whatever the transport still
// has is already in its channel. What remains is the choice the language
// specifies as uniform: each run failed half the time, so a hundred passing by
// luck is a 2^-100 event.
func TestACancelledStreamEndsCanceledWhateverTheSelectPicks(t *testing.T) {
	for _, tc := range []struct {
		name   string
		frames []string
	}{
		{"the close, after a frame that finished the answer", []string{
			`{"choices":[{"delta":{"content":"a"},"finish_reason":"stop"}]}`,
		}},
		{"a frame still waiting", []string{
			`{"choices":[{"delta":{"content":"a"}}]}`,
			"[DONE]",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const runs = 100
			ended := map[string]int{}
			for i := 0; i < runs; i++ {
				ended[howItEnded(interruptedStream(t, tc.frames))]++
			}
			if ended[string(ErrClassCanceled)] != runs {
				t.Errorf("a cancelled stream ended %v across %d runs; want %v in every one",
					ended, runs, ErrClassCanceled)
			}
		})
	}
}

// interruptedStream runs the pump over a transport that already holds frames
// and its close, with the user interrupting while the first frame is decoded.
func interruptedStream(t *testing.T, frames []string) []StreamEvent {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	raw := make(chan WireEvent, len(frames))
	for _, f := range frames {
		raw <- WireEvent{Data: []byte(f)}
	}
	close(raw)

	out := make(chan StreamEvent, 16)
	dec := &interruptedWhileDecoding{Decoder: MiniMaxM3{}.NewDecoder(tools()), cancel: cancel}
	new(composed).pump(ctx, raw, dec, out)
	return drain(t, out)
}

// interruptedWhileDecoding is the user interrupting while a frame is decoded:
// after the pump has taken it, before it asks the transport for anything else.
type interruptedWhileDecoding struct {
	Decoder
	cancel context.CancelFunc
}

func (d *interruptedWhileDecoding) Decode(ev WireEvent) ([]StreamEvent, error) {
	d.cancel()
	return d.Decoder.Decode(ev)
}

// howItEnded names a stream's terminal event: its error class, or its type.
func howItEnded(got []StreamEvent) string {
	if len(got) == 0 {
		return "nothing"
	}
	last := got[len(got)-1]
	if last.Type == EventError && last.Err != nil {
		return string(last.Err.Class)
	}
	return string(last.Type)
}
