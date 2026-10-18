package loop

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// A cut with no line boundary in its window lands where a character starts.
// Cutting on the byte handed the model half of an "é" — invalid UTF-8 at the
// head of the very text it is asked to read — whenever the window's first byte
// fell inside a multibyte character.
func TestTheTailNeverSplitsACharacter(t *testing.T) {
	for _, unit := range []string{"é", "日", "🙂"} {
		s := strings.Repeat(unit, 200)
		for max := 37; max <= 44; max++ {
			got, cut := tail(s, max)
			if !cut {
				t.Fatalf("%q×200 at %d: not reported as cut", unit, max)
			}
			if !utf8.ValidString(got) {
				t.Errorf("%q×200 at %d: the tail starts mid-character: % x", unit, max, got[:min(4, len(got))])
			}
			if len(got) > max {
				t.Errorf("%q×200 at %d: %d bytes, over the ceiling", unit, max, len(got))
			}
			if !strings.HasSuffix(s, got) {
				t.Errorf("%q×200 at %d: what came back is not the end of the output", unit, max)
			}
			if len(got) <= max-utf8.UTFMax {
				t.Errorf("%q×200 at %d: %d bytes, more was dropped than one character", unit, max, len(got))
			}
		}
	}
}
