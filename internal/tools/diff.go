package tools

import (
	"fmt"
	"strings"
)

// Unified diffs, for the human rather than for the model.
//
// The model already knows what it changed — it wrote the edit — so the diff
// never enters the history. It rides on the event, where a client can render it
// and it costs nothing in tokens.

// DiffContext is how many unchanged lines surround each change. Three is the
// convention every reviewer already reads.
const DiffContext = 3

// DiffMaxLines caps the diff carried on an event.
//
// A whole-file rewrite would otherwise produce a payload larger than the file,
// and the client can only show a screenful anyway. The cap is generous enough
// that an ordinary edit is never touched.
const DiffMaxLines = 400

// UnifiedDiff renders the change between two texts.
//
// Empty when nothing changed: a diff with no hunks is noise, and the caller
// uses the empty string to mean "nothing to show".
func UnifiedDiff(before, after, path string) string {
	if before == after {
		return ""
	}
	a, b := splitLines(before), splitLines(after)
	hunks := buildHunks(a, b, DiffContext)
	if len(hunks) == 0 {
		return ""
	}

	var out strings.Builder
	written, truncated := 0, false
	for _, h := range hunks {
		if written >= DiffMaxLines {
			truncated = true
			break
		}
		header := fmt.Sprintf("@@ -%d,%d +%d,%d @@", h.aStart, h.aCount, h.bStart, h.bCount)
		if path != "" {
			header += " " + path
		}
		out.WriteString(header)
		out.WriteString("\n")
		written++

		for _, l := range h.lines {
			if written >= DiffMaxLines {
				// A single hunk can exhaust the budget on its own, so the mark
				// has to be reachable from here too — truncating in silence
				// reads as a complete diff.
				truncated = true
				break
			}
			out.WriteString(l)
			out.WriteString("\n")
			written++
		}
	}
	if truncated {
		fmt.Fprintf(&out, "⋯ diff truncated at %d lines\n", DiffMaxLines)
	}
	return strings.TrimRight(out.String(), "\n")
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	// A trailing newline terminates the last line rather than starting an empty
	// one, which is what every diff tool does and what makes the counts match.
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

type hunk struct {
	aStart, aCount int
	bStart, bCount int
	lines          []string
}

// op is one step of the edit script.
type op struct {
	kind byte // ' ' keep, '-' remove, '+' add
	text string
}

// buildHunks turns an edit script into hunks with context around each change.
func buildHunks(a, b []string, context int) []hunk {
	script := diffScript(a, b)

	// Mark which script positions are within context of a change, so runs of
	// untouched text collapse instead of printing the whole file.
	keep := make([]bool, len(script))
	for i, o := range script {
		if o.kind == ' ' {
			continue
		}
		lo, hi := i-context, i+context
		if lo < 0 {
			lo = 0
		}
		if hi >= len(script) {
			hi = len(script) - 1
		}
		for j := lo; j <= hi; j++ {
			keep[j] = true
		}
	}

	var (
		hunks   []hunk
		current *hunk
		aLine   = 1
		bLine   = 1
	)
	for i, o := range script {
		if keep[i] {
			if current == nil {
				hunks = append(hunks, hunk{aStart: aLine, bStart: bLine})
				current = &hunks[len(hunks)-1]
			}
			current.lines = append(current.lines, string(o.kind)+o.text)
			switch o.kind {
			case ' ':
				current.aCount++
				current.bCount++
			case '-':
				current.aCount++
			case '+':
				current.bCount++
			}
		} else {
			current = nil
		}
		switch o.kind {
		case ' ':
			aLine++
			bLine++
		case '-':
			aLine++
		case '+':
			bLine++
		}
	}
	return hunks
}

// diffScript produces the edit script from a to b.
//
// Longest common subsequence over lines. Quadratic in the worst case, which is
// why the input is bounded before it gets here: a tool result is already capped,
// and an unbounded diff of two large files would stall the turn.
func diffScript(a, b []string) []op {
	// Trim the common head and tail first. Real edits change a few lines in the
	// middle of a file, so this reduces the quadratic part to almost nothing.
	head := 0
	for head < len(a) && head < len(b) && a[head] == b[head] {
		head++
	}
	tail := 0
	for tail < len(a)-head && tail < len(b)-head &&
		a[len(a)-1-tail] == b[len(b)-1-tail] {
		tail++
	}

	midA, midB := a[head:len(a)-tail], b[head:len(b)-tail]

	out := make([]op, 0, len(a)+len(b))
	for _, l := range a[:head] {
		out = append(out, op{' ', l})
	}
	out = append(out, lcsScript(midA, midB)...)
	for _, l := range a[len(a)-tail:] {
		out = append(out, op{' ', l})
	}
	return out
}

// lcsMaxCells bounds the table. Past it the diff degrades to a plain
// replacement, which is still correct and still readable — a diff of two files
// with nothing in common was never going to be useful anyway.
const lcsMaxCells = 4_000_000

func lcsScript(a, b []string) []op {
	if len(a) == 0 || len(b) == 0 || len(a)*len(b) > lcsMaxCells {
		out := make([]op, 0, len(a)+len(b))
		for _, l := range a {
			out = append(out, op{'-', l})
		}
		for _, l := range b {
			out = append(out, op{'+', l})
		}
		return out
	}

	// table[i][j] is the LCS length of a[i:] and b[j:].
	table := make([][]int, len(a)+1)
	for i := range table {
		table[i] = make([]int, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				table[i][j] = table[i+1][j+1] + 1
			} else if table[i+1][j] >= table[i][j+1] {
				table[i][j] = table[i+1][j]
			} else {
				table[i][j] = table[i][j+1]
			}
		}
	}

	var out []op
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			out = append(out, op{' ', a[i]})
			i++
			j++
		case table[i+1][j] >= table[i][j+1]:
			// Removals before additions at the same position: it is the order
			// every diff tool prints, and reviewers read it as a replacement.
			out = append(out, op{'-', a[i]})
			i++
		default:
			out = append(out, op{'+', b[j]})
			j++
		}
	}
	for ; i < len(a); i++ {
		out = append(out, op{'-', a[i]})
	}
	for ; j < len(b); j++ {
		out = append(out, op{'+', b[j]})
	}
	return out
}
