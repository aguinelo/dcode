// Package provider is the layer between dcode and the language models.
//
// It has two orthogonal axes, and keeping them apart is the point:
//
//   - Transport is the wire format (openai, anthropic). Reusable across
//     families, carries no thresholds.
//   - Family is the adaptation — system prompt shape, tool schema, edit
//     strategy. Carries the measured behavioral thresholds and turn limits.
//
// "OpenAI-compatible" describes serialization, not behavior. Two models behind
// the same endpoint can have wildly different tool-calling quality, so treating
// a wire format as a family would apply one model's measured thresholds to
// another and ship unvalidated behavior that looks validated.
//
// MiniMax M3 is why this is not theoretical: it speaks both dialects, so one
// axis would mean duplicating the family, thresholds included.
//
// Spec: docs/specs/architecture/provider-adapter/202608072334-*.
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

// Limits are the turn defaults a family declares. The loop reads them instead
// of carrying a fixed number, because the work horizon is a property of the
// model: a model trained for long-horizon loops needs a higher iteration cap
// than one tuned for short tasks, and one global number serves both badly.
type Limits struct {
	MaxIterations   int
	MaxOutputTokens int
}

// WireRequest is a family-serialized request, opaque to the transport beyond
// its envelope.
type WireRequest struct {
	Model  string
	Body   json.RawMessage
	Stream bool
}

// WireEvent is one raw frame off the wire, before the family decodes it.
type WireEvent struct {
	// Data is the frame payload. Empty Data with Done set marks end of stream.
	Data []byte
	Done bool
	// Err is a transport-level failure: connection, timeout, status code.
	Err error
}

// Transport is the wire format. It knows nothing about prompts, tool schemas or
// thresholds; a `if family == X` inside a transport means the axes collapsed
// back into one, and the symptom only shows up at the third family.
type Transport interface {
	Name() string
	Do(ctx context.Context, wire WireRequest) (<-chan WireEvent, error)
}

// Decoder turns one stream's raw frames into neutral events.
//
// It is stateful and single-use: one per stream, never shared. Decode returns
// zero or more events for a frame — zero when the frame only carried a
// fragment, several when the end of the stream flushes calls that were being
// assembled.
type Decoder interface {
	Decode(ev WireEvent) ([]StreamEvent, error)

	// Close reports what the end of the stream means.
	//
	// A dialect can finish a message and then keep talking: the OpenAI one
	// repeats finish_reason and attaches the token usage to the last frame, so
	// terminating on the first finish throws the accounting away. The decoder
	// therefore holds the terminal event until the transport says there is
	// nothing more coming.
	Close() []StreamEvent
}

// Family is the adaptation layer.
type Family interface {
	Name() string
	// Transports lists compatible wire formats, most preferred first.
	Transports() []string
	// Models lists the model-name prefixes this family claims.
	Models() []string
	Window(model string) (int, error)
	DefaultLimits() Limits
	// Encode takes the transport name because a family that speaks two
	// dialects serializes differently into each. That parameter is exactly
	// what a single-axis design could not express.
	Encode(req Request, transport string) (WireRequest, error)
	// NewDecoder builds a decoder for one stream.
	//
	// Decoding cannot be a pure function of one frame: a tool call's arguments
	// arrive split across frames, so the whole call only exists once the stream
	// says it is finished. The decoder is what holds that partial state, and it
	// belongs to the family because how a call is split is dialect-specific.
	NewDecoder(tools []ce.ToolDef) Decoder
}

// Request is the neutral call. Only contextengine types cross this boundary.
type Request struct {
	Model     string
	Messages  []ce.Message
	Tools     []ce.ToolDef
	MaxTokens int
}

// Provider is the composition of a transport and a family. Built by the
// registry, never by hand.
type Provider interface {
	Family() Family
	Transport() Transport
	Window(model string) (int, error)
	Limits() Limits
	Stream(ctx context.Context, req Request) (<-chan StreamEvent, error)
}

type composed struct {
	family    Family
	transport Transport
}

func (c *composed) Family() Family               { return c.family }
func (c *composed) Transport() Transport         { return c.transport }
func (c *composed) Window(m string) (int, error) { return c.family.Window(m) }
func (c *composed) Limits() Limits               { return c.family.DefaultLimits() }

