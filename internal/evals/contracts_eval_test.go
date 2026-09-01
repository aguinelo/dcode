//go:build eval

// Every declared behavioural contract, measured.
//
// Thirty of these had material and no runner: the fixture test checked that
// they load, and nothing ever ran them against a model or compared the result
// to a threshold. A threshold nobody runs is not a weak measurement, it is a
// claim.
package evals

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aguinelo/dcode/internal/app"
	ce "github.com/aguinelo/dcode/internal/contextengine"
	"github.com/aguinelo/dcode/internal/loop"
	"github.com/aguinelo/dcode/internal/provider"
)

// measured is what TestEveryContract managed to measure, read by TestMain.
//
// A package variable because the count has to survive the test that produced
// it: when the suite skips, the test body never runs, and the skip is exactly
// the case that needs saying out loud.
var measured []Result

// TestMain prints how much of the declared set was measured, last, always.
//
// `go test` ends in PASS whether this suite measured thirty-five contracts or
// none — a skip is a pass. So the exit status is left alone and the count is
// printed after it, because the count is the sentence a person needs.
func TestMain(m *testing.M) {
	code := m.Run()
	fmt.Println(Summary(Measurable(Contracts), len(Contracts)-Measurable(Contracts), measured))
	os.Exit(code)
}

func TestEveryContract(t *testing.T) {
	p, cfg := setup(t)

	for _, contract := range Contracts {
		contract := contract
		if !contract.Measured() {
			continue // established by assertion; running it would print a free green
		}
		t.Run(contract.ID, func(t *testing.T) {
			f, err := LoadFixture(FixtureRoot, contract.ID)
			if err != nil {
				t.Fatal(err)
			}

			// Up to MaxEvidence distinct failing transcripts are kept, so the
			// report can show what the model actually did. A rate on its own
			// cannot be acted on: 0% of 20 reads as "the model gets this
			// wrong" and is just as often "the scenario cannot reach the
			// behaviour it judges", and those two need opposite fixes.
			//
			// Distinct, because twenty runs failing the same way are one
			// finding — spending the cap on copies is how the second cause
			// stays invisible.
			evidence := NewEvidence(MaxEvidence)
			retries := 0
			// A transport error is not behaviour. Two full runs came back
			// unsound because DNS failed partway through, after hours of paid
			// measurement — a lost packet must not cost that. It does not
			// soften the rule that a failure to measure is unsound; it
			// separates a blip from a provider that is not answering.
			once := func(ctx context.Context) (bool, error) {
				w, err := NewWorkspace(t.TempDir(), f.Files, f.ToolNames())
				if err != nil {
					return false, err
				}
				// The harness can run a child turn, so it runs one. Refusing
				// used to talk the model out of delegating for the rest of the
				// run, which made a contract about reaching for delegation
				// measure the harness talking instead.
				w.Delegate = childTurn(p, cfg.Model, f, w)
				// A qualifying turn gets the product's own half of itself: the
				// recorder that answers done_propose, and the boundary the
				// product forces on such a turn. Neither is written here — a
				// harness that described the qualifying turn in its own words
				// would measure a turn the product does not have.
				if f.World.Qualifying() {
					w.Qualify = qualifying(f.World.Qualify)
					w.Mode = app.QualifyMode(app.Options{}, true).SandboxMode
				}
				// The deadline belongs to the run, not to the contract. It
				// used to cover all of them at once, which meant one hung
				// stream ate the whole budget and every run after it failed
				// instantly — the harness reporting its own exhaustion as the
				// model's behaviour.
				runCtx, done := context.WithTimeout(ctx, runTimeout*time.Duration(contract.Rounds))
				defer done()

				tr, err := exchangeRounds(runCtx, p, cfg.Model, f, contract, w)
				if err != nil {
					return false, err
				}
				ok := contract.Judge(tr)
				if !ok {
					evidence.Record(tr.Digest(), tr.HitCeiling)
				}
				return ok, nil
			}

			attempt := func(ctx context.Context) (bool, error) {
				ok, again, err := WithRetry(ctx, TransportRetries, once)
				retries += again
				return ok, err
			}

			// A contract that claims more measures more. Resolved here rather
			// than inside Measure so the count that ran is the count that
			// gets printed, with no second place deciding it.
			runCfg := cfg
			runCfg.Runs = contract.RunCount(cfg.Runs)

			// No contract-wide deadline. Each run carries its own, and the
			// suite's ceiling is `go test -timeout`, which is the one place a
			// total budget belongs. A second ceiling here only decided which
			// contract got blamed for the suite running long.
			r := Measure(context.Background(), runCfg, contract.ID, contract.Threshold, attempt)
			r.Retries = retries
			// The prefix this ran against, printed with the number so that
			// recording the measurement is copying a line rather than a second
			// manual step nobody would do twice.
			if fp, ferr := f.Fingerprint(); ferr == nil {
				r.Prompt = fp
			}
			measured = append(measured, r)
			// Evidence on failure, and evidence whenever the contract is
			// measuring rather than judging.
			//
			// A threshold of zero means "measure and tell me", and a number
			// with no transcript behind it is half of what was asked for: the
			// first run of these came back 0%, 100% and 5% with nothing to say
			// whether that was the product or the judge — and this suite has
			// found the judge four times.
			if !r.Met() || contract.Threshold == 0 {
				if s := evidence.String(); s != "" {
					t.Log(s)
				}
			}
			report(t, r)
		})
	}

	// Worst first, so the reason to be reading this output at all is the first
	// thing on screen.
	t.Log("\n" + Report(measured))
}

