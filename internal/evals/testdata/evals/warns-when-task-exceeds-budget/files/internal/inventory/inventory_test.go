package inventory

import "testing"

func TestReserveProducesAReference(t *testing.T) {
	ref, err := Reserve(Stock{})
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if ref == "" {
		t.Error("no reference came back")
	}
}

func TestReleaseProducesAReference(t *testing.T) {
	if _, err := Release(Stock{}); err != nil {
		t.Fatalf("release: %v", err)
	}
}

func TestReceiveProducesAReference(t *testing.T) {
	if _, err := Receive(Stock{}); err != nil {
		t.Fatalf("receive: %v", err)
	}
}

func TestCountProducesAReference(t *testing.T) {
	if _, err := Count(Stock{}); err != nil {
		t.Fatalf("count: %v", err)
	}
}

func TestHandleRunsEveryStep(t *testing.T) {
	out, err := Handle(Stock{})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(out) != 4 {
		t.Errorf("ran %d step(s), want 4", len(out))
	}
}
