import { describe, expect, it } from 'vitest';
import * as P from '../../src/protocol/generated';
import { renderToStaticMarkup } from 'react-dom/server';
import { ProposalReview } from '../../src/renderer/ProposalReview';
import { continueRefusal, reviewOf } from '../../src/state/review';
import type { SessionView } from '../../src/state/session';
import { created, log } from './helpers';

// A proposed definition of done, reviewed before any work (D56): the card is
// drawn from done.proposed as the daemon measured it, never from the prose.

const FILE = '/w/dcode/specs/checkout/done.toml';

const proposal: P.DoneProposal = {
  proposal_id: 'p1',
  sequence: 1,
  proposed_at: '2026-10-14T10:05:00Z',
  path: FILE,
  protected: ['tests/**', 'specs/**/done.toml'],
  criteria: [
    { name: 'checkout total', command: 'go test ./cart -run Total', exit_code: 0, class: 'acceptance', exit: 1, expects: 'fail', why: 'o total soma o frete duas vezes', output: 'FAIL TestTotal' },
    { name: 'suíte inteira', command: 'go test ./...', exit_code: 0, class: 'regression', exit: 0, expects: 'pass' },
    { name: 'sem arg sai ≠ 0', command: 'bash pay.sh', exit_code: 0, class: 'regression', exit: 0, expects: 'fail', mismatch: true },
    { name: 'e2e', command: 'playwright test', exit_code: 0, class: 'broken', exit: 127, output: 'playwright: command not found' },
  ],
};

function qualifier(id: string, p?: P.DoneProposal): SessionView {
  const l = log(id).add('session.created', { ...created, id });
  if (p) l.add('done.proposed', p);
  return l.fold();
}

function run(over: Partial<P.LoopRun> = {}): P.LoopRun {
  return {
    id: 'L',
    workspace: '/w/dcode',
    state: 'awaiting_review',
    current: -1,
    session_id: 's2',
    started: '2026-10-14T10:00:00Z',
    specs: [
      { path: 'specs/home', phase: 'done', session_id: 's1' },
      { path: 'specs/checkout', phase: 'proposed', session_id: 's2', proposal: FILE },
    ],
    ...over,
  };
}

describe('the session keeps the last proposal it announced', () => {
  it('as the daemon measured it', () => {
    expect(qualifier('s2', proposal).proposal).toEqual(proposal);
    expect(qualifier('s2').proposal).toBeNull();
  });
});

describe('the review card', () => {
  it('is there only while the loop waits for a review', () => {
    expect(reviewOf(run({ state: 'working' }), { s2: qualifier('s2', proposal) })).toBeNull();
    expect(reviewOf(run(), { s2: qualifier('s2', proposal) })?.loopId).toBe('L');
  });

  it('shows each criterion with its class, what running it did, and where the eye should land', () => {
    const card = reviewOf(run(), { s2: qualifier('s2', proposal) });
    const p = card?.proposals[0];
    expect(p?.folder).toBe('specs/checkout');
    expect(p?.file).toBe(FILE);
    expect(p?.heading).toBe('4 critérios · medidos antes do trabalho');
    expect(p?.criteria.map((c) => [c.label, c.ran, c.passed])).toEqual([
      ['aceitação', 'exit 1', false],
      ['regressão', 'exit 0', true],
      ['regressão', 'exit 0', true],
      ['quebrado', 'exit 127', false],
    ]);
    expect(p?.criteria[0]?.why).toBe('o total soma o frete duas vezes');
    expect(p?.criteria[0]?.output).toBe('FAIL TestTotal');
    expect(p?.criteria[2]?.warning).toBe('O proponente disse que falharia, e passou: confira o porquê.');
    expect(p?.criteria[3]?.warning).toBe('Não havia o que rodar: este critério não testemunha nada e ficaria vermelho para sempre.');
    expect(p?.criteria[1]?.warning).toBeNull();
    expect(p?.protected).toEqual(['tests/**', 'specs/**/done.toml']);
    expect(p?.noAcceptance).toBeNull();
  });

  it('warns when nothing in the set is red', () => {
    const green = { ...proposal, criteria: [proposal.criteria[1]!], no_acceptance: true };
    const p = reviewOf(run(), { s2: qualifier('s2', green) })?.proposals[0];
    expect(p?.noAcceptance).toBe('Nenhum critério está vermelho: o conjunto diz pronto sem que nada mude. Num refactor pode ser o certo; fora dele, é defeito.');
  });

  it('says the proposal has not reached the window, rather than drawing an empty one', () => {
    const p = reviewOf(run(), {})?.proposals[0];
    expect(p?.criteria).toEqual([]);
    expect(p?.missing).toBe('A proposta desta pasta ainda não chegou à janela: abra o arquivo para lê-la.');
    const other = reviewOf(run(), { s2: qualifier('s2', { ...proposal, path: '/w/dcode/specs/other/done.toml' }) })?.proposals[0];
    expect(other?.missing).not.toBeNull();
  });

  it('lists every folder a goal qualified, one proposal each', () => {
    const two = run({
      specs: [
        { path: 'specs/a', phase: 'proposed', session_id: 'q1', proposal: '/w/dcode/specs/a/done.toml' },
        { path: 'specs/b', phase: 'pending' },
        { path: 'specs/c', phase: 'proposed', session_id: 'q2', proposal: '/w/dcode/specs/c/done.toml' },
      ],
    });
    expect(reviewOf(two, {})?.proposals.map((p) => p.folder)).toEqual(['specs/a', 'specs/c']);
  });
});

