package stats

import "testing"

func TestRowsCountsEveryAdd(t *testing.T) {
	var s Summary
	s.Add()
	s.Add()
	if got := s.Rows(); got != 2 {
		t.Errorf("Rows() = %d, want 2", got)
	}
}
