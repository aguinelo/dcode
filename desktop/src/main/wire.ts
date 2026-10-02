import http from 'node:http';
import { isRecord } from '../protocol/validate';
import { UNREACHABLE, type Answer, type Refusal } from '../shared/api';
import { SseParser, type SseFrame } from './sse';

// The daemon's wire as the main process speaks it: HTTP/JSON and SSE over the
// Unix socket (D2). A request always ends in an Answer — the daemon's value,
// the daemon's refusal, or why no daemon took it — and never in a throw, so no
// caller can let a failure fall into silence.
//
// Every request opens a connection of its own (`agent: false`): a pooled one
// would outlive the daemon it was opened to, and be handed to the next.

/** How long a request may take, start to end, before it counts as unanswered. */
export const REQUEST_TIMEOUT_MS = 10_000;

/**
 * Not one of the daemon's codes: an answer this client cannot read. A status
 * without the daemon's error body — an older dcode answers a route it lacks
 * with Go's plain-text 404 — or a success whose body is not JSON.
 */
export const UNEXPECTED = 'unexpected_answer';

/** What came back: an answer, read whole, or why none came. */
export type Exchange = { kind: 'answer'; status: number; body: string } | { kind: 'unreachable'; cause: string };

function seconds(ms: number): string {
  return `${Math.round(ms / 100) / 10} s`;
}

/** At most a line of what the daemon said, for a message a person reads. */
export function excerpt(text: string): string {
  const one = text.replace(/\s+/g, ' ').trim();
  return one.length > 200 ? `${one.slice(0, 200)}…` : one;
}

/** At most a line of any value: text as it is, anything else as its JSON. */
export function shown(v: unknown): string {
  return excerpt(typeof v === 'string' ? v : (JSON.stringify(v) ?? String(v)));
}

/** One request, read to its end. Resolves; never rejects. */
export function exchange(
  socket: string,
  method: string,
  path: string,
  body?: unknown,
  timeoutMs = REQUEST_TIMEOUT_MS,
): Promise<Exchange> {
  return new Promise((resolve) => {
    let settled = false;
    // Only ever called from the request's events and the timer, all of which
    // come after `timer` below is set.
    const settle = (x: Exchange): void => {
      if (settled) return;
      settled = true;
      clearTimeout(timer);
      resolve(x);
    };
    const payload = body === undefined ? undefined : JSON.stringify(body);
    const req = http.request(
      {
        socketPath: socket,
        method,
        path,
        agent: false,
        headers:
          payload === undefined
            ? {}
            : { 'content-type': 'application/json', 'content-length': Buffer.byteLength(payload) },
      },
      (res) => {
        const chunks: Buffer[] = [];
        res.on('data', (c: Buffer) => chunks.push(c));
        res.on('end', () => settle({ kind: 'answer', status: res.statusCode ?? 0, body: Buffer.concat(chunks).toString('utf8') }));
        res.on('error', (err) => settle({ kind: 'unreachable', cause: err.message }));
        // After 'end' this settles nothing; before it, the daemon hung up mid-answer.
        res.on('close', () => settle({ kind: 'unreachable', cause: 'a conexão caiu no meio da resposta' }));
      },
    );
    const timer = setTimeout(() => {
      settle({ kind: 'unreachable', cause: `nada em ${seconds(timeoutMs)}` });
      req.destroy();
    }, timeoutMs);
    req.on('error', (err) => settle({ kind: 'unreachable', cause: err.message }));
    if (payload !== undefined) req.write(payload);
    req.end();
  });
}

/** Why a request never reached a daemon, naming the socket and the error. */
export function unreachable(socket: string, method: string, path: string, cause: string): Refusal {
  return { code: UNREACHABLE, message: `O daemon em ${socket} não respondeu a ${method} ${path}: ${cause}.` };
}

function parse(text: string): { ok: true; value: unknown } | { ok: false } {
  try {
    return { ok: true, value: JSON.parse(text) as unknown };
  } catch {
    return { ok: false };
  }
}

/** The wire's error body: `{code, message}`, both said. */
function wireError(v: unknown): Refusal | null {
  if (!isRecord(v) || typeof v.code !== 'string' || v.code === '' || typeof v.message !== 'string') return null;
  return { code: v.code, message: v.message };
}

/** A refusal for a status the daemon sent without its error body. */
function unexpected(status: number, text: string, method: string, path: string): Refusal {
  const said = text === '' ? ', sem corpo' : `: ${excerpt(text)}`;
  return { code: UNEXPECTED, message: `O daemon respondeu ${status} a ${method} ${path}${said}.` };
}

