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
			var failed Transcript
			var sawFailure bool
			attempt := func(ctx context.Context) (bool, error) {
				w, err := NewWorkspace(t.TempDir(), f.Files)
				if err != nil {
					return false, err
				}
				tr, err := exchangeRounds(ctx, p, cfg.Model, f, contract, w)
				if err != nil {
					return false, err
				}
				ok := contract.Judge(tr)
				if !ok && !sawFailure {
					failed, sawFailure = tr, true
				}
				return ok, nil
			}

			ctx, cancel := context.WithTimeout(context.Background(),
				runTimeout*time.Duration(cfg.Runs*contract.Rounds))
			defer cancel()

			r := Measure(ctx, cfg, contract.ID, contract.Threshold, attempt)
			measured = append(measured, r)
			if !r.Met() && sawFailure {
				t.Log("one failing run — " + failed.Digest())
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

	rounds := c.Rounds
	if rounds < 1 {
		rounds = 1
	}
	for i := 0; i < rounds; i++ {
		msgs, err := f.Messages(p.Family().Name(), history)
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

		if i == rounds-1 {
			break
		}
		// A round with no call and nothing to say back is the model finished.
		// Asking again would spend a call on a turn the product would never
		// have taken.
		injectNow := i == 0 && c.Inject != ""
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
