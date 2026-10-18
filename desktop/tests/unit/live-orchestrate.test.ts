import { describe, expect, it } from 'vitest';
import type * as P from '../../src/protocol/generated';
import { LiveStore } from '../../src/renderer/live';
import type { Answer, DcodeApi, SessionEvents } from '../../src/shared/api';
import type { Row } from '../../src/state/sidebar';

// The window acts on its orchestrator (D66): it opens a project's orchestrator
// for a folder, reads its children and questions when its conversation opens,
// and answers a question — the first answer wins, and losing is said.

const project: P.Project = { id: 'p1', root: '/w/dcode', main: 'main', forge: 'local', autonomy: 'step', orchestrator_id: 's-orq', created: '2026-10-18T10:00:00Z' };
const orchestrator = {
  id: 's-orq',
  state: 'idle',
  workspace: '/w/dcode',
  model: 'Opus 5.5',
  sandbox_mode: 'read-only',
  mode: 'plan',
  created_at: '2026-10-18T10:00:00Z',
  last_seq: 1,
  done_criteria: 0,
  role: 'orchestrator',
  project_id: 'p1',
};

function child(id: string, state = 'working'): P.Child {
  return { id, parent_id: 's-orq', title: id, profile: 'fast', model: 'm', state, owns: ['a/'], branch: 'b', worktree: '/w/c', base: 'x', usage: { input_tokens: 0, output_tokens: 0 }, started: '2026-10-18T10:01:00Z' };
}

const question = (id: string, kind = 'merge'): P.Question => ({
  id,
  kind,
  text: `${kind}?`,
  recommended: kind === 'ceiling' ? 'stop' : 'merge',
  why: 'porque',
  options: kind === 'ceiling' ? [{ id: 'continue', label: 'Continue' }, { id: 'stop', label: 'Stop' }] : [{ id: 'merge', label: 'Merge' }, { id: 'return', label: 'Return' }, { id: 'close', label: 'Close' }],
  asked: '2026-10-18T10:02:00Z',
});

function orchestrateApi() {
  const asked: string[] = [];
  const answers: Record<string, Answer<unknown>> = {};
  const events: ((b: SessionEvents) => void)[] = [];
  const answer = (key: string, what: string): Promise<Answer<unknown>> => {
    asked.push(what);
    return Promise.resolve(answers[key] ?? { ok: false, refusal: { code: 'internal', message: 'not under test' } });
  };
  const quiet = () => () => {};
  const api = {
    onDaemon: quiet,
    onConversations: quiet,
    onSessionEvents: (l: (b: SessionEvents) => void) => (events.push(l), () => {}),
    onStreamEnd: quiet,
    loops: { on: quiet },
    follow: (id: string) => (asked.push(`follow ${id}`), Promise.resolve({ ok: true, value: null })),
    continueConversation: (c: { id: string }, model?: string) => answer('continue', `continue ${c.id}${model ? ` on ${model}` : ''}`),
    submitTurn: (id: string, text: string) => answer('submit', `submit ${text} to ${id}`),
    orchestrate: {
      openProject: (ws: string) => answer('open', `open project ${ws}`),
      orchestrator: (id: string) => answer('orchestrator', `orchestrator of ${id}`),
      children: (id: string) => answer('children', `children of ${id}`),
      questions: (id: string) => answer('questions', `questions of ${id}`),
      answer: (id: string, qid: string, a: string, note?: string) => answer('answer', `answer ${qid} of ${id}: ${a}${note ? ` (${note})` : ''}`),
    },
  } as unknown as DcodeApi;
  return { api, asked, answers, push: (b: SessionEvents) => events.forEach((l) => l(b)) };
}

function started() {
  const f = orchestrateApi();
  const store = new LiveStore(f.api);
  store.start();
  return { f, store };
}

/** The orchestrator's log so far: its creation, two children started, and a question. */
function replay(f: ReturnType<typeof orchestrateApi>) {
  const at = '2026-10-18T10:03:00Z';
  f.push({
    sessionId: 's-orq',
    events: [
      { seq: 1, session_id: 's-orq', type: 'session.created', at, payload: orchestrator },
      { seq: 2, session_id: 's-orq', type: 'child.started', at, payload: { child: child('c-a') } },
      { seq: 3, session_id: 's-orq', type: 'child.started', at, payload: { child: child('c-old', 'closed') } },
      { seq: 4, session_id: 's-orq', type: 'question.asked', at, payload: question('q1') },
    ],
  });
}

