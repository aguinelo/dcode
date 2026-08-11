package loop

import (
	"context"
	"fmt"
	"sort"
	"strings"

	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/tools"
)

// DelegateLimits bound a delegated turn.
//
// None of them is optional. Delegation without a ceiling is a cost multiplier,
// not a saving.
type DelegateLimits struct {
	// MaxIterations is the child's own cap, smaller than the parent's on
	// purpose: the child answers ONE question, and a child that needs fifty
	// iterations is a question that should have been split.
	MaxIterations int
	// MaxResultBytes caps the report. Exceeded, it truncates and declares it.
	// A child returning 50KB defeats the point — the cost of reading comes back
	// through the answer.
	MaxResultBytes int
}

// DelegateResult is what a child turn hands back.
type DelegateResult struct {
	// Conclusion is the answer, in prose.
	Conclusion string
	// Read are the paths the child actually opened.
	//
	// This is the mitigation for the problem that survives read-only
	// delegation: "I found nothing wrong in the payment module" — did it find
	// nothing, or did it not look? Redoing the work to check would cancel the
	// whole gain. A list of paths does not prove the child understood, but it
	// proves it looked, and it turns "trust me" into something a person can
	// spot-check.
	Read []string
	// Unread are paths a rule refused. Reported, never swallowed: a conclusion
	// with an undeclared hole is a wrong conclusion wearing the face of a
	// complete one.
	Unread []string
	// Truncated reports that the conclusion was cut.
	Truncated bool
}

// String renders the result for the parent's history.
func (r DelegateResult) String() string {
	var b strings.Builder
	b.WriteString(r.Conclusion)
	if r.Truncated {
		b.WriteString("\n\n… report truncated.")
	}
	if len(r.Read) > 0 {
		fmt.Fprintf(&b, "\n\nlooked at: %s", strings.Join(r.Read, ", "))
	}
	if len(r.Unread) > 0 {
		fmt.Fprintf(&b, "\ncould not read: %s", strings.Join(r.Unread, ", "))
	}
	return b.String()
}

// denyAll is the child's approver, and it is not configurable.
//
// ADR-02 is explicit that sandbox and approval are orthogonal axes and that
// approval is CONSENT. Consent given to the parent does not transfer to the
// child: the user approved that, not this.
//
// So a delegated turn never asks. Not a prompt, because the user asked ONE
// question and being interrupted by N children destroys exactly the gain that
// delegating bought — and they have no context to judge a request they did not
// make. Not silence either, because a conclusion with an undeclared hole is a
// wrong conclusion that looks complete.
type denyAll struct{ denied []string }

func (d *denyAll) Approve(_ context.Context, req protocol.ApprovalRequest) (protocol.ApprovalDecision, error) {
	target := req.Command
	if target == "" {
		target = req.Tool
	}
	d.denied = append(d.denied, target)
	return protocol.ApprovalDeny, nil
}

// Delegate runs a read-only sub-turn and returns its report.
//
// The lock is structural, not conventional. The child is built here with
// policy.ModeReadOnly, and the tool it would need to delegate again is not in
// its registry — so nesting is impossible rather than forbidden. Compare
// DoctrineOverlay, where Safety is not a field: the guarantee is the absence,
// not a condition somewhere that could be edited out.
func (e *Engine) Delegate(ctx context.Context, task, path string, lim DelegateLimits) (DelegateResult, error) {
	reg, readable := readOnlyRegistry(e.cfg.Tools)

	approver := &denyAll{}
	child := New(Config{
		Provider: e.cfg.Provider,
		Tools:    reg,
		State:    tools.NewState(e.cfg.State.Resolver, e.cfg.State.Limits),
		Emitter:  nil, // the child's steps are not the parent session's events
		Approver: approver,
		// Read-only, fixed here. It is not a field of the tool input and the
		// model never gets to pass it.
		Mode:      policy.ModeReadOnly,
		Policy:    e.cfg.Policy,
		Model:     e.cfg.Model,
		Parallel:  e.cfg.Parallel,
		CtxConfig: e.cfg.CtxConfig,
		Rules:     e.cfg.Rules,
		Limits:    Limits{MaxIterations: lim.MaxIterations, MaxIdenticalCalls: e.cfg.Limits.MaxIdenticalCalls},
		Reminders: e.cfg.Reminders,
	}, ce.Session{
		// The task, not the parent's history. That is the entire point:
		// copying the history back would return the cost delegation exists to
		// avoid.
		Instructions: delegateInstructions(readable),
		Tools:        reg.Defs(),
	})

	out, err := child.Run(ctx, delegateTask(task, path))
	if err != nil {
		return DelegateResult{}, err
	}

	// The child's tokens are debited from the parent. Without this the parent's
	// ceiling is fiction: a turn could delegate its way past any budget.
	e.delegated.InputTokens += out.Usage.InputTokens
	e.delegated.OutputTokens += out.Usage.OutputTokens
	e.delegated.CacheReadTokens += out.Usage.CacheReadTokens

	res := DelegateResult{
		Conclusion: lastText(child.Session()),
		Read:       child.cfg.State.ReadPaths(),
		Unread:     uniqueSortedStrings(approver.denied),
	}
	if n := lim.MaxResultBytes; n > 0 && len(res.Conclusion) > n {
		res.Conclusion = res.Conclusion[:n]
		res.Truncated = true
	}
	return res, nil
}

