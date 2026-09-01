package behavior

import (
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/specguard"
)

// Two groups of these invariants are asserted outside this package, and the
// reason is the same in both cases: this package is pure, and the invariant is
// about what the impure layers must not do with it.
//
// The workspace-overlay rule needs a workspace, which only internal/app has.
// The verification rules describe a seal derived from records the loop keeps.
// Duplicating either here would mean re-creating the very thing the purity of
// this package exists to keep out.
var behaviorDirs = []string{
	".",
	filepath.Join("..", "app"),
	filepath.Join("..", "loop"),
	filepath.Join("..", "vcs"),
	filepath.Join("..", "workspace"),
}

var behaviorInvariants = map[string]string{
	"saída byte-a-byte idêntica":         "TestBuildIsPure",
	"não emite timestamp":                "TestPromptCarriesNoVolatileData",
	"Ordem dos blocos":                   "TestGoldenPromptMarkdown",
	"Instrução adicionada após":          "TestNothingDiscoveredAfterAssemblyReachesThePrefix",
	"por par de fontes conflitantes":     "TestInstructionsStackWithTheMostSpecificLast",
	"tente afrouxar segurança":           "TestAnInstructionThatTriesToLoosenSafetyIsReported",
	"`Emit` é pura":                      "TestEmitIsPureAndOrdered",
	"Nenhum lembrete aparece no prefixo": "TestNoReminderTextEverReachesThePrefix",
	"idêntico entre emissões":            "TestEmitNormalisesOrderAndDuplicates",
	"apenas uma linha por skill":         "TestIndexCarriesOnlyTheLine",
	"numa palavra que nenhuma outra":     "TestMatchNeedsAWordThatBelongsToThisSkillAlone",
	"Palavra vazia de português":         "TestMatchDoesNotFireOnPortugueseFillerWords",
	"Vizinhas de um mesmo domínio":       "TestSkillsInTheSameDomainStayReachable",
	"famílias distintas produzem":        "TestTwoFamiliesProduceDifferentPrompts",

	// The doctrine overlay, whose whole point is what it cannot reach.
	"devolve a doutrina embarcada":                  "TestApplyOfAnEmptyOverlayChangesNothing",
	"== DefaultDoctrine().Safety":                   "TestNoOverlayCanEverChangeSafety",
	"como prefixo":                                  "TestToolPolicyIsAppendedToAndNeverReplaced",
	"raiz do **workspace**":                         "TestWorkspaceDoctrineFilesLeaveThePromptByteIdentical",
	"`safety.md` presente na raiz":                  "TestSafetyFileIsIgnoredAndRecorded",
	"Truncamento por teto":                          "TestOversizeFileIsTruncatedAndSaysSo",
	"Sobreposição resolvida após":                   "TestNothingDiscoveredAfterAssemblyReachesThePrefix",
	"auditoria do prompt reporta":                   "TestOriginsReportEverySectionAndSafetyIsAlwaysBuiltin",
	"`Practices` vazia **não** faz `Build` falhar":  "TestAnEmptyFloorDoesNotFailTheBuild",
	"posição é a precedência":                       "TestTheFloorSitsAfterSafetyAndBeforeWhatAnyoneSaid",
	"último bloco do prefixo":                       "TestProjectInstructionsStillComeAfterTheFloor",
	"não existe variante que acrescenta":            "TestPracticesMdReplacesAndDoesNotAppend",
	"continua sem alcançar `Safety`":                "TestTheOverlayReachesPracticesAndNeverSafety",
	"Piso substituído é reportado":                  "TestAReplacedFloorIsReportedAsReplaced",
	"pura com a seção de práticas":                  "TestTheFloorKeepsBuildPure",
	"vence sem discussão":                           "TestTheShippedFloorSaysWhoWins",
	"proíbe repetir-se":                             "TestTheShippedFloorForbidsRepeatingItself",
	"cobre os três defeitos":                        "TestTheShippedFloorCoversTheThreeDefects",
	"teto de tamanho da doutrina **inclui** o piso": "TestDoctrineStaysSmall",

	// The verification seal, kept by the loop.
	"`Verification` é função pura":               "TestTheSealIsAFunctionOfTheRecordsAndNothingElse",
	"Edição sem verificação":                     "TestVerificationSeal",
	"só leu arquivos produz `clean`":             "TestATurnThatChangedNothingRunsNoCheck",
	"continuação forçada é limitada":             "TestNoProgressEndsTheTurnAsIncomplete",
	"lembrete de verificação aparece no prefixo": "TestNoReminderTextEverReachesThePrefix",
	"trabalho sem plano é emitido":               "TestUnplannedWorkIsPointedOutOnceAndRearmsAfterAPlan",
	"carrega contagem no texto":                  "TestTheUnplannedNoticeCarriesNoCount",

	// Where the agent is working.
	"prefixo carrega branch":                    "TestThePromptSaysWhereInTheRepositoryWeAre",
	"não** é repositório é dito uma vez":        "TestAWorkspaceWithNoRepositorySaysSo",
	"não reivindica branch, árvore nem commits": "TestAnAbsentRepositoryClaimsNothingElse",
	"Instantâneo **não tomado**":                "TestASnapshotThatWasNeverTakenSaysNothing",
	"declarado como instantâneo":                "TestTheRepositorySnapshotSaysItIsASnapshot",
	"Árvore limpa é dita":                       "TestACleanTreeIsStatedRatherThanLeftBlank",
	"nunca é reportada como branch":             "TestADetachedHeadIsNotGivenABranchName",
	"limitado e o corte é declarado":            "TestAVeryDirtyTreeIsCutAndSaysSo",
	"continua pura com o repositório":           "TestTheRepositorySectionIsPure",
	"chegam ao prefixo com nome e comando":      "TestTheDeclaredGatesReachThePrefix",
	"afirma que eles passam":                    "TestTheGateListSaysNothingHasRunThem",
	"não gera seção":                            "TestNoDeclaredGatesMeansNoClaim",
	"cortada diz que foi cortada":               "TestATruncatedGateListSaysSo",
	"distinguidos pela fonte":                   "TestGatesThatShareANameAreToldApart",
	"repositório e portões juntos":              "TestTheWorkspaceBlockCarriesBothFacts",
	"pura com os portões":                       "TestTheGateSectionIsPure",
	"**não executa** nenhum deles":              "TestPackageScriptsBecomeGates",
	"regra com padrão não viram portão":         "TestMakefileNoiseIsNotAGate",
	"Sonda cancelada":                           "TestACancelledProbeReadsNothing",
}

func TestEveryInvariantHasATest(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	findings, err := specguard.Check(root, "behavior-definition", behaviorDirs, behaviorInvariants)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Errorf("behavior-definition: %s", f)
	}
}
