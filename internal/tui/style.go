package tui

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

// Style is one visual role, not one colour.
//
// Roles rather than colours because the palette has to answer "what does this
// mean" in three different terminals. A caller that asks for "red" has already
// decided something the theme should decide.
type Style uint8

const (
	StyleNone Style = iota
	StyleDim
	StyleBold
	StyleAccent  // the product's own mark
	StyleAdded   // a diff line that arrived
	StyleRemoved // a diff line that left
	StyleError
	StyleWarn
	StyleOK
	StyleDanger // full-access, and nothing else
	StyleCursor
	// The three amber tones and the terracotta of the mark. They exist so the
	// mascot renders as itself in the terminal; nothing else uses them, because
	// the moment a second thing does, the eye stops being a marker.
	StyleHighlight
	StyleBody
	StyleShadow
	StyleEye
	// StyleOnAccent is amber ground with near-black text. The design gives it
	// to structure and to nothing else: two of them side by side would say that
	// everything is structure.
	StyleOnAccent
)

// Palette turns a role into an escape sequence.
//
// The zero value writes nothing at all, which is what makes colour removable:
// every rendering path is the same code, and a monochrome terminal simply gets
// empty strings rather than a second implementation.
type Palette struct {
	Enabled bool
}

// ansi codes per role, foreground unless noted.
var ansi = map[Style]string{
	StyleDim:     "2",
	StyleBold:    "1",
	StyleAccent:  "38;5;179", // the amber of the mark
	StyleAdded:   "32",
	StyleRemoved: "31",
	StyleError:   "31",
	StyleWarn:    "33",
	StyleOK:      "32",
	StyleDanger:  "1;97;41", // white on red: the one thing that must not be missed
	StyleCursor:  "7",       // reverse video, so it survives any theme

	// Nearest 256-colour neighbours of #EFC066, #E0A030, #B87D1E and #A8452A.
	StyleHighlight: "38;5;222",
	StyleBody:      "38;5;179",
	StyleShadow:    "38;5;136",
	StyleEye:       "38;5;131",

	// Amber ground, near-black text — the contrast the design pins at 8:1.
	StyleOnAccent: "48;5;179;38;5;234",
}

