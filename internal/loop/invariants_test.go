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
var qualifierDirs = []string{filepath.Join(".", "qualifier")}

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
