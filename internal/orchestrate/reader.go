package orchestrate

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// LogLine is one complete line of a decision log as a reader finds it.
//
// A reader never stops at a line that does not read: the log is read when
// something already went wrong, and the lines after a bad one are the ones
// that say what. Err says why this one did not; the line is still counted.
type LogLine struct {
	// N is the line's number, from 1.
	N int
	// Raw is the line as written, without its newline.
	Raw []byte
	// Decision is the line read; valid when Err is nil.
	Decision Decision
	// Err is why the line does not read as a decision of this format.
	Err error
	// Gap is set when Decision.Seq does not follow Prev, the seq of the last
	// line that read: a gap, a repetition, or a seq out of order.
	Gap  bool
	Prev uint64
}

// LogReader reads a decisions.jsonl line by line, read-only.
//
// Only complete lines come out. What follows the last newline is a line still
// being written, or one a crash cut short; it waits in Pending, and a reader
// that follows the log gets it once its newline arrives.
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §9.2.
type LogReader struct {
	r       *bufio.Reader
	n       int
	seq     uint64
	pending []byte
}

// NewLogReader reads the log from r.
func NewLogReader(r io.Reader) *LogReader {
	return &LogReader{r: bufio.NewReader(r)}
}

// Next returns the next complete line, or io.EOF when there is none yet. On a
// file that grows, a later Next continues where this one stopped.
func (l *LogReader) Next() (LogLine, error) {
	chunk, err := l.r.ReadBytes('\n')
	l.pending = append(l.pending, chunk...)
	if errors.Is(err, io.EOF) {
		return LogLine{}, io.EOF
	}
	if err != nil {
		return LogLine{}, err
	}
	raw := bytes.TrimSuffix(l.pending, []byte{'\n'})
	l.pending = nil
	l.n++
	line := LogLine{N: l.n, Raw: raw}

	if err := json.Unmarshal(raw, &line.Decision); err != nil {
		line.Err = fmt.Errorf("unreadable decision: %w", err)
		return line, nil
	}
	if v := line.Decision.V; v != logVersion {
		line.Err = fmt.Errorf("format v%d, this reader knows v%d", v, logVersion)
		return line, nil
	}
	line.Prev = l.seq
	line.Gap = line.Decision.Seq != l.seq+1
	l.seq = line.Decision.Seq
	return line, nil
}

// Pending is the partial last line, nil when the log ends in a newline.
func (l *LogReader) Pending() []byte {
	if len(l.pending) == 0 {
		return nil
	}
	return bytes.Clone(l.pending)
}
