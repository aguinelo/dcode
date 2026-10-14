import { describe, expect, it } from 'vitest';
import { loopArgs, loopId, stopWhen } from '../../src/main/inputs';
import { countLiveLoops, foldLoops } from '../../src/main/loops';
import type { LoopRun } from '../../src/protocol/generated';

// The list of loops as the main process keeps it (D55), and what a window may
// ask about a loop, checked before it reaches the daemon.

function run(id: string, over: Partial<LoopRun> = {}): LoopRun {
  return { id, workspace: '/w', state: 'working', specs: [], current: -1, started: '2026-10-14T10:00:00Z', ...over };
}

function fold(...frames: unknown[]): LoopRun[] | null {
  let list: LoopRun[] | null = null;
  for (const f of frames) {
    const got = foldLoops(list, f);
    if (!got.ok) throw new Error(got.problem);
    list = got.list;
  }
  return list;
}

describe('the list of loops the main process keeps', () => {
  it('starts from the snapshot and puts each change in its place, a new run last', () => {
    const list = fold(
      { kind: 'snapshot', loops: [run('a'), run('b')] },
      { kind: 'changed', loop: run('b', { state: 'stopped' }) },
      { kind: 'changed', loop: run('c') },
    );
    expect(list?.map((r) => `${r.id}:${r.state}`)).toEqual(['a:working', 'b:stopped', 'c:working']);
  });

  it('reads a snapshot without the field as no loops', () => {
    expect(fold({ kind: 'snapshot' })).toEqual([]);
  });

  it('says what it cannot fold, never guesses', () => {
    expect(foldLoops(null, { kind: 'changed', loop: run('a') })).toEqual({ ok: false, problem: 'a change before the first snapshot' });
    expect(foldLoops([], { kind: 'changed', loop: { state: 'x' } })).toMatchObject({ ok: false });
    expect(foldLoops([], { kind: 'removed', id: 'a' })).toEqual({ ok: false, problem: 'an unknown kind “removed”' });
    expect(foldLoops([], 'x')).toMatchObject({ ok: false });
  });

  it('counts the runs still alive', () => {
    const states = ['surveying', 'qualifying', 'awaiting_review', 'working', 'finished', 'stopped', 'failed'];
    expect(countLiveLoops(states.map((state, i) => run(String(i), { state })))).toBe(4);
  });
});

function refused(got: { ok: boolean; refusal?: { code: string; message: string } }): string {
  expect(got.ok).toBe(false);
  expect(got.refusal?.code).toBe('invalid_input');
  return got.refusal?.message ?? '';
}

describe('what a window asks about a loop', () => {
  it('takes the argument of /loop as typed, and refuses an empty or unreadable one', () => {
    expect(loopArgs('specs/home --protect tests/**')).toEqual({ ok: true, value: 'specs/home --protect tests/**' });
    expect(refused(loopArgs('   '))).toContain('/loop precisa');
    expect(refused(loopArgs(42))).toContain('/loop precisa');
    expect(refused(loopArgs('a\0b'))).toContain('caractere nulo');
    expect(refused(loopArgs('x'.repeat(4097)))).toContain('4096');
  });

  it('takes a loop id only in the daemon’s alphabet, since it becomes a path there', () => {
    expect(loopId('0193a-b_c')).toEqual({ ok: true, value: '0193a-b_c' });
    expect(refused(loopId('../x'))).toContain('O id do loop');
    expect(refused(loopId(''))).toContain('O id do loop');
  });

  it('stops now or after the folder, and nothing else', () => {
    expect(stopWhen('now')).toEqual({ ok: true, value: 'now' });
    expect(stopWhen('after_spec')).toEqual({ ok: true, value: 'after_spec' });
    expect(refused(stopWhen('later'))).toContain('now, after_spec');
  });
});
