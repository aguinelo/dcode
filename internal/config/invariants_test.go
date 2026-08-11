package config

import (
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/specguard"
)

// Eight of these invariants are about credentials, and none of them is asserted
// here: a secret is stored, masked, fingerprinted and read in
// internal/credential, and this package's part is only refusing to let one be
// written into config.toml. Two more are about whether every declared key is
// actually wired, which only internal/app can answer, since it owns the wiring.
//
// Listing the directories keeps that visible. The alternative — an invariant
// reading as unclaimed because its test sits one package over — invites exactly
// the fix the mapping exists to avoid.
var configDirs = []string{
	".",
	filepath.Join("..", "credential"),
	filepath.Join("..", "app"),
}

var configInvariants = map[string]string{
	"Cada raiz resolve":                      "TestRootsAreSeparateByDefault",
	"colapsa as quatro raízes":               "TestDcodeHomeCollapsesEveryRoot",
	"criada com `0700`":                      "TestEnsureCreatesOwnerOnly",
	"Chave desconhecida":                     "TestUnknownKeyIsAnErrorAndNamesTheAlternatives",
	"nome de credencial faz a inicialização": "TestCredentialShapedKeysAreRefusedInAnySection",

	// The credential rules, kept where the secret is.
	"nunca aparece no prompt":       "TestTheKeyNeverReachesThePrompt",
	"argumento de linha de comando": "TestTheSecretNeverReachesACommandLine",
	"Exibição padrão é mascarada":   "TestMaskShowsEnoughToRecogniseAndNotEnoughToUse",
	"Máscara de segredo curto":      "TestMaskHidesAShortSecretEntirely",
	"Impressão digital é estável":   "TestFingerprintIdentifiesWithoutRevealing",
	"escrito `0600`":                "TestFileStoreWritesOwnerOnly",
	"resolvem o mesmo backend":      "TestOpenHonoursAnExplicitBackend",
	"é recusado antes de alcançar":  "TestNamesAreBounded",

	// The key surface.
	"mapeamento é bijetivo":            "TestKeyToEnvMappingIsBijective",
	"é lida por alguém":                "TestEveryKnownKeyIsAccountedFor",
	"parte de `KnownKeys`":             "TestNonSessionKeysAreReadSomewhere",
	"por par de camadas adjacentes":    "TestPrecedenceChain",
	"devolve o valor travado":          "TestLockedOverrideIsWarnedAboutNotSwallowed",
	"pura sobre camadas já carregadas": "TestResolveIsPureOverTheLayersItIsGiven",
	"tem `Origin` não vazio":           "TestEveryValueCarriesItsOrigin",

	// Instruction discovery and commands.
	"nunca lê acima da raiz":      "TestDiscoveryNeverReadsAboveTheWorkspace",
	"depois de `AGENTS.md`":       "TestDcodeFileWinsOverAgentsFileInTheSameDirectory",
	"congelada na criação":        "TestDiscoveryIsAWalkWithNoMemorySoTheChainCanOnlyChangeOnPurpose",
	"em diretório tocado e fora":  "TestOutOfChainFindsAnInstructionTheSessionNeverLoaded",
	"não realiza I/O nem executa": "TestExpansionCannotReachForExecutionOrTheDisk",
	"vence comando de usuário":    "TestDiscoverCommandsLetsTheProjectWinAndRecordsIt",
}

func TestEveryInvariantHasATest(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	findings, err := specguard.Check(root, "configuration", configDirs, configInvariants)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Errorf("configuration: %s", f)
	}
}
