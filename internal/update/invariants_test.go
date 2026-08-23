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
// cmd/dcode joins the list because how a released binary REPORTS itself is a
// distribution concern: the build stamp is already asserted from here, and the
// flag that prints it lives there.
var distributionDirs = []string{
	".", filepath.Join("..", "version"), filepath.Join("..", "..", "cmd", "dcode"),
}

var distributionInvariants = map[string]string{
	"pela versão para a qual vai":     "TestADevBuildIsNamedForTheVersionItIsHeadingTo",
	"`-v` imprime a versão":           "TestDashVPrintsTheVersion",
	"quando a assinatura não confere": "TestASignatureThatDoesNotVerifyInstallsNothingAndLeavesNoResidue",
	"quando o SHA-256 do artefato":    "TestAChecksumMismatchInstallsNothing",
	"Sem cosign na máquina":           "TestAMissingCosignStillChecksTheChecksumAndInstalls",
	"o SHA-256 divergente continua":   "TestAMissingCosignStillRefusesAMismatchedChecksum",
	"coberta pelo digest carregado":   "TestAnInstallCoveredByItsCarriedDigestSaysNothingAboutCosign",
	"coberta pela assinatura também":  "TestAnInstallCoveredByItsSignatureIsAlsoQuiet",
	"nunca um pacote a instalar":      "TestAnUncoveredInstallPointsAtThePinnedInstallerAndNotAtAPackage",
	"atualiza sem cosign":             "TestUpdateWithoutCosignUsesTheDigestCommittedToMain",
	"discorda do download aborta":     "TestUpdateRefusesWhenTheCarriedDigestDisagrees",
	"não pede pacote nenhum":          "TestUpdateRefusesWhenNothingCoveredSubstitution",
	"aborta o update mesmo com":       "TestASignatureThatFailsStopsTheUpdateEvenWithACarriedDigest",
	"entre os marcadores conta":       "TestPinnedDigestIgnoresAnythingOutsideTheMarkers",
	"não baixa assinatura nem":        "TestAMissingCosignDoesNotDownloadTheSignatureItCannotCheck",
	"diverge do digest que carrega":   "TestACarriedDigestRefusesAnArtifactTheSignedListAccepts",
	"instala o release ao qual foi":   "TestACarriedDigestInstallsTheReleaseItWasPinnedTo",
	"avisa qual carrega e aponta":     "TestAnInstallerAskedForAnotherReleaseSaysItCannotCheckIt",
	"toda plataforma publicada":       "TestTheGeneratorCarriesEveryPublishedDigest",
	"recusa release ao qual falte":    "TestTheGeneratorRefusesAReleaseMissingAPlatform",
	"publica **depois** de fixar":     "TestTheReleasePinsTheInstallerFromChecksumsItAlreadyVerified",
	"branch que o README publica":     "TestThePinnedInstallerReachesTheBranchTheReadmePublishes",
	"não reprova um release já publi": "TestAnUnreachableRepositoryDoesNotRedenAPublishedRelease",
	"em vez de deixar a `main` com":   "TestAMissingPinnedInstallerStopsRatherThanLeavingMainStale",
	"Republicar o mesmo instalador":   "TestPublishingTheSameInstallerTwiceIsNotAFailure",
	"apenas o assunto exato dele":     "TestTheVersionIgnoresThePipelinesOwnPinCommit",
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
