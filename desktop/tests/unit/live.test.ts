import { afterEach, describe, expect, it, vi } from 'vitest';
import type * as P from '../../src/protocol/generated';
import { LiveStore } from '../../src/renderer/live';
import type { Answer, DaemonStatus, DcodeApi, SessionEvents, StreamEnd } from '../../src/shared/api';

/** A main process that answers what the test says, and records what was asked. */
function fakeApi() {
  const asked: string[] = [];
  const listeners = {
    daemon: [] as ((s: DaemonStatus) => void)[],
    list: [] as ((c: P.ConversationChange) => void)[],
    events: [] as ((b: SessionEvents) => void)[],
    end: [] as ((e: StreamEnd) => void)[],
  };
  const ok = <T,>(value: T): Answer<T> => ({ ok: true, value });
  const session = (id: string) => ({ id, workspace: '/w' }) as P.Session;
  // What the daemon refuses, by the first word of what was asked.
  const refusals: Record<string, { code: string; message: string }> = {};
  const answer = (what: string): Promise<Answer<null>> => {
    asked.push(what);
    const refusal = refusals[what.split(' ')[0] ?? ''];
    return Promise.resolve(refusal ? { ok: false, refusal } : ok(null));
  };
  let resolveContinue: (a: Answer<P.Session>) => void = () => {};
  let models: Answer<P.ModelsResponse> = { ok: false, refusal: { code: 'unexpected_answer', message: 'O daemon respondeu 404 a GET /v1/models: 404 page not found.' } };
  const api: DcodeApi = {
    platform: 'test',
    user: () => Promise.resolve({ name: 'Ana' }),
    onDaemon: (l) => (listeners.daemon.push(l), () => {}),
    onConversations: (l) => (listeners.list.push(l), () => {}),
    onSessionEvents: (l) => (listeners.events.push(l), () => {}),
    onStreamEnd: (l) => (listeners.end.push(l), () => {}),
    follow: (id) => answer(`follow ${id}`),
    unfollow: () => Promise.resolve(),
    continueConversation: (c, model) => {
      asked.push(`continue ${c.id} in ${c.workspace}${model ? ` on ${model}` : ''}`);
      return new Promise((r) => (resolveContinue = r));
    },
    closeSession: (id) => answer(`close ${id}`),
    listModels: (ws) => (asked.push(`models of ${ws}`), Promise.resolve(models)),
    pickFolder: () => Promise.resolve('/w/novo'),
    createSession: (ws) => (asked.push(`create in ${ws}`), Promise.resolve(ok(session('new')))),
    submitTurn: (id, text) => answer(`turn ${id}: ${text}`),
    steer: (id, text) => answer(`steer ${id}: ${text}`),
    interrupt: (id) => answer(`interrupt ${id}`),
    resolveApproval: (id, a, d) => answer(`approve ${id} ${a} ${d}`),
  };
  return {
    api,
    asked,
    listeners,
    refuse: (what: string, code: string, message: string) => (refusals[what] = { code, message }),
    continued: (id: string) => resolveContinue(ok(session(id))),
    listing: (a: Answer<P.ModelsResponse>) => (models = a),
  };
}

const ended = { id: 'old', title: 'conserte o parser', workspace: '/w/dcode', state: 'recorded', live: false, turns: 1, started: '2026-09-30T10:00:00Z', last_activity: '2026-09-30T10:05:00Z' };

