package provider

// anError is the body of an Anthropic error, the same shape inside a stream as
// in a failed response: {"type":"error","error":{"type":...,"message":...}}.
type anError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// classifyStreamError turns an error frame into the class its HTTP twin would
// get, keeping the provider's message.
//
// Ignored, the frame left the stream to end without a terminal event, which is
// classed transport: an overload was retried by luck of the mapping, every
// other failure was retried by mistake, and in both the provider's own account
// of what went wrong was dropped.
//
// overloaded_error is the in-stream form of 529 and goes where 529 goes —
// provider, retried with backoff. An unknown type is the other side's failure
// as far as this side can tell, and goes there too, named.
func classifyStreamError(e *anError) *ProviderError {
	if e == nil {
		return &ProviderError{Class: ErrClassProvider, Message: "error frame without an error", Retryable: true}
	}
	msg := sanitize(e.Type + ": " + e.Message)
	switch e.Type {
	case "invalid_request_error":
		if overflowsTheWindow(e.Message) {
			return &ProviderError{Class: ErrClassContextSize, Message: msg, Retryable: true}
		}
		return &ProviderError{Class: ErrClassBadRequest, Message: msg}
	case "request_too_large":
		return &ProviderError{Class: ErrClassContextSize, Message: msg, Retryable: true}
	case "authentication_error", "permission_error":
		return &ProviderError{Class: ErrClassAuth, Message: msg}
	case "billing_error":
		return &ProviderError{Class: ErrClassQuota, Message: msg}
	case "not_found_error":
		return &ProviderError{Class: ErrClassBadRequest, Message: msg}
	case "rate_limit_error":
		// No header inside a stream, so no RetryAfter: the wait falls back to
		// the policy's own backoff.
		return &ProviderError{Class: ErrClassRateLimit, Message: msg, Retryable: true}
	default: // overloaded_error, api_error, timeout_error, and what is not known yet
		return &ProviderError{Class: ErrClassProvider, Message: msg, Retryable: true}
	}
}

// anStopDetails is what a refusal says about itself. Informational only: the
// stop reason is what decides, and the details can be null on a refusal.
type anStopDetails struct {
	Type        string `json:"type"`
	Category    string `json:"category"`
	Explanation string `json:"explanation"`
}

// refuse ends a stream whose model declined to answer.
//
// The answer arrives as a 200 that ends with stop_reason "refusal", and read as
// any other stop it was a clean done: the turn took the refusal, or the half an
// answer streamed before a classifier stopped it, for the answer, and the done
// check passed it.
//
// An error, not a done with a flag, so every consumer that already stops on an
// error stops on this one — the summariser, a delegated child, the loop — and
// none can read a refused answer as a finished one by not knowing a new field.
// The pending tool calls are dropped, not flushed: a call the model was making
// when it was stopped is not one it asked for. And the usage rides on the event,
// because the refused answer is billed like any other.
func (d *anthropicDecoder) refuse(e anEvent) []StreamEvent {
	// Flushed as far as the decoder is concerned: message_stop still follows,
	// and it must not release the calls this refusal dropped.
	d.terminated, d.flushed = true, true
	ev := StreamEvent{Type: EventError, Err: refusalError(e.Delta.StopDetails)}
	if u, ok := d.usage(e.Usage); ok {
		ev.Usage = &u
	}
	return []StreamEvent{ev}
}

// refusalError names the refusal and keeps what the provider said about it.
// Never retryable: the same request is the same request refused.
func refusalError(sd *anStopDetails) *ProviderError {
	msg := "the model refused to answer"
	if sd != nil {
		if sd.Category != "" {
			msg += " (category " + sd.Category + ")"
		}
		if sd.Explanation != "" {
			msg += ": " + sd.Explanation
		}
	}
	return &ProviderError{Class: ErrClassRefused, Message: sanitize(msg)}
}
