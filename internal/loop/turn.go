package loop

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aguinelo/dcode/internal/behavior"
	"github.com/aguinelo/dcode/internal/config"
	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/protocol"
	"github.com/aguinelo/dcode/internal/provider"
	"github.com/aguinelo/dcode/internal/tools"
)

// Phase is where a turn currently is.
type Phase string

const (
	PhaseAssembling Phase = "assembling"
	PhaseStreaming  Phase = "streaming"
	PhaseExecuting  Phase = "executing"
	PhaseBlocked    Phase = "blocked"
	PhaseDone       Phase = "done"
)

// Approver resolves a boundary crossing. Returning deny on timeout is the
// implementation's responsibility, not the loop's: with nobody to ask, the
// alternative to denying would be granting in silence.
type Approver interface {
	Approve(ctx context.Context, req protocol.ApprovalRequest) (protocol.ApprovalDecision, error)
}

// Emitter publishes observable facts. The loop never writes to a terminal:
// every client — TUI, IDE, none at all — sees the same session through this.
type Emitter interface {
	Emit(ev protocol.EventType, payload any)
}

// Config wires a turn.
type Config struct {
	Provider  provider.Provider
	Tools     *tools.Registry
	State     *tools.State
	Emitter   Emitter
	Approver  Approver
	Limits    Limits
	Mode      policy.SandboxMode
	Policy    policy.ApprovalPolicy
	Model     string
	Parallel  int
	CtxConfig ce.Config
	// Summarise generates compaction text. Supplied by the caller because it
	// needs a model call, which is what keeps the context engine pure.
	Summarise func(ctx context.Context, msgs []ce.Message) (string, error)

	// Skills are indexed in the prefix and loaded here, on trigger (RN-7).
	Skills []behavior.Skill
	// InstructionChain is the set of instruction files frozen at session
	// creation. Anything found outside it becomes a reminder, never a prefix
	// change (RN-6 of the configuration spec).
	InstructionChain []string
	// ReadFile backs the changed-on-disk check. Injected so the loop stays
	// testable without a filesystem; nil disables the check.
	ReadFile func(path string) (string, error)
	// Rules are the patterns that ask a question the sandbox cannot.
	Rules policy.Rules
	// ShowReasoning forwards the model's thinking to clients.
	ShowReasoning bool
	// Reminders disables the appended-notice channel when false.
	Reminders bool
	// Now is the clock used to time tool calls. Nil means the real one.
	Now func() time.Time
}

// Outcome is how a turn ended.
type Outcome struct {
	TurnID     string
	Reason     string
	Iterations int
	Usage      provider.Usage
}

// Engine runs turns against one session.
type Engine struct {
	cfg     Config
	session ce.Session
	turnSeq int
	// seenDirs stops the same out-of-chain instruction being appended once per
	// batch. Told once is guidance; told every batch is noise the model starts
	// to discount.
	seenDirs map[string]struct{}
}

// now is the engine's clock. Injectable so a test can assert a duration without
// waiting for one.
func (e *Engine) now() time.Time {
	if e.cfg.Now != nil {
		return e.cfg.Now()
	}
	return time.Now()
}

// New builds an engine over an initial session.
func New(cfg Config, session ce.Session) *Engine {
	if cfg.Parallel <= 0 {
		cfg.Parallel = 4
	}
	cfg.Limits = cfg.Limits.withFamily(familyIterations(cfg.Provider))
	return &Engine{cfg: cfg, session: session, seenDirs: map[string]struct{}{}}
}

func familyIterations(p provider.Provider) int {
	if p == nil {
		return 0
	}
	return p.Limits().MaxIterations
}

// Session returns the current session state.
func (e *Engine) Session() ce.Session { return e.session }

