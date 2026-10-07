import { afterEach, describe, expect, it } from 'vitest';
import { Connection, type Pushes } from '../../src/main/connection';
import type { Lookup } from '../../src/main/locate';
import type { DaemonStatus, SessionEvents, StreamEnd } from '../../src/shared/api';
import { fakeDaemon, tempSocket, until, type FakeDaemon, type FakeOptions } from './fake-daemon';

// The connection against a daemon that answers over a real socket: attaching,
// the list, a daemon that dies and one that comes back, and the sessions the
// window follows. Starting a daemon is in connection-start.test.ts.

const TIMING = { healthEveryMs: 40, lostEveryMs: 40, startWithinMs: 1_000, startPollMs: 10, retryFromMs: 10, retryUpToMs: 50 };

const cleanups: (() => void | Promise<void>)[] = [];
afterEach(async () => {
  for (const c of cleanups.splice(0).reverse()) await c();
});

function recorder() {
  const statuses: DaemonStatus[] = [];
  const frames: unknown[] = [];
  const batches: SessionEvents[] = [];
  const ends: StreamEnd[] = [];
  const pushes: Pushes = {
    status: (s) => statuses.push(s),
    conversations: (f) => frames.push(f),
    sessionEvents: (b) => batches.push(b),
    streamEnd: (e) => ends.push(e),
  };
  const seqs = (id: string) => batches.filter((b) => b.sessionId === id).flatMap((b) => b.events.map((e) => (e as { seq: number }).seq));
  return { statuses, frames, batches, ends, pushes, seqs, last: () => statuses[statuses.length - 1] };
}

/** A lookup with DCODE_SOCKET set and no binary anywhere: attaching must not need one. */
function attachOnly(socket: string): Lookup {
  return {
    env: { DCODE_SOCKET: socket },
    home: '/nowhere',
    cwd: '/',
    probe: () => {
      throw new Error('looked for a binary to attach');
    },
    run: () => Promise.reject(new Error('ran a command to attach')),
  };
}

async function setUp(options: FakeOptions = {}) {
  const t = tempSocket();
  cleanups.push(t.remove);
  const daemons: FakeDaemon[] = [];
  const up = async (o: FakeOptions = options) => {
    const d = await fakeDaemon(t.socket, o);
    daemons.push(d);
    cleanups.push(() => d.kill());
    return d;
  };
  const daemon = await up();
  const r = recorder();
  const conn = new Connection(
    {
      lookup: attachOnly(t.socket),
      serve: () => {
        throw new Error('started a daemon while one answers');
      },
      timing: TIMING,
    },
    r.pushes,
  );
  cleanups.push(() => conn.close());
  return { socket: t.socket, daemon, up, conn, r };
}

const connected = (r: ReturnType<typeof recorder>) => r.last()?.state === 'connected';

describe('attaching', () => {
  it('uses a daemon that answers, with its version, and never looks for a binary', async () => {
    const { socket, conn, r } = await setUp({ version: { version: '0.23.0', protocol: 'v1' } });
    expect(conn.status()).toEqual({ state: 'connecting' });
    conn.start();
    await until(() => connected(r), 'connected');
    expect(r.statuses).toEqual([{ state: 'connected', version: '0.23.0', socket, started: false }]);
    expect(conn.startedByApp()).toBe(false);
  });

  it('fails, with the reason, when the daemon speaks another protocol', async () => {
    const { conn, r } = await setUp({ version: { version: '9.0.0', protocol: 'v2' } });
    conn.start();
    await until(() => r.last()?.state === 'failed', 'failed');
    const last = r.last();
    expect(last?.state === 'failed' && last.reason).toContain('fala o protocolo v2, e este app fala o v1');
  });

  it('fails saying what to do when the daemon has no list of conversations, as dcode before 0.23.0', async () => {
    const { conn, r } = await setUp({ refuseList: { status: 404, contentType: 'text/plain', body: '404 page not found\n' } });
    conn.start();
    await until(() => r.last()?.state === 'failed', 'failed');
    const last = r.last();
    expect(last?.state === 'failed' && last.reason).toContain('recusou a lista de conversas');
    expect(last?.state === 'failed' && last.reason).toContain('dcode 0.23.0 ou mais novo (dcode update)');
  });
});

