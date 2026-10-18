import { describe, expect, it } from 'vitest';
import type * as P from '../../src/protocol/generated';
import {
  agentTree,
  applyAgentEvent,
  childStateView,
  ciView,
  emptyAgents,
  mainHealthView,
  ownsSummary,
  questionsOf,
  usageLine,
  withChildren,
  CHILD_STATES,
  type AgentList,
} from '../../src/state/agents';
import { log } from './helpers';

// The live agent list (orchestration RN-9, .p §11): folded from the
// orchestrator's own events over the daemon's answer to /children, and
// nothing else — no text read, no clock (D8).

const child = (id: string, over: Partial<P.Child> = {}): P.Child => ({
  id,
  parent_id: 's-orq',
  title: id,
  profile: 'fast',
  model: 'MiniMax-M3',
  state: 'working',
  owns: [`internal/${id}/`],
  branch: `dcode/${id}`,
  worktree: `/o/p/children/${id}`,
  base: '6ec45f1',
  usage: { input_tokens: 1000, output_tokens: 100 },
  started: '2026-10-17T12:00:00Z',
  ...over,
});

const question = (id: string, over: Partial<P.Question> = {}): P.Question => ({
  id,
  kind: 'merge',
  text: 'Merge #518?',
  recommended: 'merge',
  why: 'gate green on 1f2e3d4',
  child_id: 'c-a',
  options: [
    { id: 'merge', label: 'Merge' },
    { id: 'return', label: 'Return' },
  ],
  asked: '2026-10-17T12:05:00Z',
  ...over,
});

const fold = (from: AgentList, l: ReturnType<typeof log>) => l.events.reduce(applyAgentEvent, from);

describe('the live agent list, folded', () => {
  it('turns child.* and question.* over the initial list into the expected list', () => {
    const initial = withChildren(emptyAgents, [child('c-a', { started: '2026-10-17T12:00:00Z' }), child('c-b', { state: 'dispatched', started: '2026-10-17T12:01:00Z' })]);
    const l = log('s-orq')
      .add('child.started', { child: child('c-c', { state: 'dispatched', started: '2026-10-17T12:02:00Z' }) })
      .add('child.state', { child: child('c-b', { state: 'awaiting_person', waiting: 'approval', started: '2026-10-17T12:01:00Z' }) })
      .add('child.state', { child: child('c-a', { state: 'pr', waiting: 'merge', pr: { number: 518, ci: 'pending' }, usage: { input_tokens: 41000, output_tokens: 2000 } }) })
      .add('question.asked', question('q-1'))
      .add('question.asked', question('q-2', { kind: 'design', child_id: 'c-b', text: 'Which index?' }))
      .add('question.answered', { ...question('q-2', { kind: 'design', child_id: 'c-b' }), answer: 'a', answered: '2026-10-17T12:06:00Z' });
    const got = fold(initial, l);
    expect(got.children.map((c) => [c.id, c.state])).toEqual([
      ['c-a', 'pr'],
      ['c-b', 'awaiting_person'],
      ['c-c', 'dispatched'],
    ]);
    expect(got.children[0]!.usage.input_tokens).toBe(41000);
    expect(got.children[0]!.pr?.number).toBe(518);
    expect(got.questions.map((q) => q.id)).toEqual(['q-1']);
    expect(questionsOf(got, 'c-a').map((q) => q.id)).toEqual(['q-1']);
    expect(questionsOf(got, 'c-b')).toEqual([]);
  });

  it('changes a row on child.state without rereading the list', () => {
    const initial = withChildren(emptyAgents, [child('c-a')]);
    const got = applyAgentEvent(initial, log('s-orq').add('child.state', { child: child('c-a', { state: 'in_review' }) }).events[0]!);
    expect(got.children.map((c) => c.state)).toEqual(['in_review']);
  });

  it('lets a closed one leave when the daemon no longer carries it, and keeps a failed one', () => {
    const before = withChildren(emptyAgents, [child('c-a', { state: 'merged' }), child('c-b', { state: 'failed', reason: 'error' }), child('c-c', { state: 'closed' })]);
    // The daemon answers /children with what is still inside its retention.
    const after = withChildren(before, [child('c-b', { state: 'failed', reason: 'error' })]);
    expect(after.children.map((c) => c.id)).toEqual(['c-b']);
  });

  it('records a verdict, a gate, a refusal and the health of main', () => {
    const l = log('s-orq')
      .add('child.started', { child: child('c-a', { state: 'in_review' }) })
      .add('review.verdict', { child_id: 'c-a', session_id: 's-rev', commit: '9a8b7c6', verdict: 'approve' })
      .add('gate.measured', { child_id: 'c-a', commit: '9a8b7c6', tree: 't', base: 'b', command: 'make check', exit_code: 0, duration_ms: 81000, result: 'green', at: '2026-10-17T12:10:00Z' })
      .add('merge.refused', { child_id: 'c-a', code: 'main_red', reason: 'main is red at 1234567' })
      .add('main.health', { commit: '1234567', health: 'red', checked: '2026-10-17T12:11:00Z' });
    const got = fold(emptyAgents, l);
    expect(got.children[0]!.review?.verdict).toBe('approve');
    expect(got.children[0]!.gate?.result).toBe('green');
    expect(got.refused['c-a']?.code).toBe('main_red');
    expect(got.main?.health).toBe('red');
    const merged = applyAgentEvent(got, log('s-orq').add('merge.completed', { child_id: 'c-a', method: 'squash', merged: 'abc', measured: '9a8b7c6' }).events[0]!);
    expect(merged.refused['c-a']).toBeUndefined();
  });

  it('ignores an event about a child it does not know rather than inventing the child', () => {
    const got = applyAgentEvent(emptyAgents, log('s-orq').add('gate.measured', { child_id: 'nope', commit: 'c', tree: 't', base: 'b', command: 'x', exit_code: 1, duration_ms: 1, result: 'red', at: '2026-10-17T12:00:00Z' }).events[0]!);
    expect(got.children).toEqual([]);
  });

  it('is folded into the orchestrator’s session view', () => {
    const v = log('s-orq').add('child.started', { child: child('c-a') }).add('question.asked', question('q-1')).fold();
    expect(v.agents.children.map((c) => c.id)).toEqual(['c-a']);
    expect(v.agents.questions).toHaveLength(1);
  });
});

