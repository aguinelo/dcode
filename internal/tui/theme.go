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

// colours is one theme's palette, in the design's own names.
//
// The ROLE MAPPING below is written once and shared by all four. That is what
// makes a theme a theme rather than four screens: change which colour a heading
// is, and every theme changes with it. A per-theme mapping would drift the
// first time somebody added a role.
type colours struct {
	bg, p, ok, info, you, alt, bad string
	fg1, fg2, dim, dim2, dim3      string
	track                          string
}

// Themes are the four the design carries, in the order `t` cycles them. Neon
// first: it is the one this interface is drawn in.
func Themes() []Theme {
	return []Theme{
		build("neon", colours{
			bg: "#120d24", p: "#ff2d95", ok: "#4de0c0", info: "#3fd0ff", you: "#f6c445",
			alt: "#c77cf0", bad: "#ff5c7a", fg1: "#efeafd", fg2: "#cfc8e8",
			dim: "#8b82b5", dim2: "#5b4f85", dim3: "#4a3f70", track: "#241a42",
		}),
		build("ashes", colours{
			bg: "#12161a", p: "#5aa9e6", ok: "#7fd1ae", info: "#8ac6f2", you: "#e8c07d",
			alt: "#b48ead", bad: "#e2707f", fg1: "#e6ecf1", fg2: "#c9d3da",
			dim: "#7d8b95", dim2: "#5a666f", dim3: "#46515a", track: "#1e262c",
		}),
		build("ember", colours{
			bg: "#1c1410", p: "#ff7a3d", ok: "#c6c04a", info: "#d99a5b", you: "#ffc35c",
			alt: "#e07a8c", bad: "#e5484d", fg1: "#f2e6d8", fg2: "#dccbb8",
			dim: "#95806e", dim2: "#6d5b4c", dim3: "#54453a", track: "#2b1f18",
		}),
		build("mono", colours{
			bg: "#101012", p: "#e8e8ec", ok: "#a8d8b9", info: "#9fb6d9", you: "#e6d59a",
			alt: "#c0b6d9", bad: "#dd8a94", fg1: "#ececf0", fg2: "#cfcfd6",
			dim: "#82828b", dim2: "#5e5e66", dim3: "#4a4a51", track: "#212126",
		}),
	}
}

// Neon is the interface's own palette: violet ground, magenta mark, teal for
// what worked, amber for the person, coral for what did not.
//
// The values are the design's, not near neighbours of them. A palette copied
// approximately is a palette that reads as a copy.
func Neon() Theme { return Themes()[0] }

// NextTheme is the one after this, wrapping. Wrapping is right here and wrong
// in the conversation list, and the difference is the cost of overshooting: one
// more press, against opening somebody else's afternoon.
func NextTheme(t Theme) Theme {
	all := Themes()
	for i, c := range all {
		if c.Name == t.Name {
			return all[(i+1)%len(all)]
		}
	}
	return all[0]
}

// build turns one palette into the shared role mapping.
func build(name string, c colours) Theme {
	bg := hex(c.bg)
	return Theme{
		Name:   name,
		Ground: bg,
		Role: map[Style]paint{
			// The text hierarchy. Prose is the brightest ordinary thing on the
			// screen, because it is what the reader came for.
			StyleProse:   {fg: hex(c.fg2)},
			StyleCode:    {fg: hex(c.ok)},
			StyleHeading: {fg: hex(c.fg1), bold: true},
			StyleMeta:    {fg: hex(c.dim)},
			StyleHint:    {fg: hex(c.dim2)},
			// Chrome is dim3 and NOT the design's border colour, which measures
			// 1.26:1 against the neon ground. That is right for a one-pixel CSS
			// line on a surface lit by three radial gradients; a terminal rule
			// is a row of solid glyphs on a flat ground, and at 1.26:1 it is
			// not there.
			StyleChrome: {fg: hex(c.dim3)},

			StyleBold: {fg: hex(c.fg1), bold: true},
			StyleDim:  {fg: hex(c.dim)},

			// The product's own mark, and the states.
			StyleAccent:  {fg: hex(c.p)},
			StyleOK:      {fg: hex(c.ok)},
			StyleError:   {fg: hex(c.bad)},
			StyleWarn:    {fg: hex(c.you)},
			StyleAdded:   {fg: hex(c.ok)},
			StyleRemoved: {fg: hex(c.bad)},
			StyleDanger:  {fg: hex(c.fg1), bg: hex(c.bad), bold: true},
			StyleCursor:  {fg: bg, bg: hex(c.you)},

			// The lanes: the person, the work, the answer.
			StyleLaneYou:     {fg: hex(c.you)},
			StyleLaneProcess: {fg: hex(c.dim2)},
			StyleLaneAnswer:  {fg: hex(c.ok)},

			// Reserved treatments the design gives to structure and to the
			// mascot, unchanged in meaning.
			StyleOnAccent:  {fg: bg, bg: hex(c.ok), bold: true},
			StyleHighlight: {fg: hex(c.you)},
			StyleBody:      {fg: hex(c.p)},
			StyleShadow:    {fg: hex(c.alt)},
			StyleEye:       {fg: hex(c.info)},
			StyleTrack:     {fg: hex(c.track)},
		},
	}
}

// sgr renders a paint as an escape body, in truecolor or in the 256-colour
// approximation, or empty when the theme says nothing about this role.
//
// Two depths and not three: a terminal that cannot do 256 colours is a terminal
// this palette cannot be drawn on at all, and pretending otherwise would put a
// coloured ground behind text it cannot tint. Such a terminal gets the
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
