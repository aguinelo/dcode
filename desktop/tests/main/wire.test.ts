import http from 'node:http';
import { afterEach, describe, expect, it } from 'vitest';
import type { SseFrame } from '../../src/main/sse';
import { UNEXPECTED, openStream, request, type StreamEnded } from '../../src/main/wire';
import { tempSocket, until } from './fake-daemon';

const cleanups: (() => void | Promise<void>)[] = [];
afterEach(async () => {
  for (const c of cleanups.splice(0).reverse()) await c();
});

/** A server on a fresh socket, answering with `handler`. */
async function serve(handler: http.RequestListener): Promise<string> {
  const t = tempSocket();
  const server = http.createServer(handler);
  await new Promise<void>((resolve) => server.listen(t.socket, () => resolve()));
  cleanups.push(t.remove, () => {
    server.closeAllConnections();
    return new Promise<void>((resolve) => server.close(() => resolve()));
  });
  return t.socket;
}

function answer(status: number, contentType: string, body: string): http.RequestListener {
  return (_req, res) => {
    res.writeHead(status, { 'content-type': contentType });
    res.end(body);
  };
}

describe('a request to the daemon', () => {
  it('answers the parsed body of a success', async () => {
    const socket = await serve(answer(200, 'application/json', '{"version":"0.23.0","protocol":"v1"}\n'));
    expect(await request(socket, 'GET', '/version')).toEqual({ ok: true, value: { version: '0.23.0', protocol: 'v1' } });
  });

  it('answers null for a success with no body', async () => {
    const socket = await serve((_req, res) => res.writeHead(202).end());
    expect(await request(socket, 'POST', '/v1/sessions/s-1/turns', { text: 'oi' })).toEqual({ ok: true, value: null });
  });

  it('sends the body as JSON', async () => {
    let got = '';
    const socket = await serve((req, res) => {
      req.setEncoding('utf8');
      req.on('data', (c: string) => (got += c));
      req.on('end', () => res.writeHead(204).end());
    });
    await request(socket, 'POST', '/v1/sessions', { workspace: '/w', resume: 'c-1' });
    expect(JSON.parse(got)).toEqual({ workspace: '/w', resume: 'c-1' });
  });

  it('answers the daemon’s refusal with its code and message as it sent them', async () => {
    const socket = await serve(answer(409, 'application/json', '{"code":"turn_already_active","message":"a turn is already running"}'));
    expect(await request(socket, 'POST', '/v1/sessions/s-1/turns', { text: 'oi' })).toEqual({
      ok: false,
      refusal: { code: 'turn_already_active', message: 'a turn is already running' },
    });
  });

  it('answers what came when a failure has no error body, as an older daemon’s missing route does', async () => {
    const socket = await serve(answer(404, 'text/plain', '404 page not found\n'));
    const got = await request(socket, 'GET', '/v1/conversations');
    expect(got.ok).toBe(false);
    if (got.ok) return;
    expect(got.refusal.code).toBe(UNEXPECTED);
    expect(got.refusal.message).toBe('O daemon respondeu 404 a GET /v1/conversations: 404 page not found.');
  });

  it('answers a success it cannot read as a refusal, not as a value', async () => {
    const socket = await serve(answer(200, 'text/html', '<html>'));
    const got = await request(socket, 'GET', '/version');
    expect(!got.ok && got.refusal.code).toBe(UNEXPECTED);
  });

  it('answers unreachable when nothing listens, naming the socket and the error', async () => {
    const t = tempSocket();
    cleanups.push(t.remove);
    const got = await request(t.socket, 'GET', '/health');
    expect(got.ok).toBe(false);
    if (got.ok) return;
    expect(got.refusal.code).toBe('unreachable');
    expect(got.refusal.message).toContain(t.socket);
    expect(got.refusal.message).toContain('ENOENT');
  });

  it('answers unreachable when the daemon does not answer in time', async () => {
    const socket = await serve(() => {});
    const started = Date.now();
    const got = await request(socket, 'GET', '/health', undefined, 150);
    expect(Date.now() - started).toBeLessThan(2_000);
    expect(!got.ok && got.refusal.code).toBe('unreachable');
    expect(!got.ok && got.refusal.message).toContain('nada em');
  });

  it('answers unreachable when the daemon hangs up in the middle of its answer', async () => {
    const socket = await serve((_req, res) => {
      res.writeHead(200, { 'content-type': 'application/json', 'content-length': '100' });
      res.write('{"version":');
      setTimeout(() => res.socket?.destroy(), 20);
    });
    const got = await request(socket, 'GET', '/version');
    expect(!got.ok && got.refusal.code).toBe('unreachable');
  });
});