// Stream sends the request and returns neutral events. The channel always
// closes, and always after exactly one terminal event.
func (c *composed) Stream(ctx context.Context, req Request) (<-chan StreamEvent, error) {
	wire, err := c.family.Encode(req, c.transport.Name())
	if err != nil {
		return nil, fmt.Errorf("encode for %s/%s: %w", c.family.Name(), c.transport.Name(), err)
	}
	raw, err := c.transport.Do(ctx, wire)
	if err != nil {
		return nil, err
	}

	out := make(chan StreamEvent, 16)
	// One decoder per stream: it holds the half-assembled tool calls, and
	// sharing it between streams would splice one turn's arguments into
	// another's.
	go c.pump(ctx, raw, c.family.NewDecoder(req.Tools), out)
	return out, nil
}

// pump translates raw frames into neutral events and guarantees the terminal
// invariant: exactly one EventDone or EventError, never both, never neither.
// A stream that ends without a terminal event hangs the loop forever.
func (c *composed) pump(ctx context.Context, raw <-chan WireEvent, dec Decoder, out chan<- StreamEvent) {
	defer close(out)

	terminal := false
	emit := func(ev StreamEvent) bool {
		if terminal {
			return false
		}
		if ev.Type == EventDone || ev.Type == EventError {
			terminal = true
		}
		// A send that can succeed immediately always does. Both cases of a
		// select are chosen at random when both are ready, and on cancellation
		// both ARE ready — which dropped the cancellation event itself often
		// enough to matter, closing the stream with no terminal event at all
		// and leaving the caller unable to tell an interrupt from a clean end.
		select {
		case out <- ev:
			return true
		default:
		}
		// Only now is giving up meaningful: the buffer is full, so the consumer
		// has genuinely stopped draining.
		select {
		case out <- ev:
			return true
		case <-ctx.Done():
			return false
		}
	}

	for {
		select {
		case <-ctx.Done():
			emit(canceledEvent(ctx))
			return
		case wev, open := <-raw:
			if !open {
				// The decoder gets the last word: a dialect that ends by
				// simply stopping still finished cleanly, and only it knows
				// whether what it saw amounts to a complete message.
				for _, ev := range dec.Close() {
					if !emit(ev) {
						return
					}
				}
				// Otherwise the transport closed without saying why. Treat it
				// as a truncated stream rather than a clean finish: a silent
				// success here would hand the loop a half-formed turn.
				//
				// Unless the reason is that WE stopped it. Cancelling closes the
				// transport too, so both this and ctx.Done() become ready at the
				// same instant and the select above picks between them at
				// random. Reporting a user's interrupt as a transport failure is
				// not a cosmetic misfile: Decide sends transport to retry and
				// cancellation to silence, so the loop answered an interrupt by
				// calling the provider again.
				if !terminal {
					if ctx.Err() != nil {
						emit(canceledEvent(ctx))
						return
					}
					emit(errorEvent(&ProviderError{
						Class:     ErrClassTransport,
						Message:   "stream ended without a terminal event",
						Retryable: true,
					}))
				}
				return
			}
			if wev.Err != nil {
				emit(errorEvent(classify(wev.Err)))
				return
			}
			evs, err := dec.Decode(wev)
			if err != nil {
				emit(errorEvent(classify(err)))
				return
			}
			for _, ev := range evs {
				if ev.Type == "" {
					continue // frame carried nothing the loop cares about
				}
				if !emit(ev) {
					return
				}
				if terminal {
					return
				}
			}
		}
	}
}

// Registry resolves a model name to a Provider.
type Registry struct {
	transports map[string]Transport
	families   []Family
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{transports: map[string]Transport{}}
}

// RegisterTransport adds a wire format.
func (r *Registry) RegisterTransport(t Transport) { r.transports[t.Name()] = t }

