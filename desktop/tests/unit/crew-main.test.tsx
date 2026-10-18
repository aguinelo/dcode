import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import type * as P from '../../src/protocol/generated';
import type { CrewHandlers } from '../../src/renderer/crew/handlers';
import { SessionsPane } from '../../src/renderer/crew/SessionsPane';
import { ThreadPane, ThreadProof } from '../../src/renderer/crew/ThreadPane';
import { LoopSection } from '../../src/renderer/LoopRuler';
import { RecordedBody } from '../../src/renderer/RecordedBody';
import { DEFAULT_LOOK, lookOf } from '../../src/renderer/stored';
import { LoopContext, type LoopsView } from '../../src/renderer/useLoops';
import { arrivedSessions, emptyLoops, followLoop, type LoopsState } from '../../src/state/loop';
import { paneView } from '../../src/state/pane';
import { reviewOf } from '../../src/state/review';
import type { SessionView } from '../../src/state/session';
import { applyRaw, emptySessions } from '../../src/state/sessions';
import { projectsOf, type Row } from '../../src/state/sidebar';
import { created, log } from './helpers';

// The Crew is the main screen (D63), and it carries what the grid's maximized
// window carries: the loop's ruler with its review, and the criteria as the
// turn checks them, above the conversation.

const NOW = Date.parse('2026-09-28T17:10:00.000Z');

describe('the version of the screen', () => {
  it('is the Crew when nothing was chosen, and either one as chosen', () => {
    expect(DEFAULT_LOOK).toBe('crew');
    expect(lookOf(null)).toBe('crew');
    expect(lookOf('grade')).toBe('grade');
    expect(lookOf('crew')).toBe('crew');
    expect(lookOf('algo')).toBe('crew');
    // The fixture's reference states are the grid's, and ask for it.
    expect(lookOf(null, 'grade')).toBe('grade');
  });
});

function run(over: Partial<P.LoopRun> = {}): P.LoopRun {
  return {
    id: 'L',
    workspace: '/w/dcode',
    state: 'working',
    current: 1,
    session_id: 's2',
    started: '2026-09-28T17:00:00.000Z',
    specs: [
      { path: 'specs/a', phase: 'done', session_id: 's1' },
      { path: 'specs/b', phase: 'working', session_id: 's2' },
    ],
    ...over,
  };
}

const loopsOf = (...runs: P.LoopRun[]): LoopsState => ({
  ...emptyLoops,
  loaded: true,
  byId: Object.fromEntries(runs.map((r) => [r.id, r])),
  order: runs.map((r) => r.id),
});

describe('the Crew follows a loop in its one thread', () => {
  it('from any session the loop opened to the one it is in now', () => {
    const loops = loopsOf(run());
    expect(followLoop(loops, 's1')).toBe('s2');
    expect(followLoop(loops, 's2')).toBe('s2');
    expect(followLoop(loops, 'other')).toBe('other');
    expect(followLoop(loops, null)).toBeNull();
  });

  it('to the session a loop started here arrives at, and not a loop started elsewhere', () => {
    const before = loopsOf(run({ session_id: undefined }), run({ id: 'M', session_id: undefined }));
    const after = loopsOf(run(), run({ id: 'M', session_id: 's9' }));
    expect(arrivedSessions(before, after, ['L'])).toEqual(['s2']);
    expect(arrivedSessions(after, after, ['L'])).toEqual([]);
  });
});

const proposal: P.DoneProposal = {
  proposal_id: 'p',
  sequence: 1,
  proposed_at: '2026-09-28T17:05:00.000Z',
  path: 'specs/b/done.toml',
  criteria: [{ name: 'e2e', command: 'npm run e2e', exit_code: 1, class: 'acceptance', exit: 1 }],
};

function session(id: string, extra: (l: ReturnType<typeof log>) => void): SessionView {
  const l = log(id).add('session.created', { ...created, id });
  extra(l);
  return l.fold();
}

function view(loops: LoopsState, sessions: Record<string, SessionView>): LoopsView {
  return {
    of: (id) => (loops.byId.L && [loops.byId.L.session_id, ...loops.byId.L.specs.map((s) => s.session_id)].includes(id) ? loops.byId.L : null),
    start: async () => true,
    stopNow: () => {},
    stopAfter: () => {},
    openSession: () => {},
    review: (r) => reviewOf(r, sessions),
    startWork: () => {},
    openProposal: () => {},
  };
}

const row = (id: string, state = 'running'): Row => ({ id, title: `conversa ${id}`, workspace: '/w/dcode', state, started: 0, lastAt: '2026-09-28T17:09:00.000Z' });
const on = new Proxy({}, { get: () => () => {} }) as CrewHandlers;