// exchangeRounds runs one scenario to completion and collects what it did.
//
// Between rounds it appends what the contract injects, standing in for what the
// product would have appended — a tool error, a reminder. The wording of those
// is the product's own, because RN-3 makes tool error text a behaviour surface
// and measuring against text dcode does not emit measures a different product.
func exchangeRounds(ctx context.Context, p provider.Provider, model string, f Fixture, c Contract, w *Workspace) (Transcript, error) {
	// History, not the whole message list. The prompt is rebuilt around it on
	// every round by the same code path the product uses — which is the fix:
	// this used to be the entire list, hand-built as one user message, so the
	// model never saw the doctrine, the project instructions or the skill
	// index that most of these contracts are about.
	history := f.Opening()
	var tr Transcript
	var injected bool

	// A scenario that declares criteria runs the product's verification cycle
	// instead of being handed the reminder one would have produced. Without
	// this every contract about what a turn does when a check fails was
	// measured against an injected sentence, and checkDone, Moved and the
	// rollback never ran at all.
	cycle := NewCycle(f.Criteria, w)
	w.BeginTurn()

	rounds := c.Rounds
	if rounds < 1 {
		rounds = 1
	}
	for i := 0; i < rounds; i++ {
		msgs, err := f.Messages(ctx, p.Family().Name(), w.Dir, history)
		if err != nil {
			return Transcript{}, err
		}
		calls, text, err := exchange(ctx, p, model, msgs, f.Tools)
		if err != nil {
			return Transcript{}, err
		}
		tr.Calls = append(tr.Calls, calls...)
		tr.Text += text + "\n"
		tr.Rounds++

		if CeilingReached(i, rounds, len(calls)) {
			tr.HitCeiling = true
		}
		if i == rounds-1 {
			break
		}
		// A round with no call and nothing to say back is the model finished.
		// Asking again would spend a call on a turn the product would never
		// have taken. An injection still pending is not "nothing to say":
		// with no call to attach it to it arrives as a reminder.
		// The injection waits for the call it belongs to. A tool error that
		// lands on whatever the model happened to do first tells it something
		// impossible about that action, and the rounds after go on trying to
		// reconcile it.
		//
		// A round with no calls is the last chance to say it: `answers`
		// delivers it as a reminder there, because the alternative is a
		// scenario whose whole premise never reaches the model.
		injectNow := !injected && c.Inject != "" && c.InjectableAt(i) &&
			(InjectionTarget(c, calls) >= 0 || len(calls) == 0)
		if injectNow {
			injected = true
		}
		if len(calls) == 0 && !injectNow {
			// The turn does not end merely because the model stopped asking:
			// done is a checked condition, which is the loop's own rule.
			more := cycle.After()
			if len(more) == 0 {
				break
			}
			history = append(history, ce.Message{Role: ce.RoleAssistant, Text: text})
			history = append(history, more...)
			cycle.Begin()
			continue
		}
		// The model's turn, then what the product would have said back.
		history = append(history, ce.Message{Role: ce.RoleAssistant, Text: text, ToolCalls: calls})
		history = append(history, answers(ctx, w, c, calls, injectNow)...)
		if injectNow {
			tr.InjectedAt = len(tr.Calls)
		}
	}
	tr.CriteriaMet = cycle.Met()
	return tr, nil
}

