package stats

// Summary is what one run produced.
type Summary struct {
	count        int
	accountCount int
}

// Add records one row.
func (s *Summary) Add() {
	s.count++
}

// Rows reports the number of rows added.
func (s *Summary) Rows() int {
	return s.count
}

// Accounts reports how many distinct accounts were seen.
func (s *Summary) Accounts() int {
	return s.accountCount
}
