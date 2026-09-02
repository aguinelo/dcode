package tui

import "github.com/mattn/go-runewidth"

// ruler is how this package measures, and the only ruler it has.
//
// # The locale is not a fact about the terminal
//
// `go-runewidth` decides in its own `init()` whether the AMBIGUOUS-width
// characters — the ones Unicode says are narrow in a Western context and wide
// in an East Asian one — measure one cell or two. It decides it from the
// locale: `LANG=ja_JP.UTF-8` and every box-drawing glyph becomes two cells wide.
//
// This screen is drawn out of those characters. `│`, `└`, `·`, `▌`, `━`, `…`
// are all ambiguous, and `supportsUnicode` reaches for them on exactly the
// signal that flips the measurement — the locale being UTF-8. So one
// environment variable drove two decisions that have to agree, and drove them
// into a contradiction: pick the box-drawing set, then measure it at double
// width. Every frame came out twice the width of the terminal, in a locale
// nobody here had ever rendered a frame in.
//
// The fix is to notice what the locale actually says. It says which language
// the person reads. It does not say how many cells their terminal gives a
// vertical rule, and no portable signal does. This product already draws that
// distinction for colour: `NO_COLOR` and `TERM=dumb` are inference, and
// `DCODE_COLOR` is the user answering for their own terminal. The difference
// here was that nobody in this repository ever chose — a dependency's `init()`
// chose, from a variable read for something else.
//
// So the glyphs are measured as what they were authored as: one cell. Not
// because every terminal draws them so, but because this layout is built out of
// them at that width, and a terminal that draws them wider needs a different
// GLYPH SET rather than a different arithmetic. That set exists and is one
// variable away: `DCODE_ASCII=1`, where every mark is one cell by construction.
//
// A package-level condition rather than writing the global back: a global this
// package flipped would reach into every other package that measures, and the
// hazard would be the order two `init()`s ran in. This ruler is owned here,
// asked explicitly, and immune to whatever the process's locale says.
var ruler = &runewidth.Condition{EastAsianWidth: false}
