import { ConversationSnapshot, Version, type Conversation, type ConversationChange } from '../protocol/generated';
import { isRecord } from '../protocol/validate';
import type { Answer, DaemonStatus, Refusal, SessionEvents, StreamEnd } from '../shared/api';
import { countWorking, foldConversations } from './conversations';
import { Backoff, Followers } from './follow';
import { findBinary, findSocket, type Lookup } from './locate';
import { Requests, notConnected } from './requests';
import { exitWords, lastWords, type Exit, type Serve } from './serve';
import { decodeData } from './sse';
import { exchange, openStream, request, shown, type Stream, type StreamEnded } from './wire';

// The main process is the daemon's client (D2, D19), and this is where it
// stands with the daemon: finding one or starting one, and then knowing at
// every moment whether it still answers — saying so, with the reason, when it
// does not. Nothing here restarts a daemon on its own: one that died is lost
// until something answers on the socket again. `failed` is final for the run.

type Timer = ReturnType<typeof setTimeout>;

export interface Timing {
  /** How often a connected daemon is asked /health. */
  healthEveryMs: number;
  /** How often a lost daemon is asked whether it is back. */
  lostEveryMs: number;
  /** How long a daemon the app started has to answer /health. */
  startWithinMs: number;
  /** How often /health is asked while it starts. */
  startPollMs: number;
  /** The wait before a dropped stream is opened again: from, doubling up to. */
  retryFromMs: number;
  retryUpToMs: number;
}

export const TIMING: Timing = {
  healthEveryMs: 3_000,
  lostEveryMs: 2_000,
  startWithinMs: 15_000,
  startPollMs: 150,
  retryFromMs: 200,
  retryUpToMs: 2_000,
};

/** What the connection tells the windows, as it happens. */
export interface Pushes {
  status(status: DaemonStatus): void;
  /** A frame of the list's stream, as it came: decoded JSON, or the text when it was not JSON. */
  conversations(frame: unknown): void;
  sessionEvents(batch: SessionEvents): void;
  streamEnd(end: StreamEnd): void;
}

export interface ConnectionOptions {
  lookup: Lookup;
  /** Starts `<bin> serve --socket <socket>`: the app spawns it, a test fakes it. */
  serve(bin: string, socket: string): Serve;
  timing?: Partial<Timing>;
  /** The connection's own notes, for the main process's log. */
  log?(line: string): void;
}

const LIST_EVENTS = `/${Version}/conversations/events`;

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/** Reads `GET /version`: the version a person is shown, and a protocol this app speaks. */
function readVersion(a: Answer<unknown>): { ok: true; version: string } | { ok: false; problem: string } {
  if (!a.ok) return { ok: false, problem: `não ao /version: ${a.refusal.message}` };
  const v = a.value;
  if (!isRecord(v) || typeof v.version !== 'string' || v.version === '') {
    return { ok: false, problem: `o /version veio sem a versão: ${shown(v)}` };
  }
  if (v.protocol !== Version) return { ok: false, problem: `fala o protocolo ${shown(v.protocol)}, e este app fala o ${Version}.` };
  return { ok: true, version: v.version };
}

/** Why the list was refused, and — when the daemon does not know the route — what to do about it. */
export function listRefused(socket: string, status: number, refusal: Refusal): string {
  const said = `O daemon em ${socket} recusou a lista de conversas: ${refusal.message}`;
  // A dcode older than 0.23.0 has no /v1/conversations routes: its mux answers
  // a plain-text 404, and the person needs the way out more than the status.
  if (status === 404 || refusal.code === 'not_found') {
    return `${said} Este dcode não serve a lista de conversas — o desktop precisa do dcode 0.23.0 ou mais novo (dcode update).`;
  }
  return said;
}

function childGone(child: Serve, exit: Exit): string {
  return `O dcode serve que o app subiu saiu ${exitWords(exit)}.${lastWords(child.lines())}`;
}

function describe(s: DaemonStatus): string {
  switch (s.state) {
    case 'connecting':
      return 'connecting';
    case 'connected':
      return `connected: ${s.version} at ${s.socket}${s.started ? ', started by the app' : ''}`;
    default:
      return `${s.state}: ${s.reason}`;
  }
}

export class Connection {
  private now: DaemonStatus = { state: 'connecting' };
  private socket: string | null = null;
  private child: Serve | null = null;
  private list: Conversation[] | null = null;
  private listStream: Stream | null = null;
  private listRetry: Timer | null = null;
  private timer: Timer | null = null;
  private checking: Promise<void> | null = null;
  private begun = false;
  private stopped = false;
  private closing: Promise<void> | null = null;
  private readonly t: Timing;
  private readonly listBackoff: Backoff;
  private readonly sessions: Followers;
  /** The window's requests: each answered, and none sent while the daemon is not connected. */
  readonly requests = new Requests(() => this.now);

