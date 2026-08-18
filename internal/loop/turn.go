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
	// Backoff is how long to wait after a retryable provider error. The zero
	// value uses the shipped policy.
	Backoff provider.Backoff
	// BudgetNotice switches the occupancy warning on. Off leaves the
	// post-compaction notice as the only signal, which is what it was before.
	BudgetNotice bool
	// Done is the definition of done for this workspace. Empty means the turn
	// ends when the model stops calling tools, which is what it did before.
	Done DoneSet
	// DoneEnabled switches re-entry on unmet criteria on.
	DoneEnabled bool
	// MaxStallCycles is how many cycles without progress end the turn in
	// StopIncomplete. Two, because the legitimate case exists: one cycle to
	// diagnose, another to fix. Three is the model going in circles.
	MaxStallCycles int
	// DoneTimeout caps one criterion. A check that never finishes is not a
	// check, and hanging the turn is worse than reporting the overrun.
	DoneTimeout time.Duration
	// RunCriterion executes a criterion command. Injected, and it goes through
	// the sandbox like everything else.
	RunCriterion CriterionRunner
	// DelegateMaxIterations caps a child turn. Smaller than the parent's on
	// purpose: the child answers ONE question, and one that needs fifty
	// iterations should have been split.
	DelegateMaxIterations int
	// DelegateMaxResultBytes caps the child's report. A child returning 50KB
	// defeats the point — the cost of reading returns through the answer.
	DelegateMaxResultBytes int
	// WrittenPaths reports what the session has written, for the protected-path
	// check. Nil disables it.
	WrittenPaths func() []string
	// AfterTurn is told what the session has written, once per turn, after the
	// turn ends. Injected like every other side effect the loop needs: it knows
	// when a turn finishes and what was touched, and nothing about what any of
	// it means.
	//
	// Called on every ending, including an interrupted one — the files a
	// half-finished turn wrote are the ones most likely to need attention.
	AfterTurn func(written []string)
	// WriteSeq reports how many writes the session has made, ever-increasing.
	//
	// A COUNT, not a set and not a clock. The set cannot answer this: rewriting
	// a file already in it leaves it unchanged, and rewriting a file after the
	// check is the exact case being caught. A clock would answer it and put a
	// value that varies per run into something a person compares between turns.
	//
	// Nil leaves the seal as it was — staleness unknown, never asserted.
	WriteSeq func() uint64
	// Steer hands over what the person said while this turn was running, or ""
	// when they said nothing. Called once at the top of every round.
	//
	// A puller rather than a channel: the session owns the engine, so the
	// session is what holds the queue, and a channel here would be a second
	// place for a message to sit and be forgotten. Nil is the old behaviour and
	// costs nothing.
	Steer func() string

	// Sleep is how the loop waits out a backoff. Injectable for the same reason
	// Now is: a test that asserts retry behaviour should not spend fifteen
	// seconds proving it. Nil means the real one.
	Sleep func(ctx context.Context, d time.Duration) bool
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
	// budgetBand is the highest occupancy band already announced. It lives here
	// rather than being derived per turn because the announcement is
	// edge-triggered: emitting while the fraction is merely above a threshold
	// repeats the same reminder every turn, and a warning that is always there
	// stops being read.
	budgetBand ce.Band
	// toldUnplanned latches the one notice about work spreading with no plan.
	//
	// Per session and not per turn, and cleared the moment a plan appears: told
	// every batch is noise the model learns to discount — the same reasoning
	// seenDirs carries — but a session that plans, finishes, and sprawls again
	// on a second task is in the position it was in the first time.
	toldUnplanned bool

	// walls counts, per turn, how often a tool failed the same way on the same
	// path. Keyed by tool, code and path, because "read failed" twice on two
	// different files is two ordinary misses; twice on the SAME file is the
	// repository saying something.
	walls map[string]int
	// toldWorthRemembering latches the notice. Once per turn: repeating it every
	// round would be nagging about a fact that has not changed.
	toldWorthRemembering bool
	// lastReport is what the most recent done check found, so the client can
	// show the seal and the turn can report what was left.
	lastReport Report
	// checkedAt is the write generation the last check was stamped against. The
	// seal compares it with the current one, which is the difference between
	// "the suite passed" and "the suite passed on this code".
	checkedAt uint64
	// delegated is what child turns cost, debited from this turn. Without it
	// the parent's ceiling is fiction.
	delegated provider.Usage
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
func (e *Engine) Run(ctx context.Context, input string, images ...ce.Image) (Outcome, error) {
	e.turnSeq++
	turnID := fmt.Sprintf("t%d", e.turnSeq)
	e.emit(protocol.EventTurnStarted, protocol.TurnStarted{TurnID: turnID, Text: input})

	// The pictures ride with the question, in one message. Splitting them
	// would lose which one the question is about.
	e.session.History = append(e.session.History,
		ce.Message{Role: ce.RoleUser, Text: input, Images: images})

	// Skill bodies are appended, never prefixed. The index in the prefix is
	// what the model reads every turn; the body is only paid for in the turns
	// that actually need it.
	for _, s := range behavior.Match(input, e.cfg.Skills) {
		e.session.History = append(e.session.History, ce.Message{
			Role: ce.RoleUser, Text: behavior.RenderSkill(s), Reminder: true,
		})
	}

	// A turn is the unit that can be undone, so it is the unit that starts a
	// new set of snapshots. Anything the previous turn changed has been read
	// by the person by now; undo that reached back through it would go further
	// than the last thing, which is not undo.
	if e.cfg.State != nil {
		e.cfg.State.BeginTurn()
	}

	out := Outcome{TurnID: turnID}
	var recent []ce.ToolCall
	compacted := false
	stall := 0
	attempts := 0
	var unmet []string
	// toldUnverified latches the one announcement about work nothing could
	// check. Per turn, because repeating it every cycle would be a loop with
	// nothing to make progress on.
	toldUnverified := false

	for {
		if err := ctx.Err(); err != nil {
			return e.finishInterrupted(out), nil
		}

		// Before compaction, so a correction is part of what compaction sees
		// rather than something appended to a window already measured.
		e.deliverSteering(turnID)

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
		out.Usage.InputTokens += e.delegated.InputTokens
		out.Usage.OutputTokens += e.delegated.OutputTokens
		out.Usage.CacheReadTokens += e.delegated.CacheReadTokens
		e.delegated = provider.Usage{}

		if perr != nil {
			switch provider.Decide(perr) {
			case provider.DecisionSilent:
				return e.finish(out, protocol.StopInterrupted), nil
			case provider.DecisionCompact:
				if e.forceCompact(ctx) {
					continue
				}
				return e.finish(out, protocol.StopError), perr
			case provider.DecisionWait, provider.DecisionRetry:
				// Both were falling through to abort, so a rate limit or a
				// dropped connection ended the turn — with the work already
				// done in it lost, and nothing said about why waiting was not
				// tried.
				attempts++
				d, ok := e.cfg.Backoff.Wait(attempts, perr)
				if !ok {
					e.emit(protocol.EventSessionError, protocol.Error{
						Code: string(perr.Class), Message: perr.Message,
					})
					return e.finish(out, protocol.StopError), perr
				}
				if !e.sleep(ctx, d) {
					return e.finishInterrupted(out), nil
				}
				continue
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
			// Step 4. The turn does not end merely because the model stopped
			// asking for tools: done is a checked condition, not a declaration.
			reason, more := e.checkDone(ctx, &stall, &unmet, &toldUnverified)
			if more != nil {
				e.session.History = append(e.session.History, more...)
				out.Iterations++
				if out.Iterations >= e.cfg.Limits.MaxIterations {
					return e.finish(out, protocol.StopMaxIterations), nil
				}
				continue
			}
			return e.finish(out, reason), nil
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
		batch.BudgetCrossed = e.crossBudget()
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
	// Walls are the failures this batch produced, keyed by tool, code and path.
	// Collected here for the same reason Denied is: once a result is in history
	// there is no way to tell one failure from another.
	Walls []string
	// BudgetCrossed is set only on the iteration a band is crossed upward.
	BudgetCrossed ce.Band
}

func (e *Engine) execute(ctx context.Context, turnID string, calls []ce.ToolCall) ([]ce.Message, batchFacts) {
	execs := make([]ToolExecution, 0, len(calls))
	for i, c := range calls {
		ex := ToolExecution{Index: i, Call: c}
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
	walls := make([]string, len(execs))
	facts := batchFacts{}

	for _, ex := range execs {
		for _, a := range ex.Declare.Paths {
			facts.TouchedDirs = append(facts.TouchedDirs, filepath.Dir(a.Path))
		}
	}

	// The verdict decides the grouping, not just the outcome: a call the user
	// will be asked about runs alone (table 4.2). Evaluating here as well as in
	// runOne is free — evaluate is pure — and keeps policy out of the scheduler.
	groups := Schedule(execs, e.cfg.Parallel, func(ex ToolExecution) bool {
		return e.evaluate(ex.Declare).Decision == policy.DecisionEscalate
	})
	for _, g := range groups {
		if len(g) > facts.Parallel {
			facts.Parallel = len(g)
		}
		var wg sync.WaitGroup
		for _, ex := range g {
			wg.Add(1)
			go func(ex ToolExecution) {
				defer wg.Done()
				// Results land at the emission index, never at completion
				// order. That is what keeps history reproducible and
				// golden-testable regardless of which call finished first.
				msg, refused, wall := e.runOne(ctx, turnID, ex)
				results[ex.Index] = msg
				walls[ex.Index] = wall
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
	for _, w := range walls {
		if w != "" {
			facts.Walls = append(facts.Walls, w)
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
		BudgetCrossed: behavior.BudgetBand(f.BudgetCrossed),
	}
	if e.cfg.ReadFile != nil {
		st.ChangedFiles = e.cfg.State.ChangedSinceRead(e.cfg.ReadFile)
	}
	st.OutOfChain = e.outOfChain(f.TouchedDirs)
	st.UnplannedChange = e.unplannedChange()
	st.WorthRemembering = e.hitTheSameWallTwice(f)

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
func (e *Engine) runOne(ctx context.Context, turnID string, ex ToolExecution) (ce.Message, bool, string) {
	fail := func(msg string) ce.Message {
		return ce.Message{Role: ce.RoleTool, ToolResult: &ce.ToolResult{
			ToolCallID: ex.Call.ID, Output: msg, IsError: true,
		}}
	}

	if ex.Err != nil {
		return fail(ex.Err.Error()), false, ""
	}
	tool, ok := e.cfg.Tools.Get(ex.Call.Name)
	if !ok {
		return fail(fmt.Sprintf("tool %q is not available", ex.Call.Name)), false, ""
	}

	// Every execution passes through the evaluator. There is no alternative
	// path, in any mode, for any tool.
	verdict := e.evaluate(ex.Declare)
	if verdict.Decision == policy.DecisionEscalate {
		d, err := e.askApproval(ctx, turnID, ex, verdict)
		if err != nil || d == protocol.ApprovalDeny {
			return fail(fmt.Sprintf("denied: %s", verdict.Reason)), true, ""
		}
	}
	if verdict.Decision == policy.DecisionDeny {
		return fail(fmt.Sprintf("not permitted: %s", verdict.Reason)), false, ""
	}

	// Measured around Execute only: the wait for an approval is the user's
	// time, not the tool's, and folding it in would make every gated call look
	// slow.
	started := e.now()
	res, err := tool.Execute(ctx, ex.Call.Input, e.cfg.State)
	finished := e.now()
	elapsed := finished.Sub(started)
	// The real order, kept on the execution rather than only folded into a
	// duration: which of two concurrent calls actually went first is a fact a
	// duration cannot answer.
	if err != nil {
		return fail(err.Error()), false, ""
	}

	e.emit(protocol.EventToolCompleted, protocol.ToolCompleted{
		ToolCallID: ex.Call.ID, OK: !res.IsError,
		Output: res.Output, Truncated: res.Truncated,
		Lines: res.Meta.Lines, Files: res.Meta.Files,
		Added: res.Meta.Added, Removed: res.Meta.Removed,
		ExitCode: res.Meta.ExitCode, HasExit: res.Meta.HasExit,
		DurationMS: int(elapsed.Milliseconds()),
		StartedAt:  started,
		FinishedAt: finished,
		Diff:       res.Meta.Diff,
	})
	if ex.Call.Name == "plan" && !res.IsError {
		e.emit(protocol.EventPlanUpdated, protocol.PlanUpdated{Items: e.cfg.State.Plan()})
	}

	return ce.Message{Role: ce.RoleTool, ToolResult: &ce.ToolResult{
		ToolCallID: ex.Call.ID, Output: res.Output,
		IsError: res.IsError, Truncated: res.Truncated,
	}}, false, wallKey(ex, res)
}

// wallKey identifies a failure precisely enough that hitting it twice means
// something.
//
// Tool, code and path together. "read failed" twice on two different files is
// two ordinary misses; twice on the SAME file, the same way, is the repository
// saying something nobody wrote down. Without the path it would fire on any two
// unrelated errors, which is how a reminder becomes noise and gets ignored.
func wallKey(ex ToolExecution, res tools.Result) string {
	if !res.IsError {
		return ""
	}
	var path string
	for _, a := range ex.Declare.Paths {
		if a.Path != "" {
			path = a.Path
			break
		}
	}
	return ex.Call.Name + "|" + res.Code + "|" + path
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

func (e *Engine) askApproval(ctx context.Context, turnID string, ex ToolExecution, v policy.Verdict) (
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
	if c := e.completion(); c != nil {
		ev.Completion = c
	}
	e.emit(protocol.EventTurnCompleted, ev)
	// The single exit of every turn, which is why the hook lives here rather
	// than at the call sites: an ending added later gets it for free, and one
	// that forgot it would be the only turn whose writes nobody heard about.
	if e.cfg.AfterTurn != nil {
		e.cfg.AfterTurn(e.written())
	}
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

// crossBudget reports the occupancy band to announce this iteration, or
// ce.BandNone when there is nothing new to say.
//
// Edge-triggered, and the edge is what makes it affordable: the fraction is
// recomputed every iteration anyway, and announcing on level would repeat the
// same reminder for the rest of the session.
//
// It also rearms. Compaction drops the fraction, so the stored band falls with
// it, and climbing back up is announced again — which is right, because after a
// cut the climb genuinely is new information.
func (e *Engine) crossBudget() ce.Band {
	if !e.cfg.BudgetNotice {
		return ce.BandNone
	}
	cfg := e.cfg.CtxConfig
	if cfg.Window <= 0 && e.cfg.Provider != nil {
		if w, err := e.cfg.Provider.Window(e.cfg.Model); err == nil {
			cfg.Window = w
		}
	}
	band, announce := ce.Crossed(e.budgetBand, ce.Fraction(e.session, cfg), cfg.CompactAt)
	e.budgetBand = band
	if !announce {
		return ce.BandNone
	}
	return band
}

// checkDone is step 4: decide whether the turn may end.
//
// It returns the stop reason when the turn ends, or the messages to append and
// nil when it must continue. Re-entry costs one iteration, which is the spend
// that was authorised, and it buys the difference between a false report and an
// honest one.
//
// Exit is by PROGRESS, never by perfection. If the unmet set shrank strictly,
// the loop goes round again. If it did not shrink MaxStallCycles times, the
// turn ends in StopIncomplete carrying what was left. A loop that cannot exit
// until everything passes makes weakening the check the shortest way out.
func (e *Engine) checkDone(ctx context.Context, stall *int, unmet *[]string, told *bool) (string, []ce.Message) {
	if !e.cfg.DoneEnabled || len(e.cfg.Done.Criteria) == 0 {
		return protocol.StopDone, nil
	}
	// A turn that changed nothing has nothing to verify. Running the suite to
	// answer "what does this function do" burns a turn, and two weeks of that
	// is an uninstalled tool.
	written := e.written()
	if len(written) == 0 {
		e.lastReport = Report{}
		return protocol.StopDone, nil
	}

	// Stamped BEFORE the check runs, so anything written while it runs or after
	// it finishes postdates it. Stamping afterwards would silently absorb every
	// edit made during a long suite.
	e.checkedAt = e.writeSeq()
	rep := Check(ctx, e.cfg.Done, e.cfg.RunCriterion, e.cfg.DoneTimeout)
	rep.TouchedProtected = e.cfg.Done.TouchedProtected(written)
	e.lastReport = rep

	now := rep.Unmet()
	if len(now) == 0 {
		// Nothing is unmet, but that is not the same as everything having been
		// checked. Files changed and no criterion could run, so the turn
		// delivered work nobody confirmed — and ending it as StopDone is
		// precisely the false claim RN-9 exists to prevent.
		//
		// It does not force another iteration. There is nothing to run, and
		// insisting only produces a second guess; what it forces is saying so.
		if VerificationOf(rep, true, false) == VerificationUnavailable {
			// Once, and then the turn ends. The re-entry is not to run the
			// check again — there is nothing to run, which is the whole
			// situation — it is to give the model the one thing it still can
			// produce: a sentence saying the work was not confirmed.
			//
			// Without it the seal told the user and nothing told the model,
			// while this function's own comment claimed "what it forces is
			// saying so". Nothing forced anything.
			if !*told {
				*told = true
				return "", remindersFor(behavior.SessionState{VerificationUnavailable: true})
			}
			return protocol.StopUnverified, nil
		}
		return protocol.StopDone, nil
	}

	if Progressed(*unmet, now) || *unmet == nil {
		*stall = 0
	} else {
		*stall++
	}
	*unmet = now

	if *stall >= e.stallLimit() {
		// Not an error. This is the product being honest about work that needs
		// a person, and reporting it as failure would make switching the check
		// off the easy way out.
		return protocol.StopIncomplete, nil
	}

	return "", remindersFor(behavior.SessionState{
		UnmetCriteria:    now,
		ProtectedTouched: rep.TouchedProtected,
	})
}

// remindersFor renders a state as messages the model reads next turn.
func remindersFor(st behavior.SessionState) []ce.Message {
	var msgs []ce.Message
	for _, r := range behavior.Emit(st) {
		msgs = append(msgs, ce.Message{
			Role: ce.RoleUser, Text: behavior.Render(r), Reminder: true,
		})
	}
	return msgs
}

func (e *Engine) stallLimit() int {
	if e.cfg.MaxStallCycles > 0 {
		return e.cfg.MaxStallCycles
	}
	return 2
}

// writeSeq is the session's write generation, or zero when nothing counts.
func (e *Engine) writeSeq() uint64 {
	if e.cfg.WriteSeq == nil {
		return 0
	}
	return e.cfg.WriteSeq()
}

// stale reports that a write landed after the last check was stamped.
//
// Unknowable without a counter, and then the answer is false: an absent signal
// must not become a positive one, and "not stale" is the claim that takes
// nothing away from what the criteria already said.
func (e *Engine) stale() bool {
	return e.cfg.WriteSeq != nil && e.cfg.WriteSeq() > e.checkedAt
}

func (e *Engine) written() []string {
	if e.cfg.WrittenPaths == nil {
		return nil
	}
	return e.cfg.WrittenPaths()
}

// Report is what the most recent done check found.
func (e *Engine) Report() Report { return e.lastReport }

// Verification is the single-criterion seal for the client.
func (e *Engine) Verification() Verification {
	return VerificationOf(e.lastReport, len(e.written()) > 0, e.stale())
}

// completion is what the client shows instead of trusting the prose.
//
// Nil when there was no definition of done, which is a different fact from
// "checked and everything passed" and has to read differently on screen.
func (e *Engine) completion() *protocol.Completion {
	if len(e.lastReport.States) == 0 {
		return nil
	}
	return &protocol.Completion{
		Verification:     string(e.Verification()),
		Met:              e.lastReport.Names(CriterionMet),
		Unmet:            e.lastReport.Names(CriterionUnmet),
		Unavailable:      e.lastReport.Names(CriterionUnavailable),
		TouchedProtected: e.lastReport.TouchedProtected,
	}
}

// finishInterrupted ends a turn the user stopped, without letting the history
// lie about the disk.
//
// RN-5 is explicit that a tool already under way when the interruption arrives
// either finishes or is cancelled, but that its result is appended anyway if it
// produced an effect — a history that lies about what happened on disk is worse
// than an incomplete one.
//
// The tools that write do finish: they take the context and ignore it, because
// a half-applied edit is worse than a slow one. `bash` is the case that does
// not, and a cancelled command can leave the disk changed while its result says
// it failed. So the turn records what the SESSION wrote, which is a fact
// neither the tool result nor the model can contradict.
func (e *Engine) finishInterrupted(out Outcome) Outcome {
	if written := e.written(); len(written) > 0 && e.cfg.Reminders {
		e.session.History = append(e.session.History, ce.Message{
			Role:     ce.RoleUser,
			Reminder: true,
			Text: behavior.Render(behavior.Reminder{
				Kind: behavior.ReminderInterrupted,
				Text: "The turn was interrupted. These files were changed before " +
					"it stopped: " + strings.Join(written, ", ") +
					". Some of that work may be half-done. Check the state of " +
					"those files before continuing, and do not assume the last " +
					"thing you tried either succeeded or failed.",
			}),
		})
	}
	return e.finish(out, protocol.StopInterrupted)
}

// sleep waits, and reports whether the wait completed.
//
// Interruptible, because RN-5 says every waiting point is one: a user who asked
// to stop should not have to sit through a backoff they cannot see.
func (e *Engine) sleep(ctx context.Context, d time.Duration) bool {
	if e.cfg.Sleep != nil {
		return e.cfg.Sleep(ctx, d)
	}
	if d <= 0 {
		return true
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// Close releases what the engine owns beyond a single turn.
//
// Today that is the background processes started through bash. The chain is
// deliberate — session owns engine owns tool state owns processes — because it
// makes "a process dies with the session" a consequence of ownership rather
// than a cleanup step someone has to remember to write. There is no handler to
// forget to register, and no path where the session ends and the process does
// not.
func (e *Engine) Close() {
	if e.cfg.State != nil {
		e.cfg.State.Close()
	}
}

// unplannedFiles is how far work spreads before the absence of a plan is worth
// saying out loud.
//
// Three, while the tool description asks for a plan at more than one file. The
// gap is deliberate: the description says when to plan, and a reminder that
// fired at the first missed opportunity would be nagging. This is the backstop
// for work that has visibly become a project, not a second copy of the rule.
const unplannedFiles = 3

// unplannedChange reports whether work has spread across several files with no
// plan recorded, and that this has not been said yet.
//
// It answers plan-depth-complex, which measures 60%: the model reads the six
// files, edits every one of them correctly, and records no plan — it narrates
// one instead, into a paragraph nobody watching can follow. The tool
// description has asked for the opposite since #107 without landing, and a
// fourth sentence in the same place would be the third time that failed. A
// reminder is the other layer: zero cost until the situation exists, and
// delivered at the moment it is being ignored rather than at the top of the
// session.
func (e *Engine) unplannedChange() bool {
	if e.cfg.State == nil {
		return false
	}
	if len(e.cfg.State.Plan()) > 0 {
		e.toldUnplanned = false
		return false
	}
	if e.toldUnplanned || len(e.cfg.State.Written()) < unplannedFiles {
		return false
	}
	e.toldUnplanned = true
	return true
}

// Undo puts back what the last turn changed.
//
// Not a tool: the model does not get to undo its own work, because the judgment
// undo exists for is the person's. It is an operation on the session, asked for
// through the client, and the loop only owns it because the loop owns the state
// that knows what changed.
func (e *Engine) Undo() (restored, refused []string, err error) {
	if e.cfg.State == nil {
		return nil, nil, nil
	}
	return e.cfg.State.Undo()
}

// deliverSteering puts what the person said mid-turn into the conversation.
//
// As the person, never as a reminder. A reminder is the product talking, and
// filing a correction as one misattributes the most important thing anybody
// says during a turn — the model would weigh it as machinery rather than as the
// user changing their mind.
//
// Drained until empty, in order: two corrections reordered is one instruction
// answering the wrong question.
func (e *Engine) deliverSteering(turnID string) {
	if e.cfg.Steer == nil {
		return
	}
	for {
		text := e.cfg.Steer()
		if strings.TrimSpace(text) == "" {
			return
		}
		e.session.History = append(e.session.History,
			ce.Message{Role: ce.RoleUser, Text: text})
		e.emit(protocol.EventTurnSteered, protocol.TurnSteered{TurnID: turnID, Text: text})
	}
}

// hitTheSameWallTwice reports that this turn has failed the same way twice on
// the same path, and that nobody has been told yet.
//
// It exists because measurement said the prompt was not enough: across four
// scenario designs the model never once called `remember` on its own. A fifth
// sentence in the doctrine would be the third time that approach failed. A
// reminder is the other layer — nothing until the situation exists, delivered
// at the moment it is being missed.
//
// Latched per turn. Repeating it every round would be nagging about a fact that
// has not changed, and a reminder that repeats is one the model learns to skip.
func (e *Engine) hitTheSameWallTwice(f batchFacts) bool {
	if e.cfg.Tools == nil {
		return false
	}
	if _, ok := e.cfg.Tools.Get("remember"); !ok {
		// Nothing to ask for. Telling the model to call a tool this build does
		// not carry sends it somewhere that does not exist — the defect this
		// codebase already fixed once in a tool error message.
		return false
	}
	if e.walls == nil {
		e.walls = map[string]int{}
	}
	hit := false
	for _, w := range f.Walls {
		e.walls[w]++
		if e.walls[w] >= 2 {
			hit = true
		}
	}
	if !hit || e.toldWorthRemembering {
		return false
	}
	e.toldWorthRemembering = true
	return true
}