// Run executes one turn: appends the input, then cycles until the model stops
// asking for tools.
func (e *Engine) Run(ctx context.Context, input string) (Outcome, error) {
	e.turnSeq++
	turnID := fmt.Sprintf("t%d", e.turnSeq)
	e.emit(protocol.EventTurnStarted, protocol.TurnStarted{TurnID: turnID})

	e.session.History = append(e.session.History, ce.Message{Role: ce.RoleUser, Text: input})

	// Skill bodies are appended, never prefixed. The index in the prefix is
	// what the model reads every turn; the body is only paid for in the turns
	// that actually need it.
	for _, s := range behavior.Match(input, e.cfg.Skills) {
		e.session.History = append(e.session.History, ce.Message{
			Role: ce.RoleUser, Text: behavior.RenderSkill(s), Reminder: true,
		})
	}

	out := Outcome{TurnID: turnID}
	var recent []ce.ToolCall
	compacted := false

	for {
		if err := ctx.Err(); err != nil {
			return e.finish(out, protocol.StopInterrupted), nil
		}

		// Compaction is checked exactly once per iteration, here and nowhere
		// else. A second check point is how compaction becomes incremental by
		// accident, which invalidates the cache every turn.
		before := summaryMark(e.session)
		if err := e.maybeCompact(ctx); err != nil {
			return e.finish(out, protocol.StopError), err
		}
		compacted = compacted || summaryMark(e.session) != before

		msgs, err := ce.Assemble(e.session)
		if err != nil {
			return e.finish(out, protocol.StopError), err
		}

		text, calls, usage, perr := e.stream(ctx, turnID, msgs)
		out.Usage.InputTokens += usage.InputTokens
		out.Usage.OutputTokens += usage.OutputTokens
		out.Usage.CacheReadTokens += usage.CacheReadTokens

		if perr != nil {
			switch provider.Decide(perr) {
			case provider.DecisionSilent:
				return e.finish(out, protocol.StopInterrupted), nil
			case provider.DecisionCompact:
				if e.forceCompact(ctx) {
					continue
				}
				return e.finish(out, protocol.StopError), perr
			default:
				e.emit(protocol.EventSessionError, protocol.Error{
					Code: string(perr.Class), Message: perr.Message,
				})
				return e.finish(out, protocol.StopError), perr
			}
		}

		if text != "" || len(calls) > 0 {
			e.session.History = append(e.session.History, ce.Message{
				Role: ce.RoleAssistant, Text: text, ToolCalls: calls,
			})
		}
		if len(calls) == 0 {
			return e.finish(out, protocol.StopDone), nil
		}

		recent = append(recent, calls...)
		if IsRepeat(recent, e.cfg.Limits.MaxIdenticalCalls) {
			// The same call three times is a loop, not persistence. Stopping
			// here costs one turn; not stopping costs the budget.
			return e.finish(out, protocol.StopRepeatLoop), nil
		}

		results, batch := e.execute(ctx, turnID, calls)
		e.session.History = append(e.session.History, results...)

		batch.Compacted = compacted
		compacted = false
		e.session.History = append(e.session.History, e.reminders(batch)...)

		out.Iterations++
		if out.Iterations >= e.cfg.Limits.MaxIterations {
			return e.finish(out, protocol.StopMaxIterations), nil
		}
		if m := e.cfg.Limits.MaxTurnTokens; m > 0 && out.Usage.OutputTokens >= m {
			return e.finish(out, protocol.StopMaxTokens), nil
		}
	}
}

// summaryMark identifies the current summary, so a compaction that happened
// during this turn can be told apart from one that happened before it.
func summaryMark(s ce.Session) int {
	if s.Summary == nil {
		return -1
	}
	return s.Summary.UpToIdx
}

// stream consumes one model call.
func (e *Engine) stream(ctx context.Context, turnID string, msgs []ce.Message) (
	string, []ce.ToolCall, provider.Usage, *provider.ProviderError,
) {
	var (
		text  strings.Builder
		calls []ce.ToolCall
		usage provider.Usage
	)

	ch, err := e.cfg.Provider.Stream(ctx, provider.Request{
		Model:    e.cfg.Model,
		Messages: msgs,
		Tools:    e.cfg.Tools.Defs(),
	})
	if err != nil {
		return "", nil, usage, toProviderError(err)
	}

	for ev := range ch {
		switch ev.Type {
		case provider.EventTextDelta:
			text.WriteString(ev.Text)
			e.emit(protocol.EventMessageDelta, protocol.MessageDelta{
				TurnID: turnID, Text: ev.Text,
			})
		case provider.EventReasoningDelta:
			// Forwarded, never appended. The history is the one place this must
			// not reach: a model that reads its own thinking back as something
			// it said out loud starts defending it, and it would be paid for on
			// every subsequent turn of the session.
			//
			// Switchable because on a tool-calling turn the thinking runs five
			// to ten times the size of the answer, and it shares the session's
			// event budget with everything worth replaying.
			if e.cfg.ShowReasoning {
				e.emit(protocol.EventMessageReasoning, protocol.MessageReasoning{
					TurnID: turnID, Text: ev.Text,
				})
			}

		case provider.EventToolCall:
			calls = append(calls, *ev.ToolCall)
			e.emit(protocol.EventToolRequested, protocol.ToolRequested{
				TurnID: turnID, ToolCallID: ev.ToolCall.ID,
				Name: ev.ToolCall.Name, Input: ev.ToolCall.Input,
			})
		case provider.EventDone:
			if ev.Usage != nil {
				usage = *ev.Usage
			}
		case provider.EventError:
			// A tool-schema failure is the model's to fix, so it comes back as
			// a tool result rather than ending the turn.
			if ev.Err != nil && ev.Err.Class == provider.ErrClassToolSchema {
				e.session.History = append(e.session.History, ce.Message{
					Role: ce.RoleTool,
					ToolResult: &ce.ToolResult{
						Output:  ev.Err.Message,
						IsError: true,
					},
				})
				continue
			}
			return text.String(), calls, usage, ev.Err
		}
	}
	return text.String(), calls, usage, nil
}

