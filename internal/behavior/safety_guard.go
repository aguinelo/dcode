package behavior

import (
	"regexp"
	"sort"
	"strings"
)

// safetyClaims are the shapes an instruction takes when it tries to move the
// one line that does not move.
//
// This list does NOT decide anything. Nothing is removed from the instruction
// and no behaviour changes because of a match — the guarantees are structural
// and elsewhere: the sandbox is enforced by the operating system, approval is
// consent the loop asks for, and `Safety` is not a field of DoctrineOverlay.
//
// What this does is make the attempt VISIBLE. RN-10 requires that such an
// instruction be ignored "e o fato é registrado — não silenciosamente
// descartado", and the second half was missing: an attempt nobody can see is an
// attempt nobody investigates.
//
// Because it decides nothing, a false positive costs a line of output rather
// than a lost rule. That asymmetry is what makes a text match acceptable here
// and unacceptable as a filter — the difference the translation change turns on
// too.
var safetyClaims = []struct {
	pattern *regexp.Regexp
	what    string
}{
	{regexp.MustCompile(`(?i)\b(approvals?|confirmations?)\s+(are|is)\s+(disabled|off|not required|unnecessary)`),
		"claims approval is not required"},
	{regexp.MustCompile(`(?i)\b(skip|bypass|ignore|disable|suppress)\s+(the\s+)?(approvals?|confirmations?|sandbox|permission checks?)`),
		"asks for approval or the sandbox to be bypassed"},
	{regexp.MustCompile(`(?i)\bnever\s+ask\s+(the\s+)?(user\s+)?(for\s+)?(permission|approval|confirmation)`),
		"asks that the user never be asked"},
	{regexp.MustCompile(`(?i)\b(you\s+)?(may|can|should)\s+(write|read)\s+(anywhere|outside\s+the\s+workspace)`),
		"claims the workspace boundary does not apply"},
	{regexp.MustCompile(`(?i)\b(without|no need for)\s+(reading|read).{0,20}\bfirst\b`),
		"asks for the read-before-edit invariant to be skipped"},
	{regexp.MustCompile(`(?i)\bsandbox\s+(is\s+)?(disabled|off|not\s+(active|enabled))`),
		"claims the sandbox is not in force"},
}

// SafetyClaims reports the places an instruction tries to loosen safety.
//
// The instruction is NOT modified and NOT dropped: the rest of it is legitimate
// and discarding a whole file over one sentence is the silent-filter failure
// this project refuses everywhere else. What is returned is what to say out
// loud.
func SafetyClaims(in []Instruction) []Notice {
	var out []Notice
	for _, ins := range in {
		seen := map[string]struct{}{}
		for _, c := range safetyClaims {
			m := c.pattern.FindString(ins.Text)
			if m == "" {
				continue
			}
			if _, dup := seen[c.what]; dup {
				continue
			}
			seen[c.what] = struct{}{}
			out = append(out, Notice{
				Path: scopeOf(ins),
				Reason: "ignored: " + c.what + " (" + strings.TrimSpace(m) + "). " +
					"Safety is not adjustable by instruction — the sandbox is enforced by the " +
					"operating system and approval is the user's to give.",
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// SkillSafetyClaims reports the places a skill reaches for the boundary.
//
// The difference from SafetyClaims is provenance, and it is the whole reason
// this one DECIDES while that one only reports.
//
// An instruction is the user's, or their project's. Dropping a whole file over
// one sentence would cost them a rule they wrote, so the asymmetry there runs
// the other way: a false positive costs a line of output, and reporting is
// enough.
//
// A skill is the least trusted text this product loads. It arrives by `git
// clone` into `.dcode/skills/`, or is downloaded from a stranger's repository,
// which is precisely what RN-11 calls "not the user" — and its body goes
// straight into the turn inside a <skill> block, unread by anyone. There a
// false positive costs one skill, which the person can see named and fix,
// while a false negative loads attacker-authored text into the model's context.
//
// Both halves are screened. The body is where a payload would sit, and the
// index line is loaded on every turn, so a harmless body under a line that asks
// for the boundary is the cheapest version of the attack.
func SkillSafetyClaims(s Skill) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, text := range []string{s.WhenToUse, s.Body} {
		for _, c := range safetyClaims {
			m := c.pattern.FindString(text)
			if m == "" {
				continue
			}
			if _, dup := seen[c.what]; dup {
				continue
			}
			seen[c.what] = struct{}{}
			out = append(out, c.what+" ("+strings.TrimSpace(m)+")")
		}
	}
	sort.Strings(out)
	return out
}

func scopeOf(ins Instruction) string {
	if ins.Scope != "" {
		return ins.Scope
	}
	return string(ins.Source)
}
