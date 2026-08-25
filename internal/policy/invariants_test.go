package policy

import (
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/specguard"
)

// The family spans four packages, and each split is the same one: this package
// decides, and the others are where the decision has consequences. The
// session-creation guarantee needs a session (internal/app), the every-tool
// guarantee needs the tools (internal/tools), the locked-config guarantee needs
// the chain (internal/config), and inspectability needs the command that prints
// it (cmd/dcode).
var sandboxDirs = []string{
	".",
	filepath.Join("..", "sandbox"),
	filepath.Join("..", "app"),
	filepath.Join("..", "config"),
	filepath.Join("..", "tools"),
	filepath.Join("..", "session"),
	filepath.Join("..", "..", "cmd", "dcode"),
}

var sandboxInvariants = map[string]string{
	"Cada modo nomeia exatamente o par":       "TestModeFromIsTheExactInverse",
	"não recebe nome":                         "TestAPairThatIsNoModeHasNoName",
	"anuncia o modo que o motor está de fato": "TestASessionSaysTheModeItIsActuallyIn",
	"Duas trocas concorrentes":                "TestConcurrentSwitchesLeaveOneMode",
	"depois de **provar**":                    "TestSeatbeltProbesRatherThanTrustingThePath",
	"pular dizendo o motivo":                  "TestABoundaryTestSkipsLoudlyRatherThanPassingQuietly",

	"que uma toolchain precisa":              "TestAToolchainCanWriteToItsOwnCache",
	"Nenhuma regra concede o diretório home": "TestTheHomeDirectoryIsNeverGranted",
	"não concede nenhum deles":               "TestReadOnlyGrantsNoCache",
	"não concede nada, e não derruba":        "TestANilEnvironmentGrantsNothingAndDoesNotPanic",

	"é pura: mesma entrada":                 "TestEvaluateIsPure",
	"Toda combinação das tabelas":           "TestModeTableIsComplete",
	"nunca devolve `allow` para":            "TestReadOnlyNeverAllowsAWrite",
	"nunca devolve `escalate`":              "TestNeverPolicyNeverEscalates",
	"symlink apontando para fora":           "TestSymlinkPointingOutIsACrossing",
	"não é considerado contido":             "TestContainmentIsByComponentNotByPrefix",
	"escapando do workspace":                "TestDotDotEscapeIsACrossingNotAnError",
	"Nenhuma execução ocorre sem":           "TestEveryToolPassesThroughPolicy",
	"impede a criação da sessão":            "TestNewFailsWhenTheSandboxCannotBeEstablished",
	"travada por administrador":             "TestTheLockedLayerBeatsEveryOtherSource",
	"nunca transforma negação":              "TestARuleNeverRescuesSomethingContainmentRefused",
	"nunca vira pergunta":                   "TestTheApprovalPolicyStillGovernsRules",
	"com ninguém para perguntar":            "TestNeverAnswersARuleWithNo",
	"Rede concedida deixa de ser":           "TestWithdrawingTheNetworkGrantAsksAgain",
	"nunca contenção":                       "TestTheGrantNeverOpensWhatTheSandboxCloses",
	"Comando destrutivo pede":               "TestDestructiveCommandsAlwaysAsk",
	"que sai da máquina pede confirmação":   "TestACommandThatLeavesTheMachineAsks",
	"sem ser execução remota":               "TestOrdinaryWorkStillDoesNotAsk",
	"Trabalho comum não pergunta":           "TestOrdinaryCommandsDoNotAsk",
	"Regra dispara em `full-access":         "TestFullAccessStillAsksWhereARuleFires",
	"perguntada antes de leitura":           "TestAWriteIsAskedAboutBeforeARead",
	"não alcança caminho fora dele":         "TestARuleIgnoresWhatIsOutsideTheWorkspace",
	"carrega o padrão que casou":            "TestARuleAsksInsideTheWorkspace",
	"chaveado pela regra quando houve":      "TestAllowForTheSessionIsKeyedByTheRuleThatAsked",
	"Padrão em branco":                      "TestABlankPatternMatchesNothing",
	"inspecionáveis por `--config`":         "TestTheEffectiveRulesAreInspectableWithTheirProvenance",
	"Workspace sob `/tmp`":                  "TestAWorkspaceUnderTmpSurvivesTheWritableTmpfs",
	"alcança o primeiro quadro":             "TestAChromiumReachesItsFirstFrameInsideTheSandbox",
	"não depende de o diretório já existir": "TestTheWorkspaceMountDoesNotDependOnThePathExisting",

	"não entrega socket unix":                              "TestSeatbeltGrantsTheNetworkWithoutTheMachinesOwnSockets",
	"Rede concedida inclui escutar":                        "TestAPortCanBeBoundInsideTheSandbox",
	"onde já se pode escrever":                             "TestAUnixSocketIsReachableWhereWritingIs",
	"não é lido de dentro do sandbox":                      "TestARealNamedStoreCannotBeReadFromInside",
	"não esconde nada":                                     "TestSeatbeltFullAccessHidesNothing",
	"o home inteiro nunca é um nome válido":                "TestUnreadableExpandsHomeAndRefusesIt",
	"esconde os cofres de credencial mesmo assim":          "TestUnreadableDefaultsToHidingCredentialStores",
	"própria credencial do dcode está entre os escondidos": "TestTheDefaultHidesDcodesOwnCredential",
	"acompanha o que o cofre escreve":                      "TestTheHiddenCredentialNameMatchesTheStore",
	"Socket concedido por nome é alcançável":               "TestSeatbeltReachesAGrantedSocket",
	"gravável fora do workspace, e `read-only` não ganha":  "TestSeatbeltReadOnlyIgnoresAGrantedPath",
	"assim que o agente é alcançável":                      "TestTheKeyIsHiddenOnceTheAgentIsReachable",
	"sem agente rodando concede nada":                      "TestTheAgentTokenGrantsNothingWhenNoAgentIsRunning",
	"deixa de ser socket dentro":                           "TestARealRuntimeSocketIsCoveredInside",
	"mantém a concessão ampla":                             "TestSeatbeltFullAccessKeepsTheBlanketGrant",
}

func TestEveryInvariantHasATest(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	findings, err := specguard.Check(root, "sandbox-policy", sandboxDirs, sandboxInvariants)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Errorf("sandbox-policy: %s", f)
	}
}