  constructor(
    private readonly o: ConnectionOptions,
    private readonly push: Pushes,
  ) {
    this.t = { ...TIMING, ...o.timing };
    this.listBackoff = new Backoff(this.t.retryFromMs, this.t.retryUpToMs);
    this.sessions = new Followers({
      socket: () => this.socket,
      retryFromMs: this.t.retryFromMs,
      retryUpToMs: this.t.retryUpToMs,
      events: (batch) => this.push.sessionEvents(batch),
      ended: (end) => this.push.streamEnd(end),
      dropped: () => this.stillThere(),
    });
  }

  /** Where the window stands with the daemon now. */
  status(): DaemonStatus {
    return this.now;
  }

  /** The whole list, as a snapshot, for a window that opens after the stream did; null before the first. */
  conversationsNow(): ConversationChange | null {
    return this.list === null ? null : { kind: ConversationSnapshot, conversations: this.list };
  }

  /** The app started the daemon, and it still runs: quitting stops it (D19). */
  startedByApp(): boolean {
    return this.child !== null && this.child.ended() === null;
  }

  /** Live sessions that stopping the daemon would cut off: running, or waiting on an approval. */
  working(): number {
    return this.list === null ? 0 : countWorking(this.list);
  }

  /** What each followed session has been read up to. */
  followed(): Record<string, number> {
    return this.sessions.positions();
  }

  start(): void {
    if (this.begun) return;
    this.begun = true;
    this.connect().catch((err: unknown) => {
      this.fail(`O app não conseguiu se ligar ao daemon: ${err instanceof Error ? err.message : String(err)}`);
    });
  }

  private log(line: string): void {
    this.o.log?.(line);
  }

  private async connect(): Promise<void> {
    const where = await findSocket(this.o.lookup);
    if (this.stopped) return;
    if (!where.ok) return this.fail(where.reason);
    const socket = where.path;
    this.socket = socket;
    const nobody = await this.healthy();
    if (this.stopped) return;
    if (nobody === null) return this.establish(socket, true);

    // Nothing answers there, so one is started: the only step that needs a binary.
    let bin = where.bin;
    if (bin === null) {
      const found = findBinary(this.o.lookup);
      if (!found.ok) return this.fail(`Nenhum daemon responde em ${socket} (${nobody}), e não deu para subir um: ${found.reason}`);
      bin = found.path;
    }
    const child = this.o.serve(bin, socket);
    this.child = child;
    this.log(`started ${bin} serve --socket ${socket} (pid ${child.pid ?? 'none'})`);
    void child.exited.then((exit) => this.childEnded(child, exit));
    const up = await this.waitUntilUp(child, socket);
    if (this.stopped) return;
    if (up !== null) return this.fail(up);
    await this.establish(socket, true);
  }

  /** Asks /health: null when it answers 200, else why not. */
  private async healthy(timeoutMs?: number): Promise<string | null> {
    if (this.socket === null) return 'não há socket';
    const x = await exchange(this.socket, 'GET', '/health', undefined, timeoutMs);
    if (x.kind === 'unreachable') return x.cause;
    return x.status === 200 ? null : `o /health respondeu ${x.status}`;
  }

  /** Waits for the daemon the app started: null once something answers, else why nothing will. */
  private async waitUntilUp(child: Serve, socket: string): Promise<string | null> {
    const deadline = Date.now() + this.t.startWithinMs;
    for (;;) {
      if (this.stopped) return null;
      const exit = child.ended();
      if (exit) {
        // Another client may have started one at the same moment, and won the socket.
        if ((await this.healthy()) === null) return null;
        return `O dcode serve que o app subiu saiu ${exitWords(exit)} antes de responder em ${socket}.${lastWords(child.lines())}`;
      }
      const left = deadline - Date.now();
      if (left <= 0) {
        await child.stop();
        const within = `${this.t.startWithinMs / 1000} s`;
        return `O dcode serve que o app subiu não respondeu em ${socket} em ${within}, e o app o encerrou.${lastWords(child.lines())}`;
      }
      if ((await this.healthy(Math.min(left, 2_000))) === null) return null;
      await Promise.race([child.exited, delay(this.t.startPollMs)]);
    }
  }

  /** The daemon answers: its version, the list's stream, and every followed session resumed. */
  private async establish(socket: string, first: boolean): Promise<void> {
    const version = readVersion(await request(socket, 'GET', '/version'));
    if (this.stopped) return;
    if (!version.ok) {
      const reason = `O daemon em ${socket} respondeu ao /health, mas ${version.problem}`;
      if (first) return this.fail(reason);
      this.set({ state: 'lost', reason });
      return this.schedule(this.t.lostEveryMs);
    }
    this.set({ state: 'connected', version: version.version, socket, started: this.startedByApp() });
    this.openList(socket);
    this.sessions.resumeAll();
    this.schedule(this.t.healthEveryMs);
  }

  private openList(socket: string): void {
    this.closeList();
    this.listStream = openStream(socket, LIST_EVENTS, {
      frames: (frames) => {
        this.listBackoff.reset();
        for (const f of frames) this.listFrame(decodeData(f.data));
      },
      ended: (end) => this.listEnded(socket, end),
    });
  }

