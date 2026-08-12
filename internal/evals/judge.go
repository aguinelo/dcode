package evals

import (
	"strings"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

// Transcript is what one measured run produced.
//
// Deliberately small. A judge that can reach for more than this starts encoding
// the implementation rather than the contract, and the contracts are about what
// the model did, not how it got there.
type Transcript struct {
	// Calls are every tool call across every round, in order.
	Calls []ce.ToolCall
	// Text is everything the model said, joined.
	Text string
	// Rounds is how many times it was asked.
	Rounds int
	// InjectedAt is how many calls had happened when the product spoke —
	// a tool error, a reminder. Zero when nothing was injected, which makes
	// "after the injection" mean "all of it" for a single-round scenario.
	//
	// Without it a contract about reacting to a reminder cannot be judged. The
	// model reads in the first round; if the judge only asks "was there a read
	// before an edit", a run that read once at the start and edited without
	// ever re-reading passes — and not re-reading is precisely the failure.
	InjectedAt int
}

// Since narrows a transcript to what happened after the product spoke.
func (t Transcript) Since() Transcript {
	at := t.InjectedAt
	if at < 0 || at > len(t.Calls) {
		at = 0
	}
	return Transcript{Calls: t.Calls[at:], Text: t.Text, Rounds: t.Rounds}
}

// SinceInjection applies inner to only what the model did after the reminder
// or the error reached it.
func SinceInjection(inner Judge) Judge {
	return func(t Transcript) bool { return inner(t.Since()) }
}

// Judge answers whether one run behaved as the contract says.
//
// A bool, not a score. A threshold is already the statistical half; letting a
// judge return "mostly" would put a second, invisible threshold underneath the
// declared one.
type Judge func(Transcript) bool

// Called reports a call to any of these tools.
func Called(names ...string) Judge {
	return func(t Transcript) bool {
		for _, c := range t.Calls {
			for _, n := range names {
				if c.Name == n {
					return true
				}
			}
		}
		return false
	}
}

// NotCalled is the negation, and it is not redundant: several contracts are
// about restraint, and "did not reach for bash" is the whole assertion.
func NotCalled(names ...string) Judge {
	inner := Called(names...)
	return func(t Transcript) bool { return !inner(t) }
}

// CalledWith reports a call to name whose arguments contain every fragment.
//
// Substring rather than a decoded field, because the fragment being looked for
// is usually a path or a value, and decoding would need a schema per tool that
// the harness has no business carrying.
func CalledWith(name string, fragments ...string) Judge {
	return func(t Transcript) bool {
		for _, c := range t.Calls {
			if c.Name != name {
				continue
			}
			args := string(c.Input)
			ok := true
			for _, f := range fragments {
				if !strings.Contains(args, f) {
					ok = false
					break
				}
			}
			if ok {
				return true
			}
		}
		return false
	}
}

// CalledWithout reports that every call to name avoided all the fragments.
//
// The inverse of CalledWith, and not the same as its negation: this is false
// when the tool was never called at all. A contract about which of two
// conflicting conventions was followed needs both halves — that the right one
// appears, and that the wrong one does not — and a model that wrote nothing
// must not satisfy the second half by having written nothing.
func CalledWithout(name string, fragments ...string) Judge {
	return func(t Transcript) bool {
		seen := false
		for _, c := range t.Calls {
			if c.Name != name {
				continue
			}
			seen = true
			args := string(c.Input)
			for _, f := range fragments {
				if strings.Contains(args, f) {
					return false
				}
			}
		}
		return seen
	}
}

// CalledBefore reports that a came before b, with both present.
//
// Order matters in exactly the contracts about re-reading: the read has to
// happen before the edit, and a run that edits then reads has not recovered.
func CalledBefore(a, b string) Judge {
	return func(t Transcript) bool {
		seenA := false
		for _, c := range t.Calls {
			switch c.Name {
			case a:
				seenA = true
			case b:
				if !seenA {
					return false
				}
				return true
			}
		}
		return false
	}
}

// Says reports that the text carries any of these fragments, case-insensitively.
func Says(fragments ...string) Judge {
	return func(t Transcript) bool {
		lower := strings.ToLower(t.Text)
		for _, f := range fragments {
			if strings.Contains(lower, strings.ToLower(f)) {
				return true
			}
		}
		return false
	}
}

// SaysNone is the negation, for the contracts about not claiming something.
func SaysNone(fragments ...string) Judge {
	inner := Says(fragments...)
	return func(t Transcript) bool { return !inner(t) }
}

// CallCount reports that the number of calls to name falls within a range.
// A max of zero means no upper bound.
func CallCount(name string, min, max int) Judge {
	return func(t Transcript) bool {
		n := 0
		for _, c := range t.Calls {
			if c.Name == name {
				n++
			}
		}
		return n >= min && (max == 0 || n <= max)
	}
}

// Distinct reports that repeated calls to name were not identical.
//
// This is what no-blind-retry measures and what the repeat detector cannot: the
// detector needs input to be byte-identical, and a model trying to escape
// varies the attempt each round while repeating the same conceptual error.
// Here, two attempts that differ only in whitespace count as the same attempt.
func Distinct(name string, want int) Judge {
	return func(t Transcript) bool {
		seen := map[string]struct{}{}
		for _, c := range t.Calls {
			if c.Name != name {
				continue
			}
			seen[strings.Join(strings.Fields(string(c.Input)), " ")] = struct{}{}
		}
		return len(seen) >= want
	}
}

// All requires every judge to hold.
func All(js ...Judge) Judge {
	return func(t Transcript) bool {
		for _, j := range js {
			if !j(t) {
				return false
			}
		}
		return true
	}
}

// Any requires at least one.
func Any(js ...Judge) Judge {
	return func(t Transcript) bool {
		for _, j := range js {
			if j(t) {
				return true
			}
		}
		return false
	}
}
