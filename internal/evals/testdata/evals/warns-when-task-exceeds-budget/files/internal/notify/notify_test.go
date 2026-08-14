package notify

import "testing"

func TestQueueProducesAReference(t *testing.T) {
	ref, err := Queue(Message{})
	if err != nil {
		t.Fatalf("queue: %v", err)
	}
	if ref == "" {
		t.Error("no reference came back")
	}
}

func TestSendProducesAReference(t *testing.T) {
	if _, err := Send(Message{}); err != nil {
		t.Fatalf("send: %v", err)
	}
}

func TestRetryProducesAReference(t *testing.T) {
	if _, err := Retry(Message{}); err != nil {
		t.Fatalf("retry: %v", err)
	}
}

func TestSuppressProducesAReference(t *testing.T) {
	if _, err := Suppress(Message{}); err != nil {
		t.Fatalf("suppress: %v", err)
	}
}

func TestHandleRunsEveryStep(t *testing.T) {
	out, err := Handle(Message{})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(out) != 4 {
		t.Errorf("ran %d step(s), want 4", len(out))
	}
}
