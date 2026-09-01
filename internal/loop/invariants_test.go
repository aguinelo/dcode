package loop

import (
	"os"
	"path/filepath"
	"strings"
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

// The /loop façade is one package down, and the specguard's own glob only
// reaches internal/*/invariants_test.go — one level. A guard file inside
// loopcommand/ would never be read, which is why the claim for that family
// lives here.
var loopCommandDirs = []string{
	filepath.Join(".", "loopcommand"),
	filepath.Join("..", "tui"),
	filepath.Join("..", "app"),
	filepath.Join("..", "session"),
}

var loopCommandInvariants = map[string]string{
	"não panic) para `tasks.md` malformado":  "TestLoadSpecMalformedReturnsError",
	"abre sessão de **qualificação**":        "TestAGoalWithNoSpecFolderIsQualified",
	"nunca em pasta derivada da frase":       "TestTheHandlerDivertsAGoalWithNoFoldersInsteadOfRefusing",
	"**nomeia a frase**":                     "TestTheQualifyingTurnForAGoalNamesTheSentence",
	"continua desenhando o plano":            "TestTheHandlerStillDrawsThePlanWhenFoldersExist",
	"vira **objetivo qualificado**":          "TestLoopOneDivertsABareWordThatNamesNothing",
	"qualificar um caminho digitado errado":  "TestAMistypedPathIsNotQualifiedAway",
	"evento de **turno concluído**":          "TestTheProposalIsNotCommittedBeforeTheTurnStarts",
	"sem nenhuma linha de tarefa":            "TestLoadSpecWithoutTaskLinesIsAnError",
	"declaração de zero critérios":           "TestLoadSpecZeroCriteriaIsNotAnError",
	"não é sintaxe":                          "TestLoadSpecSeparatorIsNotSyntax",
	"sem comando entre crases":               "TestLoadSpecBadVerifyNamesTheLine",
	"não lê como `, exit: K`":                "TestLoadSpecUnreadableExitCodeIsAnError",
	"mesmo número de tarefa duas vezes":      "TestLoadSpecDuplicateTaskNumberIsAnError",
	"régua do markdown":                      "TestLoadSpecHorizontalRuleIsNotFrontmatter",
	"preserva a ordem de aparição":           "TestLoadSpecPreservesOrder",
	"vira critério igual a":                  "TestLoadSpecCheckedTaskIsStillACriterion",
	"união do declarado no arquivo":          "TestLoadSpecWithProtectLayersBoth",
	"aparece uma vez só":                     "TestLoadSpecProtectIsNotDuplicated",
	"vence o `tasks.md`":                     "TestASpecFolderCanDeclareItsOwnDoneFile",
	"nunca queda para o `tasks.md`":          "TestAnEmptySpecDoneFileIsAnError",
	"Ausência de `done.toml`":                "TestNoSpecDoneFileFallsToTasks",
	"frase é objetivo":                       "TestASentenceIsAGoalAndAPathIsAPath",
	"uma sessão por spec":                    "TestThePlanShowsEverySpecAndWhereItStands",
	"nunca contando marcações":               "TestPendingIsWhatTheCriteriaSay",
	"ausência de prova":                      "TestAFolderThatDeclaresNothingIsPending",
	"sem tarefas é pendente":                 "TestASpecWithNoTasksYetIsPendingAndNotBroken",
	"indisponível conta como trabalho":       "TestAnUnavailableCriterionIsWorkLeft",
	"não só as pendentes":                    "TestThePlanShowsEverySpecAndWhereItStands",
	"cancelada devolve o que já tinha":       "TestACancelledDiscoveryStopsWhereItIs",
	"depois da fila digitada":                "TestTheDaemonListsSpecsAndWhatIsPending",
	"pergunta não roda critério nenhum":      "TestTheLoopAsksBeforeItOpens",
	"qualificada antes de ser trabalhada":    "TestTheLoopQualifiesAFolderThatDeclaresNothing",
	"Survey que falha":                       "TestASurveyThatFailsDoesNotStopTheWork",
	"**não** entra no turno qualificador":    "TestAQualifyingTurnDoesNotCarryTheTask",
	"sem emendar no trabalho":                "TestCommittingAProposalReportsAndStops",
	"**submete um turno**":                   "TestLoopSubmitsSomethingToDo",
	"não flag mistecleada":                   "TestAWordAfterThePathIsTheTaskAndNotAFlag",
	"vence o texto padrão":                   "TestLoopSubmitsSomethingToDo",
	"não** repete os critérios":              "TestLoopSubmitsSomethingToDo",
	"não** vira entrada de turno":            "TestLoopIsACommandAndNotTurnInput",
	"não pode sombrear":                      "TestLoopCannotBeShadowed",
	"Flag desconhecida":                      "TestAMistypedFlagStopsTheCommand",
	"sem argumento mostra uso":               "TestParseLoopArgs",
	"sai do workspace é recusado":            "TestASpecPathCannotClimbOutOfTheWorkspace",
	"resolve contra ele":                     "TestASpecPathInsideResolvesAgainstTheWorkspace",
	"`done.toml` não é consultado":           "TestASpecNamedIsTheDefinitionOfDone",
	"encerra a criação da sessão":            "TestAnUnreadableSpecStopsTheSession",
	"zero critério e **não** é erro":         "TestASpecWithNoRunnableCriterionIsNotAnError",
	"zero é resposta, não ausência":          "TestTheSessionReportsHowManyCriteriaItCarries",
	"nenhum caminho é protegido por posição": "TestLoadSpecWithoutProtectDeclaresNothing",
	"ignora a presença de `done.toml`":       "TestLoadSourceLoopSpecReadsFile",
	"ignora a presença de `specPath`":        "TestLoadSourceDoneFileIgnoresSpecPath",
	"cai no `verifyCommand` legado":          "TestLoadSourceAutoMissingSpecFallsThroughToVerify",
	"presente e ilegível":                    "TestLoadSourceAutoMalformedSpecIsAnError",
	"mesmos argumentos, mesma `DoneSet`":     "TestLoadIsDeterministic",
	"igual ao `DoneSet` da `LoopSpec`":       "TestSessionConfigDoneMatchesSpec",
	"se e só se há ao menos um critério":     "TestSessionConfigDoneEnabledReflectsCriteria",
	"carrega prefixo, basename e instante":   "TestSessionConfigNameCarriesPrefixSpecAndInstant",
	"a fachada não tem orçamento próprio":    "TestSessionConfigCarriesTheLimitsUntouched",
}

// This replaces a test that claimed the same family and asserted nothing: it
// built a map it never read, then compared a constant to itself. It existed so
// the specguard's strings.Contains would find the family name in a file, and it
// could not fail. A guard that a literal satisfies is a guard that a literal
// will satisfy — the fix is to run the real check, which reads every line of
// the `.p §8` and fails when a named test does not exist.
func TestEveryLoopCommandInvariantHasATest(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	findings, err := specguard.Check(root, "loop-command", loopCommandDirs, loopCommandInvariants)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Errorf("loop-command: %s", f)
	}
}

