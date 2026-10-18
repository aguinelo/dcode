package orchestrate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The evidence a merge question shows is what the harness measured, read
// back from the decision log — the last review.verdict of the child and its
// gate.measured of the commit proposed — and never what the child said: the
// claim is a field of the gate's line, and no part of the evidence (RN-5).
//
// Spec: docs/specs/architecture/orchestration/202610132000-orchestration.p.spec.md §5.5.

// Evidence reads the log for what was measured of child at commit. A log
// without both lines is an error: a question with no evidence is not asked.
func (l *DecisionLog) Evidence(child, commit string) (protocol.QuestionEvidence, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	f, err := os.Open(l.path)
	if err != nil {
		return protocol.QuestionEvidence{}, fmt.Errorf("orchestrate: the log could not be read for the evidence: %w", err)
	}
	defer f.Close()
	var (
		ev               protocol.QuestionEvidence
		gateAt, reviewAt uint64
	)
	r := NewLogReader(f)
	for {
		line, err := r.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return protocol.QuestionEvidence{}, fmt.Errorf("orchestrate: the log could not be read for the evidence: %w", err)
		}
		d := line.Decision
		if line.Err != nil || d.Child != child {
			continue
		}
		switch d.Kind {
		case "gate.measured":
			var g protocol.GateResult
			if json.Unmarshal(d.Measured, &g) != nil || g.Commit != commit {
				continue
			}
			ev.Gate, ev.Command, ev.Commit, ev.Base = g.Result, g.Command, g.Commit, g.Base
			ev.ExitCode, ev.DurationMS, ev.GateSeq, gateAt = g.ExitCode, g.DurationMS, d.Seq, d.Seq
		case "review.verdict":
			var v protocol.Verdict
			if json.Unmarshal(d.Measured, &v) != nil {
				continue
			}
			ev.Verdict, ev.Reviewer, ev.ReviewedCommit, ev.Findings = v.Verdict, v.SessionID, v.Commit, len(v.Findings)
			ev.ReviewSeq, reviewAt = d.Seq, d.Seq
		}
	}
	switch {
	case gateAt == 0:
		return ev, fmt.Errorf("orchestrate: the log has no gate.measured of %s for %s, so there is no evidence to show", short(commit), child)
	case reviewAt == 0:
		return ev, fmt.Errorf("orchestrate: the log has no review.verdict for %s, so there is no evidence to show", child)
	}
	return ev, nil
}
