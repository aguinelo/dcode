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
	"fmt"
	"os"
	"testing"
	"time"

	ce "github.com/aguinelo/dcode/internal/contextengine"
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

			// The first failing run is kept so the report can show what the
			// model actually did. A rate on its own cannot be acted on: 0% of
			// 20 reads as "the model gets this wrong" and is just as often
			// "the scenario cannot reach the behaviour it judges", and those
			// two need opposite fixes.
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

	rounds := c.Rounds
	if rounds < 1 {
		rounds = 1
	}
	for i := 0; i < rounds; i++ {
		msgs, err := f.Messages(p.Family().Name(), w.Dir, history)
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
			break
		}
		// The model's turn, then what the product would have said back.
		history = append(history, ce.Message{Role: ce.RoleAssistant, Text: text, ToolCalls: calls})
		history = append(history, answers(ctx, w, c, calls, injectNow)...)
		if injectNow {
			tr.InjectedAt = len(tr.Calls)
		}
	}
	return tr, nil
}
