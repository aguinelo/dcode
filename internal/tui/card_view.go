package tui

import (
	"fmt"
	"strings"

	"github.com/aguinelo/dcode/internal/protocol"
)

// The question card, drawn in the transcript as text (orchestration .p §5.5):
// the header and "1 de N" on the first row, the question, the summary and the
// evidence of a merge, the options numbered with their descriptions — the
// selected one marked by a character, the recommended one said — and, while
// the card waits, the line that says it needs the person. Inline, like the
// approval: the transcript keeps what was asked and what was answered.

// cardStrings are the card's words in each language. Their own table: the
// card is drawn from this file alone, and the wire's labels stay as sent.
type cardStrings struct {
	Of, Recommended, Other, OtherHint, YouAsked, Needs, PendingOne, PendingMany string
	KeyHint, KeyHintTyping, Answered, Skipped, Relayed, Review, Gate            string
	OtherEmpty, Failed, Refused, Note                                           string
}

var cardTexts = map[Lang]cardStrings{
	PtBR: {
		Of: "de", Recommended: "Recomendado", Other: "Outro", OtherHint: "nas suas palavras: vão à orquestradora",
		YouAsked: "Você pediu", Needs: "Precisa da sua contribuição", PendingOne: "pendente", PendingMany: "pendentes",
		KeyHint:       "↑↓ escolhe · 1-9 marca · ←→ itens · s pula · enter próximo · a recomendação a todos · esc solta",
		KeyHintTyping: "digite a resposta · ↑↓ escolhe · enter próximo · esc solta",
		Answered:      "respondida", Skipped: "pulada: continua esperando",
		Relayed: "suas palavras foram à orquestradora; a pergunta espera a proposta dela",
		Review:  "revisão", Gate: "gate",
		OtherEmpty: "escreva a resposta em Outro, ou escolha uma opção: %s",
		Failed:     "as respostas do cartão não foram enviadas: %v",
		Refused:    "a resposta a %s foi recusada: %s — %s",
		Note:       "nota",
	},
	En: {
		Of: "of", Recommended: "Recommended", Other: "Other", OtherHint: "in your words: they go to the orchestrator",
		YouAsked: "You asked", Needs: "Needs your input", PendingOne: "pending", PendingMany: "pending",
		KeyHint:       "up/down choose · 1-9 mark · left/right items · s skips · enter next · a recommendation to all · esc lets go",
		KeyHintTyping: "type the answer · up/down choose · enter next · esc lets go",
		Answered:      "answered", Skipped: "skipped: still waiting",
		Relayed: "your words went to the orchestrator; the question waits for its proposal",
		Review:  "review", Gate: "gate",
		OtherEmpty: "write the answer in Other, or choose an option: %s",
		Failed:     "the card's answers did not go through: %v",
		Refused:    "the answer to %s was refused: %s — %s",
		Note:       "note",
	},
}

func cardText(l Lang) cardStrings {
	if t, ok := cardTexts[l]; ok {
		return t
	}
	return cardTexts[En]
}

// optionLabel is an option in the person's language where this client knows
// the answer, as the daemon wrote it otherwise.
func optionLabel(it protocol.CardItem, id string, lang Lang) string {
	if id == protocol.AnswerOther {
		return cardText(lang).Other
	}
	return answerLabel(it.Question, id, Text(lang))
}

