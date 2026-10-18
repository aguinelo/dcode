package orchestrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// Spec: orchestration .p §8 — state.json is rewritten whole or not at all, and
// reads back what was saved.
func TestTheStateIsSavedWholeAndReadsBack(t *testing.T) {
	dir := t.TempDir()
	s := storeAt(dir, testProjectID)
	want := State{V: stateVersion, Project: protocol.Project{
		ID: testProjectID, Root: "/r", Main: "main", Autonomy: "step", OrchestratorID: "o",
		Created: time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC),
	}, LogSeq: 3}
	if err := s.Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Project != want.Project || got.LogSeq != want.LogSeq || got.V != want.V {
		t.Errorf("Load = %+v, want %+v", got, want)
	}
	entries, err := os.ReadDir(filepath.Join(dir, testProjectID))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != stateFile {
		t.Errorf("the project directory holds %v, want only %s", entries, stateFile)
	}
	if info, _ := entries[0].Info(); info.Mode().Perm() != 0o600 {
		t.Errorf("state.json mode is %v, want 0600", info.Mode().Perm())
	}
}

func TestAStateOfAnotherVersionOrProjectIsRefused(t *testing.T) {
	dir := t.TempDir()
	s := storeAt(dir, testProjectID)
	for name, body := range map[string]string{
		"version": `{"v":2,"project":{"id":"` + testProjectID + `"}}`,
		"project": `{"v":1,"project":{"id":"ffffffffffffffffffffffff"}}`,
	} {
		if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(s.path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Load(); err == nil || !strings.Contains(err.Error(), s.path) {
			t.Errorf("%s: Load = %v, want an error naming the file", name, err)
		}
	}
}
