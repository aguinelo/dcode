package shipping

import "testing"

func TestQuoteProducesAReference(t *testing.T) {
	ref, err := Quote(Rate{})
	if err != nil {
		t.Fatalf("quote: %v", err)
	}
	if ref == "" {
		t.Error("no reference came back")
	}
}

func TestCompareProducesAReference(t *testing.T) {
	if _, err := Compare(Rate{}); err != nil {
		t.Fatalf("compare: %v", err)
	}
}

func TestSelectProducesAReference(t *testing.T) {
	if _, err := Select(Rate{}); err != nil {
		t.Fatalf("select: %v", err)
	}
}

func TestRefundProducesAReference(t *testing.T) {
	if _, err := Refund(Rate{}); err != nil {
		t.Fatalf("refund: %v", err)
	}
}

func TestHandleRunsEveryStep(t *testing.T) {
	out, err := Handle(Rate{})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(out) != 4 {
		t.Errorf("ran %d step(s), want 4", len(out))
	}
}
