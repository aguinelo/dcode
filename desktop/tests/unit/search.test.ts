import { describe, expect, it } from 'vitest';
import type * as P from '../../src/protocol/generated';
import { emptyPrefs } from '../../src/state/prefs';
import { lastEventText, longState, searchGroups, searchSide } from '../../src/state/search';
import { projectsOf, rowOfConversation } from '../../src/state/sidebar';

function conversation(over: Partial<P.Conversation>): P.Conversation {
  return {
    id: 'x',
    title: '',
    workspace: '/w/dcode',
    state: 'idle',
    live: true,
    turns: 1,
    started: '2026-09-30T10:00:00Z',
    last_activity: '2026-09-30T10:05:00Z',
    ...over,
  };
}

const rows = [
  conversation({ id: 'parser', title: 'conserte o parser', started: '2026-09-30T09:00:00Z' }),
  conversation({ id: 'readme', title: 'escreva o README', started: '2026-09-30T11:00:00Z' }),
  conversation({ id: 'sessao', title: 'Ajuste a sessão', workspace: '/w/eva', state: 'running', last_activity: '2026-09-30T12:00:00Z' }),
  conversation({ id: 'old', title: 'migre o banco', workspace: '/w/eva', state: 'recorded', live: false }),
].map(rowOfConversation);
const projects = projectsOf(rows, emptyPrefs);
const ids = (q: string, scope: 'todas' | 'ativas' | 'projeto' = 'todas', project: string | null = null) =>
  searchGroups(projects, q, scope, project).map((g) => [g.label, g.rows.map((r) => r.id)]);

describe('⌘K', () => {
  it('lists the active first, then each project in the sidebar’s order', () => {
    expect(ids('')).toEqual([
      ['Ativas', ['sessao']],
      ['eva', ['old']],
      ['dcode', ['readme', 'parser']],
    ]);
  });

  it('finds by title and by project, without case or accents, every word', () => {
    expect(ids('readme')).toEqual([['dcode', ['readme']]]);
    expect(ids('sessao')).toEqual([['Ativas', ['sessao']]]);
    expect(ids('eva banco')).toEqual([['eva', ['old']]]);
    expect(ids('parser banco')).toEqual([]);
  });

  it('keeps to what is active, or to the open session’s project', () => {
    expect(ids('', 'ativas')).toEqual([['Ativas', ['sessao']]]);
    expect(ids('', 'projeto', '/w/dcode')).toEqual([['dcode', ['readme', 'parser']]]);
  });

  it('says what each one is doing, and how its last turn was sealed', () => {
    const now = Date.parse('2026-09-30T12:05:00Z');
    const [readme, sessao] = [rows[1]!, rows[2]!];
    expect(searchSide(sessao, now)).toEqual({ text: 'rodando', tone: 'dim' });
    expect(searchSide(readme, now)).toEqual({ text: '2h', tone: 'faint' });
    expect(longState(rowOfConversation(conversation({ verification: 'passed' })))).toEqual({ text: '✓ verified', tone: 'ok' });
    expect(longState(rows[3]!).text).toMatch(/^Terminada/);
    expect(lastEventText(rowOfConversation(conversation({ last_event: 'turn.completed' })), now)).toBe('turno terminou · 2h');
    expect(lastEventText(rowOfConversation(conversation({ last_event: 'plan.updated' })), now)).toBe('plan.updated · 2h');
  });
});
