import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { orchestraRecording, ORCHESTRA_AT } from '../../src/fixtures/orchestra-recording';
import { SessionsPane } from '../../src/renderer/crew/SessionsPane';
import { sessionGroups } from '../../src/state/crew';
import { emptyPrefs } from '../../src/state/prefs';
import { applyRaw, emptySessions } from '../../src/state/sessions';
import { projectsOf, rowOfConversation, rowsOfSessions, type Row } from '../../src/state/sidebar';

// An orchestrator's children work each in its own worktree; the list of
// sessions still shows them under their project, beneath the orchestrator
// (D66), instead of one project per worktree (the gap D64 left).

const NOW = Date.parse(ORCHESTRA_AT);
const minute = 60_000;

function row(id: string, workspace: string, over: Partial<Row> = {}): Row {
  return { id, title: id, workspace, state: 'idle', started: NOW - 10 * minute, lastAt: new Date(NOW - minute).toISOString(), ...over };
}

describe('the children of an orchestrator in the list of sessions', () => {
  const orq = row('s-orq', '/w/dcode', { role: 'orchestrator', projectId: 'p1', started: NOW - 30 * minute });
  const a = row('c-a', '/w/dcode/.dcode/children/c-a', { role: 'child', parentId: 's-orq', projectId: 'p1', started: NOW - 20 * minute });
  const b = row('c-b', '/w/dcode/.dcode/children/c-b', { role: 'child', parentId: 's-orq', projectId: 'p1', started: NOW - 15 * minute });
  const grand = row('c-b1', '/w/dcode/.dcode/children/c-b1', { role: 'child', parentId: 'c-b', projectId: 'p1', started: NOW - 5 * minute });
  const plain = row('s-plain', '/w/dcode', { started: NOW - 2 * minute });

  it('go under the project of their orchestrator, not one project per worktree', () => {
    const projects = projectsOf([orq, a, b, grand, plain], emptyPrefs);
    expect(projects.map((p) => p.id)).toEqual(['/w/dcode']);
    expect(projects[0]!.rows.map((r) => r.id).sort()).toEqual(['c-a', 'c-b', 'c-b1', 's-orq', 's-plain']);
  });

  it('find the project by project_id when their parent is not listed', () => {
    const orphan = row('c-z', '/w/dcode/.dcode/children/c-z', { role: 'child', parentId: 's-gone', projectId: 'p1' });
    expect(projectsOf([orq, orphan], emptyPrefs).map((p) => p.id)).toEqual(['/w/dcode']);
    // Nothing to place it by: its own folder, never dropped.
    const alone = row('c-y', '/w/x/.dcode/children/c-y', { role: 'child', parentId: 's-gone', projectId: 'p9' });
    expect(projectsOf([alone], emptyPrefs).map((p) => p.id)).toEqual(['/w/x/.dcode/children/c-y']);
  });

  it('are indented under the orchestrator, a grandchild under its parent', () => {
    const groups = sessionGroups(projectsOf([orq, a, b, grand, plain], emptyPrefs), '', NOW);
    const g = groups.find((x) => x.key === 'p:/w/dcode')!;
    expect(g.rows.map((r) => [r.id, g.depth[r.id] ?? 0])).toEqual([
      ['s-plain', 0],
      ['s-orq', 0],
      ['c-b', 1],
      ['c-b1', 2],
      ['c-a', 1],
    ]);
  });

  it('stand at the top of a group their parent is not in', () => {
    const running = { ...a, state: 'running' };
    const groups = sessionGroups(projectsOf([orq, running], emptyPrefs), '', NOW);
    const active = groups.find((x) => x.key === 'ativas')!;
    expect(active.rows.map((r) => [r.id, active.depth[r.id] ?? 0])).toEqual([['c-a', 0]]);
  });

  it('read the orchestrator’s place from the daemon’s list as from the recording', () => {
    const listed = rowOfConversation({ id: 'c-a', title: 't', workspace: '/w/c', state: 'idle', live: true, turns: 0, started: ORCHESTRA_AT, last_activity: ORCHESTRA_AT, parent_id: 's-orq', project_id: 'p1', role: 'child' });
    expect([listed.parentId, listed.projectId, listed.role]).toEqual(['s-orq', 'p1', 'child']);
    const recorded = rowsOfSessions(applyRaw(emptySessions, orchestraRecording.events));
    const projects = projectsOf(recorded, emptyPrefs);
    expect(projects.map((p) => p.id)).toEqual(['/Users/aguinelo/work/dcode']);
  });

  it('draw indented in the Crew', () => {
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
    expect(html).toMatch(/class="crew-item crew-child"[^>]*data-session-id="c-tui"/);
    expect(html).toContain('data-depth="1"');
  });
});
