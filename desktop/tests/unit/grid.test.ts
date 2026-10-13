import { describe, expect, it } from 'vitest';
import type * as P from '../../src/protocol/generated';
import { beginAt, emptyAttention, forgetMissing, hold, markSeen, parseAttention, release, togglePin, type Attention } from '../../src/state/attention';
import { capacityFor, claimOf, gridShape, planGrid } from '../../src/state/grid';
import { paneView } from '../../src/state/pane';
import { rowOfConversation, type Row } from '../../src/state/sidebar';
import { log } from './helpers';

function row(id: string, over: Partial<P.Conversation> = {}): Row {
  return rowOfConversation({
    id,
    title: id,
    workspace: '/w/dcode',
    state: 'idle',
    live: true,
    turns: 1,
    started: '2026-10-01T10:00:00Z',
    last_activity: '2026-10-01T10:05:00Z',
    last_event: 'turn.completed',
    ...over,
  });
}

const since = { ...emptyAttention, since: '2026-10-01T09:00:00Z' };

describe('who claims a place on the grid', () => {
  it('is what waits for you, what you pinned, what runs, and what finished since you looked', () => {
    expect(claimOf(row('a', { state: 'blocked' }), since)).toBe('waiting');
    expect(claimOf(row('b', { state: 'running' }), since)).toBe('running');
    expect(claimOf(row('c'), since)).toBe('unseen');
    expect(claimOf(row('c'), markSeen(since, 'c', '2026-10-01T10:05:00Z'))).toBeNull();
    expect(claimOf(row('d', { last_event: 'session.created' }), since)).toBeNull();
    expect(claimOf(row('d', { last_event: 'session.created' }), togglePin(since, 'd'))).toBe('pinned');
  });

  it('does not count as news what finished before the window first ran', () => {
    expect(claimOf(row('old', { last_activity: '2026-09-20T10:00:00Z' }), since)).toBeNull();
  });

  it('brings back what finished again after it was seen', () => {
    const seen = markSeen(since, 'c', '2026-10-01T10:05:00Z');
    expect(claimOf(row('c', { last_activity: '2026-10-01T11:00:00Z' }), seen)).toBe('unseen');
  });
});

describe('the grid', () => {
  it('keeps everyone in their place, and puts a newcomer in the next one', () => {
    const rows = [row('a', { state: 'running' }), row('b', { state: 'running' }), row('c', { state: 'running' })];
    const first = planGrid(rows, since, [], 6).ids;
    expect(first.slice().sort()).toEqual(['a', 'b', 'c']);
    const all = [...rows, row('d', { state: 'blocked' })];
    const second = planGrid(all, since, first, 6).ids;
    expect(second).toEqual([...first, 'd']);
    const third = planGrid(all.filter((r) => r.id !== first[0]), since, second, 6).ids;
    expect(third).toEqual(second.filter((id) => id !== first[0]));
  });

  it('gives the last places, when they run out, to what waits for you first', () => {
    const rows = [
      row('run1', { state: 'running', last_activity: '2026-10-01T12:00:00Z' }),
      row('run2', { state: 'running', last_activity: '2026-10-01T11:00:00Z' }),
      row('done', {}),
      row('wait', { state: 'blocked' }),
    ];
    const plan = planGrid(rows, since, [], 2);
    expect(plan.ids.slice().sort()).toEqual(['run1', 'wait']);
    expect(plan.overflow.map((r) => r.id)).toEqual(['run2', 'done']);
    expect(plan.claims.wait).toBe('waiting');
  });

  it('takes the shape of how many it holds', () => {
    expect([1, 2, 3, 4, 5, 6, 7].map((n) => gridShape(n))).toEqual([
      { cols: 1, rows: 1 },
      { cols: 2, rows: 1 },
      { cols: 2, rows: 2 },
      { cols: 2, rows: 2 },
      { cols: 3, rows: 2 },
      { cols: 3, rows: 2 },
      { cols: 3, rows: 3 },
    ]);
    expect([capacityFor(1440, 900), capacityFor(1920, 1080)]).toEqual([6, 9]);
  });
});