describe('an event stream', () => {
  function recorder() {
    const batches: SseFrame[][] = [];
    const ends: StreamEnded[] = [];
    return { batches, ends, on: { frames: (f: SseFrame[]) => batches.push(f), ended: (e: StreamEnded) => ends.push(e) } };
  }

  it('hands over each chunk’s frames as one batch, and says when the daemon ends it', async () => {
    const socket = await serve((_req, res) => {
      res.writeHead(200, { 'content-type': 'text/event-stream' });
      res.write('id: 1\ndata: {"seq":1}\n\nid: 2\ndata: {"seq":2}\n\n');
      setTimeout(() => res.write('id: 3\ndata: {"se'), 20);
      setTimeout(() => res.write('q":3}\n\n: ping\n\n'), 40);
      setTimeout(() => res.end(), 60);
    });
    const r = recorder();
    openStream(socket, '/v1/sessions/s-1/events?from=1', r.on);
    await until(() => r.ends.length > 0, 'the stream to end');
    expect(r.batches.map((b) => b.map((f) => f.id))).toEqual([['1', '2'], ['3']]);
    expect(r.ends).toEqual([{ kind: 'dropped', cause: 'o daemon encerrou o fluxo' }]);
  });

  it('is refused with the daemon’s code, message and status', async () => {
    const socket = await serve(answer(404, 'application/json', '{"code":"session_not_found","message":"no session s-9"}'));
    const r = recorder();
    openStream(socket, '/v1/sessions/s-9/events?from=1', r.on);
    await until(() => r.ends.length > 0, 'the refusal');
    expect(r.ends).toEqual([{ kind: 'refused', status: 404, refusal: { code: 'session_not_found', message: 'no session s-9' } }]);
  });

  it('is unreachable when nothing listens', async () => {
    const t = tempSocket();
    cleanups.push(t.remove);
    const r = recorder();
    openStream(t.socket, '/v1/conversations/events', r.on);
    await until(() => r.ends.length > 0, 'the failure');
    expect(r.ends[0]?.kind).toBe('unreachable');
  });

  it('is dropped when the daemon dies with it open', async () => {
    const socket = await serve((_req, res) => {
      res.writeHead(200, { 'content-type': 'text/event-stream' });
      res.write('data: {"kind":"snapshot"}\n\n');
      setTimeout(() => res.socket?.destroy(), 20);
    });
    const r = recorder();
    openStream(socket, '/v1/conversations/events', r.on);
    await until(() => r.ends.length > 0, 'the drop');
    expect(r.batches).toHaveLength(1);
    expect(r.ends[0]?.kind).toBe('dropped');
  });

  it('says nothing more once closed here', async () => {
    const socket = await serve((_req, res) => {
      res.writeHead(200, { 'content-type': 'text/event-stream' });
      res.write('data: 1\n\n');
      setTimeout(() => res.write('data: 2\n\n'), 50);
    });
    const r = recorder();
    const stream = openStream(socket, '/v1/conversations/events', r.on);
    await until(() => r.batches.length > 0, 'the first frame');
    stream.close();
    await new Promise((resolve) => setTimeout(resolve, 120));
    expect(r.batches).toHaveLength(1);
    expect(r.ends).toEqual([]);
  });
});
