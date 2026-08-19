package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// fakeDelegator stands in for the loop, which owns turns.
type fakeDelegator struct {
	owns       []string
	conclusion string
	read       []string
	unread     []string
	truncated  bool
	err        error
	task, path string
}

func (f *fakeDelegator) Explore(_ context.Context, task, path string, owns []string) (string, []string, []string, bool, error) {
	f.owns = owns
	f.task, f.path = task, path
	return f.conclusion, f.read, f.unread, f.truncated, f.err
}

func TestExploreReportsWhereItLookedAndWhatItCouldNotRead(t *testing.T) {
	s, _ := setup(t)
	d := &fakeDelegator{
		conclusion: "Validation is in pay/validate.go:42.",
		read:       []string{"pay/validate.go", "pay/types.go"},
		unread:     []string{"pay/.env"},
	}
	res := run(t, Explore{Delegator: d}, s, ExploreInput{Task: "where is payment validated", Path: "pay"})

	if res.IsError {
		t.Fatal(res.Output)
	}
	for _, want := range []string{"validate.go:42", "looked at:", "pay/types.go", "could not read:", "pay/.env"} {
		t.Run(want, func(t *testing.T) {
			if !strings.Contains(res.Output, want) {
				t.Errorf("missing %q from:\n%s", want, res.Output)
			}
		})
	}
	if d.task != "where is payment validated" || d.path != "pay" {
		t.Errorf("the delegator got task=%q path=%q", d.task, d.path)
	}
	if res.Meta.Files != 2 {
		t.Errorf("files = %d, want the count it read", res.Meta.Files)
	}
}

func TestExploreDeclaresTruncation(t *testing.T) {
	s, _ := setup(t)
	res := run(t, Explore{Delegator: &fakeDelegator{conclusion: "…", truncated: true}}, s,
		ExploreInput{Task: "explain"})
	if !res.Truncated || !strings.Contains(res.Output, "truncated") {
		t.Fatalf("a cut report did not declare it:\n%s", res.Output)
	}
}

func TestAnEmptyTaskIsRefused(t *testing.T) {
	s, _ := setup(t)
	if res := run(t, Explore{Delegator: &fakeDelegator{}}, s, ExploreInput{Task: "  "}); !res.IsError {
		t.Fatal("a child was launched with nothing to find out")
	}
}

func TestExploreWithoutADelegatorSaysSoRatherThanPanicking(t *testing.T) {
	s, _ := setup(t)
	res := run(t, Explore{}, s, ExploreInput{Task: "find it"})
	if !res.IsError {
		t.Fatal("delegation with nothing behind it reported success")
	}
	if !strings.Contains(strings.ToLower(res.Output), "off") {
		t.Errorf("the error does not say delegation is unavailable: %s", res.Output)
	}
}

func TestADelegatedFailureBecomesAToolErrorAndNotAPanic(t *testing.T) {
	s, _ := setup(t)
	res := run(t, Explore{Delegator: &fakeDelegator{err: errors.New("provider refused")}}, s,
		ExploreInput{Task: "find it"})
	if !res.IsError {
		t.Fatal("a failed child reported success")
	}
	if !strings.Contains(res.Output, "provider refused") {
		t.Errorf("the cause was lost: %s", res.Output)
	}
}

// It declares a read and nothing else, whatever it is asked — the child's mode
// is what actually constrains it, and this is the half the policy layer sees.
func TestExploreDeclaresOnlyARead(t *testing.T) {
	in, _ := json.Marshal(ExploreInput{Task: "t", Path: "sub"})
	var probe Explore
	req, err := probe.Declare(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(req.Paths) != 1 || req.Paths[0].Write {
		t.Fatalf("declared %+v, want a single read", req.Paths)
	}
	if req.Network {
		t.Error("declared network")
	}
	// No path given falls back to the workspace root rather than to nothing.
	in2, _ := json.Marshal(ExploreInput{Task: "t"})
	req2, _ := probe.Declare(in2)
	if req2.Paths[0].Path != "." {
		t.Errorf("path = %q, want the workspace root", req2.Paths[0].Path)
	}
}

func TestExploreDescribesWhenNotToUseIt(t *testing.T) {
	var probe Explore
	d := probe.Description()
	// The cheap failure mode is delegating everything, so the description has
	// to say when not to.
	for _, want := range []string{"already read", "single known file", "cannot delegate"} {
		if !strings.Contains(d, want) {
			t.Errorf("the description does not warn about %q:\n%s", want, d)
		}
	}
	var e Explore
	if e.Name() != "explore" {
		t.Error("wrong name")
	}
	if len(e.Schema()) == 0 {
		t.Error("no schema")
	}
}
