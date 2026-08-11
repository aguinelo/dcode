package tui

import (
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/protocol"
)

func userCommands(cmds ...config.Command) config.CommandSet {
	set := config.CommandSet{Commands: map[string]config.Command{}}
	for _, c := range cmds {
		set.Commands[c.Name] = c
	}
	return set
}

func TestResolveInputClassifiesALine(t *testing.T) {
	user := userCommands(config.Command{Name: "revisar", Body: "Review $ARGUMENTS."})

	for name, tc := range map[string]struct {
		in   string
		kind CommandKind
		text string
	}{
		"plain text":   {"fix the parser", CmdText, "fix the parser"},
		"builtin":      {"/help", CmdBuiltin, ""},
		"user command": {"/revisar main.go", CmdText, "Review main.go."},
		"unknown":      {"/nope", CmdUnknown, ""},
		"bare slash":   {"/", CmdText, "/"},
	} {
		got := ResolveInput(tc.in, user)
		if got.Kind != tc.kind {
			t.Errorf("%s: got kind %v", name, got.Kind)
		}
		if tc.text != "" && got.Text != tc.text {
			t.Errorf("%s: got %q want %q", name, got.Text, tc.text)
		}
	}
}

// A user file cannot shadow `/config` into meaning something else: the moment
// it could, no advice about dcode would be true of any installation.
func TestBuiltinsBeatUserCommandsAndTheShadowingIsReported(t *testing.T) {
	user := userCommands(config.Command{
		Name: "config", Body: "do something else", Path: "/home/u/commands/config.md",
	})
	if got := ResolveInput("/config model.name", user); got.Kind != CmdBuiltin {
		t.Fatalf("the built-in must win, got %v", got.Kind)
	}
	shadowed := ShadowedBuiltins(user)
	if len(shadowed) != 1 || !strings.Contains(shadowed[0], "config.md") {
		t.Errorf("the override that did not happen must be reported: %v", shadowed)
	}
	if len(ShadowedBuiltins(userCommands())) != 0 {
		t.Error("nothing to report")
	}
}

func TestResolveInputRejectsABrokenUserCommand(t *testing.T) {
	user := userCommands(config.Command{Name: "broken", Body: ""})
	if got := ResolveInput("/broken", user); got.Kind != CmdUnknown {
		t.Errorf("a command with nothing to send is not usable, got %v", got.Kind)
	}
}

func TestHelpTextListsEverySurface(t *testing.T) {
	user := userCommands(
		config.Command{Name: "revisar", Description: "reviews the diff", Body: "x"},
		config.Command{Name: "help", Description: "shadowed", Body: "x"},
	)
	got := HelpText(user, En)

	for _, b := range Builtins {
		if !strings.Contains(got, "/"+b.Name) {
			t.Errorf("/%s is missing from help", b.Name)
		}
	}
	for _, key := range []string{"enter", "tab", "^C"} {
		if !strings.Contains(got, key) {
			t.Errorf("%q is missing from help", key)
		}
	}
	if !strings.Contains(got, "/revisar") || !strings.Contains(got, "reviews the diff") {
		t.Errorf("the user's own commands must be listed:\n%s", got)
	}
	// A shadowed name must not be advertised as if it worked.
	if strings.Count(got, "/help") != 1 {
		t.Errorf("a shadowed command must not be listed twice:\n%s", got)
	}
	if !strings.Contains(got, "Enter denies") {
		t.Errorf("the approval default belongs in help:\n%s", got)
	}
}

func TestPlanText(t *testing.T) {
	if got := PlanText(NewModel("", "", "", "", En)); !strings.Contains(got, "no plan") {
		t.Errorf("got %q", got)
	}
	got := PlanText(modelWithPlan())
	for _, want := range []string{"read the parser", "publish", "no network", "1 of 3"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q missing from:\n%s", want, got)
		}
	}
}

func TestReplanPrompt(t *testing.T) {
	if got := ReplanPrompt("  "); !strings.Contains(got, "Re-examine") {
		t.Errorf("got %q", got)
	}
	if got := ReplanPrompt("skip the docs"); !strings.Contains(got, "skip the docs") {
		t.Errorf("got %q", got)
	}
}

// `/init` has to read the repository before it can say anything true about it,
// so it is a turn rather than a template.
func TestInitPromptAsksForAReadFirst(t *testing.T) {
	for _, want := range []string{"DCODE.md", "Read what is already here", "already exists"} {
		if !strings.Contains(InitPrompt, want) {
			t.Errorf("%q missing from the init prompt", want)
		}
	}
}

func TestSessionList(t *testing.T) {
	if got := SessionList(nil, ""); !strings.Contains(got, "no other sessions") {
		t.Errorf("got %q", got)
	}
	got := SessionList([]protocol.Session{
		{ID: "s1", State: protocol.SessionStateIdle, Workspace: "/a"},
		{ID: "s2", State: protocol.SessionStateRunning, Workspace: "/b"},
	}, "s2")
	if !strings.Contains(got, "s1") || !strings.Contains(got, "/b") {
		t.Errorf("got:\n%s", got)
	}
	// The session the user is already in must be distinguishable.
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "s2") && !strings.HasPrefix(line, "*") {
			t.Errorf("the current session must be marked:\n%s", got)
		}
	}
}