// execute runs the calls and returns their results in emission order.
// batchFacts is what one batch of tool calls produced that the reminder
// channel cares about. Collected here rather than inferred later: after the
// results are in history there is no way to tell a denial from an ordinary
// error.
type batchFacts struct {
	Parallel    int
	Denied      []string
	TouchedDirs []string
	Compacted   bool
}

func (e *Engine) execute(ctx context.Context, turnID string, calls []ce.ToolCall) ([]ce.Message, batchFacts) {
	execs := make([]Execution, 0, len(calls))
	for i, c := range calls {
		ex := Execution{Index: i, Call: c}
		tool, ok := e.cfg.Tools.Get(c.Name)
		if !ok {
			ex.Err = fmt.Errorf("tool %q is not available; available: %s",
				c.Name, strings.Join(e.cfg.Tools.Names(), ", "))
		} else if req, err := tool.Declare(c.Input); err != nil {
			ex.Err = err
		} else {
			ex.Declare = req
		}
		execs = append(execs, ex)
	}

	results := make([]ce.Message, len(execs))
	denied := make([]string, len(execs))
	facts := batchFacts{}

	for _, ex := range execs {
		for _, a := range ex.Declare.Paths {
			facts.TouchedDirs = append(facts.TouchedDirs, filepath.Dir(a.Path))
		}
	}

	groups := Schedule(execs, e.cfg.Parallel)
	for _, g := range groups {
		if len(g) > facts.Parallel {
			facts.Parallel = len(g)
		}
		var wg sync.WaitGroup
		for _, ex := range g {
			wg.Add(1)
			go func(ex Execution) {
				defer wg.Done()
				// Results land at the emission index, never at completion
				// order. That is what keeps history reproducible and
				// golden-testable regardless of which call finished first.
				msg, refused := e.runOne(ctx, turnID, ex)
				results[ex.Index] = msg
				if refused {
					denied[ex.Index] = ex.Call.Name
				}
			}(ex)
		}
		wg.Wait()
	}

	for _, name := range denied {
		if name != "" {
			facts.Denied = append(facts.Denied, name)
		}
	}
	return results, facts
}

// reminders renders the appended channel for one batch.
//
// The whole channel is optional, and switching it off leaves the loop otherwise
// identical: nothing here changes what is permitted, only what the model is
// told about what already happened.
func (e *Engine) reminders(f batchFacts) []ce.Message {
	if !e.cfg.Reminders {
		return nil
	}
	st := behavior.SessionState{
		DeniedTools:   f.Denied,
		Compacted:     f.Compacted,
		ParallelBatch: f.Parallel,
	}
	if e.cfg.ReadFile != nil {
		st.ChangedFiles = e.cfg.State.ChangedSinceRead(e.cfg.ReadFile)
	}
	st.OutOfChain = e.outOfChain(f.TouchedDirs)

	var out []ce.Message
	for _, r := range behavior.Emit(st) {
		out = append(out, ce.Message{
			Role: ce.RoleUser, Text: behavior.Render(r), Reminder: true,
		})
	}
	return out
}

