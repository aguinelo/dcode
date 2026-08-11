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
func LoadSkills(dirs []string, maxBytes int) ([]Skill, error) {
	byName := map[string]Skill{}

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
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
				return nil, err
			}
			if maxBytes > 0 && len(data) > maxBytes {
				return nil, fmt.Errorf("behavior: %s is larger than the %d byte limit for a skill", path, maxBytes)
			}
			s, err := ParseSkill(string(data), path)
			if err != nil {
				return nil, err
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
	return out, nil
}

// ParseSkill reads frontmatter and body. Pure over its input.
func ParseSkill(text, path string) (Skill, error) {
	s := Skill{Path: path}
	body := text

	if strings.HasPrefix(text, "---\n") {
		rest := text[4:]
		end := strings.Index(rest, "\n---")
		if end < 0 {
			return s, fmt.Errorf("behavior: %s has an unterminated frontmatter block", path)
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
		return s, fmt.Errorf("behavior: %s has no body, so there is nothing to load", path)
	}
	if s.WhenToUse == "" {
		return s, fmt.Errorf(
			"behavior: %s has no `when_to_use` line. Without it the skill cannot be indexed, "+
				"and an unindexed skill is one the model never learns exists", path)
	}
	if len(s.WhenToUse) > MaxWhenToUse {
		return s, fmt.Errorf(
			"behavior: %s has a `when_to_use` of %d characters, over the %d limit. "+
				"It goes in the prefix of every turn, so it has to be one line",
			path, len(s.WhenToUse), MaxWhenToUse)
	}
	return s, nil
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
// when-to-use line is matched on its own significant words, and two distinct
// hits are required — a single common word would load a skill into a task that
// merely mentioned it in passing.
func Match(task string, skills []Skill) []Skill {
	lower := strings.ToLower(task)
	words := significantWords(lower)

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
		hits := 0
		for w := range significantWords(strings.ToLower(s.WhenToUse)) {
			if _, ok := words[w]; ok {
				hits++
			}
		}
		if hits >= 2 {
			out = append(out, s)
		}
	}
	return out
}

// RenderSkill wraps a loaded body so the model can tell it from the user
// speaking, exactly as a reminder is wrapped.
func RenderSkill(s Skill) string {
	return "<skill name=\"" + s.Name + "\">\n" + s.Body + "\n</skill>"
}

// stopWords are the words too common to carry meaning in a trigger match.
var stopWords = map[string]struct{}{}

func init() {
	for _, w := range strings.Fields(`about after also because been before being
between both could does each from have here into just like made make more most
much must only other over same should some such than that their them then there
these they this those through under were what when where which while will with
would your`) {
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
