package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// `dcode orchestrate` (orchestration .i step 13, decision 4): the project of
// the folder, resolved by the daemon, and the TUI on its orchestrator. Wired
// against a fake daemon: what is asked of it, and what is said when it cannot.

type fakeProjects struct {
	healthErr  error
	openErr    error
	orchErr    error
	session    protocol.Session
	opened     []string
	askedFor   []string
	healthSeen int
}

func (f *fakeProjects) Health(context.Context) error { f.healthSeen++; return f.healthErr }

func (f *fakeProjects) OpenProject(_ context.Context, ws string) (protocol.Project, error) {
	f.opened = append(f.opened, ws)
	if f.openErr != nil {
		return protocol.Project{}, f.openErr
	}
	return protocol.Project{ID: "p1", Root: ws, OrchestratorID: f.session.ID}, nil
}

func (f *fakeProjects) Orchestrator(_ context.Context, id string) (protocol.Session, error) {
	f.askedFor = append(f.askedFor, id)
	return f.session, f.orchErr
}

func TestOrchestrateOpensTheProjectOfTheFolderAndAttachesToItsOrchestrator(t *testing.T) {
	f := &fakeProjects{session: protocol.Session{ID: "orq", Role: protocol.RoleOrchestrator}}
	got, err := openOrchestrator(context.Background(), f, "/sock", "/w/sub")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "orq" {
		t.Fatalf("attached to %q, not the project's orchestrator", got.ID)
	}
	if len(f.opened) != 1 || f.opened[0] != "/w/sub" {
		t.Fatalf("the project was not opened by POST /v1/projects for the folder: %v", f.opened)
	}
	if len(f.askedFor) != 1 || f.askedFor[0] != "p1" {
		t.Fatalf("the orchestrator was not asked of the project the daemon resolved: %v", f.askedFor)
	}
}

func TestOrchestrateWithNoDaemonSaysToRunServe(t *testing.T) {
	f := &fakeProjects{healthErr: errors.New("connection refused")}
	_, err := openOrchestrator(context.Background(), f, "/sock", "/w")
	if err == nil {
		t.Fatal("no daemon was not an error")
	}
	for _, want := range []string{"/sock", "dcode serve", "connection refused"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not say %q: %v", want, err)
		}
	}
	if len(f.opened) != 0 {
		t.Fatal("a project was asked of a daemon that did not answer")
	}
}

func TestADisabledOrchestratorIsSaidNamingTheKey(t *testing.T) {
	f := &fakeProjects{openErr: protocol.Errorf(protocol.CodeOrchestratorUnavailable,
		"orchestrator.enabled is false: the orchestrator is opt-in in this slice")}
	_, err := openOrchestrator(context.Background(), f, "/sock", "/w")
	if err == nil {
		t.Fatal("a disabled orchestrator was not an error")
	}
	for _, want := range []string{"orchestrator.enabled", "record.enabled", "dcode serve"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not name %q: %v", want, err)
		}
	}
}

func TestASessionThatIsNotAnOrchestratorIsRefused(t *testing.T) {
	f := &fakeProjects{session: protocol.Session{ID: "s1"}}
	if _, err := openOrchestrator(context.Background(), f, "/sock", "/w"); err == nil {
		t.Fatal("the TUI would have opened an ordinary session as the orchestrator")
	}
}

func TestOrchestrateDispatchesAndNeverStartsADaemonOfItsOwn(t *testing.T) {
	t.Setenv("DCODE_HOME", t.TempDir())
	sock := filepath.Join(t.TempDir(), "none.sock")
	err := dispatch([]string{"orchestrate", "--socket", sock, "--workspace", t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "dcode serve") {
		t.Fatalf("orchestrate with no daemon did not say to run one: %v", err)
	}
}
