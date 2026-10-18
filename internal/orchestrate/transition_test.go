package orchestrate

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// Spec: orchestration .p §8, §14 → §13 — every transition of §6 has its line
// in the log before its event reaches the wire, and a line that does not
// write stops the transition: the state does not move, nothing says it did,
// and the failure is said instead.
func TestATransitionIsLoggedBeforeItIsSaidAndALineThatFailsStopsIt(t *testing.T) {
	p := openWith(t, nil)
	logPath := filepath.Join(p.dir, p.o.Project().ID, decisionsFile)
	var mu sync.Mutex
	var said []protocol.EventType
	var unlogged []string
	p.reg.emit = func(_ string, typ protocol.EventType, payload any) {
		mu.Lock()
		defer mu.Unlock()
		said = append(said, typ)
		cc, ok := payload.(protocol.ChildChanged)
		if !ok {
			return
		}
		last := lastDecision(t, logPath)
		if last.Child != cc.Child.ID {
			unlogged = append(unlogged, string(typ)+" of "+cc.Child.ID+" after "+last.Kind+" of "+last.Child)
		}
	}

	c := p.one(t, item("index", "internal/index"))
	p.o.move(c.ID, stateFailed, protocol.StopError, "a transition the log takes")
	mu.Lock()
	if len(unlogged) != 0 {
		t.Errorf("events reached the wire before their line: %v", unlogged)
	}
	before := len(said)
	mu.Unlock()

	stateBefore, err := os.ReadFile(filepath.Join(p.dir, p.o.Project().ID, stateFile))
	if err != nil {
		t.Fatal(err)
	}
	_ = p.o.log.f.Close() // the log can no longer take a line
	got := p.o.move(c.ID, stateWorking, "", "a transition the log cannot take")

	if got.State != stateFailed || p.o.child(c.ID).State != stateFailed {
		t.Errorf("a transition whose line did not write moved the child to %s", p.o.child(c.ID).State)
	}
	stateAfter, err := os.ReadFile(filepath.Join(p.dir, p.o.Project().ID, stateFile))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stateBefore, stateAfter) {
		t.Error("state.json changed for a transition the log did not take: the disk is ahead of the log")
	}
	mu.Lock()
	defer mu.Unlock()
	after := said[before:]
	if len(after) != 1 || after[0] != protocol.EventSessionError {
		t.Errorf("after the failed line the wire got %v, want the failure said and no child.state", after)
	}
}

// lastDecision is the last complete line of the log at path.
func lastDecision(t *testing.T, path string) Decision {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := bytes.Split(bytes.TrimRight(raw, "\n"), []byte("\n"))
	var d Decision
	if err := json.Unmarshal(lines[len(lines)-1], &d); err != nil {
		t.Fatalf("the log's last line does not read: %v", err)
	}
	return d
}
