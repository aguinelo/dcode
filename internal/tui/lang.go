package tui

import (
	"strings"
)

// Lang is a declared interface language.
type Lang string

const (
	PtBR Lang = "pt-BR"
	En   Lang = "en"
)

// Fallback is where an unknown or absent language lands.
//
// pt-BR, and that is product identity rather than imposition: whoever has their
// machine in English gets English, and what the fallback decides is only which
// language wins when there is no information at all. A product whose fallback
// is pt-BR is a Brazilian product; one whose fallback is English is an English
// product with a translation bolted on.
const Fallback = PtBR

// Resolve picks the interface language.
//
// DCODE_LANG beats LC_ALL, which beats LANG. An unknown language resolves to
// the fallback WITHOUT an error: refusing to start over an unrecognised locale
// would be a worse answer than showing Portuguese.
func Resolve(get func(string) string) Lang {
	for _, name := range []string{"DCODE_LANG", "LC_ALL", "LANG"} {
		if l, ok := parseLang(get(name)); ok {
			return l
		}
	}
	return Fallback
}

// parseLang reads a locale in any of the shapes an environment uses:
// `pt_BR.UTF-8`, `pt-br`, `en_US`, `en`.
func parseLang(v string) (Lang, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", false
	}
	// Cut the encoding and any modifier: pt_BR.UTF-8@euro → pt_BR
	if i := strings.IndexAny(v, ".@"); i >= 0 {
		v = v[:i]
	}
	v = strings.ReplaceAll(v, "_", "-")
	lower := strings.ToLower(v)

	switch {
	case lower == "pt" || strings.HasPrefix(lower, "pt-"):
		return PtBR, true
	case lower == "en" || strings.HasPrefix(lower, "en-"):
		return En, true
	}
	// A language dcode does not have. Not an error, and not a silent English
	// fallback either: it lands where everything else with no information lands.
	return "", false
}

