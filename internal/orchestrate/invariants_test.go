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
	"devolvem o mesmo `Project`":                     "TestTwoOpensOfOneTreeAreOneProject",
	"volta do reinício do daemon **sob o mesmo id**": "TestTheOrchestratorSurvivesADaemonRestart",

	"worktree descartável no commit exato":  "TestTheGateRunsInAThrowawayTreeWithoutGitEnvironment",
	"sai sempre, até depois de um pânico":   "TestTheThrowawayWorktreeIsRemovedEvenOnAPanic",
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