// readOnlyRegistry is the child's tool set: everything that only reads, and
// nothing that delegates.
//
// The absence of the delegating tool is what makes nesting impossible. Nested
// delegation is exponential cost, and the error lands far from its cause.
func readOnlyRegistry(parent *tools.Registry) (*tools.Registry, []string) {
	var kept []tools.Tool
	var names []string
	for _, name := range parent.Names() {
		if name == ExploreToolName || !readOnlyTools[name] {
			continue
		}
		if t, ok := parent.Get(name); ok {
			kept = append(kept, t)
			names = append(names, name)
		}
	}
	return tools.NewRegistry(kept...), names
}

// readOnlyTools names the tools a child may have.
//
// An allow list rather than a deny list: a tool added later is excluded until
// someone thinks about it, which is the safe direction to be wrong in.
var readOnlyTools = map[string]bool{
	"read":   true,
	"glob":   true,
	"grep":   true,
	"symbol": true,
}

// ExploreToolName is the delegating tool, named here so the child's registry can
// exclude it without importing the tools package's own definition.
const ExploreToolName = "explore"

func delegateInstructions(toolNames []string) string {
	return "You are answering one question about a codebase, for another agent.\n\n" +
		"You can only read. You have " + strings.Join(toolNames, ", ") +
		" and nothing else — no writing, no commands, no delegating.\n\n" +
		"Answer in at most a few paragraphs. Say what you found and where, with " +
		"paths and line numbers. If something was unreadable, say so rather than " +
		"concluding around it: an answer with an undeclared hole is worse than a " +
		"short one.\n\n" +
		"Do not describe your search. The answer is what is wanted, not the route to it."
}

func delegateTask(task, path string) string {
	if strings.TrimSpace(path) == "" {
		return task
	}
	return task + "\n\nLook under: " + path
}

// lastText is the child's final answer.
func lastText(s ce.Session) string {
	for i := len(s.History) - 1; i >= 0; i-- {
		if m := s.History[i]; m.Role == ce.RoleAssistant && strings.TrimSpace(m.Text) != "" {
			return strings.TrimSpace(m.Text)
		}
	}
	return ""
}

func uniqueSortedStrings(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// Explore adapts the engine to the tools.Delegator interface.
//
// The adapter lives here rather than in the tools package because everything it
// decides — read-only mode, the reduced registry, the denying approver, the
// budget debit — is a property of what a turn is, and turns belong to the loop.
func (e *Engine) Explore(ctx context.Context, task, path string) (string, []string, []string, bool, error) {
	res, err := e.Delegate(ctx, task, path, DelegateLimits{
		MaxIterations:  e.cfg.DelegateMaxIterations,
		MaxResultBytes: e.cfg.DelegateMaxResultBytes,
	})
	if err != nil {
		return "", nil, nil, false, err
	}
	return res.Conclusion, res.Read, res.Unread, res.Truncated, nil
}