// outOfChain finds instruction files in directories this batch touched that
// were not in the chain frozen at session creation.
func (e *Engine) outOfChain(dirs []string) []behavior.OutOfChainInstruction {
	if e.cfg.ReadFile == nil || len(dirs) == 0 {
		return nil
	}
	var out []behavior.OutOfChainInstruction
	for _, dir := range dirs {
		if _, seen := e.seenDirs[dir]; seen {
			continue
		}
		e.seenDirs[dir] = struct{}{}

		path, found := config.OutOfChain(dir, e.cfg.InstructionChain)
		if !found {
			continue
		}
		text, err := e.cfg.ReadFile(path)
		if err != nil {
			continue
		}
		out = append(out, behavior.OutOfChainInstruction{Path: path, Text: text})
	}
	return out
}

// runOne evaluates policy, asks for approval if needed, and executes. The
// second return reports a refusal by the user, which is not the same thing as
// an error and must not be reported as one.
func (e *Engine) runOne(ctx context.Context, turnID string, ex Execution) (ce.Message, bool) {
	fail := func(msg string) ce.Message {
		return ce.Message{Role: ce.RoleTool, ToolResult: &ce.ToolResult{
			ToolCallID: ex.Call.ID, Output: msg, IsError: true,
		}}
	}

	if ex.Err != nil {
		return fail(ex.Err.Error()), false
	}
	tool, ok := e.cfg.Tools.Get(ex.Call.Name)
	if !ok {
		return fail(fmt.Sprintf("tool %q is not available", ex.Call.Name)), false
	}

	// Every execution passes through the evaluator. There is no alternative
	// path, in any mode, for any tool.
	verdict := e.evaluate(ex.Declare)
	if verdict.Decision == policy.DecisionEscalate {
		d, err := e.askApproval(ctx, turnID, ex, verdict)
		if err != nil || d == protocol.ApprovalDeny {
			return fail(fmt.Sprintf("denied: %s", verdict.Reason)), true
		}
	}
	if verdict.Decision == policy.DecisionDeny {
		return fail(fmt.Sprintf("not permitted: %s", verdict.Reason)), false
	}

	// Measured around Execute only: the wait for an approval is the user's
	// time, not the tool's, and folding it in would make every gated call look
	// slow.
	started := e.now()
	res, err := tool.Execute(ctx, ex.Call.Input, e.cfg.State)
	elapsed := e.now().Sub(started)
	if err != nil {
		return fail(err.Error()), false
	}

	e.emit(protocol.EventToolCompleted, protocol.ToolCompleted{
		ToolCallID: ex.Call.ID, OK: !res.IsError,
		Output: res.Output, Truncated: res.Truncated,
		Lines: res.Meta.Lines, Files: res.Meta.Files,
		Added: res.Meta.Added, Removed: res.Meta.Removed,
		ExitCode: res.Meta.ExitCode, HasExit: res.Meta.HasExit,
		DurationMS: int(elapsed.Milliseconds()),
		Diff:       res.Meta.Diff,
	})
	if ex.Call.Name == "plan" && !res.IsError {
		e.emit(protocol.EventPlanUpdated, protocol.PlanUpdated{Items: e.cfg.State.Plan()})
	}

	return ce.Message{Role: ce.RoleTool, ToolResult: &ce.ToolResult{
		ToolCallID: ex.Call.ID, Output: res.Output,
		IsError: res.IsError, Truncated: res.Truncated,
	}}, false
}

func (e *Engine) evaluate(req policy.Request) policy.Verdict {
	resolver := e.cfg.State.Resolver
	resolved := policy.Request{
		Tool: req.Tool, Network: req.Network, Command: req.Command,
	}
	for _, a := range req.Paths {
		acc, err := resolver.Resolve(a.Path, a.Write)
		if err != nil {
			// An unresolvable path is treated as outside the workspace. The
			// safe reading of "cannot tell" is never "allow".
			resolved.Paths = append(resolved.Paths, policy.Access{Path: a.Path, Write: a.Write})
			continue
		}
		resolved.Paths = append(resolved.Paths, acc)
	}
	return policy.Evaluate(resolved, e.cfg.Mode, e.cfg.Policy, e.cfg.Rules, resolver.InWorkspace)
}

