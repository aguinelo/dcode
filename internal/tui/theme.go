package tui

import (
	"fmt"
	"strings"
)

// A theme is the whole palette as colours, and neon is the one this interface
// is drawn in.
//
// Until now the roles mapped to a handful of ANSI codes chosen to sit politely
// inside whatever the terminal's own theme was. That politeness is what made
// the screen read as grey: three weights and one amber, over a background the
// product did not choose, is not a palette — it is an absence of one.
//
// A theme carries its own ground. That is the decision, and it is not free: the
// interface stops inheriting the terminal's colours and starts owning them.
// It is what the design asks for, and what makes an accent an accent — amber
// over an unknown background is a colour; amber over #120d24 is a signal.
//
// Colour that is switched off gets NONE of this: Palette{} still writes nothing
// at all, background included, and the screen falls back to the terminal's own.
// That path is not a second implementation, it is the same code with an empty
// table.
type Theme struct {
	Name string
	// Ground is painted behind every row. Empty means the terminal's own.
	Ground rgb
	// Role is the colour of each role. A role absent from the map is drawn
	// without colour, which is how a theme says "normal weight" for prose.
	Role map[Style]paint
}

// paint is a colour plus the attributes that go with it.
type paint struct {
	fg   rgb
	bg   rgb
	bold bool
	// faint is SGR 2, which no theme colour replaces: it is the one weight that
	// adapts to a ground the theme did not choose, and the ASCII and no-colour
	// paths still rely on it.
	faint bool
}

type rgb struct{ r, g, b uint8 }

func (c rgb) zero() bool { return c == rgb{} }

func hex(s string) rgb {
	var r, g, b uint8
	_, _ = fmt.Sscanf(strings.TrimPrefix(s, "#"), "%02x%02x%02x", &r, &g, &b)
	return rgb{r, g, b}
}

// Neon is the interface's own palette: violet ground, magenta mark, teal for
// what worked, amber for the person, coral for what did not.
//
// The values are the design's, not near neighbours of them. A palette copied
// approximately is a palette that reads as a copy.
func Neon() Theme {
	var (
		bg    = hex("#120d24")
		p     = hex("#ff2d95")
		ok    = hex("#4de0c0")
		info  = hex("#3fd0ff")
		you   = hex("#f6c445")
		alt   = hex("#c77cf0")
		bad   = hex("#ff5c7a")
		fg1   = hex("#efeafd")
		fg2   = hex("#cfc8e8")
		dim   = hex("#8b82b5")
		dim2  = hex("#5b4f85")
		bd    = hex("#4a3f70")
		track = hex("#241a42")
	)
	return Theme{
		Name:   "neon",
		Ground: bg,
		Role: map[Style]paint{
			// The text hierarchy. Prose is the brightest ordinary thing on the
			// screen, because it is what the reader came for.
			StyleProse:   {fg: fg2},
			StyleCode:    {fg: ok},
			StyleHeading: {fg: fg1, bold: true},
			StyleMeta:    {fg: dim},
			StyleHint:    {fg: dim2},
			// The design's own border colour is #2a1f4a, which measures 1.26:1
			// against this ground. That is right for a one-pixel CSS line on a
			// surface lit by three radial gradients; a terminal rule is a row
			// of solid glyphs on a flat ground, and at 1.26:1 it is not there.
			// This is the design's `dim3` instead, at 2.02:1 — visible, and
			// still quiet enough not to compete with what it separates.
			StyleChrome: {fg: bd},

			StyleBold: {fg: fg1, bold: true},
			StyleDim:  {fg: dim},

			// The product's own mark, and the states.
			StyleAccent:  {fg: p},
			StyleOK:      {fg: ok},
			StyleError:   {fg: bad},
			StyleWarn:    {fg: you},
			StyleAdded:   {fg: ok},
			StyleRemoved: {fg: bad},
			StyleDanger:  {fg: fg1, bg: bad, bold: true},
			StyleCursor:  {fg: bg, bg: you},

			// The lanes: the person in amber, the work in violet-grey, the
			// answer in teal.
			StyleLaneYou:     {fg: you},
			StyleLaneProcess: {fg: dim2},
			StyleLaneAnswer:  {fg: ok},

			// Reserved treatments the design gives to structure and to the
			// mascot, unchanged in meaning.
			StyleOnAccent:  {fg: bg, bg: ok, bold: true},
			StyleHighlight: {fg: you},
			StyleBody:      {fg: p},
			StyleShadow:    {fg: alt},
			StyleEye:       {fg: info},
			StyleTrack:     {fg: track},
		},
	}
}

// sgr renders a paint as an escape body, in truecolor or in the 256-colour
// approximation, or empty when the theme says nothing about this role.
//
// Two depths and not three: a terminal that cannot do 256 colours is a terminal
// this palette cannot be drawn on at all, and pretending otherwise would put a
// violet ground behind text it cannot tint. Such a terminal gets the
// no-colour path, which is a screen this product is tested to be readable on.
func (p paint) sgr(depth Depth) string {
	var parts []string
	if p.bold {
		parts = append(parts, "1")
	}
	if p.faint {
		parts = append(parts, "2")
	}
	if !p.fg.zero() {
		parts = append(parts, colour(p.fg, depth, false))
	}
	if !p.bg.zero() {
		parts = append(parts, colour(p.bg, depth, true))
	}
	return strings.Join(parts, ";")
}

// close undoes exactly what sgr set, and nothing else.
//
// A plain reset would undo the GROUND too — the background is painted once per
// row, and SGR 0 after every styled run punched a hole in it, which the first
// version patched by re-emitting the ground after every reset. That worked and
// cost a full background escape per word: on a 100-column screen of prose it
// tripled the bytes of every frame, repainted every 120ms.
//
// Closing what was opened costs four bytes and leaves the row's ground alone.
func (p paint) close() string {
	var parts []string
	if p.bold || p.faint {
		parts = append(parts, "22")
	}
	if !p.fg.zero() {
		parts = append(parts, "39")
	}
	if !p.bg.zero() {
		parts = append(parts, "49")
	}
	return strings.Join(parts, ";")
}

func colour(c rgb, depth Depth, background bool) string {
	lead := "38"
	if background {
		lead = "48"
	}
	if depth == DepthTrue {
		return fmt.Sprintf("%s;2;%d;%d;%d", lead, c.r, c.g, c.b)
	}
	return fmt.Sprintf("%s;5;%d", lead, cube(c))
}

// cube maps a colour into the 6×6×6 cube, or into the grey ramp when it is
// close enough to grey that the cube would tint it.
//
// The greys matter more than the hues here: half this palette is violet-grey
// text, and a grey pushed into the colour cube comes back faintly green.
func cube(c rgb) int {
	r, g, b := int(c.r), int(c.g), int(c.b)
	if max3(r, g, b)-min3(r, g, b) < 12 {
		lvl := (r + g + b) / 3
		if lvl < 8 {
			return 16
		}
		if lvl > 248 {
			return 231
		}
		return 232 + (lvl-8)*24/240
	}
	q := func(v int) int { return (v*5 + 127) / 255 }
	return 16 + 36*q(r) + 6*q(g) + q(b)
}

func max3(a, b, c int) int {
	if b > a {
		a = b
	}
	if c > a {
		a = c
	}
	return a
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}
