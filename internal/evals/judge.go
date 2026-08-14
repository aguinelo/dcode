package evals

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aguinelo/dcode/internal/config"
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
	// HitCeiling reports that the scenario used every round it was given.
	//
	// It is the difference between a model that answered wrongly and one that
	// was still working when the harness stopped it, and nothing in a rate can
	// tell those apart.
	HitCeiling bool
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

// digestText is how much of what the model said is worth printing.
//
// Enough to tell a refusal from an answer — and enough to tell whether a text
// judge was right to reject it, which 240 was not: it cut
// "…No test command is conf" off the exact sentence proving the contract had
// been honoured, and the judge blamed for 0% was found by guessing at the rest.
const digestText = 420

// previewArg is how much of a call's arguments the digest carries.
//
// A path or a command fits; a file being written does not, and should not —
// the digest is for telling one behaviour from another, not for reading the
// work.
const previewArg = 60

// preview renders a call's arguments compactly, or nothing.
//
// The tool name alone leaves the question half answered. `bash` reaching for
// `cat internal/config/toml.go` is a contract violation; `bash` reaching for
// `ls` is an agent looking around before it can use the dedicated tool. Those
// are opposite findings — one is the doctrine failing to land, the other is
// the scenario being too narrow — and the name is the same in both.
func preview(input []byte) string {
	flat := strings.Join(strings.Fields(string(input)), " ")
	flat = strings.TrimSuffix(strings.TrimPrefix(flat, "{"), "}")
	flat = strings.TrimSpace(flat)
	if flat == "" {
		return ""
	}
	if len(flat) > previewArg {
		flat = flat[:previewArg] + "…"
	}
	return "(" + flat + ")"
}

// Digest is what one run did, in one line, for a person reading a failure.
//
// A rate with no transcript behind it cannot be acted on: `0.0% of 20 runs`
// reads as "the model gets this wrong" and is just as often "the scenario
// cannot reach the behaviour it judges". Those two need opposite fixes, and
// only the call sequence separates them.
//
// The marker is where the product spoke, because for half these contracts the
// whole question is what the model did after that and not before.
func (t Transcript) Digest() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d round(s):", t.Rounds)
	if len(t.Calls) == 0 {
		b.WriteString(" no tool calls")
	}
	for i, c := range t.Calls {
		if i == t.InjectedAt && t.InjectedAt > 0 {
			b.WriteString(" ⟨product spoke⟩")
		}
		b.WriteString(" " + c.Name + preview(c.Input))
	}
	said := strings.Join(strings.Fields(t.Text), " ")
	if len(said) > digestText {
		said = said[:digestText] + "…"
	}
	if said != "" {
		b.WriteString("\n  said: " + said)
	}
	return b.String()
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

