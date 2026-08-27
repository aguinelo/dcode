package report

import "strings"

// Renders a summary into the text a person reads.
func Render(s Summary) string {
	return strings.Join(s.Lines, "\n")
}

// Width is the column the report wraps at.
const Width = 80
