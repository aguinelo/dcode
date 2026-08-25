package tui

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/aguinelo/dcode/internal/protocol"
)

func shiftTab() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
}

// TestTheBarFollowsTheAnnouncement: the mode reaches the client over the event
// log, never from the reply to the call that changed it. A client that was not
// attached when the switch happened learns it the same way as one that was.
func TestTheBarFollowsTheAnnouncement(t *testing.T) {
	m := Model{Mode: protocol.ModeAssist}
	payload, err := json.Marshal(protocol.SessionModeChanged{
		Previous: protocol.ModeAssist, Mode: protocol.ModeAuto,
	})
	if err != nil {
		t.Fatal(err)
	}
	m = m.Apply(protocol.Event{Type: protocol.EventSessionModeChanged, Payload: payload})
	if m.Mode != protocol.ModeAuto {
		t.Errorf("Mode = %q, want auto", m.Mode)
	}
}

// TestEveryModeGoesThroughOnTheFirstTry covers both ways in, auto included.
//
// `/mode auto` used to warn and require the gesture again. Typing eleven
// deliberate characters IS the decision — asking the person to repeat what they
// just said is not a safeguard, it is a step to learn past. What says there is
// no boundary is the badge on the bar, and it says so for as long as it is
// true, rather than once, to someone who had already decided.
func TestEveryModeGoesThroughOnTheFirstTry(t *testing.T) {
	t.Run("slash command", func(t *testing.T) {
		for _, mode := range []string{protocol.ModePlan, protocol.ModeAssist, protocol.ModeAuto} {
			t.Run(mode, func(t *testing.T) {
				p, tr := newProgram(t)
				p.model.Mode = ""

				run(t, p, typeLine(t, p, "/mode "+mode))
				if got := tr.modes(); len(got) != 1 || got[0] != mode {
					t.Errorf("modes = %v, want [%s] on the first try", got, mode)
				}
			})
		}
	})

	t.Run("shift+tab reaches auto in one press", func(t *testing.T) {
		p, tr := newProgram(t)
		p.model.Mode = protocol.ModeAssist

		_, cmd := p.Update(shiftTab())
		run(t, p, cmd)
		if got := tr.modes(); len(got) != 1 || got[0] != protocol.ModeAuto {
			t.Errorf("modes = %v, want [auto] on the first press", got)
		}
	})

	t.Run("nothing is left on screen asking for a repeat", func(t *testing.T) {
		p, _ := newProgram(t)
		p.model.Mode = protocol.ModeAssist

		run(t, p, typeLine(t, p, "/mode auto"))
		if strings.Contains(p.model.Flash, "again") || strings.Contains(p.model.Flash, "de novo") {
			t.Errorf("a confirmation prompt survived: %q", p.model.Flash)
		}
	})
}

// TestShiftTabCycles pins the order, including where an unnamed boundary enters.
func TestShiftTabCycles(t *testing.T) {
	cases := []struct{ from, want string }{
		{protocol.ModePlan, protocol.ModeAssist},
		{protocol.ModeAssist, protocol.ModeAuto},
		{protocol.ModeAuto, protocol.ModePlan},
		{"", protocol.ModePlan},
	}
	for _, c := range cases {
		t.Run(c.from+"->"+c.want, func(t *testing.T) {
			if got := nextMode(c.from); got != c.want {
				t.Errorf("nextMode(%q) = %q, want %q", c.from, got, c.want)
			}
		})
	}
}

// TestModeWithNoArgumentDoesNotInventOne is the client half of the defect.
//
// It used to answer "current mode: assist" whenever it did not know, which is
// the same lie the daemon was telling — a boundary named after the one the
// person would have hoped for.
func TestModeWithNoArgumentDoesNotInventOne(t *testing.T) {
	p, _ := newProgram(t)
	p.model.Mode = ""

	run(t, p, typeLine(t, p, "/mode"))
	last := p.model.Entries[len(p.model.Entries)-1].Summary
	// Not "contains assist" — the answer legitimately lists the three names as
	// what you may pick. What must not appear is the claim that one is in force.
	claimed := fmt.Sprintf(Text(p.model.Lang).CmdModeCurrent, protocol.ModeAssist)
	if strings.Contains(last, claimed) {
		t.Errorf("an unnamed boundary was reported as a mode: %q", last)
	}
	if !strings.Contains(last, "/mode") {
		t.Errorf("the answer must say how to pick one, got %q", last)
	}

	p.model.Mode = protocol.ModePlan
	run(t, p, typeLine(t, p, "/mode"))
	if last := p.model.Entries[len(p.model.Entries)-1].Summary; !strings.Contains(last, "plan") {
		t.Errorf("a named mode must be reported, got %q", last)
	}
}

// TestAnUnknownModeNeverLeavesTheClient keeps a typo off the network.
func TestAnUnknownModeNeverLeavesTheClient(t *testing.T) {
	p, tr := newProgram(t)
	p.model.Mode = protocol.ModeAssist

	run(t, p, typeLine(t, p, "/mode yolo"))
	if got := tr.modes(); len(got) != 0 {
		t.Errorf("a typo reached the daemon: %v", got)
	}
	if last := p.model.Entries[len(p.model.Entries)-1].Summary; !strings.Contains(last, "yolo") {
		t.Errorf("the refusal must name what was typed, got %q", last)
	}
}

// TestTheTopBarFollowsTheSwitchToo is the third instance of one defect.
//
// The badge learned the new mode and the top bar did not, so a session in
// `auto` announced `workspace-write` — and §2.1 names the sandbox field the one
// place where being wrong is dangerous. It announced a limit that had just been
// lifted.
func TestTheTopBarFollowsTheSwitchToo(t *testing.T) {
	m := Model{Mode: protocol.ModeAssist, Sandbox: "workspace-write", Window: 1000}
	payload, err := json.Marshal(protocol.SessionModeChanged{
		Previous: protocol.ModeAssist, Mode: protocol.ModeAuto, SandboxMode: "full-access",
	})
	if err != nil {
		t.Fatal(err)
	}
	m = m.Apply(protocol.Event{Type: protocol.EventSessionModeChanged, Payload: payload})

	if m.Sandbox != "full-access" {
		t.Errorf("Sandbox = %q, want full-access", m.Sandbox)
	}
	screen := Render(m, DefaultGeometry(120, 30))
	if strings.Contains(screen, "workspace-write") {
		t.Errorf("the bar still names the boundary that was lifted:\n%s", screen)
	}
	// Case-insensitive: full-access is rendered loud — `!! FULL-ACCESS !!` —
	// by the invariant that the mode with no boundary is never quiet. What is
	// asserted here is that the bar names it at all, not how it shouts.
	if !strings.Contains(strings.ToLower(screen), "full-access") {
		t.Errorf("the bar does not name the boundary in force:\n%s", screen)
	}
}

// TestAnAnnouncementWithoutASandboxLeavesTheBarAlone: an older daemon says
// nothing about the sandbox, and a bar that blanks itself on silence is worse
// than one that keeps the last thing it was told.
func TestAnAnnouncementWithoutASandboxLeavesTheBarAlone(t *testing.T) {
	m := Model{Mode: protocol.ModeAssist, Sandbox: "workspace-write"}
	payload, err := json.Marshal(protocol.SessionModeChanged{Mode: protocol.ModeAuto})
	if err != nil {
		t.Fatal(err)
	}
	m = m.Apply(protocol.Event{Type: protocol.EventSessionModeChanged, Payload: payload})

	if m.Sandbox != "workspace-write" {
		t.Errorf("Sandbox = %q, want the previous value kept", m.Sandbox)
	}
}
