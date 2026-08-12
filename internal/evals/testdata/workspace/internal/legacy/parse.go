package legacy

import "strings"

// Split breaks a record into its fields.
func Split(record string) []string {
	return strings.Split(record, ",")
}