// ---------- the `/` menu ----------

// The menu is a consequence of what is typed, never a mode the user has to
// leave: once there is an argument the user has chosen.
func TestCompleteOnlyOffersOnABarePrefix(t *testing.T) {
	user := userCommands(config.Command{Name: "revisar", Description: "revisa o diff", Body: "x"})

	if got := Complete("/", user, En); len(got) < len(Builtins) {
		t.Errorf("a bare slash offers everything, got %d", len(got))
	}
	if got := Complete("/pl", user, En); len(got) != 1 || got[0].Name != "plan" {
		t.Errorf("got %+v", got)
	}
	if got := Complete("/plan agora", user, En); got != nil {
		t.Errorf("an argument means the choice is made, got %+v", got)
	}
	if got := Complete("texto normal", user, En); got != nil {
		t.Errorf("got %+v", got)
	}
	if got := Complete("/zzz", user, En); got != nil {
		t.Errorf("nothing matches, so nothing is offered: %+v", got)
	}
}

// A user command can never shadow a built-in, so mixing them would suggest a
// competition that does not exist.
func TestCompleteListsBuiltinsFirstAndSkipsShadowed(t *testing.T) {
	user := userCommands(
		config.Command{Name: "revisar", Description: "revisa", Body: "x"},
		config.Command{Name: "plan", Description: "sombreado", Body: "x"},
	)
	got := Complete("/", user, En)

	var sawUser bool
	for _, c := range got {
		if c.Name == "revisar" {
			sawUser = true
		}
		if !sawUser && !isBuiltin(c.Name) {
			t.Errorf("built-ins come first, %q broke the order", c.Name)
		}
	}
	if !sawUser {
		t.Error("the user's own commands must be offered too")
	}
	// The shadowed name appears once — as the built-in.
	var n int
	for _, c := range got {
		if c.Name == "plan" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("a shadowed name must not be offered twice, got %d", n)
	}
}

func TestTheMenuFollowsTheLine(t *testing.T) {
	user := userCommands()
	m := NewModel("s", "/w", "m", "read-only", En)

	m = m.SetInput("/pl").Refresh(user)
	if len(m.Completions) != 1 {
		t.Fatalf("got %+v", m.Completions)
	}
	m = m.SetInput("/plan tudo").Refresh(user)
	if len(m.Completions) != 0 {
		t.Errorf("the menu closes when the choice is made: %+v", m.Completions)
	}
}

func TestTheMenuWrapsAtBothEnds(t *testing.T) {
	m := NewModel("s", "/w", "m", "read-only", En).SetInput("/").Refresh(userCommands())
	n := len(m.Completions)
	if n < 3 {
		t.Fatalf("setup: got %d candidates", n)
	}

	if got := m.MoveCompletion(-1); got.CompletionAt != n-1 {
		t.Errorf("up from the first wraps to the last, got %d", got.CompletionAt)
	}
	m.CompletionAt = n - 1
	if got := m.MoveCompletion(1); got.CompletionAt != 0 {
		t.Errorf("down from the last wraps to the first, got %d", got.CompletionAt)
	}
}

// The user's next keystroke is the argument; making them type the separator is
// a small tax on every single use.
func TestAcceptingACompletionLeavesASpaceOnlyWhenThereAreArguments(t *testing.T) {
	m := NewModel("s", "/w", "m", "read-only", En).SetInput("/conf").Refresh(userCommands())
	got := m.AcceptCompletion()
	if got.Input != "/config " {
		t.Errorf("a command taking an argument gets a space, got %q", got.Input)
	}
	if got.InputCursor != len([]rune(got.Input)) {
		t.Errorf("the caret goes to the end, got %d", got.InputCursor)
	}
	if len(got.Completions) != 0 {
		t.Error("accepting closes the menu")
	}

	m = NewModel("s", "/w", "m", "read-only", En).SetInput("/hel").Refresh(userCommands())
	if got := m.AcceptCompletion(); got.Input != "/help" {
		t.Errorf("a command without arguments needs no space, got %q", got.Input)
	}
	// Nothing to accept is a no-op rather than a panic.
	empty := NewModel("s", "/w", "m", "read-only", En)
	if got := empty.AcceptCompletion(); got.Input != "" {
		t.Errorf("got %q", got.Input)
	}
}

// Esc dismissed the menu for the line as it was, not for every line that
// follows.
func TestClosingTheMenuLastsUntilTheLineChanges(t *testing.T) {
	user := userCommands()
	m := NewModel("s", "/w", "m", "read-only", En).SetInput("/pl").Refresh(user)
	if len(m.Completions) == 0 {
		t.Fatal("setup")
	}

	m = m.CloseCompletions()
	if len(m.Completions) != 0 {
		t.Fatal("esc closes it")
	}
	m = m.Refresh(user)
	if len(m.Completions) != 0 {
		t.Error("and it stays closed while the line is unchanged")
	}

	m = m.Insert("a").Refresh(user)
	if m.CompletionsOff {
		t.Error("typing reopens it")
	}
}