// The qualifier is one package down, and the specguard's glob reaches only
// internal/*/invariants_test.go — one level. Same reason loopcommand's claim
// lives here.
var qualifierDirs = []string{
	filepath.Join(".", "qualifier"),
	filepath.Join("..", "app"),
	filepath.Join("..", "tools"),
}

var qualifierInvariants = map[string]string{
	"exatamente uma vez, na ordem proposta":   "TestEveryCriterionRunsOnceInOrder",
	"produzem `ClassBroken`":                  "TestNothingToRunIsBrokenAndNotRed",
	"nunca `Exit == 0`":                       "TestPassingIsComparedAgainstTheDeclaredExitCode",
	"legítimas por motivos opostos":           "TestFailingIsAcceptanceAndPassingIsRegression",
	"não é discordância; é condição própria":  "TestNothingToRunIsBrokenAndNotRed",
	"discordância entre o que o proponente":   "TestTheDisagreementIsFlagged",
	"devolve `ErrEmptyProposal`":              "TestAnEmptyProposalIsAnError",
	"nomeado** e nunca recusado":              "TestASetWithNothingRedIsNamedAndNotRefused",
	"Medir sem runner é erro":                 "TestNoRunnerIsAnError",
	"truncado em `MaxOutput`":                 "TestOutputIsCappedAndSaysSo",
	"prazo limita um critério":                "TestATimeoutBoundsOneCriterion",
	"não altera a proposta que recebeu":       "TestMeasureLeavesTheProposalAlone",
	"**não toca em nada**":                    "TestProposingIsAllowedInPlanModeBecauseItTouchesNothing",
	"pedido não muda isso":                    "TestAQualifyingSessionIsAlwaysPlanMode",
	"não carrega definição de pronto":         "TestAQualifyingSessionIsMeasuredAgainstNothing",
	"registra e termina":                      "TestTheDaemonHoldsAProposalUntilTheLoopTakesIt",
	"segunda proposta substitui":              "TestASecondProposalReplacesTheFirst",
	"não sobrevive para ser gravada":          "TestTheDaemonHoldsAProposalUntilTheLoopTakesIt",
	"sob a fronteira em que o trabalho":       "TestAProposalIsMeasuredWhereItsCriteriaCanRun",
	"Gravar nada é erro":                      "TestCommittingNothingIsAnError",
	"o carregador lê de volta":                "TestAProposalRoundTripsIntoADefinitionOfDone",
	"**gravado comentado**":                   "TestABrokenCriterionIsWrittenDownAndNotDeclared",
	"cortada antes de chegar ao arquivo":      "TestACommittedProposalCutsAHugeOutput",
	"medido de novo antes de congelar":        "TestAnEditedCriterionIsMeasuredAgain",
	"**acrescentou** volta ao operador":       "TestAnAddedCriterionGoesBackOnce",
	"assenta na hora":                         "TestAnEditThatKeepsTheClassSettlesAtOnce",
	"falha do canal terminam em `ErrRefused`": "TestAFailedAskIsARefusal",
	"Prazo esgotado nunca aprova":             "TestTheDeadlineRefusesAndNeverApproves",
	"Assinar sem runner é recusado":           "TestSigningWithNoRunnerIsRefusedUpFront",
	"Assinar conjunto vazio é recusado":       "TestSigningAnEmptySetIsRefused",
	"quebrado não entra na `DoneSet`":         "TestABrokenCriterionDoesNotReachTheFrozenSet",
	"**fica** vazio ao congelar":              "TestASetThatEmptiesItselfIsRefused",
	"mostrada junto dele":                     "TestTheConditionIsShownWithTheSet",
}

