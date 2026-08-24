package tui

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// renderProse draws what the model wrote.
//
// The model writes markdown. The screen used to print it raw, so `**A**`
// arrived with its asterisks and a package name arrived with its backticks —
// every answer with emphasis in it read as though the product had forgotten to
// finish something.
//
// Only two inline forms are understood, and the restriction is the point.
// `**bold**` and “ `code` “ are unambiguous; a single `*` is how every model
// writes a bullet, and treating it as italics would eat the bullet of every
// list. A marker with no partner stays on screen as text, because deleting it
// would delete something that was written on purpose.
func renderProse(text string, w int, g Geometry) []string {
	var out []string
	for _, blk := range splitFences(text) {
		if blk.fenced {
			out = append(out, renderVerbatim(blk.text, w, g)...)
			continue
		}
		out = append(out, wrapStyled(parseInline(blk.text), w, g)...)
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}

// textRun is a stretch of text with one role.
type textRun struct {
	text  string
	style Style
}

// proseBlock is a fenced region or the prose between fences.
type proseBlock struct {
	text   string
	fenced bool
}

// splitFences separates verbatim regions from prose.
//
// An unclosed fence takes the rest of the message: the model opened a block and
// meant everything after it to be one, and re-flowing that as prose would wrap
// a command into something that looks runnable and is not.
func splitFences(text string) []proseBlock {
	lines := strings.Split(text, "\n")
	var out []proseBlock
	var buf []string
	fenced := false

	flush := func() {
		if len(buf) > 0 {
			out = append(out, proseBlock{text: strings.Join(buf, "\n"), fenced: fenced})
			buf = nil
		}
	}
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			flush()
			fenced = !fenced
			continue
		}
		buf = append(buf, l)
	}
	flush()
	return out
}

// renderVerbatim draws a fenced block with the bar a diff gets.
//
// The screen already means "this is exactly what it says" with that shape, and
// a person about to run something needs to see where it starts and ends.
// Nothing is re-flowed: wrapping a command produces a second line that looks
// runnable, and re-indenting changes what gets copied.
func renderVerbatim(text string, w int, g Geometry) []string {
	bar := "│ "
	if !g.Unicode {
		bar = "| "
	}
	var out []string
	for _, l := range strings.Split(text, "\n") {
		body := clipStyled(l, w-visibleWidth(bar)-2)
		out = append(out, "  "+g.Palette.Apply(StyleDim, bar)+body)
	}
	return out
}

// parseInline turns one prose block into runs.
//
// Prose is dim and the technical term returns to normal brightness. That is the
// design's own separation and it costs no new colour — the eye lands on the
// name of the file, not on the sentence around it.
func parseInline(text string) []textRun {
	var runs []textRun
	var plain strings.Builder

	flush := func() {
		if plain.Len() > 0 {
			runs = append(runs, textRun{plain.String(), StyleDim})
			plain.Reset()
		}
	}

	for i := 0; i < len(text); {
		switch {
		case strings.HasPrefix(text[i:], "**"):
			if end := strings.Index(text[i+2:], "**"); end >= 0 {
				flush()
				runs = append(runs, textRun{text[i+2 : i+2+end], StyleBold})
				i += 2 + end + 2
				continue
			}
		case text[i] == '`':
			if end := strings.IndexByte(text[i+1:], '`'); end >= 0 {
				flush()
				// StyleNone: normal brightness against dim prose.
				runs = append(runs, textRun{text[i+1 : i+1+end], StyleNone})
				i += 1 + end + 1
				continue
			}
		}
		plain.WriteByte(text[i])
		i++
	}
	flush()
	return runs
}

// wrapStyled lays runs into lines no wider than w.
//
// The unit is the WORD, and each word keeps the role of the run it came from.
// Styling a whole run and then wrapping it would put the escape on one line and
// the reset on another, which paints the rest of the screen in that colour —
// and measuring the styled text would report a coloured line as too wide.
func wrapStyled(runs []textRun, w int, g Geometry) []string {
	if w <= 0 {
		return []string{""}
	}
	p := g.Palette

	var out []string
	var line strings.Builder
	var width int

	newline := func() {
		out = append(out, line.String())
		line.Reset()
		width = 0
	}
	// blank appends a paragraph break, and refuses a second one.
	//
	// `strings.Split("a\n\n", "\n")` has THREE parts, and the last is the end
	// of the text rather than a paragraph — so a trailing blank line produced
	// two of them. A run boundary made it worse: `antes:\n\n**x**` is parsed as
	// a plain run and a bold run, each split on its own, and the block came out
	// with three blank rows between two sentences.
	//
	// Collapsed here rather than at the caller, because the caller sees lines
	// and this is the only place that knows which of them were paragraph breaks.
	blank := func() {
		if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
			return
		}
		out = append(out, "")
	}
	put := func(word string, style Style, space bool) {
		if space {
			line.WriteString(" ")
			width++
		}
		line.WriteString(p.Apply(style, word))
		width += runewidth.StringWidth(word)
	}

	for _, r := range runs {
		// A blank line in the source is a paragraph break, and the model uses
		// them to separate one thought from the next.
		for pi, para := range strings.Split(r.text, "\n") {
			if pi > 0 {
				if width > 0 {
					newline()
				}
				if strings.TrimSpace(para) == "" {
					blank()
					continue
				}
			}
			for _, word := range strings.Fields(para) {
				// A word wider than the column is broken rather than merely
				// given its own line: a path longer than the stream would
				// otherwise wrap in the terminal and destroy the layout.
				for runewidth.StringWidth(word) > w {
					if width > 0 {
						newline()
					}
					head := runewidth.Truncate(word, w, "")
					put(head, r.style, false)
					newline()
					word = word[len(head):]
				}
				if word == "" {
					continue
				}
				space := width > 0
				cost := runewidth.StringWidth(word)
				if space {
					cost++
				}
				if width+cost > w {
					newline()
					space = false
				}
				put(word, r.style, space)
			}
		}
	}
	if width > 0 {
		newline()
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}
