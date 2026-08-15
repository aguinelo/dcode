package loop

import (
	"context"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// A sub-turn cannot ask anybody anything: nobody is watching it, and a prompt
// with no reader is a hang. So it denies, and records what it denied — silence
// would leave a conclusion with an undeclared hole, which is a wrong conclusion
// that looks complete.
//
// What it records has to identify the thing refused. A command names itself; a
// tool with no command is named by the tool, because "denied: " tells the
// parent nothing it can act on.
func TestASubTurnDeniesAndSaysWhatItDenied(t *testing.T) {
	d := &denyAll{}
	ctx := context.Background()

	got, err := d.Approve(ctx, protocol.ApprovalRequest{Tool: "bash", Command: "rm -rf /"})
	if err != nil {
		t.Fatal(err)
	}
	if got != protocol.ApprovalDeny {
		t.Errorf("decision = %v, want deny", got)
	}

	if _, err := d.Approve(ctx, protocol.ApprovalRequest{Tool: "write"}); err != nil {
		t.Fatal(err)
	}

	if len(d.denied) != 2 {
		t.Fatalf("recorded %v, want both refusals", d.denied)
	}
	if d.denied[0] != "rm -rf /" {
		t.Errorf("recorded %q, want the command", d.denied[0])
	}
	if d.denied[1] != "write" {
		t.Errorf("recorded %q, want the tool name when there is no command", d.denied[1])
	}
}