func TestEveryQualifierInvariantHasATest(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	findings, err := specguard.Check(root, "done-qualifier", qualifierDirs, qualifierInvariants)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Errorf("done-qualifier: %s", f)
	}
}

// The failure feedback family: what the loop knows when a criterion fails, and
// what it hands back.
var feedbackDirs = []string{".", filepath.Join("..", "behavior")}

var feedbackInvariants = map[string]string{
	"guarda a saída de todo critério que não passou":   "TestAFailingCriterionKeepsWhatItPrinted",
	"**não** guarda a saída de um critério que passou": "TestAPassingCriterionKeepsNothing",
	"indisponível não tem saída guardada":              "TestAnUnavailableCriterionKeepsNothing",
	"cortada em `MaxCriterionOutput`":                  "TestTheCeilingIsPerCriterionAndNotPerReport",
	"preserva o **fim**":                               "TestTruncationKeepsTheEnd",
	"fronteira de linha quando há uma":                 "TestTruncationCutsOnALineWhenItCan",
	"teto é por critério":                              "TestTheCeilingIsPerCriterionAndNotPerReport",
	"não lê saída":                                     "TestProgressDoesNotReadOutput",
	"**nomeia o seu comando**":                         "TestASilentCriterionNamesItsCommand",
	"Saída existente vence":                            "TestOutputWinsOverTheCommand",
	"passou não contribui nada":                        "TestAPassingCriterionNamesNothing",
	"renderiza o lembrete de hoje":                     "TestWithNoOutputTheReminderIsUnchanged",
	"vem **depois** da frase":                          "TestTheOutputFollowsTheSentence",
	"resultado observado e não instrução":              "TestTheBorrowedTextIsMarkedAsEvidenceOnce",
	"não ganha bloco vazio":                            "TestACriterionWithNoOutputGetsNoBlock",
	"fronteira dele ser visível":                       "TestTheBorrowedTextIsSetApart",
}

func TestEveryFailureFeedbackInvariantHasATest(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	findings, err := specguard.Check(root, "failure-feedback", feedbackDirs, feedbackInvariants)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Errorf("failure-feedback: %s", f)
	}
}

// The recoverable cycle: what one cycle did, and putting it back when it broke
// something.
var cycleDirs = []string{".", filepath.Join("..", "behavior"), filepath.Join("..", "tools")}

var cycleInvariants = map[string]string{
	"distingue avanço, empate e regressão":       "TestMovedTellsForwardFromNowhereFromBackward",
	"Trocar uma falha por outra é **regressão**": "TestSwappingAFailureIsRegressionAndNotADraw",
	"Conjunto que esvazia é avanço":              "TestMovedTellsForwardFromNowhereFromBackward",
	"não apaga o que o turno guardou":            "TestUndoCycleKeepsTheTurnUndoable",
	"restaura só o que **este ciclo escreveu**":  "TestUndoCycleLeavesTheEarlierCyclesAlone",
	"volta ao que o ciclo anterior deixou":       "TestUndoCycleLeavesTheEarlierCyclesAlone",
	"recusa, por arquivo":                        "TestUndoCycleLeavesTheEarlierCyclesAlone",
	"desfaz em regressão":                        "TestTheLoopUndoesARegressionAndNotADraw",
	"continua contando como ciclo parado":        "TestARolledBackCycleStillCountsAsAStall",
	"informado de que foi desfeito":              "TestARolledBackCycleIsToldToTheModel",
	"Nada é restaurado sem dizer":                "TestNoRollbackNoNotice",
	"não tem ferramenta que desfaça":             "TestUndoIsNotATheModelCanCall",
}

func TestEveryRecoverableCycleInvariantHasATest(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	findings, err := specguard.Check(root, "recoverable-cycle", cycleDirs, cycleInvariants)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Errorf("recoverable-cycle: %s", f)
	}
}

// Undo is the loop's or the person's, never the model's. An agent that can
// revert its own work can revert the evidence, and erasing what came back red
// is the cleanest way out of a loop that only ends when the red does.
func TestUndoIsNotATheModelCanCall(t *testing.T) {
	// Read from the source rather than from a registry built here: what the
	// product offers is decided where the product builds it, and a registry
	// assembled in a test would answer about itself.
	root, err := filepath.Abs(filepath.Join("..", "app"))
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join(root, "app.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"tools.Undo{", "tools.UndoTool{"} {
		if strings.Contains(string(src), forbidden) {
			t.Errorf("%s is registered, and undo may not be a tool the model calls", forbidden)
		}
	}
	if !strings.Contains(string(src), "NewRegistry(") {
		t.Fatal("the registry is no longer built here; this guard is reading nothing")
	}
}