describe('the window connected', () => {
  it('draws the daemon’s status and its list', () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    store.start();
    expect(store.getSnapshot().daemon).toEqual({ state: 'connecting' });
    f.listeners.daemon[0]?.({ state: 'connected', version: '0.0.0-check', socket: '/s', started: false });
    f.listeners.list[0]?.({ kind: 'snapshot', conversations: [ended] });
    const snap = store.getSnapshot();
    expect(snap.daemon.state).toBe('connected');
    expect(snap.rows.map((r) => [r.id, r.state, r.title])).toEqual([['old', 'recorded', 'conserte o parser']]);
  });

  it('continues an ended conversation once, however many times it is opened', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    store.start();
    f.listeners.list[0]?.({ kind: 'snapshot', conversations: [ended] });
    const row = store.getSnapshot().rows[0]!;
    const first = store.actions.open(row);
    const second = store.actions.open(row);
    f.continued('new');
    expect(await first).toEqual({ ok: true, value: 'new' });
    expect(await second).toEqual({ ok: true, value: 'new' });
    expect(f.asked).toEqual(['continue old in /w/dcode', 'follow new']);

    // Once the list shows the continuation live, opening the ended one opens
    // it — already followed, so its stream is not asked for a second time.
    f.listeners.list[0]?.({ kind: 'changed', conversation: { ...ended, id: 'new', state: 'idle', live: true, continued_from: 'old' } });
    expect(await store.actions.open(row)).toEqual({ ok: true, value: 'new' });
    expect(f.asked.slice(2)).toEqual([]);
  });

  it('follows a panel’s conversation once, and an ended one not at all', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    store.start();
    f.listeners.list[0]?.({ kind: 'snapshot', conversations: [ended, { ...ended, id: 'live', state: 'running', live: true }] });
    const byId = new Map(store.getSnapshot().rows.map((r) => [r.id, r]));
    store.actions.watch(byId.get('old')!);
    store.actions.watch(byId.get('live')!);
    store.actions.watch(byId.get('live')!);
    await Promise.resolve();
    expect(f.asked).toEqual(['follow live']);
    // A stream that ended for good is asked for again the next time.
    f.listeners.end[0]?.({ sessionId: 'live', reason: 'a sessão não existe mais' });
    store.actions.watch(byId.get('live')!);
    expect(f.asked).toEqual(['follow live', 'follow live']);
  });

  it('continues a live conversation on the model chosen, closes the one it left, and follows the new one', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    store.start();
    f.listeners.list[0]?.({ kind: 'snapshot', conversations: [{ ...ended, id: 'live', state: 'idle', live: true }] });
    const switched = store.actions.switchModel('live', '/w/dcode', 'qwen-local');
    f.continued('on-qwen');
    expect(await switched).toEqual({ ok: true, value: 'on-qwen' });
    expect(f.asked).toEqual(['continue live in /w/dcode on qwen-local', 'close live', 'follow on-qwen']);
  });

  it('closes nothing when the conversation had already ended, and says so when the old one stays open', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    const said: string[] = [];
    store.onNotice((t) => said.push(t));
    store.start();
    f.listeners.list[0]?.({ kind: 'snapshot', conversations: [ended, { ...ended, id: 'live', state: 'idle', live: true }] });
    const fromEnded = store.actions.switchModel('old', '/w/dcode', 'gemini');
    f.continued('on-gemini');
    expect(await fromEnded).toEqual({ ok: true, value: 'on-gemini' });
    expect(f.asked).toEqual(['continue old in /w/dcode on gemini', 'follow on-gemini']);

    // The continuation opened: that it is shown matters more than the old one closing.
    f.refuse('close', 'session_not_found', 'no session live');
    const fromLive = store.actions.switchModel('live', '/w/dcode', 'gemini');
    f.continued('on-gemini-2');
    expect(await fromLive).toEqual({ ok: true, value: 'on-gemini-2' });
    expect(said).toEqual(['A conversa continuou em gemini, e a sessão anterior não fechou: no session live']);
  });

  it('reads the daemon’s list of models, and says why when there is none or it cannot be read', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    expect(await store.actions.listModels('/w/dcode')).toEqual({
      ok: false,
      why: 'O daemon não disse quais modelos existem: O daemon respondeu 404 a GET /v1/models: 404 page not found.',
    });
    const menu = {
      default: { name: 'MiniMax-M3', model: 'MiniMax-M3', family: 'minimax-m3', transport: 'openai', measured: true },
      profiles: [{ name: 'local', model: 'qwen3.5-9b', family: 'generic', transport: 'openai', measured: false, notice: 'nobody measured this endpoint' }],
    };
    f.listing({ ok: true, value: menu });
    expect(await store.actions.listModels('/w/dcode')).toEqual({ ok: true, value: menu });
    f.listing({ ok: true, value: { default: { name: 'x' } } as unknown as P.ModelsResponse });
    const unreadable = await store.actions.listModels('/w/dcode');
    expect(unreadable.ok).toBe(false);
    expect(!unreadable.ok && unreadable.why).toContain('Resposta ilegível do daemon: lista de modelos: default.model');
    expect(f.asked).toEqual(['models of /w/dcode', 'models of /w/dcode', 'models of /w/dcode']);
  });

  it('steers a running turn, and says the daemon’s reason when it refuses', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    expect(await store.actions.send('s1', 'na verdade, resuma', true)).toEqual({ ok: true, value: null });
    f.refuse('turn', 'session_not_found', 'session s1 not found');
    expect(await store.actions.send('s1', 'oi', false)).toEqual({ ok: false, why: 'A mensagem não foi enviada: session s1 not found' });
    expect(f.asked).toEqual(['steer s1: na verdade, resuma', 'turn s1: oi']);
  });

  it('sends the other way when the daemon knows better what the session is doing', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    // The turn started, and its event has not reached the window yet.
    f.refuse('turn', 'turn_already_active', 'a turn is already running');
    expect(await store.actions.send('s1', 'na verdade, resuma', false)).toEqual({ ok: true, value: null });
    // The turn ended, and the window still shows it running.
    f.refuse('steer', 'no_active_turn', 'no turn is running');
    expect(await store.actions.send('s2', 'e agora?', true)).toEqual({
      ok: false,
      why: 'A mensagem não foi enviada: a turn is already running',
    });
    expect(f.asked).toEqual(['turn s1: na verdade, resuma', 'steer s1: na verdade, resuma', 'steer s2: e agora?', 'turn s2: e agora?']);
  });

  it('opens a new session in the folder chosen, and follows it', async () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    expect(await store.actions.newSession()).toEqual({ ok: true, value: 'new' });
    expect(f.asked).toEqual(['create in /w/novo', 'follow new']);
  });

  it('folds the events it follows, and says what it cannot read or what stopped', () => {
    const f = fakeApi();
    const store = new LiveStore(f.api);
    const said: string[] = [];
    store.onNotice((t) => said.push(t));
    store.start();
    f.listeners.events[0]?.({
      sessionId: 's1',
      events: [
        { seq: 1, session_id: 's1', type: 'turn.started', at: '2026-09-30T10:00:00Z', payload: { turn_id: 't1', text: 'oi' } },
        'not an event',
      ],
    });
    expect(store.getSnapshot().sessions.byId.s1?.state).toBe('running');
    vi.useFakeTimers();
    f.listeners.end[0]?.({ sessionId: 's1', reason: 'a sessão não existe mais' });
    vi.runAllTimers();
    expect(said).toEqual([
      'Evento ilegível, sem sessão para mostrá-lo: o evento não é um objeto',
      'Os eventos de “s1” pararam de chegar: a sessão não existe mais',
    ]);
  });

  it('says nothing when the stream that ended belongs to a conversation that ended', () => {
    vi.useFakeTimers();
    const f = fakeApi();
    const store = new LiveStore(f.api);
    const said: string[] = [];
    store.onNotice((t) => said.push(t));
    store.start();
    f.listeners.list[0]?.({ kind: 'snapshot', conversations: [{ ...ended, id: 'live', live: true, state: 'idle' }] });
    f.listeners.end[0]?.({ sessionId: 'live', reason: 'no session live' });
    // The list says it ended a moment after the stream did.
    f.listeners.list[0]?.({ kind: 'changed', conversation: { ...ended, id: 'live' } });
    vi.runAllTimers();
    expect(said).toEqual([]);
  });
});

afterEach(() => {
  vi.useRealTimers();
});
