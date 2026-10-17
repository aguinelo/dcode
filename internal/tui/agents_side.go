package tui

import (
	"fmt"
	"strings"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The live list, drawn: the top pane of an orchestrator's side column, where
// an ordinary session has its diff. The orchestrator never writes (decision 3
// of the plan), so its diff would be empty for good; what it has to show is
// its children.
//
// Each child is two rows — the mark, the title and the state; then tokens in,
// out and cache, with the PR and its CI at the right — and one more for what
// it waits on, how its last turn ended when it failed, or a refused merge. The
// three token figures are named once, under the head, as the diff pane names
// its scale: a slash-separated triple nobody was told the order of is a
// triple that will be read wrong.

// agentMarks are the marks of the list. The eight states stay distinct by
// character in ASCII too: a state told apart only by colour is not told apart
// on a terminal without any.
type agentMarks struct {
	state                          map[string]string
	unknown, ask                   string
	ciPass, ciFail, ciPending, nan string
}

func agentGlyphs(unicode bool) agentMarks {
	if unicode {
		return agentMarks{
			state: map[string]string{
				childDispatched: "○", childWorking: "●", childAwaiting: "◆", childInReview: "◐",
				childPR: "◎", childMerged: "✓", childClosed: "–", childFailed: "⊘",
			},
			unknown: "?", ask: "◆", ciPass: "✓", ciFail: "✗", ciPending: "…", nan: "?",
		}
	}
	return agentMarks{
		state: map[string]string{
			childDispatched: "o", childWorking: "*", childAwaiting: "?", childInReview: "~",
			childPR: "@", childMerged: "x", childClosed: "-", childFailed: "!",
		},
		unknown: "#", ask: "?", ciPass: "ok", ciFail: "!!", ciPending: "..", nan: "?",
	}
}

func agentStateStyle(state string) Style {
	switch state {
	case childWorking, childInReview, childPR:
		return StyleAccent
	case childAwaiting:
		return StyleWarn
	case childMerged:
		return StyleOK
	case childFailed:
		return StyleError
	}
	return StyleMeta
}

// agentStateWord is the state in the person's language. A state this client
// does not know is drawn as it came: naming it as one it knows would be a
// guess, and an empty cell would be a state nobody can see.
func agentStateWord(state string, t Strings) string {
	switch state {
	case childDispatched:
		return t.AgentDispatched
	case childWorking:
		return t.AgentWorking
	case childAwaiting:
		return t.AgentAwaiting
	case childInReview:
		return t.AgentInReview
	case childPR:
		return t.AgentPR
	case childMerged:
		return t.AgentMerged
	case childClosed:
		return t.AgentClosed
	case childFailed:
		return t.AgentFailed
	case "":
		return t.AgentNoState
	}
	return state
}

func agentWaitingWord(waiting string, t Strings) string {
	switch waiting {
	case "approval":
		return t.AgentWaitsApproval
	case "question":
		return t.AgentWaitsQuestion
	case "merge":
		return t.AgentWaitsMerge
	}
	return waiting
}

// renderAgentsPane draws the list in at most height rows.
func renderAgentsPane(m Model, g Geometry, w, height int) []string {
	gl := glyphs(g.Unicode)
	am := agentGlyphs(g.Unicode)
	p := g.Palette
	t := Text(m.Lang)
	a := m.Agents

	meta := ""
	if h := a.Main; h != nil {
		meta = t.AgentsMain + " " + healthMark(h.Health, am)
	}
	out := []string{paneHead(gl, p, "a", t.SideAgents, meta, w)}

	if n := len(a.Questions); n > 0 {
		line := t.AgentsQuestionOne
		if n != 1 {
			line = fmt.Sprintf(t.AgentsQuestionMany, n)
		}
		out = append(out, p.Apply(StyleWarn, am.ask+" "+line))
	}
	if len(a.Children) == 0 {
		return append(out, p.Apply(StyleMeta, t.AgentsNone))
	}
	out = append(out, p.Apply(StyleHint, t.AgentsTokens))

	for i, r := range a.Children {
		rows := agentRows(r, am, gl, p, t, w)
		if len(out)+len(rows) > height {
			// The rest counted, never cut silently. The count takes the last
			// row there is, even when that costs part of a child above it.
			if len(out) >= height {
				out = out[:maxInt(0, height-1)]
			}
			return append(out, p.Apply(StyleMeta, fmt.Sprintf(t.AgentsMore, len(a.Children)-i)))
		}
		out = append(out, rows...)
	}
	return out
}

func agentRows(r Agent, am agentMarks, gl marks, p Palette, t Strings, w int) []string {
	mark, ok := am.state[r.State]
	if !ok {
		mark = am.unknown
	}
	style := agentStateStyle(r.State)
	state := agentStateWord(r.State, t)
	title := r.Title
	if title == "" {
		title = t.ChildUnnamed
	}
	if room := w - 2 - visibleWidth(state) - 1; room > 1 {
		title = ellipsisTail(title, room, gl.ell)
	}
	head := p.Apply(style, mark) + " " + p.Apply(StyleProse, title)
	rows := []string{rightAlign(head, p.Apply(style, state), w)}

	usage := "  " + p.Apply(StyleMeta, fmt.Sprintf("%s/%s/%s",
		tokenCount(r.Usage.InputTokens), tokenCount(r.Usage.OutputTokens), tokenCount(r.Usage.CacheReadTokens)))
	rows = append(rows, rightAlign(usage, prCell(r.PR, am, p, t), w))

	if r.Waiting != "" {
		rows = append(rows, "  "+p.Apply(StyleWarn, t.AgentWaits+": "+agentWaitingWord(r.Waiting, t)))
	}
	if r.State == childFailed && r.Reason != "" {
		rows = append(rows, "  "+p.Apply(StyleError, t.AgentEnded+": "+r.Reason))
	}
	if r.Refused != nil {
		rows = append(rows, "  "+p.Apply(StyleError, t.AgentRefused+": "+r.Refused.Code))
	}
	return rows
}

// prCell is the PR and its CI, in GitChecks' grammar. A local merge has no
// number, and says so rather than drawing `#0`.
func prCell(pr *protocol.ChildPR, am agentMarks, p Palette, t Strings) string {
	if pr == nil {
		return ""
	}
	label := t.AgentLocal
	if pr.Number > 0 {
		label = fmt.Sprintf("#%d", pr.Number)
	}
	switch pr.CI {
	case protocol.GitChecksPass:
		return p.Apply(StyleMeta, label) + " " + p.Apply(StyleOK, am.ciPass)
	case protocol.GitChecksFail:
		return p.Apply(StyleMeta, label) + " " + p.Apply(StyleError, am.ciFail)
	case protocol.GitChecksPending:
		return p.Apply(StyleMeta, label) + " " + p.Apply(StyleWarn, am.ciPending)
	}
	return p.Apply(StyleMeta, label)
}

// healthMark is main's health — green, red, pending or unknown — as a mark.
func healthMark(health string, am agentMarks) string {
	switch health {
	case "green":
		return am.ciPass
	case "red":
		return am.ciFail
	case "pending":
		return am.ciPending
	}
	return am.nan
}

// rightAlign puts right at the end of a row of w, or leaves it off when the
// two do not fit together.
func rightAlign(left, right string, w int) string {
	if right == "" {
		return left
	}
	gap := w - visibleWidth(left) - visibleWidth(right)
	if gap < 1 {
		return left
	}
	return left + strings.Repeat(" ", gap) + right
}

// tokenCount is humanTokens with zero said as zero: a child that has spent
// nothing yet has spent 0, and an empty slot in a triple shifts the other two.
func tokenCount(n int) string {
	if n <= 0 {
		return "0"
	}
	return humanTokens(n)
}
