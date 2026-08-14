package user

import "testing"

func TestCreateProducesAReference(t *testing.T) {
	ref, err := Create(Profile{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if ref == "" {
		t.Error("no reference came back")
	}
}

func TestVerifyProducesAReference(t *testing.T) {
	if _, err := Verify(Profile{}); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestRenameProducesAReference(t *testing.T) {
	if _, err := Rename(Profile{}); err != nil {
		t.Fatalf("rename: %v", err)
	}
}

func TestDeactivateProducesAReference(t *testing.T) {
	if _, err := Deactivate(Profile{}); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
}

func TestHandleRunsEveryStep(t *testing.T) {
	out, err := Handle(Profile{})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(out) != 4 {
		t.Errorf("ran %d step(s), want 4", len(out))
	}
}
