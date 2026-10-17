package orchestrate

import (
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/specguard"
)

// The orchestration family's verifiable invariants, .p §13. Only the lines
// that have a test are there; the rest wait in §14 and move up, each with its
// claim here, in the step that gives it one.
var orchestrationInvariants = map[string]string{
	"contínuo por projeto, através de reinícios": "TestReopeningTheLogContinuesTheSeqWithoutAGap",
	"é recusada e nada é escrito":                "TestADecisionWithoutAReasonIsRefusedAndNothingIsWritten",
	"nunca uma linha perdida em silêncio":        "TestAWriteThatFailsIsAnErrorAndNeverALineLostInSilence",
	"devolvida por `Torn`":                       "TestATornLastLineIsCutOnReopenAndSaid",
	"Nenhuma linha do log passa de 64 KiB":       "TestALineOver64KiBCutsTheLargestFieldAndSaysHowMuch",
}

func TestEveryInvariantHasATest(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	findings, err := specguard.Check(root, "orchestration", []string{"."}, orchestrationInvariants)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Errorf("orchestration: %s", f)
	}
}
