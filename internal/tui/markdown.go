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
//
// The one exception is a marker at the very END of the text, with nothing after
// it. While a turn is streaming, every emphasised word arrives as `**` first
// and its partner some deltas later, so the screen flashed a pair of asterisks
// before each of them — `1. **` sitting alone on a line was the last thing a
// reader saw of every heading the model wrote. Nothing follows an opener at the
// end of the text, so there is nothing for it to be marking up yet.
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
		out = append(out, "  "+g.Palette.Apply(StyleChrome, bar)+body)
	}
	return out
}

// parseInline turns one prose block into runs.
//
// The sentence is READ and the technical term inside it is picked out. It used
// to be the other way round — the sentence dim, the term at normal brightness —
// which does put the eye on the file name, and dims the answer to do it. The
// model's prose is most of what is on the screen, so that faded most of the
// screen, and the one thing the reader came for was the one thing faded.
func parseInline(text string) []textRun {
	var runs []textRun
	var plain strings.Builder

	flush := func() {
		if plain.Len() > 0 {
			runs = append(runs, textRun{plain.String(), StyleProse})
			plain.Reset()
		}
	}

	text = dropUnfinishedMarker(text)

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
				runs = append(runs, textRun{text[i+1 : i+1+end], StyleCode})
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

// dropUnfinishedMarker removes a marker that opened at the end of the text and
// has not been closed yet.
//
// Two conditions, and both are needed. It must be at the END — `**bold` with
// words after it is a marker somebody wrote and left, and deleting that would
// delete something written on purpose. And it must be UNPAIRED, which is what
// the count answers: a text ending in `**` because a pair closed there is a
// finished pair, and the first version of this took the closer off it.
func dropUnfinishedMarker(text string) string {
	trimmed := strings.TrimRight(text, " ")
	for _, mark := range []string{"**", "`"} {
		if strings.HasSuffix(trimmed, mark) && strings.Count(trimmed, mark)%2 == 1 {
			return strings.TrimSuffix(trimmed, mark)
		}
	}
	return text
}
