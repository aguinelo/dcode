package contextengine

import "testing"

// An image costs context, and a budget that does not count it drifts silently:
// the band says 60% while the window is nearly full, and compaction arrives
// after the model has already lost the thread.
func TestAnImageCostsContext(t *testing.T) {
	cfg := Config{Window: 100_000}
	text := []Message{{Role: RoleUser, Text: "look at this"}}
	withImage := []Message{{Role: RoleUser, Text: "look at this",
		Images: []Image{{MediaType: "image/png", Data: make([]byte, 200<<10)}}}}

	plain, shot := Estimate(text, cfg), Estimate(withImage, cfg)
	if shot <= plain {
		t.Fatalf("an image added %d tokens", shot-plain)
	}
	// And it is a fixed cost per image rather than a share of its bytes: a
	// model prices an image by what it sees, not by how well it compressed.
	small := []Message{{Role: RoleUser, Text: "look at this",
		Images: []Image{{MediaType: "image/png", Data: make([]byte, 4<<10)}}}}
	if Estimate(small, cfg) != shot {
		t.Error("the cost tracked the file size rather than the image")
	}
}

// Estimate stays pure: the same messages give the same number, always.
func TestEstimatingAnImageIsStable(t *testing.T) {
	cfg := Config{Window: 100_000}
	msgs := []Message{{Role: RoleUser, Images: []Image{{MediaType: "image/png", Data: []byte("x")}}}}
	if Estimate(msgs, cfg) != Estimate(msgs, cfg) {
		t.Error("two calls disagreed")
	}
}
