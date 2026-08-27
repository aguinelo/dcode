package loopcommand

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/loop"
)

func workspaceWith(t *testing.T, folders map[string]map[string]string) string {
	t.Helper()
	ws := t.TempDir()
	for name, files := range folders {
		dir := filepath.Join(ws, SpecsDirName, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		for f, body := range files {
			if err := os.WriteFile(filepath.Join(dir, f), []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	return ws
}

func runner(answers map[string]int) loop.CriterionRunner {
	return func(_ context.Context, cmd string) (int, string, error) {
		return answers[cmd], "", nil
	}
}

// Pending is answered by RUNNING the folder's criteria, not by counting
// checkboxes: a box is marked by whoever felt like marking it.
func TestPendingIsWhatTheCriteriaSay(t *testing.T) {
	ws := workspaceWith(t, map[string]map[string]string{
		"a-done":    {"done.toml": "[x]\ncommand = \"passes\"\n"},
		"b-partial": {"done.toml": "[x]\ncommand = \"passes\"\n\n[y]\ncommand = \"fails\"\n"},
	})
	got := Discover(context.Background(), ws, runner(map[string]int{"passes": 0, "fails": 1}), 0)
	if len(got) != 2 {
		t.Fatalf("found %+v", got)
	}
	if got[0].Path != filepath.Join("specs", "a-done") || got[0].Unmet != 0 || got[0].Pending() {
		t.Errorf("a spec whose criteria all pass is pending: %+v", got[0])
	}
	if got[1].Unmet != 1 || !got[1].Pending() {
		t.Errorf("a spec with an unmet criterion is not pending: %+v", got[1])
	}
}

// A folder that declares nothing is PENDING. There is no evidence it is
// finished, and treating "nothing to check" as "done" is the defect this whole
// family exists to prevent.
func TestAFolderThatDeclaresNothingIsPending(t *testing.T) {
	ws := workspaceWith(t, map[string]map[string]string{
		"c-manual": {"tasks.md": "- [ ] 1. `a.ts` — smoke manual, sem comando.\n"},
	})
	got := Discover(context.Background(), ws, runner(nil), 0)
	if len(got) != 1 || got[0].Criteria != 0 || !got[0].Pending() {
		t.Fatalf("a folder with no criteria is not pending: %+v", got)
	}
}

// Ordered by folder name, which for a dated spec is chronological. A list that
// reshuffles between runs is one nobody can act on.
func TestDiscoveryIsOrdered(t *testing.T) {
	ws := workspaceWith(t, map[string]map[string]string{
		"2026-08-25-c": {"done.toml": "[x]\ncommand = \"p\"\n"},
		"2026-08-23-a": {"done.toml": "[x]\ncommand = \"p\"\n"},
		"2026-08-24-b": {"done.toml": "[x]\ncommand = \"p\"\n"},
	})
	got := Discover(context.Background(), ws, runner(map[string]int{"p": 0}), 0)
	want := []string{"2026-08-23-a", "2026-08-24-b", "2026-08-25-c"}
	for i, w := range want {
		if got[i].Path != filepath.Join("specs", w) {
			t.Fatalf("order is %+v", got)
		}
	}
}

// A folder under specs/ that carries none of a spec's files is a folder, not a
// broken spec. One that looks like a spec and cannot be read earns a line.
func TestSomethingElseUnderSpecsIsNotReported(t *testing.T) {
	ws := workspaceWith(t, map[string]map[string]string{
		"assets":  {"logo.png": "not a spec"},
		"d-broke": {"tasks.md": "isto não é lista de tarefas\n"},
	})
	got := Discover(context.Background(), ws, runner(nil), 0)
	if len(got) != 1 {
		t.Fatalf("found %+v, want only the unreadable spec", got)
	}
	if got[0].Err == "" {
		t.Errorf("an unreadable spec was reported without a reason: %+v", got[0])
	}
	// It is not pending: nobody can work a folder nobody can read, and
	// queueing it would spend a session discovering that again.
	if got[0].Pending() {
		t.Error("an unreadable spec was queued as work")
	}
}

// No specs/ at all is an empty list, not an error: most workspaces have none.
func TestNoSpecsFolderIsAnEmptyList(t *testing.T) {
	if got := Discover(context.Background(), t.TempDir(), runner(nil), 0); got != nil {
		t.Errorf("got %+v, want nothing", got)
	}
}

// A cancelled discovery returns what it had rather than pretending it looked
// at everything. Running criteria across a backlog takes real time.
func TestACancelledDiscoveryStopsWhereItIs(t *testing.T) {
	ws := workspaceWith(t, map[string]map[string]string{
		"a": {"done.toml": "[x]\ncommand = \"p\"\n"},
		"b": {"done.toml": "[x]\ncommand = \"p\"\n"},
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := Discover(ctx, ws, runner(map[string]int{"p": 0}), 0); len(got) != 0 {
		t.Errorf("a cancelled discovery reported %+v", got)
	}
}

// Unavailable criteria count as work left: a criterion nobody could run is not
// evidence that anything passed.
func TestAnUnavailableCriterionIsWorkLeft(t *testing.T) {
	ws := workspaceWith(t, map[string]map[string]string{
		"e": {"done.toml": "[x]\ncommand = \"\"\n"},
	})
	got := Discover(context.Background(), ws, runner(nil), 0)
	if len(got) != 1 {
		t.Fatalf("found %+v", got)
	}
	if got[0].Unavailable != 1 || !got[0].Pending() {
		t.Errorf("an unavailable criterion did not count as work left: %+v", got[0])
	}
}

// A spec.md with no tasks.md is a spec that has not been broken into tasks
// yet, not an unreadable one.
//
// It is the most pending thing in the list: nothing says it is finished and
// nothing has even been planned. Calling it an error kept eleven of them out
// of the queue in the first real workspace this ran against.
func TestASpecWithNoTasksYetIsPendingAndNotBroken(t *testing.T) {
	ws := workspaceWith(t, map[string]map[string]string{
		"planned-only": {"spec.md": "# Uma spec escrita, ainda sem tarefas\n"},
	})
	got := Discover(context.Background(), ws, runner(nil), 0)
	if len(got) != 1 {
		t.Fatalf("found %+v", got)
	}
	if got[0].Err != "" {
		t.Errorf("a spec without tasks was reported as unreadable: %+v", got[0])
	}
	if got[0].Criteria != 0 || !got[0].Pending() {
		t.Errorf("a spec without tasks is not pending: %+v", got[0])
	}
}
