import { describe, expect, it } from 'vitest';
import { age, clock, elapsed } from '../../src/state/format';
import { dropBefore, emptyPrefs, moveProject, parsePrefs, relabel, setAllCollapsed } from '../../src/state/prefs';
import { applyRaw, emptySessions } from '../../src/state/sessions';
import { countStates, projectsOf, sessionSide, sessionTitle } from '../../src/state/sidebar';
import { recordedClient, recording, RECORDED_AT } from '../../src/fixtures/recording';

const state = applyRaw(emptySessions, recording.events);
const now = Date.parse(RECORDED_AT);

describe('the recording', () => {
  it('reads without a single problem', () => {
    expect(state.problems).toEqual([]);
    for (const v of Object.values(state.byId)) {
      expect(v.entries.filter((e) => e.kind === 'note' && e.note.kind === 'protocol')).toEqual([]);
    }
  });
});

describe('the sidebar', () => {
  const projects = projectsOf(state, recordedClient.prefs);

  it('orders projects by recent activity and sessions newest first, as the handoff shows them', () => {
    expect(projects.map((p) => p.label)).toEqual(['dcode', 'bizpack', 'eva']);
    expect(projects[0]?.sessions.map(sessionTitle)).toEqual([
      'Catalogar chamadas de .Save',
      'Extrair tokens de cor',
      'Fazer a suíte de integração passar',
      'Stop the done check after an interrupt',
      'Fix Rebuild orphaning multi-call batch',
      'Make specguard reject ambiguous invariants',
    ]);
  });

  it('marks what waits for you, leaves the running bare, and dates the idle', () => {
    const sides = projects.flatMap((p) => p.sessions.map((s) => sessionSide(s, now)?.text ?? ''));
    expect(sides).toEqual(['', 'aprovar', '', '2h', 'ontem', 'ontem', '1sem', '2sem', '3sem', '1mês']);
  });

  it('says a collapsed project’s count, and who waits before who runs', () => {
    expect(projects.map((p) => p.meta?.text ?? null)).toEqual([null, null, '2']);
    const allCollapsed = projectsOf(state, setAllCollapsed(emptyPrefs, projects.map((p) => p.id), true));
    expect(allCollapsed.map((p) => [p.meta?.text, p.meta?.tone])).toEqual([
      ['1 espera', 'warn'],
      ['2', 'faint'],
      ['2', 'faint'],
    ]);
  });

  it('counts what the bottom bar says, from the states on the wire', () => {
    expect(countStates(Object.values(state.byId))).toEqual({ running: 2, blocked: 1 });
  });

  it('keeps the person’s order and labels, and a derived title when nobody named the session', () => {
    const ids = projects.map((p) => p.id);
    const moved = moveProject(emptyPrefs, ids, ids[2] as string, 0);
    const labelled = relabel(moved, ids[0] as string, 'meu dcode');
    expect(projectsOf(state, labelled).map((p) => p.label)).toEqual(['eva', 'meu dcode', 'bizpack']);
    expect(relabel(labelled, ids[0] as string, '   ')).toBe(labelled);
    expect(dropBefore(emptyPrefs, ids, ids[1] as string, ids[0] as string).order).toEqual([ids[1], ids[0], ids[2]]);

    const unnamed = applyRaw(emptySessions, [
      { seq: 1, session_id: 'x', type: 'turn.started', at: RECORDED_AT, payload: { turn_id: 't', text: 'linha um\nlinha dois' } },
    ]);
    expect(sessionTitle(unnamed.byId.x!)).toBe('linha um');
  });
});

describe('local preferences', () => {
  it('reads what it stored, and says so when what is stored cannot be read', () => {
    expect(parsePrefs(null)).toEqual({ prefs: null, problem: null });
    expect(parsePrefs(JSON.stringify(emptyPrefs)).prefs).toEqual(emptyPrefs);
    expect(parsePrefs('{nope').problem).toMatch(/ilegíveis/);
    expect(parsePrefs(JSON.stringify({ order: 'x', labels: {}, collapsed: {} })).problem).toMatch(/formato/);
  });
});

describe('durations', () => {
  it('reads as the design writes them', () => {
    const s = 1000;
    expect([age(30 * s), age(12 * 60 * s), age(2 * 3600 * s + 5), age(30 * 3600 * s), age(3 * 86400 * s)]).toEqual([
      'agora',
      '12min',
      '2h',
      'ontem',
      '3d',
    ]);
    expect([age(8 * 86400 * s), age(22 * 86400 * s), age(35 * 86400 * s), age(70 * 86400 * s)]).toEqual([
      '1sem',
      '3sem',
      '1mês',
      '2meses',
    ]);
    expect([clock(48 * s), clock(95 * s), clock(3729 * s)]).toEqual(['0:48', '1:35', '1:02:09']);
    expect([elapsed(42 * s), elapsed(102 * s), elapsed(922 * s), elapsed(3720 * s)]).toEqual(['42s', '1m42s', '15m22s', '1h02m']);
  });
});
