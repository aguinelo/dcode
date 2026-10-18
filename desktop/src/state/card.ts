// The orchestrator's question card (orchestration .p §5.5, D67): the
// questions of one kind that wait together, shown in the conversation one
// item at a time, each with its options, the selected one checked, and room
// for the person's own words. What the person picks here is a draft until
// they confirm it: one POST with every answer and the items they skipped.
// Pure, like the rest of state/: the card comes from the wire, the draft from
// what the person did, and nothing here reads a clock or the DOM.

import type * as P from '../protocol/generated';
import { AnswerOther, AnswerRefused, CardItemAnswered, CardItemRelayed, CardItemSkipped } from '../protocol/generated';
import { NOTE_LIMIT } from './questions';
import type { SessionView } from './session';

export { NOTE_LIMIT };

/** One item's pick: an option's id, or AnswerOther with the person's words as the note. */
export interface CardPick {
  answer: string;
  note: string;
}

/** What the person did with a card before confirming it. */
export interface CardDraft {
  cardId: string;
  /** The item shown, by its place in the card. */
  index: number;
  /** The picks the person changed, by question id; an item not here keeps its `selected`. */
  picks: Readonly<Record<string, CardPick>>;
  /** The items the person passed over, by question id: they keep waiting. */
  skipped: readonly string[];
  /** The items the person was shown: only those are answered on confirm. */
  seen: readonly string[];
}

/** Whether an item still takes an answer: its question has none yet. */
export const isOpen = (item: P.CardItem): boolean => item.state !== CardItemAnswered;

/** Whether the card still needs the person (the selo "Precisa da sua contribuição"). */
export const needsPerson = (card: P.QuestionCard): boolean => !card.closed && card.pending > 0;

const uniq = (ids: readonly string[], id: string): string[] => (ids.includes(id) ? [...ids] : [...ids, id]);

/** A card opened: on its first item that still waits, nothing changed yet. */
export function startDraft(card: P.QuestionCard): CardDraft {
  const first = Math.max(0, card.items.findIndex(isOpen));
  const shown = card.items[first];
  return { cardId: card.id, index: first, picks: {}, skipped: [], seen: shown ? [shown.id] : [] };
}

/** The item shown, if the card has any. */
export const currentItem = (card: P.QuestionCard, d: CardDraft): P.CardItem | undefined => card.items[Math.min(d.index, card.items.length - 1)];

/** Whether the item shown is the card's last: there "Próximo" confirms. */
export const isLast = (card: P.QuestionCard, d: CardDraft): boolean => d.index >= card.items.length - 1;

/**
 * An item's pick: what the person chose, or what the daemon selected — the
 * recommendation, or on a proposed card the answer the orchestrator read in
 * the person's words, with its note. A relayed item keeps the words the
 * person wrote.
 */
export function pickOf(d: CardDraft, item: P.CardItem): CardPick {
  const changed = d.picks[item.id];
  if (changed) return changed;
  if (item.state === CardItemRelayed && item.other) return { answer: AnswerOther, note: item.other };
  return { answer: item.selected || item.recommended, note: item.selected_note ?? '' };
}

/** Shows the item at `index`, clamped to the card. */
export function goTo(card: P.QuestionCard, d: CardDraft, index: number): CardDraft {
  const i = Math.max(0, Math.min(card.items.length - 1, index));
  const shown = card.items[i];
  return { ...d, index: i, seen: shown ? uniq(d.seen, shown.id) : d.seen };
}

/** The pick of the item shown changed: an option, or "Outro" with the words written so far. */
export function choose(card: P.QuestionCard, d: CardDraft, answer: string, note = ''): CardDraft {
  const item = currentItem(card, d);
  if (!item || !isOpen(item)) return d;
  return { ...d, picks: { ...d.picks, [item.id]: { answer, note } }, skipped: d.skipped.filter((id) => id !== item.id) };
}

/** The choices of an item, in the order ↑ and ↓ walk them: its options, then "Outro" when it takes words. */
export function choicesOf(item: P.CardItem): string[] {
  const ids = item.options.map((o) => o.id);
  return item.allow_other ? [...ids, AnswerOther] : ids;
}

/** ↑ or ↓ on the card: the pick moves to the choice before or after, staying inside the list. */
export function step(card: P.QuestionCard, d: CardDraft, by: number): CardDraft {
  const item = currentItem(card, d);
  if (!item || !isOpen(item)) return d;
  const choices = choicesOf(item);
  const now = pickOf(d, item);
  const at = choices.indexOf(now.answer);
  const next = choices[Math.max(0, Math.min(choices.length - 1, (at < 0 ? -1 : at) + by))];
  if (next === undefined || next === now.answer) return d;
  return choose(card, d, next, next === AnswerOther ? (d.picks[item.id]?.note ?? '') : '');
}

/** "Pular": the item shown keeps waiting, and the next one shows. */
export function skip(card: P.QuestionCard, d: CardDraft): CardDraft {
  const item = currentItem(card, d);
  const skipped = item && isOpen(item) ? uniq(d.skipped, item.id) : [...d.skipped];
  return goTo(card, { ...d, skipped }, d.index + 1);
}

