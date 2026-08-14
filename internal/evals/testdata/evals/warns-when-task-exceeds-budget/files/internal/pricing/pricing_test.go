package pricing

import "testing"

func TestApplyProducesAReference(t *testing.T) {
	ref, err := Apply(Rule{})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if ref == "" {
		t.Error("no reference came back")
	}
}

func TestStackProducesAReference(t *testing.T) {
	if _, err := Stack(Rule{}); err != nil {
		t.Fatalf("stack: %v", err)
	}
}

func TestExpireProducesAReference(t *testing.T) {
	if _, err := Expire(Rule{}); err != nil {
		t.Fatalf("expire: %v", err)
	}
}

func TestQuoteProducesAReference(t *testing.T) {
	if _, err := Quote(Rule{}); err != nil {
		t.Fatalf("quote: %v", err)
	}
}

func TestHandleRunsEveryStep(t *testing.T) {
	out, err := Handle(Rule{})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(out) != 4 {
		t.Errorf("ran %d step(s), want 4", len(out))
	}
}
