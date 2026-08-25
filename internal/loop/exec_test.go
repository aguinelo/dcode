package loop

import (
	"context"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/tools"
)

// A command the person typed goes through the tool, so the sandbox and the
// approval machinery apply to it unchanged. `!` is a shortcut past the model,
// never past the boundary — a shell escape that skipped it would be a hole in
// the one thing this product is built around.
func TestATypedCommandGoesThroughTheTool(t *testing.T) {
	e, _ := newEngine(t, nil, tools.NewRegistry(tools.Bash{}))
	if _, err := e.Exec(context.Background(), "echo hello"); err != nil {
		t.Fatalf("the command did not run: %v", err)
	}
	h := e.Session().History
	if len(h) == 0 {
		t.Fatal("the command left nothing in the history")
	}
	last := h[len(h)-1]
	if !strings.Contains(last.Text, "echo hello") {
		t.Errorf("the history does not say what was run: %q", last.Text)
	}
	// As something the user did, because they did. A tool result with no tool
	// call before it is a shape no provider accepts, and inventing an assistant
	// call would put words in the model's mouth to hold an output it never
	// asked for.
	if last.ToolResult != nil {
		t.Error("the output was attached as a tool result with no call before it")
	}
}

// An empty command is refused rather than run. There is nothing there.
func TestAnEmptyCommandIsRefused(t *testing.T) {
	e, _ := newEngine(t, nil, tools.NewRegistry(tools.Bash{}))
	if _, err := e.Exec(context.Background(), "   "); err == nil {
		t.Error("an empty command was accepted")
	}
}
