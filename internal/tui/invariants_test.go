package tui

import (
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/specguard"
)

var tuiInvariants = map[string]string{
	"mesma entrada, mesma saída":        "TestRenderIsPureOverTheModelAndTheGeometry",
	"Nenhum estado de sessão vive":      "TestReattachingAndReplayingReproducesTheSameScreen",
	"Fechar e reabrir reproduz":         "TestReattachingAndReplayingReproducesTheSameScreen",
	"Sucesso vem recolhido":             "TestErrorsOpenAndSuccessesStayCollapsed",
	"Diff nunca renderiza o conteúdo":   "TestADiffNeverRendersTheWholeFile",
	"Painel colapsa abaixo de":          "TestPanelCollapsesOnANarrowTerminalAndTheSummaryMoves",
	"`PanelShown` exibe o painel":       "TestTheUserCanForceThePanelOnANarrowTerminal",
	"Plano vazio não exibe painel":      "TestNoPlanMeansNoPanelEvenWhenForced",
	"Painel colapsado com plano vivo":   "TestAHiddenPanelSaysHowToShowIt",
	"Largura do painel nunca passa":     "TestThePanelGrowsOnAWideTerminalAndIsCapped",
	"Preferência explícita do usuário":  "TestTheUserCanForceThePanelOnANarrowTerminal",
	"usam a **mesma** formulação":       "TestThePanelAndTheStatusCountThePlanTheSameWay",
	"Item `blocked` nunca é escondido":  "TestPanelCollapsesOnANarrowTerminalAndTheSummaryMoves",
	"Modal de aprovação bloqueia":       "TestKeystrokesDoNotFallThroughTheModal",
	"Modal renderiza `ApprovalRequest":  "TestApprovalModalShowsTheCommandAndDefaultsToDeny",
	"Modal **não** exibe o plano":       "TestTheApprovalModalDoesNotShowThePlan",
	"é sempre renderizado com destaque": "TestFullAccessIsLoudInPlainText",
	"Entrada durante turno ativo":       "TestInputIsQueuedWhileRunningAndDrainsAsOneTurn",
	"produz **um** turno":               "TestQueueJoinsIntoOneTurnInTypingOrder",
	"durante turno interrompe":          "TestCtrlCInterruptsMidTurnAndQuitsWhenIdle",
	"não sobrescreve embutido":          "TestBuiltinsBeatUserCommandsAndTheShadowingIsReported",
	"Estado vazio some no primeiro":     "TestEmptyStateDisappearsOnTheFirstTurnAndNeverReturns",
	"Sessão retomada com histórico":     "TestAResumedSessionNeverShowsTheEmptyState",
	"mascote degrada para ASCII":        "TestTheMascotKeepsItsShapeWithoutUnicodeAndWithoutColour",

	// Prose rendering.
	"Nenhum marcador de markdown":      "TestEmphasisMarkersDoNotReachTheScreen",
	"prosa estilizada excede a coluna": "TestStyledProseNeverExceedsItsColumn",

	// The bottom bar.
	"worktree ativo nunca é descartado": "TestTheWorktreeSurvivesEveryWidth",
	"Segmento sem dado não desenha":     "TestWhatIsWaitingNeverDropsAndVanishesWhenThereIsNothing",
	// The input box.
	"desenha exatamente as linhas que o layout reservou": "TestTheFrameReservesExactlyWhatTheBoxDraws",
	"cobre a largura inteira do quadro":                  "TestEveryRowOfTheBoxCoversItsWidth",
	"só na primeira linha da caixa":                      "TestOnlyTheFirstRowCarriesThePrompt",
	"rola por dentro, mantendo o caret":                  "TestTheBoxScrollsToKeepTheCaretVisible",
	"cede antes do fluxo":                                "TestTheBoxNeverTakesTheWholeWindow",
	"início e ao fim da linha do caret":                  "TestTheLineKeysStayOnTheirLine",
	"diz o que aconteceu em todos":                       "TestPastingReportsEveryOutcome",
	"vai com a **próxima** mensagem":                     "TestAnAttachedImageGoesWithTheNextMessage",
	"nomeando o modelo":                                  "TestPastingIntoAModelThatCannotSeeNamesTheModel",
	"preserva as quebras e não envia":                    "TestAMultiLinePasteDoesNotSendAnything",
}

func TestEveryInvariantHasATest(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	findings, err := specguard.Check(root, "client-tui", []string{"."}, tuiInvariants)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Errorf("client-tui: %s", f)
	}
}
