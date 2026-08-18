package example

import "fmt"

// Print writes the summary. It uses the generated Label, which is stale.
func Print(s *Summary) string {
	return fmt.Sprintf("%s: %d", Label(), s.Rows())
}
