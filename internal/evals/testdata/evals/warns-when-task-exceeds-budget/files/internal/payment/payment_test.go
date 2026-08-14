package payment

import "testing"

func TestAuthoriseProducesAReference(t *testing.T) {
	ref, err := Authorise(Charge{})
	if err != nil {
		t.Fatalf("authorise: %v", err)
	}
	if ref == "" {
		t.Error("no reference came back")
	}
}

func TestCaptureProducesAReference(t *testing.T) {
	if _, err := Capture(Charge{}); err != nil {
		t.Fatalf("capture: %v", err)
	}
}

func TestRefundProducesAReference(t *testing.T) {
	if _, err := Refund(Charge{}); err != nil {
		t.Fatalf("refund: %v", err)
	}
}

func TestSettleProducesAReference(t *testing.T) {
	if _, err := Settle(Charge{}); err != nil {
		t.Fatalf("settle: %v", err)
	}
}

func TestHandleRunsEveryStep(t *testing.T) {
	out, err := Handle(Charge{})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(out) != 4 {
		t.Errorf("ran %d step(s), want 4", len(out))
	}
}
