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
	"em arquivo não lido":              "TestEditRefusesAFileNotReadInThisSession",
	"após modificação externa":         "TestEditRefusesAFileChangedSinceItWasRead",
	"aparecendo duas vezes":            "TestEditRefusesAnAmbiguousMatchAndChangesNothing",
	"segunda edição seguida":           "TestTwoEditsInARowNeedNoReread",
	"sobre arquivo existente não":      "TestWriteOverExistingFileRequiresARead",
	"Escrita é atômica":                "TestAtomicWriteLeavesNoTemporaryBehind",
	"a mesma ordem em execuções":       "TestGlobIsSortedAndStable",
	"sem qualquer efeito":              "TestDeclareTouchesNothing",
	"Nenhuma saída contém":             "TestNoOutputCarriesAClockOrAnAbsolutePath",
	"em diretório diz que é diretório": "TestReadingADirectorySaysToUseGlob",
	"só nomeia ferramenta que":         "TestAToolErrorNamesGlobOnlyWhenTheSessionHasIt",
	"Truncamento sempre":               "TestTruncationIsDeclared",
	"saída diferente de zero":          "TestBashNonZeroExitIsAResultNotAnError",
	"sem veredito favorável":           "TestEveryToolPassesThroughPolicy",
	"dois itens `active":               "TestRejectedPlanLeavesThePreviousOneIntact",
	"`blocked` sem motivo":             "TestPlanAcceptsBlockedWithAReason",
	"não reporta caminho nem":          "TestPlanNeverCrossesABoundary",

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

	// RN-12, a command that does not end. Three of these are asserted in
	// internal/sandbox, which is why the search covers that directory too: the
	// lifetime of a process is a property of the boundary that started it, not
	// of the tool that asked.
	"enquanto o comando ainda roda":     "TestABackgroundCommandReturnsWhileItIsStillRunning",
	"morre na janela de assentamento":   "TestABackgroundCommandThatDiesDuringStartupSaysSo",
	"identificador é sequência":         "TestAProcessIdentifierCarriesNoClock",
	"sem identificador lista":           "TestProcessWithNoIdentifierListsWhatIsRunning",
	"identificador desconhecido":        "TestProcessNamesTheIdentifiersItKnowsWhenGivenAnUnknownOne",
	"process.Declare":                   "TestProcessCrossesNoBoundaryAndSoNeverAsks",
	"chamada de primeiro plano declara": "TestBackgroundDeclaresExactlyWhatAForegroundCommandDeclares",
	"sem executor capaz":                "TestBackgroundIsRefusedWhenNothingCanRunIt",
	"encerra **todo** processo":         "TestClosingTheStateStopsEveryProcess",
	"alcança o **grupo**":               "TestStopKillsWhatTheCommandStartedToo",
	"sobrevive ao turno":                "TestACommandOutlivesTheTurnThatStartedIt",
	"mantém o **fim**":                  "TestProcessKeepsTheTailWhenTheOutputIsTooLong",
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
	// The sandbox directory is searched too. A spec family is not a Go package,
	// and the invariants about how long a process lives are asserted where the
	// process is actually started — excusing them here would be the escape
	// hatch this guard exists to close.
	findings, err := specguard.Check(root, family, []string{".", "../sandbox"}, mapping)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Errorf("%s: %s", family, f)
	}
}
