package tui

import (
	"path/filepath"
	"testing"

	"github.com/aguinelo/dcode/internal/specguard"
)

var tuiInvariants = map[string]string{
	"mesma entrada, mesma saída":          "TestRenderIsPureOverTheModelAndTheGeometry",
	"contada em bytes de si mesma":        "TestAnArrivingCallIsCountedInBytesAndSaysSo",
	"nunca desenha uma segunda":           "TestTheCompleteCallFillsTheLineRatherThanAddingOne",
	"continua desenhando no":              "TestAProviderThatSaysNothingStillDrawsTheCall",
	"aparece na tela na hora":             "TestACallAppearsWhileItIsStillArriving",
	"mostra o que já percorreu":           "TestACallInFlightShowsWhatItHasGotThrough",
	"nunca na última que começou":         "TestAResultLandsOnItsOwnCallAndNotTheLastOneStarted",
	"pousa na chamada que ele nomeia":     "TestACallsProgressLandsOnThatCall",
	"antes de o daemon ter dito":          "TestTheTurnSectionSaysNothingBeforeTheDaemonDoes",
	"zera a contagem e conserva":          "TestANewTurnStartsItsCountersAtZero",
	"não move os contadores do turno":     "TestProgressForAToolDoesNotMoveTheTurnsCounters",
	"muda de estilo ao se aproximar":      "TestTheRoundCountWarnsAsItNearsTheCeiling",
	"toda tecla é do nome":                "TestNamingTakesEveryKeyWhileItIsOpen",
	"nunca do título derivado":            "TestNamingStartsFromTheNameAndNotTheDerivedTitle",
	"mantém o que havia":                  "TestEscapingNamingKeepsWhatWasThere",
	"mesmo limite que o daemon":           "TestTheDraftStopsAtTheLimitTheDaemonEnforces",
	"marcado como dado":                   "TestAGivenNameIsMarkedAsGiven",
	"a coluna toma o teclado":             "TestTheRailTakesTheKeyboardAndGivesItBack",
	"nunca só cor, e `↑↓` não dão":        "TestTheRailCursorStopsAtBothEnds",
	"é **caractere**":                     "TestTheCursorIsACharacterAndNotOnlyAColour",
	"limpa o filtro primeiro":             "TestTheRailTakesTheKeyboardAndGivesItBack",
	"escolhe nada, e a lista diz":         "TestAnEmptyResultSaysSoRatherThanGoingBlank",
	"conversa já aberta não faz nada":     "TestChoosingTheOpenConversationDoesNothing",
	"Lista vazia não abre o modo":         "TestTheRailDoesNotOpenOntoAnEmptyList",
	"adjacentes são um bloco só":          "TestADelegationIsSeparatedFromWhatSurroundsIt",
	"nunca dobrado junto dos que":         "TestTheChildThatDidNotAnswerIsNamedOnItsOwnLine",
	"quantos ficaram sem resposta":        "TestTheHeaderCountsTheChildrenAndTheOnesMissing",
	"fronteira que recebeu":               "TestEveryChildShowsTheBoundaryItWasGiven",
	"nenhum filho é desenhado duas":       "TestChildrenAreNotDrawnTwice",
	"não pelo formato da string":          "TestAChildsNameIsNotAFile",
	"terminal que declarou não desenhá":   "TestNoBoxDrawingRuneSurvivesAsciiMode",
	"marcada por caractere, não só":       "TestTheOpenConversationIsMarkedByACharacter",
	"não abre a coluna de arquivos":       "TestConversationsDoNotOpenTheFileColumn",
	"diz que foi cortado":                 "TestATruncatedTitleSaysItWasTruncated",
	"mesmo filtro do `dcode -r`":          "TestThePickListLeavesOutSessionsNobodyAskedAnythingIn",
	"em células e não em bytes":           "TestATitleIsCutInCellsAndNotInBytes",
	"mesma sessão reaberta, mesma lista":  "TestTheSameEntriesProduceTheSameTree",
	"nenhuma ferramenta tocou não é":      "TestAPathNoToolTouchedIsNotDrawn",
	"linha de comando não entram":         "TestPatternsAndCommandsStayOutOfTheFileList",
	"não tocou nada não abre coluna":      "TestATurnThatTouchedNothingGetsNoSidebar",
	"nomeia a tecla que a traz":           "TestASidebarHiddenByWidthSaysSoAndNamesTheKey",
	"visível, não diz nada":               "TestAVisibleSidebarSaysNothingAboutItself",
	"vazia, não é anunciada":              "TestAnEmptySidebarIsNotAdvertised",
	"abaixo de 100 colunas ela some":      "TestTheSidebarDisappearsOnANarrowTerminal",
	"vence nos dois sentidos":             "TestAnExplicitChoiceWinsAtAnyWidthBothWays",
	"nunca lida de volta da frase":        "TestTheCountComesFromTheToolAndNotFromItsSentence",
	"não ultrapassa a largura dela":       "TestNoSidebarRowOverflowsTheColumn",
	"ela não emite escape nenhum":         "TestTheSidebarEmitsNoEscapeWithoutColour",
	"distintos por caractere":             "TestTheStatesStayApartWithoutUnicode",
	"exatamente uma vez, numa função":     "TestTheStreamPaysForEveryColumnExactlyOnce",
	"carrega corpo é bloco":               "TestACallWithABodyIsSeparatedFromWhatCameBefore",
	"e do que vem depois dela":            "TestWhatFollowsABlockIsSeparatedFromIt",
	"continua uma linha só":               "TestCallsWithoutBodiesStayPacked",
	"duas linhas em branco seguidas":      "TestTwoBlocksAreSeparatedByExactlyOneBlankLine",
	"termina em linha em branco":          "TestTheStreamDoesNotEndOnABlankLine",
	"nunca é desenhado sem a ferramenta":  "TestTheVerbNeverAppearsWithoutTheToolItDescribes",
	"fala um idioma só":                   "TestTheWayOutSpeaksTheSameLanguageAsTheLine",
	"dica de expansão sob corpo":          "TestTheExpansionHintSpeaksTheInterfaceLanguage",
	"troca a cada 20 quadros":             "TestTheVerbHoldsForTwentyFramesAndThenChanges",
	"tira a palavra e não os fatos":       "TestTurningTheVerbsOffLeavesTheFactsAlone",
	"verbo para toda fase":                "TestEveryLanguageHasAVerbForEveryPhase",
	"não agenda quadro nenhum":            "TestTheTickStopsWhenTheSessionIsIdle",
	"religa exatamente um":                "TestWorkStartingAgainRestartsExactlyOneTick",
	"Nenhum estado de sessão vive":        "TestReattachingAndReplayingReproducesTheSameScreen",
	"Fechar e reabrir reproduz":           "TestReattachingAndReplayingReproducesTheSameScreen",
	"Sucesso vem recolhido":               "TestErrorsOpenAndSuccessesStayCollapsed",
	"Diff nunca renderiza o conteúdo":     "TestADiffNeverRendersTheWholeFile",
	"A barra carrega o contador do plano": "TestTheBarKeepsThePlanCount",
	"Item `blocked` nunca é escondido":    "TestASCIIFallbackKeepsBlockedDistinctFromDone",
	"Aprovação pendente bloqueia":         "TestKeystrokesDoNotFallThroughTheModal",
	"Bloco de aprovação renderiza":        "TestTheApprovalBlockShowsTheCommandAndDefaultsToDeny",
	"Bloco de aprovação **não** exibe":    "TestTheApprovalBlockAsksOneQuestion",
	"Pergunta respondida permanece":       "TestTheAnsweredQuestionStaysInTheStream",
	"é sempre renderizado com destaque":   "TestFullAccessIsLoudInPlainText",
	"Palavra digitada durante turno":      "TestWordsTypedMidTurnSteerIt",
	"Embutido que custa um turno":         "TestBuiltinTurnsAreQueuedWhileRunning",
	"Imagem anexada segura a mensagem":    "TestAnAttachedImageHoldsTheMessageBack",
	"Correção recusada porque o turno":    "TestACorrectionThatArrivedTooLateBecomesAMessage",
	"produz **um** turno":                 "TestQueueJoinsIntoOneTurnInTypingOrder",
	"durante turno interrompe":            "TestCtrlCInterruptsMidTurnAndTakesTwoWhenIdle",
	"não sobrescreve embutido":            "TestBuiltinsBeatUserCommandsAndTheShadowingIsReported",
	"Estado vazio some no primeiro":       "TestEmptyStateDisappearsOnTheFirstTurnAndNeverReturns",
	"Sessão retomada com histórico":       "TestAResumedSessionNeverShowsTheEmptyState",
	"mascote degrada para ASCII":          "TestTheMascotKeepsItsShapeWithoutUnicodeAndWithoutColour",

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

	// One line is one line.
	"achatado antes de ser medido": "TestAToolLineSurvivesAMultiLineCommand",
	"guarda o **fim**":             "TestACommandKeepsItsBeginningAndAPathItsEnd",
	// One file, one row.
	"é **uma linha só**":             "TestAFileIsCountedOnceWhicheverWayTheToolSpeltIt",
	"mantém a grafia que o encontra": "TestAPathOutsideTheWorkspaceKeepsItsFullSpelling",

	// A cut is announced, and colour costs nothing.
	"diz que foi cortada": "TestALineThatWasCutSaysSo",
	"nunca o que ela diz": "TestColourNeverChangesWhatIsOnTheScreen",
	// ASCII reaches every screen.
	"nenhuma runa acima de 127": "TestAsciiModeDrawsNothingButAscii",
	"o modelo produz texto":     "TestAsciiModeDrawsNothingButAscii",

	// Where an exchange begins.
	"Toda pergunta abre com uma régua": "TestATurnBeginsWithAVisibleBoundary",

	// The column is summoned, not resident.
	"A coluna **nasce escondida**": "TestTheSidebarStartsHidden",

	// The list is legible.
	"diz **quando** e **quanto**": "TestConversationRowsSayWhenAndHowMuch",
	"não lê relógio nenhum":       "TestTheConversationListReadsNoClock",

	// The panel earns its width.

	// One language per screen.
	"Nenhuma tela é escrita em literal": "TestNoEnglishSurvivesAPortugueseScreen",

	// Text has a hierarchy.
	"é **mais clara** que o que a qualifica":  "TestTheAnswerIsBrighterThanWhatQualifiesIt",
	"legível contra o fundo que o tema pinta": "TestEveryRoleIsLegibleAgainstTheGround",

	// A marker still arriving.
	"ainda sem par não é desenhado": "TestAMarkerStillArrivingIsNotDrawn",

	// The input area is a field.
	"delimitada nos quatro lados": "TestTheInputAreaIsDelimited",
	"a cor não muda a forma dela": "TestTheFrameIsTheSameShapeWithAndWithoutColour",

	// Lanes.
	"está numa **raia**":       "TestEveryStreamRowIsInALane",
	"não custa coluna nenhuma": "TestTheLaneCostsNoColumns",

	// The plan is a block in the stream.
	"plano é um bloco no fluxo":   "TestThePlanIsABlockInTheStream",
	"contam o mesmo plano":        "TestThePlanBlockAndTheStatusCountThePlanTheSameWay",
	"teto do turno chega à barra": "TestTheCeilingReachesTheBarOnceItIsClose",

	// Colour off means colour off.
	"Cor desligada não emite escape": "TestColourOffPaintsNoGround",

	// The side column.
	"o painel de diff sobre o de sessão": "TestTheSideColumnIsTwoFifths",
	"Ela aparece sozinha a partir de":    "TestTheSideColumnAppearsOnATerminalWideEnoughForIt",

	// The column earns its width.
	"não repete o fluxo":                "TestTheSideColumnSaysWhatIsNowhereElse",
	"maior mudança do turno":            "TestTheBarsAreScaledToTheLargestChangeAndSaySo",
	"legenda das raias aparece uma vez": "TestTheLaneLegendAppearsOnlyWhenItExplainsSomething",
	// The meter measures the context.
	"nunca passa de 100%": "TestTheMeterOnTheScreenIsTheOneThatWasFixed",

	// Resuming paints once.
	"desenha **uma linha** enquanto lê": "TestResumingPaintsALoadingLineUntilTheBacklogIsRead",
	"A linha se move":                   "TestTheLoadingLineKeepsTicking",

	// The context says it is filling.
	"avisada de que o contexto está enchendo": "TestTheContextSaysItIsFillingBeforeItIsCut",

	// Copy mode.
	"`^O` abre a cópia":                        "TestTheChordOpensCopyMode",
	"nunca é atalho":                           "TestVIsALetterWhereverTheCursorIs",
	"Entrar no fluxo é **deliberado**":         "TestSteppingIntoTheTranscriptIsDeliberate",
	"toda tecla que ele não nomeia é engolida": "TestNavModeSwallowsEveryKeyItDoesNotName",
	"`t` percorre os temas":                    "TestTheThemeKeyOnlyExistsInsideTheMode",
	"ela é dona do teclado":                    "TestCopyModeOwnsTheKeyboardWhileItIsOpen",
	"A cópia sai por `Esc`":                    "TestEveryWayOutOfCopyModeWorks",

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