describe('the list of conversations', () => {
  const a = { id: 'a', title: 'a', workspace: '/w', state: 'idle', live: true, turns: 0, started: '', last_activity: '' };
  const b = { ...a, id: 'b', title: 'b' };

  it('passes every frame on as it came, and keeps the list folded for a window that opens later', async () => {
    const { daemon, conn, r } = await setUp({ conversations: [a] });
    expect(conn.conversationsNow()).toBeNull();
    conn.start();
    await until(() => r.frames.length === 1, 'the snapshot');
    daemon.sendList({ kind: 'changed', conversation: b });
    daemon.sendList({ kind: 'changed', conversation: { ...a, state: 'running' } });
    daemon.sendList('{"kind":"chan');
    daemon.sendList({ kind: 'removed', id: 'b' });
    await until(() => r.frames.length === 5, 'four more frames');
    expect(r.frames[3]).toBe('{"kind":"chan');
    expect(conn.conversationsNow()).toEqual({ kind: 'snapshot', conversations: [{ ...a, state: 'running' }] });
    expect(conn.working()).toBe(1);
  });

  it('opens again on a fresh snapshot when the daemon drops it and still answers', async () => {
    const { daemon, conn, r } = await setUp({ conversations: [a] });
    conn.start();
    await until(() => r.frames.length === 1, 'the snapshot');
    daemon.dropStreams();
    await until(() => r.frames.length === 2, 'a second snapshot');
    expect(r.frames[1]).toEqual({ kind: 'snapshot', conversations: [a] });
    expect(r.statuses.map((s) => s.state)).toEqual(['connected']);
  });
});

describe('a daemon that dies', () => {
  it('is lost, saying which socket stopped answering — and nothing is sent to it', async () => {
    const { socket, daemon, conn, r } = await setUp();
    conn.start();
    await until(() => connected(r), 'connected');
    await daemon.kill();
    await until(() => r.last()?.state === 'lost', 'lost');
    const last = r.last();
    expect(last?.state === 'lost' && last.reason).toContain(`O daemon em ${socket} parou de responder`);
    const sent = await conn.requests.submitTurn('s-1', 'ainda aí?');
    expect(sent.ok).toBe(false);
    expect(!sent.ok && sent.refusal.code).toBe('unreachable');
    expect(conn.follow('s-1')).toMatchObject({ ok: false, refusal: { code: 'unreachable' } });
  });

  it('is connected again when one answers, with a fresh snapshot and each followed session from where it was', async () => {
    const { daemon, up, conn, r } = await setUp({ sessions: { 's-1': 3 } });
    conn.start();
    await until(() => connected(r), 'connected');
    expect(conn.follow('s-1')).toEqual({ ok: true, value: null });
    await until(() => r.seqs('s-1').length === 3, 'events 1 to 3');
    await daemon.kill();
    await until(() => r.last()?.state === 'lost', 'lost');

    const back = await up({ sessions: { 's-1': 5 } });
    await until(() => connected(r), 'connected again');
    await until(() => r.seqs('s-1').length === 5, 'events 4 and 5');
    expect(r.seqs('s-1')).toEqual([1, 2, 3, 4, 5]);
    expect(back.requests).toContain('GET /v1/sessions/s-1/events?from=4');
    expect(r.frames.filter((f) => (f as { kind?: string }).kind === 'snapshot')).toHaveLength(2);
    expect(r.statuses.map((s) => s.state)).toEqual(['connected', 'lost', 'connected']);
  });
});

describe('following a session', () => {
  it('answers unreachable before there is a daemon', async () => {
    const { conn } = await setUp();
    expect(conn.follow('s-1')).toMatchObject({ ok: false, refusal: { code: 'unreachable' } });
  });

  it('pushes its events as they come, one batch per chunk', async () => {
    const { daemon, conn, r } = await setUp({ sessions: { 's-1': 2 } });
    conn.start();
    await until(() => connected(r), 'connected');
    conn.follow('s-1');
    await until(() => r.seqs('s-1').length === 2, 'the first two events');
    daemon.emit('s-1', 1);
    await until(() => r.seqs('s-1').length === 3, 'the third');
    expect(r.batches.at(-1)).toEqual({ sessionId: 's-1', events: [expect.objectContaining({ seq: 3, session_id: 's-1' })] });
    expect(conn.followed()).toEqual({ 's-1': 3 });
  });

  it('resumes from the event after the last when the stream drops and the daemon still answers', async () => {
    const { daemon, conn, r } = await setUp({ sessions: { 's-1': 2 } });
    conn.start();
    await until(() => connected(r), 'connected');
    conn.follow('s-1');
    await until(() => r.seqs('s-1').length === 2, 'the first two events');
    daemon.dropStreams();
    await until(() => daemon.requests.includes('GET /v1/sessions/s-1/events?from=3'), 'the stream reopened from 3');
    daemon.emit('s-1', 1);
    await until(() => r.seqs('s-1').length === 3, 'the third event');
    expect(r.seqs('s-1')).toEqual([1, 2, 3]);
  });

  it('ends one the daemon does not have, with the daemon’s message, and stops following it', async () => {
    const { conn, r } = await setUp();
    conn.start();
    await until(() => connected(r), 'connected');
    expect(conn.follow('s-9')).toEqual({ ok: true, value: null });
    await until(() => r.ends.length === 1, 'the end of the stream');
    expect(r.ends[0]?.sessionId).toBe('s-9');
    expect(r.ends[0]?.reason).toContain('no session s-9');
    expect(conn.followed()).toEqual({});
  });

  it('starts over from the first event when followed again, and stops when unfollowed', async () => {
    const { daemon, conn, r } = await setUp({ sessions: { 's-1': 2 } });
    conn.start();
    await until(() => connected(r), 'connected');
    conn.follow('s-1');
    await until(() => r.seqs('s-1').length === 2, 'the events');
    conn.follow('s-1');
    await until(() => r.seqs('s-1').length === 4, 'the events again');
    conn.unfollow('s-1');
    daemon.emit('s-1', 1);
    await new Promise((resolve) => setTimeout(resolve, 60));
    expect(r.seqs('s-1')).toEqual([1, 2, 1, 2]);
    expect(conn.followed()).toEqual({});
  });
});