// Strings is every piece of text the client composes itself.
//
// What is NOT here is anything the model reads: tool descriptions, tool error
// text, and the doctrine. RN-3 of behavior-definition makes tool error text a
// behaviour surface — the layer where recovery is taught — so translating it is
// changing the prompt. And the behavioural thresholds were measured in English,
// so translating invalidates the measurement without breaking anything
// visibly. The client translates only what it wraps around them.
type Strings struct {
	// Working is the activity line with no tool running — the one plain word
	// it falls back to. Deliberately not one of the rotating verbs: see
	// activity.go.
	Working string
	// RailFiles heads the sidebar, and RailTouchedOne/Many count what the turn
	// has touched — the header still says something when the column is narrow.
	ChildOne      string
	ChildMany     string
	ChildOwns     string
	ChildNoAnswer string
	ChildUnnamed  string
	PanelRounds   string
	PanelInFlight string
	RailHidden    string
	RailFiles     string
	RailSessions  string
	RailFilter    string
	RailNaming    string
	RailNoMatch   string

	// The side column.
	SideDiff       string
	SideSession    string
	SideNothingYet string
	SideContext    string
	SideAllowed    string
	SideRecent     string
	SideBarScale   string

	// The lane legend.
	LaneYou     string
	LaneProcess string
	LaneAnswer  string

	// The nav bar.
	NavBadge     string
	NavSessions  string
	NavColumn    string
	NavKeys      string
	NavEnter     string
	NavMove      string
	NavOpen      string
	NavPrompt    string
	NavLeave     string
	SideToolOne  string
	SideToolMany string
	// While the history of a continued conversation is being read.
	Loading    string
	LoadedOne  string
	LoadedMany string

	// The context filling up, and the summary when it does.
	ContextFilling string
	Compacted      string
	CompactedCount string

	// The approval modal. It was written in English literals — the ONE screen
	// that asks whether a boundary may be crossed, in a language the reader may
	// not have. Consent given to a sentence somebody could not read is not
	// consent.
	ApprovalCrosses        string
	ApprovalNetwork        string
	ApprovalEnter          string
	ShellHint              string
	LeavingTakesTwo        string
	UpdateApplied          string
	UpdateCurrent          string
	UpdateUnavailable      string
	ApprovalCrossing       string
	ApprovalRule           string
	ApprovalAnswered       string
	ApprovalDenied         string
	ApprovalAllowedOnce    string
	ApprovalAllowedSession string
	ApprovalAllowedProject string
	ApprovalAllowedAlways  string
	KeyDeny                string
	KeyAllow               string
	KeySession             string
	KeyNo                  string
	KeyOnce                string
	KeyProject             string
	KeyAlways              string
	// PanelPlan is the panel's own heading, which was a literal too.
	PanelPlan string
	// PlanOf and PlanBlockedCount build the plan's footer count.
	PlanOf           string
	PlanBlockedCount string
	SessionsMoreOne  string
	SessionsMoreMany string
	SessionsKeys     string
	RailTouchedOne   string
	RailTouchedMany  string

	// LineOne and LineMany count hidden lines in a collapsed body, and
	// ExpandHint says how to see them. All three because the hint said
	// "Tab expande" in Portuguese next to a count in English, on one line, in
	// both interfaces.
	LineOne    string
	LineMany   string
	ExpandHint string

	// WorkingInterrupt is the way out, on the activity line. Its own string
	// rather than Interrupt, which is the `esc` hint somewhere else: two keys
	// with one sentence between them is how a hint ends up naming the wrong
	// key in one of the languages.
	WorkingInterrupt string

	// Status line
	VerifiedLabel    string
	NotVerifiedLabel string
	UnverifiedLabel  string

	// Completion report
	VerifiedSummary     string // takes a count
	NotVerifiedSummary  string // takes the failing names
	NothingCouldCheck   string
	ChangedAfterCheck   string
	CompletionMet       string
	CompletionUnmet     string
	CompletionUnchecked string
	CompletionMeasure   string

	// Approval
	ApprovalDeny         string
	ApprovalAllowOnce    string
	ApprovalAllowSession string
	ApprovalEnterDenies  string
	ApprovalHeading      string

	// Help. The key and command descriptions live here too: a /help with
	// translated headings and English descriptions is worse than an untranslated
	// one, because it reads as a bug rather than as a language.
	HelpCommands     string
	HelpApprovals    string
	HelpKeys         string
	HelpYours        string
	KeyEnter         string
	KeyNewline       string
	KeyPasteImage    string
	CmdUndo          string
	CmdUpdate        string
	CmdImage         string
	CmdImageArgs     string
	ImageUsage       string
	ImageAttached    string
	ImageFailed      string
	ImageUnsupported string
	ImagePasted      string
	ImageTooBig      string
	ClipboardEmpty   string
	ClipboardMissing string
	UndoRestored     string
	UndoRefused      string
	UndoNothing      string
	UndoFailed       string
	KeyArrows        string
	KeyPage          string
	KeyTab           string
	KeyEsc           string
	KeyPanel         string
	KeyDequeue       string
	KeyEditing       string
	KeyInterrupt     string
	KeyQuit          string
	CmdHelp          string
	CmdInit          string
	CmdClear         string
	CmdPlan          string
	CmdPlanArgs      string
	CmdConfig        string
	CmdConfigArgs    string
	CmdModel         string
	CmdModelArgs     string
	CmdResume        string
	CmdResumeArgs    string
	CmdMode          string
	CmdModeArgs      string
	CmdModeCurrent   string // takes a mode name
	CmdModeUnnamed   string // the boundary in force is none of the three
	CmdModeUnknown   string // takes the name that is not a mode

	// CLI. The usage block is one string per language rather than a field per
	// line: it is prose with alignment, and cutting it into thirty fields
	// would make it harder to translate well, not easier.
	Usage string

	// Copy mode
	CopySelected string // takes a line count
	CopyKeys     string
	CopyDone     string
	CopyEmpty    string

	// Empty state and general
	Interrupt string
	Queued    string
	// The bottom bar names what it counts, so the numbers survive a terminal
	// with no colour to group them.
	BarFiles   string
	BarWaiting string
	NoPlan     string
	// The session picker.
	PickerTitle     string
	PickerKeys      string
	PickerEmpty     string
	PickerUntitled  string
	PickerYesterday string
	// PickerTurnOne and PickerTurnMany are the noun, and plural() puts the
	// number in front. It used to be one string reading "%d turn(s)", which is
	// the parenthetical plural nobody ever comes back to replace — and it now
	// appears on every row of the conversation list rather than only in the
	// picker, where it was easy not to look at.
	PickerTurnOne  string
	PickerTurnMany string

	// Resumed opens a continued conversation. It takes the session it came
	// from and how many turns, in that order.
	Resumed string
}

