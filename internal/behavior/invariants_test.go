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
	"famílias distintas produzem":        "TestTwoFamiliesProduceDifferentPrompts",

	// The doctrine overlay, whose whole point is what it cannot reach.
	"devolve a doutrina embarcada": "TestApplyOfAnEmptyOverlayChangesNothing",
	"== DefaultDoctrine().Safety":  "TestNoOverlayCanEverChangeSafety",
	"como prefixo":                 "TestToolPolicyIsAppendedToAndNeverReplaced",
	"raiz do **workspace**":        "TestWorkspaceDoctrineFilesLeaveThePromptByteIdentical",
	"`safety.md` presente na raiz": "TestSafetyFileIsIgnoredAndRecorded",
	"Truncamento por teto":         "TestOversizeFileIsTruncatedAndSaysSo",
	"Sobreposição resolvida após":  "TestNothingDiscoveredAfterAssemblyReachesThePrefix",
	"auditoria do prompt reporta":  "TestOriginsReportAllFourSectionsAndSafetyIsAlwaysBuiltin",

	// The verification seal, kept by the loop.
	"`Verification` é função pura":               "TestTheSealIsAFunctionOfTheRecordsAndNothingElse",
	"Edição sem verificação":                     "TestVerificationSeal",
	"só leu arquivos produz `clean`":             "TestATurnThatChangedNothingRunsNoCheck",
	"continuação forçada é limitada":             "TestNoProgressEndsTheTurnAsIncomplete",
	"lembrete de verificação aparece no prefixo": "TestNoReminderTextEverReachesThePrefix",
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
