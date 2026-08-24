package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func choices() []SessionChoice {
	at := time.Date(2026, 8, 18, 14, 46, 0, 0, time.UTC)
	return []SessionChoice{
		{ID: "1a015fb", Title: "tem acesso a rede?", Turns: 2, When: at},
		{ID: "1a014aa", Title: "arruma o parser", Turns: 11, When: at.Add(-2 * time.Hour)},
		{ID: "1a013cc", Title: "sobe a cobertura", Turns: 4, When: at.Add(-26 * time.Hour)},
	}
}

// The list is what a person chooses from, so every row carries the question
// that was asked. A column of ids is a column nobody picks from.
func TestThePickerShowsWhatEachConversationWasAbout(t *testing.T) {
	out := RenderPicker(NewPicker(choices(), En), DefaultGeometry(100, 30))
	for _, want := range []string{"tem acesso a rede?", "arruma o parser", "sobe a cobertura"} {
		if !strings.Contains(out, want) {
			t.Errorf("the list does not carry %q:\n%s", want, out)
		}
	}
}

// Moving stops at both ends rather than wrapping. Wrapping in a list somebody
// is reading turns "one too far" into "somewhere else entirely".
func TestTheCursorStopsAtBothEnds(t *testing.T) {
	p := NewPicker(choices(), En)
	if p.Cursor != 0 {
		t.Fatalf("the picker opened at %d, want the newest", p.Cursor)
	}
	if got := p.Move(-1).Cursor; got != 0 {
		t.Errorf("moving up from the top landed at %d", got)
	}
	end := p.Move(1).Move(1).Move(1).Move(1)
	if end.Cursor != len(choices())-1 {
		t.Errorf("moving past the bottom landed at %d", end.Cursor)
	}
}

// Choosing returns the session under the cursor, and cancelling returns
// nothing. "Nothing" has to be distinguishable from "the first one", or Esc
// silently opens a conversation.
func TestChoosingReturnsTheOneUnderTheCursorAndCancellingReturnsNothing(t *testing.T) {
	p := NewPicker(choices(), En).Move(1)
	if got := p.Chosen(); got != "1a014aa" {
		t.Errorf("chose %q, want the row under the cursor", got)
	}
	if got := NewPicker(nil, En).Chosen(); got != "" {
		t.Errorf("an empty picker chose %q", got)
	}
}

// The row under the cursor is marked without colour. A list whose selection is
// only a colour is a list that cannot be used over a pipe or by somebody who
// turned colour off.
func TestTheSelectedRowIsMarkedWithoutColour(t *testing.T) {
	geo := DefaultGeometry(100, 30)
	geo.Palette = Palette{Enabled: false}

	first := RenderPicker(NewPicker(choices(), En), geo)
	second := RenderPicker(NewPicker(choices(), En).Move(1), geo)
	if first == second {
		t.Errorf("moving the cursor changed nothing on a monochrome terminal:\n%s", first)
	}
}

// Every row fits its terminal, at any width. A long question is what people
// actually type.
func TestNoRowExceedsTheTerminal(t *testing.T) {
	long := []SessionChoice{{
		ID:    "1a015fb",
		Title: strings.Repeat("uma pergunta bem comprida ", 20),
		Turns: 3, When: time.Now(),
	}}
	for _, w := range []int{40, 60, 80, 100, 200} {
		geo := DefaultGeometry(w, 30)
		for _, line := range strings.Split(RenderPicker(NewPicker(long, En), geo), "\n") {
			if got := clipWidth(line); got > w {
				t.Errorf("at width %d a row is %d wide: %q", w, got, line)
			}
		}
	}
}

// Nothing to choose from says so. An empty frame with keys at the bottom is a
// list somebody waits at.
func TestAnEmptyPickerSaysThereIsNothingToChoose(t *testing.T) {
	out := RenderPicker(NewPicker(nil, En), DefaultGeometry(80, 24))
	if strings.TrimSpace(out) == "" {
		t.Fatal("an empty picker rendered nothing at all")
	}
	if !strings.Contains(out, Text(En).PickerEmpty) {
		t.Errorf("it does not say the list is empty:\n%s", out)
	}
}

// More conversations than rows scrolls, keeping the cursor on screen.
func TestALongListScrollsAndKeepsTheCursorVisible(t *testing.T) {
	var many []SessionChoice
	for i := range 40 {
		many = append(many, SessionChoice{
			ID: string(rune('a'+i%26)) + "x", Title: "conversa " + string(rune('a'+i%26)),
			Turns: i, When: time.Now(),
		})
	}
	p := NewPicker(many, En)
	for range 30 {
		p = p.Move(1)
	}
	geo := DefaultGeometry(80, 12)
	out := RenderPicker(p, geo)
	if lines := strings.Count(out, "\n") + 1; lines > geo.Height {
		t.Errorf("the picker drew %d lines into %d", lines, geo.Height)
	}
	if !strings.Contains(out, many[p.Cursor].Title) {
		t.Errorf("the selected row scrolled off the screen:\n%s", out)
	}
}

