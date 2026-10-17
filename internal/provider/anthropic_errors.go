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
