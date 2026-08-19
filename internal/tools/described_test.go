package tools

import (
	"strings"
	"testing"
)

// A tool must not deny a capability it offers.
//
// The schema says what the model MAY send; the description is what it reads to
// decide whether to send it. When the two disagree the description wins, because
// that is the sentence the model is reasoning over — so a capability the words
// deny is a capability that does not exist.
//
// This is the shape this repository keeps finding in itself, inverted: not
// something declared that nobody reads, but something built that the words deny.
// It cost a whole measured run. Asked to write five independent files — the
// easiest possible case for dividing work — the model delegated none of them,
// because `explore` still said "it can only read" while its schema had been
// offering `owns` for three pull requests.
//
// The guard is deliberately narrow. A first version asserted that every schema
// field appears in the description, and it fired on fourteen fields across most
// tools: the model reads the per-field descriptions inside the schema too, so an
// unmentioned field is not an invisible one. What is actually harmful is the
// contradiction, and only `explore` makes a claim about its own limits.
func TestTheDelegationToolDoesNotDenyWhatItOffers(t *testing.T) {
	described := strings.ToLower(Explore{}.Description())

	for _, denial := range []string{"can only read", "no editing"} {
		if strings.Contains(described, denial) {
			t.Errorf("explore offers owns and still says %q", denial)
		}
	}
	if !strings.Contains(described, "owns") {
		t.Error("explore offers owns and never says so; the model reads this sentence, not the schema")
	}
}