// The keys somebody reaches for all move the cursor: arrows for the person who
// uses arrows, jk for the person who uses vim, and the readline pair for the
// person whose fingers already know it.
func TestEveryMovementKeyMovesTheCursor(t *testing.T) {
	for _, k := range []tea.KeyPressMsg{special(tea.KeyDown), key("j"), ctrl('n')} {
		p := &pickerProgram{picker: NewPicker(choices(), En), geo: DefaultGeometry(80, 24)}
		p.Update(k)
		if p.picker.Cursor != 1 {
			t.Errorf("%v left the cursor at %d", k, p.picker.Cursor)
		}
	}
	for _, k := range []tea.KeyPressMsg{special(tea.KeyUp), key("k"), ctrl('p')} {
		p := &pickerProgram{picker: NewPicker(choices(), En).Move(2), geo: DefaultGeometry(80, 24)}
		p.Update(k)
		if p.picker.Cursor != 1 {
			t.Errorf("%v left the cursor at %d", k, p.picker.Cursor)
		}
	}
}

// Enter takes the row under the cursor and closes the list.
func TestEnterChoosesAndCloses(t *testing.T) {
	p := &pickerProgram{picker: NewPicker(choices(), En).Move(1), geo: DefaultGeometry(80, 24)}
	if _, cmd := p.Update(special(tea.KeyEnter)); cmd == nil {
		t.Error("choosing did not close the list")
	}
	if p.chosen != "1a014aa" {
		t.Errorf("chose %q", p.chosen)
	}
}

// Every way out leaves without choosing. Opening the first row for somebody who
// pressed Esc is choosing on their behalf at the moment they declined to.
func TestEveryWayOutOfThePickerChoosesNothing(t *testing.T) {
	for _, k := range []tea.KeyPressMsg{special(tea.KeyEscape), key("q"), ctrl('c')} {
		p := &pickerProgram{picker: NewPicker(choices(), En).Move(1), geo: DefaultGeometry(80, 24)}
		if _, cmd := p.Update(k); cmd == nil {
			t.Errorf("%v did not close the list", k)
		}
		if p.chosen != "" {
			t.Errorf("%v chose %q", k, p.chosen)
		}
	}
}

// The list follows the window, so a terminal resized while it is open keeps
// drawing rows that fit.
func TestThePickerFollowsTheWindow(t *testing.T) {
	p := &pickerProgram{picker: NewPicker(choices(), En), geo: DefaultGeometry(80, 24)}
	p.Update(tea.WindowSizeMsg{Width: 42, Height: 10})
	if p.geo.Width != 42 || p.geo.Height != 10 {
		t.Fatalf("the geometry is %dx%d", p.geo.Width, p.geo.Height)
	}
	_ = p.View()
	for _, line := range strings.Split(RenderPicker(p.picker, p.geo), "\n") {
		if clipWidth(line) > 42 {
			t.Errorf("a row is %d wide after the resize: %q", clipWidth(line), line)
		}
	}
	if p.Init() != nil {
		t.Error("the picker asked for work before drawing")
	}
}

// When says when without making somebody parse a date: a time today, a word
// yesterday, a date before that.
func TestWhenIsSaidTheWayPeopleSayIt(t *testing.T) {
	now := time.Now()
	if got := relativeDay(now, now, En); !strings.Contains(got, ":") {
		t.Errorf("today reads %q, want a time", got)
	}
	if got := relativeDay(now.AddDate(0, 0, -1), now, En); got != Text(En).PickerYesterday {
		t.Errorf("yesterday reads %q", got)
	}
	if got := relativeDay(now.AddDate(0, 0, -9), now, En); !strings.Contains(got, "/") {
		t.Errorf("last week reads %q, want a date", got)
	}
}

// A session nobody asked anything in still gets a row that reads as something,
// for the case where it is the only thing there.
func TestARowWithNoQuestionStillReads(t *testing.T) {
	p := NewPicker([]SessionChoice{{ID: "x", When: time.Now()}}, En)
	if out := RenderPicker(p, DefaultGeometry(80, 24)); !strings.Contains(out, Text(En).PickerUntitled) {
		t.Errorf("an untitled row reads as nothing:\n%s", out)
	}
}

// An empty list has nothing to move through, and a terminal too small to hold
// a row still draws something. Both are the shapes a real terminal produces at
// the edges, and neither may panic.
func TestThePickerSurvivesAnEmptyListAndATinyTerminal(t *testing.T) {
	if got := NewPicker(nil, En).Move(1).Cursor; got != 0 {
		t.Errorf("moving an empty list landed at %d", got)
	}
	for _, geo := range []Geometry{
		DefaultGeometry(0, 0), DefaultGeometry(1, 1), DefaultGeometry(12, 3),
	} {
		out := RenderPicker(NewPicker(choices(), En), geo)
		for _, line := range strings.Split(out, "\n") {
			if w := geo.Width; w > 0 && clipWidth(line) > w {
				t.Errorf("at %dx%d a row is %d wide: %q", geo.Width, geo.Height, clipWidth(line), line)
			}
		}
	}
}
