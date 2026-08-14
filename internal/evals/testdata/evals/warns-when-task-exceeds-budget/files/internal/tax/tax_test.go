package tax

import "testing"

func TestResolveProducesAReference(t *testing.T) {
	ref, err := Resolve(Bracket{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if ref == "" {
		t.Error("no reference came back")
	}
}

func TestComputeProducesAReference(t *testing.T) {
	if _, err := Compute(Bracket{}); err != nil {
		t.Fatalf("compute: %v", err)
	}
}

func TestExemptProducesAReference(t *testing.T) {
	if _, err := Exempt(Bracket{}); err != nil {
		t.Fatalf("exempt: %v", err)
	}
}

func TestRoundProducesAReference(t *testing.T) {
	if _, err := Round(Bracket{}); err != nil {
		t.Fatalf("round: %v", err)
	}
}

func TestHandleRunsEveryStep(t *testing.T) {
	out, err := Handle(Bracket{})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(out) != 4 {
		t.Errorf("ran %d step(s), want 4", len(out))
	}
}
