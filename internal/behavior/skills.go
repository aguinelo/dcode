package behavior

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// Skill is a body of guidance that only matters in a specific context.
//
// The index line goes in the prefix; the body is appended when a trigger fires.
// Loading every body into the prefix is the fastest route to a system prompt of
// tens of thousands of tokens, paid on every turn of every session, with the
// model's attention spread across all of it (RN-7).
type Skill struct {
	Name      string
	WhenToUse string
	Triggers  []string
	Body      string
	Path      string
}

// SkillsDirName is the directory searched under each root.
const SkillsDirName = "skills"

// MaxWhenToUse caps the index line. One line is the contract: the index is paid
// for on every turn, so a skill that describes itself in a paragraph is
// charging every session for a context most of them never enter.
const MaxWhenToUse = 120

// LoadSkills reads skills from a set of directories, later directories winning
// on a name collision.
//
// A skill is either `<dir>/<name>.md` or `<dir>/<name>/SKILL.md`. The second
// form exists so a skill can carry files beside it.
//
// A bad skill file never stops the product. It used to: a real skill from the
// ecosystem this format came from — `web-design-engineer`, 455 characters of
// `description` where the cap is 120 — made LoadSkills return an error, which
// app.go propagated, which made dcode exit 1 in that workspace, `--dump-prompt`
// included. `.dcode/skills/` arrives by `git clone`, so one file in a cloned
// repository decided whether the binary ran at all.
//
// The caps are right and being fatal was not, which is the rule the rest of
// this package already follows: the index cap announces what it left out rather
// than truncating in silence, and an over-size instruction is trimmed with a
// notice. Only a directory that cannot be read is still an error, because that
// is the machine failing rather than a file being wrong.
func LoadSkills(dirs []string, maxBytes int) ([]Skill, []Notice, error) {
	byName := map[string]Skill{}
	var notices []Notice

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		for _, e := range entries {
			path := filepath.Join(dir, e.Name())
			if e.IsDir() {
				path = filepath.Join(path, "SKILL.md")
				if _, err := os.Stat(path); err != nil {
					continue
				}
			} else if !strings.HasSuffix(e.Name(), ".md") {
				continue
			}

			data, err := os.ReadFile(path)
			if err != nil {
				notices = append(notices, Notice{Path: path, Reason: err.Error()})
				continue
			}
			// Skipped rather than cut. A body truncated at the cap is guidance
			// that stops mid-sentence, and guidance that is absent and said to
			// be absent is the better of the two.
			if maxBytes > 0 && len(data) > maxBytes {
				notices = append(notices, Notice{Path: path, Reason: fmt.Sprintf(
					"%d bytes, over the %d byte limit for a skill; not loaded", len(data), maxBytes)})
				continue
			}
			s, note, err := ParseSkill(string(data), path)
			if note != nil {
				notices = append(notices, *note)
			}
			if err != nil {
				notices = append(notices, Notice{Path: path, Reason: err.Error()})
				continue
			}
			if s.Name == "" {
				base := strings.TrimSuffix(e.Name(), ".md")
				if e.IsDir() {
					base = e.Name()
				}
				s.Name = base
			}
			byName[s.Name] = s
		}
	}

	out := make([]Skill, 0, len(byName))
	for _, s := range byName {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	sort.Slice(notices, func(i, j int) bool { return notices[i].Path < notices[j].Path })
	return out, notices, nil
}

