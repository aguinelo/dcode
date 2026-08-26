package behavior

import (
	"strings"
	"testing"
)

func doctrineWith(practices string) Doctrine {
	d := DefaultDoctrine([]string{"read", "edit", "bash"})
	d.Practices = practices
	return d
}

func promptFrom(t *testing.T, d Doctrine) string {
	t.Helper()
	out, err := Build(Prompt{Doctrine: d}, FormulationFor(""))
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// An empty floor is a floor switched off, and switching it off is a legitimate
// choice. Identity and Safety are not: an agent with no identity is not a
// degraded agent, it is an unpredictable one, and Build refuses both.
func TestAnEmptyFloorDoesNotFailTheBuild(t *testing.T) {
	if _, err := Build(Prompt{Doctrine: doctrineWith("")}, FormulationFor("")); err != nil {
		t.Fatalf("an empty floor refused to assemble: %v", err)
	}

	d := doctrineWith("something")
	d.Identity = ""
	if _, err := Build(Prompt{Doctrine: d}, FormulationFor("")); err == nil {
		t.Error("an empty identity assembled; it must not")
	}
	d = doctrineWith("something")
	d.Safety = ""
	if _, err := Build(Prompt{Doctrine: d}, FormulationFor("")); err == nil {
		t.Error("an empty safety section assembled; it must not")
	}
}

// The position IS the precedence, and there is no resolver.
//
// The floor sits after Safety and before everything anyone actually said. What
// comes earlier in the prefix is context for reading what comes later, and the
// project instructions are the last block of all — so a default rendered here
// is outranked by any instruction, which is what a default should be.
func TestTheFloorSitsAfterSafetyAndBeforeWhatAnyoneSaid(t *testing.T) {
	out := promptFrom(t, doctrineWith("FLOOR-MARK"))

	safety := strings.Index(out, "Safety")
	floor := strings.Index(out, "FLOOR-MARK")
	tools := strings.Index(out, "Using tools")
	if safety < 0 || floor < 0 || tools < 0 {
		t.Fatalf("a section is missing:\n%s", out)
	}
	if !(safety < floor && floor < tools) {
		t.Errorf("the floor is not between Safety and Using tools (%d, %d, %d)", safety, floor, tools)
	}
}

// The project's instructions stay the last block, which is the position of
// greatest weight and the reason the floor needs no precedence machinery.
func TestProjectInstructionsStillComeAfterTheFloor(t *testing.T) {
	out, err := Build(Prompt{
		Doctrine:     doctrineWith("FLOOR-MARK"),
		Instructions: []Instruction{{Source: SourceProject, Text: "PROJECT-MARK"}},
	}, FormulationFor(""))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Index(out, "FLOOR-MARK") > strings.Index(out, "PROJECT-MARK") {
		t.Errorf("the floor outranks the project instructions by position:\n%s", out)
	}
}

// practices.md replaces the shipped text. There is no appending variant, and
// the absence is the rule rather than an omission: appending to a floor
// produces two floors, and the second is never read alongside the first.
func TestPracticesMdReplacesAndDoesNotAppend(t *testing.T) {
	d := doctrineWith("BUILTIN-FLOOR").Apply(DoctrineOverlay{Practices: "MY-FLOOR"})
	if d.Practices != "MY-FLOOR" {
		t.Fatalf("practices was not replaced: %q", d.Practices)
	}
	if strings.Contains(d.Practices, "BUILTIN-FLOOR") {
		t.Errorf("the shipped floor survived a replacement: %q", d.Practices)
	}
}

// The overlay reaches Practices and still cannot reach Safety. The second half
// is the one worth a test: it is the guarantee, and a guarantee nobody asserts
// is a comment.
func TestTheOverlayReachesPracticesAndNeverSafety(t *testing.T) {
	before := doctrineWith("floor")
	after := before.Apply(DoctrineOverlay{Practices: "mine", Identity: "me", Style: "terse"})

	if after.Practices != "mine" {
		t.Errorf("practices not applied: %q", after.Practices)
	}
	if after.Safety != before.Safety {
		t.Error("safety changed through an overlay; it has no field and must not be reachable")
	}
	got := (DoctrineOverlay{Practices: "mine"}).Origins()
	if got.Safety != OriginBuiltin {
		t.Errorf("safety origin is %q with a practices overlay in force", got.Safety)
	}
}

// A replaced floor says so. An invisible replacement would be worse than the
// immutability it replaces, because the only way a user has of knowing what
// reached the model is to read it.
func TestAReplacedFloorIsReportedAsReplaced(t *testing.T) {
	if got := (DoctrineOverlay{}).Origins().Practices; got != OriginBuiltin {
		t.Errorf("with no overlay the floor is %q, want builtin", got)
	}
	if got := (DoctrineOverlay{Practices: "mine"}).Origins().Practices; got != OriginReplaced {
		t.Errorf("with an overlay the floor is %q, want replaced", got)
	}
}

// Build is pure and has to stay pure with the new section: the same doctrine
// must produce a byte-identical prefix, or every cached prefix goes with it.
func TestTheFloorKeepsBuildPure(t *testing.T) {
	d := doctrineWith("a floor with several\nlines in it")
	first := promptFrom(t, d)
	for i := 0; i < 5; i++ {
		if got := promptFrom(t, d); got != first {
			t.Fatal("the same doctrine produced a different prefix")
		}
	}
}
