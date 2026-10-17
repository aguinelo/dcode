package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

// StreamEventType identifies what a StreamEvent carries.
type StreamEventType string

const (
	EventTextDelta StreamEventType = "text_delta"
	// EventReasoningDelta carries the model's thinking, which is not its
	// answer. A client may show it; it never enters the history, because a
	// model that reads its own reasoning back as something it said out loud
	// starts defending it.
	EventReasoningDelta StreamEventType = "reasoning_delta"
	// EventToolCallOpened says a tool call has begun arriving. The name and
	// the id are known long before the arguments finish, and throwing them
	// away is what makes a screen sit still while a model writes a file.
	EventToolCallOpened StreamEventType = "tool_call_opened"
	// EventToolCallProgress is how much of a call's arguments have arrived.
	EventToolCallProgress StreamEventType = "tool_call_progress"
	EventToolCall         StreamEventType = "tool_call"
	EventDone             StreamEventType = "done"
	EventError            StreamEventType = "error"
)

// StreamEvent is the neutral event the loop consumes.
//
// Ordering is guaranteed: zero or more TextDelta and ToolCall interleaved,
// terminated by exactly one Done or one Error. Never both, never neither.
//
// ToolCallOpened and ToolCallProgress may appear before the ToolCall they
// describe, and a consumer that ignores them still sees exactly the sequence it
// saw before — which is what lets a provider that cannot report them stay
// silent rather than pretend.
type StreamEvent struct {
	Type     StreamEventType
	Text     string
	ToolCall *ce.ToolCall
	// CallID and CallName identify a call still arriving; Bytes is how much of
	// its arguments have landed. Set only on the two opened/progress types.
	CallID   string
	CallName string
	Bytes    int
	// Usage is set on EventDone, and on an EventError whose answer was billed
	// all the same — a refusal.
	Usage *Usage
	Err   *ProviderError
	// Truncated is set on EventDone when the provider stopped because the
	// answer reached the output ceiling of the request, not because the model
	// finished. The answer that came before it is real and incomplete.
	Truncated bool
}

// Usage reports what the call consumed.
//
// CacheReadTokens is not decorative telemetry: it is the only direct measure
// that append-only context is working. If it stays near zero across a long
// session, the context engine has regressed and nothing else will say so.
type Usage struct {
	InputTokens      int
	OutputTokens     int
	CacheReadTokens  int
	CacheWriteTokens int
}

// ErrorClass is the stable classification the loop decides on.
type ErrorClass string

const (
	ErrClassAuth        ErrorClass = "auth"
	ErrClassQuota       ErrorClass = "quota"
	ErrClassRateLimit   ErrorClass = "rate_limit"
	ErrClassContextSize ErrorClass = "context_size"
	ErrClassBadRequest  ErrorClass = "bad_request"
	ErrClassToolSchema  ErrorClass = "tool_schema"
	ErrClassTransport   ErrorClass = "transport"
	ErrClassProvider    ErrorClass = "provider"
	ErrClassCanceled    ErrorClass = "canceled"
	// ErrClassRefused is the model declining to answer, on a response that
	// otherwise succeeded. Not the provider failing and not our request being
	// wrong: the same request sent again is the same request refused, so
	// nothing retries it, and what came before the refusal is not an answer.
	ErrClassRefused ErrorClass = "refused"
)

// ProviderError is a classified failure. A bare string would force the loop to
// guess between retry, wait, alternative and abort, and guessing is how an
// agent corrupts a file.
//
// Message never contains a credential.
type ProviderError struct {
	Class      ErrorClass
	Message    string
	RetryAfter time.Duration
	Retryable  bool
}

func (e *ProviderError) Error() string {
	if e.Message == "" {
		return string(e.Class)
	}
	return fmt.Sprintf("%s: %s", e.Class, e.Message)
}

// Decision is what the loop should do about an error.
type Decision string

const (
	DecisionAbort      Decision = "abort"
	DecisionRetry      Decision = "retry"
	DecisionWait       Decision = "wait"
	DecisionCompact    Decision = "compact"
	DecisionFeedToolIn Decision = "feed_to_model"
	DecisionSilent     Decision = "silent"
)

