package update

import (
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/specguard"
)

// The build stamp is asserted here rather than in internal/version, because
// what matters is not that String() renders a field but that the release
// pipeline sets it. A field the binary reports and no build writes reads as
// "local build" to every user of a published release.
var distributionDirs = []string{".", filepath.Join("..", "version")}

var distributionInvariants = map[string]string{
	"quando a assinatura não confere": "TestASignatureThatDoesNotVerifyInstallsNothingAndLeavesNoResidue",
	"quando o SHA-256 do artefato":    "TestAChecksumMismatchInstallsNothing",
	"Sem cosign na máquina":           "TestAMissingCosignStillChecksTheChecksumAndInstalls",
	"o SHA-256 divergente continua":   "TestAMissingCosignStillRefusesAMismatchedChecksum",
	"nomeia a ferramenta":             "TestAMissingCosignSaysTheSignatureWasNotVerified",
	"não baixa assinatura nem":        "TestAMissingCosignDoesNotDownloadTheSignatureItCannotCheck",
	"aborta listando as suportadas":   "TestAnUnsupportedPlatformAbortsAndListsWhatIsSupported",
	"com artefato corrompido":         "TestApplyLeavesTheCurrentBinaryIntactOnEveryFailure",
	"binário novo que não executa":    "TestApplyRefusesABinaryThatDoesNotRun",
	"mesmo sistema de arquivos":       "TestApplyStagesBesideTheTargetAndNotSomewhereElse",
	"sem chamada explícita a `update": "TestNothingAppliesAnUpdateWithoutTheUpdateCommand",
	"no máximo uma vez por":           "TestCheckHonoursTheInterval",
	"não altera o código de saída":    "TestCheckIsSilentWhenTheNetworkFails",
	"`CheckedAt` nunca entra":         "TestTheUpdateNoticeCannotReachWhatTheModelSees",
	"commit e data injetados":         "TestInjectedBuildReportsEverything",
	"Build local reporta":             "TestALocalBuildSaysSoAndAReleaseDoesNot",
	"recusa build local sem":          "TestApplyRefusesToOverwriteALocalBuild",
	"pipeline de release injeta":      "TestTheReleasePipelineStampsEveryFieldTheBinaryReportsOn",
	"mesmo SHA-256 para a mesma":      "TestTheFormulaCarriesTheReleaseDigests",
}

func TestEveryInvariantHasATest(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	findings, err := specguard.Check(root, "distribution", distributionDirs, distributionInvariants)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Errorf("distribution: %s", f)
	}
}
