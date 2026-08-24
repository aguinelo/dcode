package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

// SessionChoice is one recorded conversation, as somebody choosing needs to
// see it.
//
// Deliberately not the server's summary type: the client renders and does not
// learn what a session is. Whoever opens the picker maps one to the other, and
// that mapping is the only place the two vocabularies meet.
type SessionChoice struct {
	ID string
	// Title is derived from the first question; Name is what a person called
	// it. Kept apart so the listing can say which it is showing — a derived
	// title and a chosen one are not the same claim.
	Title string
	Name  string
	Turns int
	When  time.Time
}

// Picker is the list, and where the cursor is in it.
type Picker struct {
	Choices []SessionChoice
	Cursor  int
	Lang    Lang
}

// NewPicker opens on the newest, which is what somebody continuing almost
// always wants — the same reasoning that orders the list.
func NewPicker(choices []SessionChoice, lang Lang) Picker {
	return Picker{Choices: choices, Lang: lang}
}

// Move walks the cursor and stops at both ends.
//
// Wrapping would turn "one too far" into "somewhere else entirely", and this is
// a list where landing somewhere else opens the wrong afternoon's work.
func (p Picker) Move(d int) Picker {
	if len(p.Choices) == 0 {
		return p
	}
	p.Cursor += d
	if p.Cursor < 0 {
		p.Cursor = 0
	}
	if p.Cursor > len(p.Choices)-1 {
		p.Cursor = len(p.Choices) - 1
	}
	return p
}

// Chosen is the session under the cursor, empty when there is nothing to
// choose. Empty has to be distinguishable from the first row, or cancelling
// silently opens a conversation.
func (p Picker) Chosen() string {
	if len(p.Choices) == 0 {
		return ""
	}
	return p.Choices[p.Cursor].ID
}

// pickerChrome is the header and the key line: two rows that are always there,
// whatever the list does.
const pickerChrome = 3

// RenderPicker draws the list. Pure over the picker and the geometry, like
// every other surface here — the property that makes a screen testable.
func RenderPicker(p Picker, geo Geometry) string {
	t := Text(p.Lang)
	w := geo.Width
	if w < 1 {
		w = 1
	}

	var b strings.Builder
	b.WriteString(geo.Palette.Apply(StyleBold, clip(t.PickerTitle, w)) + "\n\n")

	if len(p.Choices) == 0 {
		b.WriteString(clip(t.PickerEmpty, w) + "\n")
		return strings.TrimRight(b.String(), "\n")
	}

	rows := geo.Height - pickerChrome - 1
	if rows < 1 {
		rows = 1
	}
	first := 0
	// Keep the cursor on screen by scrolling the window rather than the cursor:
	// a list that moves the selection to stay visible is a list that opens the
	// wrong session.
	if p.Cursor >= rows {
		first = p.Cursor - rows + 1
	}
	last := min(first+rows, len(p.Choices))

	for i := first; i < last; i++ {
		b.WriteString(pickerRow(p, i, w, geo) + "\n")
	}
	b.WriteString("\n" + geo.Palette.Apply(StyleDim, clip(t.PickerKeys, w)))
	return b.String()
}

// pickerRow draws one conversation: the mark, the question, and what it cost.
func pickerRow(p Picker, i, w int, geo Geometry) string {
	c := p.Choices[i]

	// The mark is a character, not a colour. A selection that only exists as
	// colour cannot be seen with colour off, and this list is unusable then.
	mark := "  "
	style := StyleNone
	if i == p.Cursor {
		mark = "> "
		style = StyleAccent
	}

	meta := fmt.Sprintf("%s %s %s", relativeDay(c.When, time.Now(), p.Lang),
		glyphs(geo.Unicode).dot, turnCount(c.Turns, p.Lang))
	// The question gets whatever the meta does not need, and never less than
	// nothing: at 40 columns the meta alone can fill the row.
	room := w - runeLen(mark) - runeLen(meta) - 2
	if room < 0 {
		room = 0
	}
	title := c.Title
	if title == "" {
		title = Text(p.Lang).PickerUntitled
	}
	line := mark + pad(clip(title, room), room) + "  " + meta
	return geo.Palette.Apply(style, clip(line, w))
}

func runeLen(s string) int { return clipWidth(s) }

// relativeDay says when without making somebody parse a date.
//
// The clock is an ARGUMENT. The picker is its own program and could read one
// directly, but the same line is drawn inside the main render now, and that one
// is pure over the model and the geometry — a clock read inside it makes two
// draws of the same state differ, and the symptom is a flicker nobody can
// reproduce. Model.Now exists for exactly this.
func relativeDay(at, now time.Time, lang Lang) string {
	t := Text(lang)
	switch days := int(now.Truncate(24*time.Hour).Sub(at.Truncate(24*time.Hour)).Hours() / 24); {
	case days <= 0:
		return at.Format("15:04")
	case days == 1:
		return t.PickerYesterday
	default:
		return at.Format("02/01")
	}
}

func turnCount(n int, lang Lang) string {
	t := Text(lang)
	return plural(n, t.PickerTurnOne, t.PickerTurnMany)
}

// pickerProgram is the picker as a bubbletea program: the smallest possible
// wrapper around the pure model above.
type pickerProgram struct {
	picker Picker
	geo    Geometry
	chosen string
}

func (p *pickerProgram) Init() tea.Cmd { return nil }

func (p *pickerProgram) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		p.geo.Width, p.geo.Height = m.Width, m.Height
	case tea.KeyPressMsg:
		switch m.String() {
		case "up", "k", "ctrl+p":
			p.picker = p.picker.Move(-1)
		case "down", "j", "ctrl+n":
			p.picker = p.picker.Move(1)
		case "enter":
			p.chosen = p.picker.Chosen()
			return p, tea.Quit
		case "esc", "q", "ctrl+c":
			// Cancelling starts fresh rather than continuing the first row.
			// Silently opening a conversation nobody chose is worse than
			// opening none.
			p.chosen = ""
			return p, tea.Quit
		}
	}
	return p, nil
}

func (p *pickerProgram) View() tea.View {
	return tea.NewView(RenderPicker(p.picker, p.geo))
}

// Pick asks which conversation to continue and returns its id, empty when the
// person chose none.
func Pick(ctx context.Context, choices []SessionChoice, geo Geometry, lang Lang) (string, error) {
	p := &pickerProgram{picker: NewPicker(choices, lang), geo: geo}
	if _, err := tea.NewProgram(p, tea.WithContext(ctx)).Run(); err != nil {
		return "", err
	}
	return p.chosen, nil
}
