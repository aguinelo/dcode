package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// Writing does not add a mode. It adds ownership.
//
// `owns` absent is the read-only child that already exists, so nothing changes
// for anyone already delegating. `owns` present asks for a writing child, and
// the ask is a ceiling that can only narrow — the intersection with what the
// parent may write happens in the loop, never here.
func TestExploreWithoutOwnsDeclaresOnlyARead(t *testing.T) {
	req, err := Explore{}.Declare(mustJSON(t, ExploreInput{Task: "map it", Path: "pay"}))
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range req.Paths {
		if a.Write {
			t.Errorf("a child with no owns declares no write: %+v", req.Paths)
		}
	}
}

func TestExploreWithOwnsDeclaresAWriteForEachOwnedPath(t *testing.T) {
	req, err := Explore{}.Declare(mustJSON(t, ExploreInput{
		Task: "catalogue it",
		Path: "repos/billing",
		Owns: []string{"repos/billing/ARCHITECTURE.md"},
	}))
	if err != nil {
		t.Fatal(err)
	}

	var wrote bool
	for _, a := range req.Paths {
		if a.Path == "repos/billing/ARCHITECTURE.md" && a.Write {
			wrote = true
		}
	}
	if !wrote {
		t.Errorf("an owned path must be declared as a write, so the scheduler can serialise it: %+v", req.Paths)
	}
}

// The declaration is what the scheduler serialises on. A child whose ownership
// never reached the declaration would run beside another child writing the same
// file, and the collision would be discovered by the filesystem.
func TestTwoChildrenOwningTheSamePathDeclareAConflict(t *testing.T) {
	one, _ := Explore{}.Declare(mustJSON(t, ExploreInput{Task: "a", Owns: []string{"docs/x.md"}}))
	two, _ := Explore{}.Declare(mustJSON(t, ExploreInput{Task: "b", Owns: []string{"docs/x.md"}}))

	var shared bool
	for _, a := range one.Paths {
		for _, b := range two.Paths {
			if a.Path == b.Path && (a.Write || b.Write) {
				shared = true
			}
		}
	}
	if !shared {
		t.Error("two children owning one path must be visible as a conflict before either runs")
	}
}

// An empty set is not "everything". It is a declaration error, the same
// standing a write with no declared path already has.
func TestExploreWithAnEmptyOwnsIsADeclarationError(t *testing.T) {
	s, _ := setup(t)
	// Raw JSON on purpose: this is what a model sends, and it is the only way
	// to tell an empty set from an absent one. A round trip through the Go
	// struct loses the difference, and the difference is the whole test.
	res, err := Explore{Delegator: &fakeDelegator{}}.Execute(
		context.Background(), json.RawMessage(`{"task":"catalogue it","owns":[]}`), s)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Error("owning nothing is not permission for everything")
	}
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// A child that did not answer is named, never summarised away.
//
// With several children in flight, N−1 answers is the ordinary case, not the
// exceptional one. Returning N−1 quietly is an incomplete result wearing the
// face of a complete one — the shape of defect this repository keeps finding in
// itself.
func TestAChildThatDidNotAnswerIsNamed(t *testing.T) {
	s, _ := setup(t)
	res := run(t, Explore{Delegator: &fakeDelegator{err: errors.New("stream closed")}}, s,
		ExploreInput{Task: "catalogue the billing repo", Path: "repos/billing"})

	if !res.IsError {
		t.Fatal("a failed child must be an error, not an empty answer")
	}
	if !strings.Contains(res.Output, "catalogue the billing repo") {
		t.Errorf("the failure does not say which child it was:\n%s", res.Output)
	}
}

// What the child wrote reaches the parent's history, beside where it looked.
func TestExploreReportsWhatTheChildWrote(t *testing.T) {
	s, _ := setup(t)
	res := run(t, Explore{Delegator: &fakeDelegator{
		conclusion: "catalogued",
		wrote:      []string{"repos/billing/ARCHITECTURE.md"},
	}}, s, ExploreInput{Task: "catalogue it", Owns: []string{"repos/billing/ARCHITECTURE.md"}})

	if !strings.Contains(res.Output, "wrote: repos/billing/ARCHITECTURE.md") {
		t.Errorf("the parent cannot see what the child changed:\n%s", res.Output)
	}
}
