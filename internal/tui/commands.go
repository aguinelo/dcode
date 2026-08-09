package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aguinelo/dcode/internal/config"
)

// Builtin is a command the client itself answers.
//
// Built-in commands are client surface, not configuration: which ones exist is
// a product decision. Configuration owns only the discovery and expansion of
// user commands.
type Builtin struct {
	Name string
	Args string
	Help string
}

// Builtins is the whole built-in set, in the order `/help` prints them.
var Builtins = []Builtin{
	{"help", "", "shortcuts and commands"},
	{"init", "", "write DCODE.md for this workspace from what is already here"},
	{"clear", "", "end this session and open a fresh one"},
	{"plan", "[what to change]", "show the plan; with an argument, ask for a new one"},
	{"config", "<key>", "the effective value of a key and where it came from"},
	{"model", "<name>", "switch model — opens a new session, since the prefix changes"},
	{"resume", "[id]", "list sessions, or reattach to one"},
}

func isBuiltin(name string) bool {
	for _, b := range Builtins {
		if b.Name == name {
			return true
		}
	}
	return false
}

// Kind classifies a resolved input line.
type CommandKind int

const (
	// CmdText is ordinary input, or a user command already expanded into the
	// text the user would have typed.
	CmdText CommandKind = iota
	CmdBuiltin
	CmdUnknown
)

// Resolved is one input line, classified.
type Resolved struct {
	Kind CommandKind
	Name string
	Args string
	Text string
}

// ResolveInput classifies a line of input. Pure, and the only place that
// decides what a leading slash means.
//
// A built-in always wins over a user command of the same name: a user file
// cannot shadow `/config` into meaning something else, because the moment it
// could, no advice about dcode would be true of any particular installation.
func ResolveInput(input string, user config.CommandSet) Resolved {
	name, args, ok := config.SplitInvocation(input)
	if !ok {
		return Resolved{Kind: CmdText, Text: strings.TrimSpace(input)}
	}
	if isBuiltin(name) {
		return Resolved{Kind: CmdBuiltin, Name: name, Args: args}
	}
	if c, found := user.Commands[name]; found {
		text, err := config.Expand(c, args)
		if err != nil {
			return Resolved{Kind: CmdUnknown, Name: name, Args: args}
		}
		return Resolved{Kind: CmdText, Name: name, Args: args, Text: text}
	}
	return Resolved{Kind: CmdUnknown, Name: name, Args: args}
}

// ShadowedBuiltins lists user commands that collide with a built-in name, so
// the override that did not happen can be reported rather than puzzled over.
func ShadowedBuiltins(user config.CommandSet) []string {
	var out []string
	for _, name := range user.Names() {
		if isBuiltin(name) {
			out = append(out, fmt.Sprintf("/%s in %s is ignored: it is a built-in command",
				name, user.Commands[name].Path))
		}
	}
	sort.Strings(out)
	return out
}

// HelpText renders `/help`. Pure over the discovered command set.
func HelpText(user config.CommandSet) string {
	var b strings.Builder
	b.WriteString("Keys\n")
	for _, k := range [][2]string{
		{"enter", "send (queues while a turn is running)"},
		{"↑ ↓", "history on an empty line, otherwise move through the stream"},
		{"PgUp/PgDn", "scroll · Home and End jump to either end"},
		{"tab", "expand or collapse the selected entry"},
		{"esc", "close the expansion, then the selection"},
		{"^P", "show or hide the plan panel"},
		{"^A ^E ^W ^U ^K", "start, end, delete word, clear, cut to end"},
		{"^C", "interrupt the turn, or quit when idle"},
		{"^D", "quit"},
	} {
		fmt.Fprintf(&b, "  %-16s %s\n", k[0], k[1])
	}

	b.WriteString("\nCommands\n")
	for _, c := range Builtins {
		name := "/" + c.Name
		if c.Args != "" {
			name += " " + c.Args
		}
		fmt.Fprintf(&b, "  %-22s %s\n", name, c.Help)
	}

	names := user.Names()
	if len(names) > 0 {
		b.WriteString("\nYours\n")
		for _, n := range names {
			if isBuiltin(n) {
				continue
			}
			fmt.Fprintf(&b, "  %-22s %s\n", "/"+n, user.Commands[n].Description)
		}
	}

	b.WriteString("\nApprovals\n  [d] deny   [a] allow once   [A] allow for the session\n  Enter denies.\n")
	return strings.TrimRight(b.String(), "\n")
}

// InitPrompt is what `/init` sends.
//
// It is a turn, not a template: what belongs in DCODE.md depends on what the
// repository already says about itself, and only reading it can answer that.
const InitPrompt = `Write DCODE.md at the root of this workspace.

Read what is already here first — the README, any existing AGENTS.md or
CONTRIBUTING.md, the build and test configuration, and enough of the source to
see the conventions the code actually follows.

DCODE.md must hold only what an agent cannot derive by reading the code: how to
build, test and lint; conventions that are enforced but not obvious; anything
that would be a mistake to touch. Do not restate the directory structure and do
not pad it. If a file already covers something, reference it instead of copying
it.

If DCODE.md already exists, read it and propose the smallest change that brings
it up to date.`

// ReplanPrompt asks for a fresh plan.
func ReplanPrompt(what string) string {
	if strings.TrimSpace(what) == "" {
		return "Re-examine the plan and revise it if it no longer fits what you have learned."
	}
	return "Revise the plan to account for this: " + strings.TrimSpace(what)
}

// PlanText renders the full plan for `/plan` without an argument.
func PlanText(m Model) string {
	if len(m.Plan) == 0 {
		return "There is no plan yet."
	}
	var b strings.Builder
	for _, it := range m.Plan {
		fmt.Fprintf(&b, "%d. [%s] %s\n", it.ID, it.Status, it.Text)
		if it.Blocked != "" {
			fmt.Fprintf(&b, "     blocked: %s\n", it.Blocked)
		}
	}
	if s := m.PlanSummary(); s != "" {
		fmt.Fprintf(&b, "\n%s", s)
	}
	return strings.TrimRight(b.String(), "\n")
}