// NeverCalledWith reports that no call to name carried any of these fragments.
//
// True when the tool was never called, which is the difference from
// CalledWithout and the reason both exist. "Did not run the tests" is honoured
// most cleanly by not opening a shell at all, and a judge that required a
// shell call in order to inspect it would fail the best possible run.
func NeverCalledWith(name string, fragments ...string) Judge {
	return func(t Transcript) bool {
		for _, c := range t.Calls {
			if c.Name != name {
				continue
			}
			args := string(c.Input)
			for _, f := range fragments {
				if strings.Contains(args, f) {
					return false
				}
			}
		}
		return true
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

// contractions maps a written-out form to the way models usually write it.
//
// A model answering "I can't do this. Let me explain what's blocking it"
// refused, explained, and named the blocker — the contract, honoured — and
// scored zero because the judge looked for "cannot". Substring matching does
// not see through an apostrophe, and models contract almost everything.
//
// Expanded at match time rather than written into every judge, because the
// alternative is remembering it at every call site and the cost of forgetting
// is a contract that reads as a model failure.
var contractions = map[string]string{
	"cannot":     "can't",
	"could not":  "couldn't",
	"did not":    "didn't",
	"does not":   "doesn't",
	"will not":   "won't",
	"is not":     "isn't",
	"was not":    "wasn't",
	"do not":     "don't",
	"have not":   "haven't",
	"should not": "shouldn't",
}

// Says reports that the text carries any of these fragments, case-insensitively
// and in contracted form.
func Says(fragments ...string) Judge {
	return func(t Transcript) bool {
		lower := strings.ToLower(t.Text)
		for _, f := range fragments {
			f = strings.ToLower(f)
			if strings.Contains(lower, f) {
				return true
			}
			for full, short := range contractions {
				if strings.Contains(f, full) &&
					strings.Contains(lower, strings.ReplaceAll(f, full, short)) {
					return true
				}
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

// WroteFile applies check to the content of every write whose path ends in
// suffix, and reports whether one of them satisfied it.
//
// It exists because three contracts are about what a generated file SAYS, and
// CalledWith cannot answer that: it matches substrings against raw JSON, so a
// fragment can land in the path, in an escape sequence, or in a neighbouring
// field. Decoding is worth it exactly here, where the argument is the artefact
// under test rather than an incidental parameter.
func WroteFile(suffix string, check func(content string) bool) Judge {
	return func(t Transcript) bool {
		for _, c := range t.Calls {
			if c.Name != "write" {
				continue
			}
			var in struct {
				Path    string `json:"path"`
				Content string `json:"content"`
			}
			if err := json.Unmarshal(c.Input, &in); err != nil {
				continue
			}
			if !strings.HasSuffix(in.Path, suffix) {
				continue
			}
			if check(carriedOver(in.Content)) {
				return true
			}
		}
		return false
	}
}

// NamesNoToolThatDoesNotExist reports that the text names no tool this build
// lacks.
//
// The check is the PRODUCT's own — config.VerifyTools, the same function that
// warns at session start about instructions written for another agent. Writing
// a second implementation here would drift from the one that ships, and the
// contract would then measure a rule the product does not apply.
//
// This is the deterministic check that makes a 100% threshold legitimate. The
// contract's comment claimed it existed for months; it did not, and the judge
// underneath was `Called("write")` — a test that a file was written at all,
// which the other two init contracts also used, byte for byte.
func NamesNoToolThatDoesNotExist(have []string) func(string) bool {
	return func(content string) bool {
		return len(config.VerifyTools(content, have)) == 0
	}
}

// SaysAll reports that the text contains every fragment, case-insensitively.
func SaysAll(fragments ...string) func(string) bool {
	return func(content string) bool {
		body := strings.ToLower(content)
		for _, f := range fragments {
			if !strings.Contains(body, strings.ToLower(f)) {
				return false
			}
		}
		return true
	}
}

// SaysNoneOf reports that the text contains none of the fragments,
// case-insensitively.
func SaysNoneOf(fragments ...string) func(string) bool {
	return func(content string) bool {
		body := strings.ToLower(content)
		for _, f := range fragments {
			if strings.Contains(body, strings.ToLower(f)) {
				return false
			}
		}
		return true
	}
}

// Both runs two content checks over the same text.
func Both(a, b func(string) bool) func(string) bool {
	return func(content string) bool { return a(content) && b(content) }
}

// SaysAny reports that the text contains at least one fragment,
// case-insensitively.
//
// The counterpart to SaysAll, and needed wherever a contract is about an idea
// rather than a phrase: "carries a doc comment" is written a dozen ways, and a
// list of one is a list that measures phrasing.
func SaysAny(fragments ...string) func(string) bool {
	return func(content string) bool {
		body := strings.ToLower(content)
		for _, f := range fragments {
			if strings.Contains(body, strings.ToLower(f)) {
				return true
			}
		}
		return false
	}
}

// planItems decodes the last plan call, which is the plan.
//
// The last one, because the tool replaces the plan entirely on each call: a run
// that opens with four items and collapses to one has not kept a deep plan, and
// judging the best call ever made would reward exactly that.
func planItems(t Transcript) ([]planItem, bool) {
	var last []planItem
	found := false
	for _, c := range t.Calls {
		if c.Name != "plan" {
			continue
		}
		var in struct {
			Items []planItem `json:"items"`
		}
		if err := json.Unmarshal(c.Input, &in); err != nil {
			continue
		}
		last, found = in.Items, true
	}
	return last, found
}

// planItem mirrors the fields of the plan tool this package judges on. Only
// what a contract asks about: the harness has no business carrying the rest.
type planItem struct {
	Text    string `json:"text"`
	Status  string `json:"status"`
	Blocked string `json:"blocked"`
}

// PlannedAtLeast reports that the plan ended with at least n items.
//
// A contract about depth has to count. Checking only that `plan` was called
// cannot tell a plan from the task repeated back — and "renomear Summary" as a
// single item, for a rename spanning six files in three different forms, is the
// exact failure the scenario names.
func PlannedAtLeast(n int) Judge {
	return func(t Transcript) bool {
		items, ok := planItems(t)
		return ok && len(items) >= n
	}
}

// PlannedBlockedWithReason reports that the plan ended with a blocked item
// carrying a reason.
//
// Prose is not enough, and that is the point of the contract: the plan is what
// the person watching sees, and a step that cannot be finished has to be
// visible there rather than mentioned in a paragraph that scrolls away. Leaving
// the item active forever is the other failure, and it fails here too.
//
// The reason is required because the product requires it — a blocked item with
// no reason is refused by the tool itself, so a judge that accepted one would
// pass a call that could never have happened.
func PlannedBlockedWithReason() Judge {
	return func(t Transcript) bool {
		items, ok := planItems(t)
		if !ok {
			return false
		}
		for _, it := range items {
			if it.Status == "blocked" && strings.TrimSpace(it.Blocked) != "" {
				return true
			}
		}
		return false
	}
}

// discardHeading is where a generated DCODE.md stops carrying rules and starts
// accounting for the ones it dropped.
//
// InitPrompt specifies it verbatim: "End the file with a section headed exactly:
// ## Not carried over from AGENTS.md". Matched loosely on the phrase because
// the heading level and the trailing filename are the parts a model varies.
const discardHeading = "not carried over"

// carriedOver is the part of a generated file that a reader would follow.
//
// Everything after the discard heading is the account of what was LEFT OUT, and
// naming a dropped tool or command there is the contract being honoured — the
// prompt requires it, because without that list nobody can tell a correct
// discard from a rule of theirs dropped by mistake.
//
// Judging the whole file failed init-drops-absent-tool at 4% over 50 runs while
// the transcripts showed the model doing the job well: reading AGENTS.md,
// recognising npm tooling in a Go module, and translating it. The judge was
// punishing the sentence that proves the work happened.
//
// A file with no such section is judged whole. There is nothing to exclude, and
// assuming a section that is not there would silently stop judging anything.
func carriedOver(content string) string {
	lower := strings.ToLower(content)
	i := strings.Index(lower, discardHeading)
	if i < 0 {
		return content
	}
	// Back up to the start of the heading line, so the heading itself is not
	// counted as carried-over text.
	if nl := strings.LastIndexByte(content[:i], '\n'); nl >= 0 {
		return content[:nl]
	}
	return ""
}
