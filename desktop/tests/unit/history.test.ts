import { describe, expect, it } from 'vitest';
import { filesTouched } from '../../src/state/context';
import { decodePage, foldHistory, readHistory, type PageFetcher } from '../../src/state/history';
import { done, log } from './helpers';

/** A recorded conversation as the daemon renumbers it: one sequence, under its own id. */
function recorded() {
  return log('old')
    .add('turn.started', { turn_id: 't-1', text: 'conserte o parser' })
    .add('message.delta', { turn_id: 't-1', text: 'Vou ler ' })
    .add('message.delta', { turn_id: 't-1', text: 'o arquivo.' })
    .add('tool.requested', { turn_id: 't-1', tool_call_id: 'c1', name: 'edit', input: { path: 'parser.go' } })
    .add('tool.completed', done('c1', { added: 3, removed: 1 }))
    .add('turn.completed', { turn_id: 't-1', reason: 'done' }).events;
}

/** Serves `events` in pages of `size`, the way the route does, and records each `from` asked. */
function pages(events: readonly unknown[], size: number) {
  const asked: number[] = [];
  const fetch: PageFetcher = async (from) => {
    asked.push(from);
    const end = Math.min(from - 1 + size, events.length);
    return { ok: true, value: { events: events.slice(from - 1, end), total: events.length, next: end < events.length ? end + 1 : undefined } };
  };
  return { fetch, asked };
}

describe('a recorded conversation', () => {
  it('reads as it did while it ran: the question, one answer, the call, and the files it changed', async () => {
    const { fetch, asked } = pages(recorded(), 2);
    const h = await readHistory('old', fetch);
    expect(asked).toEqual([1, 3, 5]);
    if (h.state !== 'loaded') throw new Error(`not loaded: ${JSON.stringify(h)}`);
    expect(h.cut).toBe(0);
    expect(h.view.state).toBe('recorded');
    expect(h.view.entries.map((e) => e.kind)).toEqual(['user', 'model', 'tool']);
    expect(h.view.entries[1]).toMatchObject({ kind: 'model', text: 'Vou ler o arquivo.' });
    expect(filesTouched(h.view.entries)).toEqual([{ path: 'parser.go', added: 3, removed: 1, failed: false }]);
    expect(h.view.turns).toBe(1);
  });

  it('says how much was left out when it is longer than the window reads', async () => {
    const { fetch } = pages(recorded(), 2);
    const h = await readHistory('old', fetch, 2);
    expect(h).toMatchObject({ state: 'loaded', total: 6, cut: 2 });
  });

  it('fails with the reason when a page is refused, never showing half a conversation', async () => {
    let n = 0;
    const fetch: PageFetcher = async () =>
      ++n === 1
        ? { ok: true, value: { events: recorded().slice(0, 2), total: 6, next: 3 } }
        : { ok: false, why: 'O histórico não foi lido: no recorded conversation' };
    expect(await readHistory('old', fetch)).toEqual({ state: 'failed', reason: 'O histórico não foi lido: no recorded conversation' });
  });

  it('fails on an answer of the wrong shape, and on a next that does not move forward', async () => {
    const bad: PageFetcher = async () => ({ ok: true, value: { events: 'x', total: 1 } });
    expect(await readHistory('old', bad)).toMatchObject({ state: 'failed', reason: expect.stringContaining('events deveria ser uma lista') });
    const stuck: PageFetcher = async () => ({ ok: true, value: { events: [], total: 9, next: 1 } });
    expect(await readHistory('old', stuck)).toMatchObject({ state: 'failed', reason: expect.stringContaining('próxima começa em 1') });
  });

  it('reads an empty conversation as empty, and an unreadable event as a note in its place', () => {
    expect(decodePage({ events: null, total: 0 })).toEqual({ ok: true, page: { events: [], total: 0, next: 0 } });
    const v = foldHistory('old', [{ seq: 1, session_id: 'old', type: 'turn.started', at: '2026-09-28T17:00:00Z', payload: 'x' }]);
    expect(v.entries).toMatchObject([{ kind: 'note', note: { kind: 'protocol' } }]);
  });

  it('keeps out, and says, events of another conversation', () => {
    const v = foldHistory('old', [...recorded(), ...log('other').add('turn.started', { turn_id: 't', text: 'intrusa' }).events]);
    expect(v.entries.some((e) => e.kind === 'user' && e.text === 'intrusa')).toBe(false);
    expect(v.entries.at(-1)).toMatchObject({ kind: 'note', note: { kind: 'protocol', reason: expect.stringContaining('other') } });
  });
});
