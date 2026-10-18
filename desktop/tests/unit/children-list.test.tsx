import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { orchestraRecording, ORCHESTRA_AT } from '../../src/fixtures/orchestra-recording';
import { orchestrasRecording } from '../../src/fixtures/orchestras-recording';
import type * as P from '../../src/protocol/generated';
import { SessionsPane } from '../../src/renderer/crew/SessionsPane';
import { sessionGroups } from '../../src/state/crew';
import { crewLine, foldCrew } from '../../src/state/orchestrated';
import { emptyPrefs } from '../../src/state/prefs';
import { applyRaw, emptySessions } from '../../src/state/sessions';
import { projectsOf, rowOfConversation, rowsOfSessions, type Row } from '../../src/state/sidebar';

// One row per orchestrated project (D69, replacing D66's grouping): the
// sidebar lists the orchestrator alone, with a line counting its agents by
// phase; its children live in the plan and in the Agentes tab, and a
// reviewer is never a conversation of the list, in the search either.

const NOW = Date.parse(ORCHESTRA_AT);
const minute = 60_000;

function row(id: string, workspace: string, over: Partial<Row> = {}): Row {
  return { id, title: id, workspace, state: 'idle', started: NOW - 10 * minute, lastAt: new Date(NOW - minute).toISOString(), ...over };
}

function kid(id: string, state: string, over: Partial<P.Child> = {}): P.Child {
  return {
    id,
    parent_id: 's-orq',
    title: id,
    profile: 'fast',
    model: 'm',
    state,
    owns: ['a'],
    branch: `dcode/${id}`,
    worktree: `/w/${id}`,
    base: 'b',
    usage: { input_tokens: 0, output_tokens: 0 },
    started: ORCHESTRA_AT,
    ...over,
  };
}

