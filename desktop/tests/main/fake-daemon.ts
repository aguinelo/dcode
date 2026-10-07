// A daemon for the main process's tests: the routes the desktop uses, over a
// real Unix socket, answering the way `dcode serve` answers — JSON bodies,
// `{code, message}` refusals, and SSE streams that stay open until the test
// drops them or kills the daemon.

import { mkdtempSync, rmSync } from 'node:fs';
import http from 'node:http';
import os from 'node:os';
import path from 'node:path';

/** A short temp folder with a socket path in it: macOS caps socket paths near 104 bytes. */
export function tempSocket(): { dir: string; socket: string; remove(): void } {
  const dir = mkdtempSync(path.join(os.tmpdir(), 'dcm-'));
  return { dir, socket: path.join(dir, 's.sock'), remove: () => rmSync(dir, { recursive: true, force: true }) };
}

export interface FakeOptions {
  /** What /version answers. */
  version?: unknown;
  /** The conversations the list's stream opens with. */
  conversations?: unknown[];
  /** Each session's events, by id; seq is set from the position. */
  sessions?: Record<string, number>;
  /** Refuses the list's stream with this status and body, as an older daemon does. */
  refuseList?: { status: number; contentType: string; body: string };
  /** What GET /v1/models answers; without it, the route is not there, as on a daemon before it. */
  models?: unknown;
}

export interface FakeDaemon {
  socket: string;
  /** Every request, in order: "METHOD url". */
  requests: string[];
  /** Every JSON body received, in order. */
  bodies: unknown[];
  /** Sends one frame on every open list stream: a value as JSON, a string as it is. */
  sendList(frame: unknown): void;
  /** Appends events to a session and sends them on its open streams. */
  emit(sessionId: string, count: number): void;
  /** Ends every open stream, and stays up: as a daemon dropping a client that fell behind. */
  dropStreams(): void;
  /** Dies: every connection cut, nothing listening. */
  kill(): Promise<void>;
}

function event(sessionId: string, seq: number): unknown {
  return { seq, session_id: sessionId, type: seq === 1 ? 'session.created' : 'turn.started', at: '2026-10-01T10:00:00Z', payload: {} };
}

export async function fakeDaemon(socket: string, o: FakeOptions = {}): Promise<FakeDaemon> {
  const requests: string[] = [];
  const bodies: unknown[] = [];
  const seqs = new Map(Object.entries(o.sessions ?? {}));
  const lists = new Set<http.ServerResponse>();
  const streams = new Map<string, Set<http.ServerResponse>>();
  let opened = 0;

  const json = (res: http.ServerResponse, status: number, body: unknown) => {
    res.writeHead(status, { 'content-type': 'application/json' });
    res.end(JSON.stringify(body));
  };
  const sse = (res: http.ServerResponse) => {
    res.writeHead(200, { 'content-type': 'text/event-stream' });
  };

  function route(method: string, url: string, body: unknown, res: http.ServerResponse): void {
    const u = new URL(url, 'http://daemon');
    const session = /^\/v1\/sessions\/([^/]+)(\/.*)?$/.exec(u.pathname);
    const id = session ? decodeURIComponent(session[1] ?? '') : '';
    const rest = session?.[2] ?? '';
    if (method === 'GET' && u.pathname === '/health') return json(res, 200, { status: 'ok' });
    if (method === 'GET' && u.pathname === '/version') return json(res, 200, o.version ?? { version: '0.0.0-fake', protocol: 'v1' });
    if (method === 'GET' && u.pathname === '/v1/conversations/events') {
      if (o.refuseList) {
        res.writeHead(o.refuseList.status, { 'content-type': o.refuseList.contentType });
        return void res.end(o.refuseList.body);
      }
      sse(res);
      res.write(`data: ${JSON.stringify({ kind: 'snapshot', conversations: o.conversations ?? [] })}\n\n`);
      lists.add(res);
      res.on('close', () => lists.delete(res));
      return;
    }
    if (method === 'POST' && u.pathname === '/v1/sessions') {
      const sid = `s-new-${++opened}`;
      seqs.set(sid, 1);
      const ws = (body as { workspace?: string } | undefined)?.workspace ?? '';
      return json(res, 201, { id: sid, state: 'idle', workspace: ws, model: 'fake', sandbox_mode: 'workspace-write', mode: 'assist', created_at: '2026-10-01T10:00:00Z', last_seq: 1, done_criteria: 0 });
    }
    if (method === 'GET' && u.pathname === '/v1/models' && o.models !== undefined) return json(res, 200, o.models);
    if (session && !seqs.has(id)) return json(res, 404, { code: 'session_not_found', message: `no session ${id}` });
    if (method === 'DELETE' && rest === '') {
      seqs.delete(id);
      for (const s of streams.get(id) ?? []) s.end();
      return void res.writeHead(204).end();
    }
    if (method === 'GET' && rest === '/events') {
      const from = Number(u.searchParams.get('from') ?? '1');
      sse(res);
      for (let seq = from; seq <= (seqs.get(id) ?? 0); seq++) res.write(`id: ${seq}\ndata: ${JSON.stringify(event(id, seq))}\n\n`);
      const set = streams.get(id) ?? new Set();
      set.add(res);
      streams.set(id, set);
      res.on('close', () => set.delete(res));
      return;
    }
    if (method === 'POST' && ['/turns', '/steer'].includes(rest)) return void res.writeHead(202).end();
    if (method === 'POST' && (rest === '/interrupt' || rest.startsWith('/approvals/'))) return void res.writeHead(204).end();
    res.writeHead(404, { 'content-type': 'text/plain; charset=utf-8' });
    res.end('404 page not found\n');
  }

  const server = http.createServer((req, res) => {
    const chunks: Buffer[] = [];
    req.on('data', (c: Buffer) => chunks.push(c));
    req.on('end', () => {
      const raw = Buffer.concat(chunks).toString('utf8');
      const body: unknown = raw ? JSON.parse(raw) : undefined;
      requests.push(`${req.method} ${req.url}`);
      if (body !== undefined) bodies.push(body);
      route(req.method ?? '', req.url ?? '', body, res);
    });
  });
  await new Promise<void>((resolve, reject) => {
    server.once('error', reject);
    server.listen(socket, () => resolve());
  });

  return {
    socket,
    requests,
    bodies,
    sendList(frame) {
      const data = typeof frame === 'string' ? frame : JSON.stringify(frame);
      for (const res of lists) res.write(`data: ${data}\n\n`);
    },
    emit(sessionId, count) {
      const last = seqs.get(sessionId) ?? 0;
      seqs.set(sessionId, last + count);
      for (const res of streams.get(sessionId) ?? []) {
        for (let seq = last + 1; seq <= last + count; seq++) res.write(`id: ${seq}\ndata: ${JSON.stringify(event(sessionId, seq))}\n\n`);
      }
    },
    dropStreams() {
      for (const res of [...lists, ...[...streams.values()].flatMap((s) => [...s])]) res.end();
    },
    async kill() {
      server.closeAllConnections();
      await new Promise<void>((resolve) => server.close(() => resolve()));
      rmSync(socket, { force: true });
    },
  };
}

/** Polls until `probe` holds, failing with `what` and the last value seen. */
export async function until<T>(probe: () => T, what: string, timeoutMs = 3_000): Promise<NonNullable<T>> {
  const deadline = Date.now() + timeoutMs;
  for (;;) {
    const v = probe();
    if (v) return v as NonNullable<T>;
    if (Date.now() > deadline) throw new Error(`${what} (waited ${timeoutMs} ms)`);
    await new Promise((r) => setTimeout(r, 10));
  }
}
