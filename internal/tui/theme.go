package tui

import (
	"strconv"
	"strings"
)

// The interface has one palette, and it does not paint the ground.
//
// It used to have five: four owned their own background and carried RGB
// measured against it, and this one gave the ground back to the terminal. The
// four were removed rather than kept as options — colour with no meaning is
// decoration, and decoration that owns its own ground is an interface that
// stopped inheriting the terminal it runs in. See
// docs/specs/architecture/client-tui/changelog/202609011200-um-tema-que-nao-pinta-o-chao.md
// for why the surviving one is built the way it is.
//
// What is left is what a terminal keeps over a background nobody chose: three
// weights (bold, faint, italic as a fourth) and the sixteen named colours,
// which are exactly the ones the terminal's own theme already picked to be
// readable against its own ground. Inheriting the ground and inheriting the
// palette are the same decision made once, and it is drawable at any depth a
// terminal has colour at all — the sixteen are the one thing every terminal
// with colour can draw.
type Theme struct {
	Name string
	// Role is the colour of each role. A role absent from the map is drawn
	// without colour, which is how the theme says "normal weight" for prose.
	Role map[Style]paint
}

// paint is a colour plus the attributes that go with it.
//
// Colour is always indexed: one of the sixteen named colours the TERMINAL
// chose, never an RGB this product picked. Indexed colour is the only kind
// that can be trusted to read against a ground this product does not own.
type paint struct {
	fgIdx, bgIdx ansi
	bold         bool
	// italic is SGR 3, the fourth weight. Like faint it survives an unknown
	// ground, and it is what the theme gives StyleReasoning and nothing else.
	italic bool
	// faint is SGR 2, the weight that adapts to a ground the theme did not
	// choose — the one the no-colour and ASCII paths still rely on.
	faint bool
}

// ansi is one of the sixteen named colours, or none at all.
//
// The zero value has to mean "this role carries no colour", which is why the
// colours are offset by one rather than being 0-15 directly: black is a
// colour somebody may choose, and a role left unset must render as nothing.
type ansi uint8

const (
	ansiNone ansi = iota
	ansiBlack
	ansiRed
	ansiGreen
	ansiYellow
	ansiBlue
	ansiMagenta
	ansiCyan
	ansiWhite
	ansiBrightBlack
	ansiBrightRed
	ansiBrightGreen
	ansiBrightYellow
	ansiBrightBlue
	ansiBrightMagenta
	ansiBrightCyan
	ansiBrightWhite
)

// code is the SGR parameter for this colour, as foreground or as background.
func (a ansi) code(background bool) string {
	n := int(a) - 1
	base := 30
	if n >= 8 {
		n, base = n-8, 90
	}
	if background {
		base += 10
	}
	return strconv.Itoa(base + n)
}

// Default is the interface's one theme.
func Default() Theme {
	return Theme{
		Name: "claude",
		Role: map[Style]paint{
			// Text is weight, never a fixed grey: the grey that reads on a dark
			// terminal is the grey that vanishes on a light one, and this theme
			// knows which terminal it is on no better than the reader does.
			StyleProse:     {},
			StyleHeading:   {bold: true},
			StyleBold:      {bold: true},
			StyleMeta:      {faint: true},
			StyleHint:      {faint: true},
			StyleChrome:    {faint: true},
			StyleDim:       {faint: true},
			StyleTrack:     {faint: true},
			StyleReasoning: {faint: true, italic: true},

			// A technical term inside a sentence still buys its contrast with a
			// colour, because weight is already spent on the hierarchy.
			StyleCode: {fgIdx: ansiCyan},

			// State, in the terminal's own colours.
			StyleAccent:  {fgIdx: ansiYellow},
			StyleOK:      {fgIdx: ansiGreen},
			StyleError:   {fgIdx: ansiRed},
			StyleWarn:    {fgIdx: ansiBrightYellow},
			StyleAdded:   {fgIdx: ansiGreen},
			StyleRemoved: {fgIdx: ansiRed},
			StyleDanger:  {fgIdx: ansiBrightWhite, bgIdx: ansiRed, bold: true},
			StyleCursor:  {fgIdx: ansiBlack, bgIdx: ansiYellow},

			StyleLaneYou:     {fgIdx: ansiYellow},
			StyleLaneProcess: {faint: true},
			StyleLaneAnswer:  {fgIdx: ansiGreen},

			StyleOnAccent:  {fgIdx: ansiBlack, bgIdx: ansiGreen, bold: true},
			StyleHighlight: {fgIdx: ansiYellow},
			StyleBody:      {fgIdx: ansiRed},
			StyleShadow:    {fgIdx: ansiMagenta},
			StyleEye:       {fgIdx: ansiCyan},
		},
	}
}

// sgr renders a paint as an escape body, empty when the theme says nothing
// about this role.
func (p paint) sgr() string {
	var parts []string
	if p.bold {
		parts = append(parts, "1")
	}
	if p.faint {
		parts = append(parts, "2")
	}
	if p.italic {
		parts = append(parts, "3")
	}
	if p.fgIdx != ansiNone {
		parts = append(parts, p.fgIdx.code(false))
	}
	if p.bgIdx != ansiNone {
		parts = append(parts, p.bgIdx.code(true))
	}
	return strings.Join(parts, ";")
}

// close undoes exactly what sgr set, and nothing else.
//
// A plain reset would undo more than this role opened, on a terminal whose own
// theme is already painting the row underneath. Closing what was opened costs
// a few bytes and leaves everything else alone.
func (p paint) close() string {
	var parts []string
	if p.bold || p.faint {
		parts = append(parts, "22")
	}
	if p.italic {
		parts = append(parts, "23")
	}
	if p.fgIdx != ansiNone {
		parts = append(parts, "39")
	}
	if p.bgIdx != ansiNone {
		parts = append(parts, "49")
	}
	return strings.Join(parts, ";")
}
