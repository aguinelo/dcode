import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import type * as P from '../../src/protocol/generated';
import { PlanPin } from '../../src/renderer/PlanPin';
import { pinnedPlan } from '../../src/state/plan';
import { PHASES, childPhase, stepAgents } from '../../src/state/plan-agents';
import { planRecording } from '../../src/fixtures/plan-recording';
import { applyRaw, emptySessions } from '../../src/state/sessions';
import { created, log } from './helpers';

// The orchestrator's plan as its main view (D68): pinned at the top of its
// conversation while children work, and each step with the agents that
// fulfil it — the name, the phase pipeline with the current phase in its
// tone, the tokens — and a click that opens the child.

const usage = { input_tokens: 1200, output_tokens: 300, cache_read_tokens: 0, cache_write_tokens: 0 };

function child(id: string, state: string, step: number, extra: Partial<P.Child> = {}): P.Child {
  return {
    id,
    parent_id: 's-orq',
    title: `filho ${id}`,
    profile: '',
    model: 'm',
    state,
    owns: ['internal/x'],
    branch: `orq/${id}`,
    worktree: `/w/${id}`,
    base: 'abc',
    usage,
    started: '2026-10-18T12:00:00Z',
    plan_step: step,
    ...extra,
  } as P.Child;
}

const plan: P.PlanItem[] = [
  { id: 1, text: 'Contar as palavras', status: 'done' },
  { id: 2, text: 'Documentar', status: 'active' },
  { id: 3, text: 'Testar', status: 'active' },
  { id: 4, text: 'Revisar a release', status: 'pending' },
];

const at = (c: P.Child) => PHASES[childPhase(c).at]!.key;

describe('the phase of a child, on the pipeline (D68)', () => {
  it('is the six phases in the order a child moves through them', () => {
    expect(PHASES.map((p) => p.word)).toEqual(['despachado', 'trabalhando', 'revisão', 'gate', 'PR', 'merged']);
  });

  it('places each state on its phase, in its tone', () => {
    expect(at(child('a', 'dispatched', 1))).toBe('dispatched');
    expect(childPhase(child('a', 'working', 1)).tone).toBe('accent');
    expect(at(child('a', 'awaiting_person', 1, { waiting: 'approval' }))).toBe('working');
    expect(childPhase(child('a', 'awaiting_person', 1, { waiting: 'approval' })).tone).toBe('warn');
    expect(at(child('a', 'in_review', 1))).toBe('review');
    const verdict = { session_id: 'r', commit: 'c', verdict: 'approve' } as P.Verdict;
    expect(at(child('a', 'in_review', 1, { review: verdict }))).toBe('gate');
    expect(at(child('a', 'pr', 1, { waiting: 'merge' }))).toBe('pr');
    expect(childPhase(child('a', 'pr', 1, { waiting: 'merge' }))).toMatchObject({ tone: 'warn', word: 'espera o merge' });
    expect(childPhase(child('a', 'merged', 1))).toMatchObject({ at: 5, tone: 'ok' });
  });

  it('says a failure where it stopped, and a return, in their tones', () => {
    const failed = childPhase(child('a', 'failed', 1, { reason: 'incomplete', review: { session_id: 'r', commit: 'c', verdict: 'approve' } as P.Verdict }));
    expect(failed).toMatchObject({ at: 3, tone: 'err' });
    expect(failed.word).toContain('falhou');
    expect(childPhase(child('a', 'working', 1, { returned: 2 }))).toMatchObject({ at: 1, tone: 'warn', word: 'devolvido 2×' });
    expect(childPhase(child('a', 'closed', 1)).tone).toBe('dim');
  });
});

describe('the agents of each plan step (D68)', () => {
  it('groups the children under the step they name, and keeps the rest outside the plan', () => {
    const kids = [child('a', 'merged', 1), child('b', 'in_review', 2), child('c', 'pr', 3), child('d', 'working', 0), child('e', 'working', 9)];
    const g = stepAgents(plan, kids);
    expect(g.byStep[1]!.map((c) => c.id)).toEqual(['a']);
    expect(g.byStep[2]!.map((c) => c.id)).toEqual(['b']);
    expect(g.byStep[4]).toBeUndefined();
    expect(g.outside.map((c) => c.id)).toEqual(['d', 'e']);
  });

  it('pins the orchestrator plan with no turn running: its children work while it is idle', () => {
    const v = log('s-orq')
      .add('session.created', { ...created, id: 's-orq', role: 'orchestrator' })
      .add('turn.started', { turn_id: 't-1', text: 'Planeje e despache.' })
      .add('plan.updated', { items: plan })
      .add('turn.completed', { turn_id: 't-1', reason: 'done' })
      .fold();
    expect(v.turn).toBeNull();
    expect(pinnedPlan(v)).toMatchObject({ total: 4 });
  });

  it('draws each step with its agents inline: name, pipeline, tokens; a step with no agent as before', () => {
    const pin = { items: plan, step: 2, total: 4, current: 'Documentar' };
    const kids = [
      child('a', 'merged', 1),
      child('b', 'in_review', 2),
      child('c', 'pr', 3, { waiting: 'merge' }),
      child('d', 'working', 0),
    ];
    const opened: string[] = [];
    const html = renderToStaticMarkup(
      <PlanPin pin={pin} compact={false} agents={stepAgents(plan, kids)} canOpen={(id) => id !== 'd'} onOpen={(id) => opened.push(id)} />,
    );
    expect(html.match(/data-plan-child-id=/g)).toHaveLength(4);
    expect(html).toMatch(/data-plan-child-id="a"[^>]*data-phase-current="merged"/);
    expect(html).toMatch(/data-plan-child-id="b"[^>]*data-phase-current="review"/);
    expect(html).toMatch(/data-plan-child-id="c"[^>]*data-phase-current="pr"/);
    expect(html).toContain('espera o merge');
    expect(html).toContain('1,5k tokens');
    expect(html).toContain('Fora do plano');
    for (const w of ['despachado', 'trabalhando', 'revisão', 'gate', 'PR', 'merged']) expect(html).toContain(w);
    // The step with no agent is drawn as D57 drew it: no list under it.
    expect(html).toMatch(/Revisar a release<\/span><\/li>/);
    expect(html).toContain('disabled');
  });

  it('draws a plain plan, as D57 did, when no agent is given', () => {
    const pin = { items: plan, step: 2, total: 4, current: 'Documentar' };
    const html = renderToStaticMarkup(<PlanPin pin={pin} compact={false} />);
    expect(html).not.toContain('data-plan-child-id');
    expect(html).not.toContain('Fora do plano');
  });
});

describe('the plan recording (D68)', () => {
  it('replays three steps whose agents are merged, under review and waiting for the merge, the card below', () => {
    const orq = applyRaw(emptySessions, planRecording.events).byId['s-plano']!;
    const pin = pinnedPlan(orq);
    expect(pin?.items.map((i) => i.status)).toEqual(['done', 'active', 'active']);
    const g = stepAgents(pin!.items, orq.agents.children);
    expect([1, 2, 3].map((s) => at(g.byStep[s]![0]!))).toEqual(['merged', 'review', 'pr']);
    expect(orq.entries.some((e) => e.kind === 'card')).toBe(true);
  });
});