const agentsOf = (store: LiveStore) => store.getSnapshot().sessions.byId['s-orq']!.agents;

describe('opening a project’s orchestrator', () => {
  it('opens the folder’s project, asks for its orchestrator and follows it', async () => {
    const { f, store } = started();
    f.answers.open = { ok: true, value: project };
    f.answers.orchestrator = { ok: true, value: orchestrator };
    expect(await store.actions.openOrchestrator('/w/dcode')).toEqual({ ok: true, value: 's-orq' });
    expect(f.asked).toEqual(['open project /w/dcode', 'orchestrator of p1', 'follow s-orq']);
  });

  it('says the key to set when the daemon conducts no projects', async () => {
    const { f, store } = started();
    f.answers.open = { ok: false, refusal: { code: 'orchestrator_unavailable', message: 'orchestrator.enabled is false: the orchestrator is opt-in in this slice' } };
    const got = await store.actions.openOrchestrator('/w/dcode');
    expect(got.ok).toBe(false);
    const why = got.ok ? '' : got.why;
    expect(why).toContain('orchestrator.enabled is false');
    expect(why).toContain('orchestrator.enabled = true');
    expect(why).toContain('DCODE_ORCHESTRATOR_ENABLED');
    expect(f.asked).toEqual(['open project /w/dcode']);
  });

  it('says a project answer it cannot read, and opens nothing', async () => {
    const { f, store } = started();
    f.answers.open = { ok: true, value: { id: 'p1' } };
    const got = await store.actions.openOrchestrator('/w/dcode');
    expect(got.ok ? '' : got.why).toMatch(/^Resposta ilegível do daemon: /);
    expect(f.asked).toEqual(['open project /w/dcode']);
  });
});

// An orchestrator is never continued: it is the live session of its project,
// and a new ordinary session would have neither its role nor its tools. A
// child or a reviewer that ended is not continued either.
describe('opening an orchestrator’s conversation, or one of its agents’', () => {
  const row = (over: Partial<Row>): Row => ({ id: 's-orq', title: 'orq', workspace: '/w/dcode', state: 'recorded', started: 0, lastAt: null, ...over });

  it('opens a recorded orchestrator as the live one of its project, never a continuation', async () => {
    const { f, store } = started();
    f.answers.orchestrator = { ok: true, value: orchestrator };
    expect(await store.actions.open(row({ role: 'orchestrator', projectId: 'p1' }))).toEqual({ ok: true, value: 's-orq' });
    expect(f.asked).toEqual(['orchestrator of p1', 'follow s-orq']);
  });

  it('opens a live orchestrator through its project too', async () => {
    const { f, store } = started();
    f.answers.orchestrator = { ok: true, value: orchestrator };
    expect(await store.actions.open(row({ state: 'idle', role: 'orchestrator', projectId: 'p1' }))).toEqual({ ok: true, value: 's-orq' });
    expect(f.asked).toEqual(['orchestrator of p1', 'follow s-orq']);
  });

  it('says why when the orchestrator is unavailable, with the keys that turn it on, and continues nothing', async () => {
    const { f, store } = started();
    f.answers.orchestrator = { ok: false, refusal: { code: 'orchestrator_unavailable', message: 'record.enabled is false' } };
    const got = await store.actions.open(row({ role: 'orchestrator', projectId: 'p1' }));
    const why = got.ok ? '' : got.why;
    expect(why).toContain('record.enabled is false');
    expect(why).toContain('orchestrator.enabled = true');
    expect(f.asked).toEqual(['orchestrator of p1']);
  });

  it('says an orchestrator without its project, rather than continuing it', async () => {
    const { f, store } = started();
    const got = await store.actions.open(row({ role: 'orchestrator' }));
    expect(got.ok ? '' : got.why).toContain('projeto');
    expect(f.asked).toEqual([]);
  });

  it('never continues a child or a reviewer that ended, and points to the orchestrator', async () => {
    for (const role of ['child', 'reviewer']) {
      const { f, store } = started();
      const got = await store.actions.open(row({ id: 'c-a', role, parentId: 's-orq', projectId: 'p1' }));
      expect(got.ok ? '' : got.why).toContain('orquestradora');
      expect(f.asked).toEqual([]);
    }
  });

  it('never switches an orchestrator’s model by continuing it in a new session', async () => {
    const { f, store } = started();
    replay(f);
    const got = await store.actions.switchModel('s-orq', '/w/dcode', 'Sonnet');
    expect(got.ok ? '' : got.why).toContain('orquestradora');
    expect(f.asked).toEqual([]);
  });

  it('says a session the daemon no longer has in words, not as the daemon’s raw error', async () => {
    const { f, store } = started();
    f.answers.submit = { ok: false, refusal: { code: 'session_not_found', message: 'no session c-a' } };
    const got = await store.actions.send('c-a', 'oi', false);
    const why = got.ok ? '' : got.why;
    expect(why).not.toContain('no session');
    expect(why).toContain('não está mais no daemon');
  });
});