/** Reads an answer: the parsed body of a success (null when empty), else the daemon's refusal. */
export function answerOf(status: number, body: string, method: string, path: string): Answer<unknown> {
  const text = body.trim();
  const json = text === '' ? null : parse(text);
  if (status >= 200 && status < 300) {
    if (json === null) return { ok: true, value: null };
    if (json.ok) return { ok: true, value: json.value };
    return {
      ok: false,
      refusal: { code: UNEXPECTED, message: `O daemon respondeu ${status} a ${method} ${path} com algo que não é JSON: ${excerpt(text)}` },
    };
  }
  const refusal = json?.ok ? wireError(json.value) : null;
  return { ok: false, refusal: refusal ?? unexpected(status, text, method, path) };
}

/** One request to the daemon, answered: its value, its refusal, or why it never got there. */
export async function request(
  socket: string,
  method: string,
  path: string,
  body?: unknown,
  timeoutMs = REQUEST_TIMEOUT_MS,
): Promise<Answer<unknown>> {
  const x = await exchange(socket, method, path, body, timeoutMs);
  if (x.kind === 'unreachable') return { ok: false, refusal: unreachable(socket, method, path, x.cause) };
  return answerOf(x.status, x.body, method, path);
}

/** How a stream ended when nobody here closed it. */
export type StreamEnded =
  /** The daemon answered, and not with a stream: its refusal, and the status it came with. */
  | { kind: 'refused'; status: number; refusal: Refusal }
  /** No daemon took the request. */
  | { kind: 'unreachable'; cause: string }
  /** The stream was open, and closed. */
  | { kind: 'dropped'; cause: string };

export interface StreamHandlers {
  /** The frames of one chunk, in order. Never called with none: a ping alone is not a batch. */
  frames(frames: SseFrame[]): void;
  /** Called once, unless close() came first. */
  ended(end: StreamEnded): void;
}

export interface Stream {
  /** Stops reading; `ended` is not called after it. */
  close(): void;
}

/**
 * Opens an event stream. The daemon has `openWithinMs` to answer it; after
 * that the stream may stay quiet for as long as there is nothing to say.
 */
export function openStream(socket: string, path: string, on: StreamHandlers, openWithinMs = REQUEST_TIMEOUT_MS): Stream {
  let done = false;
  let answered = false;
  // As in exchange: only called from events that come after `timer` is set.
  const finish = (end: StreamEnded): void => {
    if (done) return;
    done = true;
    clearTimeout(timer);
    on.ended(end);
  };
  const req = http.request(
    { socketPath: socket, method: 'GET', path, agent: false, headers: { accept: 'text/event-stream' } },
    (res) => {
      answered = true;
      clearTimeout(timer);
      const status = res.statusCode ?? 0;
      if (status !== 200) {
        const chunks: Buffer[] = [];
        res.on('data', (c: Buffer) => chunks.push(c));
        res.on('end', () => {
          const answer = answerOf(status, Buffer.concat(chunks).toString('utf8'), 'GET', path);
          const refusal = answer.ok
            ? { code: UNEXPECTED, message: `O daemon respondeu ${status} a GET ${path}, e não com um fluxo.` }
            : answer.refusal;
          finish({ kind: 'refused', status, refusal });
        });
        res.on('error', (err) => finish({ kind: 'dropped', cause: err.message }));
        res.on('close', () => finish({ kind: 'dropped', cause: 'a conexão caiu no meio da resposta' }));
        return;
      }
      const parser = new SseParser();
      res.setEncoding('utf8');
      res.on('data', (chunk: string) => {
        if (done) return;
        const frames = parser.push(chunk);
        if (frames.length > 0) on.frames(frames);
      });
      res.on('end', () => finish({ kind: 'dropped', cause: 'o daemon encerrou o fluxo' }));
      res.on('error', (err) => finish({ kind: 'dropped', cause: err.message }));
      res.on('close', () => finish({ kind: 'dropped', cause: 'a conexão caiu' }));
    },
  );
  const timer = setTimeout(() => {
    finish({ kind: 'unreachable', cause: `nada em ${seconds(openWithinMs)}` });
    req.destroy();
  }, openWithinMs);
  req.on('error', (err) => finish(answered ? { kind: 'dropped', cause: err.message } : { kind: 'unreachable', cause: err.message }));
  req.end();
  return {
    close() {
      if (done) return;
      done = true;
      clearTimeout(timer);
      req.destroy();
    },
  };
}