  private closeList(): void {
    this.listStream?.close();
    this.listStream = null;
    if (this.listRetry) clearTimeout(this.listRetry);
    this.listRetry = null;
  }

  /** Folds the frame into the list kept here, and passes it on as it came. */
  private listFrame(frame: unknown): void {
    const folded = foldConversations(this.list, frame);
    if (folded.ok) this.list = folded.list;
    else this.log(`a frame of the list of conversations was not folded (${folded.problem}): ${shown(frame)}`);
    this.push.conversations(frame);
  }

  private listEnded(socket: string, end: StreamEnded): void {
    this.listStream = null;
    if (this.stopped || this.now.state !== 'connected') return;
    // A refusal is the daemon's answer, and asking again gets the same one: a
    // daemon without the list cannot serve this window.
    if (end.kind === 'refused') return this.fail(listRefused(socket, end.status, end.refusal));
    this.log(`the list of conversations closed (${end.cause}); asking /health`);
    void this.stillThere().then((up) => {
      if (!up || this.listStream !== null || this.listRetry !== null) return;
      // A daemon drops a client that fell behind; it reconnects to a fresh snapshot.
      this.listRetry = setTimeout(() => {
        this.listRetry = null;
        if (!this.stopped && this.now.state === 'connected' && this.listStream === null) this.openList(socket);
      }, this.listBackoff.next());
    });
  }

  /** Asks /health now — once, however many ask at the same time — and moves the status with the answer. */
  private check(): Promise<void> {
    this.checking ??= this.runCheck().finally(() => {
      this.checking = null;
    });
    return this.checking;
  }

  /** A stream closed: true when, asked right away, the daemon still answers. */
  private async stillThere(): Promise<boolean> {
    await this.check();
    return !this.stopped && this.now.state === 'connected';
  }

  private async runCheck(): Promise<void> {
    const was = this.now.state;
    if (this.stopped || this.socket === null || (was !== 'connected' && was !== 'lost')) return;
    this.unschedule();
    const why = await this.healthy();
    // The child's end may have moved the status meanwhile, and said more than this check can.
    if (this.stopped || this.now.state !== was) return;
    if (was === 'connected') {
      if (why === null) return this.schedule(this.t.healthEveryMs);
      return this.lose(this.lostReason(why));
    }
    if (why !== null) return this.schedule(this.t.lostEveryMs);
    this.log('the daemon answers again; reconnecting');
    await this.establish(this.socket, false);
  }

  private lostReason(why: string): string {
    const exit = this.child?.ended();
    if (this.child && exit) return childGone(this.child, exit);
    return `O daemon em ${this.socket} parou de responder: ${why}.`;
  }

  private lose(reason: string): void {
    this.closeList();
    this.sessions.suspendAll();
    this.set({ state: 'lost', reason });
    this.schedule(this.t.lostEveryMs);
  }

  private childEnded(child: Serve, exit: Exit): void {
    this.log(`dcode serve ended ${exitWords(exit)}`);
    if (this.stopped || this.child !== child) return;
    if (this.now.state === 'connected') this.lose(childGone(child, exit));
    else if (this.now.state === 'lost') this.set({ state: 'lost', reason: childGone(child, exit) });
    // While connecting, waitUntilUp says it; once failed, it was already said.
  }

  private fail(reason: string): void {
    this.unschedule();
    this.closeList();
    this.sessions.closeAll();
    this.set({ state: 'failed', reason });
  }

  private set(status: DaemonStatus): void {
    if (JSON.stringify(status) === JSON.stringify(this.now)) return;
    this.now = status;
    this.log(`daemon ${describe(status)}`);
    this.push.status(status);
  }

  private schedule(ms: number): void {
    this.unschedule();
    if (this.stopped) return;
    this.timer = setTimeout(() => {
      this.timer = null;
      void this.check();
    }, ms);
  }

  private unschedule(): void {
    if (this.timer) clearTimeout(this.timer);
    this.timer = null;
  }

  /** Follows a session's events from the first; answered at once, the events pushed as they come. */
  follow(sessionId: string): Answer<null> {
    if (this.now.state !== 'connected') return { ok: false, refusal: notConnected(this.now) };
    this.sessions.follow(sessionId);
    return { ok: true, value: null };
  }

  unfollow(sessionId: string): void {
    this.sessions.unfollow(sessionId);
  }

  /**
   * Stops listening, and stops the daemon when the app started it — never one
   * it did not (D19). Bounded by the child's stop, so quitting never hangs here.
   */
  close(): Promise<void> {
    if (this.closing === null) {
      this.stopped = true;
      this.closing = this.shutDown();
    }
    return this.closing;
  }

  private async shutDown(): Promise<void> {
    this.unschedule();
    // The streams first: a daemon told to stop waits on the connections still open to it.
    this.closeList();
    this.sessions.closeAll();
    const child = this.child;
    if (child === null || child.ended() !== null) return;
    this.log('stopping the dcode serve the app started');
    const exit = await child.stop();
    this.log(`dcode serve stopped ${exitWords(exit)}`);
  }
}