describe('the orchestrator’s list read from the daemon', () => {
  it('takes /children and /questions over the events: a closed child it no longer carries leaves', async () => {
    const { f, store } = started();
    replay(f);
    expect(agentsOf(store).children.map((c) => c.id)).toEqual(['c-a', 'c-old']);
    f.answers.children = { ok: true, value: { children: [child('c-a', 'pr')] } };
    f.answers.questions = { ok: true, value: { questions: [question('q1'), { ...question('q0'), answer: 'merge', answered: '2026-10-18T10:02:30Z' }] } };
    expect(await store.actions.readAgents('s-orq')).toEqual({ ok: true, value: null });
    expect(agentsOf(store).children.map((c) => [c.id, c.state])).toEqual([['c-a', 'pr']]);
    expect(agentsOf(store).questions.map((q) => q.id)).toEqual(['q1']);
    expect(f.asked).toEqual(['children of s-orq', 'questions of s-orq']);
  });

  it('says why when the daemon did not answer, and keeps what the events said', async () => {
    const { f, store } = started();
    replay(f);
    f.answers.children = { ok: false, refusal: { code: 'unreachable', message: 'O daemon caiu.' } };
    const got = await store.actions.readAgents('s-orq');
    expect(got.ok ? '' : got.why).toContain('O daemon caiu.');
    expect(agentsOf(store).children.map((c) => c.id)).toEqual(['c-a', 'c-old']);
  });
});

describe('answering a question', () => {
  it('sends the option and the note, and takes the answered question out of the queue', async () => {
    const { f, store } = started();
    replay(f);
    f.answers.answer = { ok: true, value: { ...question('q1'), answer: 'return', note: 'falta o teste', answered: '2026-10-18T10:04:00Z' } };
    const got = await store.actions.answerQuestion('s-orq', 'q1', 'return', 'falta o teste');
    expect(got.ok).toBe(true);
    expect(f.asked).toEqual(['answer q1 of s-orq: return (falta o teste)']);
    expect(agentsOf(store).questions).toEqual([]);
  });

  it('says another answer came first, and reads the queue again', async () => {
    const { f, store } = started();
    replay(f);
    f.answers.answer = { ok: false, refusal: { code: 'question_already_answered', message: 'question q1 was already answered "merge"' } };
    f.answers.questions = { ok: true, value: { questions: [] } };
    f.answers.children = { ok: true, value: { children: [child('c-a')] } };
    const got = await store.actions.answerQuestion('s-orq', 'q1', 'close', '');
    expect(got.ok ? '' : got.why).toContain('já foi respondida');
    expect(got.ok ? '' : got.why).toContain('question q1 was already answered "merge"');
    expect(f.asked).toEqual(['answer q1 of s-orq: close', 'children of s-orq', 'questions of s-orq']);
    expect(agentsOf(store).questions).toEqual([]);
  });

  it('leaves the queue to question.answered when another window answers', () => {
    const { f, store } = started();
    replay(f);
    f.push({ sessionId: 's-orq', events: [{ seq: 5, session_id: 's-orq', type: 'question.answered', at: '2026-10-18T10:05:00Z', payload: { ...question('q1'), answer: 'merge', answered: '2026-10-18T10:05:00Z' } }] });
    expect(agentsOf(store).questions).toEqual([]);
  });
});
