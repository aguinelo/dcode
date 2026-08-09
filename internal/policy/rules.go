package policy

import (
	"strings"
)

// Rules raise a question the sandbox cannot.
//
// Everything inside the workspace is uniform to the sandbox, and it is right
// about that: a file in `src/` and a file in `.git/hooks/` are the same kind of
// write. They are not the same kind of *consequence*, and that is what these
// patterns are for.
//
// **They are attention, not containment.** A command pattern is trivially
// avoided — `bash -c`, an alias, a script — and a path pattern only sees the
// path a tool declares. What contains the agent is the sandbox; what these do is
// make a person look before something with a long tail happens. Reading them as
// a boundary is the one way to be worse off than not having them.
type Rules struct {
	// ConfirmWrite are paths that ask before being written.
	ConfirmWrite []string
	// ConfirmRead are paths that ask before being read, because reading sends
	// the contents to the model provider — off this machine.
	ConfirmRead []string
	// ConfirmCommand are shell commands that ask before running.
	ConfirmCommand []string
}

// DefaultRules is what someone gets without configuring anything.
//
// Short on purpose. Every entry earns its place by being different in *kind*
// from an ordinary source file, not merely important:
//
//   - `.git/**` — a write to `hooks/` runs on the next commit, outside the
//     sandbox, as the user. Deferred execution that escapes the boundary.
//   - `.dcode/**` — configures the agent. An agent that edits its own
//     configuration can widen its own reach.
//   - secrets — reading one sends it to the model provider, and the workspace
//     is exactly where a `.env` lives.
func DefaultRules() Rules {
	return Rules{
		ConfirmWrite: []string{".git/**", ".dcode/**"},
		ConfirmRead: []string{
			".env", ".env.*", "**/.env", "**/.env.*",
			"**/*.pem", "**/*.key", "**/id_rsa", "**/id_ed25519",
			"**/.npmrc", "**/.netrc",
		},
	}
}

// MatchPath reports the first pattern that claims a path.
//
// The path is workspace-relative with forward slashes, so a rule reads the same
// on every platform and cannot be dodged by an absolute spelling.
func (r Rules) MatchPath(rel string, write bool) (string, bool) {
	patterns := r.ConfirmRead
	if write {
		patterns = r.ConfirmWrite
	}
	return matchAny(patterns, normalise(rel))
}

// MatchCommand reports the first pattern that claims a command.
func (r Rules) MatchCommand(command string) (string, bool) {
	// A command is text, not a path: `/` in `rm -rf build/` is a character,
	// not a boundary, so `*` has to cross it here and must not in a path.
	return matchAnyText(r.ConfirmCommand, strings.TrimSpace(command))
}

func matchAny(patterns []string, s string) (string, bool) {
	return match(patterns, s, Glob)
}

func matchAnyText(patterns []string, s string) (string, bool) {
	return match(patterns, s, GlobText)
}

func match(patterns []string, s string, fn func(string, string) bool) (string, bool) {
	if s == "" {
		return "", false
	}
	for _, p := range patterns {
		if p = strings.TrimSpace(p); p != "" && fn(p, s) {
			return p, true
		}
	}
	return "", false
}

// normalise makes a path comparable: forward slashes, no leading `./`.
func normalise(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	p = strings.TrimPrefix(p, "./")
	return strings.TrimPrefix(p, "/")
}

// Glob reports whether a pattern claims a string.
//
// `*` matches within one segment, `**` across segments, `?` one character.
// Written here rather than taken from the standard library because
// `path.Match` has no `**`, and a rule that cannot say "anywhere under this
// directory" is a rule people work around by listing every depth.
func Glob(pattern, s string) bool {
	return globFrom(pattern, s, 0, 0, '/')
}

// GlobText matches without treating any character as a separator. It is what a
// command pattern uses, where a slash is part of an argument.
func GlobText(pattern, s string) bool {
	return globFrom(pattern, s, 0, 0, 0)
}

func globFrom(pattern, s string, pi, si int, sep byte) bool {
	for pi < len(pattern) {
		switch pattern[pi] {
		case '*':
			// `**` crosses separators; a single `*` stops at one.
			if pi+1 < len(pattern) && pattern[pi+1] == '*' {
				next := pi + 2
				// `**/` also matches zero directories, so `**/x` claims `x`.
				if sep != 0 && next < len(pattern) && pattern[next] == sep {
					if globFrom(pattern, s, next+1, si, sep) {
						return true
					}
				}
				for at := si; at <= len(s); at++ {
					if globFrom(pattern, s, next, at, sep) {
						return true
					}
				}
				return false
			}
			for at := si; at <= len(s); at++ {
				if sep != 0 && at > si && s[at-1] == sep {
					break
				}
				if globFrom(pattern, s, pi+1, at, sep) {
					return true
				}
			}
			return false

		case '?':
			if si >= len(s) || (sep != 0 && s[si] == sep) {
				return false
			}
			pi++
			si++

		default:
			if si >= len(s) || s[si] != pattern[pi] {
				return false
			}
			pi++
			si++
		}
	}
	return si == len(s)
}

// ---------- serialisation ----------
//
// A list of patterns is a list, and configuration carries it as one string so a
// single key maps to a single environment variable — the bijection the
// configuration surface rests on.

// ListSeparator joins patterns in a configuration value. A comma, because a
// glob never needs one and a path may well contain a colon or a space.
const ListSeparator = ","

// SplitList reads a configured list, dropping blanks.
func SplitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ListSeparator) {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// JoinList writes a list back.
func JoinList(patterns []string) string { return strings.Join(patterns, ListSeparator) }