// ParseSkill reads frontmatter and body. Pure over its input.
//
// The error and the notice are different answers. An error means this file
// cannot be a skill at all and the caller skips it; a notice means it was
// loaded with something changed, and the only such case is a when-to-use line
// trimmed to the cap.
func ParseSkill(text, path string) (Skill, *Notice, error) {
	s := Skill{Path: path}
	body := text

	if strings.HasPrefix(text, "---\n") {
		rest := text[4:]
		end := strings.Index(rest, "\n---")
		if end < 0 {
			return s, nil, fmt.Errorf("behavior: %s has an unterminated frontmatter block", path)
		}
		front := rest[:end]
		body = strings.TrimPrefix(rest[end+4:], "\n")

		for _, line := range strings.Split(front, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			colon := strings.IndexByte(line, ':')
			if colon < 0 {
				continue
			}
			key := strings.TrimSpace(line[:colon])
			val := strings.Trim(strings.TrimSpace(line[colon+1:]), `"'`)
			switch key {
			case "name":
				s.Name = val
			case "when_to_use", "description", "whenToUse":
				s.WhenToUse = val
			case "triggers":
				s.Triggers = splitList(val)
			}
		}
	}

	s.Body = strings.TrimSpace(body)
	if s.Body == "" {
		return s, nil, fmt.Errorf("behavior: %s has no body, so there is nothing to load", path)
	}
	if s.WhenToUse == "" {
		return s, nil, fmt.Errorf(
			"behavior: %s has no `when_to_use` line. Without it the skill cannot be indexed, "+
				"and an unindexed skill is one the model never learns exists", path)
	}
	// Trimmed, not refused. The line is the index economy and the cap stands;
	// what the cap cannot justify is throwing away a working body because the
	// sentence describing it was written for a product with no cap.
	if len(s.WhenToUse) > MaxWhenToUse {
		full := len(s.WhenToUse)
		s.WhenToUse = trimToWord(s.WhenToUse, MaxWhenToUse)
		return s, &Notice{Path: path, Reason: fmt.Sprintf(
			"`when_to_use` was %d characters and the index cap is %d; the line was trimmed. "+
				"It goes in the prefix of every turn, so it has to be one line",
			full, MaxWhenToUse)}, nil
	}
	return s, nil, nil
}

// trimToWord cuts at the last space before the cap, so the index line ends on a
// word rather than inside one. A line that ends mid-word reads as corruption
// and invites the reader to go looking for what went wrong.
func trimToWord(s string, max int) string {
	if len(s) <= max {
		return s
	}
	// The ellipsis is part of the line and so part of the budget. Cutting to
	// the cap and then appending it produces a line over the cap, which is the
	// same class of mistake as the one this function exists to stop.
	const ellipsis = "…"
	room := max - len(ellipsis)
	cut := s[:room]
	if i := strings.LastIndexByte(cut, ' '); i > room/2 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, " ,;:-") + ellipsis
}

func splitList(v string) []string {
	v = strings.Trim(strings.TrimSpace(v), "[]")
	var out []string
	for _, part := range strings.Split(v, ",") {
		if p := strings.Trim(strings.TrimSpace(part), `"'`); p != "" {
			out = append(out, strings.ToLower(p))
		}
	}
	return out
}

// Index renders the prefix entries, one line each.
func Index(skills []Skill) []SkillIndexEntry {
	return IndexCapped(skills, DefaultMaxIndex)
}

// DefaultMaxIndex caps how many skills reach the prefix.
//
// The index line is paid for on every turn of every session, so an unbounded
// index is an unbounded tax: a directory grown to two hundred skills quietly
// adds two hundred lines to every prompt, and the cost arrives as a slow bill
// rather than as an error.
const DefaultMaxIndex = 64