// Apply wraps text in a role.
//
// Width is unaffected by design: every caller measures display cells before
// styling, and a style that changed the measured width would break the layout
// only on colour terminals — the hardest kind of bug to see in a screenshot.
func (p Palette) Apply(s Style, text string) string {
	if !p.Enabled || s == StyleNone || text == "" {
		return text
	}
	code, ok := ansi[s]
	if !ok {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

// ContextStyle grades the context meter.
//
// Colour here is a warning, not decoration: the number matters only when it is
// close to the point where history starts being summarised away.
func ContextStyle(pct int) Style {
	switch {
	case pct >= 90:
		return StyleError
	case pct >= 75:
		return StyleWarn
	default:
		return StyleDim
	}
}

// DiffStyle classifies one line of a unified diff.
func DiffStyle(line string) Style {
	switch {
	case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
		return StyleDim
	case strings.HasPrefix(line, "+"):
		return StyleAdded
	case strings.HasPrefix(line, "-"):
		return StyleRemoved
	case strings.HasPrefix(line, "@@"):
		return StyleDim
	default:
		return StyleNone
	}
}

// ColorEnabled decides whether to emit escapes at all.
//
// NO_COLOR is honoured because it is the convention users already know, and
// TERM=dumb because a terminal that says it cannot should be believed. An
// explicit DCODE_COLOR wins over both: it is the user answering for their own
// terminal, and this is exactly the case where they know better than the
// heuristics.
func ColorEnabled(env func(string) string) bool {
	switch strings.ToLower(env("DCODE_COLOR")) {
	case "always", "1", "true", "yes":
		return true
	case "never", "0", "false", "no":
		return false
	}
	if env("NO_COLOR") != "" {
		return false
	}
	term := env("TERM")
	if term == "" || term == "dumb" {
		return false
	}
	return true
}

// spinnerFrames animate the working indicator.
//
// Braille rather than a rotating slash: it is one cell wide in every font that
// has it, so the line does not jitter as it turns.
var spinnerFrames = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

// asciiSpinnerFrames are the fallback. Distinct shapes, not a rotating
// character, because a terminal without Braille usually has no fine spacing
// either and the motion has to read at a glance.
var asciiSpinnerFrames = []rune{'|', '/', '-', '\\'}

// Spinner returns the frame for a tick. Pure over the counter, so a golden test
// pins an exact frame instead of racing a clock.
func Spinner(frame int, unicode bool) string {
	set := spinnerFrames
	if !unicode {
		set = asciiSpinnerFrames
	}
	if frame < 0 {
		frame = -frame
	}
	return string(set[frame%len(set)])
}

// humanTokens shortens a token count to something scannable. Exact numbers
// below a thousand, because there the exact figure is still readable.
func humanTokens(n int) string {
	switch {
	case n <= 0:
		return ""
	case n < 1000:
		return strconv.Itoa(n)
	case n < 1_000_000:
		return strconv.FormatFloat(float64(n)/1000, 'f', 1, 64) + "k"
	default:
		return strconv.FormatFloat(float64(n)/1_000_000, 'f', 1, 64) + "M"
	}
}

// Styled strings carry escape sequences that occupy no display cells, so width
// has to be measured on the text with the escapes removed. Measuring the raw
// string is how a coloured line silently overflows a column that a monochrome
// one fits.

// visibleWidth measures display cells, ignoring escape sequences.
func visibleWidth(s string) int {
	return runewidth.StringWidth(stripANSI(s))
}

// stripANSI removes CSI sequences. Only what this package emits, deliberately:
// a general terminal parser here would be a lot of surface for no gain.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && !(s[j] >= '@' && s[j] <= '~') {
				j++
			}
			if j < len(s) {
				j++
			}
			i = j
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// clipStyled truncates to w display cells, keeping escapes intact.
//
// It also flattens the line, and that is the load-bearing half. Every line of
// every column passes through here, so this is the one place that can promise
// what the layout assumes everywhere else: a line is a line. A newline that got
// this far — from a tool called with a shell command wrapped over four lines,
// say — used to be written straight to the screen, where it pushed the rest of
// the row into the next one and took the sidebar, the divider and the panel
// with it. The frame came apart on a `curl` with a trailing backslash.
func clipStyled(s string, w int) string {
	s = flatten(s)
	if visibleWidth(s) <= w {
		return s
	}
	// Truncating inside an escape would leave the terminal in that style for
	// the rest of the screen, so a reset is appended — but only when there was
	// an escape to begin with. Emitting one unconditionally puts escape bytes
	// in monochrome output, which is exactly what NO_COLOR asked not to happen.
	styled := strings.IndexByte(s, 0x1b) >= 0
	var b strings.Builder
	cells := 0
	for i := 0; i < len(s); {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && !(s[j] >= '@' && s[j] <= '~') {
				j++
			}
			if j < len(s) {
				j++
			}
			b.WriteString(s[i:j])
			i = j
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		rw := runewidth.RuneWidth(r)
		if cells+rw > w {
			break
		}
		b.WriteRune(r)
		cells += rw
		i += size
	}
	if styled {
		b.WriteString("\x1b[0m")
	}
	return b.String()
}

// padStyled pads to w display cells.
func padStyled(s string, w int) string {
	d := w - visibleWidth(s)
	if d <= 0 {
		return clipStyled(s, w)
	}
	return s + strings.Repeat(" ", d)
}

// ContextLabel renders the context meter.
//
// A percentage alone disappears on a large window: five thousand tokens of a
// million is zero in integer division, so the meter a user most wants early in
// a long session is exactly the one that never shows. Below one percent it says
// so rather than rounding to nothing.
func ContextLabel(used, window int) string {
	if used <= 0 || window <= 0 {
		return ""
	}
	pct := 100 * used / window
	if pct < 1 {
		return "ctx <1%"
	}
	return "ctx " + strconv.Itoa(pct) + "%"
}

// flatten turns a value that may span lines into one line.
//
// A space rather than a marker: what breaks is a field that was never meant to
// hold a newline, so the reader is better served by the words running together
// than by punctuation implying something was cut. What was actually cut, if
// anything, is said by the ellipsis the caller already applies.
func flatten(s string) string {
	if !strings.ContainsAny(s, "\n\r\t") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	space := false
	for _, r := range s {
		switch r {
		case '\n', '\r', '\t':
			space = true
			continue
		}
		if space && b.Len() > 0 {
			b.WriteByte(' ')
		}
		space = false
		b.WriteRune(r)
	}
	return b.String()
}
