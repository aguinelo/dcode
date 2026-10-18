package app

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/behavior"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/tools"
)

// The live defect: an orchestrator built by the daemon got the prompt of an
// ordinary read-only session, asked the person to switch the sandbox mode and
// never delegated. Asserted on the session the daemon's own modes build, not
// on behavior alone: the role is only there if this package passes it.
func TestEachRoleIsBuiltWithItsOwnSection(t *testing.T) {
	ws := projectRepo(t)
	for _, tc := range []struct {
		name string
		mode func(*Options)
		want string
	}{
		{"orchestrator", (&Daemon{}).orchestratorMode("p"), "You are this project's orchestrator"},
		{"child", (&Daemon{}).childMode("c", []string{"docs"}), "You are a child of this project's orchestrator"},
		{"reviewer", reviewerMode(func(context.Context, tools.VerdictInput) (string, error) { return "ok", nil }),
			"You are a reviewer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts := baseOpts(t)
			opts.Workspace = ws
			if tc.name == "child" {
				requireSandbox(t, opts)
			}
			tc.mode(&opts)
			s, err := New(opts, &ConsoleEmitter{W: io.Discard}, DenyAll{})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(s.Prompt, tc.want) {
				t.Errorf("the %s's prompt has no role section:\n%s", tc.name, s.Prompt)
			}
			if strings.Contains(s.Prompt, "let the person decide whether to change the mode") {
				t.Errorf("the %s is told to let the person change the mode:\n%s", tc.name, s.Prompt)
			}
		})
	}

	plain := baseOpts(t)
	plain.Workspace = ws
	s, err := New(plain, &ConsoleEmitter{W: io.Discard}, DenyAll{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(s.Prompt, "Your role") {
		t.Errorf("an ordinary session was given a role section:\n%s", s.Prompt)
	}
}

// behavior spells the roles as strings rather than importing protocol, for the
// reason it spells the modes that way; the two spellings are compared here.
func TestTheRoleSpellingsAgreeWithTheProtocol(t *testing.T) {
	for _, tc := range [][2]string{
		{behavior.RoleOrchestrator, protocol.RoleOrchestrator},
		{behavior.RoleChild, protocol.RoleChild},
		{behavior.RoleReviewer, protocol.RoleReviewer},
	} {
		if tc[0] != tc[1] {
			t.Errorf("behavior spells a role %q and protocol spells it %q", tc[0], tc[1])
		}
	}
}
