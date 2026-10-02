import { describe, expect, it } from 'vitest';
import { countWorking, foldConversations } from '../../src/main/conversations';
import type { Conversation } from '../../src/protocol/generated';

function conv(id: string, over: Partial<Conversation> = {}): Conversation {
  return {
    id,
    title: `conversa ${id}`,
    workspace: '/w',
    state: 'idle',
    live: true,
    turns: 0,
    started: '2026-10-01T10:00:00Z',
    last_activity: '2026-10-01T10:00:00Z',
    ...over,
  };
}

/** Folds the frames in order, failing on any that does not fold. */
function fold(...frames: unknown[]): Conversation[] | null {
  let list: Conversation[] | null = null;
  for (const f of frames) {
    const got = foldConversations(list, f);
    if (!got.ok) throw new Error(got.problem);
    list = got.list;
  }
  return list;
}

const ids = (list: Conversation[] | null) => list?.map((c) => c.id);

describe('the list kept in the main process', () => {
  it('is the snapshot, whole', () => {
    expect(ids(fold({ kind: 'snapshot', conversations: [conv('b'), conv('a')] }))).toEqual(['b', 'a']);
  });

  it('is empty after a snapshot that left the empty list out, as Go does', () => {
    expect(fold({ kind: 'snapshot' })).toEqual([]);
  });

  it('puts a changed conversation in its place, and a new one first', () => {
    const list = fold(
      { kind: 'snapshot', conversations: [conv('b'), conv('a')] },
      { kind: 'changed', conversation: conv('a', { state: 'running' }) },
      { kind: 'changed', conversation: conv('c') },
    );
    expect(ids(list)).toEqual(['c', 'b', 'a']);
    expect(list?.find((c) => c.id === 'a')?.state).toBe('running');
  });

  it('takes out a removed conversation', () => {
    expect(ids(fold({ kind: 'snapshot', conversations: [conv('b'), conv('a')] }, { kind: 'removed', id: 'b' }))).toEqual(['a']);
  });

  it('starts over with a later snapshot, as a reconnection opens', () => {
    const list = fold(
      { kind: 'snapshot', conversations: [conv('a')] },
      { kind: 'changed', conversation: conv('b') },
      { kind: 'snapshot', conversations: [conv('z')] },
    );
    expect(ids(list)).toEqual(['z']);
  });

  it('does not take a change before the first snapshot for the whole list', () => {
    const got = foldConversations(null, { kind: 'changed', conversation: conv('a') });
    expect(got).toEqual({ ok: false, problem: 'a change before the first snapshot' });
  });

  it('says what it could not fold, and leaves the list as it was', () => {
    const list = [conv('a')];
    for (const frame of ['not json', { kind: 'changed' }, { kind: 'removed' }, { kind: 'renamed', id: 'a' }, { conversations: [] }]) {
      const got = foldConversations(list, frame);
      expect(got.ok, JSON.stringify(frame)).toBe(false);
    }
    expect(foldConversations(list, { kind: 'snapshot', conversations: [{ title: 'sem id' }] }).ok).toBe(false);
  });
});

describe('what quitting would stop', () => {
  it('counts the live conversations running or waiting on an approval, and nothing else', () => {
    const list = [
      conv('a', { state: 'running' }),
      conv('b', { state: 'blocked' }),
      conv('c', { state: 'idle' }),
      conv('d', { state: 'recorded', live: false }),
    ];
    expect(countWorking(list)).toBe(2);
  });
});