// RegisterFamily adds an adaptation layer. Returns an error when its model
// prefixes overlap one already registered: an ambiguous prefix would silently
// resolve to whichever family happened to be added first.
func (r *Registry) RegisterFamily(f Family) error {
	for _, existing := range r.families {
		for _, a := range existing.Models() {
			for _, b := range f.Models() {
				if strings.HasPrefix(a, b) || strings.HasPrefix(b, a) {
					return fmt.Errorf(
						"provider: model prefix %q of family %q overlaps %q of family %q",
						b, f.Name(), a, existing.Name())
				}
			}
		}
	}
	r.families = append(r.families, f)
	return nil
}

// Resolve composes a Provider for model. transportOverride selects a dialect;
// empty uses the family's preferred one.
//
// An unknown model is an error listing the available families. It never falls
// back to a generic family, because a silent default ships bad tool-calling
// with no signal that anything is wrong.
func (r *Registry) Resolve(model, transportOverride string) (Provider, error) {
	fam := r.familyFor(model)
	if fam == nil {
		return nil, fmt.Errorf("provider: no family claims model %q; available: %s",
			model, strings.Join(r.FamilyNames(), ", "))
	}

	name := transportOverride
	if name == "" {
		if len(fam.Transports()) == 0 {
			return nil, fmt.Errorf("provider: family %q declares no transport", fam.Name())
		}
		name = fam.Transports()[0]
	} else if !contains(fam.Transports(), name) {
		return nil, fmt.Errorf("provider: family %q does not speak transport %q; it speaks: %s",
			fam.Name(), name, strings.Join(fam.Transports(), ", "))
	}

	t, ok := r.transports[name]
	if !ok {
		return nil, fmt.Errorf("provider: transport %q is not registered", name)
	}
	return &composed{family: fam, transport: t}, nil
}

// ResolveFamily composes a Provider with an explicitly named family, bypassing
// prefix resolution. This is the escape hatch for an unsupported model; the
// caller is expected to warn that no thresholds were measured for it.
func (r *Registry) ResolveFamily(familyName, transportOverride string) (Provider, error) {
	for _, f := range r.families {
		if f.Name() == familyName {
			return r.resolveWith(f, transportOverride)
		}
	}
	return nil, fmt.Errorf("provider: no family named %q; available: %s",
		familyName, strings.Join(r.FamilyNames(), ", "))
}

func (r *Registry) resolveWith(fam Family, transportOverride string) (Provider, error) {
	name := transportOverride
	if name == "" {
		if len(fam.Transports()) == 0 {
			return nil, fmt.Errorf("provider: family %q declares no transport", fam.Name())
		}
		name = fam.Transports()[0]
	} else if !contains(fam.Transports(), name) {
		return nil, fmt.Errorf("provider: family %q does not speak transport %q; it speaks: %s",
			fam.Name(), name, strings.Join(fam.Transports(), ", "))
	}
	t, ok := r.transports[name]
	if !ok {
		return nil, fmt.Errorf("provider: transport %q is not registered", name)
	}
	return &composed{family: fam, transport: t}, nil
}

// familyFor returns the family claiming model.
//
// A first match is enough because RegisterFamily rejects overlapping prefixes,
// so at most one family can ever claim a given name. A longest-prefix tie-break
// would be unreachable code, and unreachable code that looks like a safety net
// is worse than none.
func (r *Registry) familyFor(model string) Family {
	f, _ := FamilyFor(model, r.families)
	return f
}

// FamilyFor reports which family claims a model.
//
// Exported because a caller may need the family *before* a provider exists —
// resolving a credential, for one, since the credential is what the transport
// is built with. Two implementations of this matching would eventually disagree
// about which key a model uses, which is a failure with no visible symptom
// until an auth error.
func FamilyFor(model string, families []Family) (Family, bool) {
	for _, f := range families {
		for _, p := range f.Models() {
			if strings.HasPrefix(model, p) {
				return f, true
			}
		}
	}
	return nil, false
}

// FamilyNames lists registered families, sorted for a stable error message.
func (r *Registry) FamilyNames() []string {
	out := make([]string, 0, len(r.families))
	for _, f := range r.families {
		out = append(out, f.Name())
	}
	sort.Strings(out)
	return out
}

// TransportNames lists registered transports, sorted.
func (r *Registry) TransportNames() []string {
	out := make([]string, 0, len(r.transports))
	for n := range r.transports {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func contains(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}
