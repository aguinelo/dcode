package stats

import "fmt"

// Print writes the summary.
func Print(s *Summary) string {
	return fmt.Sprintf("rows: %d", s.Rows())
}
