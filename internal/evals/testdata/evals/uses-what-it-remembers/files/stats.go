package stats

// Summary counts rows.
type Summary struct {
	count int
}

// Add records one row.
func (s *Summary) Add() { s.count++ }

// Rows reports how many rows were recorded.
func (s *Summary) Rows() int { return s.count }
