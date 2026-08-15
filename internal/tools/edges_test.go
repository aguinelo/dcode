package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Arguments that are not the declared shape become a tool error the model reads
// and corrects, naming the tool and pointing at its schema. A decode failure
// that surfaced as a panic or a bare JSON message would end the turn on
// something the model could have fixed in one call.
func TestArgumentsThatDoNotMatchTheSchemaBecomeACorrectableError(t *testing.T) {
	var in struct {
		Path string `json:"path"`
	}
	err := decode("read", json.RawMessage(`{"path":42}`), &in)
	if err == nil {
		t.Fatal("arguments of the wrong type decoded successfully")
	}
	// What the model reads is the Result, not the Go error: the hint lives in
	// its own field and only reaches the conversation through here.
	te, ok := err.(*ToolError)
	if !ok {
		t.Fatalf("err is %T, want a tool error the loop can turn into a result", err)
	}
	shown := te.Result().Output
	if !strings.Contains(shown, "read") {
		t.Errorf("what the model reads does not name the tool: %q", shown)
	}
	if !strings.Contains(shown, "schema") {
		t.Errorf("what the model reads does not point at the schema: %q", shown)
	}
	if !te.Result().IsError {
		t.Error("the result is not marked as an error")
	}

	// No arguments at all is an empty object, not a failure: a tool whose
	// fields are all optional is called with nothing.
	if err := decode("process", nil, &in); err != nil {
		t.Errorf("a call with no arguments was refused: %v", err)
	}
}

// The one word that separates a live process from a finished one. A model
// reading output from a process it believes is running will wait for more; a
// model reading a finished one has everything there will ever be.
func TestAProcessSaysWhetherItIsStillRunning(t *testing.T) {
	for _, tc := range []struct {
		name    string
		code    int
		done    bool
		stopped bool
		want    string
	}{
		{"still running", 0, false, false, "running"},
		{"stopped by us", 0, true, true, "stopped"},
		{"finished cleanly", 0, true, false, "exit 0"},
		{"finished badly", 2, true, false, "exit 2"},
		// Stopped wins over the exit code: a process we ended has an exit code
		// too, and reporting it would read as the command's own verdict.
		{"stopped, with a code", 143, true, true, "stopped"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := &procEntry{
				handle:  &fakeHandle{code: tc.code, done: tc.done},
				stopped: tc.stopped,
			}
			if got := status(e); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// A path whose parent is a file cannot be made into a directory, and the write
// fails rather than reporting a success that put the content nowhere. Both the
// atomic and the direct path have to fail the same way.
func TestWritingUnderAFileFailsRatherThanReportingSuccess(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, atomic := range []bool{true, false} {
		if err := writeFile(filepath.Join(blocker, "b.txt"), "hello", atomic); err == nil {
			t.Errorf("atomic=%v: writing under a file reported success", atomic)
		}
	}
}
