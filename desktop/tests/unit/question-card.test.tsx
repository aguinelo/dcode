import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import type * as P from '../../src/protocol/generated';
import { AgentsPanel } from '../../src/renderer/AgentsPanel';
import { QuestionCardView } from '../../src/renderer/QuestionCard';
import { emptyAgents } from '../../src/state/agents';
import {
  blocked,
  cardFor,
  cardsOf,
  choose,
  confirmBatch,
  goTo,
  isLast,
  recommendAll,
  resultsById,
  skip,
  startDraft,
  step,
  withCards,
} from '../../src/state/card';
import { emptySession } from '../../src/state/session';
import { log } from './helpers';

// The orchestrator's question card in the conversation (.p §5.5, D67): one
// item at a time with "1 de N", the options as radio rows, the selected one
// checked and the recommended one said, "Outro" for the person's own words,
// "Pular" and "Próximo", and on the last item one confirm with every answer.

const evidence: P.QuestionEvidence = {
  verdict: 'approve',
  reviewer: 's-rev',
  reviewed_commit: '1f2e3d4c',
  review_seq: 7,
  gate: 'green',
  command: 'make check',
  commit: '1f2e3d4c5b',
  base: '6ec45f1',
  exit_code: 0,
  duration_ms: 4000,
  gate_seq: 9,
};

const options: P.QuestionOption[] = [
  { id: 'merge', label: 'Merge', description: 'Entra na main.' },
  { id: 'return', label: 'Devolver', description: 'Volta ao filho com a nota.' },
  { id: 'close', label: 'Fechar' },
];

function item(id: string, header: string, extra: Partial<P.CardItem> = {}): P.CardItem {
  return {
    id,
    header,
    kind: 'merge',
    text: `Merge "${header}"?`,
    recommended: 'merge',
    why: 'aprovado e verde',
    child_id: `c-${id}`,
    options,
    allow_other: true,
    summary: `Muda ${header}.\nCom teste.`,
    evidence,
    asked: '2026-10-18T10:00:00Z',
    state: 'pending',
    selected: 'merge',
    ...extra,
  };
}

function card(extra: Partial<P.QuestionCard> = {}): P.QuestionCard {
  return {
    id: 'card-1',
    kind: 'merge',
    title: 'Merges prontos',
    items: [item('q1', 'índice'), item('q2', 'notas'), item('q3', 'docs')],
    pending: 3,
    created: '2026-10-18T10:00:00Z',
    updated: '2026-10-18T10:00:00Z',
    ...extra,
  };
}

const proposed = card({
  id: 'card-2',
  proposed: true,
  from: 'faz merge dos dois primeiros e devolve o de docs, falta teste',
  why: 'li duas aprovações e uma devolução',
  items: [item('q1', 'índice'), item('q2', 'notas'), item('q3', 'docs', { selected: 'return', selected_note: 'falta teste' })],
});