// childRounds is the ceiling on a delegated turn inside a scenario.
//
// Smaller than the parent's, for the product's own reason: a child does ONE
// piece of work, and one that needs many rounds is a piece that should have
// been split. Small here also bounds what measuring delegation costs — five
// children in fifty runs is two hundred and fifty child turns, and each round
// of each of them is a paid call.
const childRounds = 4

// childTurn answers a delegated call by running one, against the same provider
// the parent is talking to.
//
// The child gets the PRODUCT's instructions, imported rather than copied, for
// the reason BudgetText is exported: a harness that paraphrases measures the
// paraphrase. It gets a fresh history holding only the task — copying the
// parent's would return exactly the cost delegation exists to avoid — and the
// reading tools, plus the writing ones when it owns paths.
//
// What it does NOT reproduce is the product's containment: narrowing the
// child's resolver to `owns` lives in internal/policy and is asserted there,
// against the kernel and by unit test. What the contracts measure here is the
// PARENT's decision to divide the work, and that is visible in the call it
// emits before any child runs.
func childTurn(p provider.Provider, model string, f Fixture, w *Workspace) func(context.Context, string, string, []string) (string, bool) {
	return func(ctx context.Context, task, path string, owns []string) (string, bool) {
		names := []string{"read", "glob", "grep", "symbol"}
		if len(owns) > 0 {
			names = append(names, "write", "edit")
		}
		// The child's tool set is a subset of the scenario's, filtered by name.
		// Taken from the fixture rather than rebuilt, so a child is never
		// offered something the scenario does not carry.
		var defs []ce.ToolDef
		for _, d := range f.Tools {
			for _, want := range names {
				if d.Name == want {
					defs = append(defs, d)
					break
				}
			}
		}

		history := []ce.Message{{Role: ce.RoleUser, Text: delegateTask(task, path)}}
		var conclusion string

		for i := 0; i < childRounds; i++ {
			msgs := append([]ce.Message{{
				Role: ce.RoleSystem,
				Text: loop.DelegateInstructions(names, owns),
			}}, history...)

			calls, text, err := exchange(ctx, p, model, msgs, defs)
			if err != nil {
				// Named, never swallowed: the parent is told which child did
				// not answer, which is the third contract's whole subject.
				return fmt.Sprintf("the delegated turn failed: %v (task: %s)", err, task), true
			}
			if strings.TrimSpace(text) != "" {
				conclusion = strings.TrimSpace(text)
			}
			if len(calls) == 0 {
				break
			}
			history = append(history, ce.Message{Role: ce.RoleAssistant, Text: text, ToolCalls: calls})
			history = append(history, answers(ctx, w, Contract{}, calls, false)...)
		}

		if conclusion == "" {
			conclusion = "the delegated turn produced no answer."
		}
		// The paths travel with the conclusion, as the product's do: they do
		// not prove the child understood, they prove what it touched.
		if read := w.state.ReadPaths(); len(read) > 0 {
			conclusion += "\n\nlooked at: " + strings.Join(read, ", ")
		}
		return conclusion, false
	}
}

// delegateTask mirrors what the product hands a child: the task, and where to
// look when the parent said.
func delegateTask(task, path string) string {
	if strings.TrimSpace(path) == "" {
		return task
	}
	return task + "\n\nLook under: " + path
}

// qualifying is the product's own recorder, answering done_propose.
//
// A fresh Proposals per run: a slot shared between runs would let one run read
// what another proposed, and the judge reads the call rather than the slot
// anyway. What the product decides here is the ANSWER the model reads back —
// "recorded N criteria, they are not measured yet, you are done" — which is
// most of what stops a qualifying turn from going on to do the work.
func qualifying(spec string) func(context.Context, json.RawMessage) (string, bool) {
	return func(ctx context.Context, input json.RawMessage) (string, bool) {
		tool := app.QualifyingTool(spec, &app.Proposals{})
		res, err := tool.Execute(ctx, input, nil)
		if err != nil {
			return err.Error(), true
		}
		return res.Output, res.IsError
	}
}
