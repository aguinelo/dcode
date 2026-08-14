package delivery

import "testing"

func TestBookProducesAReference(t *testing.T) {
	ref, err := Book(Shipment{})
	if err != nil {
		t.Fatalf("book: %v", err)
	}
	if ref == "" {
		t.Error("no reference came back")
	}
}

func TestTrackProducesAReference(t *testing.T) {
	if _, err := Track(Shipment{}); err != nil {
		t.Fatalf("track: %v", err)
	}
}

func TestRerouteProducesAReference(t *testing.T) {
	if _, err := Reroute(Shipment{}); err != nil {
		t.Fatalf("reroute: %v", err)
	}
}

func TestCompleteProducesAReference(t *testing.T) {
	if _, err := Complete(Shipment{}); err != nil {
		t.Fatalf("complete: %v", err)
	}
}

func TestHandleRunsEveryStep(t *testing.T) {
	out, err := Handle(Shipment{})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(out) != 4 {
		t.Errorf("ran %d step(s), want 4", len(out))
	}
}