describe('a card, drafted', () => {
  it('opens on its first item that waits, nothing picked yet', () => {
    const c = card({ items: [item('q1', 'a', { state: 'answered', answer: 'merge' }), item('q2', 'b'), item('q3', 'c')] });
    const d = startDraft(c);
    expect(d.index).toBe(1);
    expect(d.seen).toEqual(['q2']);
  });

  it('pages with the arrows inside the card, and knows the last item', () => {
    const c = card();
    let d = startDraft(c);
    d = goTo(c, d, d.index - 1);
    expect(d.index).toBe(0);
    d = goTo(c, d, 2);
    expect(isLast(c, d)).toBe(true);
    d = goTo(c, d, 9);
    expect(d.index).toBe(2);
    expect(d.seen).toEqual(['q1', 'q3']);
  });

  it('moves the pick with ↑ and ↓ through the options and then “Outro”', () => {
    const c = card();
    let d = startDraft(c);
    d = step(c, d, 1);
    expect(d.picks.q1).toEqual({ answer: 'return', note: '' });
    d = step(c, step(c, d, 1), 1);
    expect(d.picks.q1?.answer).toBe('other');
    d = step(c, d, 1);
    expect(d.picks.q1?.answer).toBe('other');
    d = step(c, d, -3);
    expect(d.picks.q1?.answer).toBe('merge');
  });

  it('skips an item, which goes in skipped and keeps waiting', () => {
    const c = card();
    let d = skip(c, startDraft(c));
    expect(d.index).toBe(1);
    d = skip(c, d);
    d = choose(c, d, 'close');
    expect(confirmBatch(c, d)).toEqual({ card_id: 'card-1', answers: [{ question_id: 'q3', answer: 'close' }], skipped: ['q1', 'q2'] });
  });

  it('confirms every answer the person was shown in one batch; one never shown is skipped', () => {
    const c = card();
    let d = startDraft(c);
    d = choose(c, d, 'return');
    d = goTo(c, d, 2);
    expect(confirmBatch(c, d)).toEqual({
      card_id: 'card-1',
      answers: [
        { question_id: 'q1', answer: 'return' },
        { question_id: 'q3', answer: 'merge' },
      ],
      skipped: ['q2'],
    });
  });

  it('sends “Outro” with the words as the note, and refuses it without words', () => {
    const c = card();
    let d = choose(c, startDraft(c), 'other', '');
    expect(blocked(c, d)).toContain('Outro');
    d = choose(c, d, 'other', ' espera o CI ');
    expect(blocked(c, d)).toBeNull();
    expect(confirmBatch(c, d).answers).toEqual([{ question_id: 'q1', answer: 'other', note: 'espera o CI' }]);
  });

  it('applies the recommendation to every item still waiting', () => {
    const c = card({ items: [item('q1', 'a', { state: 'answered' }), item('q2', 'b', { recommended: 'return' }), item('q3', 'c')] });
    expect(recommendAll(c)).toEqual({
      card_id: 'card-1',
      answers: [
        { question_id: 'q2', answer: 'return' },
        { question_id: 'q3', answer: 'merge' },
      ],
    });
  });

  it('takes a proposed card’s selections, with their notes, as the picks', () => {
    let d = startDraft(proposed);
    d = goTo(proposed, goTo(proposed, d, 1), 2);
    expect(confirmBatch(proposed, d).answers).toEqual([
      { question_id: 'q1', answer: 'merge' },
      { question_id: 'q2', answer: 'merge' },
      { question_id: 'q3', answer: 'return', note: 'falta teste' },
    ]);
  });
});

const render = (c: P.QuestionCard, index = 0, results: P.AnswerResult[] = []) => {
  const d = goTo(c, startDraft(c), index);
  return renderToStaticMarkup(<QuestionCardView card={c} initial={d} results={resultsById(results)} answer={async () => ({ ok: true, value: [] })} />);
};