// renderCardBlock draws a card entry. The focus is the keyboard's, when ^Q
// put it on this card.
func renderCardBlock(e Entry, focus *CardFocus, cursor, cont string, g Geometry, lang Lang, w int) []string {
	c := e.Card
	if c == nil {
		return nil
	}
	p := g.Palette
	t := cardText(lang)
	gl := railGlyphs(g.Unicode)
	am := agentGlyphs(g.Unicode)
	inner := maxInt(8, w-4)
	var out []string
	line := func(prefix, s string) { out = append(out, clipStyled(prefix+s, w)) }
	text := func(style Style, s string, indent string) {
		for _, l := range wrap(s, maxInt(4, inner-visibleWidth(indent))) {
			line(cont, indent+p.Apply(style, l))
		}
	}

	if !cardNeedsPerson(c) {
		line(cursor, p.Apply(StyleMeta, am.ask+" "+c.Title))
		for _, it := range c.Items {
			if it.State != protocol.CardItemAnswered {
				continue
			}
			answer := t.Answered
			if it.Answer != "" {
				answer = optionLabel(it, it.Answer, lang)
			}
			label := it.Header
			if label == "" {
				label = it.Text
			}
			line(cont, "  "+p.Apply(StyleOK, gl.done)+" "+p.Apply(StyleMeta, label)+" — "+p.Apply(StyleBold, answer))
		}
		return out
	}

	f := focus
	if f == nil || f.CardID != c.ID {
		f = focusOn(c)
		f.Results = nil
	}
	it, _ := f.item(c)
	if c.Proposed && c.From != "" {
		text(StyleProse, t.YouAsked+": "+c.From, "")
		if c.Why != "" {
			text(StyleMeta, c.Why, "  ")
		}
	}
	head := am.ask + " " + c.Kind
	if it.Header != "" {
		head += " " + gl.dot + " " + it.Header
	}
	count := fmt.Sprintf("%d %s %d", min(f.Index, len(c.Items)-1)+1, t.Of, len(c.Items))
	line(cursor, rightAlign(p.Apply(StyleWarn, head), p.Apply(StyleMeta, count), inner))
	text(StyleBold, it.Text, "  ")
	for _, l := range strings.Split(it.Summary, "\n") {
		if strings.TrimSpace(l) != "" {
			text(StyleMeta, l, "  ")
		}
	}
	if ev := it.Evidence; ev != nil {
		commit := ev.Commit
		if len(commit) > 7 {
			commit = commit[:7]
		}
		line(cont, "  "+p.Apply(StyleMeta, strings.Join([]string{t.Review + " " + ev.Verdict, t.Gate + " " + ev.Gate, ev.Command, commit}, " "+gl.dot+" ")))
	}
	if len(it.Sensitive) > 0 {
		text(StyleWarn, Text(lang).AskSensitive+": "+strings.Join(it.Sensitive, ", "), "  ")
	}

	pick := f.pick(it)
	focused := focus != nil && focus.CardID == c.ID
	if itemOpen(it) {
		desc := map[string]string{protocol.AnswerOther: t.OtherHint}
		for _, o := range it.Options {
			desc[o.ID] = o.Description
		}
		for n, id := range cardChoices(it) {
			mark, style := "( )", StyleProse
			if id == pick.answer {
				mark, style = "(•)", StyleAccent
				if !g.Unicode {
					mark = "(x)"
				}
			}
			lead := "  "
			if focused && id == pick.answer {
				lead = p.Apply(StyleAccent, gl.cursor) + " "
			}
			label := optionLabel(it, id, lang)
			if id == it.Recommended {
				label += " (" + t.Recommended + ")"
			}
			if id == protocol.AnswerOther && pick.answer == protocol.AnswerOther {
				label += ": " + pick.note
				if focused {
					label += gl.caret
				}
			}
			line(cont, lead+p.Apply(style, fmt.Sprintf("%d. %s ", n+1, mark))+p.Apply(StyleBold, label))
			if d := desc[id]; d != "" {
				text(StyleMeta, d, "        ")
			}
		}
		if pick.answer != protocol.AnswerOther && pick.note != "" {
			text(StyleMeta, t.Note+": "+pick.note, "  ")
		}
	} else {
		line(cont, "  "+p.Apply(StyleOK, gl.done)+" "+p.Apply(StyleBold, optionLabel(it, it.Answer, lang)))
	}
	switch {
	case it.State == protocol.CardItemRelayed:
		text(StyleMeta, t.Relayed, "  ")
	case contains(f.Skipped, it.ID):
		text(StyleMeta, t.Skipped, "  ")
	}
	if r, ok := f.Results[it.ID]; ok && r.Status == protocol.AnswerRefused {
		text(StyleError, "! "+r.Code+": "+r.Message, "  ")
	}
	if focused {
		keys := t.KeyHint
		if it.AllowOther && pick.answer == protocol.AnswerOther {
			keys = t.KeyHintTyping
		}
		text(StyleHint, keys, "  ")
	}
	pending := t.PendingMany
	if c.Pending == 1 {
		pending = t.PendingOne
	}
	line(cont, p.Apply(StyleWarn, fmt.Sprintf("%s %s %s %d %s", am.ask, t.Needs, gl.dot, c.Pending, pending)))
	return out
}

func contains(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}
