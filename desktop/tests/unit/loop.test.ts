import { describe, expect, it } from 'vitest';
import * as P from '../../src/protocol/generated';
import { decodeLoopChange, decodeLoopRun, decodeSpecs, LOOP_CHANGE_KINDS } from '../../src/protocol/loops';
import { emptyAttention, hold } from '../../src/state/attention';
import { parseInput } from '../../src/state/commands';
import {
  applyLoopChange,
  carryHolds,
  emptyLoops,
  liveLoopIn,
  loopArgsOf,
  loopNews,
  loopOfSession,
  loopRuler,
  loopsUnavailable,
  planWithLoops,
  startRefusal,
  type LoopsState,
} from '../../src/state/loop';
import { rowOfConversation, type Row } from '../../src/state/sidebar';

// A loop the daemon runs, as the window starts, follows and stops it (D55).

function run(id: string, over: Partial<P.LoopRun> = {}): P.LoopRun {
  return { id, workspace: '/w/dcode', state: 'working', specs: [], current: -1, started: '2026-10-14T10:00:00Z', ...over };
}

function loops(...runs: P.LoopRun[]): LoopsState {
  return applyLoopChange(emptyLoops, { kind: 'snapshot', loops: runs });
}

function row(id: string, over: Partial<P.Conversation> = {}): Row {
  return rowOfConversation({
    id,
    title: id,
    workspace: '/w/dcode',
    state: 'running',
    live: true,
    turns: 1,
    started: '2026-10-14T10:00:00Z',
    last_activity: '2026-10-14T10:05:00Z',
    ...over,
  });
}

describe('the stream of loops, read at the boundary', () => {
  it('reads a snapshot and a change as the daemon sends them', () => {
    const r = run('l1', { specs: [{ path: 'specs/a', phase: 'working', session_id: 's1', standing: { verification: 'failed', unmet: ['e2e'] } }] });
    expect(decodeLoopChange({ kind: 'snapshot', loops: [r] })).toEqual({ ok: true, change: { kind: 'snapshot', loops: [r] } });
    expect(decodeLoopChange({ kind: 'snapshot', loops: null })).toEqual({ ok: true, change: { kind: 'snapshot', loops: [] } });
    expect(decodeLoopChange({ kind: 'changed', loop: r })).toEqual({ ok: true, change: { kind: 'changed', loop: r } });
  });

  it('holds the kinds it reads to the generated constants', () => {
    expect([...LOOP_CHANGE_KINDS].sort()).toEqual([P.LoopChanged, P.LoopSnapshot].sort());
  });

  it('says what does not match, never guesses', () => {
    expect(decodeLoopChange({ kind: 'changed', loop: { id: 'l1' } })).toMatchObject({ ok: false });
    expect(decodeLoopChange({ kind: 'removed' })).toEqual({ ok: false, reason: 'lista de loops: tipo de mudança que esta versão não conhece: removed' });
    expect(decodeLoopRun({ ...run('l1'), specs: [{ path: 3 }] })).toMatchObject({ ok: false });
    expect(decodeLoopRun(run('l1'))).toEqual({ ok: true, value: run('l1') });
  });

  it('reads the folders the daemon declares', () => {
    const folder = { path: 'specs/a', criteria: 0, unmet: 0, pending: false, measured: false };
    expect(decodeSpecs({ specs: [folder] })).toEqual({ ok: true, value: [folder] });
    expect(decodeSpecs({ specs: null })).toEqual({ ok: true, value: [] });
    expect(decodeSpecs({ specs: [{ path: 'x' }] })).toMatchObject({ ok: false });
  });
});

describe('/loop in the field', () => {
  it('sends the argument as it was typed: the grammar is the daemon’s', () => {
    expect(parseInput('/loop specs/home --protect tests/**')).toEqual({ kind: 'loop', args: 'specs/home --protect tests/**' });
    expect(parseInput('  /loop implemente as  specs pendentes ')).toEqual({ kind: 'loop', args: 'implemente as  specs pendentes' });
  });

  it('says what a bare /loop lacks', () => {
    const got = parseInput('/loop');
    expect(got.kind === 'bad' && got.why).toContain('/loop precisa de uma pasta de spec ou de um objetivo');
  });
});

describe('the list of loops the window keeps', () => {
  it('follows the snapshot and each change, and says when the daemon has none to give', () => {
    let s = loops(run('a'));
    expect(s.loaded).toBe(true);
    s = applyLoopChange(s, { kind: 'changed', loop: run('b') });
    s = applyLoopChange(s, { kind: 'changed', loop: run('a', { state: 'stopped' }) });
    expect(s.order).toEqual(['a', 'b']);
    expect(s.byId.a?.state).toBe('stopped');
    expect(loopsUnavailable(s, 'sem rota').problem).toBe('sem rota');
  });

  it('knows the live loop of a project and the loop a session belongs to', () => {
    const s = loops(
      run('old', { state: 'finished', session_id: 's0' }),
      run('now', { session_id: 's2', specs: [{ path: 'specs/a', phase: 'done', session_id: 's1' }, { path: 'specs/b', phase: 'working', session_id: 's2' }] }),
    );
    expect(liveLoopIn(s, '/w/dcode')?.id).toBe('now');
    expect(liveLoopIn(s, '/w/other')).toBeNull();
    expect(loopOfSession(s, 's1')?.id).toBe('now');
    expect(loopOfSession(s, 's2')?.id).toBe('now');
    expect(loopOfSession(s, 's0')?.id).toBe('old');
    expect(loopOfSession(s, 'zz')).toBeNull();
  });
});