describe('the card, drawn', () => {
  it('shows one item: “1 de 3”, the header, the question as title, the options with the selected one checked', () => {
    const html = render(card());
    expect(html).toContain('data-card-id="card-1"');
    expect(html).toContain('1 de 3');
    expect(html).toContain('índice');
    expect(html).toContain('Merge &quot;índice&quot;?');
    expect(html).toContain('Merge (Recomendado)');
    expect(html).toContain('Volta ao filho com a nota.');
    expect(html.match(/aria-checked="true"/g)).toHaveLength(1);
    expect(html).toContain('data-option="merge"');
    expect(html).toContain('data-option="other"');
    expect(html).toContain('Pular');
    expect(html).toContain('Próximo');
    expect(html).toContain('⌘↵');
    expect(html).toContain('Aplicar a recomendação a todos');
    expect(html).toContain('Precisa da sua contribuição');
    expect(html).not.toContain('data-question-id');
    expect(html).not.toContain('data-answer');
  });

  it('shows the summary and the evidence of a merge under the question', () => {
    const html = render(card());
    expect(html).toContain('Muda índice.');
    expect(html).toContain('revisão: approve');
    expect(html).toContain('gate: green');
    expect(html).toContain('make check');
    expect(html).toContain('1f2e3d4');
    expect(html).not.toContain('1f2e3d4c5b');
  });

  it('turns “Próximo” into “Confirmar” on the last item', () => {
    const html = render(card(), 2);
    expect(html).toContain('3 de 3');
    expect(html).toContain('Confirmar');
    expect(html).toContain('data-card-confirm');
    expect(html).not.toContain('Próximo');
  });

  it('shows the person’s words on a proposed card, the proposal checked', () => {
    const html = render(proposed, 2);
    expect(html).toContain('Você pediu:');
    expect(html).toContain('faz merge dos dois primeiros');
    expect(html).toContain('li duas aprovações');
    expect(html).toContain('Nota que vai junto: falta teste');
    expect(html).toMatch(/data-option="return"[^>]*aria-checked="true"|aria-checked="true"[^>]*data-option="return"/);
  });

  it('says what became of an answer under its item, a refusal with the daemon’s words', () => {
    const html = render(card(), 0, [{ question_id: 'q1', status: 'refused', code: 'question_already_answered', message: 'q1 was already answered "merge"' }]);
    expect(html).toContain('já foi respondida');
    expect(html).toContain('q1 was already answered');
  });

  it('draws a closed card as what was answered, without the selo', () => {
    const c = card({ pending: 0, closed: '2026-10-18T10:09:00Z', items: [item('q1', 'a', { state: 'answered', answer: 'merge' })] });
    const html = render(c);
    expect(html).toContain('Merge');
    expect(html).not.toContain('Precisa da sua contribuição');
    expect(html).not.toContain('Confirmar');
  });
});

describe('the cards of a conversation', () => {
  it('draws a card where it was first said and changes it in place', () => {
    const l = log('s-orq').add('message.delta', { turn_id: 't', text: 'três prontos' }).add('question.card', card());
    l.add('message.delta', { turn_id: 't2', text: 'esperando' });
    l.add('question.card', card({ pending: 2, updated: '2026-10-18T10:01:00Z' }));
    const view = l.fold();
    expect(view.entries.map((e) => e.kind)).toEqual(['model', 'card', 'model']);
    expect(cardsOf(view).map((c) => c.pending)).toEqual([2]);
  });

  it('takes the daemon’s cards over older ones, and adds one the flow has not seen at the end', () => {
    const view = withCards(log('s-orq').add('question.card', card({ updated: '2026-10-18T10:05:00Z', pending: 1 })).fold(), [
      card({ pending: 3 }),
      proposed,
    ]);
    expect(cardsOf(view).map((c) => [c.id, c.pending])).toEqual([
      ['card-1', 1],
      ['card-2', 3],
    ]);
    expect(withCards(emptySession('x'), []).entries).toEqual([]);
  });

  it('finds the open card that holds a waiting question, for “◆ N perguntas”', () => {
    const q = (id: string) => ({ id }) as P.Question;
    expect(cardFor([card()], [q('q2')])).toBe('card-1');
    expect(cardFor([card({ pending: 0, closed: '2026-10-18T10:09:00Z' })], [q('q2')])).toBeNull();
    expect(cardFor([card()], [q('q9')])).toBeNull();
    expect(cardFor([card(), proposed], [q('q2')])).toBe('card-2');
  });

  it('points “◆ N perguntas” at the card that holds them, and at the queue when none does', () => {
    const agents = { ...emptyAgents, questions: [item('q2', 'notas')] };
    const panel = (cards: P.QuestionCard[]) => renderToStaticMarkup(<AgentsPanel agents={agents} rootId="s-orq" canOpen={() => false} onOpen={() => {}} cards={cards} />);
    expect(panel([card()])).toContain('data-card-target="card-1"');
    expect(panel([])).not.toContain('data-card-target');
    expect(panel([])).toContain('aria-expanded="false"');
  });
});