describe('the grid’s memory', () => {
  it('reads what it stored, starts once, and forgets what no longer exists', () => {
    expect(parseAttention(null)).toEqual({ attention: null, problem: null });
    const a: Attention = togglePin(markSeen(beginAt(emptyAttention, '2026-10-01T09:00:00Z'), 'x', '2026-10-01T10:00:00Z'), 'y');
    expect(parseAttention(JSON.stringify(a)).attention).toEqual(a);
    expect(beginAt(a, '2026-10-02T00:00:00Z').since).toBe('2026-10-01T09:00:00Z');
    expect(parseAttention('{nope').problem).toMatch(/ilegível/);
    expect(parseAttention(JSON.stringify({ pinned: 'x', seen: {}, since: null })).problem).toMatch(/formato/);
    expect(forgetMissing(a, new Set(['y']))).toEqual({ ...a, seen: {} });
    expect(togglePin(a, 'y').pinned).toEqual([]);
  });

  it('holds a window the person opened or wrote in, and lets it go, seen, when closed', () => {
    const held = hold(hold(since, 'a'), 'a');
    expect(held.pinned).toEqual(['a']);
    expect(claimOf(row('a', { last_event: 'session.created' }), held)).toBe('pinned');
    const closed = release(held, 'a', '2026-10-01T10:05:00Z');
    expect(closed.pinned).toEqual([]);
    expect(closed.seen.a).toBe('2026-10-01T10:05:00Z');
    // What it had finished counts as seen: closed, it does not come straight back.
    expect(claimOf(row('a'), closed)).toBeNull();
    // What still works keeps its place, closed or not.
    expect(claimOf(row('a', { state: 'running' }), closed)).toBe('running');
  });
});

describe('a panel', () => {
  const now = Date.parse('2026-10-01T10:00:30Z');

  it('leads with what the harness measured, criterion by criterion', () => {
    const v = log('s1', '2026-10-01T10:00:00Z')
      .add('turn.started', { turn_id: 't1', text: 'rode a régua' })
      .add('message.delta', { turn_id: 't1', text: 'Dois testes falharam.' })
      .add('turn.completed', {
        turn_id: 't1',
        reason: 'unverified',
        completion: { verification: 'failed', met: ['typecheck', 'lint'], unmet: ['test'] },
      })
      .fold();
    const p = paneView(row('s1'), v, now);
    expect(p.glyph).toBe('✗');
    expect(p.lights).toEqual([
      { name: 'typecheck', state: 'met' },
      { name: 'lint', state: 'met' },
      { name: 'test', state: 'unmet' },
    ]);
    // The criteria say it; the seal would repeat them.
    expect(p.seal).toBeNull();
  });

  it('says when the turn wrote where the work is measured, and a pass is no longer a plain one', () => {
    const v = log('s1', '2026-10-01T10:00:00Z')
      .add('turn.started', { turn_id: 't1', text: 'conserte o teste' })
      .add('turn.completed', {
        turn_id: 't1',
        reason: 'done',
        completion: { verification: 'passed', met: ['test'], touched_protected: ['parser_test.go'] },
      })
      .fold();
    const p = paneView(row('s1'), v, now);
    expect(p.touched).toEqual(['parser_test.go']);
    expect([p.glyph, p.tone]).toEqual(['✓', 'warn']);
    expect(paneView(row('s2'), log('s2', '2026-10-01T10:00:00Z').fold(), now).touched).toEqual([]);
  });

  it('breathes while it runs, and turns amber while it waits for you', () => {
    const running = log('s2', '2026-10-01T10:00:00Z')
      .add('turn.started', { turn_id: 't1', text: 'leia o README' })
      .add('tool.requested', { turn_id: 't1', tool_call_id: 'c1', name: 'read', input: { path: 'README.md' } });
    const p = paneView(row('s2', { state: 'running' }), running.fold(), now);
    expect([p.live, p.glyph, p.tone]).toEqual([true, '●', 'accent']);
    expect(p.doneNote).toBe('sem definição de pronto');

    const waiting = running
      .add('tool.approval_required', {
        approval_id: 'a1',
        turn_id: 't1',
        tool_call_id: 'c2',
        tool: 'bash',
        command: 'npm ci',
        boundary_crossed: 'network',
        expires_at: '2026-10-01T10:02:00Z',
      })
      .fold();
    const q = paneView(row('s2', { state: 'blocked' }), waiting, now);
    expect([q.live, q.glyph, q.tone]).toEqual([false, '◆', 'warn']);
  });

  it('reads an ended conversation from the list alone', () => {
    const p = paneView(row('old', { state: 'recorded', live: false, verification: 'passed', turns: 3 }), undefined, now);
    expect([p.glyph, p.seal?.text, p.when.endsWith('3 turnos')]).toEqual(['✓', '✓ verificado', true]);
  });
});
