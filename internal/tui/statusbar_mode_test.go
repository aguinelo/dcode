package tui

import (
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// TestModeSegmentRendersAllModes covers the three modes plus the empty case.
// The empty case must NOT render, because a bar of empty slots is the bar
// describing itself rather than the session.
func TestModeSegmentRendersAllModes(t *testing.T) {
	cases := []struct {
		mode string
		want string
	}{
		{"", ""}, // empty: no segment
		{protocol.ModePlan, "[plan]"},
		{protocol.ModeAssist, "[assist]"},
		{protocol.ModeAuto, "[auto]"},
	}
	for _, c := range cases {
		t.Run(c.mode, func(t *testing.T) {
			m := Model{Mode: c.mode}
			seg, ok := modeSegment(m, DefaultGeometry(100, 30))
			if c.want == "" {
				if ok {
					t.Errorf("mode %q: got segment, want none", c.mode)
				}
				return
			}
			if !ok {
				t.Fatalf("mode %q: no segment returned", c.mode)
			}
			if !strings.Contains(seg.text, c.want) {
				t.Errorf("mode %q: text = %q, want substring %q", c.mode, seg.text, c.want)
			}
		})
	}
}
