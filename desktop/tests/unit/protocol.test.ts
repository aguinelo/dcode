import { describe, expect, it } from 'vitest';
import * as generated from '../../src/protocol/generated';
import { decodeEvent, EVENT_KINDS } from '../../src/protocol/events';

const envelope = (type: string, payload: unknown) => ({
  seq: 1,
  session_id: 's-1',
  type,
  at: '2026-09-28T17:00:00Z',
  payload,
});

describe('the event types', () => {
  it('reads every event type internal/protocol declares, and no other', () => {
    // A type the core adds shows up in the generated file first, and fails
    // here until the desktop decides how to read it.
    const declared = Object.entries(generated)
      .filter(([name, value]) => name.startsWith('Event') && typeof value === 'string')
      .map(([, value]) => value as string)
      .sort();
    expect([...EVENT_KINDS].sort()).toEqual(declared);
  });
});

describe('decodeEvent', () => {
  it('accepts an event whose payload matches its type', () => {
    const d = decodeEvent(envelope('turn.started', { turn_id: 't-1', text: 'oi' }));
    expect(d.ok).toBe(true);
  });

  it('ignores fields it does not know, so a newer daemon stays readable', () => {
    const d = decodeEvent(envelope('session.renamed', { name: 'x', added_later: 3 }));
    expect(d.ok).toBe(true);
  });

  it('reads a nil slice encoded as null as an empty list', () => {
    expect(decodeEvent(envelope('plan.updated', { items: null })).ok).toBe(true);
  });

  it('says why an event cannot be read, and keeps whatever names its session', () => {
    const cases: Array<[unknown, RegExp]> = [
      ['not an object', /não é um objeto/],
      [{ ...envelope('turn.started', { turn_id: 't' }), session_id: '' }, /de qual sessão/],
      [{ ...envelope('turn.started', { turn_id: 't' }), seq: 0 }, /seq/],
      [{ ...envelope('turn.started', { turn_id: 't' }), at: 'ontem' }, /instante/],
      [envelope('turn.exploded', {}), /não conhece: turn.exploded/],
      [envelope('turn.started', { text: 'sem id' }), /turn_id está ausente/],
      [envelope('tool.completed', { tool_call_id: 'c', ok: 'yes', output: '', truncated: false }), /ok deveria ser boolean/],
      [envelope('turn.completed', { turn_id: 't', reason: 'done', usage: { input_tokens: '1' } }), /usage.input_tokens/],
    ];
    for (const [raw, why] of cases) {
      const d = decodeEvent(raw);
      expect(d.ok).toBe(false);
      if (!d.ok) expect(d.problem.reason).toMatch(why);
    }
    const bad = decodeEvent(envelope('turn.started', {}));
    expect(bad.ok ? null : bad.problem.sessionId).toBe('s-1');
  });
});
