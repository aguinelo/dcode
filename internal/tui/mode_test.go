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

// TestDroppingTheBoundaryTakesTwo covers both ways into auto.
//
// The first gesture warns and asks for nothing else; only the second reaches
// the daemon. Everything short of auto goes through on the first, because
// arming a mode that keeps the boundary would train the confirmation away.
func TestDroppingTheBoundaryTakesTwo(t *testing.T) {
	t.Run("slash command", func(t *testing.T) {
		p, tr := newProgram(t)
		p.model.Mode = protocol.ModeAssist

		run(t, p, typeLine(t, p, "/mode auto"))
		if got := tr.modes(); len(got) != 0 {
			t.Fatalf("the first /mode auto reached the daemon: %v", got)
		}
		if !strings.Contains(p.model.Flash, "auto") {
			t.Errorf("no warning on screen, Flash = %q", p.model.Flash)
		}
		run(t, p, typeLine(t, p, "/mode auto"))
		if got := tr.modes(); len(got) != 1 || got[0] != protocol.ModeAuto {
			t.Errorf("after the second: modes = %v, want [auto]", got)
		}
	})

	t.Run("shift+tab", func(t *testing.T) {
		p, tr := newProgram(t)
		p.model.Mode = protocol.ModeAssist

		_, cmd := p.Update(shiftTab())
		run(t, p, cmd)
		if got := tr.modes(); len(got) != 0 {
			t.Fatalf("the first shift+tab onto auto reached the daemon: %v", got)
		}
		if !strings.Contains(p.model.Flash, "auto") {
			t.Errorf("no warning on screen, Flash = %q", p.model.Flash)
		}
		_, cmd = p.Update(shiftTab())
		run(t, p, cmd)
		if got := tr.modes(); len(got) != 1 || got[0] != protocol.ModeAuto {
			t.Errorf("after the second: modes = %v, want [auto]", got)
		}
	})

	t.Run("a bounded mode is not armed", func(t *testing.T) {
		p, tr := newProgram(t)
		p.model.Mode = protocol.ModeAuto

		run(t, p, typeLine(t, p, "/mode plan"))
		if got := tr.modes(); len(got) != 1 || got[0] != protocol.ModePlan {
			t.Errorf("modes = %v, want [plan] on the first try", got)
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
