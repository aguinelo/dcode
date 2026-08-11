package behavior

import (
	"fmt"
	"strings"
)

// Formulation is how a model family prefers a rule to be worded.
//
// RN-8 draws the line and it is worth stating plainly: the RULE is single and
// lives in this spec; the FORMULATION belongs to the family. Two families do
// not answer the same sentence the same way — one prefers marked-up structure,
// another plain prose — and a family that needs to change the *rule* rather
// than the wording is a sign that the rule is wrong, or that the model is not
// supportable.
//
// So this interface can only change how a section is delimited. There is no
// method here that could add, remove or reword a rule, and that is deliberate:
// the way this abstraction stays honest is by not being able to express the
// thing it must not do.
type Formulation interface {
	// Family names the family, so an assembled prompt can be traced back to
	// the formulation that produced it.
	Family() string
	// Section renders one titled block. An empty title is the leading block
	// that carries no heading.
	Section(title, body string) string
}

// markdown is the default formulation: headings, as most models expect.
type markdown struct{ family string }

func (m markdown) Family() string { return m.family }

func (markdown) Section(title, body string) string {
	if strings.TrimSpace(body) == "" {
		return ""
	}
	if title == "" {
		return body + "\n"
	}
	return "## " + title + "\n\n" + body + "\n"
}

// tagged wraps each block in a named tag.
//
// Anthropic's own guidance is that Claude follows structure delimited this way
// more reliably than by heading, and the effect is strongest exactly where it
// matters here — a long system prompt with several distinct concerns, which is
// what this prefix is.
type tagged struct{ family string }

func (t tagged) Family() string { return t.family }

func (tagged) Section(title, body string) string {
	if strings.TrimSpace(body) == "" {
		return ""
	}
	if title == "" {
		return body + "\n"
	}
	tag := strings.ToLower(strings.ReplaceAll(title, " ", "_"))
	return fmt.Sprintf("<%s>\n%s\n</%s>\n", tag, body, tag)
}

// FormulationFor picks the wording for a family.
//
// The choice lives here rather than in the provider package on purpose. If each
// family carried its own wording, the wording would drift into the rule — a
// family would grow a sentence, then a caveat, then an exception, and two
// models would end up behaving differently while the spec said one thing. Here
// the whole surface is a delimiter.
//
// An unknown family gets markdown. It is the safe default: a prompt that reads
// slightly less well is recoverable, and refusing to assemble one is not.
func FormulationFor(family string) Formulation {
	switch strings.ToLower(strings.TrimSpace(family)) {
	case "claude":
		return tagged{family: "claude"}
	case "":
		return markdown{family: "unknown"}
	default:
		return markdown{family: strings.ToLower(strings.TrimSpace(family))}
	}
}
