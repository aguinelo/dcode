import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import type * as P from '../../src/protocol/generated';
import { decodeQuestions } from '../../src/protocol/projects';
import { FlowBlocks } from '../../src/renderer/Flow';
import { cardsOf, withCards } from '../../src/state/card';
import { flowBlocks } from '../../src/state/flow';
import { applyRaw, emptySessions } from '../../src/state/sessions';
import type { SessionView } from '../../src/state/session';
import recorded from './live-question-cards.json';

// The question card of a real orchestrator (D67), read from the daemon that
// ran it (0.24.0-dev+ff7a8e18): three children reach their merge question one
// after the other, the harness keeps one card "c1" and sends it six times as
// it grows and as the orchestrator writes each summary — and after the first
// version come three more turns of reports and replies. The payloads are the
// daemon's own, byte for byte; only the reasoning and the tool calls were left
// out, the deltas of a turn joined, and the seqs counted again from 1.
//
// The card was in the flow, at its first question.card, under everything that
// came after: the window, at the end of the conversation, showed the
// orchestrator saying the merges waited for the person and no card to answer.

const ID = '1a14f751ca5d7c3a40c';
const stream = recorded as Array<{ seq: number; type: string; payload: unknown }>;

const fold = (events: readonly unknown[] = stream): SessionView => {
  const s = applyRaw(emptySessions, events);
  expect(s.problems).toEqual([]);
  const v = s.byId[ID];
  if (!v) throw new Error(`no view of ${ID}`);
  return v;
};

const lastCard = (): P.QuestionCard => {
  const last = [...stream].reverse().find((e) => e.type === 'question.card');
  if (!last) throw new Error('the stream has no question.card');
  return last.payload as P.QuestionCard;
};

const kinds = (v: SessionView) => flowBlocks(v.entries).map((b) => (b.kind === 'entry' ? b.entry.kind : b.kind));

describe('the live question card (D67)', () => {
  it('reads every version the daemon sent, and keeps one card that changes in place', () => {
    const v = fold();
    expect(v.entries.filter((e) => e.kind === 'note')).toEqual([]);
    const cards = cardsOf(v);
    expect(cards).toHaveLength(1);
    expect(cards[0]).toMatchObject({ id: 'c1', kind: 'merge', pending: 3 });
    expect(cards[0]?.items.map((i) => i.summary ? 'summary' : 'none')).toEqual(['summary', 'summary', 'summary']);
  });

  it('draws the card that waits for the person after everything said since, where the window looks', () => {
    const v = fold();
    const k = kinds(v);
    expect(k[k.length - 1]).toBe('card');
    // The orchestrator's last words come before the card they point to.
    expect(k.lastIndexOf('model')).toBeLessThan(k.indexOf('card'));
  });

  it('renders the card last in the conversation, with its three items and the seal', () => {
    const v = fold();
    const html = renderToStaticMarkup(<FlowBlocks blocks={flowBlocks(v.entries)} sessionId={ID} now={Date.parse('2026-10-18T15:00:00Z')} onAnswer={() => {}} />);
    expect(html).toContain('data-card-id="c1"');
    expect(html).toContain('1 de 3');
    expect(html).toContain('3 pendentes');
    expect(html.lastIndexOf('data-card-id="c1"')).toBeGreaterThan(html.lastIndexOf('Você decide.'));
  });

  it('puts a card back where it was first said once it no longer waits', () => {
    const closed: P.QuestionCard = {
      ...lastCard(),
      items: lastCard().items.map((i) => ({ ...i, state: 'answered', answer: 'merge', answered: '2026-10-18T14:50:00Z' })),
      pending: 0,
      updated: '2026-10-18T14:50:00Z',
      closed: '2026-10-18T14:50:00Z',
    };
    const last = stream[stream.length - 1] as { seq: number };
    const v = fold([...stream, { seq: last.seq + 1, session_id: ID, type: 'question.card', at: '2026-10-18T11:50:00-03:00', payload: closed }]);
    const k = kinds(v);
    expect(k[k.length - 1]).toBe('model');
    // Right after the turn whose question opened it.
    expect(k.slice(0, k.indexOf('card') + 1)).toEqual(['user', 'model', 'card']);
  });

  it('draws the card read from GET /questions when the stream never brought it, at the end', () => {
    const noCards = stream.filter((e) => e.type !== 'question.card').map((e, i) => ({ ...e, seq: i + 1 }));
    const read = decodeQuestions({ questions: [], cards: [lastCard()] });
    if (!read.ok) throw new Error(read.reason);
    const v = withCards(fold(noCards), read.value.cards);
    const k = kinds(v);
    expect(k[k.length - 1]).toBe('card');
    expect(cardsOf(v)[0]?.pending).toBe(3);
  });

  it('says a card it cannot read, naming the event and the field, never drops it', () => {
    const broken: Record<string, unknown> = { ...lastCard().items[0] };
    delete broken.selected;
    const bad = { ...lastCard(), items: [broken] };
    const s = applyRaw(emptySessions, [...stream.slice(0, 5), { seq: 6, session_id: ID, type: 'question.card', at: '2026-10-18T11:41:44-03:00', payload: bad }]);
    const notes = (s.byId[ID]?.entries ?? []).filter((e) => e.kind === 'note');
    expect(notes).toHaveLength(1);
    expect(notes[0]).toMatchObject({ note: { kind: 'protocol', reason: expect.stringMatching(/^question\.card: payload\.items\[0\]\.selected/) } });
  });
});
