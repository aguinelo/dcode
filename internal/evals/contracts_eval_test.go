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

			attempt := func(ctx context.Context) (bool, error) {
				tr, err := exchangeRounds(ctx, p, cfg.Model, f, contract)
				if err != nil {
					return false, err
				}
				return contract.Judge(tr), nil
			}

			ctx, cancel := context.WithTimeout(context.Background(),
				runTimeout*time.Duration(cfg.Runs*contract.Rounds))
			defer cancel()

			r := Measure(ctx, cfg, contract.ID, contract.Threshold, attempt)
			measured = append(measured, r)
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
func exchangeRounds(ctx context.Context, p provider.Provider, model string, f Fixture, c Contract) (Transcript, error) {
	msgs := []ce.Message{{Role: ce.RoleUser, Text: f.Task}}
	var tr Transcript

	rounds := c.Rounds
	if rounds < 1 {
		rounds = 1
	}
	for i := 0; i < rounds; i++ {
		calls, text, err := exchange(ctx, p, model, msgs, f.Tools)
		if err != nil {
			return Transcript{}, err
		}
		tr.Calls = append(tr.Calls, calls...)
		tr.Text += text + "\n"
		tr.Rounds++

		if i == rounds-1 || c.Inject == "" {
			break
		}
		// The model's turn, then what the product would have said back.
		msgs = append(msgs, ce.Message{Role: ce.RoleAssistant, Text: text, ToolCalls: calls})
		if len(calls) > 0 {
			msgs = append(msgs, ce.Message{Role: ce.RoleTool, ToolResult: &ce.ToolResult{
				ToolCallID: calls[0].ID, Output: c.Inject, IsError: true,
			}})
		} else {
			msgs = append(msgs, ce.Message{Role: ce.RoleUser, Text: c.Inject, Reminder: true})
		}
	}
	return tr, nil
}