// IndexCapped is Index with an explicit ceiling.
//
// Over the cap it keeps the first n by name and says how many it left out. It
// does not truncate in silence: a skill missing from the index is one the model
// never learns exists, and discovering that from behaviour costs far more than
// reading one line.
func IndexCapped(skills []Skill, max int) []SkillIndexEntry {
	out := make([]SkillIndexEntry, 0, len(skills))
	for _, s := range skills {
		out = append(out, SkillIndexEntry{Name: s.Name, WhenToUse: s.WhenToUse})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	if max > 0 && len(out) > max {
		dropped := len(out) - max
		out = out[:max:max]
		out = append(out, SkillIndexEntry{
			Name: "…",
			WhenToUse: fmt.Sprintf("%d more skills are installed and not listed here; the index is capped at %d.",
				dropped, max),
		})
	}
	return out
}

// Match selects the skills whose triggers fire for a task. Deterministic: the
// same text always loads the same bodies, which is what keeps a replayed
// session identical to the live one.
//
// An explicit `triggers` list is matched as a phrase. Without one, the
// when-to-use line is matched on its own significant words, and two conditions
// have to hold together: two distinct hits, and at least one of them on a word
// that belongs to this skill and to no other in the index.
//
// The second condition is what the first alone could not do. Two skills that
// both say "projeto" and "versão" are not told apart by those words, so a task
// mentioning both was loading both bodies. Requiring a discriminating hit also
// keeps neighbours in one domain reachable — they share the domain word and
// each still has its own.
func Match(task string, skills []Skill) []Skill {
	lower := strings.ToLower(task)
	words := significantWords(lower)
	own := discriminating(skills)

	var out []Skill
	for _, s := range skills {
		if strings.Contains(lower, strings.ToLower("/"+s.Name)) {
			out = append(out, s)
			continue
		}
		if len(s.Triggers) > 0 {
			for _, t := range s.Triggers {
				if strings.Contains(lower, t) {
					out = append(out, s)
					break
				}
			}
			continue
		}
		hits, distinct := 0, false
		for w := range significantWords(strings.ToLower(s.WhenToUse)) {
			if _, ok := words[w]; !ok {
				continue
			}
			hits++
			if _, ok := own[s.Name][w]; ok {
				distinct = true
			}
		}
		if hits >= 2 && distinct {
			out = append(out, s)
		}
	}
	return out
}

// discriminating maps each skill to the significant words of its when-to-use
// line that no other skill's when-to-use carries.
//
// A single installed skill discriminates by everything it says, which is the
// right answer: with no neighbour there is nothing to be confused with.
func discriminating(skills []Skill) map[string]map[string]struct{} {
	seen := map[string]int{}
	per := make(map[string]map[string]struct{}, len(skills))
	for _, s := range skills {
		w := significantWords(strings.ToLower(s.WhenToUse))
		per[s.Name] = w
		for word := range w {
			seen[word]++
		}
	}
	out := make(map[string]map[string]struct{}, len(skills))
	for name, w := range per {
		own := map[string]struct{}{}
		for word := range w {
			if seen[word] == 1 {
				own[word] = struct{}{}
			}
		}
		out[name] = own
	}
	return out
}

// RenderSkill wraps a loaded body so the model can tell it from the user
// speaking, exactly as a reminder is wrapped.
func RenderSkill(s Skill) string {
	return "<skill name=\"" + s.Name + "\">\n" + s.Body + "\n</skill>"
}

// stopWords are the words too common to carry meaning in a trigger match.
//
// Both languages this product is written in, because the list used to hold only
// one of them. `quando`, `projeto` and `estiver` counted as significant, two of
// them were enough, and a Portuguese task about nothing in particular pulled
// whole skill bodies into the turn — while the same sentence in English was
// protected by `when` and `that` being on the list.
//
// A per-language list only ever covers the languages on it. What covers the
// rest is the `triggers` field, which is matched as a phrase and never goes
// through here.
var stopWords = map[string]struct{}{}

func init() {
	for _, w := range strings.Fields(`about after also because been before being
between both could does each from have here into just like made make more most
much must only other over same should some such than that their them then there
these they this those through under were what when where which while will with
would your`) {
		stopWords[w] = struct{}{}
	}
	for _, w := range strings.Fields(`agora ainda algum alguma antes apenas
aquele aquela assim ates cada coisa como dele dela deles delas depois dessa
desse desta deste enquanto entao entre esse essa esta este estar estava esteja
estiver estou fazer feito ficar foram forem isso isto mais menos mesmo muito
nada nossa nosso onde outra outro para pela pelo pode podem pois porque pouco
pronta pronto qual quando quanto seja sempre sendo ser sobre sua suas teve tudo
voce`) {
		stopWords[w] = struct{}{}
	}
}

func significantWords(s string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, f := range strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if len(f) < 4 {
			continue
		}
		if _, skip := stopWords[f]; skip {
			continue
		}
		out[f] = struct{}{}
	}
	return out
}
