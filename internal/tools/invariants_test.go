package tools

import (
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/specguard"
)

// The mapping is a MAPPING rather than sixty duplicated tests. Most of these
// invariants already have an assertion somewhere and copying it here would give
// two places to keep in step; what was missing is the proof that each line
// points at one.
var toolInvariants = map[string]string{
	"em arquivo não lido":         "TestEditRefusesAFileNotReadInThisSession",
	"após modificação externa":    "TestEditRefusesAFileChangedSinceItWasRead",
	"aparecendo duas vezes":       "TestEditRefusesAnAmbiguousMatchAndChangesNothing",
	"segunda edição seguida":      "TestTwoEditsInARowNeedNoReread",
	"sobre arquivo existente não": "TestWriteOverExistingFileRequiresARead",
	"Escrita é atômica":           "TestAtomicWriteLeavesNoTemporaryBehind",
	"a mesma ordem em execuções":  "TestGlobIsSortedAndStable",
	"sem qualquer efeito":         "TestDeclareTouchesNothing",
	"Nenhuma saída contém":        "TestNoOutputCarriesAClockOrAnAbsolutePath",
	"Truncamento sempre":          "TestTruncationIsDeclared",
	"saída diferente de zero":     "TestBashNonZeroExitIsAResultNotAnError",
	"sem veredito favorável":      "TestEveryToolPassesThroughPolicy",
	"dois itens `active":          "TestRejectedPlanLeavesThePreviousOneIntact",
	"`blocked` sem motivo":        "TestPlanAcceptsBlockedWithAReason",
	"não reporta caminho nem":     "TestPlanNeverCrossesABoundary",

	// RN-9, the echoed diff.
	"afetando mais de uma ocorrência": "TestEditEchoesTheDiffToTheModelOnlyOnAMultiOccurrenceReplaceAll",
	"ocorrência única":                "TestEchoDiffReturnsTheDiffOnlyWhereTheModelCannotDeriveIt",
	"nunca devolve diff":              "TestWriteNeverEchoesADiffInAnyMode",
	"acima do teto declara":           "TestAnEchoedDiffThatIsTruncatedSaysSo",

	// RN-10, symbol.
	"com `Path` de arquivo buscam nele": "TestGrepSearchesTheFileItWasPointedAt",
	"escapa `Name`":                     "TestSymbolMatchesOnBoundaryNotOnLetters",
	"respeita fronteira":                "TestSymbolMatchesOnBoundaryNotOnLetters",
	"casa `func Parse(`":                "TestDeclarationPatternsPerLanguage",
	"Extensão sem padrão":               "TestUnknownExtensionAnswersAndSaysTheKindIsUnknown",
	"declaração de limite":              "TestEveryResultDeclaresItsOwnLimit",
	"estável entre execuções":           "TestOrderingIsStableBetweenRuns",

	// RN-11, delegation.
	"não** o histórico do pai": "TestExploreDeclaresOnlyARead",
	"lista de caminhos lidos":  "TestExploreReportsWhereItLookedAndWhatItCouldNotRead",
	"reporta leitura, nunca":   "TestExploreDeclaresOnlyARead",
}

func TestEveryInvariantHasATest(t *testing.T) {
	checkInvariants(t, "tool-suite", toolInvariants)
}

func checkInvariants(t *testing.T, family string, mapping map[string]string) {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	findings, err := specguard.Check(root, family, []string{"."}, mapping)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Errorf("%s: %s", family, f)
	}
}
