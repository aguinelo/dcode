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
	"é pura: mesma entrada":            "TestEvaluateIsPure",
	"Toda combinação das tabelas":      "TestModeTableIsComplete",
	"nunca devolve `allow` para":       "TestReadOnlyNeverAllowsAWrite",
	"nunca devolve `escalate`":         "TestNeverPolicyNeverEscalates",
	"symlink apontando para fora":      "TestSymlinkPointingOutIsACrossing",
	"não é considerado contido":        "TestContainmentIsByComponentNotByPrefix",
	"escapando do workspace":           "TestDotDotEscapeIsACrossingNotAnError",
	"Nenhuma execução ocorre sem":      "TestEveryToolPassesThroughPolicy",
	"impede a criação da sessão":       "TestNewFailsWhenTheSandboxCannotBeEstablished",
	"travada por administrador":        "TestTheLockedLayerBeatsEveryOtherSource",
	"nunca transforma negação":         "TestARuleNeverRescuesSomethingContainmentRefused",
	"nunca é avaliada sob política":    "TestTheApprovalPolicyStillGovernsRules",
	"ao menos tão permissivo":          "TestNeverDoesNotTurnARuleIntoADenial",
	"Regra dispara em `full-access":    "TestFullAccessStillAsksWhereARuleFires",
	"perguntada antes de leitura":      "TestAWriteIsAskedAboutBeforeARead",
	"não alcança caminho fora dele":    "TestARuleIgnoresWhatIsOutsideTheWorkspace",
	"carrega o padrão que casou":       "TestARuleAsksInsideTheWorkspace",
	"chaveado pela regra quando houve": "TestAllowForTheSessionIsKeyedByTheRuleThatAsked",
	"Padrão em branco":                 "TestABlankPatternMatchesNothing",
	"inspecionáveis por `--config`":    "TestTheEffectiveRulesAreInspectableWithTheirProvenance",
	"Workspace sob `/tmp`":             "TestAWorkspaceUnderTmpSurvivesTheWritableTmpfs",
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