func (e *Engine) askApproval(ctx context.Context, turnID string, ex Execution, v policy.Verdict) (
	protocol.ApprovalDecision, error,
) {
	req := protocol.ApprovalRequest{
		ApprovalID:      fmt.Sprintf("%s-%d", turnID, ex.Index),
		TurnID:          turnID,
		ToolCallID:      ex.Call.ID,
		Tool:            ex.Call.Name,
		Command:         ex.Declare.Command,
		BoundaryCrossed: string(v.Boundary),
		Reason:          v.Reason,
		Rule:            v.Rule,
	}
	e.emit(protocol.EventApprovalRequired, req)

	if e.cfg.Approver == nil {
		// No approver means nobody to ask. Denying is the only safe reading.
		e.emit(protocol.EventApprovalResolved, protocol.ApprovalResolved{
			ApprovalID: req.ApprovalID, Decision: protocol.ApprovalDeny,
		})
		return protocol.ApprovalDeny, nil
	}

	d, err := e.cfg.Approver.Approve(ctx, req)
	if err != nil || !d.Valid() {
		d = protocol.ApprovalDeny
	}
	e.emit(protocol.EventApprovalResolved, protocol.ApprovalResolved{
		ApprovalID: req.ApprovalID, Decision: d,
	})
	return d, err
}

// maybeCompact checks once per iteration whether the context needs compacting.
func (e *Engine) maybeCompact(ctx context.Context) error {
	cfg := e.cfg.CtxConfig
	if cfg.Window <= 0 && e.cfg.Provider != nil {
		if w, err := e.cfg.Provider.Window(e.cfg.Model); err == nil {
			cfg.Window = w
		}
	}
	plan, ok := ce.Plan(e.session, cfg)
	if !ok {
		return nil
	}
	return e.applyCompaction(ctx, plan)
}

// forceCompact is the recovery path when the provider says the context is too
// large: compact even if the local estimate disagreed.
func (e *Engine) forceCompact(ctx context.Context) bool {
	cfg := e.cfg.CtxConfig
	cfg.Window = 1 // force the trigger; the estimate cannot be below this
	plan, ok := ce.Plan(e.session, cfg)
	if !ok {
		return false
	}
	return e.applyCompaction(ctx, plan) == nil
}

func (e *Engine) applyCompaction(ctx context.Context, plan ce.CompactionPlan) error {
	text := "Earlier work in this session was summarised to save room."
	if e.cfg.Summarise != nil {
		span := e.session.History[plan.FromIdx:plan.ToIdx]
		if s, err := e.cfg.Summarise(ctx, span); err == nil && strings.TrimSpace(s) != "" {
			text = s
		}
	}
	e.session = ce.Apply(e.session, plan, text)
	e.emit(protocol.EventSessionCompacted, protocol.SessionCompacted{
		FromSeq: uint64(plan.FromIdx), ToSeq: uint64(plan.ToIdx),
	})
	return nil
}

func (e *Engine) finish(out Outcome, reason string) Outcome {
	out.Reason = reason
	ev := protocol.TurnCompleted{TurnID: out.TurnID, Reason: reason}
	// Absent rather than zero when the provider said nothing: unknown tokens
	// and no tokens are different facts, and a client shows them differently.
	if out.Usage != (provider.Usage{}) {
		ev.Usage = &protocol.Usage{
			InputTokens:      out.Usage.InputTokens,
			OutputTokens:     out.Usage.OutputTokens,
			CacheReadTokens:  out.Usage.CacheReadTokens,
			CacheWriteTokens: out.Usage.CacheWriteTokens,
		}
	}
	e.emit(protocol.EventTurnCompleted, ev)
	return out
}

func (e *Engine) emit(t protocol.EventType, payload any) {
	if e.cfg.Emitter != nil {
		e.cfg.Emitter.Emit(t, payload)
	}
}

func toProviderError(err error) *provider.ProviderError {
	var pe *provider.ProviderError
	if ok := asProviderError(err, &pe); ok {
		return pe
	}
	return &provider.ProviderError{
		Class: provider.ErrClassTransport, Message: err.Error(), Retryable: true,
	}
}

func asProviderError(err error, target **provider.ProviderError) bool {
	if pe, ok := err.(*provider.ProviderError); ok {
		*target = pe
		return true
	}
	type unwrapper interface{ Unwrap() error }
	if u, ok := err.(unwrapper); ok && u.Unwrap() != nil {
		return asProviderError(u.Unwrap(), target)
	}
	return false
}

// SortedPlan returns the session plan in a stable order, for clients that
// render it.
func SortedPlan(items []protocol.PlanItem) []protocol.PlanItem {
	out := make([]protocol.PlanItem, len(items))
	copy(out, items)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
