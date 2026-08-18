package loop

import (
	"context"
	"strings"
	"testing"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/provider"
	"github.com/aguinelo/dcode/internal/tools"
)

func sawWorthRemembering(msgs []ce.Message) bool {
	for _, m := range msgs {
		if m.Reminder && strings.Contains(m.Text, "same wall twice") {
			return true
		}
	}
	return false
}

// Once is a mistake; twice is the repository teaching something nobody wrote
// down, and that is the moment to ask for it to be written down.
//
// This exists because measurement said the prompt was not enough: across four
// scenario designs the model never once called `remember` on its own. A fifth
// sentence in the doctrine would have been the third time that approach failed.
func TestTheSameWallTwiceAsksForItToBeRemembered(t *testing.T) {
	reg := tools.NewRegistry(tools.Read{}, tools.Remember{})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "read", `{"path":"nope.go"}`), done()},
		{call("c2", "read", `{"path":"nope.go"}`), done()},
		{text("I see"), done()},
	}}
	e, _ := newEngine(t, p, reg, func(c *Config) { c.Reminders = true })

	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}
	if !sawWorthRemembering(e.Session().History) {
		t.Error("the same failure twice on the same path asked for nothing")
	}
}

// Two different files failing once each is two ordinary misses. Firing there
// would make the reminder noise, and a reminder that is noise gets skipped.
func TestTwoDifferentFailuresAskForNothing(t *testing.T) {
	reg := tools.NewRegistry(tools.Read{}, tools.Remember{})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "read", `{"path":"one.go"}`), done()},
		{call("c2", "read", `{"path":"two.go"}`), done()},
		{text("moving on"), done()},
	}}
	e, _ := newEngine(t, p, reg, func(c *Config) { c.Reminders = true })

	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}
	if sawWorthRemembering(e.Session().History) {
		t.Error("two unrelated misses were read as a wall")
	}
}

// Once per turn. Repeating it every round is nagging about a fact that has not
// changed, and a reminder that repeats is one the model learns to skip.
func TestTheWallIsMentionedOnce(t *testing.T) {
	reg := tools.NewRegistry(tools.Read{}, tools.Remember{})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "read", `{"path":"nope.go"}`), done()},
		{call("c2", "read", `{"path":"nope.go"}`), done()},
		{call("c3", "read", `{"path":"nope.go"}`), done()},
		{call("c4", "read", `{"path":"nope.go"}`), done()},
		{text("done"), done()},
	}}
	e, _ := newEngine(t, p, reg, func(c *Config) { c.Reminders = true })

	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, m := range e.Session().History {
		if m.Reminder && strings.Contains(m.Text, "same wall twice") {
			n++
		}
	}
	if n != 1 {
		t.Errorf("mentioned %d times, want once", n)
	}
}

// A build with no `remember` is asked for nothing. Telling the model to call a
// tool this session does not carry sends it somewhere that does not exist —
// the defect this codebase already fixed once in a tool error message.
func TestWithoutTheToolNothingIsAsked(t *testing.T) {
	reg := tools.NewRegistry(tools.Read{})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "read", `{"path":"nope.go"}`), done()},
		{call("c2", "read", `{"path":"nope.go"}`), done()},
		{text("ok"), done()},
	}}
	e, _ := newEngine(t, p, reg, func(c *Config) { c.Reminders = true })

	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}
	if sawWorthRemembering(e.Session().History) {
		t.Error("a session with no remember tool was told to call it")
	}
}

// No count in the text. A number that varies between identical runs breaks the
// reproducibility the context engine guarantees (RN-7).
func TestTheWallNoticeCarriesNoCount(t *testing.T) {
	reg := tools.NewRegistry(tools.Read{}, tools.Remember{})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "read", `{"path":"nope.go"}`), done()},
		{call("c2", "read", `{"path":"nope.go"}`), done()},
		{call("c3", "read", `{"path":"nope.go"}`), done()},
		{text("ok"), done()},
	}}
	e, _ := newEngine(t, p, reg, func(c *Config) { c.Reminders = true })
	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}
	for _, m := range e.Session().History {
		if m.Reminder && strings.Contains(m.Text, "same wall twice") {
			for _, digit := range "0123456789" {
				if strings.ContainsRune(m.Text, digit) {
					t.Errorf("the notice carries a number: %q", m.Text)
					return
				}
			}
		}
	}
}

// It asks for what the repository teaches, not for a diary of mistakes. A
// memory of an ordinary error is noise the next session pays for.
func TestTheNoticeSaysWhatNotToRemember(t *testing.T) {
	reg := tools.NewRegistry(tools.Read{}, tools.Remember{})
	p := &scriptedProvider{turns: [][]provider.StreamEvent{
		{call("c1", "read", `{"path":"nope.go"}`), done()},
		{call("c2", "read", `{"path":"nope.go"}`), done()},
		{text("ok"), done()},
	}}
	e, _ := newEngine(t, p, reg, func(c *Config) { c.Reminders = true })
	if _, err := e.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}
	for _, m := range e.Session().History {
		if m.Reminder && strings.Contains(m.Text, "same wall twice") {
			if !strings.Contains(m.Text, "your own mistake") {
				t.Errorf("the notice does not say when to skip it: %q", m.Text)
			}
			return
		}
	}
	t.Fatal("no notice was emitted")
}