// Every proposal the qualifying turn made reaches the card (D62): the last one
// is what was measured and written, and the earlier ones are there to be read,
// oldest first, as the model sent them — the history is the defence against a
// re-proposal that swapped a criterion for a trivial one.
describe('the earlier proposals', () => {
  const third: P.DoneProposal = {
    ...proposal,
    sequence: 3,
    earlier: [
      {
        sequence: 1,
        proposed_at: '2026-10-14T10:01:00Z',
        criteria: [{ name: 'checkout total', command: 'go test ./cart', exit_code: 0, expects: 'fail', why: 'solto demais' }],
        protected: ['tests/**'],
      },
      { sequence: 2, proposed_at: '2026-10-14T10:03:30Z', criteria: [{ name: 'checkout total', command: 'true', exit_code: 0, expects: 'pass' }] },
    ],
  };

  it('are counted and listed oldest first, as the model sent them, with how long before the last', () => {
    const p = reviewOf(run(), { s2: qualifier('s2', third) })?.proposals[0];
    expect(p?.heading).toBe('4 critérios · medidos antes do trabalho · 3ª proposta do turno');
    expect(p?.earlierLabel).toBe('2 propostas anteriores');
    expect(p?.earlier.map((e) => e.title)).toEqual(['#1 · 1 critério · 4m00s antes da última', '#2 · 1 critério · 1m30s antes da última']);
    expect(p?.earlier[0]?.criteria).toEqual([{ name: 'checkout total', command: 'go test ./cart', said: 'disse que falharia', why: 'solto demais' }]);
    expect(p?.earlier[0]?.protected).toEqual(['tests/**']);
    expect(p?.earlier[1]?.criteria[0]?.said).toBe('disse que passaria');
  });

  it('say nothing when the turn proposed once', () => {
    const p = reviewOf(run(), { s2: qualifier('s2', proposal) })?.proposals[0];
    expect(p?.earlier).toEqual([]);
    expect(p?.earlierLabel).toBeNull();
    expect(p?.heading).toBe('4 critérios · medidos antes do trabalho');
  });

  it('are drawn folded under the written one, and only when there are any', () => {
    const card = reviewOf(run(), { s2: qualifier('s2', third) });
    const html = renderToStaticMarkup(<ProposalReview review={card!} onOpen={() => {}} onStop={() => {}} onStart={() => {}} />);
    expect(html).toContain('data-earlier-proposals="2"');
    expect(html).toContain('2 propostas anteriores');
    expect(html).toContain('#1 · 1 critério · 4m00s antes da última');
    expect(html).toContain('go test ./cart');
    const once = reviewOf(run(), { s2: qualifier('s2', proposal) });
    expect(renderToStaticMarkup(<ProposalReview review={once!} onOpen={() => {}} onStop={() => {}} onStart={() => {}} />)).not.toContain('data-earlier-proposals');
  });
});

describe('a review another window answered first', () => {
  it('is said with the daemon’s reason', () => {
    expect(continueRefusal(P.CodeLoopNotAwaitingReview, 'loop L is working, not waiting for a review')).toBe(
      'O trabalho já começou, ou o loop saiu da revisão — outra janela ou a TUI respondeu antes: loop L is working, not waiting for a review',
    );
    expect(continueRefusal('loop_not_found', 'no loop L')).toBe('O trabalho não começou: no loop L');
  });
});
