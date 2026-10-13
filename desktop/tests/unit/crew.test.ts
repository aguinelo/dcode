import { describe, expect, it } from 'vitest';
import { contextPercent, groupCount, isCrewSection, OLDER_AFTER_MS, sessionGroups, skillsSeen, stateWord } from '../../src/state/crew';
import { emptyPrefs } from '../../src/state/prefs';
import { applyRaw, emptySessions } from '../../src/state/sessions';
import { projectsOf, type Row } from '../../src/state/sidebar';
import { created, log } from './helpers';

const now = Date.parse('2026-10-07T12:00:00.000Z');
const ago = (ms: number) => new Date(now - ms).toISOString();
const DAY = 24 * 60 * 60 * 1000;

function row(id: string, workspace: string, state: string, lastAt: string | null, title = id, started = 0): Row {
  return { id, title, workspace, state, started, lastAt };
}

describe('the groups of the sessions panel', () => {
  const rows = [
    row('a', '/w/dcode', 'running', ago(1000), 'leia o README', 3),
    row('b', '/w/dcode', 'recorded', ago(2 * DAY), 'conserte o parser', 2),
    row('c', '/w/eva', 'blocked', ago(5000), 'rode os testes de corrida', 1),
    row('d', '/w/eva', 'idle', ago(OLDER_AFTER_MS + DAY), 'migre os webhooks', 4),
    row('e', '/w/eva', 'recorded', null, 'sem instante', 5),
    row('f', '/w/dcode', 'recorded', ago(10 * DAY), 'velha de dez dias', 0),
  ];
  const projects = projectsOf(rows, emptyPrefs);

  it('puts what runs or waits first, the waiting on top, then each project, then the older ones newest first', () => {
    const groups = sessionGroups(projects, '', now);
    expect(groups.map((g) => [g.key, g.rows.map((r) => r.id), g.collapsible])).toEqual([
      ['ativas', ['c', 'a'], false],
      ['p:/w/dcode', ['b'], false],
      ['mais-antigas', ['d', 'f', 'e'], true],
    ]);
    expect(groupCount(groups)).toBe(6);
  });

  it('keeps only rows whose title or project has every word, without case or accents', () => {
    expect(sessionGroups(projects, 'PARSER', now).flatMap((g) => g.rows.map((r) => r.id))).toEqual(['b']);
    expect(sessionGroups(projects, 'eva corrida', now).flatMap((g) => g.rows.map((r) => r.id))).toEqual(['c']);
    expect(sessionGroups(projects, 'nada disso', now)).toEqual([]);
  });

  it('drops groups left empty', () => {
    const groups = sessionGroups(projectsOf([row('x', '/w/a', 'idle', ago(DAY))], emptyPrefs), '', now);
    expect(groups.map((g) => g.key)).toEqual(['p:/w/a']);
  });
});

describe('the words and numbers of the panel', () => {
  it('names each state, and says a stopped one is stopped', () => {
    expect(['running', 'blocked', 'recorded', 'idle'].map(stateWord)).toEqual(['rodando', 'esperando você', 'terminada', 'parada']);
  });

  it('knows the rail sections, and nothing else', () => {
    expect(isCrewSection('skills')).toBe(true);
    expect(isCrewSection('chat')).toBe(false);
    expect(isCrewSection(null)).toBe(false);
  });

  it('reads the context only once a turn reported it', () => {
    const before = log().add('session.created', created).fold();
    expect(contextPercent(before)).toBeNull();
    expect(contextPercent(undefined)).toBeNull();
    const after = log()
      .add('session.created', created)
      .add('turn.started', { turn_id: 't' })
      .add('turn.completed', {
        turn_id: 't',
        reason: 'end_turn',
        usage: { input_tokens: 1, output_tokens: 1, cache_read_tokens: 0, cache_write_tokens: 0, context_tokens: 50_000 },
      })
      .fold();
    expect(contextPercent(after)).toBe(25);
  });
});

describe('the skills seen in the followed conversations', () => {
  it('lists each skill once, by name, with how many conversations it entered', () => {
    const one = log('s-1')
      .add('session.created', created)
      .add('skill.loaded', { name: 'tdd', when_to_use: 'antes de mudar código' })
      .add('skill.loaded', { name: 'tdd' });
    const two = log('s-2')
      .add('session.created', { ...created, id: 's-2' })
      .add('skill.loaded', { name: 'go-review' })
      .add('skill.loaded', { name: 'tdd' });
    const state = applyRaw(emptySessions, [...one.events, ...two.events]);
    expect(skillsSeen(state)).toEqual([
      { name: 'go-review', whenToUse: '', sessions: 1 },
      { name: 'tdd', whenToUse: 'antes de mudar código', sessions: 2 },
    ]);
  });
});
