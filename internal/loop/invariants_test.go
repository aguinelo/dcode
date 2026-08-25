package loop

import (
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/specguard"
)

// The policy invariant is asserted in internal/tools, where the two halves of
// it live: every tool declares a request the evaluator can rule on, and the
// loop has exactly one place a tool runs. Re-stating it here would give two
// places to keep in step.
var loopDirs = []string{".", filepath.Join("..", "tools")}

var loopInvariants = map[string]string{
	"muda o **veredito** da próxima chamada":      "TestSetModeChangesTheVerdict",
	"vista por **todos** os leitores do par":      "TestSetModeUnderConcurrentReads",
	"anexados em ordem de `Index`":                "TestResultsAppendInEmissionOrderNotCompletionOrder",
	"nunca se sobrepõem no tempo":                 "TestConflictingPathsAreSeparated",
	"comando de sistema executa concorrentemente": "TestTwoSystemCommandsNeverOverlap",
	"sem passar pelo avaliador":                   "TestEveryToolPassesThroughPolicy",
	"Nenhum timestamp, contagem ou ID":            "TestConcurrentExecutionIsAnnouncedWithAConstantNote",
	"insensível à ordem de chaves":                "TestIsRepeatIgnoresKeyOrder",
	"mesma ferramenta com input diferente":        "TestIsRepeatDistinguishesDifferentInputs",
	"Interrupção em qualquer fase":                "TestInterruptEndsTheTurnCleanly",
	"efeito no disco anexa o resultado":           "TestAnInterruptedTurnRecordsWhatWasAlreadyWritten",
	"Compactação verificada exatamente uma vez":   "TestCompactionRunsOnceAndIsAnnounced",
	"resumo não incrementa o contador":            "TestSummarisingDoesNotSpendAnIteration",
	"termina em `StopMaxIterations`":              "TestIterationCapIsTheBackstop",

	// Definition of done.
	"usa a reentrada da RN-10":         "TestNoProgressEndsTheTurnAsIncomplete",
	"não percorrem nenhum caminho":     "TestAnUnfinishedTurnIsAStateAndNotAnError",
	"encolher estritamente":            "TestProgressedRequiresTheUnmetSetToShrinkStrictly",
	"encerra em `StopIncomplete`":      "TestNoProgressEndsTheTurnAsIncomplete",
	"anexa o lembrete **uma vez**":     "TestChangedWithNothingAbleToCheckEndsUnverified",
	"não provoca reentrada":            "TestChangedWithNothingAbleToCheckEndsUnverified",
	"Mudança em caminho de `Protected": "TestWritingATestFileIsSurfaced",

	// Delegation.
	"construído em `ModeReadOnly`":  "TestTheChildIsReadOnlyEvenWithAWritingToolInReach",
	"**não contém** `explore`":      "TestTheChildCannotDelegateAgain",
	"nunca chega ao aprovador":      "TestTheChildIsReadOnlyEvenWithAWritingToolInReach",
	"aparece no relatório do filho": "TestTheResultCarriesWhereItLookedAndWhatItCouldNotRead",
	"debitados do orçamento do pai": "TestChildTokensAreDebitedFromTheParent",
	"acima do teto declara":         "TestALongReportIsCutAndSaysSo",
}

func TestEveryInvariantHasATest(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	findings, err := specguard.Check(root, "agent-loop", loopDirs, loopInvariants)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Errorf("agent-loop: %s", f)
	}
}

// The delegated-writing family spans three packages, and the split is the same
// one as everywhere else here: the loop decides what a child turn is, the
// policy package answers containment, and the tools package is where the
// declaration is made.
var delegatedWritingDirs = []string{
	".",
	filepath.Join("..", "policy"),
	filepath.Join("..", "tools"),
}

var delegatedWritingInvariants = map[string]string{
	"ausente produz um filho somente-leitura":      "TestAChildWithoutOwnsIsStillReadOnly",
	"não produz filho que escreve":                 "TestAWritingChildIsRefusedWhenTheParentCannotWrite",
	"vazio é erro de declaração":                   "TestExploreWithAnEmptyOwnsIsADeclarationError",
	"declaram conflito antes":                      "TestTwoChildrenOwningTheSamePathDeclareAConflict",
	"fora do que possui é negado pela contenção":   "TestANarrowedResolverRefusesAWriteOutsideWhatIsOwned",
	"por componente de caminho, nunca por prefixo": "TestOwnershipIsByComponentNotByPrefix",
	"não o traz para dentro":                       "TestOwningNeverReachesOutsideTheWorkspace",
	"Estreitar um filho não estreita o pai":        "TestOwningLeavesTheParentResolverAlone",
	"não carrega ferramenta opaca":                 "TestAWritingChildCarriesWritingToolsAndNothingOpaque",
	"conjunto de desfazimento do turno do pai":     "TestTheParentCanUndoWhatItsChildWrote",
	"fotografia que o pai tirou primeiro":          "TestAdoptKeepsTheSnapshotTheParentTookFirst",
	"nomeia os caminhos que escreveu":              "TestAWritingChildReportsWhatItWrote",
	"nomeado, nunca resumido":                      "TestAChildThatDidNotAnswerIsNamed",
	"recusada é reportada como escrita":            "TestARefusedWriteIsReportedAsAWrite",
	"nenhum pedido do modelo o amplia":             "TestAChildDoesNotWidenTheSessionsConcurrency",
	"escreve são debitados do pai":                 "TestAWritingChildIsPaidForByTheParent",
	"não carrega critério de pronto":               "TestAChildCarriesNoDefinitionOfDone",
	"não nega a capacidade que o schema oferece":   "TestTheDelegationToolDoesNotDenyWhatItOffers",
}

func TestEveryDelegatedWritingInvariantHasATest(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	findings, err := specguard.Check(root, "delegated-writing", delegatedWritingDirs, delegatedWritingInvariants)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Errorf("delegated-writing: %s", f)
	}
}
