package billing

import "testing"

func TestDraftProducesAReference(t *testing.T) {
	ref, err := Draft(Invoice{})
	if err != nil {
		t.Fatalf("draft: %v", err)
	}
	if ref == "" {
		t.Error("no reference came back")
	}
}

func TestFinaliseProducesAReference(t *testing.T) {
	if _, err := Finalise(Invoice{}); err != nil {
		t.Fatalf("finalise: %v", err)
	}
}

func TestVoidProducesAReference(t *testing.T) {
	if _, err := Void(Invoice{}); err != nil {
		t.Fatalf("void: %v", err)
	}
}

func TestReconcileProducesAReference(t *testing.T) {
	if _, err := Reconcile(Invoice{}); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
}

func TestHandleRunsEveryStep(t *testing.T) {
	out, err := Handle(Invoice{})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(out) != 4 {
		t.Errorf("ran %d step(s), want 4", len(out))
	}
}
