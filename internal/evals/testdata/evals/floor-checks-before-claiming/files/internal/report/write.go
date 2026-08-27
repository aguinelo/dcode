package report

import "os"

// The file every report is written to when none is named.
const DefaultPath = "report.txt"

// Write puts the rendered report on disk.
func Write(path, text string) error {
	if path == "" {
		path = DefaultPath
	}
	return os.WriteFile(path, []byte(text), 0o644)
}
