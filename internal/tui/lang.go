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
	// Status line
	VerifiedLabel    string
	NotVerifiedLabel string
	UnverifiedLabel  string
	ContextLeft      string

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
	HelpCommands  string
	HelpApprovals string
	HelpKeys      string
	HelpYours     string
	KeyEnter      string
	KeyArrows     string
	KeyPage       string
	KeyTab        string
	KeyEsc        string
	KeyPanel      string
	KeyDequeue    string
	KeyEditing    string
	KeyInterrupt  string
	KeyQuit       string
	CmdHelp       string
	CmdInit       string
	CmdClear      string
	CmdPlan       string
	CmdPlanArgs   string
	CmdConfig     string
	CmdConfigArgs string
	CmdModel      string
	CmdModelArgs  string
	CmdResume     string
	CmdResumeArgs string

	// Empty state and general
	EmptyHint      string
	Interrupt      string
	Queued         string
	NoPlan         string
	SessionCleared string
}

// catalogue is embedded, not loaded.
//
// A file per language would make translation contributions easier and would
// create a runtime load — which is a new failure mode in a product whose
// interface has to work before any configuration has been read. It is also what
// keeps the static, cgo-free binary of ADR-01 whole.
var catalogue = map[Lang]Strings{
	En: {
		VerifiedLabel:    "verified",
		NotVerifiedLabel: "NOT VERIFIED",
		UnverifiedLabel:  "unverified",
		ContextLeft:      "context",

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

		HelpCommands:  "Commands",
		HelpApprovals: "Approvals",
		HelpKeys:      "Keys",
		HelpYours:     "Yours",
		KeyEnter:      "send (queues while a turn is running)",
		KeyArrows:     "history on an empty line, otherwise move through the stream",
		KeyPage:       "scroll · Home and End jump to either end",
		KeyTab:        "expand or collapse the selected entry",
		KeyEsc:        "close the expansion, then the selection",
		KeyPanel:      "show or hide the plan panel",
		KeyDequeue:    "remove the oldest queued message",
		KeyEditing:    "start, end, delete word, clear, cut to end",
		KeyInterrupt:  "interrupt the turn, or quit when idle",
		KeyQuit:       "quit",
		CmdHelp:       "shortcuts and commands",
		CmdInit:       "write DCODE.md for this workspace from what is already here",
		CmdClear:      "end this session and open a fresh one",
		CmdPlan:       "show the plan; with an argument, ask for a new one",
		CmdPlanArgs:   "[what to change]",
		CmdConfig:     "the effective value of a key and where it came from",
		CmdConfigArgs: "<key>",
		CmdModel:      "switch model — opens a new session, since the prefix changes",
		CmdModelArgs:  "<name>",
		CmdResume:     "list sessions, or reattach to one",
		CmdResumeArgs: "[id]",

		EmptyHint:      "Ask for something, or type / for commands.",
		Interrupt:      "esc interrupts",
		Queued:         "queued",
		NoPlan:         "There is no plan yet.",
		SessionCleared: "New session started.",
	},
	PtBR: {
		VerifiedLabel:    "conferido",
		NotVerifiedLabel: "NÃO CONFERIDO",
		UnverifiedLabel:  "sem conferir",
		ContextLeft:      "contexto",

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

		HelpCommands:  "Comandos",
		HelpApprovals: "Aprovações",
		HelpKeys:      "Teclas",
		HelpYours:     "Seus",
		KeyEnter:      "envia (enfileira enquanto um turno roda)",
		KeyArrows:     "histórico em linha vazia; fora dela, navega no stream",
		KeyPage:       "rola · Home e End vão para as pontas",
		KeyTab:        "abre ou fecha a entrada selecionada",
		KeyEsc:        "fecha a expansão, depois a seleção",
		KeyPanel:      "mostra ou esconde o painel de plano",
		KeyDequeue:    "descarta a mensagem mais antiga da fila",
		KeyEditing:    "início, fim, apaga palavra, limpa, corta até o fim",
		KeyInterrupt:  "interrompe o turno, ou sai quando ocioso",
		KeyQuit:       "sai",
		CmdHelp:       "atalhos e comandos",
		CmdInit:       "escreve o DCODE.md deste workspace a partir do que já existe",
		CmdClear:      "encerra esta sessão e abre uma nova",
		CmdPlan:       "mostra o plano; com argumento, pede um novo",
		CmdPlanArgs:   "[o que mudar]",
		CmdConfig:     "o valor efetivo de uma chave e de onde ele veio",
		CmdConfigArgs: "<chave>",
		CmdModel:      "troca de modelo — abre nova sessão, porque o prefixo muda",
		CmdModelArgs:  "<nome>",
		CmdResume:     "lista sessões, ou reconecta a uma",
		CmdResumeArgs: "[id]",

		EmptyHint:      "Peça alguma coisa, ou digite / para ver os comandos.",
		Interrupt:      "esc interrompe",
		Queued:         "na fila",
		NoPlan:         "Ainda não há plano.",
		SessionCleared: "Nova sessão iniciada.",
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
