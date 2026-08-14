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
var protocolDirs = []string{".", filepath.Join("..", "loop"), filepath.Join("..", "tui"), filepath.Join("..", "session")}

var protocolInvariants = map[string]string{
	"estritamente crescente e sem lacunas": "TestEventsReplayThenStreamLive",
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
