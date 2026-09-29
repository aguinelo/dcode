import { describe, expect, it } from 'vitest';
import { asOf } from '../../src/fixtures/record';
import { RECORDED_AT, recording, referenceStates } from '../../src/fixtures/recording';
import { nameParts } from '../../src/renderer/host';
import { delegation, flowBlocks, toolSummary } from '../../src/state/flow';
import { clock, elapsed, since } from '../../src/state/format';
import { applyRaw, emptySessions } from '../../src/state/sessions';

// The visual check compares literally, so the recording has to say what the
// references show. These are the reference numbers, derived from the events.
const state = applyRaw(emptySessions, recording.events);
const now = Date.parse(RECORDED_AT);

describe('the recording behind the reference states', () => {
  it('opens each reference on its own session', () => {
    expect(referenceStates).toEqual({
      '02-janela-principal-rodando': 's-catalogar',
      '03-janela-principal-aprovacao': 's-tokens',
    });
  });

  it('shows 02 running for 1m42s, delegated to four children of which two came back', () => {
    const v = state.byId['s-catalogar']!;
    expect(v.state).toBe('running');
    expect(elapsed(since(v.turn!.startedAt, now))).toBe('1m42s');
    const blocks = flowBlocks(v.entries);
    const grep = blocks.find((b) => b.kind === 'tools');
    expect(grep?.kind === 'tools' && toolSummary(grep.calls[0]!)).toBe('11 matches · 4 arquivos');
    const d = blocks.find((b) => b.kind === 'delegation');
    expect(d?.kind === 'delegation' && delegation(d.calls)).toMatchObject({ count: 4, finished: 2, disjoint: true });
  });

  it('shows 03 waiting 1:35 for an answer, after a read and a write', () => {
    const v = state.byId['s-tokens']!;
    expect(v.state).toBe('blocked');
    const approval = v.entries.find((e) => e.kind === 'approval');
    expect(approval && clock(since(approval.at, now))).toBe('1:35');
    const tools = flowBlocks(v.entries).find((b) => b.kind === 'tools');
    expect(tools?.kind === 'tools' && tools.calls.map(toolSummary)).toEqual(['96 linhas', 'criado, 54 linhas']);
    expect([v.added, v.removed]).toEqual([54, 0]);
  });

  it('replays as if it had just happened, keeping every interval', () => {
    const later = now + 3 * 86_400_000;
    const moved = applyRaw(emptySessions, asOf(recording, later));
    const approval = moved.byId['s-tokens']!.entries.find((e) => e.kind === 'approval');
    expect(approval && clock(since(approval.at, later))).toBe('1:35');
    expect(moved.byId['s-catalogar']!.info!.created_at).toBe(new Date(now - 180_000 + 3 * 86_400_000).toISOString());
  });
});

describe('the footer', () => {
  it('shows the first name and the initials of the name the system has', () => {
    expect(nameParts('Aguinelo Koczkodai')).toEqual({ first: 'Aguinelo', initials: 'AK' });
    expect(nameParts('aguinelo')).toEqual({ first: 'aguinelo', initials: 'AG' });
  });
});
