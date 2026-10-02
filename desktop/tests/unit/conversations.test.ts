import { describe, expect, it } from 'vitest';
import { decodeChange, LIST_CHANGE_KINDS } from '../../src/protocol/conversations';
import * as P from '../../src/protocol/generated';
import { applyListChange, emptyConversations, liveContinuationOf } from '../../src/state/conversations';
import { projectsOf, rowOfConversation, UNTITLED } from '../../src/state/sidebar';
import { emptyPrefs } from '../../src/state/prefs';

function conversation(over: Partial<P.Conversation> = {}): P.Conversation {
  return {
    id: 'c1',
    title: 'conserte o parser',
    workspace: '/w/dcode',
    state: 'idle',
    live: true,
    turns: 1,
    started: '2026-09-30T10:00:00Z',
    last_activity: '2026-09-30T10:05:00Z',
    ...over,
  };
}

function read(raw: unknown) {
  const d = decodeChange(raw);
  if (!d.ok) throw new Error(d.reason);
  return d.change;
}

describe('the list of conversations, read at the boundary', () => {
  it('reads the kinds the core declares, and only those', () => {
    expect([...LIST_CHANGE_KINDS].sort()).toEqual([P.ConversationChanged, P.ConversationRemoved, P.ConversationSnapshot].sort());
  });

  it('reads an empty snapshot whether the list is left out or null', () => {
    expect(read({ kind: 'snapshot' })).toEqual({ kind: 'snapshot', conversations: [] });
    expect(read({ kind: 'snapshot', conversations: null })).toEqual({ kind: 'snapshot', conversations: [] });
  });

  it('says what is wrong with a frame instead of guessing from it', () => {
    const bad = decodeChange({ kind: 'changed', conversation: { ...conversation(), live: 'yes' } });
    expect(bad).toEqual({ ok: false, reason: expect.stringContaining('conversation.live') });
    expect(decodeChange({ kind: 'removed' })).toEqual({ ok: false, reason: expect.stringContaining('sem o id') });
    expect(decodeChange({ kind: 'renamed' })).toEqual({ ok: false, reason: expect.stringContaining('renamed') });
    expect(decodeChange('x')).toEqual({ ok: false, reason: expect.stringContaining('não é um objeto') });
  });
});

describe('folding the list', () => {
  it('replaces everything on a snapshot, upserts a change, and drops a removal', () => {
    let s = applyListChange(emptyConversations, read({ kind: 'snapshot', conversations: [conversation(), conversation({ id: 'c2' })] }));
    expect(s.loaded).toBe(true);
    expect(Object.keys(s.byId).sort()).toEqual(['c1', 'c2']);
    s = applyListChange(s, read({ kind: 'changed', conversation: conversation({ id: 'c2', state: 'running' }) }));
    expect(s.byId.c2?.state).toBe('running');
    s = applyListChange(s, read({ kind: 'removed', id: 'c1' }));
    expect(Object.keys(s.byId)).toEqual(['c2']);
    expect(applyListChange(s, read({ kind: 'removed', id: 'nobody' }))).toBe(s);
    s = applyListChange(s, read({ kind: 'snapshot', conversations: [conversation({ id: 'c9' })] }));
    expect(Object.keys(s.byId)).toEqual(['c9']);
  });

  it('finds the live continuation of an ended conversation, the newest one', () => {
    const s = applyListChange(
      emptyConversations,
      read({
        kind: 'snapshot',
        conversations: [
          conversation({ id: 'old', state: 'recorded', live: false }),
          conversation({ id: 'a', continued_from: 'old', started: '2026-09-30T11:00:00Z' }),
          conversation({ id: 'b', continued_from: 'old', started: '2026-09-30T12:00:00Z' }),
          conversation({ id: 'gone', continued_from: 'old', live: false, state: 'recorded', started: '2026-09-30T13:00:00Z' }),
        ],
      }),
    );
    expect(liveContinuationOf(s, 'old')?.id).toBe('b');
    expect(liveContinuationOf(s, 'a')).toBeNull();
  });
});

describe('a conversation as a sidebar row', () => {
  it('reads recorded when it ended, and is untitled when nothing was asked', () => {
    expect(rowOfConversation(conversation({ state: 'recorded', live: false })).state).toBe('recorded');
    expect(rowOfConversation(conversation({ state: 'blocked' })).state).toBe('blocked');
    expect(rowOfConversation(conversation({ title: '' })).title).toBe(UNTITLED);
    expect(rowOfConversation(conversation({ title: 'linha um\nlinha dois' })).title).toBe('linha um');
  });

  it('groups by project, the newest first within one', () => {
    const rows = [
      conversation({ id: 'a', started: '2026-09-30T09:00:00Z' }),
      conversation({ id: 'b', started: '2026-09-30T11:00:00Z' }),
      conversation({ id: 'c', workspace: '/w/eva', last_activity: '2026-09-30T12:00:00Z' }),
    ].map(rowOfConversation);
    const projects = projectsOf(rows, emptyPrefs);
    expect(projects.map((p) => [p.label, p.rows.map((r) => r.id)])).toEqual([
      ['eva', ['c']],
      ['dcode', ['b', 'a']],
    ]);
  });
});
