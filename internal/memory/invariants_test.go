package memory

import (
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/specguard"
)

// The family spans three packages, and each split is the same one: the ordering
// guarantee lives where instructions are ranked, the reading and the rendering
// live here, and the wiring that puts a memory in a real prompt lives where
// sessions are built.
var memoryDirs = []string{
	".",
	filepath.Join("..", "behavior"),
	filepath.Join("..", "app"),
	filepath.Join("..", "tools"),
	filepath.Join("..", "loop"),
}

var memoryInvariants = map[string]string{
	"Nada aprendido ordena acima":     "TestNothingLearnedOutranksAnythingAPersonWrote",
	"Nenhuma chave de configuração":   "TestNoConfigurationReachesTheAuthorityTable",
	"nomeia a procedência aprendida":  "TestThePromptNamesLearnedProvenanceAsLearned",
	"Memória cujo commit não existe":  "TestAMemoryFromAVanishedCommitIsMarkedAndKept",
	"além do teto declara o corte":    "TestPastTheCapTheOldestGoAndTheCutIsDeclared",
	"Bloco torto no arquivo":          "TestACrookedBlockIsReportedAndTheRestSurvives",
	"Memória sem procedência":         "TestAMemoryWrittenByHandNeedsNoProvenance",
	"Lista de tipos fechada em três":  "TestOnlyThreeKindsAreValid",
	"Workspace sem memória":           "TestAWorkspaceWithNoMemoryIsUnchanged",
	"continua pura com memória":       "TestTheLearnedBlockIsPure",
	"Workspace sem `.dcode/memory.md": "TestAWorkspaceWithNoMemoryReadsAsEmpty",

	// The tool.
	"recusa tipo fora dos três":       "TestAKindOutsideTheThreeIsRefusedByName",
	"recusa assunto vazio":            "TestAMemoryWithNoSubjectIsRefused",
	"acrescenta e nunca reescreve":    "TestRememberingAppendsAndLeavesWhatWasThere",
	"declara escrita no caminho":      "TestRememberDeclaresTheMemoryAndNothingElse",
	"carrega data e commit":           "TestEveryMemoryCarriesItsProvenance",
	"vale a partir da próxima sessão": "TestTheResultSaysItLandsNextSession",

	// The Layer 2 counterweight, built because measurement said the prompt was
	// not enough: four scenario designs, never one call.
	"duas vezes num turno pede":     "TestTheSameWallTwiceAsksForItToBeRemembered",
	"caminhos diferentes não pedem": "TestTwoDifferentFailuresAskForNothing",
	"sai **uma vez** por turno":     "TestTheWallIsMentionedOnce",
	"não é mandada chamá-la":        "TestWithoutTheToolNothingIsAsked",
	"não carrega contagem":          "TestTheWallNoticeCarriesNoCount",
}

func TestEveryInvariantHasATest(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	findings, err := specguard.Check(root, "learned-memory", memoryDirs, memoryInvariants)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Errorf("learned-memory: %s", f)
	}
}