// catalogue is embedded, not loaded.
//
// A file per language would make translation contributions easier and would
// create a runtime load — which is a new failure mode in a product whose
// interface has to work before any configuration has been read. It is also what
// keeps the static, cgo-free binary of ADR-01 whole.
var catalogue = map[Lang]Strings{
	En: {
		Working:                "working",
		WorkingInterrupt:       "^C interrupts",
		LineOne:                "line",
		LineMany:               "lines",
		ExpandHint:             "Tab expands",
		ChildOne:               "child",
		ChildMany:              "children",
		ChildOwns:              "owns",
		ChildNoAnswer:          "no answer",
		ChildUnnamed:           "a child",
		PanelRounds:            "round",
		PanelInFlight:          "in flight",
		RailHidden:             "sidebar",
		RailFiles:              "files",
		RailSessions:           "sessions",
		RailFilter:             "/",
		RailNaming:             "naming · esc cancels",
		RailNoMatch:            "nothing matches",
		SideDiff:               "diff",
		SideSession:            "session",
		SideNothingYet:         "nothing changed yet",
		SideContext:            "context",
		SideAllowed:            "allowed",
		SideRecent:             "recent",
		SideBarScale:           "bars scaled to",
		LaneYou:                "you",
		LaneProcess:            "work",
		LaneAnswer:             "answer",
		NavBadge:               "nav",
		NavSessions:            "sessions",
		NavColumn:              "column",
		NavKeys:                "keys",
		NavEnter:               "browse",
		NavMove:                "move",
		NavOpen:                "open",
		NavPrompt:              "prompt",
		NavLeave:               "leave",
		SideToolOne:            "call",
		SideToolMany:           "calls",
		Loading:                "reading the conversation",
		LoadedOne:              "line",
		LoadedMany:             "lines",
		ContextFilling:         "the context is %d%% of the way to a summary",
		Compacted:              "earlier history was summarised",
		CompactedCount:         "%d earlier messages were summarised; %d kept",
		ApprovalCrosses:        "crosses:",
		ApprovalNetwork:        "Commands in this project may reach the network.",
		ApprovalEnter:          "enter denies",
		ShellHint:              "! runs here, unsent — the model reads the output",
		LeavingTakesTwo:        "^C again to leave",
		UpdateApplied:          "updated %s to %s. Reopen dcode to run it: this one is still the old binary.",
		UpdateCurrent:          "dcode %s is the latest release.",
		UpdateUnavailable:      "this build cannot update itself.",
		ApprovalCrossing:       "approve",
		ApprovalRule:           "rule:",
		ApprovalAnswered:       "you answered:",
		ApprovalDenied:         "denied",
		ApprovalAllowedOnce:    "allowed, once",
		ApprovalAllowedSession: "allowed for this session",
		ApprovalAllowedProject: "allowed for this project",
		ApprovalAllowedAlways:  "allowed always",
		KeyDeny:                "deny",
		KeyAllow:               "allow",
		KeySession:             "whole session",
		KeyNo:                  "no",
		KeyOnce:                "once",
		KeyProject:             "this project",
		KeyAlways:              "always",
		PanelPlan:              "PLAN",
		PlanOf:                 "%d of %d",
		PlanBlockedCount:       "(%d blocked)",
		SessionsMoreOne:        "more below",
		SessionsMoreMany:       "more below",
		SessionsKeys:           "up/down choose, enter opens, r renames, esc closes",
		RailTouchedOne:         "touched",
		RailTouchedMany:        "touched",

		VerifiedLabel:    "verified",
		NotVerifiedLabel: "NOT VERIFIED",
		UnverifiedLabel:  "unverified",

		VerifiedSummary:     "verified — %d %s passed",
		NotVerifiedSummary:  "NOT verified — %s did not pass",
		NothingCouldCheck:   "not verified — nothing here could check this",
		ChangedAfterCheck:   "not verified — files changed after the last check",
		CompletionMet:       "met",
		CompletionUnmet:     "not met",
		CompletionUnchecked: "could not be checked",
		CompletionMeasure:   "measurement changed",

		ApprovalDeny:         "deny",
		ApprovalAllowOnce:    "allow once",
		ApprovalAllowSession: "allow for the session",
		ApprovalEnterDenies:  "Enter denies.",
		ApprovalHeading:      "Approvals",

		HelpCommands:     "Commands",
		HelpApprovals:    "Approvals",
		HelpKeys:         "Keys",
		HelpYours:        "Yours",
		KeyEnter:         "send (queues while a turn is running)",
		KeyNewline:       "break the line without sending",
		KeyPasteImage:    "paste an image from the clipboard",
		CmdUndo:          "put back the files the last turn changed",
		CmdUpdate:        "install the latest release",
		CmdImage:         "show the model a picture with your next message",
		CmdImageArgs:     "<path>",
		ImageUsage:       "Usage: /image <path to a png, jpeg, gif or webp>",
		ImageAttached:    "attached %s — %d image(s) will go with your next message",
		ImageFailed:      "could not attach it:",
		ImageUnsupported: "%s does not read pictures. Switch with /model to one that does.",
		ImagePasted:      "pasted the image — %d will go with your next message",
		ImageTooBig:      "that image is %dMB and the limit is %dMB",
		ClipboardEmpty:   "no image on the clipboard — copy one, or use /image <path>",
		ClipboardMissing: "this machine has no clipboard tool; install wl-clipboard or xclip, or use /image <path>",
		UndoRestored:     "put back:",
		UndoRefused:      "left alone, changed since the turn:",
		UndoNothing:      "the last turn changed no files",
		UndoFailed:       "could not undo:",
		KeyArrows:        "history on an empty line, otherwise move through the stream",
		KeyPage:          "scroll · Home and End jump to either end",
		KeyTab:           "expand or collapse the selected entry",
		KeyEsc:           "close the expansion, then the selection",
		KeyPanel:         "show or hide the plan panel",
		KeyDequeue:       "remove the oldest queued message",
		KeyEditing:       "start, end, delete word, clear, cut to end",
		KeyInterrupt:     "interrupt the turn, or quit when idle",
		KeyQuit:          "quit",
		CmdHelp:          "shortcuts and commands",
		CmdInit:          "write DCODE.md for this workspace from what is already here",
		CmdClear:         "end this session and open a fresh one",
		CmdPlan:          "show the plan; with an argument, ask for a new one",
		CmdPlanArgs:      "[what to change]",
		CmdConfig:        "the effective value of a key and where it came from",
		CmdConfigArgs:    "<key>",
		CmdModel:         "switch model — opens a new session, since the prefix changes",
		CmdModelArgs:     "<name>",
		CmdResume:        "list sessions, or reattach to one",
		CmdResumeArgs:    "[id]",
		CmdMode:          "show the current mode, or switch to plan/assist/auto — shift+tab cycles",
		CmdModeArgs:      "[plan|assist|auto]",
		CmdModeCurrent:   "current mode: %s",
		CmdModeUnnamed:   "this session's boundary is not one of the three modes; /mode plan, assist or auto picks one",
		CmdModeUnknown:   "%s is not a mode — want plan, assist or auto",

		Usage: `dcode %s — an agentic coding harness

Usage:
  dcode                      open the terminal interface
  dcode [flags] <task>       run one task and exit
  dcode serve [flags]        run the daemon
  dcode tui [flags]          open the terminal interface
  dcode login [flags]        store the model credential, read without echo
  dcode config [key]         the effective configuration and where it came from
  dcode sessions [show <id>] what dcode has done here, and what it did
  dcode -c                   continue the most recent session here
  dcode -r                   choose which session to continue
  dcode update [flags]       install the latest release

Examples:
  dcode
  dcode "add a test for the parser"
  dcode --dump-prompt
  dcode --config model.name
  dcode serve &  dcode tui

Run a subcommand with --help for its flags.

Environment:
  DCODE_API_KEY            model credential; overrides anything stored
  DCODE_MODEL              model name (default MiniMax-M3)
  DCODE_TRANSPORT          wire format: openai or anthropic
  DCODE_SANDBOX_MODE       read-only, workspace-write or full-access
  DCODE_APPROVAL_POLICY    untrusted, on-request or never
  DCODE_ALLOW_NETWORK      grant network without asking
  DCODE_LANG               interface language: pt-BR or en
  DCODE_HOME               configuration root (default ~/.dcode)
  DCODE_SOCKET             daemon socket path
`,

		CopySelected:    "%d line(s) selected",
		CopyKeys:        "↑ ↓ extend · y copy · esc leave",
		CopyDone:        "copied to the clipboard",
		CopyEmpty:       "nothing selected",
		Interrupt:       "esc interrupts",
		Queued:          "queued",
		BarFiles:        "files",
		BarWaiting:      "wait",
		Resumed:         "continuing %s — %d turn(s) from before",
		PickerTitle:     "Continue a conversation",
		PickerKeys:      "↑↓ move · enter continue · esc start fresh",
		PickerEmpty:     "nothing recorded in this workspace yet",
		PickerUntitled:  "(nothing asked yet)",
		PickerYesterday: "yesterday",
		PickerTurnOne:   "turn",
		PickerTurnMany:  "turns",
		NoPlan:          "There is no plan yet.",
	},
	PtBR: {
		Working:                "trabalhando",
		WorkingInterrupt:       "^C interrompe",
		LineOne:                "linha",
		LineMany:               "linhas",
		ExpandHint:             "Tab expande",
		ChildOne:               "filho",
		ChildMany:              "filhos",
		ChildOwns:              "possui",
		ChildNoAnswer:          "sem resposta",
		ChildUnnamed:           "um filho",
		PanelRounds:            "iteração",
		PanelInFlight:          "em vôo",
		RailHidden:             "coluna",
		RailFiles:              "arquivos",
		RailSessions:           "sessões",
		RailFilter:             "/",
		RailNaming:             "nomeando · esc cancela",
		RailNoMatch:            "nada corresponde",
		SideDiff:               "diff",
		SideSession:            "sessão",
		SideNothingYet:         "nada mudou ainda",
		SideContext:            "contexto",
		SideAllowed:            "permitido",
		SideRecent:             "recentes",
		SideBarScale:           "barras na escala de",
		LaneYou:                "você",
		LaneProcess:            "trabalho",
		LaneAnswer:             "resposta",
		NavBadge:               "nav",
		NavSessions:            "sessões",
		NavColumn:              "coluna",
		NavKeys:                "teclas",
		NavEnter:               "navegar",
		NavMove:                "mover",
		NavOpen:                "abrir",
		NavPrompt:              "escrever",
		NavLeave:               "sair",
		SideToolOne:            "chamada",
		SideToolMany:           "chamadas",
		Loading:                "lendo a conversa",
		LoadedOne:              "linha",
		LoadedMany:             "linhas",
		ContextFilling:         "o contexto está a %d%% do ponto em que a conversa é resumida",
		Compacted:              "o histórico anterior foi resumido",
		CompactedCount:         "%d mensagens anteriores foram resumidas; %d mantidas",
		ApprovalCrosses:        "cruza:",
		ApprovalNetwork:        "Comandos deste projeto podem alcançar a rede.",
		ApprovalEnter:          "enter nega",
		ShellHint:              "! roda aqui, sem enviar — o modelo lê a saída",
		LeavingTakesTwo:        "^C de novo para sair",
		UpdateApplied:          "atualizado de %s para %s. Reabra o dcode para rodar: este ainda é o binário antigo.",
		UpdateCurrent:          "dcode %s é a release mais recente.",
		UpdateUnavailable:      "este build não se atualiza sozinho.",
		ApprovalCrossing:       "aprovar",
		ApprovalRule:           "regra:",
		ApprovalAnswered:       "você respondeu:",
		ApprovalDenied:         "negado",
		ApprovalAllowedOnce:    "permitido, uma vez",
		ApprovalAllowedSession: "permitido nesta sessão",
		ApprovalAllowedProject: "permitido neste projeto",
		ApprovalAllowedAlways:  "permitido sempre",
		KeyDeny:                "negar",
		KeyAllow:               "permitir",
		KeySession:             "sessão inteira",
		KeyNo:                  "não",
		KeyOnce:                "uma vez",
		KeyProject:             "este projeto",
		KeyAlways:              "sempre",
		PanelPlan:              "PLANO",
		PlanOf:                 "%d de %d",
		PlanBlockedCount:       "(%d bloqueado)",
		SessionsMoreOne:        "abaixo",
		SessionsMoreMany:       "abaixo",
		SessionsKeys:           "cima/baixo escolhe, enter abre, r renomeia, esc fecha",
		RailTouchedOne:         "tocado",
		RailTouchedMany:        "tocados",

		VerifiedLabel:    "conferido",
		NotVerifiedLabel: "NÃO CONFERIDO",
		UnverifiedLabel:  "sem conferir",

		VerifiedSummary:     "conferido — %d %s passaram",
		NotVerifiedSummary:  "NÃO conferido — %s não passou",
		NothingCouldCheck:   "sem conferir — não há aqui como conferir isto",
		ChangedAfterCheck:   "sem conferir — arquivos mudaram depois da última conferência",
		CompletionMet:       "cumprido",
		CompletionUnmet:     "não cumprido",
		CompletionUnchecked: "não pôde ser conferido",
		CompletionMeasure:   "medição alterada",

		ApprovalDeny:         "negar",
		ApprovalAllowOnce:    "permitir uma vez",
		ApprovalAllowSession: "permitir na sessão",
		ApprovalEnterDenies:  "Enter nega.",
		ApprovalHeading:      "Aprovações",

		HelpCommands:     "Comandos",
		HelpApprovals:    "Aprovações",
		HelpKeys:         "Teclas",
		HelpYours:        "Seus",
		KeyEnter:         "envia (enfileira enquanto um turno roda)",
		KeyNewline:       "quebra a linha sem enviar",
		KeyPasteImage:    "cola uma imagem da área de transferência",
		CmdUndo:          "restaura os arquivos que o último turno mudou",
		CmdUpdate:        "instala a release mais recente",
		CmdImage:         "mostra uma imagem ao modelo junto da próxima mensagem",
		CmdImageArgs:     "<caminho>",
		ImageUsage:       "Uso: /image <caminho de png, jpeg, gif ou webp>",
		ImageAttached:    "anexado %s — %d imagem(ns) vão com a próxima mensagem",
		ImageFailed:      "não foi possível anexar:",
		ImageUnsupported: "%s não lê imagem. Troque com /model para um que leia.",
		ImagePasted:      "imagem colada — %d vão com a próxima mensagem",
		ImageTooBig:      "essa imagem tem %dMB e o limite é %dMB",
		ClipboardEmpty:   "nenhuma imagem na área de transferência — copie uma, ou use /image <caminho>",
		ClipboardMissing: "esta máquina não tem ferramenta de clipboard; instale wl-clipboard ou xclip, ou use /image <caminho>",
		UndoRestored:     "restaurado:",
		UndoRefused:      "intocado, mudou depois do turno:",
		UndoNothing:      "o último turno não mexeu em arquivo nenhum",
		UndoFailed:       "não foi possível desfazer:",
		KeyArrows:        "histórico em linha vazia; fora dela, navega no stream",
		KeyPage:          "rola · Home e End vão para as pontas",
		KeyTab:           "abre ou fecha a entrada selecionada",
		KeyEsc:           "fecha a expansão, depois a seleção",
		KeyPanel:         "mostra ou esconde o painel de plano",
		KeyDequeue:       "descarta a mensagem mais antiga da fila",
		KeyEditing:       "início, fim, apaga palavra, limpa, corta até o fim",
		KeyInterrupt:     "interrompe o turno, ou sai quando ocioso",
		KeyQuit:          "sai",
		CmdHelp:          "atalhos e comandos",
		CmdInit:          "escreve o DCODE.md deste workspace a partir do que já existe",
		CmdClear:         "encerra esta sessão e abre uma nova",
		CmdPlan:          "mostra o plano; com argumento, pede um novo",
		CmdPlanArgs:      "[o que mudar]",
		CmdConfig:        "o valor efetivo de uma chave e de onde ele veio",
		CmdConfigArgs:    "<chave>",
		CmdModel:         "troca de modelo — abre nova sessão, porque o prefixo muda",
		CmdModelArgs:     "<nome>",
		CmdResume:        "lista sessões, ou reconecta a uma",
		CmdResumeArgs:    "[id]",
		CmdMode:          "mostra o modo atual, ou troca para plan/assist/auto — shift+tab cicla",
		CmdModeArgs:      "[plan|assist|auto]",
		CmdModeCurrent:   "modo atual: %s",
		CmdModeUnnamed:   "o limite desta sessão não é nenhum dos três modos; /mode plan, assist ou auto escolhe um",
		CmdModeUnknown:   "%s não é um modo — use plan, assist ou auto",

		Usage: `dcode %s — um harness de programação agêntica

Uso:
  dcode                      abre a interface de terminal
  dcode [flags] <tarefa>     roda uma tarefa e sai
  dcode serve [flags]        roda o daemon
  dcode tui [flags]          abre a interface de terminal
  dcode login [flags]        guarda a credencial do modelo, lida sem eco
  dcode config [chave]       a configuração efetiva e de onde ela veio
  dcode sessions [show <id>] o que o dcode fez aqui, e o que ele fez
  dcode -c                   continua a sessão mais recente daqui
  dcode -r                   escolhe qual sessão continuar
  dcode update [flags]       instala a última versão

Exemplos:
  dcode
  dcode "adicione um teste para o parser"
  dcode --dump-prompt
  dcode --config model.name
  dcode serve &  dcode tui

Rode um subcomando com --help para ver as flags dele.

Ambiente:
  DCODE_API_KEY            credencial do modelo; vence qualquer uma guardada
  DCODE_MODEL              nome do modelo (default MiniMax-M3)
  DCODE_TRANSPORT          formato de fio: openai ou anthropic
  DCODE_SANDBOX_MODE       read-only, workspace-write ou full-access
  DCODE_APPROVAL_POLICY    untrusted, on-request ou never
  DCODE_ALLOW_NETWORK      concede rede sem perguntar
  DCODE_LANG               idioma da interface: pt-BR ou en
  DCODE_HOME               raiz de configuração (default ~/.dcode)
  DCODE_SOCKET             caminho do socket do daemon
`,

		CopySelected:    "%d linha(s) selecionada(s)",
		CopyKeys:        "↑ ↓ estende · y copia · esc sai",
		CopyDone:        "copiado para a área de transferência",
		CopyEmpty:       "nada selecionado",
		Interrupt:       "esc interrompe",
		Queued:          "na fila",
		BarFiles:        "arq",
		BarWaiting:      "espera",
		Resumed:         "continuando %s — %d turno(s) de antes",
		PickerTitle:     "Continuar uma conversa",
		PickerKeys:      "↑↓ move · enter continua · esc começa do zero",
		PickerEmpty:     "nada gravado neste workspace ainda",
		PickerUntitled:  "(nada perguntado ainda)",
		PickerYesterday: "ontem",
		PickerTurnOne:   "turno",
		PickerTurnMany:  "turnos",
		NoPlan:          "Ainda não há plano.",
	},
}

// Text returns the catalogue for a language.
//
// The language lives on the Model and nowhere else. Carrying it on Geometry as
// well would be two sources for one fact, and the first thing that produces is
// a status line in one language and a report in another.
//
// An undeclared language returns the fallback rather than an empty struct: a
// blank interface is the one outcome worse than the wrong language.
func Text(l Lang) Strings {
	if s, ok := catalogue[l]; ok {
		return s
	}
	return catalogue[Fallback]
}

// Languages are the declared languages, for the coverage guard.
func Languages() []Lang { return []Lang{PtBR, En} }
