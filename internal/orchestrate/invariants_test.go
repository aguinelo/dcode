package orchestrate

import (
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/specguard"
)

// The orchestration family's verifiable invariants, .p §13. Only the lines
// that have a test are there; the rest wait in §14 and move up, each with its
// claim here, in the step that gives it one. `dcode decisions` is the log's
// reader, and its tests live with the command; the daemon opens projects, and
// the tests of one orchestrator per project live with it.
var orchestrationInvariants = map[string]string{
	"contínuo por projeto, através de reinícios":     "TestReopeningTheLogContinuesTheSeqWithoutAGap",
	"é recusada e nada é escrito":                    "TestADecisionWithoutAReasonIsRefusedAndNothingIsWritten",
	"nunca uma linha perdida em silêncio":            "TestAWriteThatFailsIsAnErrorAndNeverALineLostInSilence",
	"devolvida por `Torn`":                           "TestATornLastLineIsCutOnReopenAndSaid",
	"Nenhuma linha do log passa de 64 KiB":           "TestALineOver64KiBCutsTheLargestFieldAndSaysHowMuch",
	"linha ilegível é dita com o número":             "TestAnUnreadableLineIsSaidWithItsNumberAndTheRestIsPrinted",
	"nunca sobe um próprio":                          "TestOrchestrateDispatchesAndNeverStartsADaemonOfItsOwn",
	"devolvem o mesmo `Project`":                     "TestTwoOpensOfOneTreeAreOneProject",
	"volta do reinício do daemon **sob o mesmo id**": "TestTheOrchestratorSurvivesADaemonRestart",

	"worktree descartável no commit exato":  "TestTheGateRunsInAThrowawayTreeWithoutGitEnvironment",
	"sai sempre, até depois de um pânico":   "TestTheThrowawayWorktreeIsRemovedEvenOnAPanic",
	"no home é gate vermelho":               "TestTheGateCannotWriteOutsideItsWorktree",
	"num `TMPDIR` só dele":                  "TestAGateWritesItsTreeAndItsOwnTemporaryDirectory",
	"Sem sandbox o gate não roda":           "TestWithoutTheSandboxTheGateDoesNotRunAndSaysWhy",
	"trancada pelo git morto no meio":       "TestALockedHalfMadeGateWorktreeIsRemoved",
	"devolve o cancelamento, nunca o erro":  "TestAGateStoppedWhileItsWorktreeIsMadeIsTheCancellation",
	"vai ao log como `claim`":               "TestAChildThatClaimsGreenIsRecordedRedBesideItsClaim",
	"nada roda, o resultado é `undeclared`": "TestAnUndeclaredGateIsSaidAndNothingRuns",
	"que não os do merge é `gate_stale`":    "TestAMeasurementOfAnotherCommitOrMainIsStale",

	"O revisor não tem ferramenta que escreve": "TestTheReviewerRegistryHasNoToolThatWrites",
	"nunca conta como aprovação":               "TestAReviewThatEndsWithoutAVerdictIsMissingNeverAnApproval",
	"nunca chega cortado ao revisor":           "TestADiffOverTheLimitIsCutAndSaidNeverTruncated",

	"a orquestradora fica `idle`, nunca `blocked`":   "TestCreatingChildrenLeavesTheOrchestratorIdleWhileTheyRun",
	"com os caminhos em disputa, e nenhuma worktree": "TestABatchWhoseOwnsCrossIsRefusedWholeAndNoWorktreeIsMade",
	"recusado inteiro com `max_children_reached`":    "TestABatchAboveMaxChildrenIsRefusedWhole",
	"recusado com `model_not_allowed`":               "TestAProfileOutsideTheListIsModelNotAllowed",
	"`steer_child`, `close_child` nem `ask`":         "TestTheChildsRegistryHasNoToolThatConducts",
	"é negado sem perguntar e com o porquê":          "TestGitThatWritesIsDeniedToAChildWithoutAsking",
	"Aprovação de filho não expira":                  "TestAChildsApprovalWaitsPastTheTimeoutAndSurvivesTheDaemon",
	"`hooks.changed` diz os dois valores":            "TestAWorktreeThatChangesTheHooksPathIsSaid",
	"no reinício termina com `daemon_restart`":       "TestAChildIsReopenedAfterARestartAndAnOpenTurnFails",
	"`delegate` é recusado com `ceiling_reached`":    "TestAtTheCeilingDispatchStopsAndContinueOpensANewWindow",
	"Fechado sai da lista depois da retenção":        "TestAClosedChildLeavesTheListAfterTheRetentionAndAFailedOneStays",
	"Merged sai da lista `closed_retention` depois":  "TestAMergedChildLeavesTheListAfterTheRetentionAndAFailedOneStays",

	"volta ao filho sem commit e sem revisão":               "TestADiffOutsideOwnsIsNeverCommittedAndReturnsWithoutAReviewer",
	"só é aceito depois de `red_test.confirmed`":            "TestTheRedTestMustFailWithoutTheFixBeforeTheFixIsTaken",
	"nunca chega ao gate":                                   "TestARedTestThatPassesIsNotTakenAndAnUnconfirmedFixNeverReachesTheGate",
	"não há terceira":                                       "TestAReviewWithoutAVerdictRunsOnceMoreThenStaysInReview",
	"Além de `max_returns` devoluções":                      "TestPastMaxReturnsTheChildFailsIncomplete",
	"mede o `Head` exato do filho":                          "TestTheGateMeasuresTheExactCommitAndALyingChildIsRecordedRed",
	"só no começo de um turno dela, com `Origin: children`": "TestAReportDuringATurnWaitsForThatTurnToEnd",
	"que inclui a main de agora":                            "TestTheGateMeasuresTheCommitThatHoldsMainAsItIsNow",
	"byte a byte o de antes (ADR-03)":                       "TestTheReportIsAppendedAndThePrefixIsByteForByteTheSame",

	// Step 12: the scripted scenarios, and the two tests that came with them.
	"tem um dos nomes da RN-7":                          "TestScenarioEveryEndOfAChildsTurnIsNamed",
	"roda `rebase`, `push --force` ou `commit --amend`": "TestNoCodeOfThisSliceRebasesForcePushesOrAmends",
	"linha que não grava impede a transição":            "TestATransitionIsLoggedBeforeItIsSaidAndALineThatFailsStopsIt",
	"nunca a saída do gate":                             "TestAProposalWithoutAGreenGateOpensNothing",
	"o filho sai da fila para medir de novo":            "TestAMainThatMovedAfterTheGateIsStaleAndNothingMerges",
	"antes da resposta `merge` da pessoa":               "TestAMergeWaitsForThePersonsYes",
	"passa `--match-head-commit`":                       "TestTheMergeOnGitHubPassesMatchHeadCommitWithTheMeasuredCommit",
	"com o valor antigo errado":                         "TestAConditionalUpdateWithTheWrongOldValueLeavesMainAlone",
	"em checkout em qualquer worktree":                  "TestALocalMergeIsRefusedWithMainCheckedOutSayingWhere",
	"salvo `fixes_main`":                                "TestARedMainStopsTheQueueAndFixesMainPasses",
	"vira `unknown` e para a fila":                      "TestPendingCIPastTheTimeoutIsUnknownAndStopsTheQueue",
	"a orquestradora não resolve conteúdo":              "TestAConflictReturnsToTheChildWithItsPathsAndTheIntegrationEndsClean",
	"numa worktree que a pessoa usa":                    "TestTheIntegrationIsADetachedWorktreeOfItsOwn",

	// The person decides in the conversation (.p §5.5, the owner's decision of 2026-10-18).
	"são um cartão só, na ordem em que foram feitas":  "TestPendingQuestionsOfOneKindShareOneCard",
	"dentro do lote também; item pulado":              "TestTheBatchAnswersEachItemAndTheFirstAnswerWins",
	"ferramentas da orquestradora alcançam responde":  "TestNoToolOfTheOrchestratorReachesAnAnswer",
	"Cartão proposto não é resposta":                  "TestProposingAnswersMakesAProposedCardAndAnswersNothing",
	"pergunta respondida não o aceita":                "TestASummaryIsAttachedToAWaitingMergeQuestion",
	"com o `seq` delas, nunca do que o filho disse":   "TestTheMergeEvidenceIsWhatTheLogMeasuredNeverTheClaim",
	"numa de merge vão à orquestradora, e a pergunta": "TestThePersonsOwnWordsAnswerADesignAndRelayAMerge",

	// The orchestrator's plan follows its children (the owner's request of 2026-10-18).
	"segue o estado dos filhos dele":     "TestAPlanStepFollowsTheStateOfItsChildren",
	"lido de volta, não manda outro":     "TestTheHarnessAdvancesThePlanStepFromTheChild",
	"`plan_step` negativo recusa o lote": "TestADelegateItemWithANegativePlanStepIsRefused",
	"longo ou ausente não recusa o lote": "TestALongOrMissingTitleGivesTheChildAShortOneAndTheLogSaysSo",
	"corta numa palavra e sai igual":     "TestAChildTitleIsShortAndDerivedTheSameWayEachTime",
}

func TestEveryInvariantHasATest(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	findings, err := specguard.Check(root, "orchestration", []string{".", filepath.Join("..", "..", "cmd", "dcode"), filepath.Join("..", "app"), filepath.Join("..", "tools")}, orchestrationInvariants)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Errorf("orchestration: %s", f)
	}
}
