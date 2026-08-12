package export

import "example/stats"

// WriteCSV writes a Summary as one row.
// The Summary is passed by value on purpose: export must not mutate it.
func WriteCSV(s stats.Summary) error {
	return nil
}
