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
		ConfirmWrite: []string{
			".git/**", ".dcode/**",
			// Secrets are confirmed on the way in as well as on the way out. A
			// list that guards reading and not writing protects the value and
			// not the file, and the file is what somebody else depends on.
			".env", ".env.*", "**/.env", "**/.env.*",
			// Lock files are the build's memory. Rewriting one is a change to
			// every machine that installs afterwards, and it is the change
			// least visible in a diff.
			"**/go.sum", "**/package-lock.json", "**/yarn.lock",
			"**/pnpm-lock.yaml", "**/Cargo.lock", "**/poetry.lock",
		},
		ConfirmRead: []string{
			".env", ".env.*", "**/.env", "**/.env.*",
			"**/*.pem", "**/*.key", "**/id_rsa", "**/id_ed25519",
			"**/.npmrc", "**/.netrc", "**/credentials", "**/credentials.*",
		},
		ConfirmCommand: destructiveCommands,
	}
}

// rulesDoc is the reasoning behind the command list, kept next to it because a
// list of patterns with no reasoning is a list that grows by superstition.
const rulesDoc = `The command rules are attention, not a boundary — they are
not a boundary and cannot be one. A command is text, and the same destruction
can always be spelled another way: through a script, an alias, a variable. What
the list buys is friction against the accident, which is what actually happens.
Containment is the sandbox, and only the sandbox.`

// destructiveCommands are the commands worth a question even when everything
// else is granted.
//
// Three kinds, and they are here for different reasons. Some destroy work that
// cannot be recovered from the repository — a forced push, a hard reset, a
// recursive delete. Others reach the outside irreversibly: publishing is not
// destruction, but it cannot be taken back either, and "I did not know it would
// publish" is the same afternoon lost.
//
// The third kind leaves the machine, and it is different in a way that matters:
// for the first two the sandbox still bounds where the damage lands, and for
// this one it bounds nothing. The question is not a second line of defence
// there — it is the only one.
//
// The spec has declared this list since before it existed. DefaultRules shipped
// none of it, so the promise of confirmation was made in documentation and
// nowhere else.
var destructiveCommands = []string{
	// Deleting, recursively and without asking.
	"*rm -rf*", "*rm -fr*", "*rm -r -f*", "*rm -f -r*",
	// Running as somebody else, which leaves the workspace by definition.
	"*sudo *", "*doas *",
	// Rewriting history somebody else may already have.
	"*git push*--force*", "*git push*-f *", "*git reset --hard*",
	"*git clean*-f*", "*git branch*-D*", "*git filter-branch*",
	// Publishing: not destruction, and just as irreversible.
	"*npm publish*", "*cargo publish*", "*gem push*", "*twine upload*",
	"*gh release create*", "*docker push*",
	// Writing straight to a device, or making a filesystem on one.
	"*mkfs*", "*dd if=*of=/dev/*", "*> /dev/sd*", "*> /dev/nvme*",
	// Fetching and running in one breath, which is trust with no reading.
	"*curl*|*sh*", "*curl*|*bash*", "*wget*|*sh*", "*wget*|*bash*",
	// Opening permissions on everything.
	"*chmod -R 777*", "*chmod 777 /*", "*chown -R*/*",
	// Leaving the machine, which is the one kind the sandbox cannot follow.
	//
	// Every other entry on this list destroys work or publishes irreversibly,
	// and containment still bounds WHERE it can happen. These happen somewhere
	// containment has no reach at all: `ssh host 'systemctl stop postgres'`
	// runs on a machine this process has no boundary on, and nothing can be
	// undone from here. So the question fires on the crossing itself rather
	// than on what is being asked over there — the far side cannot be read,
	// and pretending to judge it would be worse than admitting we cannot.
	//
	// Anchored at the start, and at the usual chaining, so `cat ~/.ssh/config`
	// is not a connection to anywhere. A command is text and this is attention,
	// not a boundary: attention that fires on every push is attention nobody
	// reads.
	"ssh *", "*&& ssh *", "*; ssh *", "*| ssh *",
	"scp *", "*&& scp *", "*; scp *",
	"rsync *:*",
	"kubectl exec*", "kubectl cp*", "kubectl port-forward*",
	"ansible*",
	"aws ssm start-session*", "aws ssm send-command*",
	"docker -H *", "docker --host *", "*DOCKER_HOST=*",
	// Taking the machine with it.
	"*shutdown*", "*reboot*", "*halt*", "*:(){*",
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
