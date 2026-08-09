package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

func thinking(t *testing.T, m Model, fragments ...string) Model {
	t.Helper()
	for i, f := range fragments {
		m = m.Apply(ev(t, uint64(i+1), protocol.EventMessageReasoning,
			protocol.MessageReasoning{TurnID: "t1", Text: f}))
	}
	return m
}

// Thinking arrives in fragments and reads as thinking only when it flows, so it
// accumulates into one entry rather than a line per fragment.
func TestThoughtFragmentsAccumulate(t *testing.T) {
	m := thinking(t, NewModel("s", "/w", "m", "read-only"),
		"Preciso ", "ler o handler ", "antes de editar.")

	if len(m.Entries) != 1 {
		t.Fatalf("got %d entries", len(m.Entries))
	}
	if m.Entries[0].Kind != KindReasoning {
		t.Errorf("got %s", m.Entries[0].Kind)
	}
	if m.Entries[0].Summary != "Preciso ler o handler antes de editar." {
		t.Errorf("got %q", m.Entries[0].Summary)
	}
}

// A thought closes when the turn moves on. Live it is the most informative
// thing on screen; once the model has acted it is scratch work sitting in front
// of the result.
func TestAThoughtClosesWhenTheTurnMovesOn(t *testing.T) {
	for name, closer := range map[string]protocol.Event{
		"an answer": ev(t, 9, protocol.EventMessageDelta, protocol.MessageDelta{Text: "pronto"}),
		"a tool call": ev(t, 9, protocol.EventToolRequested,
			protocol.ToolRequested{Name: "read"}),
		"the end of the turn": ev(t, 9, protocol.EventTurnCompleted,
			protocol.TurnCompleted{TurnID: "t1"}),
	} {
		m := thinking(t, NewModel("s", "/w", "m", "read-only"), "pensando")
		if m.Entries[0].Closed {
			t.Fatalf("%s: setup — the thought should still be open", name)
		}
		m = m.Apply(closer)
		if !m.Entries[0].Closed {
			t.Errorf("%s: the thought must close", name)
		}
	}
}

// Once closed it is one line, because thinking runs several times the length of
// the answer and left expanded it buries the result it was leading to.
func TestAClosedThoughtIsOneLine(t *testing.T) {
	m := NewModel("s", "/w", "m", "read-only")
	m.Now = time.Unix(1000, 0)
	m = thinking(t, m, strings.Repeat("uma linha inteira de raciocínio. ", 30))
	m.Now = time.Unix(1004, 0)
	m = m.Apply(ev(t, 9, protocol.EventMessageDelta, protocol.MessageDelta{Text: "a resposta"}))

	g := DefaultGeometry(100, 20)
	rows := lines(Render(m, g))

	var thoughtRows int
	for _, r := range rows {
		if strings.Contains(r, "thought") || strings.Contains(r, "raciocínio") {
			thoughtRows++
		}
	}
	if thoughtRows != 1 {
		t.Errorf("a closed thought is one line, got %d:\n%s", thoughtRows, Render(m, g))
	}
	got := Render(m, g)
	if !strings.Contains(got, "thought for 4.0s") {
		t.Errorf("how long it thought is the useful part:\n%s", got)
	}
	if !strings.Contains(got, "a resposta") {
		t.Errorf("and the answer must be there:\n%s", got)
	}
	// Tab is what opens it, and the line has to say so.
	if !strings.Contains(got, "Tab") {
		t.Errorf("got:\n%s", got)
	}
}

// While open it streams, and only the tail: what it is thinking now is what
// matters, the same reason the stream follows its own end.
func TestAnOpenThoughtStreamsItsTail(t *testing.T) {
	m := NewModel("s", "/w", "m", "read-only")
	var b strings.Builder
	for i := 0; i < 40; i++ {
		b.WriteString("linha de pensamento número ")
		b.WriteString(strings.Repeat("x", 20))
		b.WriteString(". ")
	}
	m = thinking(t, m, b.String())

	g := DefaultGeometry(100, 30)
	g.ThoughtLines = 4
	got := Render(m, g)

	var shown int
	for _, r := range lines(got) {
		if strings.Contains(r, "pensamento") {
			shown++
		}
	}
	if shown == 0 || shown > 4 {
		t.Errorf("want at most 4 lines of live thought, got %d:\n%s", shown, got)
	}
	if n := widest(got); n > 100 {
		t.Errorf("the thought overflowed: %d cells", n)
	}
}

func TestAThoughtExpandsOnTab(t *testing.T) {
	m := NewModel("s", "/w", "m", "read-only")
	m = thinking(t, m, "primeira parte do raciocínio. segunda parte que só aparece expandida.")
	m = m.Apply(ev(t, 9, protocol.EventTurnCompleted, protocol.TurnCompleted{TurnID: "t1"}))

	g := DefaultGeometry(100, 20)
	if strings.Contains(Render(m, g), "segunda parte") {
		t.Fatal("closed, the body must be hidden")
	}
	m = m.ToggleAt(0)
	if !strings.Contains(Render(m, g), "segunda parte") {
		t.Error("Tab must reveal it, or the hint is a lie")
	}
}

// A second thought in the same turn is its own entry: the model thought, acted,
// and thought again, and collapsing those into one would misreport the sequence.
func TestASecondThoughtIsItsOwnEntry(t *testing.T) {
	m := NewModel("s", "/w", "m", "read-only")
	m = thinking(t, m, "primeiro")
	m = m.Apply(ev(t, 5, protocol.EventToolRequested, protocol.ToolRequested{Name: "read"}))
	m = thinking(t, m, "segundo")

	var thoughts int
	for _, e := range m.Entries {
		if e.Kind == KindReasoning {
			thoughts++
		}
	}
	if thoughts != 2 {
		t.Errorf("got %d", thoughts)
	}
}

// The invariant that survives everything: thinking is not the answer, and the
// two must never be confused on screen.
func TestAThoughtIsNeverTheAnswer(t *testing.T) {
	m := NewModel("s", "/w", "m", "read-only")
	m = thinking(t, m, "vou deletar tudo, talvez")
	m = m.Apply(ev(t, 9, protocol.EventMessageDelta, protocol.MessageDelta{Text: "Não vou fazer isso."}))

	for _, e := range m.Entries {
		if e.Kind == KindAssistant && strings.Contains(e.Summary, "deletar tudo") {
			t.Fatal("the thinking became the answer")
		}
	}
	var answers int
	for _, e := range m.Entries {
		if e.Kind == KindAssistant {
			answers++
		}
	}
	if answers != 1 {
		t.Errorf("got %d assistant entries", answers)
	}
}

func TestReasoningDegradesToASCII(t *testing.T) {
	m := NewModel("s", "/w", "m", "read-only")
	m = thinking(t, m, "pensando")
	m = m.Apply(ev(t, 9, protocol.EventTurnCompleted, protocol.TurnCompleted{TurnID: "t1"}))

	g := DefaultGeometry(100, 20)
	g.Unicode = false
	got := Render(m, g)
	if strings.ContainsRune(got, '✻') {
		t.Errorf("the ASCII rendering must not smuggle a glyph in:\n%s", got)
	}
	if !strings.Contains(got, "thought") {
		t.Errorf("and it must still say what it is:\n%s", got)
	}
}
