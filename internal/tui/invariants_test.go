package tui

import (
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/specguard"
)

var tuiInvariants = map[string]string{
	"mesma entrada, mesma saída":         "TestRenderIsPureOverTheModelAndTheGeometry",
	"nunca é desenhado sem a ferramenta": "TestTheVerbNeverAppearsWithoutTheToolItDescribes",
	"fala um idioma só":                  "TestTheWayOutSpeaksTheSameLanguageAsTheLine",
	"troca a cada 20 quadros":            "TestTheVerbHoldsForTwentyFramesAndThenChanges",
	"tira a palavra e não os fatos":      "TestTurningTheVerbsOffLeavesTheFactsAlone",
	"verbo para toda fase":               "TestEveryLanguageHasAVerbForEveryPhase",
	"não agenda quadro nenhum":           "TestTheTickStopsWhenTheSessionIsIdle",
	"religa exatamente um":               "TestWorkStartingAgainRestartsExactlyOneTick",
	"Nenhum estado de sessão vive":       "TestReattachingAndReplayingReproducesTheSameScreen",
	"Fechar e reabrir reproduz":          "TestReattachingAndReplayingReproducesTheSameScreen",
	"Sucesso vem recolhido":              "TestErrorsOpenAndSuccessesStayCollapsed",
	"Diff nunca renderiza o conteúdo":    "TestADiffNeverRendersTheWholeFile",
	"Painel colapsa abaixo de":           "TestPanelCollapsesOnANarrowTerminalAndTheSummaryMoves",
	"`PanelShown` exibe o painel":        "TestTheUserCanForceThePanelOnANarrowTerminal",
	"Plano vazio não exibe painel":       "TestNoPlanMeansNoPanelEvenWhenForced",
	"Painel colapsado com plano vivo":    "TestAHiddenPanelSaysHowToShowIt",
	"Largura do painel nunca passa":      "TestThePanelGrowsOnAWideTerminalAndIsCapped",
	"Preferência explícita do usuário":   "TestTheUserCanForceThePanelOnANarrowTerminal",
	"usam a **mesma** formulação":        "TestThePanelAndTheStatusCountThePlanTheSameWay",
	"Item `blocked` nunca é escondido":   "TestPanelCollapsesOnANarrowTerminalAndTheSummaryMoves",
	"Modal de aprovação bloqueia":        "TestKeystrokesDoNotFallThroughTheModal",
	"Modal renderiza `ApprovalRequest":   "TestApprovalModalShowsTheCommandAndDefaultsToDeny",
	"Modal **não** exibe o plano":        "TestTheApprovalModalDoesNotShowThePlan",
	"é sempre renderizado com destaque":  "TestFullAccessIsLoudInPlainText",
	"Palavra digitada durante turno":     "TestWordsTypedMidTurnSteerIt",
	"Embutido que custa um turno":        "TestBuiltinTurnsAreQueuedWhileRunning",
	"Imagem anexada segura a mensagem":   "TestAnAttachedImageHoldsTheMessageBack",
	"Correção recusada porque o turno":   "TestACorrectionThatArrivedTooLateBecomesAMessage",
	"produz **um** turno":                "TestQueueJoinsIntoOneTurnInTypingOrder",
	"durante turno interrompe":           "TestCtrlCInterruptsMidTurnAndQuitsWhenIdle",
	"não sobrescreve embutido":           "TestBuiltinsBeatUserCommandsAndTheShadowingIsReported",
	"Estado vazio some no primeiro":      "TestEmptyStateDisappearsOnTheFirstTurnAndNeverReturns",
	"Sessão retomada com histórico":      "TestAResumedSessionNeverShowsTheEmptyState",
	"mascote degrada para ASCII":         "TestTheMascotKeepsItsShapeWithoutUnicodeAndWithoutColour",

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

	// Copy mode.
	"letra que as pessoas escrevem": "TestVIsALetterWhileTyping",
	"ela é dona do teclado":         "TestCopyModeOwnsTheKeyboardWhileItIsOpen",
	"A cópia sai por `Esc`":         "TestEveryWayOutOfCopyModeWorks",

	// Continuing. The flag lives in the command, one directory up from here.
	"em que algo foi perguntado":        "TestContinuingSkipsASessionNobodyAskedAnythingIn",
	"Só haver registro vazio":           "TestNothingButEmptyRecordsSaysSo",
	"Conversa continuada é **exibida**": "TestAContinuedConversationIsOnTheScreenAndSaysWhereItCameFrom",

	// The picker.
	"carrega a pergunta que foi feita": "TestThePickerShowsWhatEachConversationWasAbout",
	"cursor para nas duas pontas":      "TestTheCursorStopsAtBothEnds",
	"marcada **sem cor**":              "TestTheSelectedRowIsMarkedWithoutColour",
	"sair sem escolher":                "TestChoosingReturnsTheOneUnderTheCursorAndCancellingReturnsNothing",
	"não oferece sessão em que nada":   "TestThePickListLeavesOutSessionsNobodyAskedAnythingIn",
	"`-c` continua a última":           "TestResumeAsksAndContinueTakesTheLast",
}

func TestEveryInvariantHasATest(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	findings, err := specguard.Check(root, "client-tui",
		[]string{".", filepath.Join("..", "..", "cmd", "dcode")}, tuiInvariants)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Errorf("client-tui: %s", f)
	}
}
