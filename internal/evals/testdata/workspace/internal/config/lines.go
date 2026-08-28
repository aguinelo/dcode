package config

import "strings"

// splitLines breaks a configuration file into its lines, dropping blanks and
// comments.
//
// A comment is a line whose first non-space character is `#`. Trailing
// comments are not stripped: a value may contain one, and guessing where it
// starts is how `password = a#b` loses half its password.
func splitLines(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

// cut splits a line on the first separator and reports whether it was there.
//
// The first, not the last: a value may contain the separator, and a key may
// not. Splitting on the last would read `model.name = a=b` as a key nobody
// declared.
func cut(line, sep string) (key, value string, ok bool) {
	i := strings.Index(line, sep)
	if i < 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+len(sep):]), true
}