function thread(s: SessionView, loops: LoopsState): string {
  const r = row(s.id, s.state);
  return renderToStaticMarkup(
    <LoopContext.Provider value={view(loops, { [s.id]: s })}>
      <ThreadProof sessionId={s.id} view={paneView(r, s, NOW)} now={NOW} measure={null} />
    </LoopContext.Provider>,
  );
}

describe('the thread of the Crew', () => {
  it('opens, with no conversation in focus, on the mascot and Nova sessão', () => {
    const html = renderToStaticMarkup(
      <ThreadPane row={null} view={null} session={undefined} now={NOW} verbTick={0} draft="" history={undefined} onRetryHistory={() => {}} on={on} git={{ kind: 'idle' }} measure={null} />,
    );
    expect(html).toContain('Nova sessão');
  });

  it('carries the loop’s ruler and the criteria as the turn checks them, above the conversation', () => {
    const s = session('s2', (l) =>
      l
        .add('turn.started', { turn_id: 't1', text: 'trabalhe specs/b' })
        .add('done.checked', { turn_id: 't1', cycle: 3, met: ['tsc'], unmet: ['e2e'], moved: 'stalled', stall: 1, stall_limit: 3 }),
    );
    const html = thread(s, loopsOf(run()));
    expect(html).toContain('data-loop-id="L"');
    expect(html).toContain('ciclo 3 · parado 1 de 3');
    expect(html).toContain('✓ tsc');
    expect(html).toContain('Parar agora');
  });

  it('carries the review card of a loop waiting for one, and the Pronto tab only points to it', () => {
    const s = session('s2', (l) => l.add('done.proposed', proposal));
    const loops = loopsOf(run({ state: 'awaiting_review', specs: [{ path: 'specs/b', phase: 'proposed', session_id: 's2', proposal: 'specs/b/done.toml' }], current: 0 }));
    expect(thread(s, loops)).toContain('data-loop-review="L"');
    const tab = renderToStaticMarkup(
      <LoopContext.Provider value={view(loops, { s2: s })}>
        <LoopSection sessionId="s2" />
      </LoopContext.Provider>,
    );
    expect(tab).not.toContain('data-loop-review');
    expect(tab).toContain('A proposta espera revisão no alto da conversa.');
  });

  it('draws nothing of a loop for a conversation that is not one’s', () => {
    const html = thread(session('s7', () => {}), loopsOf(run()));
    expect(html).not.toContain('data-loop-id');
    expect(html).toContain('sem definição de pronto');
  });
});

describe('the sessions list of the Crew', () => {
  it('reads each row’s state, as the grid’s sidebar does', () => {
    const sessions = applyRaw(emptySessions, log('s3').add('session.created', { ...created, id: 's3' }).events);
    const projects = projectsOf([row('s3', 'idle'), row('s4', 'recorded')], { order: [], labels: {}, collapsed: {} });
    const html = renderToStaticMarkup(
      <SessionsPane projects={projects} sessions={sessions} focusId={null} now={NOW} onPick={() => {}} onStart={() => {}} onLoop={() => {}} onOrchestrate={() => {}}
        listModels={async () => ({ ok: false, why: '' })} pickFolder={async () => null} />,
    );
    expect(html).toContain('data-session-id="s3" data-state="idle"');
    expect(html).toContain('data-session-id="s4" data-state="recorded"');
    expect(html).toContain('aria-label="Nova sessão"');
  });
});

// A conversation that ended is continued by writing (D33, D42) — but not an
// orchestrator's agent, which a new ordinary session would strip of its role
// and its tools, and not the orchestrator, which is always the live one.
describe('the field under an ended conversation', () => {
  const ended = (over: Partial<Row>): Row => ({ id: 'c-a', title: 'índice', workspace: '/w/c', state: 'recorded', started: 0, lastAt: null, turns: 2, ...over });
  const draw = (row: Row) =>
    renderToStaticMarkup(
      <RecordedBody row={row} history={{ state: 'loading' }} compact={false} now={NOW} draft="" onDraft={() => {}} onContinue={() => {}} onRetry={() => {}} />,
    );

  it('continues an ordinary conversation in a new session', () => {
    const html = draw(ended({}));
    expect(html).toContain('Escrever continua a conversa numa sessão nova');
    expect(html).toContain('<textarea');
  });

  it('offers no field under a child or a reviewer that ended, and points to the orchestrator', () => {
    for (const role of ['child', 'reviewer']) {
      const html = draw(ended({ role, parentId: 's-orq', projectId: 'p1' }));
      expect(html).not.toContain('<textarea');
      expect(html).not.toContain('sessão nova, com o histórico');
      expect(html).toContain('orquestradora');
    }
  });

  it('says writing to the orchestrator opens the live one of the project', () => {
    const html = draw(ended({ id: 's-orq', role: 'orchestrator', projectId: 'p1' }));
    expect(html).not.toContain('sessão nova');
    expect(html).toContain('orquestradora');
  });
});
