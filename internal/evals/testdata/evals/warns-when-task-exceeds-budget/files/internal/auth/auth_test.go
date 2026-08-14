package auth

import "testing"

func TestIssueProducesAReference(t *testing.T) {
	ref, err := Issue(Session{})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if ref == "" {
		t.Error("no reference came back")
	}
}

func TestVerifyProducesAReference(t *testing.T) {
	if _, err := Verify(Session{}); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestRevokeProducesAReference(t *testing.T) {
	if _, err := Revoke(Session{}); err != nil {
		t.Fatalf("revoke: %v", err)
	}
}

func TestRefreshProducesAReference(t *testing.T) {
	if _, err := Refresh(Session{}); err != nil {
		t.Fatalf("refresh: %v", err)
	}
}

func TestHandleRunsEveryStep(t *testing.T) {
	out, err := Handle(Session{})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(out) != 4 {
		t.Errorf("ran %d step(s), want 4", len(out))
	}
}
