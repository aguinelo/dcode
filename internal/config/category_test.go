package config

import "testing"

// "Go tools" is a category. "the Task tool" is a name.
//
// The capitalised form was matched whether the noun was singular or plural, so
// every heading a DCODE.md for a Go repository naturally carries — "Go tools",
// "Build tools", "Testing tools", "CLI tools" — was reported as a tool dcode
// does not have.
//
// English decides this for us and the existing cases already agree: a specific
// tool is named in the singular and with a determiner, and a family of them is
// plural. Every true positive on record reads "the X tool"; every false one
// reads "X tools".
//
// Found by measurement, not by reading: init-drops-absent-tool measured 78%
// against a threshold of 100% that is legitimate precisely because the check is
// deterministic. A deterministic check that is wrong is worse than a noisy one,
// because the threshold says it cannot be.
func TestACategoryOfToolsIsNotAToolName(t *testing.T) {
	have := []string{"bash", "edit", "glob", "grep", "plan", "read", "write"}

	for _, s := range []string{
		"Go tools: `go build`, `go test ./...`",
		"Build tools are in the Makefile.",
		"The CLI tools live under cmd/.",
		"Standard tools apply.",
		"Formatting tools run in CI.",
	} {
		if f := VerifyTools(s, have); len(f) > 0 {
			t.Errorf("a category was read as a tool name: %q -> %v", s, f)
		}
	}
}

// A backticked identifier on a line that happens to mention a tool is not
// thereby a tool.
//
// The sweep took every backticked name on any line containing "tool", so a
// sentence naming a helper and a tool in the same breath reported the helper.
// The forms that genuinely name a tool — "the `glob` tool", "tools: a, b" —
// are matched by their own patterns and do not need this.
func TestAnIdentifierNearTheWordToolIsNotATool(t *testing.T) {
	have := []string{"bash", "edit", "glob", "read"}

	f := VerifyTools("Use the `Split` helper before reaching for the `Dispatch` tool.", have)
	if len(f) != 1 || f[0].Subject != "Dispatch" {
		t.Errorf("got %v, want only Dispatch — `Split` is a helper on the same line", f)
	}
}

// The two forms that carry their own evidence keep working in the plural,
// because neither can be confused with prose: English has no underscores, and
// nobody backticks a category.
func TestTheUnmistakableFormsStillCountWhenPlural(t *testing.T) {
	have := []string{"read", "write"}

	if f := VerifyTools("Register the memory_store tools first.", have); len(f) != 1 {
		t.Errorf("snake_case in the plural was dropped: %v", f)
	}
	if f := VerifyTools("Use the `Dispatcher` tools.", have); len(f) != 1 {
		t.Errorf("a backticked name in the plural was dropped: %v", f)
	}
}
