// Package loop is the turn cycle: it orchestrates the context engine, the
// provider, the tools and the sandbox. Everything else exists to serve it.
//
// It is also where the product gains or loses its character. A loop that aborts
// on the first tool error produces a brittle agent; one with no ceiling
// produces an expensive one; one that executes outside the boundary produces a
// dangerous one.
//
// Spec: docs/specs/architecture/agent-loop/202608072335-*.
package loop

import (
	"bytes"
	"encoding/json"
	"sort"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

// Limits bound a turn.
//
// The three are deliberately redundant, defence in depth. If you find yourself
// lowering MaxIterations to control cost, the real problem is usually the
// repeat detector switched off or a tool returning uninformative errors.
type Limits struct {
	// MaxIterations is the backstop. Zero means "use the family default",
	// because the work horizon is a property of the model: a cap sized for a
	// ten-file refactor truncates legitimate work on a long-horizon model.
	MaxIterations int
	// MaxIdenticalCalls is the real mechanism. A pathological loop shows up in
	// three repeats, long before any sane iteration cap.
	MaxIdenticalCalls int
	// MaxTurnTokens is a hard cost ceiling. Zero is unlimited, and that is the
	// default: cutting mid-turn leaves the workspace half-edited, which is a
	// worse state than the one it started in. The iteration cap already bounds
	// spend by construction.
	MaxTurnTokens int
}

// DefaultLimits returns the documented defaults. MaxIterations stays zero so
// the family supplies it.
func DefaultLimits() Limits {
	return Limits{MaxIdenticalCalls: 3}
}

// withFamily fills anything the caller left unset from the family's defaults.
func (l Limits) withFamily(familyMaxIterations int) Limits {
	if l.MaxIterations <= 0 {
		l.MaxIterations = familyMaxIterations
	}
	if l.MaxIterations <= 0 {
		l.MaxIterations = 50
	}
	if l.MaxIdenticalCalls < 0 {
		l.MaxIdenticalCalls = 0
	}
	return l
}

// IsRepeat reports whether the last n calls are identical.
//
// Identity is over the tool *and* its canonicalised input. That is what lets
// the detector coexist with recovery: retrying with a different approach is not
// an identical call, so it never trips, while the same failing command three
// times does.
func IsRepeat(calls []ce.ToolCall, n int) bool {
	if n <= 0 || len(calls) < n {
		return false
	}
	tail := calls[len(calls)-n:]
	first := fingerprint(tail[0])
	for _, c := range tail[1:] {
		if fingerprint(c) != first {
			return false
		}
	}
	return true
}

// fingerprint canonicalises a call so that reordering JSON keys does not slip
// past the detector. Without this, a model that emits the same arguments in a
// different order would loop forever.
func fingerprint(c ce.ToolCall) string {
	return c.Name + "\x00" + string(canonicalJSON(c.Input))
}

// canonicalJSON re-encodes with object keys sorted and whitespace removed.
// Falls back to the raw bytes when the input is not valid JSON: an unparseable
// input is still comparable, and refusing to fingerprint it would disable the
// detector exactly when the model is producing garbage.
func canonicalJSON(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return []byte("{}")
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return bytes.TrimSpace(raw)
	}
	return encodeSorted(v)
}

func encodeSorted(v any) []byte {
	var buf bytes.Buffer
	writeSorted(&buf, v)
	return buf.Bytes()
}

func writeSorted(buf *bytes.Buffer, v any) {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			kb, _ := json.Marshal(k)
			buf.Write(kb)
			buf.WriteByte(':')
			writeSorted(buf, t[k])
		}
		buf.WriteByte('}')
	case []any:
		buf.WriteByte('[')
		for i, e := range t {
			if i > 0 {
				buf.WriteByte(',')
			}
			writeSorted(buf, e)
		}
		buf.WriteByte(']')
	default:
		b, _ := json.Marshal(t)
		buf.Write(b)
	}
}