// Decide maps a class to the loop's action. Section 4 of the planning spec.
func Decide(e *ProviderError) Decision {
	if e == nil {
		return DecisionAbort
	}
	switch e.Class {
	case ErrClassRateLimit:
		return DecisionWait
	case ErrClassContextSize:
		return DecisionCompact
	case ErrClassToolSchema:
		// Not the loop's problem to solve: hand the error back to the model as
		// a tool result and let it correct itself.
		return DecisionFeedToolIn
	case ErrClassTransport, ErrClassProvider:
		return DecisionRetry
	case ErrClassCanceled:
		return DecisionSilent
	case ErrClassRefused:
		// Named rather than left to the default: a refusal is the one failure
		// a retry is guaranteed to repeat, and abort is the decision, not a
		// fallthrough.
		return DecisionAbort
	default: // auth, quota, bad_request
		return DecisionAbort
	}
}

func errorEvent(e *ProviderError) StreamEvent {
	return StreamEvent{Type: EventError, Err: e}
}

func canceledEvent(ctx context.Context) StreamEvent {
	msg := "canceled"
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		msg = "deadline exceeded"
	}
	return errorEvent(&ProviderError{Class: ErrClassCanceled, Message: msg})
}

// classify turns an arbitrary error into a ProviderError, preserving one that
// is already classified.
//
// The context is consulted because cancelling closes the transport, and what
// the read then reports is whatever the operating system says about a socket
// that went away — "use of closed network connection", a reset, an unexpected
// EOF. None of those satisfy errors.Is(err, context.Canceled), so an interrupt
// arrived as a transport error, and Decide sends transport to retry: the loop
// answered the user calling it off by calling the provider again.
//
// A live context still classes those as transport, or cancellation handling
// would swallow every real failure.
func classify(ctx context.Context, err error) *ProviderError {
	if err == nil {
		return &ProviderError{Class: ErrClassProvider, Message: "unknown error"}
	}
	var pe *ProviderError
	if errors.As(err, &pe) {
		return pe
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return &ProviderError{Class: ErrClassCanceled, Message: err.Error()}
	}
	if ctx != nil && ctx.Err() != nil {
		return &ProviderError{Class: ErrClassCanceled, Message: err.Error()}
	}
	return &ProviderError{Class: ErrClassTransport, Message: err.Error(), Retryable: true}
}

// ClassifyStatus maps an HTTP status to a class. Shared by every transport so
// the loop sees one classification regardless of dialect.
// retryAfter is the raw Retry-After header, empty when the response had none.
// Threaded through rather than attached by the caller afterwards so the
// invariant is structural: only the rate-limit branch can carry a wait, and no
// later caller can attach one to an error that has none to give.
func ClassifyStatus(status int, body, retryAfter string) *ProviderError {
	switch {
	case status == 401 || status == 403:
		return &ProviderError{Class: ErrClassAuth, Message: "authentication rejected"}
	case status == 402:
		return &ProviderError{Class: ErrClassQuota, Message: "quota or billing limit reached"}
	case status == 429:
		return &ProviderError{
			Class: ErrClassRateLimit, Message: "rate limited", Retryable: true,
			RetryAfter: RetryAfterOf(retryAfter, time.Now()),
		}
	case status == 413:
		return &ProviderError{Class: ErrClassContextSize, Message: "request exceeds the context window", Retryable: true}
	case status == 400 && overflowsTheWindow(body):
		return &ProviderError{Class: ErrClassContextSize, Message: sanitize(body), Retryable: true}
	case status >= 400 && status < 500:
		return &ProviderError{Class: ErrClassBadRequest, Message: sanitize(body)}
	case status >= 500:
		return &ProviderError{Class: ErrClassProvider, Message: sanitize(body), Retryable: true}
	}
	return nil
}

// windowOverflow is how a provider says, inside a 400, that the request does
// not fit the window. Anthropic never answers an overlong prompt with 413: it
// answers invalid_request_error with one of these, and as a bad_request the
// turn aborted on the one failure compaction exists to solve.
//
// Phrases rather than "any 400": every other 400 is still our bug, and a
// bad_request compacted and retried would hide it behind a smaller context.
var windowOverflow = []string{
	"prompt is too long",
	"exceed context limit",
}

func overflowsTheWindow(body string) bool {
	low := strings.ToLower(body)
	for _, p := range windowOverflow {
		if strings.Contains(low, p) {
			return true
		}
	}
	return false
}
