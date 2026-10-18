package tools

import (
	"strings"
)

// The git a child of an orchestrator may not run: every subcommand that
// writes to the repository it shares with the person. The harness commits,
// merges and pushes for the child (orchestration .p §10); a child that did it
// itself would be work no one measured reaching a branch.
//
// The words are matched in the shell text, conservatively: a command split on
// the shell's separators, each part read past its environment assignments and
// wrappers, and git read past its global options. An alias or a script that
// runs git still meets the sandbox, which leaves the common .git outside what
// a child may write.
var gitWrites = map[string]bool{
	"commit": true, "merge": true, "rebase": true, "push": true, "reset": true, "checkout": true,
	"switch": true, "stash": true, "config": true, "worktree": true, "tag": true, "branch": true,
}

// gitTakesValue are git's global options whose value is the next word.
var gitTakesValue = map[string]bool{
	"-C": true, "-c": true, "--git-dir": true, "--work-tree": true, "--namespace": true,
	"--super-prefix": true, "--config-env": true,
}

// wrappers run the command that follows them.
var wrappers = map[string]bool{
	"sudo": true, "env": true, "command": true, "exec": true, "time": true, "nohup": true,
	"nice": true, "xargs": true, "builtin": true,
}

// DenyGitWrite is a Bash.Deny: why command is refused when it runs git that
// writes, and empty otherwise.
func DenyGitWrite(command string) string {
	for _, part := range shellParts(command) {
		if sub := gitSubcommand(part); gitWrites[sub] {
			return "git " + sub + " is refused in a child of an orchestrator, without asking anyone: " +
				"git that writes is the harness's. It commits what you changed, by name, at the end of each of your turns, " +
				"and merges and pushes for you. Leave the changes in the worktree and end the turn."
		}
	}
	return ""
}

// shellParts splits a command line on what separates one command from the
// next, and gives each one's words, quotes taken off.
func shellParts(command string) [][]string {
	sep := strings.NewReplacer(";", "\n", "&", "\n", "|", "\n", "(", "\n", ")", "\n", "`", "\n", "$(", "\n",
		"{", "\n", "}", "\n")
	var parts [][]string
	for _, line := range strings.Split(sep.Replace(command), "\n") {
		var words []string
		for _, w := range strings.Fields(line) {
			if w = strings.Trim(w, `"'\`); w != "" {
				words = append(words, w)
			}
		}
		if len(words) > 0 {
			parts = append(parts, words)
		}
	}
	return parts
}

// gitSubcommand is the git subcommand words run, or empty when they do not
// run git.
func gitSubcommand(words []string) string {
	i := 0
	for i < len(words) && (wrappers[words[i]] || isAssignment(words[i]) || (i > 0 && strings.HasPrefix(words[i], "-") && wrappers[words[i-1]])) {
		i++
	}
	if i >= len(words) || (words[i] != "git" && !strings.HasSuffix(words[i], "/git")) {
		return ""
	}
	for i++; i < len(words); i++ {
		w := words[i]
		if !strings.HasPrefix(w, "-") {
			return w
		}
		if gitTakesValue[w] {
			i++
		}
	}
	return ""
}

// isAssignment is NAME=value before a command.
func isAssignment(w string) bool {
	name, _, ok := strings.Cut(w, "=")
	if !ok || name == "" {
		return false
	}
	for _, r := range name {
		if !(r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}