/** Why the card cannot be confirmed as it is, or null: "Outro" without words is no answer. */
export function blocked(card: P.QuestionCard, d: CardDraft): string | null {
  for (const item of card.items) {
    if (!isOpen(item) || !d.seen.includes(item.id) || d.skipped.includes(item.id)) continue;
    const pick = pickOf(d, item);
    if (pick.answer === AnswerOther && !pick.note.trim()) return `Escreva a resposta em “Outro”, ou escolha uma opção: ${item.header || item.text}`;
    if (pick.note.length > NOTE_LIMIT) return `A resposta em “Outro” tem ${pick.note.length} caracteres, e vai até ${NOTE_LIMIT}: ${item.header || item.text}`;
  }
  return null;
}

/**
 * The confirm (.p §5.5): every item the person was shown and did not skip,
 * with its pick, in one POST; the skipped ones, and those never shown, go
 * in `skipped` and keep waiting. An item already answered is left out.
 */
export function confirmBatch(card: P.QuestionCard, d: CardDraft): P.AnswerQuestionsRequest {
  const answers: P.AnswerItem[] = [];
  const skipped: string[] = [];
  for (const item of card.items) {
    if (!isOpen(item)) continue;
    if (!d.seen.includes(item.id) || d.skipped.includes(item.id)) {
      skipped.push(item.id);
      continue;
    }
    const pick = pickOf(d, item);
    const note = pick.note.trim();
    answers.push(note ? { question_id: item.id, answer: pick.answer, note } : { question_id: item.id, answer: pick.answer });
  }
  return skipped.length > 0 ? { card_id: card.id, answers, skipped } : { card_id: card.id, answers };
}

/** "Aplicar a recomendação a todos": the same POST, the recommendation of every item still waiting. */
export function recommendAll(card: P.QuestionCard): P.AnswerQuestionsRequest {
  return { card_id: card.id, answers: card.items.filter(isOpen).map((i) => ({ question_id: i.id, answer: i.recommended })) };
}

/** The label of an option of an item, or the id when the item does not offer it. */
export function labelOf(item: P.Question, answer: string): string {
  if (answer === AnswerOther) return 'Outro';
  return item.options.find((o) => o.id === answer)?.label ?? answer;
}

/** What became of one answer, said under its item: refused in red with the daemon's words, the rest quiet. */
export function resultView(r: P.AnswerResult, refusal: (code: string, message: string) => string): { tone: string; text: string } {
  switch (r.status) {
    case AnswerRefused:
      return { tone: 'err', text: refusal(r.code ?? '', r.message ?? '') };
    case CardItemAnswered:
      return { tone: 'ok', text: 'Respondida.' };
    case CardItemRelayed:
      return { tone: 'dim', text: 'Suas palavras foram à orquestradora; a pergunta continua esperando a proposta dela.' };
    case CardItemSkipped:
      return { tone: 'dim', text: 'Pulada: a pergunta continua esperando.' };
    default:
      return { tone: 'err', text: `O daemon disse “${r.status}” desta resposta, que o app não conhece.` };
  }
}

/** The results of a batch, by question id: the last one of a question wins. */
export function resultsById(results: readonly P.AnswerResult[]): Record<string, P.AnswerResult> {
  const out: Record<string, P.AnswerResult> = {};
  for (const r of results) out[r.question_id] = r;
  return out;
}

/** The evidence of a merge in a line: the reviewer's verdict, the gate's result, its command and the commit measured. */
export function evidenceLine(e: P.QuestionEvidence): { verdict: string; gate: string; command: string; commit: string } {
  const findings = e.findings ? ` · ${e.findings} ${e.findings === 1 ? 'apontamento' : 'apontamentos'}` : '';
  return {
    verdict: `revisão: ${e.verdict}${findings}`,
    gate: `gate: ${e.gate}${e.exit_code !== 0 ? ` (saiu ${e.exit_code})` : ''}`,
    command: e.command,
    commit: e.commit.slice(0, 7),
  };
}

/**
 * The latest open card that holds one of the questions waiting, for "◆ N
 * perguntas" to take the person to it; null when none does, and the queue answers.
 */
export function cardFor(cards: readonly P.QuestionCard[], waiting: readonly P.Question[]): string | null {
  const ids = new Set(waiting.map((q) => q.id));
  // The latest first: a proposed card, made from the person's words, comes after the card it answers.
  const card = [...cards].reverse().find((c) => needsPerson(c) && c.items.some((i) => isOpen(i) && ids.has(i.id)));
  return card?.id ?? null;
}

/**
 * Cards as the daemon holds them (D67): each one changes its entry in place,
 * where it was first said, unless the entry already holds a newer version; a
 * card the flow has not seen is drawn at the end — read from /questions, or
 * from the answer of a confirm.
 */
export function withCards(view: SessionView, cards: readonly P.QuestionCard[], seq = view.lastSeq, at = view.lastAt ?? ''): SessionView {
  let entries = view.entries;
  for (const card of cards) {
    const i = entries.findIndex((e) => e.kind === 'card' && e.card.id === card.id);
    if (i < 0) {
      entries = [...entries, { kind: 'card', seq, at: at || card.created, card }];
      continue;
    }
    const current = entries[i];
    if (current?.kind !== 'card' || Date.parse(current.card.updated) > Date.parse(card.updated)) continue;
    entries = entries.slice();
    entries[i] = { ...current, card };
  }
  return entries === view.entries ? view : { ...view, entries };
}

/** The cards of a conversation, in the order they were first said. */
export function cardsOf(view: SessionView): P.QuestionCard[] {
  return view.entries.flatMap((e) => (e.kind === 'card' ? [e.card] : []));
}
