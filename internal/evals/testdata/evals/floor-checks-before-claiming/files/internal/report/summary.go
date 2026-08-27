package report

// Summary is one report, as it will be printed.
type Summary struct {
	Rows  int
	Lines []string
}

// Add records one line.
func (s *Summary) Add(line string) {
	s.Lines = append(s.Lines, line)
	s.Rows++
}
