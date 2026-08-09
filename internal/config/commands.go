package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Command is a user-defined prompt fragment invoked as `/<name>`.
//
// It is a prompt and nothing else. A command that could run something would be
// a second, undeclared tool surface with none of the sandbox or approval
// machinery pointed at it — the boundary would be bypassed by a markdown file.
type Command struct {
	Name        string
	Description string
	Body        string
	Path        string
	Source      Source
}

// CommandsDirName is the directory searched under each root.
const CommandsDirName = "commands"

// CommandSet is the resolved set of commands, keyed by name.
type CommandSet struct {
	Commands map[string]Command
	// Collisions records project commands that shadowed a user command, so the
	// override is reportable. Silent shadowing is how a user ends up debugging
	// the wrong file.
	Collisions []string
}

// Names returns the command names in a stable order.
func (s CommandSet) Names() []string {
	out := make([]string, 0, len(s.Commands))
	for n := range s.Commands {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// DiscoverCommands loads user commands then project commands.
//
// Project wins: the repository is the more specific context, and someone who
// checked a command into a project meant it to apply there.
func DiscoverCommands(roots Roots, workspace string, maxBytes int) (CommandSet, error) {
	set := CommandSet{Commands: map[string]Command{}}

	for _, d := range []struct {
		dir    string
		source Source
	}{
		{filepath.Join(roots.Config, CommandsDirName), SourceUser},
		{filepath.Join(workspace, ".dcode", CommandsDirName), SourceProject},
	} {
		found, err := loadCommandDir(d.dir, d.source, maxBytes)
		if err != nil {
			return set, err
		}
		for _, c := range found {
			if prev, ok := set.Commands[c.Name]; ok {
				set.Collisions = append(set.Collisions,
					fmt.Sprintf("/%s from %s overrides %s", c.Name, c.Path, prev.Path))
			}
			set.Commands[c.Name] = c
		}
	}
	sort.Strings(set.Collisions)
	return set, nil
}

func loadCommandDir(dir string, source Source, maxBytes int) ([]Command, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var out []Command
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if maxBytes > 0 && len(data) > maxBytes {
			return nil, fmt.Errorf("config: %s is larger than the %d byte limit for a command", path, maxBytes)
		}
		c, err := ParseCommand(string(data), path)
		if err != nil {
			return nil, err
		}
		if c.Name == "" {
			// The filename is the obvious fallback, and it means a command
			// works without frontmatter at all.
			c.Name = strings.TrimSuffix(e.Name(), ".md")
		}
		c.Source = source
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// ParseCommand reads the frontmatter and the body. Pure over its input.
func ParseCommand(text, path string) (Command, error) {
	c := Command{Path: path}
	body := text

	if strings.HasPrefix(text, "---\n") {
		rest := text[4:]
		end := strings.Index(rest, "\n---")
		if end < 0 {
			return c, fmt.Errorf("config: %s has an unterminated frontmatter block", path)
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
				return c, fmt.Errorf("config: %s has a frontmatter line that is not `key: value`: %q", path, line)
			}
			key := strings.TrimSpace(line[:colon])
			val := strings.Trim(strings.TrimSpace(line[colon+1:]), `"'`)
			switch key {
			case "name":
				c.Name = strings.TrimPrefix(val, "/")
			case "description":
				c.Description = val
			}
		}
	}

	c.Body = strings.TrimSpace(body)
	if c.Body == "" {
		return c, fmt.Errorf("config: %s has no body, so there is nothing to send", path)
	}
	return c, nil
}

// ArgsPlaceholder is substituted with whatever followed the command name.
const ArgsPlaceholder = "$ARGUMENTS"

// Expand renders a command into the text the user is treated as having typed.
//
// Deterministic and side-effect free: no I/O, no substitution of anything but
// the argument string. A command that could read a file or run a process would
// be an execution path with no approval in front of it.
func Expand(c Command, args string) (string, error) {
	if strings.TrimSpace(c.Body) == "" {
		return "", fmt.Errorf("config: /%s has an empty body", c.Name)
	}
	args = strings.TrimSpace(args)
	if strings.Contains(c.Body, ArgsPlaceholder) {
		return strings.ReplaceAll(c.Body, ArgsPlaceholder, args), nil
	}
	if args == "" {
		return c.Body, nil
	}
	// Appending rather than dropping: the user typed those words meaning them
	// to reach the model, and a command author who never used the placeholder
	// did not decide otherwise.
	return c.Body + "\n\n" + args, nil
}

// SplitInvocation splits `/name rest` into its parts. Returns ok=false for
// anything that is not a slash command.
func SplitInvocation(input string) (name, args string, ok bool) {
	s := strings.TrimSpace(input)
	if !strings.HasPrefix(s, "/") || len(s) == 1 {
		return "", "", false
	}
	s = s[1:]
	if i := strings.IndexAny(s, " \t\n"); i >= 0 {
		return s[:i], strings.TrimSpace(s[i+1:]), true
	}
	return s, "", true
}

// ---------- instruction outside the frozen chain ----------

// OutOfChain reports an instruction file in a directory the session touched
// that was not part of the chain frozen at session creation.
//
// This is the only mechanism that satisfies both constraints at once: the
// user's instruction is not ignored, and the immutability of the prompt prefix
// survives. What it finds is appended as a reminder, never prefixed (RN-6).
func OutOfChain(touched string, chain []string) (path string, found bool) {
	dir, err := filepath.Abs(touched)
	if err != nil {
		return "", false
	}
	if fi, err := os.Stat(dir); err == nil && !fi.IsDir() {
		dir = filepath.Dir(dir)
	}

	known := make(map[string]struct{}, len(chain))
	for _, c := range chain {
		if abs, err := filepath.Abs(c); err == nil {
			known[abs] = struct{}{}
		}
	}

	// Highest precedence first, so the one file reported is the one that would
	// have won had it been discovered in time.
	for i := len(InstructionNames) - 1; i >= 0; i-- {
		candidate := filepath.Join(dir, InstructionNames[i])
		if _, ok := known[candidate]; ok {
			continue
		}
		if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
			return candidate, true
		}
	}
	return "", false
}