describe('the sidebar of an orchestrated project', () => {
  const orq = row('s-orq', '/w/dcode', { role: 'orchestrator', projectId: 'p1', started: NOW - 30 * minute, title: 'Orquestrar a fatia 1' });
  const a = row('c-a', '/w/dcode/.dcode/children/c-a', { role: 'child', parentId: 's-orq', projectId: 'p1', title: 'listas: Unicos' });
  const b = row('c-b', '/w/dcode/.dcode/children/c-b', { role: 'child', parentId: 's-orq', projectId: 'p1', title: 'listas: Inverte' });
  const grand = row('c-b1', '/w/dcode/.dcode/children/c-b1', { role: 'child', parentId: 'c-b', projectId: 'p1', title: 'listas: testes' });
  const review = row('s-rev', '/w/dcode/.dcode/children/c-a', { role: 'reviewer', parentId: 's-orq', projectId: 'p1', title: 'Review commit 3aa49175' });
  const plain = row('s-plain', '/w/dcode', { started: NOW - 2 * minute, title: 'Uma conversa comum' });
  const all = [orq, a, b, grand, review, plain];

  it('lists only the orchestrator and the ordinary conversations', () => {
    const projects = projectsOf(all, emptyPrefs);
    expect(projects.map((p) => p.id)).toEqual(['/w/dcode']);
    expect(projects[0]!.rows.map((r) => r.id).sort()).toEqual(['s-orq', 's-plain']);
  });

  it('folds every child under its orchestrator, a grandchild too, and drops the reviewers', () => {
    const folded = foldCrew(all);
    expect(folded.top.map((r) => r.id)).toEqual(['s-orq', 's-plain']);
    expect(folded.crew['s-orq']!.map((r) => r.id)).toEqual(['c-a', 'c-b', 'c-b1']);
    expect(Object.values(folded.crew).flat().some((r) => r.role === 'reviewer')).toBe(false);
  });

  it('finds the orchestrator by project_id when the parent is not listed', () => {
    const orphan = row('c-z', '/w/dcode/.dcode/children/c-z', { role: 'child', parentId: 's-gone', projectId: 'p1' });
    expect(foldCrew([orq, orphan]).crew['s-orq']!.map((r) => r.id)).toEqual(['c-z']);
  });

  it('keeps a child with no orchestrator listed as a row of its own, never unreachable', () => {
    const alone = row('c-y', '/w/x/.dcode/children/c-y', { role: 'child', parentId: 's-gone', projectId: 'p9' });
    expect(projectsOf([alone], emptyPrefs).flatMap((p) => p.rows.map((r) => r.id))).toEqual(['c-y']);
  });

  it('never lists a reviewer, even with no orchestrator listed', () => {
    expect(projectsOf([review], emptyPrefs).flatMap((p) => p.rows)).toEqual([]);
  });

  it('counts the agents by phase, from the orchestrator’s list', () => {
    const line = crewLine(
      [kid('c-1', 'in_review'), kid('c-2', 'awaiting_person', { waiting: 'question' }), kid('c-3', 'pr', { waiting: 'merge' }), kid('c-4', 'merged')],
      [],
    );
    expect(line).toEqual({ text: '3 agentes · 1 em revisão · 2 esperando você', tone: 'warn' });
    expect(crewLine([kid('c-1', 'working'), kid('c-2', 'dispatched'), kid('c-3', 'pr'), kid('c-4', 'failed')], [])).toEqual({
      text: '4 agentes · 2 trabalhando · 1 em PR · 1 com falha',
      tone: 'faint',
    });
    expect(crewLine([kid('c-1', 'merged'), kid('c-2', 'closed')], [])).toEqual({ text: 'nenhum agente em curso', tone: 'faint' });
    expect(crewLine([], [])).toBeNull();
  });

  it('counts by the children’s rows when the orchestrator’s list is not known', () => {
    expect(crewLine(undefined, [{ ...a, state: 'running' }, { ...b, state: 'blocked' }, { ...grand, state: 'recorded' }])).toEqual({
      text: '2 agentes · 1 trabalhando · 1 esperando você',
      tone: 'warn',
    });
  });

  it('puts the orchestrator among the active while a child works', () => {
    const groups = sessionGroups(projectsOf([orq, { ...a, state: 'running' }], emptyPrefs), '', NOW);
    expect(groups.find((g) => g.key === 'ativas')!.rows.map((r) => r.id)).toEqual(['s-orq']);
  });

  it('finds a child by its short title under its orchestrator’s row, and never a reviewer', () => {
    const projects = projectsOf(all, emptyPrefs);
    const hit = sessionGroups(projects, 'unicos', NOW);
    expect(hit.flatMap((g) => g.rows.map((r) => r.id))).toEqual(['s-orq']);
    expect(hit.flatMap((g) => Object.values(g.hits))).toEqual(['listas: Unicos']);
    expect(sessionGroups(projects, 'review commit', NOW)).toEqual([]);
    // The orchestrator's own title is a plain hit, with no child named.
    const own = sessionGroups(projects, 'fatia', NOW);
    expect(own.flatMap((g) => g.rows.map((r) => r.id))).toEqual(['s-orq']);
    expect(own.flatMap((g) => Object.values(g.hits))).toEqual([]);
  });

  it('reads the orchestrator’s place from the daemon’s list as from the recording', () => {
    const listed = rowOfConversation({ id: 'c-a', title: 'listas: Unicos', workspace: '/w/c', state: 'idle', live: true, turns: 0, started: ORCHESTRA_AT, last_activity: ORCHESTRA_AT, parent_id: 's-orq', project_id: 'p1', role: 'child', named: true });
    expect([listed.title, listed.parentId, listed.projectId, listed.role]).toEqual(['listas: Unicos', 's-orq', 'p1', 'child']);
    const recorded = rowsOfSessions(applyRaw(emptySessions, orchestraRecording.events));
    const projects = projectsOf(recorded, emptyPrefs);
    expect(projects.map((p) => p.id)).toEqual(['/Users/aguinelo/work/dcode']);
    expect(projects[0]!.rows.map((r) => r.id)).toEqual(['s-orq']);
  });

  it('draws one row for the orchestrator in the Crew, with its line and no child', () => {
    const sessions = applyRaw(emptySessions, orchestraRecording.events);
    const html = renderToStaticMarkup(
      <SessionsPane
        projects={projectsOf(rowsOfSessions(sessions), emptyPrefs)}
        sessions={sessions}
        focusId={null}
        now={NOW}
        onPick={() => {}}
        onStart={() => {}}
        onLoop={() => {}}
        onOrchestrate={() => {}}
        listModels={async () => ({ ok: false, why: 'x' })}
        pickFolder={async () => null}
      />,
    );
    expect(html).toContain('data-session-id="s-orq"');
    expect(html).not.toContain('data-session-id="c-tui"');
    expect(html).not.toContain('data-depth');
    expect(html).toMatch(/data-crew="true"[^>]*>6 agentes · 2 trabalhando · 1 em revisão · 2 esperando você · 1 com falha</);
  });

  it('lists two orchestrated projects as one row each, beside an ordinary conversation', () => {
    const sessions = applyRaw(emptySessions, orchestrasRecording.events);
    const projects = projectsOf(rowsOfSessions(sessions), emptyPrefs);
    expect(projects.map((p) => [p.label, p.rows.map((r) => r.id).sort()])).toEqual([
      ['dcode', ['s-orq']],
      ['listas', ['s-listas', 's-listas-doc']],
    ]);
    const listas = projects[1]!;
    expect(listas.crew['s-listas']!.map((r) => r.title)).toEqual(['listas: Unicos', 'listas: Inverte', 'listas: Agrupa']);
    expect(crewLine(sessions.byId['s-listas']!.agents.children, listas.crew['s-listas']!)).toEqual({
      text: '3 agentes · 1 em revisão · 2 esperando você',
      tone: 'warn',
    });
  });
});
