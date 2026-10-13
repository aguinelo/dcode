package app

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/credential"
	"github.com/aguinelo/dcode/internal/protocol"
)

func askCommands(t *testing.T, d *Daemon, ws string) (string, protocol.CommandsResponse) {
	t.Helper()
	code, body := askCrew(t, d, "commands", ws)
	if code != http.StatusOK {
		t.Fatalf("status %d: %s", code, body)
	}
	var out protocol.CommandsResponse
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("%v: %s", err, body)
	}
	return body, out
}

// The commands of a workspace are the user's and the project's, the project's
// winning by name and saying it took the name, sorted by name, each with where
// it comes from and the text it sends.
func TestTheCommandsRouteListsWhatTheTUIOffers(t *testing.T) {
	d, home := crewDaemon(t)
	ws := t.TempDir()
	write(t, home, filepath.Join("commands", "fix-issue.md"), "Fix the issue described here: $ARGUMENTS\n")
	write(t, home, filepath.Join("commands", "review.md"),
		"---\ndescription: the user's review\n---\nUSER review.\n")
	write(t, ws, filepath.Join(".dcode", "commands", "review.md"),
		"---\ndescription: review the staged change\n---\nReview the staged change.\n")
	write(t, ws, filepath.Join(".dcode", "commands", "notes.txt"), "not a command\n")

	_, got := askCommands(t, d, ws)
	want := protocol.CommandsResponse{
		Commands: []protocol.CommandInfo{
			{Name: "fix-issue", Source: protocol.CommandSourceUser, Path: "fix-issue.md",
				Body: "Fix the issue described here: $ARGUMENTS"},
			{Name: "review", Description: "review the staged change", Source: protocol.CommandSourceProject,
				Path: "review.md", Body: "Review the staged change."},
		},
		Overrides: []string{"review"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the commands are\n  %+v\nwant\n  %+v", got, want)
	}
}

// What the list offers is what the TUI resolves in the same workspace: the same
// discovery, so a command cannot mean one thing in one client and another in
// the other.
func TestTheCommandsListedAreTheOnesTheTUIResolves(t *testing.T) {
	d, home := crewDaemon(t)
	ws := t.TempDir()
	write(t, home, filepath.Join("commands", "a.md"), "---\nname: alpha\ndescription: first\n---\nA $ARGUMENTS\n")
	write(t, ws, filepath.Join(".dcode", "commands", "b.md"), "B body\n")
	_, got := askCommands(t, d, ws)

	roots, err := config.DiscoverRoots(envFrom(map[string]string{"DCODE_HOME": home}))
	if err != nil {
		t.Fatal(err)
	}
	tui, err := config.DiscoverCommands(roots, ws, config.CommandMaxBytes)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Commands) != len(tui.Commands) {
		t.Fatalf("listed %+v, the TUI resolves %+v", got.Commands, tui.Commands)
	}
	for _, c := range got.Commands {
		mine, ok := tui.Commands[c.Name]
		if !ok || mine.Body != c.Body || mine.Description != c.Description {
			t.Errorf("/%s is listed as %+v, the TUI resolves %+v", c.Name, c, mine)
		}
	}
}

// A command file that cannot be one refuses the list with the reason, as it
// refuses the TUI's start: a list without it would offer the rest as if the
// broken one did not exist.
func TestACommandThatCannotLoadRefusesWithTheReason(t *testing.T) {
	d, _ := crewDaemon(t)
	ws := t.TempDir()
	write(t, ws, filepath.Join(".dcode", "commands", "empty.md"), "---\ndescription: nothing\n---\n")
	code, body := askCrew(t, d, "commands", ws)
	if code == http.StatusOK || !strings.Contains(body, "no body") {
		t.Errorf("got %d %s, want a refusal saying the command has no body", code, body)
	}
}

// Never more than the route declares: not the key, its mask or its
// fingerprint, not where the user's config root is. The fields are a closed
// set.
func TestCommandsNeverCarryWhatTheyDoNotDeclare(t *testing.T) {
	d, home := crewDaemon(t)
	ws := t.TempDir()
	projectConfig(t, ws, "# CONFIG-BESIDE-THE-COMMANDS\n")
	write(t, home, filepath.Join("commands", "c.md"), "a body\n")
	body, _ := askCommands(t, d, ws)
	for _, secret := range []string{crewKey, credential.Mask(crewKey), credential.Fingerprint(crewKey),
		"CONFIG-BESIDE-THE-COMMANDS", home} {
		if strings.Contains(body, secret) {
			t.Errorf("the answer carries %q: %s", secret, body)
		}
	}
	closed(t, body, map[string]bool{"commands": true, "overrides": true}, map[string]map[string]bool{
		"commands": {"name": true, "description": true, "source": true, "path": true, "body": true},
	})
}

// A workspace that cannot be one is refused, as a session there would be.
func TestCommandsRefuseAWorkspaceThatCannotBeRead(t *testing.T) {
	d, _ := crewDaemon(t)
	broken := t.TempDir()
	projectConfig(t, broken, "[model]\nnmae = \"claude-opus-4\"\n")
	for _, tc := range []struct{ ws, reason string }{
		{"projects/dcode", "absolute"},
		{filepath.Join(t.TempDir(), "gone"), "cannot be opened"},
		{broken, "nmae"},
	} {
		code, body := askCrew(t, d, "commands", tc.ws)
		if code != http.StatusBadRequest || !strings.Contains(body, protocol.CodeWorkspaceInvalid) ||
			!strings.Contains(body, tc.reason) {
			t.Errorf("%s: got %d %s, want 400 %s saying %q", tc.ws, code, body, protocol.CodeWorkspaceInvalid, tc.reason)
		}
	}
}