describe('the tree of agents', () => {
  it('puts each child under its parent, in the order they started', () => {
    const list = withChildren(emptyAgents, [
      child('c-b', { started: '2026-10-17T12:01:00Z' }),
      child('c-a', { started: '2026-10-17T12:00:00Z' }),
      child('c-a1', { parent_id: 'c-a', started: '2026-10-17T12:02:00Z' }),
    ]);
    expect(agentTree(list, 's-orq').map((n) => [n.child.id, n.depth])).toEqual([
      ['c-a', 0],
      ['c-a1', 1],
      ['c-b', 0],
    ]);
  });
});

describe('a row, in words', () => {
  it('has a word and a tone for each of the eight states, by D52 and D61', () => {
    const got = Object.fromEntries(CHILD_STATES.map((s) => [s, [childStateView(child('x', { state: s })).word, childStateView(child('x', { state: s })).tone]]));
    expect(got).toEqual({
      dispatched: ['despachado', 'dim'],
      working: ['trabalhando', 'accent'],
      awaiting_person: ['esperando você', 'warn'],
      in_review: ['em revisão', 'accent'],
      pr: ['PR', 'accent'],
      merged: ['merged', 'dim'],
      closed: ['fechado', 'dim'],
      failed: ['falhou', 'err'],
    });
  });

  it('gives a PR waiting for the merge answer the tone of what waits for you', () => {
    expect(childStateView(child('x', { state: 'pr', waiting: 'merge' }))).toMatchObject({ word: 'PR', tone: 'warn', detail: 'pergunta o merge' });
  });

  it('says a state it does not know in red, never as one it knows', () => {
    expect(childStateView(child('x', { state: 'paused' }))).toMatchObject({ tone: 'err', word: 'estado "paused" desconhecido' });
  });

  it('reads the review step from Review and Gate', () => {
    expect(childStateView(child('x', { state: 'in_review' })).detail).toBe('revisando');
    expect(childStateView(child('x', { state: 'in_review', review: { session_id: 'r', commit: 'c', verdict: 'approve' } })).detail).toBe('medindo o gate');
  });

  it('says the CI of a child’s PR by D61, the core’s word or GitChecks’', () => {
    expect(['passing', 'pass', 'failing', 'fail', 'pending', 'none', '', 'weird'].map((c) => ciView(c).tone)).toEqual(['ok', 'ok', 'err', 'err', 'dim', 'faint', 'faint', 'err']);
  });

  it('sums the tokens in, out and in cache, and no money', () => {
    expect(usageLine({ input_tokens: 41000, output_tokens: 2000, cache_read_tokens: 8100 })).toBe('41,0k entrada · 2,0k saída · 8,1k cache');
    expect(usageLine({ input_tokens: 0, output_tokens: 0 })).toBe('nenhum token ainda');
  });

  it('summarizes what a child owns', () => {
    expect(ownsSummary(['internal/a/'])).toBe('internal/a/');
    expect(ownsSummary(['internal/a/', 'internal/b/', 'docs/'])).toBe('internal/a/ +2');
    expect(ownsSummary([])).toBe('sem owns');
  });

  it('says the health of main', () => {
    expect(mainHealthView({ commit: '1234567abc', health: 'green', checked: '' })).toEqual({ tone: 'ok', word: 'main verde em 1234567' });
    expect(mainHealthView({ commit: '1234567abc', health: 'pending', checked: '' }).tone).toBe('dim');
  });
});
