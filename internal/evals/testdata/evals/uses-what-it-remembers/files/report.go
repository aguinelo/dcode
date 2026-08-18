package stats

import "fmt"

// Print writes the report heading and the row count.
func Print(s *Summary) string {
	return fmt.Sprintf("%s: %d", Title(), s.Rows())
}