describe('requests', () => {
  it('opens a session in a folder, and continues a recorded conversation in a new one', async () => {
    const { daemon, conn, r } = await setUp();
    conn.start();
    await until(() => connected(r), 'connected');
    const opened = await conn.requests.createSession('/w');
    expect(opened.ok && opened.value.id).toBe('s-new-1');
    const continued = await conn.requests.continueConversation('c-old', '/w');
    expect(continued.ok && continued.value.workspace).toBe('/w');
    await conn.requests.continueConversation('c-old', '/w', 'qwen-local');
    expect(daemon.bodies).toEqual([
      { workspace: '/w' },
      { workspace: '/w', resume: 'c-old' },
      { workspace: '/w', resume: 'c-old', model: 'qwen-local' },
    ]);
  });

  it('closes a session, and asks which models a workspace can use', async () => {
    const menu = { default: { name: 'MiniMax-M3', model: 'MiniMax-M3', family: 'minimax-m3', transport: 'openai', measured: true }, profiles: [] };
    const { daemon, conn, r } = await setUp({ sessions: { 's 1': 1 }, models: menu });
    conn.start();
    await until(() => connected(r), 'connected');
    expect(await conn.requests.closeSession('s 1')).toEqual({ ok: true, value: null });
    expect(await conn.requests.closeSession('s 1')).toEqual({ ok: false, refusal: { code: 'session_not_found', message: 'no session s 1' } });
    expect(await conn.requests.listModels('/w a')).toEqual({ ok: true, value: menu });
    expect(daemon.requests.filter((q) => !q.includes('/health') && !q.includes('/version') && !q.includes('/conversations'))).toEqual([
      'DELETE /v1/sessions/s%201',
      'DELETE /v1/sessions/s%201',
      'GET /v1/models?workspace=%2Fw%20a',
    ]);
  });

  it('says so when the daemon is from before the list of models', async () => {
    const { conn, r } = await setUp();
    conn.start();
    await until(() => connected(r), 'connected');
    expect(await conn.requests.listModels('/w')).toEqual({
      ok: false,
      refusal: { code: 'unexpected_answer', message: 'O daemon respondeu 404 a GET /v1/models?workspace=%2Fw: 404 page not found.' },
    });
  });

  it('sends each command to its route, and answers null', async () => {
    const { daemon, conn, r } = await setUp({ sessions: { 's 1': 1 } });
    conn.start();
    await until(() => connected(r), 'connected');
    expect(await conn.requests.submitTurn('s 1', 'oi')).toEqual({ ok: true, value: null });
    expect(await conn.requests.steer('s 1', 'na verdade')).toEqual({ ok: true, value: null });
    expect(await conn.requests.interrupt('s 1')).toEqual({ ok: true, value: null });
    expect(await conn.requests.resolveApproval('s 1', 'a/1', 'deny')).toEqual({ ok: true, value: null });
    expect(daemon.requests.filter((q) => q.startsWith('POST'))).toEqual([
      'POST /v1/sessions/s%201/turns',
      'POST /v1/sessions/s%201/steer',
      'POST /v1/sessions/s%201/interrupt',
      'POST /v1/sessions/s%201/approvals/a%2F1',
    ]);
    expect(daemon.bodies).toEqual([{ text: 'oi' }, { text: 'na verdade' }, { decision: 'deny' }]);
  });

  it('answers the daemon’s refusal as it came', async () => {
    const { conn, r } = await setUp();
    conn.start();
    await until(() => connected(r), 'connected');
    expect(await conn.requests.submitTurn('s-9', 'oi')).toEqual({ ok: false, refusal: { code: 'session_not_found', message: 'no session s-9' } });
  });

  it('answers unreachable while still connecting, and sends nothing', async () => {
    const { daemon, conn } = await setUp();
    const got = await conn.requests.createSession('/w');
    expect(got).toMatchObject({ ok: false, refusal: { code: 'unreachable' } });
    expect(daemon.requests).toEqual([]);
  });
});
