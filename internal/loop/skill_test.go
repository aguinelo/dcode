package loop

import (
	"context"
	"testing"

	"github.com/aguinelo/dcode/internal/behavior"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
	"github.com/aguinelo/dcode/internal/tools"
)

// A skill body used to join the turn with nothing emitted: spent context and
// changed behaviour, and no trace anywhere the person looks. The index is in
// the prefix and `--dump-prompt` prints it; what fired was the one observable
// fact this protocol was not carrying.
func TestALoadedSkillIsAnnounced(t *testing.T) {
	skills := []behavior.Skill{
		{Name: "release", WhenToUse: "cutting a new version of the module", Body: "record it in RELEASING.md"},
		{Name: "migrations", WhenToUse: "editing a database migration script", Body: "never edit an applied one"},
	}
	e, rec := newEngine(t, &scriptedProvider{turns: [][]provider.StreamEvent{{text("ok"), done()}}}, tools.NewRegistry(),
		func(c *Config) { c.Skills = skills })

	if _, err := e.Run(context.Background(), "cutting a new version of the module"); err != nil {
		t.Fatal(err)
	}

	var got []protocol.SkillLoaded
	for _, p := range rec.all {
		if d, ok := p.(protocol.SkillLoaded); ok {
			got = append(got, d)
		}
	}
	if len(got) != 1 {
		t.Fatalf("got %d skill.loaded events, want exactly the one that fired: %+v", len(got), got)
	}
	if got[0].Name != "release" {
		t.Errorf("announced %q, want release", got[0].Name)
	}
	if got[0].WhenToUse != skills[0].WhenToUse {
		t.Errorf("the event carries %q; it has to be the same line the model read in the index", got[0].WhenToUse)
	}
}

// A turn that loads nothing says nothing. An event per turn regardless would
// make the stream carry a line whose only content is that a feature exists.
func TestATurnThatLoadsNoSkillAnnouncesNothing(t *testing.T) {
	skills := []behavior.Skill{
		{Name: "release", WhenToUse: "cutting a new version of the module", Body: "b"},
	}
	e, rec := newEngine(t, &scriptedProvider{turns: [][]provider.StreamEvent{{text("ok"), done()}}}, tools.NewRegistry(),
		func(c *Config) { c.Skills = skills })

	if _, err := e.Run(context.Background(), "rename a local variable"); err != nil {
		t.Fatal(err)
	}
	if n := rec.count(protocol.EventSkillLoaded); n != 0 {
		t.Errorf("got %d skill.loaded events for a task that matched nothing", n)
	}
}
