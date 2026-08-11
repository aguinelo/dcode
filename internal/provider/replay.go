package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Transcript replay.
//
// Lives in package provider rather than a sub-package: it implements Transport,
// and the provider tests consume it, so a sub-package would need the parent
// while the parent's tests need the sub-package. Same-package keeps that
// straight without an interface indirection nobody else would use.
//
// This is what makes the whole layer testable with the network off, which is
// not a CI convenience — recorded frames are the only way to make stream
// parsing deterministic, and determinism is what the coverage gate rests on.

// Transcript is a recorded exchange: the frames a transport produced.
type Transcript struct {
	// Name identifies the fixture in failure output.
	Name string `json:"name"`
	// Transport is the wire format the frames were recorded from.
	Transport string `json:"transport"`
	// Frames are the raw payloads, in order.
	Frames []string `json:"frames"`
	// FailWith, when set, makes Do return this error instead of frames.
	FailWith *RecordedError `json:"fail_with,omitempty"`
}

// RecordedError reproduces a classified failure.
type RecordedError struct {
	Status int    `json:"status,omitempty"`
	Body   string `json:"body,omitempty"`
	Class  string `json:"class,omitempty"`
}

// ReplayTransport serves recorded frames. Deterministic by construction: the
// same transcript always produces the same sequence.
type ReplayTransport struct {
	name string
	tr   Transcript
	// Sent records what the family encoded, so tests can assert on the body
	// without a second mechanism.
	Sent []WireRequest
}

// NewReplayTransport builds a transport that serves tr under the wire name.
func NewReplayTransport(name string, tr Transcript) *ReplayTransport {
	return &ReplayTransport{name: name, tr: tr}
}

// LoadTranscript reads a transcript from disk.
func LoadTranscript(path string) (Transcript, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Transcript{}, fmt.Errorf("transcript %s: %w", path, err)
	}
	var tr Transcript
	if err := json.Unmarshal(b, &tr); err != nil {
		return Transcript{}, fmt.Errorf("transcript %s: %w", path, err)
	}
	return tr, nil
}

func (r *ReplayTransport) Name() string { return r.name }

// Do serves the recorded frames, honouring cancellation between each so
// interrupt behaviour is exercised by the same fixtures.
func (r *ReplayTransport) Do(ctx context.Context, wire WireRequest) (<-chan WireEvent, error) {
	r.Sent = append(r.Sent, wire)

	if fw := r.tr.FailWith; fw != nil {
		if fw.Status != 0 {
			if pe := ClassifyStatus(fw.Status, fw.Body, ""); pe != nil {
				return nil, pe
			}
		}
		return nil, &ProviderError{Class: ErrorClass(fw.Class), Message: fw.Body}
	}

	out := make(chan WireEvent)
	go func() {
		defer close(out)
		for _, f := range r.tr.Frames {
			select {
			case <-ctx.Done():
				return
			case out <- WireEvent{Data: []byte(f)}:
			}
		}
	}()
	return out, nil
}

// ParseSSE splits an SSE body into the data payloads of each event, which is
// the shape a recorded transcript stores.
func ParseSSE(body string) []string {
	var out []string
	sc := bufio.NewScanner(strings.NewReader(body))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		out = append(out, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
	}
	return out
}
