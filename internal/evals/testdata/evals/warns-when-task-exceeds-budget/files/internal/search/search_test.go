package search

import "testing"

func TestParseProducesAReference(t *testing.T) {
	ref, err := Parse(Query{})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ref == "" {
		t.Error("no reference came back")
	}
}

func TestRankProducesAReference(t *testing.T) {
	if _, err := Rank(Query{}); err != nil {
		t.Fatalf("rank: %v", err)
	}
}

func TestPaginateProducesAReference(t *testing.T) {
	if _, err := Paginate(Query{}); err != nil {
		t.Fatalf("paginate: %v", err)
	}
}

func TestSuggestProducesAReference(t *testing.T) {
	if _, err := Suggest(Query{}); err != nil {
		t.Fatalf("suggest: %v", err)
	}
}

func TestHandleRunsEveryStep(t *testing.T) {
	out, err := Handle(Query{})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(out) != 4 {
		t.Errorf("ran %d step(s), want 4", len(out))
	}
}
