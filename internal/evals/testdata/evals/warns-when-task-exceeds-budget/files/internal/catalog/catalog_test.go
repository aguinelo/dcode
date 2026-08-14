package catalog

import "testing"

func TestAddProducesAReference(t *testing.T) {
	ref, err := Add(Item{})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if ref == "" {
		t.Error("no reference came back")
	}
}

func TestRepriceProducesAReference(t *testing.T) {
	if _, err := Reprice(Item{}); err != nil {
		t.Fatalf("reprice: %v", err)
	}
}

func TestRetireProducesAReference(t *testing.T) {
	if _, err := Retire(Item{}); err != nil {
		t.Fatalf("retire: %v", err)
	}
}

func TestSearchProducesAReference(t *testing.T) {
	if _, err := Search(Item{}); err != nil {
		t.Fatalf("search: %v", err)
	}
}

func TestHandleRunsEveryStep(t *testing.T) {
	out, err := Handle(Item{})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(out) != 4 {
		t.Errorf("ran %d step(s), want 4", len(out))
	}
}
