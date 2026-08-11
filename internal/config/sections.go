package config

import (
	"fmt"
	"strings"
)

// Sections is a TOML file read as arbitrary sections rather than as the
// configuration schema.
//
// Order is kept because the caller turns sections into an ordered list, and a
// list that reshuffles between runs is one nobody can diff.
type Sections struct {
	Order  []string
	Values map[string]map[string]string
}

// ParseSections reads the same TOML subset ParseTOML reads, but WITHOUT
// checking names against KnownKeys.
//
// The relaxation is deliberate and narrow. KnownKeys is the configuration
// schema, and an unknown key there is a typo that would otherwise be silently
// ignored — the most frustrating class of configuration bug. A file that
// declares a project's own criteria has no fixed schema to check against: the
// section names ARE the data.
//
// What is not relaxed is the credential rule. A secret parked in a section this
// reader does not recognise is still refused, for the same reason as in
// ParseTOML (RN-3), and the check runs on the key's own name.
func ParseSections(data, origin string) (Sections, error) {
	out := Sections{Values: map[string]map[string]string{}}
	section := ""

	for i, raw := range strings.Split(data, "\n") {
		line := strings.TrimSpace(stripComment(raw))
		if line == "" {
			continue
		}
		where := fmt.Sprintf("%s:%d", origin, i+1)

		if strings.HasPrefix(line, "[") {
			if !strings.HasSuffix(line, "]") {
				return Sections{}, fmt.Errorf("%s: unterminated section header", where)
			}
			section = strings.TrimSpace(line[1 : len(line)-1])
			if section == "" {
				return Sections{}, fmt.Errorf("%s: empty section name", where)
			}
			if strings.HasPrefix(section, "[") {
				return Sections{}, fmt.Errorf("%s: arrays of tables are not supported", where)
			}
			out.touch(section)
			continue
		}

		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			return Sections{}, fmt.Errorf("%s: expected `key = value`", where)
		}
		name := strings.TrimSpace(line[:eq])
		if name == "" {
			return Sections{}, fmt.Errorf("%s: empty key", where)
		}
		val, err := parseValue(strings.TrimSpace(line[eq+1:]))
		if err != nil {
			return Sections{}, fmt.Errorf("%s: %w", where, err)
		}
		if credentialKey.MatchString(name) {
			return Sections{}, credentialError(name, origin)
		}
		out.touch(section)
		out.Values[section][name] = val
	}
	return out, nil
}

func (s *Sections) touch(name string) {
	if _, ok := s.Values[name]; !ok {
		s.Values[name] = map[string]string{}
		s.Order = append(s.Order, name)
	}
}
