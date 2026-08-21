package server

import (
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/specguard"
)

// A spec family is not a Go package. Several of these invariants are about what
// the LOOP may emit and when — an ordering the server can only observe, never
// enforce — so their assertions live where the events are produced. Listing the
// directory keeps that visible; the alternative is an invariant reading as
// unclaimed because its test sits one package over, and the obvious fix for
// that is to duplicate the test.
var protocolDirs = []string{".", filepath.Join("..", "loop"), filepath.Join("..", "tui"), filepath.Join("..", "session"), filepath.Join("..", "app"), filepath.Join("..", "tools"), filepath.Join("..", "contextengine"), filepath.Join("..", "provider")}

var protocolInvariants = map[string]string{
	"estritamente crescente e sem lacunas": "TestEventsReplayThenStreamLive",
	"em vez de abrir buraco na sequência":  "TestProgressJoinsTheSequenceLikeAnyOtherEvent",
	"nomeia a chamada de onde veio":        "TestAScanReportsHowFarItHasGot",
	"está descobrindo e manda só":          "TestAWalkThatIsStillDiscoveringSendsNoTotal",
	"não reportam uma pela outra":          "TestTwoScansDoNotReportThroughEachOther",
	"sem perguntar se alguém escuta":       "TestAToolCanReportWithNobodyListening",
	"conjunto fechado":                     "TestEveryKindEmittedIsOneTheProtocolDeclares",
	"o teto viaja junto da contagem":       "TestATurnReportsItsRoundAgainstItsCeiling",
	"quantas chamadas rodam juntas":        "TestABatchReportsHowManyRunTogether",
	"respondeu numa passada":               "TestATurnThatAnsweredInOnePassReportsNoRound",
	"nunca entra no contexto enviado":      "TestProgressNeverEntersTheContextSentToTheModel",
	"Reproduzir de `from=1`":               "TestReplayReproducesTheLiveObservationExactly",
	"Nenhum evento é emitido após":         "TestNoEventCarriesACompletedTurnIDAfterItCompleted",
	"não emite `message.delta`":            "TestNothingIsStreamedWhileTheTurnIsBlocked",
	"Cliente desanexado durante turno":     "TestDisconnectingDoesNotAffectTheSession",
	"Duas resoluções concorrentes":         "TestApprovalIsResolvedOverTheWireAndSecondConflicts",
	"Aprovação expirada produz":            "TestAnApprovalNobodyAnswersResolvesOnceAndDenies",
	// The question itself. Emitted by the loop, rendered by the client, and
	// the two assertions live where each half is.
	"carrega o texto pedido":  "TestTheTurnAnnouncesWhatWasAsked",
	"vê a pergunta ao anexar": "TestAnAttachingClientSeesTheQuestion",
	// Reading a session back. The record is the session, so these assertions
	// live where records are read.
	"ordenada da mais recente":     "TestBrowsingPutsTheNewestFirst",
	"titulada pela primeira":       "TestASessionIsTitledByWhatWasAsked",
	"não é registro e não entra":   "TestRubbishInTheDirectoryIsSkipped",
	"junta os fragmentos de texto": "TestATranscriptReadsLikeTheConversation",
	// Continuing. The daemon assembles it; the rebuild is asserted where the
	// record is read.
	"cria sessão **nova** carregando": "TestContinuingASessionCarriesItsConversation",
	"Continuar sessão inexistente":    "TestContinuingAMissingSessionIsAnError",
	"entra no log da sessão nova":     "TestContinuingShowsAndRecordsWhatItCarries",
	"Continuar uma continuação":       "TestContinuingAContinuationKeepsTheWholeConversation",
	"sem resultado não entra":         "TestACallWithNoResultIsDropped",
	// Undo. The state owns what changed, the session owns when it may be
	// asked for, and the assertions live with each.
	"restaura o que o **último** turno": "TestANewTurnReplacesWhatCanBeUndone",
	"é recusado, nunca sobrescrito":     "TestUndoRefusesAFileChangedSinceTheTurn",
	"durante um turno em curso":         "TestUndoIsRefusedWhileATurnRuns",
	// Images.
	"imagem por valor":     "TestAnImageArrivesOnATurn",
	"recusado na borda":    "TestAMalformedImageIsRefused",
	"conta no orçamento":   "TestAnImageCostsContext",
	"declara se lê imagem": "TestEachFamilySaysWhetherItReadsPictures",
}

func TestEveryInvariantHasATest(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	findings, err := specguard.Check(root, "client-server-protocol", protocolDirs, protocolInvariants)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Errorf("client-server-protocol: %s", f)
	}
}