describe('a loop on the grid: one window that follows the current session', () => {
  const since = { ...emptyAttention, since: '2026-10-14T09:00:00Z' };
  const twoFolders = run('L', {
    session_id: 's2',
    current: 1,
    specs: [
      { path: 'specs/a', phase: 'done', session_id: 's1' },
      { path: 'specs/b', phase: 'working', session_id: 's2' },
    ],
  });

  it('shows only the current session of a loop, in the place the last one had', () => {
    const rows = [row('x'), row('s1', { state: 'idle', last_event: 'turn.completed' }), row('s2')];
    const plan = planWithLoops(rows, since, ['s1', 'x'], 6, loops(twoFolders));
    expect(plan.ids).toEqual(['s2', 'x']);
  });

  it('puts a loop waiting for review with what waits for you', () => {
    const review = run('L', { state: 'awaiting_review', session_id: 's1', current: 0, specs: [{ path: 'specs/a', phase: 'proposed', session_id: 's1' }] });
    const plan = planWithLoops([row('s1', { state: 'idle', last_event: 'turn.completed' })], since, [], 6, loops(review));
    expect(plan.claims.s1).toBe('waiting');
  });

  it('carries a held window to the session the loop moved to', () => {
    const a = carryHolds(hold(since, 's1'), loops(twoFolders));
    expect(a.pinned).toEqual(['s2']);
    const same = hold(since, 's2');
    expect(carryHolds(same, loops(twoFolders))).toBe(same);
  });
});

describe('the ruler at the top of a loop’s window', () => {
  const now = Date.parse('2026-10-14T10:12:04Z');

  it('says the goal, the folder it is in of how many, and each folder’s phase', () => {
    const v = loopRuler(
      run('L', {
        goal: 'implemente as specs pendentes',
        current: 1,
        session_id: 's2',
        specs: [
          { path: 'specs/a', phase: 'done', session_id: 's1' },
          { path: 'specs/b', phase: 'working', session_id: 's2' },
          { path: 'specs/c', phase: 'pending' },
        ],
      }),
      now,
    );
    expect(v.title).toBe('implemente as specs pendentes');
    expect(v.state).toBe('trabalhando');
    expect(v.step).toBe('pasta 2 de 3 · specs/b');
    expect(v.elapsed).toBe('12m04s');
    expect(v.live).toBe(true);
    expect(v.folders.map((f) => `${f.label}${f.current ? '*' : ''}`)).toEqual(['pronta', 'trabalhando*', 'pendente']);
  });

  it('names a folder run by the folder, and says it is surveying before any folder', () => {
    const v = loopRuler(run('L', { state: 'surveying' }), now);
    expect(v.title).toBe('/loop');
    expect(v.state).toBe('lendo as pastas');
    expect(v.step).toBeNull();
    const one = loopRuler(run('L', { current: 0, specs: [{ path: 'specs/home', phase: 'qualifying' }], state: 'qualifying' }), now);
    expect(one.title).toBe('specs/home');
    expect(one.step).toBe('pasta 1 de 1 · specs/home');
  });

  it('ends with the verdict, the error and the warnings, said', () => {
    const v = loopRuler(
      run('L', {
        state: 'finished',
        finished: '2026-10-14T10:30:00Z',
        verdict: { worked: 2, met: 5, total: 7, unmet: ['e2e'], unchecked: ['a11y'], unfinished: ['specs/b'] },
        warnings: ['o survey falhou'],
      }),
      now,
    );
    expect(v.live).toBe(false);
    expect(v.tone).toBe('warn');
    expect(v.elapsed).toBe('30m00s');
    expect(v.verdict).toBe('2 pastas trabalhadas · 5 de 7 critérios · não cumpridos: e2e · não conferidos: a11y · ficaram por fazer: specs/b');
    expect(v.warnings).toEqual(['o survey falhou']);
    const failed = loopRuler(run('L', { state: 'failed', error: 'o turno parou' }), now);
    expect(failed).toMatchObject({ state: 'falhou', tone: 'err', error: 'o turno parou' });
  });
});

describe('what the window says about a loop', () => {
  it('composes the argument the sheet sends as a person would type it', () => {
    expect(loopArgsOf({ target: ' specs/home ', protect: ['tests/**', ' '] })).toEqual({ ok: true, args: 'specs/home --protect tests/**' });
    expect(loopArgsOf({ target: '', protect: [] })).toMatchObject({ ok: false });
    expect(loopArgsOf({ target: 'x', protect: ['a b'] })).toEqual({ ok: false, why: 'Um caminho protegido não tem espaço: “a b”.' });
    expect(loopArgsOf({ target: '--protect', protect: [] })).toMatchObject({ ok: false });
  });

  it('says a loop that failed, once, with why', () => {
    const before = loops(run('L'));
    const after = applyLoopChange(before, { kind: 'changed', loop: run('L', { state: 'failed', error: 'sem modelo' }) });
    expect(loopNews(before, after)).toEqual(['O loop em /w/dcode falhou: sem modelo']);
    expect(loopNews(after, after)).toEqual([]);
    expect(loopNews(emptyLoops, after)).toEqual([]);
  });

  it('says why a loop did not start, with the daemon’s reason', () => {
    expect(startRefusal(P.CodeLoopAlreadyRunning, 'loop x is already running in /w')).toBe(
      'Já há um loop vivo neste projeto, e outro não começa enquanto ele roda: loop x is already running in /w',
    );
    expect(startRefusal('invalid_input', '--x is not a flag here')).toBe('O loop não começou: --x is not a flag here');
  });
});
