package tui

import (
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/specguard"
)

var tuiInvariants = map[string]string{
	"mesma entrada, mesma saída":         "TestRenderIsPureOverTheModelAndTheGeometry",
	"mostra o que já percorreu":          "TestACallInFlightShowsWhatItHasGotThrough",
	"nunca na última que começou":        "TestAResultLandsOnItsOwnCallAndNotTheLastOneStarted",
	"pousa na chamada que ele nomeia":    "TestACallsProgressLandsOnThatCall",
	"abre com ela mesmo sem plano":       "TestTheTurnsNumbersAloneOpenThePanel",
	"antes de o daemon ter dito":         "TestTheTurnSectionSaysNothingBeforeTheDaemonDoes",
	"zera a contagem e conserva":         "TestANewTurnStartsItsCountersAtZero",
	"não move os contadores do turno":    "TestProgressForAToolDoesNotMoveTheTurnsCounters",
	"muda de estilo ao se aproximar":     "TestTheRoundCountWarnsAsItNearsTheCeiling",
	"toda tecla é do nome":               "TestNamingTakesEveryKeyWhileItIsOpen",
	"nunca do título derivado":           "TestNamingStartsFromTheNameAndNotTheDerivedTitle",
	"mantém o que havia":                 "TestEscapingNamingKeepsWhatWasThere",
	"mesmo limite que o daemon":          "TestTheDraftStopsAtTheLimitTheDaemonEnforces",
	"marcado como dado":                  "TestAGivenNameIsMarkedAsGiven",
	"a coluna toma o teclado":            "TestTheRailTakesTheKeyboardAndGivesItBack",
	"nunca só cor, e `↑↓` não dão":       "TestTheRailCursorStopsAtBothEnds",
	"é **caractere**":                    "TestTheCursorIsACharacterAndNotOnlyAColour",
	"limpa o filtro primeiro":            "TestTheRailTakesTheKeyboardAndGivesItBack",
	"escolhe nada, e a lista diz":        "TestAnEmptyResultSaysSoRatherThanGoingBlank",
	"conversa já aberta não faz nada":    "TestChoosingTheOpenConversationDoesNothing",
	"Lista vazia não abre o modo":        "TestTheRailDoesNotOpenOntoAnEmptyList",
	"adjacentes são um bloco só":         "TestADelegationIsSeparatedFromWhatSurroundsIt",
	"nunca dobrado junto dos que":        "TestTheChildThatDidNotAnswerIsNamedOnItsOwnLine",
	"quantos ficaram sem resposta":       "TestTheHeaderCountsTheChildrenAndTheOnesMissing",
	"fronteira que recebeu":              "TestEveryChildShowsTheBoundaryItWasGiven",
	"nenhum filho é desenhado duas":      "TestChildrenAreNotDrawnTwice",
	"não pelo formato da string":         "TestAChildsNameIsNotAFile",
	"terminal que declarou não desenhá":  "TestNoBoxDrawingRuneSurvivesAsciiMode",
	"marcada por caractere, não só":      "TestTheOpenConversationIsMarkedByACharacter",
	"Conversas sozinhas já abrem":        "TestConversationsAloneAreEnoughToOpenTheSidebar",
	"diz que foi cortado":                "TestATruncatedTitleSaysItWasTruncated",
	"mesmo filtro do `dcode -r`":         "TestThePickListLeavesOutSessionsNobodyAskedAnythingIn",
	"em células e não em bytes":          "TestATitleIsCutInCellsAndNotInBytes",
	"mesma sessão reaberta, mesma lista": "TestTheSameEntriesProduceTheSameTree",
	"nenhuma ferramenta tocou não é":     "TestAPathNoToolTouchedIsNotDrawn",
	"linha de comando não entram":        "TestPatternsAndCommandsStayOutOfTheFileList",
	"não tocou nada não abre coluna":     "TestATurnThatTouchedNothingGetsNoSidebar",
	"nomeia a tecla que a traz":          "TestASidebarHiddenByWidthSaysSoAndNamesTheKey",
	"visível, não diz nada":              "TestAVisibleSidebarSaysNothingAboutItself",
	"vazia, não é anunciada":             "TestAnEmptySidebarIsNotAdvertised",
	"abaixo de 100 colunas ela some":     "TestTheSidebarDisappearsOnANarrowTerminal",
	"vence nos dois sentidos":            "TestAnExplicitChoiceWinsAtAnyWidthBothWays",
	"nunca lida de volta da frase":       "TestTheCountComesFromTheToolAndNotFromItsSentence",
	"não ultrapassa a largura dela":      "TestNoSidebarRowOverflowsTheColumn",
	"ela não emite escape nenhum":        "TestTheSidebarEmitsNoEscapeWithoutColour",
	"distintos por caractere":            "TestTheStatesStayApartWithoutUnicode",
	"exatamente uma vez, numa função":    "TestTheStreamPaysForEveryColumnExactlyOnce",
	"carrega corpo é bloco":              "TestACallWithABodyIsSeparatedFromWhatCameBefore",
	"e do que vem depois dela":           "TestWhatFollowsABlockIsSeparatedFromIt",
	"continua uma linha só":              "TestCallsWithoutBodiesStayPacked",
	"duas linhas em branco seguidas":     "TestTwoBlocksAreSeparatedByExactlyOneBlankLine",
	"termina em linha em branco":         "TestTheStreamDoesNotEndOnABlankLine",
	"nunca é desenhado sem a ferramenta": "TestTheVerbNeverAppearsWithoutTheToolItDescribes",
	"fala um idioma só":                  "TestTheWayOutSpeaksTheSameLanguageAsTheLine",
	"dica de expansão sob corpo":         "TestTheExpansionHintSpeaksTheInterfaceLanguage",
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
