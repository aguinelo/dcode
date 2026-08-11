package contextengine

import (
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/specguard"
)

// The band invariants are asserted here and the loop asserts the same edges
// through Engine.crossBudget; the pure half is what this package owns.
var contextDirs = []string{".", filepath.Join("..", "loop")}

var contextInvariants = map[string]string{
	"slices byte-a-byte idênticos":     "TestAssembleIsPure",
	"não altera nenhum byte":           "TestPrefixIsStableUnderAppend",
	"dígito de timestamp":              "TestNoVolatileDataInOutput",
	"Nada altera `Session.Tools`":      "TestNothingAssignsToASessionsToolsAfterItIsBuilt",
	"A ordem das seções é sempre":      "TestAssembleSectionOrder",
	"sem qualquer marcador de resumo":  "TestSummaryAbsentEmitsNoMarker",
	"separe `RoleAssistant`":           "TestPlanNeverSplitsAToolCallFromItsResults",
	"nunca inclui a última `RoleUser`": "TestPlanNeverCompactsTheCurrentTask",
	"`Estimate` é determinística":      "TestEstimateIsDeterministic",
	"são determinísticas para a mesma": "TestFractionIsPureAndDoesNotDriftBetweenCalls",
	"é monotônica":                     "TestBandForMapsWindowFractionsToBands",
	"O limiar mais alto de `Band`":     "TestEveryBandLandsBelowTheCompactionTrigger",
	"contém a fração nem a faixa":      "TestNoFractionOrBandReachesTheAssembledOutput",
	"emite **uma vez**":                "TestAnnouncementIsEdgeTriggeredNotLevelTriggered",
	"rearma a faixa anunciada":         "TestFallingBackDownAnnouncesNothingButRearms",
	"não realiza I/O":                  "TestPackageImportsNothingImpure",
}

func TestEveryInvariantHasATest(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	findings, err := specguard.Check(root, "context-engine", contextDirs, contextInvariants)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Errorf("context-engine: %s", f)
	}
}
