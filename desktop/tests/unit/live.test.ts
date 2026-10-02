import { describe, expect, it } from 'vitest';
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
  const api: DcodeApi = {
    platform: 'test',
    user: () => Promise.resolve({ name: 'Ana' }),
    onDaemon: (l) => (listeners.daemon.push(l), () => {}),
    onConversations: (l) => (listeners.list.push(l), () => {}),
    onSessionEvents: (l) => (listeners.events.push(l), () => {}),
    onStreamEnd: (l) => (listeners.end.push(l), () => {}),
    follow: (id) => answer(`follow ${id}`),
    unfollow: () => Promise.resolve(),
    continueConversation: (c) => {
      asked.push(`continue ${c.id} in ${c.workspace}`);
      return new Promise((r) => (resolveContinue = r));
    },
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

    // Once the list shows the continuation live, opening the ended one opens it.
    f.listeners.list[0]?.({ kind: 'changed', conversation: { ...ended, id: 'new', state: 'idle', live: true, continued_from: 'old' } });
    expect(await store.actions.open(row)).toEqual({ ok: true, value: 'new' });
    expect(f.asked.slice(2)).toEqual(['follow new']);
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
    f.listeners.end[0]?.({ sessionId: 's1', reason: 'a sessão não existe mais' });
    expect(said).toEqual([
      'Evento ilegível, sem sessão para mostrá-lo: o evento não é um objeto',
      'Os eventos de “s1” pararam de chegar: a sessão não existe mais',
    ]);
  });
});
